// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newAttachmentsUpdateCmd(flags *rootFlags) *cobra.Command {
	var bodyTitle string
	var bodyUrl string

	cmd := &cobra.Command{
		Use:     "update <id>",
		Short:   "Update an attachment",
		Example: `  linear-pp-cli attachments update 550e8400-... --title "New Title"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: AttachmentUpdateInput!) {
				attachmentUpdate(id: $id, input: $input) {
					success
					attachment { id title url }
				}
			}`

			input := map[string]any{}
			if bodyTitle != "" {
				input["title"] = bodyTitle
			}
			if bodyUrl != "" {
				input["url"] = bodyUrl
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
				AttachmentUpdate struct {
					Success    bool            `json:"success"`
					Attachment json.RawMessage `json:"attachment"`
				} `json:"attachmentUpdate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.AttachmentUpdate.Success {
				return apiErr(fmt.Errorf("attachment update failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.AttachmentUpdate.Attachment, flags)
		},
	}
	cmd.Flags().StringVar(&bodyTitle, "title", "", "Updated title")
	cmd.Flags().StringVar(&bodyUrl, "url", "", "Updated URL")

	return cmd
}
