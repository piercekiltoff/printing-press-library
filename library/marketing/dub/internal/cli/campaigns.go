package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/mvanhorn/printing-press-library/library/marketing/dub/internal/store"
	"github.com/spf13/cobra"
)

func newCampaignsCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var limit int
	var sortBy string

	cmd := &cobra.Command{
		Use:   "campaigns",
		Short: "Campaign performance dashboard — analytics aggregated by tag",
		Long: `Aggregates link analytics by tag to show campaign-level performance.
Requires synced data (run 'sync --full' first). Groups all links sharing a
tag and sums their clicks, leads, and sales to produce a per-campaign view
that the Dub dashboard cannot show natively.`,
		Example: `  # Show campaign performance
  dub-pp-cli campaigns

  # Sort by sales descending
  dub-pp-cli campaigns --sort sales

  # Top 5 campaigns as JSON
  dub-pp-cli campaigns --limit 5 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				dbPath = defaultDBPath("dub-pp-cli")
			}
			s, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer s.Close()

			orderCol := "clicks"
			switch sortBy {
			case "leads":
				orderCol = "leads"
			case "sales":
				orderCol = "sales"
			case "links":
				orderCol = "link_count"
			case "name":
				orderCol = "tag_name"
			}

			query := fmt.Sprintf(`
				SELECT
					t.id,
					json_extract(t.data, '$.name') AS tag_name,
					json_extract(t.data, '$.color') AS tag_color,
					COUNT(DISTINCT l.id) AS link_count,
					COALESCE(SUM(CAST(json_extract(l.data, '$.clicks') AS INTEGER)), 0) AS clicks,
					COALESCE(SUM(CAST(json_extract(l.data, '$.leads') AS INTEGER)), 0) AS leads,
					COALESCE(SUM(CAST(json_extract(l.data, '$.sales') AS INTEGER)), 0) AS sales,
					COALESCE(SUM(CAST(json_extract(l.data, '$.saleAmount') AS REAL)), 0) AS sale_amount
				FROM tags t
				LEFT JOIN links l ON (
					json_extract(l.data, '$.tagId') = t.id
					OR instr(json_extract(l.data, '$.tags'), json_extract(t.data, '$.name')) > 0
				)
				GROUP BY t.id, tag_name
				ORDER BY %s DESC
				LIMIT ?
			`, orderCol)

			rows, err := s.Query(query, limit)
			if err != nil {
				return fmt.Errorf("querying campaigns: %w", err)
			}
			defer rows.Close()

			type campaign struct {
				TagID      string  `json:"tag_id"`
				TagName    string  `json:"tag_name"`
				TagColor   string  `json:"tag_color"`
				LinkCount  int     `json:"link_count"`
				Clicks     int     `json:"clicks"`
				Leads      int     `json:"leads"`
				Sales      int     `json:"sales"`
				SaleAmount float64 `json:"sale_amount"`
			}

			var results []campaign
			for rows.Next() {
				var c campaign
				if err := rows.Scan(&c.TagID, &c.TagName, &c.TagColor, &c.LinkCount, &c.Clicks, &c.Leads, &c.Sales, &c.SaleAmount); err != nil {
					return fmt.Errorf("scanning row: %w", err)
				}
				results = append(results, c)
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
				fmt.Fprintln(cmd.OutOrStdout(), "No campaign data. Run 'sync --full' to populate.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "TAG\tLINKS\tCLICKS\tLEADS\tSALES\tREVENUE")
			for _, c := range results {
				fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t$%.2f\n",
					c.TagName, c.LinkCount, c.Clicks, c.Leads, c.Sales, c.SaleAmount/100)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().IntVar(&limit, "limit", 25, "Max campaigns to show")
	cmd.Flags().StringVar(&sortBy, "sort", "clicks", "Sort by: clicks, leads, sales, links, name")

	return cmd
}
