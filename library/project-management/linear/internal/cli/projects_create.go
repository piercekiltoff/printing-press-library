// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newProjectsCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyName string
	var bodyDescription string
	var bodyTeamId string
	var bodyLeadId string
	var bodyStartDate string
	var bodyTargetDate string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new project",
		Example: "  linear-pp-cli projects create --name \"Q1 Launch\" --teamid abc-123",
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

			mutation := `mutation($input: ProjectCreateInput!) {
				projectCreate(input: $input) {
					success
					project { id name slugId }
				}
			}`

			input := map[string]any{
				"name":    bodyName,
				"teamIds": []string{bodyTeamId},
			}
			if bodyDescription != "" {
				input["description"] = bodyDescription
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

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				ProjectCreate struct {
					Success bool            `json:"success"`
					Project json.RawMessage `json:"project"`
				} `json:"projectCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.ProjectCreate.Success {
				return apiErr(fmt.Errorf("project creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.ProjectCreate.Project, flags)
		},
	}
	cmd.Flags().StringVar(&bodyName, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&bodyDescription, "description", "", "Project description")
	cmd.Flags().StringVar(&bodyTeamId, "teamid", "", "Team ID (required)")
	cmd.Flags().StringVar(&bodyLeadId, "leadid", "", "Lead user ID")
	cmd.Flags().StringVar(&bodyStartDate, "startdate", "", "Start date in ISO 8601 format")
	cmd.Flags().StringVar(&bodyTargetDate, "targetdate", "", "Target completion date in ISO 8601 format")

	return cmd
}
