// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newIssuesGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Short:   "Get a single issue by ID",
		Example: "  linear-pp-cli issues get 550e8400-e29b-41d4-a716-446655440000",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `query($id: String!) {
				issue(id: $id) {
					id identifier title description priority estimate dueDate
					createdAt updatedAt
					state { id name }
					team { id name }
					assignee { id name }
					project { id name }
					cycle { id number }
					parent { id identifier }
					labels { nodes { id name } }
					comments { nodes { id body createdAt user { id name } } }
				}
			}`

			variables := map[string]any{
				"id": args[0],
			}

			data, err := c.GraphQL(query, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			// Extract issue from data.issue
			var resp struct {
				Issue json.RawMessage `json:"issue"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if resp.Issue == nil || string(resp.Issue) == "null" {
				return notFoundErr(fmt.Errorf("issue %s not found", args[0]))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.Issue, flags)
		},
	}

	return cmd
}
