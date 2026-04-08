// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/mvanhorn/printing-press-library/library/project-management/linear/internal/store"
	"github.com/spf13/cobra"
)

func newIssuesSearchCmd(flags *rootFlags) *cobra.Command {
	var flagFirst string
	var dbPath string

	cmd := &cobra.Command{
		Use:     "search <term>",
		Short:   "Search issues by term",
		Example: "  linear-pp-cli issues search \"login bug\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return usageErr(fmt.Errorf("term is required\nUsage: %s %s <%s>", cmd.Root().Name(), cmd.CommandPath(), "term"))
			}
			searchTerm := args[0]

			limit := 50
			if flagFirst != "" {
				var n int
				if _, scanErr := fmt.Sscanf(flagFirst, "%d", &n); scanErr == nil && n > 0 {
					limit = n
				}
			}

			// Try local FTS search first if store exists
			if dbPath == "" {
				dbPath = defaultDBPath("linear-pp-cli")
			}
			s, storeErr := store.Open(dbPath)
			if storeErr == nil {
				defer s.Close()
				results, err := s.SearchIssues(searchTerm, limit)
				if err == nil && len(results) > 0 {
					fmt.Fprintf(cmd.ErrOrStderr(), "Found %d results from local store\n", len(results))
					data, _ := json.Marshal(results)
					return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(data), flags)
				}
			}

			// Fall back to API search
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `query($query: String!) {
				searchIssues(query: $query, first: 50) {
					nodes {
						id identifier title description
						state { name }
						assignee { name }
					}
				}
			}`

			variables := map[string]any{
				"query": searchTerm,
			}

			nodes, err := c.GraphQLPaginated(query, variables, "searchIssues")
			if err != nil {
				return classifyAPIError(err)
			}

			if len(nodes) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "No results found")
				fmt.Fprintln(cmd.OutOrStdout(), "[]")
				return nil
			}

			data, _ := json.Marshal(nodes)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(data), flags)
		},
	}
	cmd.Flags().StringVar(&flagFirst, "first", "", "Number of results to return")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path for local search")

	return cmd
}
