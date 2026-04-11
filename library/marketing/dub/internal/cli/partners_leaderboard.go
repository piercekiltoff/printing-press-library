package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/mvanhorn/printing-press-library/library/marketing/dub/internal/store"
	"github.com/spf13/cobra"
)

func newPartnersLeaderboardCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var limit int
	var sortBy string

	cmd := &cobra.Command{
		Use:   "leaderboard",
		Short: "Rank partners by commission earned and conversion performance",
		Long: `Ranks partners by their earnings, conversion rates, and click volume.
Joins partners, commissions, and link data locally to produce a cross-partner
comparison that no single API call provides.`,
		Example: `  # Partner leaderboard
  dub-pp-cli partners leaderboard

  # Top 10 by earnings as JSON
  dub-pp-cli partners leaderboard --limit 10 --sort earnings --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				dbPath = defaultDBPath("dub-pp-cli")
			}
			s, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer s.Close()

			orderCol := "total_earnings"
			switch sortBy {
			case "clicks":
				orderCol = "clicks"
			case "sales":
				orderCol = "sales"
			case "commissions":
				orderCol = "commission_count"
			case "name":
				orderCol = "partner_name"
			}

			query := fmt.Sprintf(`
				SELECT
					p.id,
					json_extract(p.data, '$.name') AS partner_name,
					json_extract(p.data, '$.email') AS partner_email,
					COALESCE(json_extract(p.data, '$.status'), 'active') AS status,
					COALESCE((
						SELECT COUNT(*) FROM commissions c
						WHERE json_extract(c.data, '$.partnerId') = p.id
					), 0) AS commission_count,
					COALESCE((
						SELECT SUM(CAST(json_extract(c.data, '$.amount') AS REAL))
						FROM commissions c
						WHERE json_extract(c.data, '$.partnerId') = p.id
					), 0) AS total_earnings,
					COALESCE(CAST(json_extract(p.data, '$.clicks') AS INTEGER), 0) AS clicks,
					COALESCE(CAST(json_extract(p.data, '$.leads') AS INTEGER), 0) AS leads,
					COALESCE(CAST(json_extract(p.data, '$.sales') AS INTEGER), 0) AS sales
				FROM partners p
				ORDER BY %s DESC
				LIMIT ?
			`, orderCol)

			rows, err := s.Query(query, limit)
			if err != nil {
				return fmt.Errorf("querying leaderboard: %w", err)
			}
			defer rows.Close()

			type partnerRank struct {
				ID              string  `json:"id"`
				Name            string  `json:"name"`
				Email           string  `json:"email"`
				Status          string  `json:"status"`
				CommissionCount int     `json:"commission_count"`
				TotalEarnings   float64 `json:"total_earnings"`
				Clicks          int     `json:"clicks"`
				Leads           int     `json:"leads"`
				Sales           int     `json:"sales"`
			}

			var results []partnerRank
			for rows.Next() {
				var r partnerRank
				if err := rows.Scan(&r.ID, &r.Name, &r.Email, &r.Status, &r.CommissionCount, &r.TotalEarnings, &r.Clicks, &r.Leads, &r.Sales); err != nil {
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
				fmt.Fprintln(cmd.OutOrStdout(), "No partner data. Run 'sync --full' to populate.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "#\tPARTNER\tSTATUS\tCLICKS\tLEADS\tSALES\tCOMMISSIONS\tEARNINGS")
			for i, r := range results {
				name := r.Name
				if name == "" {
					name = r.Email
				}
				if len(name) > 25 {
					name = name[:22] + "..."
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%d\t%d\t%d\t$%.2f\n",
					i+1, name, r.Status, r.Clicks, r.Leads, r.Sales,
					r.CommissionCount, r.TotalEarnings/100)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().IntVar(&limit, "limit", 25, "Max partners to show")
	cmd.Flags().StringVar(&sortBy, "sort", "earnings", "Sort by: earnings, clicks, sales, commissions, name")

	return cmd
}
