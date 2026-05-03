package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type fleetAgent struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	SpentMonthlyCents int    `json:"spentMonthlyCents"`
	ActiveRun         bool   `json:"activeRun"`
}

func newFleetCmd(flags *rootFlags) *cobra.Command {
	var companyID string

	cmd := &cobra.Command{
		Use:   "fleet",
		Short: "Live status, costs, and active runs for every agent in a company",
		Long: `Aggregates agents, monthly costs, and live runs into a single fleet overview.

Useful for getting situational awareness of your entire agent fleet at once
without calling multiple endpoints separately.`,
		Example: `  # Show fleet status for the active company
  paperclip-pp-cli fleet

  # JSON output for scripting
  paperclip-pp-cli fleet --json

  # Specify a company explicitly
  paperclip-pp-cli fleet --company-id <companyId>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cid := companyID
			if cid == "" {
				cid = os.Getenv("PAPERCLIP_COMPANY_ID")
			}
			if flags.dryRun {
				preview := map[string]any{
					"command":    "fleet",
					"company_id": cid,
					"endpoints": []string{
						"/api/companies/{id}/agents",
						"/api/companies/{id}/costs/by-agent",
						"/api/companies/{id}/live-runs",
					},
				}
				out, _ := json.MarshalIndent(preview, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}
			if cid == "" {
				return usageErr(fmt.Errorf("company ID required: use --company-id or set PAPERCLIP_COMPANY_ID"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Fetch agents
			agentsData, err := c.Get("/api/companies/"+cid+"/agents", nil)
			if err != nil {
				return classifyAPIError(err)
			}
			var agentsList []map[string]any
			if err := json.Unmarshal(agentsData, &agentsList); err != nil {
				return fmt.Errorf("parsing agents: %w", err)
			}

			// Fetch costs by agent
			costsData, err := c.Get("/api/companies/"+cid+"/costs/by-agent", nil)
			if err != nil {
				// Non-fatal — proceed without cost data
				costsData = []byte("[]")
			}
			var costsList []map[string]any
			_ = json.Unmarshal(costsData, &costsList)

			// Index costs by agent ID
			costsByAgent := map[string]int{}
			for _, cost := range costsList {
				aid, _ := cost["agentId"].(string)
				spent, _ := cost["costCents"].(float64)
				if aid != "" {
					costsByAgent[aid] = int(spent)
				}
			}

			// Fetch live runs to detect active agents
			liveData, err := c.Get("/api/companies/"+cid+"/live-runs", nil)
			if err != nil {
				liveData = []byte("[]")
			}
			var liveList []map[string]any
			_ = json.Unmarshal(liveData, &liveList)

			activeAgents := map[string]bool{}
			for _, run := range liveList {
				if aid, ok := run["agentId"].(string); ok && aid != "" {
					activeAgents[aid] = true
				}
			}

			// Build fleet rows
			fleet := make([]fleetAgent, 0, len(agentsList))
			for _, a := range agentsList {
				id, _ := a["id"].(string)
				name, _ := a["name"].(string)
				status, _ := a["status"].(string)
				fleet = append(fleet, fleetAgent{
					ID:                id,
					Name:              name,
					Status:            status,
					SpentMonthlyCents: costsByAgent[id],
					ActiveRun:         activeAgents[id],
				})
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				out, _ := json.MarshalIndent(fleet, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "AGENT\tSTATUS\tMONTHLY SPEND\tACTIVE RUN")
			for _, fa := range fleet {
				active := "no"
				if fa.ActiveRun {
					active = green("yes")
				}
				spend := fmt.Sprintf("$%.2f", float64(fa.SpentMonthlyCents)/100)
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					truncate(fa.Name, 30), fa.Status, spend, active)
			}
			w.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&companyID, "company-id", "", "Company ID (defaults to PAPERCLIP_COMPANY_ID env var)")
	return cmd
}
