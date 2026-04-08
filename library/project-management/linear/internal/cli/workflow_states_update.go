// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newWorkflowStatesUpdateCmd(flags *rootFlags) *cobra.Command {
	var bodyName string
	var bodyColor string
	var bodyDescription string

	cmd := &cobra.Command{
		Use:     "update <id>",
		Short:   "Update a workflow state",
		Example: `  linear-pp-cli workflow_states update 550e8400-... --name "Done"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: WorkflowStateUpdateInput!) {
				workflowStateUpdate(id: $id, input: $input) {
					success
					workflowState { id name color type }
				}
			}`

			input := map[string]any{}
			if bodyName != "" {
				input["name"] = bodyName
			}
			if bodyColor != "" {
				input["color"] = bodyColor
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
				WorkflowStateUpdate struct {
					Success       bool            `json:"success"`
					WorkflowState json.RawMessage `json:"workflowState"`
				} `json:"workflowStateUpdate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.WorkflowStateUpdate.Success {
				return apiErr(fmt.Errorf("workflow state update failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.WorkflowStateUpdate.WorkflowState, flags)
		},
	}
	cmd.Flags().StringVar(&bodyName, "name", "", "Updated state name")
	cmd.Flags().StringVar(&bodyColor, "color", "", "Updated color")
	cmd.Flags().StringVar(&bodyDescription, "description", "", "Updated description")

	return cmd
}
