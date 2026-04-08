// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func newLabelsListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List issue labels",
		Example: "  linear-pp-cli labels list",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `{
				issueLabels(first: 250) {
					nodes { id name color }
					pageInfo { hasNextPage endCursor }
				}
			}`

			nodes, err := c.GraphQLPaginated(query, nil, "issueLabels")
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
