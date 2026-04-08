// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func newNotificationsListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List notifications for the authenticated user",
		Example: "  linear-pp-cli notifications list",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `{
				notifications(first: 50) {
					nodes {
						id type readAt createdAt
						issue { id identifier title }
					}
					pageInfo { hasNextPage endCursor }
				}
			}`

			nodes, err := c.GraphQLPaginated(query, nil, "notifications")
			if err != nil {
				return classifyAPIError(err)
			}

			if len(nodes) == 0 {
				return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage("[]"), flags)
			}

			data, _ := json.Marshal(nodes)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(data), flags)
		},
	}

	return cmd
}
