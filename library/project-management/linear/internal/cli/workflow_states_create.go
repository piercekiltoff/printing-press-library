// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newWorkflowStatesCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyName string
	var bodyTeamId string
	var bodyColor string
	var bodyType string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a workflow state",
		Example: `  linear-pp-cli workflow_states create --name "In Review" --teamid abc --type started`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if bodyName == "" {
				return usageErr(fmt.Errorf("--name is required"))
			}
			if bodyTeamId == "" {
				return usageErr(fmt.Errorf("--teamid is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($input: WorkflowStateCreateInput!) {
				workflowStateCreate(input: $input) {
					success
					workflowState { id name color type }
				}
			}`

			input := map[string]any{
				"name":   bodyName,
				"teamId": bodyTeamId,
			}
			if bodyColor != "" {
				input["color"] = bodyColor
			}
			if bodyType != "" {
				input["type"] = bodyType
			}

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				WorkflowStateCreate struct {
					Success       bool            `json:"success"`
					WorkflowState json.RawMessage `json:"workflowState"`
				} `json:"workflowStateCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.WorkflowStateCreate.Success {
				return apiErr(fmt.Errorf("workflow state creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.WorkflowStateCreate.WorkflowState, flags)
		},
	}
	cmd.Flags().StringVar(&bodyName, "name", "", "State name (required)")
	cmd.Flags().StringVar(&bodyTeamId, "teamid", "", "Team ID (required)")
	cmd.Flags().StringVar(&bodyColor, "color", "", "State color hex code")
	cmd.Flags().StringVar(&bodyType, "type", "", "State type (triage, backlog, unstarted, started, completed, canceled)")

	return cmd
}
