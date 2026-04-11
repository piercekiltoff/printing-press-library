package cli

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newHiringCmd(flags *rootFlags) *cobra.Command {
	var flagTech bool
	var flagRemote bool
	var flagSalary bool
	var flagLimit int
	var flagSinceMonth string

	cmd := &cobra.Command{
		Use:   "hiring [regex]",
		Short: "Filter Who's Hiring threads for jobs matching your criteria",
		Long: `Search the latest "Who is hiring?" thread on Hacker News.
Fetches the thread and filters comments by pattern or smart flags.`,
		Example: `  # All jobs mentioning Go
  hackernews-pp-cli hiring "(?i)golang|\\bgo\\b"

  # Remote jobs only
  hackernews-pp-cli hiring --remote

  # Jobs mentioning Rust with salary info
  hackernews-pp-cli hiring --tech --salary "(?i)rust"

  # JSON output
  hackernews-pp-cli hiring --remote --json --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Find the latest "Who is hiring" thread
			params := map[string]string{
				"query":       "\"who is hiring\"",
				"tags":        "story,author_whoishiring",
				"hitsPerPage": "1",
			}
			if flagSinceMonth != "" {
				params["query"] = fmt.Sprintf("\"who is hiring\" %s", flagSinceMonth)
			}

			data, err := algoliaGet("/search", params)
			if err != nil {
				return apiErr(fmt.Errorf("searching for hiring thread: %w", err))
			}

			hits, err := algoliaHits(data)
			if err != nil || len(hits) == 0 {
				return apiErr(fmt.Errorf("no Who is Hiring thread found"))
			}

			threadID := getInt(hits[0], "objectID")
			if threadID == 0 {
				// Try string objectID
				if s := getString(hits[0], "objectID"); s != "" {
					fmt.Sscanf(s, "%d", &threadID)
				}
			}
			if threadID == 0 {
				return apiErr(fmt.Errorf("could not parse thread ID"))
			}

			threadTitle := getString(hits[0], "title")
			fmt.Fprintf(cmd.ErrOrStderr(), "Thread: %s\n", threadTitle)
			fmt.Fprintf(cmd.ErrOrStderr(), "Fetching comments...\n")

			// Fetch thread from Firebase to get kids
			thread, err := fetchFirebaseItem(flags, threadID)
			if err != nil {
				return apiErr(fmt.Errorf("fetching thread: %w", err))
			}

			kidIDs := getIntSlice(thread, "kids")
			if len(kidIDs) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "No comments found in thread\n")
				return nil
			}

			// Fetch top-level comments (job postings)
			fetchLimit := len(kidIDs)
			if fetchLimit > 500 {
				fetchLimit = 500
			}
			comments, err := fetchFirebaseItems(flags, kidIDs[:fetchLimit], fetchLimit)
			if err != nil {
				return apiErr(fmt.Errorf("fetching comments: %w", err))
			}

			// Build filter patterns
			var patterns []*regexp.Regexp

			if len(args) > 0 && args[0] != "" {
				re, err := regexp.Compile(args[0])
				if err != nil {
					return usageErr(fmt.Errorf("invalid regex %q: %w", args[0], err))
				}
				patterns = append(patterns, re)
			}

			if flagTech {
				// The user regex is their tech filter; if no args, match common tech terms
				if len(args) == 0 {
					patterns = append(patterns, regexp.MustCompile(`(?i)\b(rust|go|golang|python|typescript|javascript|java|c\+\+|ruby|swift|kotlin|scala|elixir|haskell|react|vue|angular|node\.?js|django|rails|flask|spring|docker|kubernetes|aws|gcp|azure|terraform|postgres|mysql|redis|kafka|graphql)\b`))
				}
			}

			if flagRemote {
				patterns = append(patterns, regexp.MustCompile(`(?i)\b(remote|fully.remote|remote.first|remote.friendly|work.from.home|wfh|distributed)\b`))
			}

			if flagSalary {
				patterns = append(patterns, regexp.MustCompile(`(?i)(\$\d|k/yr|\d+k\s*[-–]\s*\d+k|salary|compensation|total.comp|TC\s|OTE\b|base\s+\d)`))
			}

			// Filter comments
			var matched []map[string]any
			for _, c := range comments {
				text := getString(c, "text")
				if text == "" || getString(c, "deleted") == "true" || getString(c, "dead") == "true" {
					continue
				}

				if len(patterns) == 0 {
					matched = append(matched, c)
					continue
				}

				allMatch := true
				for _, p := range patterns {
					if !p.MatchString(text) {
						allMatch = false
						break
					}
				}
				if allMatch {
					matched = append(matched, c)
				}
			}

			// Sort by score descending
			sort.Slice(matched, func(i, j int) bool {
				return getInt(matched[i], "score") > getInt(matched[j], "score")
			})

			// Apply limit
			if flagLimit > 0 && len(matched) > flagLimit {
				matched = matched[:flagLimit]
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "%d matching jobs out of %d total\n", len(matched), len(comments))

			if len(matched) == 0 {
				return nil
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				return outputJSON(cmd, flags, matched)
			}

			// Display each job posting
			for i, c := range matched {
				text := getString(c, "text")
				// Extract company name (first line)
				lines := strings.SplitN(stripHTML(text), "\n", 2)
				company := truncate(strings.TrimSpace(lines[0]), 80)

				fmt.Fprintf(cmd.OutOrStdout(), "\n%s[%d] %s%s\n", bold(""), i+1, company, bold(""))
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", truncate(stripHTML(text), 300))
				fmt.Fprintf(cmd.OutOrStdout(), "---\n")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagTech, "tech", false, "Filter for tech/language mentions")
	cmd.Flags().BoolVar(&flagRemote, "remote", false, "Filter for remote positions")
	cmd.Flags().BoolVar(&flagSalary, "salary", false, "Filter for salary/compensation info")
	cmd.Flags().IntVar(&flagLimit, "limit", 30, "Maximum jobs to show")
	cmd.Flags().StringVar(&flagSinceMonth, "since", "", "Pick a specific month's thread (e.g., 'March 2025')")

	return cmd
}

// stripHTML removes HTML tags and decodes common entities for terminal display.
func stripHTML(s string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, " ")
	// Decode common HTML entities
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#x27;", "'")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&#x2F;", "/")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	// Collapse whitespace
	ws := regexp.MustCompile(`\s+`)
	s = ws.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
