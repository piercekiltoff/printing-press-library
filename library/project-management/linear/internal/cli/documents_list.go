// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newDocumentsListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List documents",
		Example: "  linear-pp-cli documents list",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `{
				documents(first: 50) {
					nodes {
						id title slugId createdAt updatedAt
						project { id name }
					}
					pageInfo { hasNextPage endCursor }
				}
			}`

			data, err := c.GraphQL(query, nil)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				Documents struct {
					Nodes json.RawMessage `json:"nodes"`
				} `json:"documents"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			result := resp.Documents.Nodes
			if result == nil {
				result = json.RawMessage("[]")
			}

			return printOutputWithFlags(cmd.OutOrStdout(), result, flags)
		},
	}

	return cmd
}
