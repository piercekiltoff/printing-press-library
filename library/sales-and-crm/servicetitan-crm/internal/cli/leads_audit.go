// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.
// Hand-written transcendence command (Phase 3): lead-followup audit.

package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-crm/internal/store"
)

type auditedLead struct {
	ID         any    `json:"id"`
	CreatedOn  string `json:"created_on"`
	ModifiedOn string `json:"modified_on,omitempty"`
	Status     string `json:"status,omitempty"`
	CustomerID any    `json:"customer_id,omitempty"`
	LastTouch  string `json:"last_touch,omitempty"`
	Bucket     string `json:"bucket"` // untouched | converted | stale
}

// newLeadsAuditCmd builds `leads audit --since 30d` — buckets recent leads
// into untouched / converted / stale based on the lead lifecycle.
func newLeadsAuditCmd(flags *rootFlags) *cobra.Command {
	var (
		dbPath string
		since  string
		limit  int
	)
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Bucket recent leads as untouched / converted / stale (local pipeline view)",
		Long: `Returns leads from the last --since window, bucketed by lifecycle status:
  - untouched: created but never modified, no contact, no conversion
  - converted: linked to a customer record (lead.convertedToCustomerId set)
  - stale:     modified before the freshness cutoff and not converted

This answers "which leads have been worked vs. dropped" — impossible in the
ServiceTitan Web UI without an Excel pivot.

Reads only from the local SQLite store — run 'sync run' first.`,
		Example: strings.Trim(`
  servicetitan-crm-pp-cli leads audit --since 30d --json
  servicetitan-crm-pp-cli leads audit --since 7d --json --select buckets,counts
  servicetitan-crm-pp-cli leads audit --since 90d --limit 50 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:novel": "leads-audit"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			window, err := parseAgeFlag(since)
			if err != nil {
				return usageErr(fmt.Errorf("--since: %w", err))
			}
			cutoff := time.Now().UTC().Add(-window)
			// Stale freshness: modified > 14d after creation but before cutoff
			staleAge := 14 * 24 * time.Hour

			if dbPath == "" {
				dbPath = defaultDBPath("servicetitan-crm-pp-cli")
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			rows, err := db.DB().QueryContext(cmd.Context(), `
				SELECT data FROM resources
				WHERE resource_type = 'leads'
				  AND json_extract(data, '$.createdOn') >= ?
			`, cutoff.Format(time.RFC3339))
			if err != nil {
				return fmt.Errorf("scan leads: %w", err)
			}
			defer rows.Close()

			untouched := []auditedLead{}
			converted := []auditedLead{}
			stale := []auditedLead{}
			for rows.Next() {
				var raw json.RawMessage
				if err := rows.Scan(&raw); err != nil {
					continue
				}
				var l map[string]any
				if err := json.Unmarshal(raw, &l); err != nil {
					continue
				}
				al := auditedLead{
					ID:         l["id"],
					CreatedOn:  asString(l["createdOn"]),
					ModifiedOn: asString(l["modifiedOn"]),
					Status:     asString(l["status"]),
					CustomerID: l["customerId"],
				}
				custID, custOK := toInt(l["customerId"])
				switch {
				case custOK && custID > 0:
					al.Bucket = "converted"
					al.LastTouch = al.ModifiedOn
					converted = append(converted, al)
				case al.ModifiedOn == "" || al.ModifiedOn == al.CreatedOn:
					al.Bucket = "untouched"
					untouched = append(untouched, al)
				default:
					if t, err := time.Parse(time.RFC3339, al.ModifiedOn); err == nil {
						if time.Since(t) > staleAge {
							al.Bucket = "stale"
							al.LastTouch = al.ModifiedOn
							stale = append(stale, al)
						}
					}
				}
			}
			capLimit(&untouched, limit)
			capLimit(&converted, limit)
			capLimit(&stale, limit)

			out := map[string]any{
				"since":      since,
				"cutoff_utc": cutoff.Format(time.RFC3339),
				"counts": map[string]int{
					"untouched": len(untouched),
					"converted": len(converted),
					"stale":     len(stale),
					"total":     len(untouched) + len(converted) + len(stale),
				},
				"buckets": map[string]any{
					"untouched": untouched,
					"converted": converted,
					"stale":     stale,
				},
			}
			if flags.asJSON || (!isTerminal(cmd.OutOrStdout())) {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Lead audit (since %s):\n", since)
			fmt.Fprintf(cmd.OutOrStdout(), "  untouched: %d  converted: %d  stale: %d\n",
				len(untouched), len(converted), len(stale))
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "SQLite database path")
	cmd.Flags().StringVar(&since, "since", "30d", "Window: leads created within this duration (e.g., 7d, 30d, 90d)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max items per bucket (0 = unlimited)")
	return cmd
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func capLimit[T any](items *[]T, limit int) {
	if limit > 0 && len(*items) > limit {
		*items = (*items)[:limit]
	}
}
