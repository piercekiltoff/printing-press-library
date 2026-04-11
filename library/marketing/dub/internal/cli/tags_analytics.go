package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/mvanhorn/printing-press-library/library/marketing/dub/internal/store"
	"github.com/spf13/cobra"
)

func newTagsAnalyticsCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var limit int

	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "Analytics rollup per tag — which tags drive the most conversions",
		Long: `Aggregates clicks, leads, and sales across all links for each tag.
The Dub analytics API groups by link, country, or device — but not by tag.
This command joins tags and links locally to show tag-level performance.`,
		Example: `  # Tag analytics rollup
  dub-pp-cli tags analytics

  # Top 10 tags as JSON
  dub-pp-cli tags analytics --limit 10 --json`,
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
					json_extract(t.data, '$.name') AS tag_name,
					json_extract(t.data, '$.color') AS tag_color,
					COUNT(DISTINCT l.id) AS link_count,
					COALESCE(SUM(CAST(json_extract(l.data, '$.clicks') AS INTEGER)), 0) AS clicks,
					COALESCE(SUM(CAST(json_extract(l.data, '$.leads') AS INTEGER)), 0) AS leads,
					COALESCE(SUM(CAST(json_extract(l.data, '$.sales') AS INTEGER)), 0) AS sales
				FROM tags t
				LEFT JOIN links l ON (
					json_extract(l.data, '$.tagId') = t.id
					OR instr(json_extract(l.data, '$.tags'), json_extract(t.data, '$.name')) > 0
				)
				GROUP BY t.id, tag_name
				ORDER BY clicks DESC
				LIMIT ?
			`

			rows, err := s.Query(query, limit)
			if err != nil {
				return fmt.Errorf("querying tag analytics: %w", err)
			}
			defer rows.Close()

			type tagAnalytics struct {
				TagName   string `json:"tag_name"`
				TagColor  string `json:"tag_color"`
				LinkCount int    `json:"link_count"`
				Clicks    int    `json:"clicks"`
				Leads     int    `json:"leads"`
				Sales     int    `json:"sales"`
			}

			var results []tagAnalytics
			for rows.Next() {
				var r tagAnalytics
				if err := rows.Scan(&r.TagName, &r.TagColor, &r.LinkCount, &r.Clicks, &r.Leads, &r.Sales); err != nil {
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
				fmt.Fprintln(cmd.OutOrStdout(), "No tag data. Run 'sync --full' to populate.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "TAG\tCOLOR\tLINKS\tCLICKS\tLEADS\tSALES")
			for _, r := range results {
				fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%d\n",
					r.TagName, r.TagColor, r.LinkCount, r.Clicks, r.Leads, r.Sales)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	cmd.Flags().IntVar(&limit, "limit", 25, "Max tags to show")

	return cmd
}
