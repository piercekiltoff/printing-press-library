package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRoutinesHealthCmd(flags *rootFlags) *cobra.Command {
	var companyID string

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check routine run health: consecutive failures and error rates",
		Long: `Fetches all routines and their recent runs, then reports health status for
each one. Useful for finding broken scheduled automation before users notice.

Status levels:
  healthy  — no recent failures
  warning  — 1-2 consecutive failures
  critical — 3+ consecutive failures`,
		Example: `  paperclip-pp-cli routines health --company-id <id>
  paperclip-pp-cli routines health --company-id <id> --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cid := companyID
			if cid == "" {
				cid = os.Getenv("PAPERCLIP_COMPANY_ID")
			}
			if cid == "" {
				return usageErr(fmt.Errorf("company ID required: use --company-id or set PAPERCLIP_COMPANY_ID"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			routinesData, err := c.Get("/api/companies/"+cid+"/routines", nil)
			if err != nil {
				return classifyAPIError(err)
			}
			var routines []map[string]any
			if err := json.Unmarshal(routinesData, &routines); err != nil {
				return fmt.Errorf("parsing routines: %w", err)
			}

			type healthRow struct {
				ID               string `json:"id"`
				Name             string `json:"name"`
				Schedule         string `json:"schedule"`
				LastRun          string `json:"lastRun"`
				ConsecutiveFails int    `json:"consecutiveFailures"`
				Status           string `json:"status"`
			}

			rows := []healthRow{}

			for _, r := range routines {
				id, _ := r["id"].(string)
				name, _ := r["title"].(string)
				// Extract cron from first trigger
				schedule := "-"
				if triggers, ok := r["triggers"].([]any); ok && len(triggers) > 0 {
					if t, ok := triggers[0].(map[string]any); ok {
						if cron, ok := t["cronExpression"].(string); ok && cron != "" {
							schedule = cron
						}
					}
				}

				// Fetch recent runs
				runsData, err := c.Get("/api/routines/"+id+"/runs", map[string]string{"limit": "10"})
				lastRun := "-"
				consecutiveFails := 0

				if err == nil {
					var runs []map[string]any
					if json.Unmarshal(runsData, &runs) == nil {
						if len(runs) > 0 {
							first := runs[0]
							if ts, ok := first["createdAt"].(string); ok {
								lastRun = ts[:16] // trim to YYYY-MM-DDTHH:MM
							}
						}
						// Count consecutive failures from the most recent runs
						for _, run := range runs {
							status, _ := run["status"].(string)
							if status == "failed" || status == "error" {
								consecutiveFails++
							} else {
								break
							}
						}
					}
				}

				health := "healthy"
				if consecutiveFails >= 3 {
					health = red("critical")
				} else if consecutiveFails > 0 {
					health = yellow("warning")
				}

				rows = append(rows, healthRow{
					ID:               truncate(id, 8),
					Name:             truncate(name, 30),
					Schedule:         schedule,
					LastRun:          lastRun,
					ConsecutiveFails: consecutiveFails,
					Status:           health,
				})
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				// Strip ANSI for JSON
				for i := range rows {
					if rows[i].ConsecutiveFails >= 3 {
						rows[i].Status = "critical"
					} else if rows[i].ConsecutiveFails > 0 {
						rows[i].Status = "warning"
					}
				}
				out, _ := json.MarshalIndent(rows, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}

			if len(rows) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No routines found.")
				return nil
			}

			tw := newTabWriter(cmd.OutOrStdout())
			fmt.Fprintln(tw, "NAME\tSCHEDULE\tLAST RUN\tFAILURES\tSTATUS")
			for _, r := range rows {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n",
					r.Name, r.Schedule, r.LastRun, r.ConsecutiveFails, r.Status)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&companyID, "company-id", "", "Company ID (defaults to PAPERCLIP_COMPANY_ID env var)")
	return cmd
}
