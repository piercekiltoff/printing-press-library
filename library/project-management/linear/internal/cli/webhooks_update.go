// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newWebhooksUpdateCmd(flags *rootFlags) *cobra.Command {
	var bodyUrl string
	var bodyLabel string
	var bodyEnabled string

	cmd := &cobra.Command{
		Use:     "update <id>",
		Short:   "Update a webhook",
		Example: `  linear-pp-cli webhooks update 550e8400-... --url "https://new-url.com/hook"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("id is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "id"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: WebhookUpdateInput!) {
				webhookUpdate(id: $id, input: $input) {
					success
					webhook { id url }
				}
			}`

			input := map[string]any{}
			if bodyUrl != "" {
				input["url"] = bodyUrl
			}
			if bodyLabel != "" {
				input["label"] = bodyLabel
			}
			if bodyEnabled != "" {
				input["enabled"] = bodyEnabled == "true"
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
				WebhookUpdate struct {
					Success bool            `json:"success"`
					Webhook json.RawMessage `json:"webhook"`
				} `json:"webhookUpdate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.WebhookUpdate.Success {
				return apiErr(fmt.Errorf("webhook update failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.WebhookUpdate.Webhook, flags)
		},
	}
	cmd.Flags().StringVar(&bodyUrl, "url", "", "Updated callback URL")
	cmd.Flags().StringVar(&bodyLabel, "label", "", "Updated label")
	cmd.Flags().StringVar(&bodyEnabled, "enabled", "", "Updated enabled status (true/false)")

	return cmd
}
