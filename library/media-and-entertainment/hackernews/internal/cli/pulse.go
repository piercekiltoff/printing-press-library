package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

func newPulseCmd(flags *rootFlags) *cobra.Command {
	var flagSince string

	cmd := &cobra.Command{
		Use:   "pulse <topic>",
		Short: "What's HN saying about a topic this week — activity timeline and stats",
		Long: `Search for a topic across recent HN stories and show aggregate stats:
total stories, points, comments, and activity by day.`,
		Example: `  # What's happening with AI this week
  hackernews-pp-cli pulse "artificial intelligence"

  # Rust activity in the last 30 days
  hackernews-pp-cli pulse rust --since 30d

  # JSON output
  hackernews-pp-cli pulse "open source" --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			topic := args[0]

			since := "7d"
			if flagSince != "" {
				since = flagSince
			}
			dur, err := parseDuration(since)
			if err != nil {
				return usageErr(err)
			}
			ts := sinceTimestamp(dur)

			params := map[string]string{
				"query":          topic,
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

			if len(hits) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "No stories about %q in the last %s\n", topic, since)
				return nil
			}

			// Aggregate stats
			totalPoints := 0
			totalComments := 0
			dayActivity := map[string]int{}
			dayPoints := map[string]int{}

			for _, h := range hits {
				pts := getInt(h, "points")
				cmts := getInt(h, "num_comments")
				totalPoints += pts
				totalComments += cmts

				t := getInt(h, "created_at_i")
				if t > 0 {
					day := time.Unix(int64(t), 0).Format("2006-01-02")
					dayActivity[day]++
					dayPoints[day] += pts
				}
			}

			// Sort days
			var days []string
			for d := range dayActivity {
				days = append(days, d)
			}
			sort.Strings(days)

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				timeline := make([]map[string]any, 0, len(days))
				for _, d := range days {
					timeline = append(timeline, map[string]any{
						"date":    d,
						"stories": dayActivity[d],
						"points":  dayPoints[d],
					})
				}
				result := map[string]any{
					"topic":          topic,
					"period":         since,
					"total_stories":  len(hits),
					"total_points":   totalPoints,
					"total_comments": totalComments,
					"timeline":       timeline,
				}
				data, _ := json.Marshal(result)
				return printOutput(cmd.OutOrStdout(), json.RawMessage(data), true)
			}

			// Human output
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s %q (last %s)\n\n", bold("Pulse:"), topic, since)
			fmt.Fprintf(cmd.OutOrStdout(), "  Stories:    %d\n", len(hits))
			fmt.Fprintf(cmd.OutOrStdout(), "  Points:     %d\n", totalPoints)
			fmt.Fprintf(cmd.OutOrStdout(), "  Comments:   %d\n", totalComments)

			avgPts := float64(totalPoints) / float64(len(hits))
			fmt.Fprintf(cmd.OutOrStdout(), "  Avg points: %.1f\n", avgPts)

			// Timeline
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", bold("Activity Timeline"))

			// Find max for bar chart
			maxCount := 0
			for _, c := range dayActivity {
				if c > maxCount {
					maxCount = c
				}
			}

			barWidth := 30
			for _, d := range days {
				count := dayActivity[d]
				pts := dayPoints[d]
				barLen := 0
				if maxCount > 0 {
					barLen = count * barWidth / maxCount
				}
				if barLen < 1 && count > 0 {
					barLen = 1
				}
				bar := ""
				for i := 0; i < barLen; i++ {
					bar += "█"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %s  %s %d stories (%d pts)\n", d, bar, count, pts)
			}

			// Top stories
			sort.Slice(hits, func(i, j int) bool {
				return getInt(hits[i], "points") > getInt(hits[j], "points")
			})
			top := hits
			if len(top) > 5 {
				top = top[:5]
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", bold("Top Stories"))
			for i, h := range top {
				fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s (%d pts, %d comments)\n",
					i+1, truncate(getString(h, "title"), 60),
					getInt(h, "points"), getInt(h, "num_comments"))
			}
			fmt.Fprintln(cmd.OutOrStdout())

			return nil
		},
	}

	cmd.Flags().StringVar(&flagSince, "since", "7d", "Time window (default 7d)")

	return cmd
}
