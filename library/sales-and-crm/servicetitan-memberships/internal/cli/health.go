package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newHealthCmd(flags *rootFlags) *cobra.Command {
	var within int
	var limit int
	var dbPath string

	cmd := &cobra.Command{
		Use:   "health",
		Short: "One-shot rollup of every memberships audit count",
		Long: "Aggregates renewals, overdue events, upcoming schedule, drift, risk,\n" +
			"stale services, and revenue-at-risk counts from the local store into\n" +
			"one compact rollup sized for agent priming. Run this first in a\n" +
			"memberships session to see what needs attention. It also refreshes\n" +
			"the membership-status snapshot. Run 'sync' first.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli health
  servicetitan-memberships-pp-cli health --within 14 --json
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

			rep, err := memberships.Health(db, memberships.HealthThresholds{
				WithinDays:   within,
				OverdueDays:  365,
				ScheduleDays: 14,
				StaleMonths:  6,
				RiskMinScore: 0.5,
				RevenueGroup: "business-unit",
			})
			if err != nil {
				return err
			}

			table := [][]string{
				{"memberships", strconv.Itoa(rep.Memberships)},
				{"membership_types", strconv.Itoa(rep.MembershipTypes)},
				{"recurring_services", strconv.Itoa(rep.RecurringServices)},
				{"recurring_events", strconv.Itoa(rep.RecurringEvents)},
				{"invoice_templates", strconv.Itoa(rep.InvoiceTemplates)},
				{"active_memberships", strconv.Itoa(rep.ActiveMemberships)},
				{"renewals", strconv.Itoa(rep.Renewals)},
				{"overdue_events", strconv.Itoa(rep.Overdue)},
				{"upcoming_schedule", strconv.Itoa(rep.Schedule)},
				{"drift", strconv.Itoa(rep.Drift)},
				{"risk", strconv.Itoa(rep.Risk)},
				{"stale_services", strconv.Itoa(rep.StaleServices)},
				{"revenue_at_risk", f2(rep.RevenueAtRisk)},
				{"status_snapshot_rows", strconv.Itoa(rep.StatusSnapshotRows)},
			}
			if limit > 0 && len(table) > limit {
				table = table[:limit]
			}
			return mbOutput(cmd, flags, rep, []string{"METRIC", "COUNT"}, table)
		},
	}
	cmd.Flags().IntVar(&within, "within", 30, "Renewals window in days (used for the renewals metric)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum metric rows to return (0 = all)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}
