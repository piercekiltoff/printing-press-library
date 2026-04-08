// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newProjectsDeleteCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <id>",
		Short:   "Delete (archive) a project",
		Example: "  linear-pp-cli projects delete 550e8400-e29b-41d4-a716-446655440000",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!) {
				projectDelete(id: $id) { success }
			}`

			variables := map[string]any{
				"id": args[0],
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				ProjectDelete struct {
					Success bool `json:"success"`
				} `json:"projectDelete"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.ProjectDelete.Success {
				return apiErr(fmt.Errorf("project deletion failed"))
			}

			result, _ := json.Marshal(map[string]any{"success": true, "id": args[0]})
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(result), flags)
		},
	}

	return cmd
}
