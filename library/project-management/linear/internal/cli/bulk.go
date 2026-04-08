package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newBulkCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk",
		Short: "Bulk operations on issues: update-state, assign, label",
	}

	cmd.AddCommand(newBulkUpdateStateCmd(flags))
	cmd.AddCommand(newBulkAssignCmd(flags))
	cmd.AddCommand(newBulkLabelCmd(flags))

	return cmd
}

func newBulkUpdateStateCmd(flags *rootFlags) *cobra.Command {
	var stateID string

	cmd := &cobra.Command{
		Use:   "update-state <issue-ids...>",
		Short: "Batch-update workflow state for multiple issues",
		Long: `Update the workflow state of multiple issues at once.
Issue IDs can be passed as args or piped via stdin (one per line).`,
		Example: `  linear-pp-cli bulk update-state --stateid <state-uuid> ID1 ID2 ID3
  echo "ID1\nID2" | linear-pp-cli bulk update-state --stateid <state-uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if stateID == "" {
				return usageErr(fmt.Errorf("--stateid is required"))
			}

			ids := collectIDs(args)
			if len(ids) == 0 {
				return usageErr(fmt.Errorf("provide issue IDs as arguments or via stdin"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: IssueUpdateInput!) {
				issueUpdate(id: $id, input: $input) { success issue { id identifier } }
			}`

			return bulkMutate(cmd, c, flags, mutation, ids, map[string]any{"stateId": stateID})
		},
	}

	cmd.Flags().StringVar(&stateID, "stateid", "", "Target workflow state ID (required)")
	return cmd
}

func newBulkAssignCmd(flags *rootFlags) *cobra.Command {
	var assigneeID string

	cmd := &cobra.Command{
		Use:   "assign <issue-ids...>",
		Short: "Batch-assign multiple issues to a user",
		Example: `  linear-pp-cli bulk assign --assigneeid <user-uuid> ID1 ID2 ID3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if assigneeID == "" {
				return usageErr(fmt.Errorf("--assigneeid is required"))
			}

			ids := collectIDs(args)
			if len(ids) == 0 {
				return usageErr(fmt.Errorf("provide issue IDs as arguments or via stdin"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: IssueUpdateInput!) {
				issueUpdate(id: $id, input: $input) { success issue { id identifier } }
			}`

			return bulkMutate(cmd, c, flags, mutation, ids, map[string]any{"assigneeId": assigneeID})
		},
	}

	cmd.Flags().StringVar(&assigneeID, "assigneeid", "", "Target assignee user ID (required)")
	return cmd
}

func newBulkLabelCmd(flags *rootFlags) *cobra.Command {
	var labelIDs []string

	cmd := &cobra.Command{
		Use:   "label <issue-ids...>",
		Short: "Batch-add labels to multiple issues",
		Example: `  linear-pp-cli bulk label --labelid <uuid> --labelid <uuid2> ID1 ID2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(labelIDs) == 0 {
				return usageErr(fmt.Errorf("at least one --labelid is required"))
			}

			ids := collectIDs(args)
			if len(ids) == 0 {
				return usageErr(fmt.Errorf("provide issue IDs as arguments or via stdin"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($id: String!, $input: IssueUpdateInput!) {
				issueUpdate(id: $id, input: $input) { success issue { id identifier } }
			}`

			return bulkMutate(cmd, c, flags, mutation, ids, map[string]any{"labelIds": labelIDs})
		},
	}

	cmd.Flags().StringArrayVar(&labelIDs, "labelid", nil, "Label ID to add (repeatable)")
	return cmd
}

// collectIDs gathers issue IDs from args and stdin.
func collectIDs(args []string) []string {
	ids := make([]string, 0, len(args))
	for _, a := range args {
		a = strings.TrimSpace(a)
		if a != "" {
			ids = append(ids, a)
		}
	}

	// Also read from stdin if piped
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				ids = append(ids, line)
			}
		}
	}

	return ids
}

// bulkMutate runs a mutation for each issue ID with the given input fields.
func bulkMutate(cmd *cobra.Command, c interface {
	GraphQL(string, map[string]any) (json.RawMessage, error)
}, flags *rootFlags, mutation string, ids []string, inputFields map[string]any) error {
	var results []map[string]any
	for _, id := range ids {
		variables := map[string]any{
			"id":    id,
			"input": inputFields,
		}

		data, err := c.GraphQL(mutation, variables)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: error: %v\n", id, err)
			results = append(results, map[string]any{"id": id, "success": false, "error": err.Error()})
			continue
		}

		fmt.Fprintf(os.Stderr, "  %s: updated\n", id)
		var parsed any
		json.Unmarshal(data, &parsed)
		results = append(results, map[string]any{"id": id, "success": true, "data": parsed})
	}

	if flags.asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Updated %d issues\n", len(ids))
	return nil
}
