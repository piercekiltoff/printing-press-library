// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newIssuesListCmd(flags *rootFlags) *cobra.Command {
	var flagFirst string
	var flagTeamId string
	var flagAssigneeId string
	var flagStateId string
	var flagLabelId string
	var flagProjectId string
	var flagCycleId string
	var flagPriority string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List issues with optional filtering",
		Example: "  linear-pp-cli issues list\n  linear-pp-cli issues list --teamid abc --priority 1",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Build the filter object for Linear's GraphQL filter syntax
			var filterParts []string
			if flagTeamId != "" {
				filterParts = append(filterParts, fmt.Sprintf(`team: { id: { eq: "%s" } }`, flagTeamId))
			}
			if flagAssigneeId != "" {
				filterParts = append(filterParts, fmt.Sprintf(`assignee: { id: { eq: "%s" } }`, flagAssigneeId))
			}
			if flagStateId != "" {
				filterParts = append(filterParts, fmt.Sprintf(`state: { id: { eq: "%s" } }`, flagStateId))
			}
			if flagLabelId != "" {
				filterParts = append(filterParts, fmt.Sprintf(`labels: { id: { eq: "%s" } }`, flagLabelId))
			}
			if flagProjectId != "" {
				filterParts = append(filterParts, fmt.Sprintf(`project: { id: { eq: "%s" } }`, flagProjectId))
			}
			if flagCycleId != "" {
				filterParts = append(filterParts, fmt.Sprintf(`cycle: { id: { eq: "%s" } }`, flagCycleId))
			}
			if flagPriority != "" {
				if p, err := strconv.Atoi(flagPriority); err == nil {
					filterParts = append(filterParts, fmt.Sprintf(`priority: { eq: %d }`, p))
				}
			}

			first := 50
			if flagFirst != "" {
				if n, err := strconv.Atoi(flagFirst); err == nil && n > 0 {
					first = n
				}
			}

			filterStr := ""
			if len(filterParts) > 0 {
				filterStr = fmt.Sprintf(", filter: { %s }", strings.Join(filterParts, ", "))
			}

			query := fmt.Sprintf(`{
				issues(first: %d%s) {
					nodes {
						id identifier title priority estimate dueDate
						createdAt updatedAt
						team { id name }
						assignee { id name }
						state { id name }
						project { id name }
						cycle { id number }
						parent { id identifier }
						labels { nodes { id name } }
					}
					pageInfo { hasNextPage endCursor }
				}
			}`, first, filterStr)

			data, err := c.GraphQL(query, nil)
			if err != nil {
				return classifyAPIError(err)
			}

			// Extract the nodes array from data.issues.nodes
			var resp struct {
				Issues struct {
					Nodes json.RawMessage `json:"nodes"`
				} `json:"issues"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			result := resp.Issues.Nodes
			if result == nil {
				result = json.RawMessage("[]")
			}

			return printOutputWithFlags(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().StringVar(&flagFirst, "first", "", "Number of issues to return (default 50)")
	cmd.Flags().StringVar(&flagTeamId, "teamid", "", "Filter by team ID")
	cmd.Flags().StringVar(&flagAssigneeId, "assigneeid", "", "Filter by assignee ID")
	cmd.Flags().StringVar(&flagStateId, "stateid", "", "Filter by workflow state ID")
	cmd.Flags().StringVar(&flagLabelId, "labelid", "", "Filter by label ID")
	cmd.Flags().StringVar(&flagProjectId, "projectid", "", "Filter by project ID")
	cmd.Flags().StringVar(&flagCycleId, "cycleid", "", "Filter by cycle ID")
	cmd.Flags().StringVar(&flagPriority, "priority", "", "Filter by priority (0=none, 1=urgent, 2=high, 3=medium, 4=low)")

	return cmd
}
