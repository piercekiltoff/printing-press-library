package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newRiskCmd(flags *rootFlags) *cobra.Command {
	var limit int
	var minScore float64
	var dbPath string

	cmd := &cobra.Command{
		Use:   "risk",
		Short: "Per-membership risk score from follow-up, payment, billing, events, and to-date",
		Long: "Applies a rule-engine score to every active membership. Rules:\n" +
			"  followUpStatus != None      +0.3\n" +
			"  no paymentMethodId          +0.2\n" +
			"  nextScheduledBillDate past  +0.2\n" +
			"  no completed event in 180d  +0.2\n" +
			"  to-date within 30 days      +0.1\n" +
			"Memberships scoring below --min-score are dropped; sort is by score\n" +
			"descending. Run 'sync' first.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli risk --min-score 0.5
  servicetitan-memberships-pp-cli risk --min-score 0.3 --json --limit 20
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

			rows, err := memberships.Risk(db, minScore)
			if err != nil {
				return err
			}
			rows = capRows(rows, limit)
			if rows == nil {
				rows = []memberships.RiskRow{}
			}
			table := make([][]string, 0, len(rows))
			for _, r := range rows {
				table = append(table, []string{
					f2(r.Score), i64(r.MembershipID), i64(r.CustomerID), i64(r.MembershipTypeID),
					r.Status, r.FollowUpStatus, strings.Join(r.Reasons, "; "),
				})
			}
			return mbOutput(cmd, flags, rows,
				[]string{"SCORE", "MEMBERSHIP", "CUSTOMER", "TYPE", "STATUS", "FOLLOW-UP", "REASONS"},
				table)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum rows to return (0 = all)")
	cmd.Flags().Float64Var(&minScore, "min-score", 0.5, "Minimum risk score (0-1); rows below are dropped")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}
