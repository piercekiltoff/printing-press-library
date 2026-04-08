package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/project-management/linear/internal/store"
	"github.com/spf13/cobra"
)

func newLeaderboardCmd(flags *rootFlags) *cobra.Command {
	var weeks int
	var team string
	var user string
	var sortBy string
	var dbPath string
	var limit int

	cmd := &cobra.Command{
		Use:   "leaderboard",
		Short: "Rank team members by issue activity from local data",
		Long: `Analyze locally synced data to rank team members by issues created,
closed, picked up (self-assigned), and other activity metrics.
Requires a prior sync: run 'linear-pp-cli workflow archive' first.`,
		Example: `  # Team leaderboard for last 4 weeks
  linear-pp-cli leaderboard --weeks 4

  # Filter by team
  linear-pp-cli leaderboard --team ENG --weeks 4

  # Individual stats
  linear-pp-cli leaderboard --user "Matt" --weeks 8

  # Sort by closed issues
  linear-pp-cli leaderboard --sort closed --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".config", "linear-pp-cli", "store.db")
			}

			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w\nRun 'linear-pp-cli workflow archive' first.", err)
			}
			defer db.Close()

			cutoff := time.Now().AddDate(0, 0, -weeks*7).Format(time.RFC3339)

			// Build user name lookup from synced users
			userNames := map[string]string{}
			userRows, err := db.Query(`SELECT id, data FROM resources WHERE resource_type = 'users'`)
			if err == nil {
				defer userRows.Close()
				for userRows.Next() {
					var uid string
					var data []byte
					if userRows.Scan(&uid, &data) == nil {
						var obj map[string]any
						if json.Unmarshal(data, &obj) == nil {
							if n, ok := obj["name"].(string); ok {
								userNames[uid] = n
							}
						}
					}
				}
			}

			// Query issues from the issues table (has domain columns)
			query := `SELECT data FROM issues WHERE 1=1`
			var qArgs []any

			if team != "" {
				// Look up team ID by key or name
				teamID := resolveTeamID(db, team)
				if teamID != "" {
					query += ` AND team_id = ?`
					qArgs = append(qArgs, teamID)
				}
			}

			rows, err := db.Query(query, qArgs...)
			if err != nil {
				return fmt.Errorf("querying issues: %w", err)
			}
			defer rows.Close()

			type stats struct {
				Name     string `json:"name"`
				ID       string `json:"id"`
				Created  int    `json:"created"`
				Closed   int    `json:"closed"`
				Assigned int    `json:"assigned"`
				Total    int    `json:"total_active"`
			}

			board := map[string]*stats{}

			for rows.Next() {
				var data []byte
				if rows.Scan(&data) != nil {
					continue
				}
				var obj map[string]any
				if json.Unmarshal(data, &obj) != nil {
					continue
				}

				// Get assignee
				assigneeID := ""
				if a, ok := obj["assignee"].(map[string]any); ok {
					if id, ok := a["id"].(string); ok {
						assigneeID = id
					}
				}

				// Get creator (creatorId field)
				creatorID := ""
				if c, ok := obj["creator"].(map[string]any); ok {
					if id, ok := c["id"].(string); ok {
						creatorID = id
					}
				}

				createdAt, _ := obj["createdAt"].(string)
				completedAt, _ := obj["completedAt"].(string)

				// Ensure user entry exists
				ensureEntry := func(uid string) {
					if uid == "" {
						return
					}
					if _, ok := board[uid]; !ok {
						name := userNames[uid]
						if name == "" {
							name = uid[:8] + "..."
						}
						board[uid] = &stats{Name: name, ID: uid}
					}
				}

				// Count created issues in window
				if creatorID != "" && createdAt >= cutoff {
					ensureEntry(creatorID)
					if s, ok := board[creatorID]; ok {
						s.Created++
					}
				}

				// Count closed (completed) issues in window
				if assigneeID != "" && completedAt != "" && completedAt >= cutoff {
					ensureEntry(assigneeID)
					if s, ok := board[assigneeID]; ok {
						s.Closed++
					}
				}

				// Count assigned (picked up) — issue assigned to someone
				if assigneeID != "" {
					ensureEntry(assigneeID)
					if s, ok := board[assigneeID]; ok {
						s.Assigned++
					}
				}

				// Total active for assignee (not completed)
				if assigneeID != "" && completedAt == "" {
					ensureEntry(assigneeID)
					if s, ok := board[assigneeID]; ok {
						s.Total++
					}
				}
			}

			// Filter by user if specified
			if user != "" {
				user = strings.ToLower(user)
				for uid, s := range board {
					if !strings.Contains(strings.ToLower(s.Name), user) {
						delete(board, uid)
					}
				}
			}

			// Convert to slice and sort
			var entries []*stats
			for _, s := range board {
				entries = append(entries, s)
			}

			switch sortBy {
			case "created":
				sort.Slice(entries, func(i, j int) bool { return entries[i].Created > entries[j].Created })
			case "closed":
				sort.Slice(entries, func(i, j int) bool { return entries[i].Closed > entries[j].Closed })
			case "assigned", "picked":
				sort.Slice(entries, func(i, j int) bool { return entries[i].Assigned > entries[j].Assigned })
			default: // "score" - weighted composite
				sort.Slice(entries, func(i, j int) bool {
					si := entries[i].Closed*3 + entries[i].Created*2 + entries[i].Assigned
					sj := entries[j].Closed*3 + entries[j].Created*2 + entries[j].Assigned
					return si > sj
				})
			}

			if limit > 0 && len(entries) > limit {
				entries = entries[:limit]
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(entries)
			}

			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No activity found. Run 'workflow archive' to sync data first.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Leaderboard (last %d weeks):\n\n", weeks)
			fmt.Fprintf(cmd.OutOrStdout(), "  %-4s %-25s %8s %8s %8s %8s\n",
				"RANK", "NAME", "CLOSED", "CREATED", "ASSIGNED", "ACTIVE")
			fmt.Fprintf(cmd.OutOrStdout(), "  %-4s %-25s %8s %8s %8s %8s\n",
				"----", "----", "------", "-------", "--------", "------")
			for i, e := range entries {
				name := e.Name
				if len(name) > 25 {
					name = name[:22] + "..."
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-4d %-25s %8d %8d %8d %8d\n",
					i+1, name, e.Closed, e.Created, e.Assigned, e.Total)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&weeks, "weeks", 4, "Time window in weeks")
	cmd.Flags().StringVar(&team, "team", "", "Filter by team key or name")
	cmd.Flags().StringVar(&user, "user", "", "Filter to a specific user (partial name match)")
	cmd.Flags().StringVar(&sortBy, "sort", "score", "Sort by: score, closed, created, assigned")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum entries to show")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")

	return cmd
}

// resolveTeamID looks up a team ID by key or name from the local store.
func resolveTeamID(db *store.Store, keyOrName string) string {
	rows, err := db.Query(
		`SELECT id, data FROM resources WHERE resource_type = 'teams'`)
	if err != nil {
		return keyOrName // fallback to using as-is
	}
	defer rows.Close()

	lower := strings.ToLower(keyOrName)
	for rows.Next() {
		var id string
		var data []byte
		if rows.Scan(&id, &data) != nil {
			continue
		}
		var obj map[string]any
		if json.Unmarshal(data, &obj) != nil {
			continue
		}
		if key, ok := obj["key"].(string); ok && strings.ToLower(key) == lower {
			return id
		}
		if name, ok := obj["name"].(string); ok && strings.ToLower(name) == lower {
			return id
		}
	}
	return keyOrName
}
