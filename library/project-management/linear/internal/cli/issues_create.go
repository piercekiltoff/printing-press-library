// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newIssuesCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyTitle string
	var bodyDescription string
	var bodyTeamId string
	var bodyAssigneeId string
	var bodyStateId string
	var bodyPriority string
	var bodyLabelIds []string
	var bodyProjectId string
	var bodyCycleId string
	var bodyEstimate string
	var bodyParentId string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		Example: `  linear-pp-cli issues create --title "Bug report" --teamid abc-123
  linear-pp-cli issues create --title "Feature" --teamid abc --priority 2 --labelid lbl1 --labelid lbl2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if bodyTitle == "" {
				return usageErr(fmt.Errorf("--title is required"))
			}
			if bodyTeamId == "" {
				return usageErr(fmt.Errorf("--teamid is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($input: IssueCreateInput!) {
				issueCreate(input: $input) {
					success
					issue {
						id identifier title url
					}
				}
			}`

			input := map[string]any{
				"title":  bodyTitle,
				"teamId": bodyTeamId,
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
			if bodyParentId != "" {
				input["parentId"] = bodyParentId
			}

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			// Extract issueCreate result
			var resp struct {
				IssueCreate struct {
					Success bool            `json:"success"`
					Issue   json.RawMessage `json:"issue"`
				} `json:"issueCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.IssueCreate.Success {
				return apiErr(fmt.Errorf("issue creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.IssueCreate.Issue, flags)
		},
	}
	cmd.Flags().StringVar(&bodyTitle, "title", "", "Issue title (required)")
	cmd.Flags().StringVar(&bodyDescription, "description", "", "Issue description in markdown")
	cmd.Flags().StringVar(&bodyTeamId, "teamid", "", "Team ID (required)")
	cmd.Flags().StringVar(&bodyAssigneeId, "assigneeid", "", "Assignee user ID")
	cmd.Flags().StringVar(&bodyStateId, "stateid", "", "Workflow state ID")
	cmd.Flags().StringVar(&bodyPriority, "priority", "", "Priority (0=none, 1=urgent, 2=high, 3=medium, 4=low)")
	cmd.Flags().StringArrayVar(&bodyLabelIds, "labelid", nil, "Label ID (repeatable)")
	cmd.Flags().StringVar(&bodyProjectId, "projectid", "", "Project ID")
	cmd.Flags().StringVar(&bodyCycleId, "cycleid", "", "Cycle ID")
	cmd.Flags().StringVar(&bodyEstimate, "estimate", "", "Issue estimate points")
	cmd.Flags().StringVar(&bodyParentId, "parentid", "", "Parent issue ID for sub-issues")

	return cmd
}
