// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newCommentsListCmd(flags *rootFlags) *cobra.Command {
	var flagIssueId string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List comments on an issue",
		Example: "  linear-pp-cli comments list --issueid abc-123",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagIssueId == "" {
				return usageErr(fmt.Errorf("--issueid is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `query($issueId: String!) {
				issue(id: $issueId) {
					comments {
						nodes { id body createdAt user { id name } }
					}
				}
			}`

			variables := map[string]any{
				"issueId": flagIssueId,
			}

			data, err := c.GraphQL(query, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				Issue struct {
					Comments struct {
						Nodes json.RawMessage `json:"nodes"`
					} `json:"comments"`
				} `json:"issue"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			result := resp.Issue.Comments.Nodes
			if result == nil {
				result = json.RawMessage("[]")
			}

			return printOutputWithFlags(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagIssueId, "issueid", "", "Issue ID (required)")

	return cmd
}
