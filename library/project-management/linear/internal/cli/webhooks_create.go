// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newWebhooksCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyUrl string
	var bodyTeamId string
	var bodyLabel string
	var bodyResourceTypes []string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a webhook",
		Example: `  linear-pp-cli webhooks create --url "https://example.com/hook" --resourcetypes Issue --resourcetypes Comment`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if bodyUrl == "" {
				return usageErr(fmt.Errorf("--url is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($input: WebhookCreateInput!) {
				webhookCreate(input: $input) {
					success
					webhook { id url }
				}
			}`

			input := map[string]any{
				"url": bodyUrl,
			}
			if bodyTeamId != "" {
				input["teamId"] = bodyTeamId
			}
			if bodyLabel != "" {
				input["label"] = bodyLabel
			}
			if len(bodyResourceTypes) > 0 {
				input["resourceTypes"] = bodyResourceTypes
			}

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				WebhookCreate struct {
					Success bool            `json:"success"`
					Webhook json.RawMessage `json:"webhook"`
				} `json:"webhookCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.WebhookCreate.Success {
				return apiErr(fmt.Errorf("webhook creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.WebhookCreate.Webhook, flags)
		},
	}
	cmd.Flags().StringVar(&bodyUrl, "url", "", "Webhook callback URL (required)")
	cmd.Flags().StringVar(&bodyTeamId, "teamid", "", "Team ID to scope webhook to")
	cmd.Flags().StringVar(&bodyLabel, "label", "", "Webhook label")
	cmd.Flags().StringArrayVar(&bodyResourceTypes, "resourcetypes", nil, "Resource types to subscribe to (repeatable: Issue, Comment, Project, etc.)")

	return cmd
}
