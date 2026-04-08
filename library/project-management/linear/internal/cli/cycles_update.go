// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newCyclesUpdateCmd(flags *rootFlags) *cobra.Command {
	var bodyName string
	var bodyStartsAt string
	var bodyEndsAt string

	cmd := &cobra.Command{
		Use:     "update <id>",
		Short:   "Update a cycle",
		Example: "  linear-pp-cli cycles update 550e8400-... --name \"Sprint 5\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: CycleUpdateInput!) {
				cycleUpdate(id: $id, input: $input) {
					success
					cycle { id number }
				}
			}`

			input := map[string]any{}
			if bodyName != "" {
				input["name"] = bodyName
			}
			if bodyStartsAt != "" {
				input["startsAt"] = bodyStartsAt
			}
			if bodyEndsAt != "" {
				input["endsAt"] = bodyEndsAt
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
				CycleUpdate struct {
					Success bool            `json:"success"`
					Cycle   json.RawMessage `json:"cycle"`
				} `json:"cycleUpdate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.CycleUpdate.Success {
				return apiErr(fmt.Errorf("cycle update failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.CycleUpdate.Cycle, flags)
		},
	}
	cmd.Flags().StringVar(&bodyName, "name", "", "Updated cycle name")
	cmd.Flags().StringVar(&bodyStartsAt, "startsat", "", "Updated start date")
	cmd.Flags().StringVar(&bodyEndsAt, "endsat", "", "Updated end date")

	return cmd
}
