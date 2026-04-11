package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

func newMyCmd(flags *rootFlags) *cobra.Command {
	var flagSince string
	var flagLimit int

	cmd := &cobra.Command{
		Use:   "my <username>",
		Short: "Track a user's HN submissions — stats, top posts, best times to post",
		Example: `  # Your submission stats
  hackernews-pp-cli my dang

  # Last 7 days only
  hackernews-pp-cli my pg --since 7d

  # JSON output
  hackernews-pp-cli my tptacek --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			// Fetch user profile from Firebase
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			userData, err := c.Get(fmt.Sprintf("/user/%s.json", username), nil)
			if err != nil {
				return classifyAPIError(err)
			}

			var user map[string]any
			if err := json.Unmarshal(userData, &user); err != nil {
				return apiErr(fmt.Errorf("parsing user data: %w", err))
			}

			submitted := getIntSlice(user, "submitted")
			if len(submitted) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "User %s has no submissions\n", username)
				return nil
			}

			// Limit to last N items
			fetchCount := 100
			if flagLimit > 0 && flagLimit < fetchCount {
				fetchCount = flagLimit
			}
			if len(submitted) > fetchCount {
				submitted = submitted[:fetchCount]
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Fetching %d submissions for %s...\n", len(submitted), username)

			items, err := fetchFirebaseItems(flags, submitted, fetchCount)
			if err != nil {
				return apiErr(fmt.Errorf("fetching submissions: %w", err))
			}

			// Filter by time window if --since specified
			if flagSince != "" {
				dur, err := parseDuration(flagSince)
				if err != nil {
					return usageErr(err)
				}
				cutoff := time.Now().Add(-dur)
				var filtered []map[string]any
				for _, item := range items {
					t := getInt(item, "time")
					if t > 0 && time.Unix(int64(t), 0).After(cutoff) {
						filtered = append(filtered, item)
					}
				}
				items = filtered
			}

			// Separate stories from comments
			var stories []map[string]any
			var comments []map[string]any
			for _, item := range items {
				switch getString(item, "type") {
				case "story":
					stories = append(stories, item)
				case "comment":
					comments = append(comments, item)
				}
			}

			// Compute stats for stories
			totalScore := 0
			var topStories []map[string]any
			zeroPosts := 0
			dayBuckets := make(map[string]int) // day of week
			hourBuckets := make(map[int]int)   // hour of day

			for _, s := range stories {
				score := getInt(s, "score")
				totalScore += score
				if score == 0 {
					zeroPosts++
				}
				topStories = append(topStories, s)

				t := getInt(s, "time")
				if t > 0 {
					ts := time.Unix(int64(t), 0)
					dayBuckets[ts.Weekday().String()]++
					hourBuckets[ts.Hour()]++
				}
			}

			// Sort by score
			sort.Slice(topStories, func(i, j int) bool {
				return getInt(topStories[i], "score") > getInt(topStories[j], "score")
			})

			avgScore := float64(0)
			if len(stories) > 0 {
				avgScore = float64(totalScore) / float64(len(stories))
			}

			// Find best posting time
			bestDay := ""
			bestDayCount := 0
			for day, count := range dayBuckets {
				if count > bestDayCount {
					bestDay = day
					bestDayCount = count
				}
			}
			bestHour := 0
			bestHourCount := 0
			for hour, count := range hourBuckets {
				if count > bestHourCount {
					bestHour = hour
					bestHourCount = count
				}
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				top5 := topStories
				if len(top5) > 5 {
					top5 = top5[:5]
				}
				result := map[string]any{
					"username":          username,
					"karma":             getInt(user, "karma"),
					"total_submissions": len(items),
					"stories":           len(stories),
					"comments":          len(comments),
					"average_score":     avgScore,
					"zero_point_posts":  zeroPosts,
					"best_day":          bestDay,
					"best_hour":         bestHour,
					"top_stories":       top5,
				}
				data, _ := json.Marshal(result)
				return printOutput(cmd.OutOrStdout(), json.RawMessage(data), true)
			}

			// Human output
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s (karma: %d)\n\n",
				bold("User:"), username, getInt(user, "karma"))

			fmt.Fprintf(cmd.OutOrStdout(), "  Submissions fetched:  %d\n", len(items))
			fmt.Fprintf(cmd.OutOrStdout(), "  Stories:              %d\n", len(stories))
			fmt.Fprintf(cmd.OutOrStdout(), "  Comments:             %d\n", len(comments))
			fmt.Fprintf(cmd.OutOrStdout(), "  Average story score:  %.1f\n", avgScore)
			fmt.Fprintf(cmd.OutOrStdout(), "  Zero-point stories:   %d\n", zeroPosts)
			if bestDay != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Best posting day:     %s\n", bestDay)
				fmt.Fprintf(cmd.OutOrStdout(), "  Best posting hour:    %d:00 UTC\n", bestHour)
			}

			// Top 5 stories
			top5 := topStories
			if len(top5) > 5 {
				top5 = top5[:5]
			}
			if len(top5) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", bold("Top Stories"))
				rows := make([][]string, 0, len(top5))
				for i, s := range top5 {
					rows = append(rows, []string{
						fmt.Sprintf("%d", i+1),
						truncate(getString(s, "title"), 55),
						fmt.Sprintf("%d", getInt(s, "score")),
						fmt.Sprintf("%d", getInt(s, "descendants")),
					})
				}
				if err := flags.printTable(cmd, []string{"#", "TITLE", "PTS", "CMTS"}, rows); err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flagSince, "since", "", "Time window (e.g., 7d, 30d)")
	cmd.Flags().IntVar(&flagLimit, "limit", 100, "Maximum submissions to fetch")

	return cmd
}
