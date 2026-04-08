// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newAttachmentsGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Short:   "Get an attachment by ID",
		Example: "  linear-pp-cli attachments get 550e8400-e29b-41d4-a716-446655440000",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `query($id: String!) {
				attachment(id: $id) { id title url subtitle createdAt }
			}`

			variables := map[string]any{
				"id": args[0],
			}

			data, err := c.GraphQL(query, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				Attachment json.RawMessage `json:"attachment"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if resp.Attachment == nil || string(resp.Attachment) == "null" {
				return notFoundErr(fmt.Errorf("attachment %s not found", args[0]))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.Attachment, flags)
		},
	}

	return cmd
}
