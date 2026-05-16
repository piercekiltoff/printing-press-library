package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newScheduleCmd(flags *rootFlags) *cobra.Command {
	var within int
	var limit int
	var dbPath string

	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Upcoming recurring-service events grouped by date",
		Long: "Compact view of upcoming recurring-service events whose date falls\n" +
			"within --within days of today, sorted ascending so the next visits\n" +
			"lead. Excludes events already marked Completed. Run 'sync' first.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli schedule --within 14
  servicetitan-memberships-pp-cli schedule --within 7 --json
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

			rows, err := memberships.Schedule(db, within)
			if err != nil {
				return err
			}
			rows = capRows(rows, limit)
			if rows == nil {
				rows = []memberships.ScheduleRow{}
			}
			table := make([][]string, 0, len(rows))
			for _, r := range rows {
				table = append(table, []string{
					r.Date, i64(r.EventID), i64(r.MembershipID), r.MembershipName, r.ServiceName, r.Status,
				})
			}
			return mbOutput(cmd, flags, rows,
				[]string{"DATE", "EVENT", "MEMBERSHIP", "MEMBERSHIP NAME", "SERVICE", "STATUS"},
				table)
		},
	}
	cmd.Flags().IntVar(&within, "within", 14, "Days from today to include (upcoming window)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum rows to return (0 = all)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}
