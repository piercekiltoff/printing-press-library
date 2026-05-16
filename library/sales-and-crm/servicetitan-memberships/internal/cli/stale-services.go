package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newStaleServicesCmd(flags *rootFlags) *cobra.Command {
	var months int
	var limit int
	var dbPath string

	cmd := &cobra.Command{
		Use:   "stale-services",
		Short: "Active recurring-services with no completed event in N+ months",
		Long: "Lists active recurring-services attached to active memberships that\n" +
			"have had no completed event in the past --months months — the\n" +
			"recurrences that should have already happened but haven't. Services\n" +
			"that have never been completed sort to the top. Run 'sync' first.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli stale-services --months 6
  servicetitan-memberships-pp-cli stale-services --months 12 --json
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

			rows, err := memberships.StaleServices(db, months)
			if err != nil {
				return err
			}
			rows = capRows(rows, limit)
			if rows == nil {
				rows = []memberships.StaleServiceRow{}
			}
			table := make([][]string, 0, len(rows))
			for _, r := range rows {
				since := strconv.Itoa(r.DaysSinceCompleted)
				if r.DaysSinceCompleted < 0 {
					since = "never"
				}
				table = append(table, []string{
					i64(r.RecurringServiceID), r.Name, i64(r.MembershipID),
					r.LastCompletedDate, since,
				})
			}
			return mbOutput(cmd, flags, rows,
				[]string{"SERVICE", "NAME", "MEMBERSHIP", "LAST COMPLETED", "DAYS SINCE"},
				table)
		},
	}
	cmd.Flags().IntVar(&months, "months", 6, "Threshold: services without a completed event in this many months")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum rows to return (0 = all)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}
