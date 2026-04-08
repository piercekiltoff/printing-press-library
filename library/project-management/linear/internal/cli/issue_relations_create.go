// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newIssueRelationsCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyIssueId string
	var bodyRelatedIssueId string
	var bodyType string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create an issue relation",
		Example: `  linear-pp-cli issue_relations create --issueid abc --relatedissueid def --type blocks`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if bodyIssueId == "" {
				return usageErr(fmt.Errorf("--issueid is required"))
			}
			if bodyRelatedIssueId == "" {
				return usageErr(fmt.Errorf("--relatedissueid is required"))
			}
			if bodyType == "" {
				return usageErr(fmt.Errorf("--type is required (blocks, blocked_by, related, duplicate)"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($input: IssueRelationCreateInput!) {
				issueRelationCreate(input: $input) {
					success
					issueRelation { id type }
				}
			}`

			input := map[string]any{
				"issueId":        bodyIssueId,
				"relatedIssueId": bodyRelatedIssueId,
				"type":           bodyType,
			}

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				IssueRelationCreate struct {
					Success       bool            `json:"success"`
					IssueRelation json.RawMessage `json:"issueRelation"`
				} `json:"issueRelationCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.IssueRelationCreate.Success {
				return apiErr(fmt.Errorf("issue relation creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.IssueRelationCreate.IssueRelation, flags)
		},
	}
	cmd.Flags().StringVar(&bodyIssueId, "issueid", "", "Source issue ID (required)")
	cmd.Flags().StringVar(&bodyRelatedIssueId, "relatedissueid", "", "Related issue ID (required)")
	cmd.Flags().StringVar(&bodyType, "type", "", "Relation type: blocks, blocked_by, related, duplicate (required)")

	return cmd
}
