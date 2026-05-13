// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.
// Hand-written transcendence command (Phase 3): customer dedupe finder.

package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-crm/internal/store"
)

type dedupeCluster struct {
	Key       string           `json:"key"`       // normalized phone/email/address
	By        string           `json:"by"`        // phone | email | address
	Score     int              `json:"score"`     // member count (cluster size)
	Customers []map[string]any `json:"customers"` // matching customer rows
}

// newCustomersDedupeCmd builds `customers dedupe --by phone|email|address` —
// surfaces likely duplicate customer records by GROUP BY normalized field.
func newCustomersDedupeCmd(flags *rootFlags) *cobra.Command {
	var (
		dbPath  string
		by      string
		minSize int
		limit   int
	)
	cmd := &cobra.Command{
		Use:   "dedupe",
		Short: "Find likely duplicate customer records by phone/email/address",
		Long: `Detect duplicate customer records by grouping on normalized phone, email,
or address. Returns clusters of size >= --min-size (default 2) ranked by
overlap strength.

Reads only from the local SQLite store — run 'sync run' first.`,
		Example: strings.Trim(`
  servicetitan-crm-pp-cli customers dedupe --by phone --json
  servicetitan-crm-pp-cli customers dedupe --by email --min-size 3 --json
  servicetitan-crm-pp-cli customers dedupe --by address --limit 25 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:novel": "customer-dedupe"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			by = strings.ToLower(strings.TrimSpace(by))
			if by != "phone" && by != "email" && by != "address" {
				return usageErr(fmt.Errorf("--by must be one of: phone, email, address (got %q)", by))
			}
			if dbPath == "" {
				dbPath = defaultDBPath("servicetitan-crm-pp-cli")
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			rows, err := db.DB().QueryContext(cmd.Context(),
				`SELECT data FROM resources WHERE resource_type = 'customers'`)
			if err != nil {
				return fmt.Errorf("scan customers: %w", err)
			}
			defer rows.Close()

			groups := map[string][]map[string]any{}
			for rows.Next() {
				var raw json.RawMessage
				if err := rows.Scan(&raw); err != nil {
					continue
				}
				var c map[string]any
				if err := json.Unmarshal(raw, &c); err != nil {
					continue
				}
				key := normalizeForDedupe(by, c)
				if key == "" {
					continue
				}
				groups[key] = append(groups[key], c)
			}

			clusters := make([]dedupeCluster, 0, len(groups))
			for key, members := range groups {
				if len(members) < minSize {
					continue
				}
				clusters = append(clusters, dedupeCluster{
					Key:       key,
					By:        by,
					Score:     len(members),
					Customers: members,
				})
			}
			// rank descending by Score
			for i := 0; i < len(clusters); i++ {
				for j := i + 1; j < len(clusters); j++ {
					if clusters[j].Score > clusters[i].Score {
						clusters[i], clusters[j] = clusters[j], clusters[i]
					}
				}
			}
			if limit > 0 && len(clusters) > limit {
				clusters = clusters[:limit]
			}

			out := map[string]any{
				"by":             by,
				"min_size":       minSize,
				"cluster_count":  len(clusters),
				"customer_count": sumClusterMembers(clusters),
				"clusters":       clusters,
			}
			if flags.asJSON || (!isTerminal(cmd.OutOrStdout())) {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			renderDedupeText(cmd.OutOrStdout(), clusters, by)
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "SQLite database path")
	cmd.Flags().StringVar(&by, "by", "phone", "Field to group by: phone | email | address")
	cmd.Flags().IntVar(&minSize, "min-size", 2, "Minimum cluster size to report")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max clusters to return (0 = unlimited)")
	return cmd
}

func normalizeForDedupe(by string, c map[string]any) string {
	switch by {
	case "phone":
		// Customers have many phone variants; pick the first non-empty one
		// from address.country phone or similar. ServiceTitan customers often
		// have separate primary/secondary phones; this is a best-effort scan.
		for _, key := range []string{"phone", "primaryPhone", "phoneNumber"} {
			if v, ok := c[key].(string); ok && v != "" {
				return normalizePhone(v)
			}
		}
		return ""
	case "email":
		if v, ok := c["email"].(string); ok && v != "" {
			return strings.ToLower(strings.TrimSpace(v))
		}
		return ""
	case "address":
		addr, _ := c["address"].(map[string]any)
		street, _ := addr["street"].(string)
		zip, _ := addr["zip"].(string)
		if street == "" {
			return ""
		}
		return strings.ToLower(strings.TrimSpace(street)) + "|" + strings.TrimSpace(zip)
	}
	return ""
}

func normalizePhone(p string) string {
	digits := make([]rune, 0, len(p))
	for _, r := range p {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	s := string(digits)
	// strip leading country code 1 for US numbers
	if len(s) == 11 && s[0] == '1' {
		s = s[1:]
	}
	return s
}

func sumClusterMembers(cs []dedupeCluster) int {
	n := 0
	for _, c := range cs {
		n += len(c.Customers)
	}
	return n
}

func renderDedupeText(w interface{ Write(p []byte) (int, error) }, clusters []dedupeCluster, by string) {
	if len(clusters) == 0 {
		fmt.Fprintf(w, "No duplicate clusters found by %s. Run 'sync run' first if the local store is empty.\n", by)
		return
	}
	fmt.Fprintf(w, "Found %d duplicate cluster(s) by %s:\n", len(clusters), by)
	for _, c := range clusters {
		fmt.Fprintf(w, "  %-20s  members=%d\n", c.Key, c.Score)
		for _, m := range c.Customers {
			fmt.Fprintf(w, "    id=%v  name=%v\n", m["id"], m["name"])
		}
	}
}
