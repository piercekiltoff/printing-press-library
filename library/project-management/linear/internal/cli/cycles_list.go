// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newCyclesListCmd(flags *rootFlags) *cobra.Command {
	var flagTeamId string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List cycles",
		Example: "  linear-pp-cli cycles list\n  linear-pp-cli cycles list --teamid abc",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			var query string
			if flagTeamId != "" {
				query = fmt.Sprintf(`{
					cycles(first: 50, filter: { team: { id: { eq: "%s" } } }) {
						nodes {
							id number name startsAt endsAt completedAt
							team { id name }
						}
						pageInfo { hasNextPage endCursor }
					}
				}`, flagTeamId)
			} else {
				query = `{
					cycles(first: 50) {
						nodes {
							id number name startsAt endsAt completedAt
							team { id name }
						}
						pageInfo { hasNextPage endCursor }
					}
				}`
			}

			nodes, err := c.GraphQLPaginated(query, nil, "cycles")
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
	cmd.Flags().StringVar(&flagTeamId, "teamid", "", "Filter by team ID")

	return cmd
}
