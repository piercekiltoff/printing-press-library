package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newRenewalsCmd(flags *rootFlags) *cobra.Command {
	var within int
	var all bool
	var limit int
	var dbPath string

	cmd := &cobra.Command{
		Use:   "renewals",
		Short: "Active memberships whose to-date is within a window",
		Long: "Lists active memberships whose to-date falls within --within days of\n" +
			"today so the renewal task is one click away. Add --all to include\n" +
			"already-cancelled memberships for lapse-recovery sweeps. Run 'sync'\n" +
			"first.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli renewals --within 30
  servicetitan-memberships-pp-cli renewals --within 14 --json
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

			rows, err := memberships.Renewals(db, within, all)
			if err != nil {
				return err
			}
			rows = capRows(rows, limit)
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
	cmd.Flags().IntVar(&within, "within", 30, "Days from today to include (renewals due in this many days)")
	cmd.Flags().BoolVar(&all, "all", false, "Include inactive memberships (lapse-recovery sweep)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum rows to return (0 = all)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}
