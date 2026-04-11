package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/mvanhorn/printing-press-library/library/marketing/dub/internal/store"
	"github.com/spf13/cobra"
)

func newDomainsReportCmd(flags *rootFlags) *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Domain utilization report — links and clicks per domain",
		Long: `Shows which custom domains are over- or underused by joining domains,
links, and analytics locally. Helps identify domains worth keeping vs.
domains generating no traffic.`,
		Example: `  # Domain utilization report
  dub-pp-cli domains report

  # As JSON
  dub-pp-cli domains report --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				dbPath = defaultDBPath("dub-pp-cli")
			}
			s, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer s.Close()

			query := `
				SELECT
					json_extract(d.data, '$.slug') AS domain_slug,
					COALESCE(json_extract(d.data, '$.verified'), 'false') AS verified,
					COALESCE(json_extract(d.data, '$.primary'), 'false') AS is_primary,
					COUNT(DISTINCT l.id) AS link_count,
					COALESCE(SUM(CAST(json_extract(l.data, '$.clicks') AS INTEGER)), 0) AS total_clicks,
					COALESCE(SUM(CAST(json_extract(l.data, '$.leads') AS INTEGER)), 0) AS total_leads,
					COALESCE(SUM(CAST(json_extract(l.data, '$.sales') AS INTEGER)), 0) AS total_sales
				FROM domains d
				LEFT JOIN links l ON json_extract(l.data, '$.domain') = json_extract(d.data, '$.slug')
				GROUP BY d.id
				ORDER BY total_clicks DESC
			`

			rows, err := s.Query(query)
			if err != nil {
				return fmt.Errorf("querying domain report: %w", err)
			}
			defer rows.Close()

			type domainReport struct {
				Domain      string `json:"domain"`
				Verified    string `json:"verified"`
				Primary     string `json:"primary"`
				LinkCount   int    `json:"link_count"`
				TotalClicks int    `json:"total_clicks"`
				TotalLeads  int    `json:"total_leads"`
				TotalSales  int    `json:"total_sales"`
			}

			var results []domainReport
			for rows.Next() {
				var r domainReport
				if err := rows.Scan(&r.Domain, &r.Verified, &r.Primary, &r.LinkCount, &r.TotalClicks, &r.TotalLeads, &r.TotalSales); err != nil {
					return fmt.Errorf("scanning row: %w", err)
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
				fmt.Fprintln(cmd.OutOrStdout(), "No domain data. Run 'sync --full' to populate.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "DOMAIN\tVERIFIED\tPRIMARY\tLINKS\tCLICKS\tLEADS\tSALES")
			for _, r := range results {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\t%d\n",
					r.Domain, r.Verified, r.Primary, r.LinkCount,
					r.TotalClicks, r.TotalLeads, r.TotalSales)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")

	return cmd
}
