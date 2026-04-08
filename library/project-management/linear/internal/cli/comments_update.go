// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newCommentsUpdateCmd(flags *rootFlags) *cobra.Command {
	var bodyBody string

	cmd := &cobra.Command{
		Use:     "update <id>",
		Short:   "Update a comment",
		Example: `  linear-pp-cli comments update 550e8400-... --body "Updated text"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}
			if bodyBody == "" {
				return usageErr(fmt.Errorf("--body is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: CommentUpdateInput!) {
				commentUpdate(id: $id, input: $input) {
					success
					comment { id body }
				}
			}`

			input := map[string]any{
				"body": bodyBody,
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
				CommentUpdate struct {
					Success bool            `json:"success"`
					Comment json.RawMessage `json:"comment"`
				} `json:"commentUpdate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.CommentUpdate.Success {
				return apiErr(fmt.Errorf("comment update failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.CommentUpdate.Comment, flags)
		},
	}
	cmd.Flags().StringVar(&bodyBody, "body", "", "Updated comment body (required)")

	return cmd
}
