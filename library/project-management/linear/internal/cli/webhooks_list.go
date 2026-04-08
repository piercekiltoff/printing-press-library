// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newWebhooksListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List webhooks",
		Example: "  linear-pp-cli webhooks list",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `{ webhooks { nodes { id url label enabled createdAt } } }`

			data, err := c.GraphQL(query, nil)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				Webhooks struct {
					Nodes json.RawMessage `json:"nodes"`
				} `json:"webhooks"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			result := resp.Webhooks.Nodes
			if result == nil {
				result = json.RawMessage("[]")
			}

			return printOutputWithFlags(cmd.OutOrStdout(), result, flags)
		},
	}

	return cmd
}
