package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newApprovalsQueueCmd(flags *rootFlags) *cobra.Command {
	var companyID string

	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Show pending approvals with linked issues and wait time",
		Long: `Lists all pending approvals and enriches each with linked issue identifiers
and the time the approval has been waiting. Useful for triaging the human-review
backlog without clicking through the UI.`,
		Example: `  paperclip-pp-cli approvals queue --company-id <id>
  paperclip-pp-cli approvals queue --company-id <id> --json | jq '.[].id'`,
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

			// Fetch all approvals
			approvalsData, err := c.Get("/api/companies/"+cid+"/approvals", nil)
			if err != nil {
				return classifyAPIError(err)
			}
			var approvals []map[string]any
			if err := json.Unmarshal(approvalsData, &approvals); err != nil {
				return fmt.Errorf("parsing approvals: %w", err)
			}

			type queueRow struct {
				ID           string `json:"id"`
				Title        string `json:"title"`
				Status       string `json:"status"`
				LinkedIssues string `json:"linkedIssues"`
				WaitTime     string `json:"waitTime"`
				CreatedAt    string `json:"createdAt"`
			}

			rows := []queueRow{}
			now := time.Now()

			for _, a := range approvals {
				status, _ := a["status"].(string)
				// Filter to pending/in-review
				if status != "pending" && status != "in_review" && status != "" {
					continue
				}

				id, _ := a["id"].(string)
				title, _ := a["title"].(string)
				createdAt, _ := a["createdAt"].(string)

				waitStr := ""
				if createdAt != "" {
					if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
						dur := now.Sub(t)
						if dur.Hours() >= 24 {
							waitStr = fmt.Sprintf("%.0fd", dur.Hours()/24)
						} else {
							waitStr = fmt.Sprintf("%.0fh", dur.Hours())
						}
					}
				}

				// Fetch linked issues (best-effort)
				linkedIDs := []string{}
				if id != "" {
					issuesData, err := c.Get("/api/approvals/"+id+"/issues", nil)
					if err == nil {
						var issues []map[string]any
						if json.Unmarshal(issuesData, &issues) == nil {
							for _, iss := range issues {
								if ident, ok := iss["identifier"].(string); ok && ident != "" {
									linkedIDs = append(linkedIDs, ident)
								} else if iid, ok := iss["id"].(string); ok {
									linkedIDs = append(linkedIDs, truncate(iid, 8))
								}
							}
						}
					}
				}

				rows = append(rows, queueRow{
					ID:           truncate(id, 8),
					Title:        truncate(title, 40),
					Status:       status,
					LinkedIssues: strings.Join(linkedIDs, ", "),
					WaitTime:     waitStr,
					CreatedAt:    createdAt,
				})
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				out, _ := json.MarshalIndent(rows, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}

			if len(rows) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No pending approvals.")
				return nil
			}

			tw := newTabWriter(cmd.OutOrStdout())
			fmt.Fprintln(tw, "ID\tTITLE\tSTATUS\tLINKED ISSUES\tWAITING")
			for _, r := range rows {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
					r.ID, r.Title, r.Status, r.LinkedIssues, r.WaitTime)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().StringVar(&companyID, "company-id", "", "Company ID (defaults to PAPERCLIP_COMPANY_ID env var)")
	return cmd
}
