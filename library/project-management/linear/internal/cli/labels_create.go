// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newLabelsCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyName string
	var bodyColor string
	var bodyTeamId string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new label",
		Example: `  linear-pp-cli labels create --name "Bug" --color "#ff0000"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if bodyName == "" {
				return usageErr(fmt.Errorf("--name is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($input: IssueLabelCreateInput!) {
				issueLabelCreate(input: $input) {
					success
					issueLabel { id name color }
				}
			}`

			input := map[string]any{
				"name": bodyName,
			}
			if bodyColor != "" {
				input["color"] = bodyColor
			}
			if bodyTeamId != "" {
				input["teamId"] = bodyTeamId
			}

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				IssueLabelCreate struct {
					Success    bool            `json:"success"`
					IssueLabel json.RawMessage `json:"issueLabel"`
				} `json:"issueLabelCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.IssueLabelCreate.Success {
				return apiErr(fmt.Errorf("label creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.IssueLabelCreate.IssueLabel, flags)
		},
	}
	cmd.Flags().StringVar(&bodyName, "name", "", "Label name (required)")
	cmd.Flags().StringVar(&bodyColor, "color", "", "Label color hex code")
	cmd.Flags().StringVar(&bodyTeamId, "teamid", "", "Team ID (omit for workspace-level label)")

	return cmd
}
