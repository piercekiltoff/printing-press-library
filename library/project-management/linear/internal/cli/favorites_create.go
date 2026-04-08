// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newFavoritesCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyIssueId string
	var bodyProjectId string
	var bodyCycleId string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a favorite",
		Example: "  linear-pp-cli favorites create --issueid abc-123",
		RunE: func(cmd *cobra.Command, args []string) error {
			if bodyIssueId == "" && bodyProjectId == "" && bodyCycleId == "" {
				return usageErr(fmt.Errorf("one of --issueid, --projectid, or --cycleid is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($input: FavoriteCreateInput!) {
				favoriteCreate(input: $input) {
					success
					favorite { id }
				}
			}`

			input := map[string]any{}
			if bodyIssueId != "" {
				input["issueId"] = bodyIssueId
			}
			if bodyProjectId != "" {
				input["projectId"] = bodyProjectId
			}
			if bodyCycleId != "" {
				input["cycleId"] = bodyCycleId
			}

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				FavoriteCreate struct {
					Success  bool            `json:"success"`
					Favorite json.RawMessage `json:"favorite"`
				} `json:"favoriteCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.FavoriteCreate.Success {
				return apiErr(fmt.Errorf("favorite creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.FavoriteCreate.Favorite, flags)
		},
	}
	cmd.Flags().StringVar(&bodyIssueId, "issueid", "", "Issue ID to favorite")
	cmd.Flags().StringVar(&bodyProjectId, "projectid", "", "Project ID to favorite")
	cmd.Flags().StringVar(&bodyCycleId, "cycleid", "", "Cycle ID to favorite")

	return cmd
}
