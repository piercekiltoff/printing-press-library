// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newCyclesCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyTeamId string
	var bodyName string
	var bodyStartsAt string
	var bodyEndsAt string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new cycle",
		Example: "  linear-pp-cli cycles create --teamid abc --startsat 2026-04-01 --endsat 2026-04-14",
		RunE: func(cmd *cobra.Command, args []string) error {
			if bodyTeamId == "" {
				return usageErr(fmt.Errorf("--teamid is required"))
			}
			if bodyStartsAt == "" {
				return usageErr(fmt.Errorf("--startsat is required"))
			}
			if bodyEndsAt == "" {
				return usageErr(fmt.Errorf("--endsat is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($input: CycleCreateInput!) {
				cycleCreate(input: $input) {
					success
					cycle { id number }
				}
			}`

			input := map[string]any{
				"teamId":   bodyTeamId,
				"startsAt": bodyStartsAt,
				"endsAt":   bodyEndsAt,
			}
			if bodyName != "" {
				input["name"] = bodyName
			}

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				CycleCreate struct {
					Success bool            `json:"success"`
					Cycle   json.RawMessage `json:"cycle"`
				} `json:"cycleCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.CycleCreate.Success {
				return apiErr(fmt.Errorf("cycle creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.CycleCreate.Cycle, flags)
		},
	}
	cmd.Flags().StringVar(&bodyTeamId, "teamid", "", "Team ID (required)")
	cmd.Flags().StringVar(&bodyName, "name", "", "Cycle name")
	cmd.Flags().StringVar(&bodyStartsAt, "startsat", "", "Cycle start date in ISO 8601 format (required)")
	cmd.Flags().StringVar(&bodyEndsAt, "endsat", "", "Cycle end date in ISO 8601 format (required)")

	return cmd
}
