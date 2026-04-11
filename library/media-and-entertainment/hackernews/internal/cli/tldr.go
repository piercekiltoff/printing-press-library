package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

func newTldrCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tldr <id>",
		Short: "Mechanical thread digest — key takes, active commenters, controversy score",
		Long: `Generate a quick digest of an HN thread without AI summarization.
Shows comment stats, top-level replies by score, most active commenters,
and a controversy score. Pipe-friendly output for piping to claude.`,
		Example: `  # Digest a thread
  hackernews-pp-cli tldr 12345678

  # Pipe to claude for AI summary
  hackernews-pp-cli tldr 12345678 | claude "summarize the key arguments"

  # JSON output
  hackernews-pp-cli tldr 12345678 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var itemID int
			if _, err := fmt.Sscanf(args[0], "%d", &itemID); err != nil {
				return usageErr(fmt.Errorf("invalid item ID %q", args[0]))
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Fetching thread...\n")

			// Fetch root item
			root, err := fetchFirebaseItem(flags, itemID)
			if err != nil {
				return classifyAPIError(err)
			}

			title := getString(root, "title")
			points := getInt(root, "score")
			totalComments := getInt(root, "descendants")
			rootTime := getInt(root, "time")

			// Fetch top-level replies
			kidIDs := getIntSlice(root, "kids")
			if len(kidIDs) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "No comments on this thread\n")
				return nil
			}

			topLevel, err := fetchFirebaseItems(flags, kidIDs, len(kidIDs))
			if err != nil {
				return apiErr(fmt.Errorf("fetching comments: %w", err))
			}

			// Filter dead/deleted
			var validTopLevel []map[string]any
			for _, c := range topLevel {
				if getString(c, "deleted") != "true" && getString(c, "dead") != "true" {
					validTopLevel = append(validTopLevel, c)
				}
			}

			// Recursively collect all comments for stats
			type commentInfo struct {
				author string
				time   int
			}
			var allComments []commentInfo
			var collectAll func(parentIDs []int, depth int)
			collectAll = func(parentIDs []int, depth int) {
				if depth > 10 || len(allComments) > 1000 {
					return
				}
				items, err := fetchFirebaseItems(flags, parentIDs, len(parentIDs))
				if err != nil {
					return
				}
				for _, item := range items {
					if getString(item, "deleted") == "true" || getString(item, "dead") == "true" {
						continue
					}
					allComments = append(allComments, commentInfo{
						author: getString(item, "by"),
						time:   getInt(item, "time"),
					})
					kids := getIntSlice(item, "kids")
					if len(kids) > 0 {
						collectAll(kids, depth+1)
					}
				}
			}
			// Add top-level first
			for _, c := range validTopLevel {
				allComments = append(allComments, commentInfo{
					author: getString(c, "by"),
					time:   getInt(c, "time"),
				})
				kids := getIntSlice(c, "kids")
				if len(kids) > 0 {
					collectAll(kids, 2)
				}
			}

			// Unique authors
			authorCounts := map[string]int{}
			var earliestComment, latestComment int
			for _, c := range allComments {
				authorCounts[c.author]++
				if earliestComment == 0 || c.time < earliestComment {
					earliestComment = c.time
				}
				if c.time > latestComment {
					latestComment = c.time
				}
			}

			// Sort top-level by thread size (number of kids)
			sort.Slice(validTopLevel, func(i, j int) bool {
				iKids := len(getIntSlice(validTopLevel[i], "kids"))
				jKids := len(getIntSlice(validTopLevel[j], "kids"))
				return iKids > jKids
			})

			// Most active commenters
			type authorStat struct {
				name  string
				count int
			}
			var activeAuthors []authorStat
			for name, count := range authorCounts {
				activeAuthors = append(activeAuthors, authorStat{name: name, count: count})
			}
			sort.Slice(activeAuthors, func(i, j int) bool {
				return activeAuthors[i].count > activeAuthors[j].count
			})

			// Controversy score: comments/points ratio
			controversyScore := float64(0)
			if points > 0 {
				controversyScore = float64(totalComments) / float64(points)
			}

			// Time span
			timeSpan := ""
			if earliestComment > 0 && latestComment > 0 {
				span := time.Unix(int64(latestComment), 0).Sub(time.Unix(int64(earliestComment), 0))
				if span.Hours() < 1 {
					timeSpan = fmt.Sprintf("%.0f minutes", span.Minutes())
				} else if span.Hours() < 24 {
					timeSpan = fmt.Sprintf("%.0f hours", span.Hours())
				} else {
					timeSpan = fmt.Sprintf("%.0f days", span.Hours()/24)
				}
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				topReplies := validTopLevel
				if len(topReplies) > 10 {
					topReplies = topReplies[:10]
				}
				topRepliesOut := make([]map[string]any, 0, len(topReplies))
				for _, c := range topReplies {
					topRepliesOut = append(topRepliesOut, map[string]any{
						"by":      getString(c, "by"),
						"text":    truncate(stripHTML(getString(c, "text")), 200),
						"replies": len(getIntSlice(c, "kids")),
					})
				}
				top5Authors := activeAuthors
				if len(top5Authors) > 5 {
					top5Authors = top5Authors[:5]
				}
				authorsOut := make([]map[string]any, 0, len(top5Authors))
				for _, a := range top5Authors {
					authorsOut = append(authorsOut, map[string]any{
						"author":   a.name,
						"comments": a.count,
					})
				}
				result := map[string]any{
					"title":             title,
					"points":            points,
					"total_comments":    totalComments,
					"fetched_comments":  len(allComments),
					"unique_authors":    len(authorCounts),
					"time_span":         timeSpan,
					"controversy_score": controversyScore,
					"top_replies":       topRepliesOut,
					"active_authors":    authorsOut,
				}
				data, _ := json.Marshal(result)
				return printOutput(cmd.OutOrStdout(), json.RawMessage(data), true)
			}

			// Human output
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", bold(title))
			fmt.Fprintf(cmd.OutOrStdout(), "%d points | %d comments | %d unique authors | span: %s\n",
				points, totalComments, len(authorCounts), timeSpan)
			if rootTime > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Posted %s\n", timeAgo(time.Unix(int64(rootTime), 0)))
			}

			// Controversy
			controversyLabel := "low"
			if controversyScore > 2 {
				controversyLabel = "moderate"
			}
			if controversyScore > 5 {
				controversyLabel = "high"
			}
			if controversyScore > 10 {
				controversyLabel = "very high"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Controversy: %.1f (%s)\n", controversyScore, controversyLabel)

			// Key takes (top-level replies by thread size)
			keyTakes := validTopLevel
			if len(keyTakes) > 7 {
				keyTakes = keyTakes[:7]
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", bold("Key Takes (top replies by thread activity)"))
			for i, c := range keyTakes {
				author := getString(c, "by")
				text := truncate(stripHTML(getString(c, "text")), 120)
				replies := len(getIntSlice(c, "kids"))
				fmt.Fprintf(cmd.OutOrStdout(), "  %d. [%s] (%d replies) %s\n", i+1, author, replies, text)
			}

			// Most active
			top5 := activeAuthors
			if len(top5) > 5 {
				top5 = top5[:5]
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", bold("Most Active Commenters"))
			for _, a := range top5 {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s: %d comments\n", a.name, a.count)
			}
			fmt.Fprintln(cmd.OutOrStdout())

			return nil
		},
	}

	return cmd
}
