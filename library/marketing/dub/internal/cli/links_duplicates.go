package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/mvanhorn/printing-press-library/library/marketing/dub/internal/store"
	"github.com/spf13/cobra"
)

func newLinksDuplicatesCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var limit int

	cmd := &cobra.Command{
		Use:   "duplicates",
		Short: "Find links pointing to the same destination URL",
		Long: `Scans locally synced links for duplicate destination URLs. Multiple
short links pointing to the same destination waste link budget and split
analytics. Helps identify consolidation opportunities.`,
		Example: `  # Find duplicate links
  dub-pp-cli links duplicates

  # As JSON
  dub-pp-cli links duplicates --json`,
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
					json_extract(l.data, '$.url') AS destination,
					COUNT(*) AS link_count,
					GROUP_CONCAT(json_extract(l.data, '$.shortLink'), ', ') AS short_links,
					SUM(CAST(json_extract(l.data, '$.clicks') AS INTEGER)) AS total_clicks
				FROM links l
				GROUP BY json_extract(l.data, '$.url')
				HAVING link_count > 1
				ORDER BY link_count DESC
				LIMIT ?
			`

			rows, err := s.Query(query, limit)
			if err != nil {
				return fmt.Errorf("querying duplicates: %w", err)
			}
			defer rows.Close()

			type dupGroup struct {
				Destination string `json:"destination"`
				LinkCount   int    `json:"link_count"`
				ShortLinks  string `json:"short_links"`
				TotalClicks int    `json:"total_clicks"`
			}

			var results []dupGroup
			for rows.Next() {
				var r dupGroup
				if err := rows.Scan(&r.Destination, &r.LinkCount, &r.ShortLinks, &r.TotalClicks); err != nil {
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
				fmt.Fprintln(cmd.OutOrStdout(), "No duplicate links found.")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Found %d destination URLs with multiple short links:\n\n", len(results))

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "DESTINATION\tDUPLICATES\tTOTAL CLICKS\tSHORT LINKS")
			for _, r := range results {
				dest := r.Destination
				if len(dest) > 50 {
					dest = dest[:47] + "..."
				}
				links := r.ShortLinks
				if len(links) > 50 {
					links = links[:47] + "..."
				}
				fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", dest, r.LinkCount, r.TotalClicks, links)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().IntVar(&limit, "limit", 25, "Max duplicate groups to show")

	return cmd
}
