// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newOrganizationGetCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Get the current organization",
		Example: "  linear-pp-cli organization get",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `{ organization { id name urlKey createdAt } }`

			data, err := c.GraphQL(query, nil)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				Organization json.RawMessage `json:"organization"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if resp.Organization == nil || string(resp.Organization) == "null" {
				return notFoundErr(fmt.Errorf("organization not found"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.Organization, flags)
		},
	}

	return cmd
}
