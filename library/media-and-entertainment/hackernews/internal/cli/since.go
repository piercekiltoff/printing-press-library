package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

func newSinceCmd(flags *rootFlags) *cobra.Command {
	var flagLimit int
	var flagMinPoints int
	var flagSort string

	cmd := &cobra.Command{
		Use:   "since <duration>",
		Short: "What hit HN while you were away — stories from the last N hours/days",
		Long: `Show stories posted in the last N hours or days, sorted by points.
Duration format: 1h, 2h, 24h, 7d, 30d`,
		Example: `  # Stories from the last 2 hours
  hackernews-pp-cli since 2h

  # Last 24 hours, minimum 100 points
  hackernews-pp-cli since 24h --min-points 100

  # Last 7 days, top 10
  hackernews-pp-cli since 7d --limit 10

  # JSON output for piping
  hackernews-pp-cli since 24h --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			dur, err := parseDuration(args[0])
			if err != nil {
				return usageErr(err)
			}

			ts := sinceTimestamp(dur)
			params := map[string]string{
				"tags":           "story",
				"numericFilters": fmt.Sprintf("created_at_i>%d", ts),
				"hitsPerPage":    "200",
			}

			data, err := algoliaGet("/search_by_date", params)
			if err != nil {
				return apiErr(err)
			}

			hits, err := algoliaHits(data)
			if err != nil {
				return apiErr(err)
			}

			// Filter by min points
			if flagMinPoints > 0 {
				var filtered []map[string]any
				for _, h := range hits {
					if getInt(h, "points") >= flagMinPoints {
						filtered = append(filtered, h)
					}
				}
				hits = filtered
			}

			// Sort
			switch flagSort {
			case "comments":
				sort.Slice(hits, func(i, j int) bool {
					return getInt(hits[i], "num_comments") > getInt(hits[j], "num_comments")
				})
			case "date":
				sort.Slice(hits, func(i, j int) bool {
					return getInt(hits[i], "created_at_i") > getInt(hits[j], "created_at_i")
				})
			case "velocity":
				sort.Slice(hits, func(i, j int) bool {
					vi := velocity(getInt(hits[i], "points"), time.Unix(int64(getInt(hits[i], "created_at_i")), 0))
					vj := velocity(getInt(hits[j], "points"), time.Unix(int64(getInt(hits[j], "created_at_i")), 0))
					return vi > vj
				})
			default: // "points"
				sort.Slice(hits, func(i, j int) bool {
					return getInt(hits[i], "points") > getInt(hits[j], "points")
				})
			}

			// Apply limit
			if flagLimit > 0 && len(hits) > flagLimit {
				hits = hits[:flagLimit]
			}

			if len(hits) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "No stories found in the last %s\n", args[0])
				return nil
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "%d stories in the last %s\n", len(hits), args[0])

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return outputJSON(cmd, flags, hits)
			}

			// Table output
			rows := make([][]string, 0, len(hits))
			for i, h := range hits {
				created := time.Unix(int64(getInt(h, "created_at_i")), 0)
				rows = append(rows, []string{
					fmt.Sprintf("%d", i+1),
					truncate(getString(h, "title"), 55),
					fmt.Sprintf("%d", getInt(h, "points")),
					fmt.Sprintf("%d", getInt(h, "num_comments")),
					timeAgo(created),
					extractDomain(getString(h, "url")),
				})
			}
			return flags.printTable(cmd, []string{"#", "TITLE", "PTS", "CMTS", "AGE", "DOMAIN"}, rows)
		},
	}

	cmd.Flags().IntVar(&flagLimit, "limit", 30, "Maximum stories to show")
	cmd.Flags().IntVar(&flagMinPoints, "min-points", 0, "Minimum points threshold")
	cmd.Flags().StringVar(&flagSort, "sort", "points", "Sort by: points, comments, date, velocity")

	return cmd
}

// outputJSON marshals hits to JSON and passes through flag filters.
func outputJSON(cmd *cobra.Command, flags *rootFlags, hits []map[string]any) error {
	data, err := json.Marshal(hits)
	if err != nil {
		return err
	}
	raw := json.RawMessage(data)
	if flags.compact {
		raw = compactFields(raw)
	}
	if flags.selectFields != "" {
		raw = filterFields(raw, flags.selectFields)
	}
	return printOutput(cmd.OutOrStdout(), raw, true)
}
