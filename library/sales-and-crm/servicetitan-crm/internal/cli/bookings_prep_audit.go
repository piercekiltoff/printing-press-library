// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.
// Hand-written transcendence command (Phase 3): booking prep-audit.

package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-crm/internal/store"
)

type prepFinding struct {
	BookingID    any      `json:"booking_id"`
	BookingStart string   `json:"start,omitempty"`
	LocationID   any      `json:"location_id,omitempty"`
	Address      string   `json:"address,omitempty"`
	Missing      []string `json:"missing"` // confirmed_phone | gate_code | required_tag
}

// newBookingsPrepAuditCmd builds `bookings prep-audit --window 1d` —
// surfaces bookings whose linked location is missing prep info that
// dispatch needs before the truck rolls.
func newBookingsPrepAuditCmd(flags *rootFlags) *cobra.Command {
	var (
		dbPath  string
		window  string
		needTag string
		limit   int
	)
	cmd := &cobra.Command{
		Use:   "prep-audit",
		Short: "Bookings missing contact methods, gate codes, or required tags (dispatch ritual)",
		Long: `Returns bookings in the upcoming --window whose linked location is
missing one or more prep items: a confirmed contact method, a gate-code
special-instruction, or a required tag (e.g., 'dog-on-property').

Reads only from the local SQLite store — run 'sync run' first.`,
		Example: strings.Trim(`
  servicetitan-crm-pp-cli bookings prep-audit --window 1d --json
  servicetitan-crm-pp-cli bookings prep-audit --window 7d --json --select findings.address,findings.missing
  servicetitan-crm-pp-cli bookings prep-audit --window 1d --need-tag dog-on-property --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:novel": "bookings-prep-audit"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			win, err := parseAgeFlag(window)
			if err != nil {
				return usageErr(fmt.Errorf("--window: %w", err))
			}
			if dbPath == "" {
				dbPath = defaultDBPath("servicetitan-crm-pp-cli")
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			// Bookings starting within window
			start := time.Now().UTC().Format(time.RFC3339)
			end := time.Now().Add(win).UTC().Format(time.RFC3339)
			rows, err := db.DB().QueryContext(cmd.Context(), `
				SELECT data FROM resources
				WHERE resource_type = 'bookings'
				  AND json_extract(data, '$.start') >= ?
				  AND json_extract(data, '$.start') <= ?
			`, start, end)
			if err != nil {
				return fmt.Errorf("scan bookings: %w", err)
			}
			defer rows.Close()

			findings := []prepFinding{}
			for rows.Next() {
				var raw json.RawMessage
				if err := rows.Scan(&raw); err != nil {
					continue
				}
				var b map[string]any
				if err := json.Unmarshal(raw, &b); err != nil {
					continue
				}
				locID, _ := toInt(b["locationId"])
				loc, _ := getResourceJSON(cmd.Context(), db, "locations", locID)
				missing := []string{}
				if !locationHasConfirmedPhone(cmd.Context(), db, locID) {
					missing = append(missing, "confirmed_phone")
				}
				if !locationHasGateCode(loc) {
					missing = append(missing, "gate_code")
				}
				if needTag != "" && !locationHasTag(cmd.Context(), db, locID, needTag) {
					missing = append(missing, "tag:"+needTag)
				}
				if len(missing) == 0 {
					continue
				}
				addr := ""
				if loc != nil {
					if a, ok := loc["address"].(map[string]any); ok {
						street, _ := a["street"].(string)
						city, _ := a["city"].(string)
						addr = strings.TrimSpace(street + ", " + city)
					}
				}
				findings = append(findings, prepFinding{
					BookingID:    b["id"],
					BookingStart: asString(b["start"]),
					LocationID:   locID,
					Address:      addr,
					Missing:      missing,
				})
				if limit > 0 && len(findings) >= limit {
					break
				}
			}

			out := map[string]any{
				"window":    window,
				"start_utc": start,
				"end_utc":   end,
				"need_tag":  needTag,
				"count":     len(findings),
				"findings":  findings,
			}
			if flags.asJSON || (!isTerminal(cmd.OutOrStdout())) {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Booking prep-audit (window=%s): %d under-prepped\n", window, len(findings))
			for _, f := range findings {
				fmt.Fprintf(cmd.OutOrStdout(), "  booking=%v  start=%s  loc=%s  missing=%v\n",
					f.BookingID, f.BookingStart, f.Address, f.Missing)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "SQLite database path")
	cmd.Flags().StringVar(&window, "window", "1d", "Look-ahead window for upcoming bookings (e.g., 1d, 7d)")
	cmd.Flags().StringVar(&needTag, "need-tag", "", "Required tag the location must have (optional)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max findings to return (0 = unlimited)")
	return cmd
}

func getResourceJSON(ctx interface{ Done() <-chan struct{} }, db *store.Store, resourceType string, id int) (map[string]any, error) {
	if id == 0 {
		return nil, nil
	}
	row, err := db.Get(resourceType, fmt.Sprint(id))
	if err != nil || len(row) == 0 {
		return nil, err
	}
	var m map[string]any
	return m, json.Unmarshal(row, &m)
}

func locationHasConfirmedPhone(ctx interface{ Done() <-chan struct{} }, db *store.Store, locationID int) bool {
	if locationID == 0 {
		return false
	}
	row := db.DB().QueryRow(`
		SELECT COUNT(*) FROM contact_methods
		WHERE CAST(json_extract(data, '$.locationId') AS INTEGER) = ?
		  AND LOWER(json_extract(data, '$.type')) = 'phone'
		  AND COALESCE(json_extract(data, '$.confirmed'), 0) = 1
	`, locationID)
	var n int
	_ = row.Scan(&n)
	return n > 0
}

func locationHasGateCode(loc map[string]any) bool {
	if loc == nil {
		return false
	}
	si, ok := loc["specialInstructions"].(string)
	if !ok || si == "" {
		return false
	}
	low := strings.ToLower(si)
	return strings.Contains(low, "gate") || strings.Contains(low, "code")
}

func locationHasTag(ctx interface{ Done() <-chan struct{} }, db *store.Store, locationID int, tag string) bool {
	if locationID == 0 || tag == "" {
		return false
	}
	row := db.DB().QueryRow(`
		SELECT COUNT(*) FROM locations_tags
		WHERE CAST(json_extract(data, '$.locationId') AS INTEGER) = ?
		  AND LOWER(json_extract(data, '$.tagName')) = LOWER(?)
	`, locationID, tag)
	var n int
	_ = row.Scan(&n)
	return n > 0
}
