package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/project-management/linear/internal/store"
	"github.com/spf13/cobra"
)

func newDepsCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var depth int

	cmd := &cobra.Command{
		Use:   "deps <issue-id>",
		Short: "Show dependency graph for an issue",
		Long: `Recursively traverse issue relations in the local store to show
the full dependency chain: what blocks this issue, and what this issue blocks.
Useful for understanding critical paths.`,
		Example: `  linear-pp-cli deps LIN-123
  linear-pp-cli deps abc-uuid --depth 5 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueRef := args[0]

			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".config", "linear-pp-cli", "store.db")
			}
			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			// Resolve issue by identifier or ID
			issueID := resolveIssueID(db, issueRef)

			// Load all relations
			relRows, err := db.Query(
				`SELECT data FROM resources WHERE resource_type = 'issue_relations'`)
			if err != nil {
				return fmt.Errorf("querying relations: %w", err)
			}
			defer relRows.Close()

			// Build graph: issueID -> blocks []issueID, blockedBy []issueID
			type relation struct {
				IssueID   string
				RelatedID string
				Type      string
			}
			var relations []relation
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
				issue, _ := obj["issue"].(map[string]any)
				related, _ := obj["relatedIssue"].(map[string]any)
				iid, _ := issue["id"].(string)
				rid, _ := related["id"].(string)
				if iid != "" && rid != "" {
					relations = append(relations, relation{IssueID: iid, RelatedID: rid, Type: relType})
				}
			}

			// Build adjacency lists
			blocks := map[string][]string{}    // X blocks Y
			blockedBy := map[string][]string{} // X is blocked by Y
			for _, r := range relations {
				if r.Type == "blocks" {
					blocks[r.IssueID] = append(blocks[r.IssueID], r.RelatedID)
					blockedBy[r.RelatedID] = append(blockedBy[r.RelatedID], r.IssueID)
				}
			}

			// Build issue lookup
			issueLookup := buildIssueLookup(db)

			// BFS to find all upstream (blockers) and downstream (blocked)
			upstream := bfs(issueID, blockedBy, depth)
			downstream := bfs(issueID, blocks, depth)

			issueInfo := func(id string) map[string]any {
				iss := issueLookup[id]
				identifier, _ := iss["identifier"].(string)
				title, _ := iss["title"].(string)
				stateName := ""
				if s, ok := iss["state"].(map[string]any); ok {
					stateName, _ = s["name"].(string)
				}
				return map[string]any{
					"id":         id,
					"identifier": identifier,
					"title":      title,
					"state":      stateName,
				}
			}

			if flags.asJSON {
				var upList, downList []map[string]any
				for _, id := range upstream {
					upList = append(upList, issueInfo(id))
				}
				for _, id := range downstream {
					downList = append(downList, issueInfo(id))
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"issue":      issueInfo(issueID),
					"blocked_by": upList,
					"blocks":     downList,
				})
			}

			root := issueLookup[issueID]
			rootIdent, _ := root["identifier"].(string)
			rootTitle, _ := root["title"].(string)

			fmt.Fprintf(cmd.OutOrStdout(), "Dependency graph for %s: %s\n\n", rootIdent, truncate(rootTitle, 50))

			if len(upstream) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Blocked by (upstream):")
				printDepTree(cmd, issueLookup, upstream, "  ← ")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Blocked by: (none)")
			}

			fmt.Fprintln(cmd.OutOrStdout())

			if len(downstream) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Blocks (downstream):")
				printDepTree(cmd, issueLookup, downstream, "  → ")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Blocks: (none)")
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&depth, "depth", 10, "Max traversal depth")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func resolveIssueID(db *store.Store, ref string) string {
	// Try direct ID lookup
	rows, err := db.Query(`SELECT id FROM issues WHERE id = ? OR json_extract(data, '$.identifier') = ?`, ref, ref)
	if err != nil {
		return ref
	}
	defer rows.Close()
	if rows.Next() {
		var id string
		rows.Scan(&id)
		return id
	}
	return ref
}

func buildIssueLookup(db *store.Store) map[string]map[string]any {
	lookup := map[string]map[string]any{}
	rows, err := db.Query(`SELECT id, data FROM issues`)
	if err != nil {
		return lookup
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var data []byte
		if rows.Scan(&id, &data) != nil {
			continue
		}
		var obj map[string]any
		if json.Unmarshal(data, &obj) == nil {
			lookup[id] = obj
		}
	}
	return lookup
}

func bfs(start string, graph map[string][]string, maxDepth int) []string {
	visited := map[string]bool{start: true}
	queue := graph[start]
	var result []string
	depthMap := map[string]int{}
	for _, id := range queue {
		depthMap[id] = 1
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		d := depthMap[current]
		if d > maxDepth {
			continue
		}

		result = append(result, current)

		for _, next := range graph[current] {
			if !visited[next] {
				queue = append(queue, next)
				depthMap[next] = d + 1
			}
		}
	}
	return result
}

func printDepTree(cmd *cobra.Command, lookup map[string]map[string]any, ids []string, prefix string) {
	for _, id := range ids {
		iss := lookup[id]
		ident, _ := iss["identifier"].(string)
		title, _ := iss["title"].(string)
		stateName := ""
		if s, ok := iss["state"].(map[string]any); ok {
			stateName, _ = s["name"].(string)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s%-12s [%-12s] %s\n",
			prefix, ident, stateName, truncate(title, 45))
	}
}

// Ensure strings import is used by deps
var _ = strings.TrimSpace
