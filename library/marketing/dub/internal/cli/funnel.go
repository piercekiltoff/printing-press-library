package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/mvanhorn/printing-press-library/library/marketing/dub/internal/store"
	"github.com/spf13/cobra"
)

func newFunnelCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var tagName string
	var linkID string

	cmd := &cobra.Command{
		Use:   "funnel",
		Short: "Attribution funnel — click to lead to sale conversion rates",
		Long: `Shows click→lead→sale conversion rates per link or tag/campaign.
Requires synced data. Joins links, events, and customers to compute
funnel metrics that no single API call can provide.`,
		Example: `  # Funnel for all links
  dub-pp-cli funnel

  # Funnel for a specific campaign tag
  dub-pp-cli funnel --tag "summer-sale"

  # Funnel for a specific link
  dub-pp-cli funnel --link-id clx1234`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				dbPath = defaultDBPath("dub-pp-cli")
			}
			s, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer s.Close()

			var query string
			var queryArgs []any

			if linkID != "" {
				query = `
					SELECT
						json_extract(l.data, '$.shortLink') AS short_link,
						json_extract(l.data, '$.url') AS destination,
						CAST(json_extract(l.data, '$.clicks') AS INTEGER) AS clicks,
						CAST(json_extract(l.data, '$.leads') AS INTEGER) AS leads,
						CAST(json_extract(l.data, '$.sales') AS INTEGER) AS sales,
						COALESCE(CAST(json_extract(l.data, '$.saleAmount') AS REAL), 0) AS sale_amount
					FROM links l
					WHERE l.id = ? OR json_extract(l.data, '$.key') = ?
				`
				queryArgs = []any{linkID, linkID}
			} else if tagName != "" {
				query = `
					SELECT
						json_extract(t.data, '$.name') AS tag_or_link,
						'(campaign total)' AS destination,
						COALESCE(SUM(CAST(json_extract(l.data, '$.clicks') AS INTEGER)), 0) AS clicks,
						COALESCE(SUM(CAST(json_extract(l.data, '$.leads') AS INTEGER)), 0) AS leads,
						COALESCE(SUM(CAST(json_extract(l.data, '$.sales') AS INTEGER)), 0) AS sales,
						COALESCE(SUM(CAST(json_extract(l.data, '$.saleAmount') AS REAL)), 0) AS sale_amount
					FROM tags t
					LEFT JOIN links l ON (
						json_extract(l.data, '$.tagId') = t.id
						OR instr(json_extract(l.data, '$.tags'), json_extract(t.data, '$.name')) > 0
					)
					WHERE json_extract(t.data, '$.name') = ?
					GROUP BY t.id
				`
				queryArgs = []any{tagName}
			} else {
				query = `
					SELECT
						json_extract(l.data, '$.shortLink') AS short_link,
						json_extract(l.data, '$.url') AS destination,
						CAST(json_extract(l.data, '$.clicks') AS INTEGER) AS clicks,
						CAST(json_extract(l.data, '$.leads') AS INTEGER) AS leads,
						CAST(json_extract(l.data, '$.sales') AS INTEGER) AS sales,
						COALESCE(CAST(json_extract(l.data, '$.saleAmount') AS REAL), 0) AS sale_amount
					FROM links l
					WHERE CAST(json_extract(l.data, '$.clicks') AS INTEGER) > 0
					ORDER BY clicks DESC
					LIMIT 25
				`
			}

			rows, err := s.Query(query, queryArgs...)
			if err != nil {
				return fmt.Errorf("querying funnel: %w", err)
			}
			defer rows.Close()

			type funnelRow struct {
				Link        string  `json:"link"`
				Dest        string  `json:"destination"`
				Clicks      int     `json:"clicks"`
				Leads       int     `json:"leads"`
				Sales       int     `json:"sales"`
				SaleAmount  float64 `json:"sale_amount"`
				ClickToLead float64 `json:"click_to_lead_pct"`
				LeadToSale  float64 `json:"lead_to_sale_pct"`
			}

			var results []funnelRow
			for rows.Next() {
				var r funnelRow
				if err := rows.Scan(&r.Link, &r.Dest, &r.Clicks, &r.Leads, &r.Sales, &r.SaleAmount); err != nil {
					return fmt.Errorf("scanning row: %w", err)
				}
				if r.Clicks > 0 {
					r.ClickToLead = float64(r.Leads) / float64(r.Clicks) * 100
				}
				if r.Leads > 0 {
					r.LeadToSale = float64(r.Sales) / float64(r.Leads) * 100
				}
				results = append(results, r)
			}
			if err := rows.Err(); err != nil {
				return err
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			if len(results) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No funnel data. Run 'sync --full' to populate.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "LINK\tCLICKS\tLEADS\tSALES\tCLICK→LEAD\tLEAD→SALE\tREVENUE")
			for _, r := range results {
				link := r.Link
				if len(link) > 40 {
					link = link[:37] + "..."
				}
				fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%.1f%%\t%.1f%%\t$%.2f\n",
					link, r.Clicks, r.Leads, r.Sales,
					r.ClickToLead, r.LeadToSale, r.SaleAmount/100)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().StringVar(&tagName, "tag", "", "Filter by tag/campaign name")
	cmd.Flags().StringVar(&linkID, "link-id", "", "Filter by specific link ID or key")

	return cmd
}
