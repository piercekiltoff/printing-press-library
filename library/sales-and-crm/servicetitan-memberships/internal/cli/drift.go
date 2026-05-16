package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newDriftCmd(flags *rootFlags) *cobra.Command {
	var limit int
	var dbPath string

	cmd := &cobra.Command{
		Use:   "drift",
		Short: "Compare each active membership's recurring-services against its type template",
		Long: "For every active membership, compares the attached recurring-services\n" +
			"against the membership-type's recurringServices[] template and flags\n" +
			"memberships with missing or extra service type IDs. Memberships that\n" +
			"point at a membership-type ID no longer in the store are surfaced as\n" +
			"'missing-type'. Run 'sync' first.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli drift
  servicetitan-memberships-pp-cli drift --json --limit 50
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			db, err := openMembershipsStore(cmd, dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			rows, err := memberships.Drift(db)
			if err != nil {
				return err
			}
			rows = capRows(rows, limit)
			if rows == nil {
				rows = []memberships.DriftRow{}
			}
			table := make([][]string, 0, len(rows))
			for _, r := range rows {
				table = append(table, []string{
					i64(r.MembershipID), i64(r.MembershipTypeID), r.Reason,
					formatIDs(r.Missing), formatIDs(r.Extra),
				})
			}
			return mbOutput(cmd, flags, rows,
				[]string{"MEMBERSHIP", "TYPE", "REASON", "MISSING", "EXTRA"},
				table)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum rows to return (0 = all)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}

// formatIDs renders an int64 slice as a comma-separated string for table cells.
// An empty slice renders empty (not "[]") so columns stay readable.
func formatIDs(ids []int64) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d", id))
	}
	return strings.Join(parts, ",")
}
