// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newDocumentsUpdateCmd(flags *rootFlags) *cobra.Command {
	var bodyTitle string
	var bodyContent string

	cmd := &cobra.Command{
		Use:     "update <id>",
		Short:   "Update a document",
		Example: `  linear-pp-cli documents update 550e8400-... --title "New Title"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: DocumentUpdateInput!) {
				documentUpdate(id: $id, input: $input) {
					success
					document { id title }
				}
			}`

			input := map[string]any{}
			if bodyTitle != "" {
				input["title"] = bodyTitle
			}
			if bodyContent != "" {
				input["content"] = bodyContent
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
				DocumentUpdate struct {
					Success  bool            `json:"success"`
					Document json.RawMessage `json:"document"`
				} `json:"documentUpdate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.DocumentUpdate.Success {
				return apiErr(fmt.Errorf("document update failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.DocumentUpdate.Document, flags)
		},
	}
	cmd.Flags().StringVar(&bodyTitle, "title", "", "Updated title")
	cmd.Flags().StringVar(&bodyContent, "content", "", "Updated content")

	return cmd
}
