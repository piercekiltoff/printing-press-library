package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newExpiringCmd(flags *rootFlags) *cobra.Command {
	var within int
	var dbPath string

	cmd := &cobra.Command{
		Use:   "expiring",
		Short: "Every membership whose to-date falls inside a window, including cancelled",
		Long: "Lists every membership — active or already cancelled — whose to-date\n" +
			"falls within --within days of today. The lapse-recovery sweep that\n" +
			"shows the renewal funnel + churn tail in one query. Run 'sync' first.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli expiring --within 60
  servicetitan-memberships-pp-cli expiring --within 30 --json
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

			rows, err := memberships.Expiring(db, within)
			if err != nil {
				return err
			}
			if rows == nil {
				rows = []memberships.RenewalRow{}
			}
			table := make([][]string, 0, len(rows))
			for _, r := range rows {
				table = append(table, []string{
					i64(r.ID), i64(r.CustomerID), i64(r.MembershipTypeID),
					r.Status, r.FollowUpStatus, r.To,
					strconv.Itoa(r.DaysUntil), strconv.FormatBool(r.Active),
				})
			}
			return mbOutput(cmd, flags, rows,
				[]string{"ID", "CUSTOMER", "TYPE", "STATUS", "FOLLOW-UP", "TO", "DAYS", "ACTIVE"},
				table)
		},
	}
	cmd.Flags().IntVar(&within, "within", 30, "Days from today to include")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}
