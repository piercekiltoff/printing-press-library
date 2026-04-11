package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/mvanhorn/printing-press-library/library/marketing/dub/internal/store"
	"github.com/spf13/cobra"
)

func newLinksStaleCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var days int
	var limit int

	cmd := &cobra.Command{
		Use:   "stale",
		Short: "Find links with zero or declining clicks — stale campaign detector",
		Long: `Identifies links that have received zero clicks within the specified
time window, suggesting they may be stale or underperforming. Uses locally
synced link data including click counts and creation dates.`,
		Example: `  # Links with zero clicks in last 30 days
  dub-pp-cli links stale --days 30

  # Top 50 stale links as JSON
  dub-pp-cli links stale --days 14 --limit 50 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				dbPath = defaultDBPath("dub-pp-cli")
			}
			s, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer s.Close()

			cutoff := time.Now().AddDate(0, 0, -days).Format(time.RFC3339)

			query := `
				SELECT
					json_extract(l.data, '$.shortLink') AS short_link,
					json_extract(l.data, '$.url') AS destination,
					CAST(json_extract(l.data, '$.clicks') AS INTEGER) AS clicks,
					json_extract(l.data, '$.createdAt') AS created_at,
					json_extract(l.data, '$.lastClicked') AS last_clicked
				FROM links l
				WHERE CAST(json_extract(l.data, '$.clicks') AS INTEGER) = 0
				   OR json_extract(l.data, '$.lastClicked') IS NULL
				   OR json_extract(l.data, '$.lastClicked') < ?
				ORDER BY clicks ASC, created_at ASC
				LIMIT ?
			`

			rows, err := s.Query(query, cutoff, limit)
			if err != nil {
				return fmt.Errorf("querying stale links: %w", err)
			}
			defer rows.Close()

			type staleLink struct {
				ShortLink   string `json:"short_link"`
				Destination string `json:"destination"`
				Clicks      int    `json:"clicks"`
				CreatedAt   string `json:"created_at"`
				LastClicked string `json:"last_clicked"`
			}

			var results []staleLink
			for rows.Next() {
				var r staleLink
				var lastClicked *string
				if err := rows.Scan(&r.ShortLink, &r.Destination, &r.Clicks, &r.CreatedAt, &lastClicked); err != nil {
					return fmt.Errorf("scanning row: %w", err)
				}
				if lastClicked != nil {
					r.LastClicked = *lastClicked
				} else {
					r.LastClicked = "never"
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
				fmt.Fprintln(cmd.OutOrStdout(), "No stale links found. All links are active!")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Found %d stale links (no clicks in %d days):\n\n", len(results), days)

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "LINK\tCLICKS\tCREATED\tLAST CLICKED")
			for _, r := range results {
				link := r.ShortLink
				if len(link) > 40 {
					link = link[:37] + "..."
				}
				created := r.CreatedAt
				if len(created) > 10 {
					created = created[:10]
				}
				lastClicked := r.LastClicked
				if len(lastClicked) > 10 {
					lastClicked = lastClicked[:10]
				}
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", link, r.Clicks, created, lastClicked)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().IntVar(&days, "days", 30, "Number of days to consider a link stale")
	cmd.Flags().IntVar(&limit, "limit", 25, "Max stale links to show")

	return cmd
}
