package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newRevenueCmd(flags *rootFlags) *cobra.Command {
	var by string
	var dbPath string

	cmd := &cobra.Command{
		Use:   "revenue",
		Short: "Recurring revenue roll-up grouped by month, business-unit, or billing-frequency",
		Long: "Rolls up active memberships joined to their membership-type's\n" +
			"durationBilling entries. Each membership resolves to a BillingPrice +\n" +
			"SalePrice + RenewalPrice contribution, summed inside one bucket of\n" +
			"--by. Buckets where no matching durationBilling row exists still\n" +
			"count toward MembershipCount so the population is honest. Run 'sync'\n" +
			"first.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli revenue --by month
  servicetitan-memberships-pp-cli revenue --by business-unit --json
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

			rows, err := memberships.Revenue(db, by)
			if err != nil {
				return err
			}
			if rows == nil {
				rows = []memberships.RevenueRow{}
			}
			table := make([][]string, 0, len(rows))
			for _, r := range rows {
				table = append(table, []string{
					r.GroupKey, strconv.Itoa(r.MembershipCount),
					f2(r.BillingTotal), f2(r.SaleTotal), f2(r.RenewalTotal),
				})
			}
			return mbOutput(cmd, flags, rows,
				[]string{"GROUP", "MEMBERSHIPS", "BILLING", "SALE", "RENEWAL"},
				table)
		},
	}
	cmd.Flags().StringVar(&by, "by", "month", "Group key: month, business-unit, or billing-frequency")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}
