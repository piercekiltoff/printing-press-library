// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newTeamsUpdateCmd(flags *rootFlags) *cobra.Command {
	var bodyName string
	var bodyDescription string

	cmd := &cobra.Command{
		Use:     "update <id>",
		Short:   "Update a team",
		Example: "  linear-pp-cli teams update 550e8400-... --name \"New Name\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: TeamUpdateInput!) {
				teamUpdate(id: $id, input: $input) {
					success
					team { id name key }
				}
			}`

			input := map[string]any{}
			if bodyName != "" {
				input["name"] = bodyName
			}
			if bodyDescription != "" {
				input["description"] = bodyDescription
			}

			if len(input) == 0 {
				return usageErr(fmt.Errorf("at least one field to update is required"))
			}

			variables := map[string]any{
				"id":    args[0],
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				TeamUpdate struct {
					Success bool            `json:"success"`
					Team    json.RawMessage `json:"team"`
				} `json:"teamUpdate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.TeamUpdate.Success {
				return apiErr(fmt.Errorf("team update failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.TeamUpdate.Team, flags)
		},
	}
	cmd.Flags().StringVar(&bodyName, "name", "", "Updated team name")
	cmd.Flags().StringVar(&bodyDescription, "description", "", "Updated description")

	return cmd
}
