package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mvanhorn/printing-press-library/library/project-management/linear/internal/store"
	"github.com/spf13/cobra"
)

func newSearchCmd(flags *rootFlags) *cobra.Command {
	var scope string
	var dbPath string
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Cross-entity full-text search across local data",
		Long: `Search across issues, documents, projects, labels, and more
using FTS5 full-text search on locally synced data.`,
		Example: `  linear-pp-cli search "auth timeout"
  linear-pp-cli search "deploy" --scope issues
  linear-pp-cli search "onboarding" --scope all --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".config", "linear-pp-cli", "store.db")
			}
			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			type searchResult struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}

			var results []searchResult

			searchFn := func(typeName string, fn func(string, int) ([]json.RawMessage, error)) {
				items, err := fn(query, limit)
				if err != nil {
					return
				}
				for _, item := range items {
					results = append(results, searchResult{Type: typeName, Data: item})
				}
			}

			switch scope {
			case "issues":
				searchFn("issue", db.SearchIssues)
			case "documents", "docs":
				searchFn("document", db.SearchDocuments)
			case "projects":
				searchFn("project", db.SearchProjects)
			case "labels":
				searchFn("label", db.SearchLabels)
			default: // "all"
				searchFn("issue", db.SearchIssues)
				searchFn("document", db.SearchDocuments)
				searchFn("project", db.SearchProjects)
				searchFn("label", db.SearchLabels)
				searchFn("team", db.SearchTeams)

				// Also search the generic resources_fts
				generic, err := db.Search(query, limit)
				if err == nil {
					for _, item := range generic {
						results = append(results, searchResult{Type: "other", Data: item})
					}
				}
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			if len(results) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No results found.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Found %d results for \"%s\":\n\n", len(results), query)
			for _, r := range results {
				var obj map[string]any
				if json.Unmarshal(r.Data, &obj) != nil {
					continue
				}
				id := firstNonEmpty(
					fmt.Sprintf("%v", obj["identifier"]),
					fmt.Sprintf("%v", obj["id"]),
				)
				title := firstNonEmpty(
					fmt.Sprintf("%v", obj["title"]),
					fmt.Sprintf("%v", obj["name"]),
				)
				fmt.Fprintf(cmd.OutOrStdout(), "  [%-10s] %-12s %s\n",
					r.Type, id, truncate(title, 50))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "all", "Search scope: all, issues, documents, projects, labels")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max results per type")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}
