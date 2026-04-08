// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newCommentsGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Short:   "Get a single comment by ID",
		Example: "  linear-pp-cli comments get 550e8400-e29b-41d4-a716-446655440000",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `query($id: String!) {
				comment(id: $id) {
					id body createdAt updatedAt
					issue { id identifier }
					user { id name }
				}
			}`

			variables := map[string]any{
				"id": args[0],
			}

			data, err := c.GraphQL(query, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				Comment json.RawMessage `json:"comment"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if resp.Comment == nil || string(resp.Comment) == "null" {
				return notFoundErr(fmt.Errorf("comment %s not found", args[0]))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.Comment, flags)
		},
	}

	return cmd
}
