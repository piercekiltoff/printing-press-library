package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

func newRepostCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repost <url>",
		Short: "Check if a URL has been posted before — find all prior submissions",
		Example: `  # Check if a URL was posted before
  hackernews-pp-cli repost "https://example.com/article"

  # JSON output
  hackernews-pp-cli repost "https://example.com/article" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			searchURL := args[0]

			params := map[string]string{
				"query":                        searchURL,
				"tags":                         "story",
				"restrictSearchableAttributes": "url",
				"hitsPerPage":                  "50",
			}

			data, err := algoliaGet("/search", params)
			if err != nil {
				return apiErr(err)
			}

			hits, err := algoliaHits(data)
			if err != nil {
				return apiErr(err)
			}

			if len(hits) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No prior submissions found for %s\n", searchURL)
				return nil
			}

			// Sort by date descending
			sort.Slice(hits, func(i, j int) bool {
				return getInt(hits[i], "created_at_i") > getInt(hits[j], "created_at_i")
			})

			fmt.Fprintf(cmd.ErrOrStderr(), "%d prior submissions found\n", len(hits))

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return outputJSON(cmd, flags, hits)
			}

			rows := make([][]string, 0, len(hits))
			for i, h := range hits {
				t := getInt(h, "created_at_i")
				date := ""
				if t > 0 {
					date = time.Unix(int64(t), 0).Format("2006-01-02")
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", i+1),
					truncate(getString(h, "title"), 50),
					fmt.Sprintf("%d", getInt(h, "points")),
					fmt.Sprintf("%d", getInt(h, "num_comments")),
					date,
					getString(h, "author"),
				})
			}
			return flags.printTable(cmd, []string{"#", "TITLE", "PTS", "CMTS", "DATE", "AUTHOR"}, rows)
		},
	}

	return cmd
}
