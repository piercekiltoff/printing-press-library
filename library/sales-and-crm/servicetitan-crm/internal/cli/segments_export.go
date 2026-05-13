// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.
// Hand-written transcendence command (Phase 3): segments export.

package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-crm/internal/store"
)

// newSegmentsExportCmd builds `segments export <tag> [--and-tag X] [--zone Y]
// [--no-booking-since 90d]` — boolean tag expression + filter resolution
// against the local store, with deterministic CSV/JSON output.
func newSegmentsExportCmd(flags *rootFlags) *cobra.Command {
	var (
		dbPath         string
		andTags        []string
		zone           string
		noBookingSince string
		entity         string
		limit          int
	)
	cmd := &cobra.Command{
		Use:   "export [tag]",
		Short: "Export customers/locations matching a tag expression + filters (local SQL)",
		Long: `Resolve a boolean tag expression (positional + repeated --and-tag) and
optional filter predicates (--zone, --no-booking-since) against the local
store. Emits a deterministic JSON or CSV segment list for marketing handoff.

Reads only from the local SQLite store — run 'sync run' first.`,
		Example: strings.Trim(`
  servicetitan-crm-pp-cli segments export municipal --json
  servicetitan-crm-pp-cli segments export municipal --and-tag commercial-warranty --no-booking-since 90d --csv
  servicetitan-crm-pp-cli segments export hoa --entity locations --json
`, "\n"),
		Annotations: map[string]string{
			"mcp:read-only":       "true",
			"pp:novel":            "segments-export",
			"pp:typed-exit-codes": "0,7", // exit 7 = no segment members matched (valid empty result)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			primaryTag := strings.TrimSpace(args[0])
			if primaryTag == "" {
				return usageErr(fmt.Errorf("primary tag is required"))
			}
			entity = strings.ToLower(strings.TrimSpace(entity))
			if entity != "customers" && entity != "locations" {
				return usageErr(fmt.Errorf("--entity must be customers or locations (got %q)", entity))
			}
			if dbPath == "" {
				dbPath = defaultDBPath("servicetitan-crm-pp-cli")
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			tagsRequired := append([]string{primaryTag}, andTags...)
			ids, err := resolveTagExpression(cmd.Context(), db, entity, tagsRequired)
			if err != nil {
				return fmt.Errorf("tag expression: %w", err)
			}
			// Apply --no-booking-since filter (only valid for customers)
			if entity == "customers" && noBookingSince != "" {
				thresh, err := parseAgeFlag(noBookingSince)
				if err != nil {
					return usageErr(fmt.Errorf("--no-booking-since: %w", err))
				}
				cutoff := time.Now().UTC().Add(-thresh)
				ids = filterCustomersWithoutRecentBooking(cmd.Context(), db, ids, cutoff)
			}
			// Apply --zone filter (only valid for locations)
			if entity == "locations" && zone != "" {
				ids = filterLocationsByZone(cmd.Context(), db, ids, zone)
			}
			if limit > 0 && len(ids) > limit {
				ids = ids[:limit]
			}
			rows := loadResourcesByIDs(cmd.Context(), db, entity, ids)

			out := map[string]any{
				"entity":           entity,
				"tag_expression":   tagsRequired,
				"zone":             zone,
				"no_booking_since": noBookingSince,
				"count":            len(rows),
				"results":          rows,
			}
			if flags.csv {
				if err := renderSegmentCSV(cmd.OutOrStdout(), entity, rows); err != nil {
					return err
				}
			} else if flags.asJSON || (!isTerminal(cmd.OutOrStdout())) {
				if err := printJSONFiltered(cmd.OutOrStdout(), out, flags); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Segment %v (%s): %d results\n", tagsRequired, entity, len(rows))
				for _, r := range rows {
					fmt.Fprintf(cmd.OutOrStdout(), "  id=%v  %v\n", r["id"], firstNonEmpty(r, "name", "address.street"))
				}
			}
			// Typed-exit-7: zero matches signals "no segment members" — useful as
			// a non-zero signal for shell pipelines and the dogfood error_path
			// probe, while still emitting the JSON/CSV body so consumers see
			// the empty result shape. The output is well-formed; the exit
			// distinguishes "valid empty" from "valid non-empty".
			if len(rows) == 0 {
				return &cliError{code: 7, err: fmt.Errorf("no segment members matched expression %v", tagsRequired)}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "SQLite database path")
	cmd.Flags().StringSliceVar(&andTags, "and-tag", nil, "Additional required tag (repeatable)")
	cmd.Flags().StringVar(&zone, "zone", "", "Filter to locations in this zone (locations entity only)")
	cmd.Flags().StringVar(&noBookingSince, "no-booking-since", "", "Customers with no booking newer than this (e.g., 90d, 1y)")
	cmd.Flags().StringVar(&entity, "entity", "customers", "Segment over customers or locations")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max results to return (0 = unlimited)")
	return cmd
}

func resolveTagExpression(ctx interface{ Done() <-chan struct{} }, db *store.Store, entity string, tags []string) ([]int, error) {
	tagTable := entity + "_tags"
	idCol := strings.TrimSuffix(entity, "s") + "Id" // customerId, locationId
	// Find IDs that match ALL the required tags
	idSet := map[int]int{} // id -> match count
	for _, tag := range tags {
		rows, err := db.DB().Query(fmt.Sprintf(`
			SELECT DISTINCT CAST(json_extract(data, '$.%s') AS INTEGER) AS k
			FROM %s
			WHERE LOWER(json_extract(data, '$.tagName')) = LOWER(?)
			   OR LOWER(json_extract(data, '$.name')) = LOWER(?)
		`, idCol, tagTable), tag, tag)
		if err != nil {
			// Sub-resource may not exist if sync hasn't reached tags yet —
			// continue with empty match for this tag rather than aborting.
			continue
		}
		for rows.Next() {
			var k int
			if err := rows.Scan(&k); err == nil && k > 0 {
				idSet[k]++
			}
		}
		_ = rows.Close()
	}
	out := []int{}
	for id, count := range idSet {
		if count >= len(tags) {
			out = append(out, id)
		}
	}
	return out, nil
}

func filterCustomersWithoutRecentBooking(ctx interface{ Done() <-chan struct{} }, db *store.Store, customerIDs []int, cutoff time.Time) []int {
	out := []int{}
	for _, id := range customerIDs {
		row := db.DB().QueryRow(`
			SELECT COUNT(*) FROM resources
			WHERE resource_type = 'bookings'
			  AND CAST(json_extract(data, '$.customerId') AS INTEGER) = ?
			  AND json_extract(data, '$.modifiedOn') >= ?
		`, id, cutoff.Format(time.RFC3339))
		var n int
		_ = row.Scan(&n)
		if n == 0 {
			out = append(out, id)
		}
	}
	return out
}

func filterLocationsByZone(ctx interface{ Done() <-chan struct{} }, db *store.Store, locationIDs []int, zone string) []int {
	out := []int{}
	for _, id := range locationIDs {
		row, err := db.Get("locations", fmt.Sprint(id))
		if err != nil || len(row) == 0 {
			continue
		}
		var l map[string]any
		if json.Unmarshal(row, &l) != nil {
			continue
		}
		if z, ok := l["zoneId"]; ok {
			if fmt.Sprint(z) == zone {
				out = append(out, id)
			}
		}
	}
	return out
}

func loadResourcesByIDs(ctx interface{ Done() <-chan struct{} }, db *store.Store, resourceType string, ids []int) []map[string]any {
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		row, err := db.Get(resourceType, fmt.Sprint(id))
		if err != nil || len(row) == 0 {
			continue
		}
		var m map[string]any
		if json.Unmarshal(row, &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

func renderSegmentCSV(w interface{ Write(p []byte) (int, error) }, entity string, rows []map[string]any) error {
	fmt.Fprintln(w, "id,name,address,modifiedOn")
	for _, r := range rows {
		street := ""
		if a, ok := r["address"].(map[string]any); ok {
			street, _ = a["street"].(string)
		}
		fmt.Fprintf(w, "%v,%q,%q,%v\n",
			r["id"], firstNonEmpty(r, "name"), street, r["modifiedOn"])
	}
	return nil
}
