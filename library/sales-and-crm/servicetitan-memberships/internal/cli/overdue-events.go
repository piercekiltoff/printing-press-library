package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newOverdueEventsCmd(flags *rootFlags) *cobra.Command {
	var days int
	var limit int
	var dbPath string

	cmd := &cobra.Command{
		Use:   "overdue-events",
		Short: "Recurring-service events past their date on still-active memberships",
		Long: "Lists recurring-service events whose date is before today and whose\n" +
			"status is not Completed, on memberships that are still active. The\n" +
			"--days flag caps the lookback so years-old never-completed events\n" +
			"don't drown the report (default: 365 days). Run 'sync' first.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli overdue-events --days 90
  servicetitan-memberships-pp-cli overdue-events --limit 25 --json
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

			rows, err := memberships.OverdueEvents(db, days)
			if err != nil {
				return err
			}
			rows = capRows(rows, limit)
			if rows == nil {
				rows = []memberships.OverdueEventRow{}
			}
			table := make([][]string, 0, len(rows))
			for _, r := range rows {
				table = append(table, []string{
					i64(r.EventID), i64(r.MembershipID), r.MembershipName,
					i64(r.LocationRecurringServiceID), r.ServiceName,
					r.Status, r.Date, strconv.Itoa(r.DaysOverdue),
				})
			}
			return mbOutput(cmd, flags, rows,
				[]string{"EVENT", "MEMBERSHIP", "MEMBERSHIP NAME", "SVC ID", "SERVICE", "STATUS", "DATE", "DAYS OVERDUE"},
				table)
		},
	}
	cmd.Flags().IntVar(&days, "days", 365, "Maximum days past today to include (caps how far back to look)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum rows to return (0 = all)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}
