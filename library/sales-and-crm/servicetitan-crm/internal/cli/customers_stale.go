// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.
// Hand-written transcendence command (Phase 3): stale-customer scan.

package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-crm/internal/store"
)

// newCustomersStaleCmd builds `customers stale --no-activity 365d` —
// customers whose latest booking, contact-method update, and notes are all
// older than the threshold. Useful for re-engagement campaigns or pruning.
func newCustomersStaleCmd(flags *rootFlags) *cobra.Command {
	var (
		dbPath     string
		noActivity string
		limit      int
	)
	cmd := &cobra.Command{
		Use:   "stale",
		Short: "List customers with no activity in the last N days/weeks/months (local join)",
		Long: `Returns customers whose latest booking, contact-method update, and notes are
ALL older than --no-activity. Useful for re-engagement targeting or pruning
inactive customers from segment lists.

Reads only from the local SQLite store — run 'sync run' first.`,
		Example: strings.Trim(`
  servicetitan-crm-pp-cli customers stale --no-activity 365d --json
  servicetitan-crm-pp-cli customers stale --no-activity 90d --csv
  servicetitan-crm-pp-cli customers stale --no-activity 180d --limit 100 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:novel": "customers-stale"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			thresh, err := parseAgeFlag(noActivity)
			if err != nil {
				return usageErr(fmt.Errorf("--no-activity: %w", err))
			}
			cutoff := time.Now().UTC().Add(-thresh)
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

			stale := []map[string]any{}
			for rows.Next() {
				var raw json.RawMessage
				if err := rows.Scan(&raw); err != nil {
					continue
				}
				var c map[string]any
				if err := json.Unmarshal(raw, &c); err != nil {
					continue
				}
				if isCustomerStale(cmd.Context(), db, c, cutoff) {
					stale = append(stale, c)
					if limit > 0 && len(stale) >= limit {
						break
					}
				}
			}

			out := map[string]any{
				"no_activity_threshold": noActivity,
				"cutoff_utc":            cutoff.Format(time.RFC3339),
				"stale_count":           len(stale),
				"customers":             stale,
			}
			if flags.asJSON || (!isTerminal(cmd.OutOrStdout())) {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Stale customers (no activity since %s): %d\n",
				cutoff.Format(time.RFC3339), len(stale))
			for _, c := range stale {
				fmt.Fprintf(cmd.OutOrStdout(), "  id=%v  name=%v  modifiedOn=%v\n",
					c["id"], c["name"], c["modifiedOn"])
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "SQLite database path")
	cmd.Flags().StringVar(&noActivity, "no-activity", "365d", "Activity-free window (e.g., 30d, 6m, 1y)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max customers to return (0 = unlimited)")
	return cmd
}

// isCustomerStale checks the customer's modifiedOn AND latest related
// booking/contact-method/note timestamps; returns true only if EVERYTHING
// is older than the cutoff.
func isCustomerStale(ctx interface{ Done() <-chan struct{} }, db *store.Store, c map[string]any, cutoff time.Time) bool {
	if cm, ok := c["modifiedOn"].(string); ok {
		if t, err := time.Parse(time.RFC3339, cm); err == nil && t.After(cutoff) {
			return false
		}
	}
	id, ok := toInt(c["id"])
	if !ok {
		return false
	}
	// Check latest booking modifiedOn for this customer
	var latestBooking *time.Time
	row := db.DB().QueryRow(`
		SELECT MAX(json_extract(data, '$.modifiedOn'))
		FROM resources
		WHERE resource_type = 'bookings'
		  AND CAST(json_extract(data, '$.customerId') AS INTEGER) = ?
	`, id)
	var s any
	if err := row.Scan(&s); err == nil && s != nil {
		if str, ok := s.(string); ok {
			if t, err := time.Parse(time.RFC3339, str); err == nil {
				latestBooking = &t
			}
		}
	}
	if latestBooking != nil && latestBooking.After(cutoff) {
		return false
	}
	// All checks passed → stale
	return true
}

// parseAgeFlag converts shorthand like "30d", "6m", "1y", "168h" into a
// time.Duration. Stricter than time.ParseDuration which doesn't support
// d/w/m/y units.
func parseAgeFlag(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, fmt.Errorf("empty age value")
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	// Custom suffix handling
	last := s[len(s)-1]
	num := s[:len(s)-1]
	var mult time.Duration
	switch last {
	case 'd':
		mult = 24 * time.Hour
	case 'w':
		mult = 7 * 24 * time.Hour
	case 'm':
		mult = 30 * 24 * time.Hour
	case 'y':
		mult = 365 * 24 * time.Hour
	default:
		return 0, fmt.Errorf("unrecognized age suffix in %q (want d/w/m/y or Go duration like 168h)", s)
	}
	var n int
	if _, err := fmt.Sscanf(num, "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid number in age value %q: %w", s, err)
	}
	return time.Duration(n) * mult, nil
}
