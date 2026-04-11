package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

func newControversialCmd(flags *rootFlags) *cobra.Command {
	var flagSince string
	var flagLimit int
	var flagMinComments int

	cmd := &cobra.Command{
		Use:   "controversial",
		Short: "Find heated discussions — stories with high comment-to-point ratios",
		Example: `  # Most controversial stories right now
  hackernews-pp-cli controversial

  # Last 24 hours
  hackernews-pp-cli controversial --since 24h

  # Require at least 50 comments
  hackernews-pp-cli controversial --min-comments 50

  # JSON output
  hackernews-pp-cli controversial --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var hits []map[string]any

			if flagSince != "" {
				// Use Algolia for time-windowed search
				dur, err := parseDuration(flagSince)
				if err != nil {
					return usageErr(err)
				}
				ts := sinceTimestamp(dur)
				params := map[string]string{
					"tags":           "story",
					"numericFilters": fmt.Sprintf("created_at_i>%d,num_comments>10", ts),
					"hitsPerPage":    "200",
				}
				data, err := algoliaGet("/search_by_date", params)
				if err != nil {
					return apiErr(err)
				}
				var parseErr error
				hits, parseErr = algoliaHits(data)
				if parseErr != nil {
					return apiErr(parseErr)
				}
			} else {
				// Fetch top and best stories from Firebase, then get details
				c, err := flags.newClient()
				if err != nil {
					return err
				}

				if flags.dryRun {
					return nil
				}

				// Get top stories
				topData, err := c.Get("/topstories.json", nil)
				if err != nil {
					return classifyAPIError(err)
				}
				var topIDs []int
				if err := json.Unmarshal(topData, &topIDs); err != nil {
					return apiErr(fmt.Errorf("parsing top stories: %w", err))
				}

				// Limit fetching
				fetchLimit := 100
				if len(topIDs) > fetchLimit {
					topIDs = topIDs[:fetchLimit]
				}

				fmt.Fprintf(cmd.ErrOrStderr(), "Fetching %d stories for controversy analysis...\n", len(topIDs))

				items, err := fetchFirebaseItems(flags, topIDs, fetchLimit)
				if err != nil {
					return apiErr(err)
				}

				for _, item := range items {
					hit := map[string]any{
						"title":        getString(item, "title"),
						"url":          getString(item, "url"),
						"points":       getInt(item, "score"),
						"num_comments": getInt(item, "descendants"),
						"objectID":     fmt.Sprintf("%d", getInt(item, "id")),
						"created_at_i": getInt(item, "time"),
						"author":       getString(item, "by"),
					}
					hits = append(hits, hit)
				}
			}

			// Filter minimum comments
			minComments := flagMinComments
			if minComments == 0 {
				minComments = 10
			}
			var filtered []map[string]any
			for _, h := range hits {
				if getInt(h, "num_comments") >= minComments {
					filtered = append(filtered, h)
				}
			}
			hits = filtered

			// Compute controversy ratio and sort
			sort.Slice(hits, func(i, j int) bool {
				pi := getInt(hits[i], "points")
				ci := getInt(hits[i], "num_comments")
				pj := getInt(hits[j], "points")
				cj := getInt(hits[j], "num_comments")
				ri := float64(ci) / float64(max(pi, 1))
				rj := float64(cj) / float64(max(pj, 1))
				return ri > rj
			})

			if flagLimit > 0 && len(hits) > flagLimit {
				hits = hits[:flagLimit]
			}

			if len(hits) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "No controversial stories found\n")
				return nil
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "%d controversial stories\n", len(hits))

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				// Add controversy ratio to output
				for _, h := range hits {
					p := getInt(h, "points")
					c := getInt(h, "num_comments")
					h["controversy_ratio"] = float64(c) / float64(max(p, 1))
				}
				return outputJSON(cmd, flags, hits)
			}

			rows := make([][]string, 0, len(hits))
			for i, h := range hits {
				p := getInt(h, "points")
				c := getInt(h, "num_comments")
				ratio := float64(c) / float64(max(p, 1))
				age := ""
				t := getInt(h, "created_at_i")
				if t > 0 {
					age = timeAgo(time.Unix(int64(t), 0))
				}
				rows = append(rows, []string{
					fmt.Sprintf("%d", i+1),
					truncate(getString(h, "title"), 45),
					fmt.Sprintf("%d", p),
					fmt.Sprintf("%d", c),
					fmt.Sprintf("%.1f", ratio),
					age,
				})
			}
			return flags.printTable(cmd, []string{"#", "TITLE", "PTS", "CMTS", "RATIO", "AGE"}, rows)
		},
	}

	cmd.Flags().StringVar(&flagSince, "since", "", "Time window (e.g., 24h, 7d)")
	cmd.Flags().IntVar(&flagLimit, "limit", 20, "Maximum stories to show")
	cmd.Flags().IntVar(&flagMinComments, "min-comments", 10, "Minimum comments to consider")

	return cmd
}
