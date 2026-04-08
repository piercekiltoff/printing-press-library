// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newProjectsUpdateCmd(flags *rootFlags) *cobra.Command {
	var bodyName string
	var bodyDescription string
	var bodyState string
	var bodyLeadId string
	var bodyStartDate string
	var bodyTargetDate string

	cmd := &cobra.Command{
		Use:     "update <id>",
		Short:   "Update a project",
		Example: "  linear-pp-cli projects update 550e8400-... --name \"New Name\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: ProjectUpdateInput!) {
				projectUpdate(id: $id, input: $input) {
					success
					project { id name }
				}
			}`

			input := map[string]any{}
			if bodyName != "" {
				input["name"] = bodyName
			}
			if bodyDescription != "" {
				input["description"] = bodyDescription
			}
			if bodyState != "" {
				input["state"] = bodyState
			}
			if bodyLeadId != "" {
				input["leadId"] = bodyLeadId
			}
			if bodyStartDate != "" {
				input["startDate"] = bodyStartDate
			}
			if bodyTargetDate != "" {
				input["targetDate"] = bodyTargetDate
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
				ProjectUpdate struct {
					Success bool            `json:"success"`
					Project json.RawMessage `json:"project"`
				} `json:"projectUpdate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.ProjectUpdate.Success {
				return apiErr(fmt.Errorf("project update failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.ProjectUpdate.Project, flags)
		},
	}
	cmd.Flags().StringVar(&bodyName, "name", "", "Updated project name")
	cmd.Flags().StringVar(&bodyDescription, "description", "", "Updated description")
	cmd.Flags().StringVar(&bodyState, "state", "", "Updated project state")
	cmd.Flags().StringVar(&bodyLeadId, "leadid", "", "Updated lead user ID")
	cmd.Flags().StringVar(&bodyStartDate, "startdate", "", "Updated start date")
	cmd.Flags().StringVar(&bodyTargetDate, "targetdate", "", "Updated target date")

	return cmd
}
