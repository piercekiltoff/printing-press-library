package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mvanhorn/printing-press-library/library/project-management/linear/internal/store"
	"github.com/spf13/cobra"
)

func newBottleneckCmd(flags *rootFlags) *cobra.Command {
	var team string
	var dbPath string
	var limit int

	cmd := &cobra.Command{
		Use:   "bottleneck",
		Short: "Detect who/what is blocking the most issues",
		Long: `Analyze issue relations in locally synced data to find the biggest
bottlenecks — issues that block the most other issues, and people whose
assigned issues block the most work.`,
		Example: `  linear-pp-cli bottleneck --team ENG
  linear-pp-cli bottleneck --limit 10 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".config", "linear-pp-cli", "store.db")
			}
			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			// Load issue relations of type "blocks"
			relRows, err := db.Query(
				`SELECT data FROM resources WHERE resource_type = 'issue_relations'`)
			if err != nil {
				return fmt.Errorf("querying relations: %w", err)
			}
			defer relRows.Close()

			// blockerID -> list of blocked issue IDs
			blocksMap := map[string][]string{}
			for relRows.Next() {
				var data []byte
				if relRows.Scan(&data) != nil {
					continue
				}
				var obj map[string]any
				if json.Unmarshal(data, &obj) != nil {
					continue
				}
				relType, _ := obj["type"].(string)
				if relType != "blocks" {
					continue
				}
				// The issue that blocks
				issue, _ := obj["issue"].(map[string]any)
				issueID, _ := issue["id"].(string)
				// The issue being blocked
				related, _ := obj["relatedIssue"].(map[string]any)
				relatedID, _ := related["id"].(string)
				if issueID != "" && relatedID != "" {
					blocksMap[issueID] = append(blocksMap[issueID], relatedID)
				}
			}

			if len(blocksMap) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No blocking relations found. Sync relations with 'workflow archive'.")
				return nil
			}

			// Build issue lookup
			issueLookup := map[string]map[string]any{}
			issueRows, err := db.Query(`SELECT id, data FROM issues`)
			if err == nil {
				defer issueRows.Close()
				for issueRows.Next() {
					var id string
					var data []byte
					if issueRows.Scan(&id, &data) != nil {
						continue
					}
					var obj map[string]any
					if json.Unmarshal(data, &obj) == nil {
						issueLookup[id] = obj
					}
				}
			}

			// Build user name lookup
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

			// Rank blocking issues
			type blockerInfo struct {
				IssueID    string `json:"issue_id"`
				Identifier string `json:"identifier"`
				Title      string `json:"title"`
				Assignee   string `json:"assignee"`
				BlockCount int    `json:"blocks_count"`
				State      string `json:"state"`
			}

			var blockers []blockerInfo
			for issueID, blocked := range blocksMap {
				iss := issueLookup[issueID]
				identifier, _ := iss["identifier"].(string)
				title, _ := iss["title"].(string)
				assignee := ""
				if a, ok := iss["assignee"].(map[string]any); ok {
					if aid, ok := a["id"].(string); ok {
						assignee = userNames[aid]
						if assignee == "" {
							assignee = aid[:8]
						}
					}
				}
				stateName := ""
				if s, ok := iss["state"].(map[string]any); ok {
					stateName, _ = s["name"].(string)
				}

				blockers = append(blockers, blockerInfo{
					IssueID:    issueID,
					Identifier: identifier,
					Title:      title,
					Assignee:   assignee,
					BlockCount: len(blocked),
					State:      stateName,
				})
			}

			sort.Slice(blockers, func(i, j int) bool {
				return blockers[i].BlockCount > blockers[j].BlockCount
			})

			if limit > 0 && len(blockers) > limit {
				blockers = blockers[:limit]
			}

			// Also aggregate by person
			personBlocks := map[string]int{}
			for issueID, blocked := range blocksMap {
				iss := issueLookup[issueID]
				if a, ok := iss["assignee"].(map[string]any); ok {
					if aid, ok := a["id"].(string); ok {
						personBlocks[aid] += len(blocked)
					}
				}
			}

			type personInfo struct {
				Name       string `json:"name"`
				BlockCount int    `json:"total_blocks"`
			}
			var persons []personInfo
			for uid, count := range personBlocks {
				name := userNames[uid]
				if name == "" {
					name = uid[:8]
				}
				persons = append(persons, personInfo{Name: name, BlockCount: count})
			}
			sort.Slice(persons, func(i, j int) bool {
				return persons[i].BlockCount > persons[j].BlockCount
			})

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"blocking_issues": blockers,
					"blocking_people": persons,
				})
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Top Blocking Issues:")
			fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %-35s %-15s %-10s %s\n",
				"ID", "TITLE", "ASSIGNEE", "STATE", "BLOCKS")
			for _, b := range blockers {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %-35s %-15s %-10s %d\n",
					b.Identifier, truncate(b.Title, 35), truncate(b.Assignee, 15), truncate(b.State, 10), b.BlockCount)
			}

			if len(persons) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "\nBlocking by Person:")
				for _, p := range persons {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-25s %d issues blocked\n", p.Name, p.BlockCount)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&team, "team", "", "Filter by team key or name")
	cmd.Flags().IntVar(&limit, "limit", 15, "Max blocking issues to show")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}
