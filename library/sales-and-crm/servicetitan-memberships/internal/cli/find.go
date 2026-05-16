package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newFindCmd(flags *rootFlags) *cobra.Command {
	var limit int
	var minScore float64
	var dbPath string

	cmd := &cobra.Command{
		Use:   "find <description>",
		Short: "Forgiving ranked search over synced memberships",
		Long: "Runs a forgiving ranked search across every synced membership for a\n" +
			"plain-language description. Scores each membership on customer ID,\n" +
			"importId, memo, customFields (name and value), and the joined\n" +
			"membership-type name; the best field wins. Returns the fields an ops\n" +
			"user needs to find the right customer. Results below --min-score are\n" +
			"dropped; a query that matches nothing exits non-zero (grep-style).\n" +
			"Run 'sync' first.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli find "Smith household platinum"
  servicetitan-memberships-pp-cli find "PLAT-001" --limit 5 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:typed-exit-codes": "0,1"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			query := strings.TrimSpace(strings.Join(args, " "))
			db, err := openMembershipsStore(cmd, dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			results, err := memberships.Find(db, query, minScore, limit)
			if err != nil {
				return err
			}
			if results == nil {
				results = []memberships.FindResult{}
			}
			if len(results) == 0 {
				return fmt.Errorf("no memberships matched %q at or above --min-score %.2f; try different terms or lower --min-score", query, minScore)
			}

			table := make([][]string, 0, len(results))
			for _, r := range results {
				table = append(table, []string{
					f2(r.Score), i64(r.ID), i64(r.CustomerID), r.MembershipType,
					r.Status, r.FollowUpStatus, strconv.FormatBool(r.Active),
					r.ImportID, r.MatchedOn,
				})
			}
			return mbOutput(cmd, flags, results,
				[]string{"SCORE", "ID", "CUSTOMER", "TYPE", "STATUS", "FOLLOW-UP", "ACTIVE", "IMPORT ID", "MATCHED ON"},
				table)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 15, "Maximum results to return")
	cmd.Flags().Float64Var(&minScore, "min-score", 0.4, "Minimum relevance score (0-1); results below this are dropped, and a query that matches nothing exits non-zero")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}
