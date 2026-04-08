package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mvanhorn/printing-press-library/library/project-management/linear/internal/store"
	"github.com/spf13/cobra"
)

func newSimilarCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var limit int

	cmd := &cobra.Command{
		Use:   "similar <text>",
		Short: "Find potentially duplicate issues using FTS5 search",
		Long: `Search locally synced issues for potential duplicates using
full-text search. Run before creating a new issue to avoid duplicates.`,
		Example: `  linear-pp-cli similar "login page broken"
  linear-pp-cli similar "auth timeout" --limit 10 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".config", "linear-pp-cli", "store.db")
			}
			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w\nRun 'workflow archive' first.", err)
			}
			defer db.Close()

			results, err := db.SearchIssues(query, limit)
			if err != nil {
				return fmt.Errorf("searching issues: %w", err)
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			if len(results) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No similar issues found.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Found %d similar issues:\n\n", len(results))
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %-45s %-15s\n", "ID", "TITLE", "STATE")
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %-45s %-15s\n", "----", "-----", "-----")

			for _, raw := range results {
				var obj map[string]any
				if json.Unmarshal(raw, &obj) != nil {
					continue
				}
				ident, _ := obj["identifier"].(string)
				title, _ := obj["title"].(string)
				stateName := ""
				if s, ok := obj["state"].(map[string]any); ok {
					stateName, _ = s["name"].(string)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %-45s %-15s\n",
					ident, truncate(title, 45), stateName)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 10, "Max results to show")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}
