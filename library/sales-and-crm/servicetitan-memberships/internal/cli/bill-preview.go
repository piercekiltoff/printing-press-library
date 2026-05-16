package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newBillPreviewCmd(flags *rootFlags) *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "bill-preview <membership-id>",
		Short: "Resolve the next bill date and line-item amount for one membership",
		Long: "Walks one membership's bill chain — membership → membership-type's\n" +
			"durationBilling → invoice-template items — and shows the next bill\n" +
			"date plus a per-line breakdown of what the next invoice will charge.\n" +
			"Gaps in the chain (no durationBilling row, no invoice template) are\n" +
			"surfaced as notes rather than errors so partial setups still produce\n" +
			"useful output. Run 'sync' first.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli bill-preview 51234
  servicetitan-memberships-pp-cli bill-preview 51234 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			id, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
			if err != nil {
				return usageErr(err)
			}
			db, err := openMembershipsStore(cmd, dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			result, err := memberships.BillPreview(db, id)
			if err != nil {
				return err
			}
			// Header table summarising the membership; item table for line items.
			headerTable := [][]string{
				{"membership_id", i64(result.MembershipID)},
				{"membership_type_id", i64(result.MembershipTypeID)},
				{"next_bill_date", result.NextBillDate},
				{"billing_frequency", result.BillingFrequency},
				{"billing_price", f2(result.BillingPrice)},
				{"sale_price", f2(result.SalePrice)},
				{"renewal_price", f2(result.RenewalPrice)},
				{"invoice_template", result.InvoiceTemplate},
				{"total", f2(result.Total)},
			}
			for _, n := range result.Notes {
				headerTable = append(headerTable, []string{"note", n})
			}
			return mbOutput(cmd, flags, result, []string{"FIELD", "VALUE"}, headerTable)
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}
