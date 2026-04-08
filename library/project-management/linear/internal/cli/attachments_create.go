// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newAttachmentsCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyIssueId string
	var bodyUrl string
	var bodyTitle string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create an attachment on an issue",
		Example: `  linear-pp-cli attachments create --issueid abc --url "https://example.com" --title "Link"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if bodyIssueId == "" {
				return usageErr(fmt.Errorf("--issueid is required"))
			}
			if bodyUrl == "" {
				return usageErr(fmt.Errorf("--url is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($input: AttachmentCreateInput!) {
				attachmentCreate(input: $input) {
					success
					attachment { id title url }
				}
			}`

			input := map[string]any{
				"issueId": bodyIssueId,
				"url":     bodyUrl,
			}
			if bodyTitle != "" {
				input["title"] = bodyTitle
			}

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				AttachmentCreate struct {
					Success    bool            `json:"success"`
					Attachment json.RawMessage `json:"attachment"`
				} `json:"attachmentCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.AttachmentCreate.Success {
				return apiErr(fmt.Errorf("attachment creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.AttachmentCreate.Attachment, flags)
		},
	}
	cmd.Flags().StringVar(&bodyIssueId, "issueid", "", "Issue ID to attach to (required)")
	cmd.Flags().StringVar(&bodyUrl, "url", "", "Attachment URL (required)")
	cmd.Flags().StringVar(&bodyTitle, "title", "", "Attachment title")

	return cmd
}
