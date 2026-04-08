package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newTriageCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "triage",
		Short: "Triage workflow: list unassigned issues, claim, snooze",
	}

	cmd.AddCommand(newTriageListCmd(flags))
	cmd.AddCommand(newTriageClaimCmd(flags))

	return cmd
}

func newTriageListCmd(flags *rootFlags) *cobra.Command {
	var teamID string
	var first int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List unassigned/triage issues",
		Example: `  linear-pp-cli triage list --teamid <team-uuid>
  linear-pp-cli triage list --first 20 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `query($first: Int, $filter: IssueFilter) {
				issues(first: $first, filter: $filter) {
					nodes {
						id identifier title priority createdAt
						state { id name type }
						team { id name }
						labels { nodes { id name } }
					}
				}
			}`

			filter := map[string]any{
				"assignee": map[string]any{"null": true},
			}
			if teamID != "" {
				filter["team"] = map[string]any{"id": map[string]any{"eq": teamID}}
			}

			variables := map[string]any{
				"first":  first,
				"filter": filter,
			}

			data, err := c.GraphQL(query, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				Issues struct {
					Nodes json.RawMessage `json:"nodes"`
				} `json:"issues"`
			}
			if json.Unmarshal(data, &resp) != nil {
				return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.Issues.Nodes, flags)
		},
	}

	cmd.Flags().StringVar(&teamID, "teamid", "", "Filter by team ID")
	cmd.Flags().IntVar(&first, "first", 50, "Number of issues to return")
	return cmd
}

func newTriageClaimCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim <issue-id>",
		Short: "Assign an issue to yourself and set to In Progress",
		Example: `  linear-pp-cli triage claim LIN-123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Get current user ID
			meQuery := `{ viewer { id } }`
			meData, err := c.GraphQL(meQuery, nil)
			if err != nil {
				return classifyAPIError(err)
			}
			var me struct {
				Viewer struct {
					ID string `json:"id"`
				} `json:"viewer"`
			}
			if json.Unmarshal(meData, &me) != nil {
				return fmt.Errorf("parsing viewer response")
			}

			// Update issue
			mutation := `mutation($id: String!, $input: IssueUpdateInput!) {
				issueUpdate(id: $id, input: $input) {
					success
					issue { id identifier title url }
				}
			}`

			variables := map[string]any{
				"id": args[0],
				"input": map[string]any{
					"assigneeId": me.Viewer.ID,
				},
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
		},
	}

	return cmd
}
