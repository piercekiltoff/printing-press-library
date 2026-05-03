package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newIssuesStaleCmd(flags *rootFlags) *cobra.Command {
	var companyID string
	var days int
	var limit int

	cmd := &cobra.Command{
		Use:   "stale",
		Short: "Find in-progress issues with no agent activity in N days",
		Long: `Fetches in-progress issues from the live API and filters to those not updated
in --days days. Useful for finding stuck or forgotten work that an agent checked
out but stopped progressing.`,
		Example: `  paperclip-pp-cli issues stale --company-id <id>
  paperclip-pp-cli issues stale --company-id <id> --days 7 --json
  paperclip-pp-cli issues stale --company-id <id> --days 1 --agent`,
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

			issuesData, err := c.Get("/api/companies/"+cid+"/issues", map[string]string{
				"status": "in_progress",
			})
			if err != nil {
				return classifyAPIError(err)
			}

			var issues []map[string]any
			if err := json.Unmarshal(issuesData, &issues); err != nil {
				return fmt.Errorf("parsing issues: %w", err)
			}

			type staleIssue struct {
				Identifier string `json:"identifier"`
				Title      string `json:"title"`
				AgentName  string `json:"agentName"`
				DaysStale  int    `json:"daysStale"`
				UpdatedAt  string `json:"updatedAt"`
			}

			cutoff := time.Now().AddDate(0, 0, -days)
			stale := []staleIssue{}

			for _, iss := range issues {
				updatedAt, _ := iss["updatedAt"].(string)
				if updatedAt == "" {
					continue
				}
				t, err := time.Parse(time.RFC3339, updatedAt)
				if err != nil {
					continue
				}
				if t.After(cutoff) {
					continue
				}

				identifier, _ := iss["identifier"].(string)
				title, _ := iss["title"].(string)

				agentName := ""
				if assignee, ok := iss["assignee"].(map[string]any); ok {
					agentName, _ = assignee["name"].(string)
				}
				if agentName == "" {
					if aname, ok := iss["assigneeName"].(string); ok {
						agentName = aname
					}
				}

				stale = append(stale, staleIssue{
					Identifier: identifier,
					Title:      truncate(title, 50),
					AgentName:  agentName,
					DaysStale:  int(time.Since(t).Hours() / 24),
					UpdatedAt:  updatedAt,
				})

				if limit > 0 && len(stale) >= limit {
					break
				}
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				out, _ := json.MarshalIndent(stale, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}

			if len(stale) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No in-progress issues stale for %d+ days.\n", days)
				return nil
			}

			tw := newTabWriter(cmd.OutOrStdout())
			fmt.Fprintln(tw, "IDENTIFIER\tTITLE\tASSIGNED AGENT\tDAYS STALE")
			for _, s := range stale {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%d\n",
					s.Identifier, s.Title, s.AgentName, s.DaysStale)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&companyID, "company-id", "", "Company ID (defaults to PAPERCLIP_COMPANY_ID env var)")
	cmd.Flags().IntVar(&days, "days", 3, "Minimum days without update to consider stale")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum issues to return")
	return cmd
}
