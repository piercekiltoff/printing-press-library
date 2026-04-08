// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newCommentsCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyIssueId string
	var bodyBody string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a comment on an issue",
		Example: `  linear-pp-cli comments create --issueid abc-123 --body "Looks good!"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if bodyIssueId == "" {
				return usageErr(fmt.Errorf("--issueid is required"))
			}
			if bodyBody == "" {
				return usageErr(fmt.Errorf("--body is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($input: CommentCreateInput!) {
				commentCreate(input: $input) {
					success
					comment { id body }
				}
			}`

			input := map[string]any{
				"issueId": bodyIssueId,
				"body":    bodyBody,
			}

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				CommentCreate struct {
					Success bool            `json:"success"`
					Comment json.RawMessage `json:"comment"`
				} `json:"commentCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.CommentCreate.Success {
				return apiErr(fmt.Errorf("comment creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.CommentCreate.Comment, flags)
		},
	}
	cmd.Flags().StringVar(&bodyIssueId, "issueid", "", "Issue ID to comment on (required)")
	cmd.Flags().StringVar(&bodyBody, "body", "", "Comment body in markdown (required)")

	return cmd
}
