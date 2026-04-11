package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

func addAskEnhancedFlags(askCmd *cobra.Command, flags *rootFlags) {
	var flagTopic string
	var flagUnanswered bool
	var flagSince string
	var flagMinPoints int
	var flagSort string

	askCmd.Flags().StringVar(&flagTopic, "topic", "", "Search within Ask HN posts (e.g., --topic 'career advice')")
	askCmd.Flags().BoolVar(&flagUnanswered, "unanswered", false, "Show Ask HN posts with 0 comments")
	askCmd.Flags().StringVar(&flagSince, "since", "", "Time window (e.g., 1h, 24h, 7d)")
	askCmd.Flags().IntVar(&flagMinPoints, "min-points", 0, "Minimum points threshold")
	askCmd.Flags().StringVar(&flagSort, "sort", "", "Sort by: points, comments, date, velocity")

	originalRunE := askCmd.RunE
	askCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// If no enhanced flags are set, run the original command
		if flagTopic == "" && !flagUnanswered && flagSince == "" && flagMinPoints == 0 && flagSort == "" {
			return originalRunE(cmd, args)
		}

		// Enhanced mode: use Algolia
		params := map[string]string{
			"tags":        "ask_hn",
			"hitsPerPage": "200",
		}

		var searchPath string

		if flagTopic != "" {
			params["query"] = flagTopic
			searchPath = "/search"
		}

		if flagUnanswered {
			if params["numericFilters"] != "" {
				params["numericFilters"] += ",num_comments=0"
			} else {
				params["numericFilters"] = "num_comments=0"
			}
			searchPath = "/search_by_date"
		}

		if flagSince != "" {
			dur, err := parseDuration(flagSince)
			if err != nil {
				return usageErr(err)
			}
			ts := sinceTimestamp(dur)
			filter := fmt.Sprintf("created_at_i>%d", ts)
			if params["numericFilters"] != "" {
				params["numericFilters"] += "," + filter
			} else {
				params["numericFilters"] = filter
			}
			if searchPath == "" {
				searchPath = "/search_by_date"
			}
		}

		if searchPath == "" {
			searchPath = "/search"
		}

		data, err := algoliaGet(searchPath, params)
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
		sortBy := flagSort
		if sortBy == "" {
			sortBy = "points"
		}
		switch sortBy {
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
		default:
			sort.Slice(hits, func(i, j int) bool {
				return getInt(hits[i], "points") > getInt(hits[j], "points")
			})
		}

		// Apply limit from the existing --limit flag
		limitVal, _ := cmd.Flags().GetInt("limit")
		if limitVal > 0 && len(hits) > limitVal {
			hits = hits[:limitVal]
		}

		if len(hits) == 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "No Ask HN posts found\n")
			return nil
		}

		label := "Ask HN"
		if flagTopic != "" {
			label += fmt.Sprintf(" matching %q", flagTopic)
		}
		if flagUnanswered {
			label += " (unanswered)"
		}
		if flagSince != "" {
			label += " (last " + flagSince + ")"
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "%d %s posts\n", len(hits), label)

		if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
			return outputJSON(cmd, flags, hits)
		}

		rows := make([][]string, 0, len(hits))
		for i, h := range hits {
			created := time.Unix(int64(getInt(h, "created_at_i")), 0)
			rows = append(rows, []string{
				fmt.Sprintf("%d", i+1),
				truncate(getString(h, "title"), 55),
				fmt.Sprintf("%d", getInt(h, "points")),
				fmt.Sprintf("%d", getInt(h, "num_comments")),
				timeAgo(created),
			})
		}
		return flags.printTable(cmd, []string{"#", "TITLE", "PTS", "CMTS", "AGE"}, rows)
	}
}
