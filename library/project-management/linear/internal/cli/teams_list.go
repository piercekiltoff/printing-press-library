// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newTeamsListCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all teams",
		Example: "  linear-pp-cli teams list",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `{ teams { nodes { id name key description } } }`

			data, err := c.GraphQL(query, nil)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				Teams struct {
					Nodes json.RawMessage `json:"nodes"`
				} `json:"teams"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			result := resp.Teams.Nodes
			if result == nil {
				result = json.RawMessage("[]")
			}

			return printOutputWithFlags(cmd.OutOrStdout(), result, flags)
		},
	}

	return cmd
}
