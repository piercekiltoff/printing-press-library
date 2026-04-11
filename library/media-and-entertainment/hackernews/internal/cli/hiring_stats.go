package cli

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newHiringStatsCmd(flags *rootFlags) *cobra.Command {
	var flagMonths int
	var flagCompare bool

	cmd := &cobra.Command{
		Use:   "hiring-stats",
		Short: "Aggregate hiring data across Who's Hiring threads",
		Long: `Analyze tech trends, remote work, and salary mentions across
recent Who is Hiring threads on Hacker News.`,
		Example: `  # Stats for last 3 months (default)
  hackernews-pp-cli hiring-stats

  # Stats for last 6 months with comparison
  hackernews-pp-cli hiring-stats --months 6 --compare

  # JSON output
  hackernews-pp-cli hiring-stats --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Find recent hiring threads
			params := map[string]string{
				"query":       "\"who is hiring\"",
				"tags":        "story,author_whoishiring",
				"hitsPerPage": fmt.Sprintf("%d", flagMonths),
			}

			data, err := algoliaGet("/search_by_date", params)
			if err != nil {
				return apiErr(fmt.Errorf("searching for hiring threads: %w", err))
			}

			threads, err := algoliaHits(data)
			if err != nil || len(threads) == 0 {
				return apiErr(fmt.Errorf("no Who is Hiring threads found"))
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Found %d hiring threads, analyzing...\n", len(threads))

			type monthData struct {
				title  string
				techs  map[string]int
				remote int
				salary int
				total  int
			}

			techPatterns := map[string]*regexp.Regexp{
				"Python":     regexp.MustCompile(`(?i)\bpython\b`),
				"JavaScript": regexp.MustCompile(`(?i)\b(javascript|js)\b`),
				"TypeScript": regexp.MustCompile(`(?i)\btypescript\b`),
				"Go":         regexp.MustCompile(`(?i)\b(golang|\bgo\b)`),
				"Rust":       regexp.MustCompile(`(?i)\brust\b`),
				"Java":       regexp.MustCompile(`(?i)\bjava\b`),
				"C++":        regexp.MustCompile(`(?i)\bc\+\+\b`),
				"Ruby":       regexp.MustCompile(`(?i)\bruby\b`),
				"Swift":      regexp.MustCompile(`(?i)\bswift\b`),
				"Kotlin":     regexp.MustCompile(`(?i)\bkotlin\b`),
				"React":      regexp.MustCompile(`(?i)\breact\b`),
				"Node.js":    regexp.MustCompile(`(?i)\bnode\.?js\b`),
				"Docker":     regexp.MustCompile(`(?i)\bdocker\b`),
				"K8s":        regexp.MustCompile(`(?i)\b(kubernetes|k8s)\b`),
				"AWS":        regexp.MustCompile(`(?i)\baws\b`),
				"PostgreSQL": regexp.MustCompile(`(?i)\b(postgres|postgresql)\b`),
				"Redis":      regexp.MustCompile(`(?i)\bredis\b`),
				"GraphQL":    regexp.MustCompile(`(?i)\bgraphql\b`),
			}
			remoteRe := regexp.MustCompile(`(?i)\b(remote|fully.remote|remote.first)\b`)
			salaryRe := regexp.MustCompile(`(?i)(\$\d|k/yr|\d+k\s*[-–]\s*\d+k|salary|compensation|total.comp|TC\s|OTE\b)`)

			var months []monthData

			for _, thread := range threads {
				threadID := getInt(thread, "objectID")
				if threadID == 0 {
					if s := getString(thread, "objectID"); s != "" {
						fmt.Sscanf(s, "%d", &threadID)
					}
				}
				if threadID == 0 {
					continue
				}

				title := getString(thread, "title")
				fmt.Fprintf(cmd.ErrOrStderr(), "  Analyzing: %s\n", title)

				item, err := fetchFirebaseItem(flags, threadID)
				if err != nil {
					continue
				}

				kidIDs := getIntSlice(item, "kids")
				fetchLimit := len(kidIDs)
				if fetchLimit > 500 {
					fetchLimit = 500
				}
				if fetchLimit == 0 {
					continue
				}

				comments, err := fetchFirebaseItems(flags, kidIDs[:fetchLimit], fetchLimit)
				if err != nil {
					continue
				}

				md := monthData{
					title: title,
					techs: make(map[string]int),
				}

				for _, c := range comments {
					text := getString(c, "text")
					if text == "" {
						continue
					}
					md.total++

					for tech, re := range techPatterns {
						if re.MatchString(text) {
							md.techs[tech]++
						}
					}
					if remoteRe.MatchString(text) {
						md.remote++
					}
					if salaryRe.MatchString(text) {
						md.salary++
					}
				}

				months = append(months, md)
			}

			if len(months) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "No data collected\n")
				return nil
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				result := make([]map[string]any, 0, len(months))
				for _, md := range months {
					entry := map[string]any{
						"title":        md.title,
						"total_jobs":   md.total,
						"remote_count": md.remote,
						"salary_count": md.salary,
						"techs":        md.techs,
					}
					result = append(result, entry)
				}
				data, _ := json.Marshal(result)
				return printOutput(cmd.OutOrStdout(), json.RawMessage(data), true)
			}

			// Collect all techs across months
			allTechs := map[string]bool{}
			for _, md := range months {
				for tech := range md.techs {
					allTechs[tech] = true
				}
			}

			// Sort techs by most recent count
			type techCount struct {
				name  string
				count int
			}
			var sorted []techCount
			for tech := range allTechs {
				count := 0
				if len(months) > 0 {
					count = months[0].techs[tech]
				}
				sorted = append(sorted, techCount{name: tech, count: count})
			}
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].count > sorted[j].count
			})

			// Print tech table
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n\n", bold("Tech Mentions Across Hiring Threads"))

			headers := []string{"TECH"}
			for _, md := range months {
				// Extract month name from title
				title := md.title
				if idx := strings.Index(title, "("); idx > 0 {
					title = strings.TrimSpace(title[idx:])
				}
				headers = append(headers, truncate(title, 20))
			}
			if flagCompare && len(months) >= 2 {
				headers = append(headers, "TREND")
			}

			rows := make([][]string, 0, len(sorted))
			for _, tc := range sorted {
				if tc.count == 0 && len(months) > 1 && months[1].techs[tc.name] == 0 {
					continue
				}
				row := []string{tc.name}
				for _, md := range months {
					pct := float64(0)
					if md.total > 0 {
						pct = float64(md.techs[tc.name]) / float64(md.total) * 100
					}
					row = append(row, fmt.Sprintf("%d (%.0f%%)", md.techs[tc.name], pct))
				}
				if flagCompare && len(months) >= 2 {
					curr := months[0].techs[tc.name]
					prev := months[1].techs[tc.name]
					if prev > 0 {
						change := float64(curr-prev) / float64(prev) * 100
						if change > 5 {
							row = append(row, fmt.Sprintf("+%.0f%%", change))
						} else if change < -5 {
							row = append(row, fmt.Sprintf("%.0f%%", change))
						} else {
							row = append(row, "~")
						}
					} else if curr > 0 {
						row = append(row, "new")
					} else {
						row = append(row, "-")
					}
				}
				rows = append(rows, row)
			}
			if err := flags.printTable(cmd, headers, rows); err != nil {
				return err
			}

			// Summary
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s\n", bold("Summary"))
			for _, md := range months {
				remotePct := float64(0)
				salaryPct := float64(0)
				if md.total > 0 {
					remotePct = float64(md.remote) / float64(md.total) * 100
					salaryPct = float64(md.salary) / float64(md.total) * 100
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %s: %d jobs, %d remote (%.0f%%), %d with salary (%.0f%%)\n",
					truncate(md.title, 40), md.total, md.remote, remotePct, md.salary, salaryPct)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&flagMonths, "months", 3, "Number of months to analyze")
	cmd.Flags().BoolVar(&flagCompare, "compare", false, "Show trend comparison between months")

	return cmd
}
