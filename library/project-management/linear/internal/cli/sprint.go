package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/project-management/linear/internal/store"
	"github.com/spf13/cobra"
)

func newSprintCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sprint",
		Short: "Sprint analytics: status, burndown, velocity, carry-over",
	}

	cmd.AddCommand(newSprintStatusCmd(flags))
	cmd.AddCommand(newSprintBurndownCmd(flags))
	cmd.AddCommand(newSprintVelocityCmd(flags))
	cmd.AddCommand(newSprintCarryOverCmd(flags))

	return cmd
}

func newSprintStatusCmd(flags *rootFlags) *cobra.Command {
	var team string
	var dbPath string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current sprint overview with progress",
		Example: `  linear-pp-cli sprint status --team ENG
  linear-pp-cli sprint status --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, dbPath := openDB(dbPath)
			if db == nil {
				return fmt.Errorf("opening local database at %s\nRun 'linear-pp-cli workflow archive' first.", dbPath)
			}
			defer db.Close()

			teamID := ""
			if team != "" {
				teamID = resolveTeamID(db, team)
			}

			// Find current cycle
			cycle, err := findCurrentCycle(db, teamID)
			if err != nil || cycle == nil {
				return fmt.Errorf("no active cycle found. Sync data with 'workflow archive' first")
			}

			cycleName, _ := cycle["name"].(string)
			cycleNumber, _ := cycle["number"].(float64)
			startsAt, _ := cycle["startsAt"].(string)
			endsAt, _ := cycle["endsAt"].(string)
			cycleID, _ := cycle["id"].(string)

			// Get issues in this cycle
			issues := getIssuesForCycle(db, cycleID)

			total := len(issues)
			done := 0
			inProgress := 0
			todo := 0
			totalEstimate := 0
			doneEstimate := 0

			for _, iss := range issues {
				state, _ := iss["state"].(map[string]any)
				stateName, _ := state["name"].(string)
				stateType, _ := state["type"].(string)

				est := 0
				if e, ok := iss["estimate"].(float64); ok {
					est = int(e)
				}
				totalEstimate += est

				category := classifyState(stateType, stateName)
				switch category {
				case "done":
					done++
					doneEstimate += est
				case "active":
					inProgress++
				default:
					todo++
				}
			}

			pct := 0.0
			if total > 0 {
				pct = float64(done) / float64(total) * 100
			}

			result := map[string]any{
				"cycle_name":     cycleName,
				"cycle_number":   int(cycleNumber),
				"starts_at":      startsAt,
				"ends_at":        endsAt,
				"total_issues":   total,
				"done":           done,
				"in_progress":    inProgress,
				"todo":           todo,
				"percent_done":   math.Round(pct*10) / 10,
				"total_estimate": totalEstimate,
				"done_estimate":  doneEstimate,
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Sprint: %s (Cycle %d)\n", cycleName, int(cycleNumber))
			fmt.Fprintf(cmd.OutOrStdout(), "Period: %s → %s\n\n", formatDate(startsAt), formatDate(endsAt))

			bar := progressBar(pct, 30)
			fmt.Fprintf(cmd.OutOrStdout(), "  Progress: %s %.1f%%\n\n", bar, pct)
			fmt.Fprintf(cmd.OutOrStdout(), "  Done:        %d issues (%d pts)\n", done, doneEstimate)
			fmt.Fprintf(cmd.OutOrStdout(), "  In Progress: %d issues\n", inProgress)
			fmt.Fprintf(cmd.OutOrStdout(), "  Todo:        %d issues\n", todo)
			fmt.Fprintf(cmd.OutOrStdout(), "  Total:       %d issues (%d pts)\n", total, totalEstimate)

			// Days remaining
			if end, err := time.Parse(time.RFC3339, endsAt); err == nil {
				remaining := int(time.Until(end).Hours() / 24)
				if remaining > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "\n  %d days remaining\n", remaining)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "\n  Sprint ended\n")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&team, "team", "", "Filter by team key or name")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func newSprintBurndownCmd(flags *rootFlags) *cobra.Command {
	var team string
	var dbPath string

	cmd := &cobra.Command{
		Use:   "burndown",
		Short: "ASCII burndown chart for current sprint",
		Example: `  linear-pp-cli sprint burndown --team ENG`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, dbPath := openDB(dbPath)
			if db == nil {
				return fmt.Errorf("opening local database at %s", dbPath)
			}
			defer db.Close()

			teamID := ""
			if team != "" {
				teamID = resolveTeamID(db, team)
			}

			cycle, err := findCurrentCycle(db, teamID)
			if err != nil || cycle == nil {
				return fmt.Errorf("no active cycle found")
			}

			cycleID, _ := cycle["id"].(string)
			startsAt, _ := cycle["startsAt"].(string)
			endsAt, _ := cycle["endsAt"].(string)

			startTime, _ := time.Parse(time.RFC3339, startsAt)
			endTime, _ := time.Parse(time.RFC3339, endsAt)
			totalDays := int(endTime.Sub(startTime).Hours()/24) + 1

			issues := getIssuesForCycle(db, cycleID)
			totalIssues := len(issues)

			if totalIssues == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No issues in current cycle.")
				return nil
			}

			// Calculate remaining issues per day
			remaining := make([]int, totalDays)
			for i := range remaining {
				remaining[i] = totalIssues
			}

			for _, iss := range issues {
				completedAt, _ := iss["completedAt"].(string)
				if completedAt == "" {
					continue
				}
				ct, err := time.Parse(time.RFC3339, completedAt)
				if err != nil {
					continue
				}
				dayIdx := int(ct.Sub(startTime).Hours() / 24)
				if dayIdx < 0 {
					dayIdx = 0
				}
				for i := dayIdx; i < totalDays; i++ {
					remaining[i]--
				}
			}

			if flags.asJSON {
				type dayData struct {
					Day       int    `json:"day"`
					Date      string `json:"date"`
					Remaining int    `json:"remaining"`
					Ideal     float64 `json:"ideal"`
				}
				var data []dayData
				for i := 0; i < totalDays; i++ {
					d := startTime.AddDate(0, 0, i)
					ideal := float64(totalIssues) * (1.0 - float64(i)/float64(totalDays-1))
					data = append(data, dayData{
						Day:       i + 1,
						Date:      d.Format("2006-01-02"),
						Remaining: remaining[i],
						Ideal:     math.Round(ideal*10) / 10,
					})
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(data)
			}

			// ASCII burndown
			chartHeight := 15
			maxVal := totalIssues
			fmt.Fprintf(cmd.OutOrStdout(), "Burndown (Sprint %s):\n\n", formatDate(startsAt))

			elapsed := int(time.Since(startTime).Hours()/24) + 1
			if elapsed > totalDays {
				elapsed = totalDays
			}

			for row := chartHeight; row >= 0; row-- {
				threshold := float64(maxVal) * float64(row) / float64(chartHeight)
				label := fmt.Sprintf("%3d ", int(threshold))
				fmt.Fprint(cmd.OutOrStdout(), label)
				for day := 0; day < totalDays && day < elapsed; day++ {
					ideal := float64(maxVal) * (1.0 - float64(day)/float64(totalDays-1))
					if float64(remaining[day]) >= threshold {
						fmt.Fprint(cmd.OutOrStdout(), "█")
					} else if ideal >= threshold {
						fmt.Fprint(cmd.OutOrStdout(), "·")
					} else {
						fmt.Fprint(cmd.OutOrStdout(), " ")
					}
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			fmt.Fprintf(cmd.OutOrStdout(), "    ")
			for i := 0; i < totalDays && i < elapsed; i++ {
				if i%5 == 0 {
					fmt.Fprint(cmd.OutOrStdout(), "|")
				} else {
					fmt.Fprint(cmd.OutOrStdout(), "-")
				}
			}
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "    Day 1%s Day %d\n", strings.Repeat(" ", totalDays-8), totalDays)
			fmt.Fprintf(cmd.OutOrStdout(), "\n  █ = actual remaining  · = ideal burndown\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&team, "team", "", "Filter by team key or name")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func newSprintVelocityCmd(flags *rootFlags) *cobra.Command {
	var team string
	var dbPath string
	var n int

	cmd := &cobra.Command{
		Use:   "velocity",
		Short: "Show velocity across recent sprints",
		Example: `  linear-pp-cli sprint velocity --team ENG
  linear-pp-cli sprint velocity -n 8 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, dbPath := openDB(dbPath)
			if db == nil {
				return fmt.Errorf("opening local database at %s", dbPath)
			}
			defer db.Close()

			teamID := ""
			if team != "" {
				teamID = resolveTeamID(db, team)
			}

			cycles := getRecentCycles(db, teamID, n)
			if len(cycles) == 0 {
				return fmt.Errorf("no cycles found")
			}

			type velocityEntry struct {
				CycleNumber int    `json:"cycle_number"`
				CycleName   string `json:"cycle_name"`
				StartsAt    string `json:"starts_at"`
				EndsAt      string `json:"ends_at"`
				Total       int    `json:"total_issues"`
				Completed   int    `json:"completed"`
				Points      int    `json:"points_completed"`
			}

			var entries []velocityEntry
			for _, cyc := range cycles {
				cycID, _ := cyc["id"].(string)
				cycNum, _ := cyc["number"].(float64)
				cycName, _ := cyc["name"].(string)
				startsAt, _ := cyc["startsAt"].(string)
				endsAt, _ := cyc["endsAt"].(string)

				issues := getIssuesForCycle(db, cycID)
				total := len(issues)
				completed := 0
				points := 0

				for _, iss := range issues {
					state, _ := iss["state"].(map[string]any)
					stateName, _ := state["name"].(string)
					stateType, _ := state["type"].(string)
					est := 0
					if e, ok := iss["estimate"].(float64); ok {
						est = int(e)
					}
					if classifyState(stateType, stateName) == "done" {
						completed++
						points += est
					}
				}

				entries = append(entries, velocityEntry{
					CycleNumber: int(cycNum),
					CycleName:   cycName,
					StartsAt:    startsAt,
					EndsAt:      endsAt,
					Total:       total,
					Completed:   completed,
					Points:      points,
				})
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(entries)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Velocity (last %d cycles):\n\n", len(entries))
			fmt.Fprintf(cmd.OutOrStdout(), "  %-8s %-20s %8s %8s %8s\n",
				"CYCLE", "PERIOD", "TOTAL", "DONE", "POINTS")
			fmt.Fprintf(cmd.OutOrStdout(), "  %-8s %-20s %8s %8s %8s\n",
				"-----", "------", "-----", "----", "------")

			totalCompleted := 0
			totalPoints := 0
			for _, e := range entries {
				period := fmt.Sprintf("%s→%s", formatDate(e.StartsAt), formatDate(e.EndsAt))
				fmt.Fprintf(cmd.OutOrStdout(), "  %-8d %-20s %8d %8d %8d\n",
					e.CycleNumber, period, e.Total, e.Completed, e.Points)
				totalCompleted += e.Completed
				totalPoints += e.Points
			}

			if len(entries) > 0 {
				avgIssues := float64(totalCompleted) / float64(len(entries))
				avgPoints := float64(totalPoints) / float64(len(entries))
				fmt.Fprintf(cmd.OutOrStdout(), "\n  Average: %.1f issues/sprint, %.1f points/sprint\n",
					avgIssues, avgPoints)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&team, "team", "", "Filter by team key or name")
	cmd.Flags().IntVarP(&n, "count", "n", 6, "Number of recent cycles to show")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func newSprintCarryOverCmd(flags *rootFlags) *cobra.Command {
	var team string
	var dbPath string

	cmd := &cobra.Command{
		Use:   "carry-over",
		Short: "Show incomplete issues from the last completed sprint",
		Example: `  linear-pp-cli sprint carry-over --team ENG
  linear-pp-cli sprint carry-over --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, dbPath := openDB(dbPath)
			if db == nil {
				return fmt.Errorf("opening local database at %s", dbPath)
			}
			defer db.Close()

			teamID := ""
			if team != "" {
				teamID = resolveTeamID(db, team)
			}

			cycles := getRecentCycles(db, teamID, 2)
			if len(cycles) < 1 {
				return fmt.Errorf("no completed cycles found")
			}

			// Find the most recent completed cycle
			var lastCompleted map[string]any
			for _, cyc := range cycles {
				if ca, _ := cyc["completedAt"].(string); ca != "" {
					lastCompleted = cyc
					break
				}
			}
			if lastCompleted == nil {
				// Use previous cycle
				if len(cycles) >= 2 {
					lastCompleted = cycles[1]
				} else {
					lastCompleted = cycles[0]
				}
			}

			cycleID, _ := lastCompleted["id"].(string)
			cycleNum, _ := lastCompleted["number"].(float64)
			issues := getIssuesForCycle(db, cycleID)

			var incomplete []map[string]any
			for _, iss := range issues {
				state, _ := iss["state"].(map[string]any)
				stateType, _ := state["type"].(string)
				if stateType != "completed" && stateType != "canceled" {
					incomplete = append(incomplete, iss)
				}
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"cycle_number":     int(cycleNum),
					"total_issues":     len(issues),
					"incomplete_count": len(incomplete),
					"incomplete":       incomplete,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Carry-over from Cycle %d (%d of %d incomplete):\n\n",
				int(cycleNum), len(incomplete), len(issues))

			if len(incomplete) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "  All issues completed!")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %-40s %-15s\n", "ID", "TITLE", "STATE")
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %-40s %-15s\n", "----", "-----", "-----")
			for _, iss := range incomplete {
				id, _ := iss["identifier"].(string)
				title, _ := iss["title"].(string)
				state, _ := iss["state"].(map[string]any)
				stateName, _ := state["name"].(string)
				fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %-40s %-15s\n",
					id, truncate(title, 40), stateName)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&team, "team", "", "Filter by team key or name")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

// Helper functions

func openDB(dbPath string) (*store.Store, string) {
	if dbPath == "" {
		home, _ := os.UserHomeDir()
		dbPath = filepath.Join(home, ".config", "linear-pp-cli", "store.db")
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return nil, dbPath
	}
	return db, dbPath
}

func findCurrentCycle(db *store.Store, teamID string) (map[string]any, error) {
	query := `SELECT data FROM resources WHERE resource_type = 'cycles' ORDER BY json_extract(data, '$.startsAt') DESC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		var data []byte
		if rows.Scan(&data) != nil {
			continue
		}
		var obj map[string]any
		if json.Unmarshal(data, &obj) != nil {
			continue
		}

		if teamID != "" {
			t, _ := obj["team"].(map[string]any)
			tid, _ := t["id"].(string)
			if tid != teamID {
				continue
			}
		}

		startsAt, _ := obj["startsAt"].(string)
		endsAt, _ := obj["endsAt"].(string)
		start, err1 := time.Parse(time.RFC3339, startsAt)
		end, err2 := time.Parse(time.RFC3339, endsAt)
		if err1 != nil || err2 != nil {
			continue
		}
		if now.After(start) && now.Before(end) {
			return obj, nil
		}
	}
	return nil, fmt.Errorf("no current cycle")
}

func getIssuesForCycle(db *store.Store, cycleID string) []map[string]any {
	rows, err := db.Query(`SELECT data FROM issues WHERE cycle_id = ?`, cycleID)
	if err != nil {
		// fallback to resources table
		rows, err = db.Query(
			`SELECT data FROM resources WHERE resource_type = 'issues' AND json_extract(data, '$.cycle.id') = ?`,
			cycleID)
		if err != nil {
			return nil
		}
	}
	defer rows.Close()

	var issues []map[string]any
	for rows.Next() {
		var data []byte
		if rows.Scan(&data) != nil {
			continue
		}
		var obj map[string]any
		if json.Unmarshal(data, &obj) == nil {
			issues = append(issues, obj)
		}
	}
	return issues
}

func getRecentCycles(db *store.Store, teamID string, n int) []map[string]any {
	query := `SELECT data FROM resources WHERE resource_type = 'cycles' ORDER BY json_extract(data, '$.startsAt') DESC`
	rows, err := db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var cycles []map[string]any
	for rows.Next() {
		var data []byte
		if rows.Scan(&data) != nil {
			continue
		}
		var obj map[string]any
		if json.Unmarshal(data, &obj) != nil {
			continue
		}
		if teamID != "" {
			t, _ := obj["team"].(map[string]any)
			tid, _ := t["id"].(string)
			if tid != teamID {
				continue
			}
		}
		cycles = append(cycles, obj)
		if len(cycles) >= n {
			break
		}
	}
	return cycles
}

func progressBar(pct float64, width int) string {
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

func formatDate(s string) string {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Format("Jan 2")
	}
	if len(s) > 10 {
		return s[:10]
	}
	return s
}

// classifyState maps state type or name to a simple category.
func classifyState(stateType, stateName string) string {
	// If we have the type field, use it
	switch stateType {
	case "completed", "canceled":
		return "done"
	case "started":
		return "active"
	case "backlog", "triage", "unstarted":
		return "todo"
	}
	// Fall back to name matching
	lower := strings.ToLower(stateName)
	switch {
	case lower == "done" || lower == "canceled" || lower == "cancelled" || lower == "duplicate" || lower == "closed":
		return "done"
	case lower == "in progress" || lower == "in review" || lower == "started" || strings.Contains(lower, "progress") || strings.Contains(lower, "review"):
		return "active"
	default:
		return "todo"
	}
}
