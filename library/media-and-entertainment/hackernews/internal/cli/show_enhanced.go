package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

func addShowEnhancedFlags(showCmd *cobra.Command, flags *rootFlags) {
	var flagHot bool
	var flagSince string
	var flagMinPoints int
	var flagSort string

	showCmd.Flags().BoolVar(&flagHot, "hot", false, "Sort by velocity (points/hour)")
	showCmd.Flags().StringVar(&flagSince, "since", "", "Time window (e.g., 1h, 24h, 7d)")
	showCmd.Flags().IntVar(&flagMinPoints, "min-points", 0, "Minimum points threshold")
	showCmd.Flags().StringVar(&flagSort, "sort", "", "Sort by: points, comments, date, velocity")

	originalRunE := showCmd.RunE
	showCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// If no enhanced flags are set, run the original command
		if !flagHot && flagSince == "" && flagMinPoints == 0 && flagSort == "" {
			return originalRunE(cmd, args)
		}

		// Enhanced mode: use Algolia
		params := map[string]string{
			"tags":        "show_hn",
			"hitsPerPage": "200",
		}

		var searchPath string
		if flagSince != "" {
			dur, err := parseDuration(flagSince)
			if err != nil {
				return usageErr(err)
			}
			ts := sinceTimestamp(dur)
			params["numericFilters"] = fmt.Sprintf("created_at_i>%d", ts)
			searchPath = "/search_by_date"
		} else {
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

		// Determine sort
		sortBy := flagSort
		if flagHot {
			sortBy = "velocity"
		}
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
		default: // "points"
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
			fmt.Fprintf(cmd.ErrOrStderr(), "No Show HN posts found\n")
			return nil
		}

		label := "Show HN"
		if flagSince != "" {
			label += " (last " + flagSince + ")"
		}
		if flagHot {
			label += " sorted by velocity"
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "%d %s posts\n", len(hits), label)

		if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
			return outputJSON(cmd, flags, hits)
		}

		rows := make([][]string, 0, len(hits))
		for i, h := range hits {
			created := time.Unix(int64(getInt(h, "created_at_i")), 0)
			vel := velocity(getInt(h, "points"), created)
			rows = append(rows, []string{
				fmt.Sprintf("%d", i+1),
				truncate(getString(h, "title"), 50),
				fmt.Sprintf("%d", getInt(h, "points")),
				fmt.Sprintf("%d", getInt(h, "num_comments")),
				fmt.Sprintf("%.1f", vel),
				timeAgo(created),
			})
		}
		return flags.printTable(cmd, []string{"#", "TITLE", "PTS", "CMTS", "VEL", "AGE"}, rows)
	}
}
