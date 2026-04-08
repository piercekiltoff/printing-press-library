// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newWorkflowStatesListCmd(flags *rootFlags) *cobra.Command {
	var flagTeamId string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List workflow states",
		Example: "  linear-pp-cli workflow_states list\n  linear-pp-cli workflow_states list --teamid abc",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			var query string
			if flagTeamId != "" {
				query = fmt.Sprintf(`{
					workflowStates(first: 250, filter: { team: { id: { eq: "%s" } } }) {
						nodes {
							id name color type position
							team { id name }
						}
						pageInfo { hasNextPage endCursor }
					}
				}`, flagTeamId)
			} else {
				query = `{
					workflowStates(first: 250) {
						nodes {
							id name color type position
							team { id name }
						}
						pageInfo { hasNextPage endCursor }
					}
				}`
			}

			nodes, err := c.GraphQLPaginated(query, nil, "workflowStates")
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
