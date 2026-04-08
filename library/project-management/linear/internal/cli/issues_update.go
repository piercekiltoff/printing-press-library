// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newIssuesUpdateCmd(flags *rootFlags) *cobra.Command {
	var bodyTitle string
	var bodyDescription string
	var bodyAssigneeId string
	var bodyStateId string
	var bodyPriority string
	var bodyLabelIds []string
	var bodyProjectId string
	var bodyCycleId string
	var bodyEstimate string
	var bodyDueDate string
	var bodyParentId string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an existing issue",
		Example: `  linear-pp-cli issues update 550e8400-... --title "New title"
  linear-pp-cli issues update 550e8400-... --priority 1 --stateid abc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: IssueUpdateInput!) {
				issueUpdate(id: $id, input: $input) {
					success
					issue {
						id identifier title url
					}
				}
			}`

			input := map[string]any{}
			if bodyTitle != "" {
				input["title"] = bodyTitle
			}
			if bodyDescription != "" {
				input["description"] = bodyDescription
			}
			if bodyAssigneeId != "" {
				input["assigneeId"] = bodyAssigneeId
			}
			if bodyStateId != "" {
				input["stateId"] = bodyStateId
			}
			if bodyPriority != "" {
				if p, err := strconv.Atoi(bodyPriority); err == nil {
					input["priority"] = p
				}
			}
			if len(bodyLabelIds) > 0 {
				input["labelIds"] = bodyLabelIds
			}
			if bodyProjectId != "" {
				input["projectId"] = bodyProjectId
			}
			if bodyCycleId != "" {
				input["cycleId"] = bodyCycleId
			}
			if bodyEstimate != "" {
				if e, err := strconv.Atoi(bodyEstimate); err == nil {
					input["estimate"] = e
				}
			}
			if bodyDueDate != "" {
				input["dueDate"] = bodyDueDate
			}
			if bodyParentId != "" {
				input["parentId"] = bodyParentId
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
				IssueUpdate struct {
					Success bool            `json:"success"`
					Issue   json.RawMessage `json:"issue"`
				} `json:"issueUpdate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.IssueUpdate.Success {
				return apiErr(fmt.Errorf("issue update failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.IssueUpdate.Issue, flags)
		},
	}
	cmd.Flags().StringVar(&bodyTitle, "title", "", "Updated title")
	cmd.Flags().StringVar(&bodyDescription, "description", "", "Updated description")
	cmd.Flags().StringVar(&bodyAssigneeId, "assigneeid", "", "Updated assignee user ID")
	cmd.Flags().StringVar(&bodyStateId, "stateid", "", "Updated workflow state ID")
	cmd.Flags().StringVar(&bodyPriority, "priority", "", "Updated priority")
	cmd.Flags().StringArrayVar(&bodyLabelIds, "labelid", nil, "Updated label ID (repeatable)")
	cmd.Flags().StringVar(&bodyProjectId, "projectid", "", "Updated project ID")
	cmd.Flags().StringVar(&bodyCycleId, "cycleid", "", "Updated cycle ID")
	cmd.Flags().StringVar(&bodyEstimate, "estimate", "", "Updated estimate points")
	cmd.Flags().StringVar(&bodyDueDate, "duedate", "", "Updated due date")
	cmd.Flags().StringVar(&bodyParentId, "parentid", "", "Updated parent issue ID")

	return cmd
}
