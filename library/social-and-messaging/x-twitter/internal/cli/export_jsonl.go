// Copyright 2026 dinakar-sarbada. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// exportJsonlSummary is what the command emits as JSON.
type exportJsonlSummary struct {
	Resource    string            `json:"resource"`
	OutputDir   string            `json:"output_dir"`
	Yearly      bool              `json:"yearly"`
	FilesWritten []string         `json:"files_written"`
	RowsByYear  map[string]int    `json:"rows_by_year"`
	TotalRows   int               `json:"total_rows"`
}

func newExportJsonlCmd(flags *rootFlags) *cobra.Command {
	var resource, outputDir string
	var yearly bool
	cmd := &cobra.Command{
		Use:   "jsonl",
		Short: "Export local data as Git-friendly JSONL files (yearly shards optional)",
		Long: strings.Trim(`
Write the contents of a local table to JSONL files. Useful for version-controlled
backups, agent ingestion, or migrating between CLIs.

Resources supported: tweets, users, follows
`, "\n"),
		Example: strings.Trim(`
  x-twitter-pp-cli export jsonl --resource tweets --output ./backup --yearly
  x-twitter-pp-cli export jsonl --resource follows --output ./backup
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			db, err := openXStore(flags)
			if err != nil {
				return err
			}
			defer db.Close()
			if err := os.MkdirAll(outputDir, 0o755); err != nil {
				return fmt.Errorf("creating output dir: %w", err)
			}

			summary := &exportJsonlSummary{
				Resource:    resource,
				OutputDir:   outputDir,
				Yearly:      yearly,
				FilesWritten: []string{},
				RowsByYear:  map[string]int{},
			}

			var query, dateCol string
			switch resource {
			case "tweets":
				query = `SELECT tweet_id, author_handle, full_text, lang, like_count, retweet_count, reply_count, COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', created_at), '') AS created_at FROM x_tweets ORDER BY created_at`
				dateCol = "created_at"
			case "users":
				query = `SELECT user_id, handle, display_name, bio, followers_count, following_count, tweet_count, COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', account_created_at), '') AS created_at FROM x_users ORDER BY handle`
				dateCol = "created_at"
			case "follows":
				query = `SELECT account_handle, direction, user_id, handle, COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', scraped_at), '') AS scraped_at FROM x_follows ORDER BY scraped_at`
				dateCol = "scraped_at"
			default:
				return fmt.Errorf("unsupported resource %q (expected: tweets, users, follows)", resource)
			}
			rows, err := db.DB().Query(query)
			if err != nil {
				return fmt.Errorf("query: %w", err)
			}
			defer rows.Close()
			cols, err := rows.Columns()
			if err != nil {
				return err
			}

			yearFiles := map[string]*os.File{}
			defer func() {
				for _, f := range yearFiles {
					f.Close()
				}
			}()

			openFile := func(year string) (*os.File, error) {
				if f, ok := yearFiles[year]; ok {
					return f, nil
				}
				name := fmt.Sprintf("%s.jsonl", resource)
				if yearly && year != "" {
					name = fmt.Sprintf("%s.%s.jsonl", resource, year)
				}
				path := filepath.Join(outputDir, name)
				f, err := os.Create(path)
				if err != nil {
					return nil, err
				}
				yearFiles[year] = f
				summary.FilesWritten = append(summary.FilesWritten, path)
				return f, nil
			}

			for rows.Next() {
				vals := make([]any, len(cols))
				ptrs := make([]any, len(cols))
				for i := range vals {
					ptrs[i] = &vals[i]
				}
				if err := rows.Scan(ptrs...); err != nil {
					return fmt.Errorf("scanning row: %w", err)
				}
				m := make(map[string]any, len(cols))
				var dateStr string
				for i, name := range cols {
					m[name] = vals[i]
					if name == dateCol {
						if s, ok := vals[i].(string); ok {
							dateStr = s
						}
					}
				}
				year := ""
				if yearly && dateStr != "" {
					if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
						year = fmt.Sprintf("%04d", t.Year())
					}
				}
				f, err := openFile(year)
				if err != nil {
					return err
				}
				b, err := json.Marshal(m)
				if err != nil {
					continue
				}
				if _, err := f.Write(append(b, '\n')); err != nil {
					return err
				}
				summary.RowsByYear[year]++
				summary.TotalRows++
			}

			w := cmd.OutOrStdout()
			if flags.asJSON || !isTerminal(w) {
				return printJSONFiltered(w, summary, flags)
			}
			fmt.Fprintf(w, "Exported %d rows of %s to %s\n", summary.TotalRows, resource, outputDir)
			for _, path := range summary.FilesWritten {
				fmt.Fprintf(w, "  %s\n", path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&resource, "resource", "tweets", "Resource to export (tweets, users, follows)")
	cmd.Flags().StringVar(&outputDir, "output", "./x-twitter-export", "Output directory for JSONL files")
	cmd.Flags().BoolVar(&yearly, "yearly", false, "Partition output into yearly shards (e.g. tweets.2026.jsonl)")
	return cmd
}
