// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newTeamsCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyName string
	var bodyKey string
	var bodyDescription string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new team",
		Example: "  linear-pp-cli teams create --name \"Engineering\" --key ENG",
		RunE: func(cmd *cobra.Command, args []string) error {
			if bodyName == "" {
				return usageErr(fmt.Errorf("--name is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($input: TeamCreateInput!) {
				teamCreate(input: $input) {
					success
					team { id name key }
				}
			}`

			input := map[string]any{
				"name": bodyName,
			}
			if bodyKey != "" {
				input["key"] = bodyKey
			}
			if bodyDescription != "" {
				input["description"] = bodyDescription
			}

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				TeamCreate struct {
					Success bool            `json:"success"`
					Team    json.RawMessage `json:"team"`
				} `json:"teamCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.TeamCreate.Success {
				return apiErr(fmt.Errorf("team creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.TeamCreate.Team, flags)
		},
	}
	cmd.Flags().StringVar(&bodyName, "name", "", "Team name (required)")
	cmd.Flags().StringVar(&bodyKey, "key", "", "Team key (short identifier used in issue IDs)")
	cmd.Flags().StringVar(&bodyDescription, "description", "", "Team description")

	return cmd
}
