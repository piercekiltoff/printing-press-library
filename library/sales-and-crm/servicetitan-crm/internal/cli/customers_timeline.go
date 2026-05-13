// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.
// Hand-written transcendence command (Phase 3): customer timeline.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-crm/internal/store"
)

// timelineEvent is one row in the chronological customer history.
type timelineEvent struct {
	At      string         `json:"at"`             // ISO-8601 timestamp
	Kind    string         `json:"kind"`           // customer_created, location_added, booking, tag_added, contact_method_updated
	Summary string         `json:"summary"`        // human-readable description
	Source  string         `json:"source"`         // source table
	ItemID  any            `json:"item_id"`        // entity id of the source item
	Data    map[string]any `json:"data,omitempty"` // raw row data when --verbose
}

// newCustomersTimelineCmd builds `customers timeline <id>` — chronological
// event stream for one customer across creation, locations added, bookings,
// tag changes, and contact-method updates from the local store.
func newCustomersTimelineCmd(flags *rootFlags) *cobra.Command {
	var (
		dbPath  string
		limit   int
		verbose bool
	)
	cmd := &cobra.Command{
		Use:   "timeline [customer-id]",
		Short: "Chronological event stream for one customer (UNION over local tables)",
		Long: `Returns a unified time-ordered list of every event for the named customer:
creation, locations added, bookings created, tags added, contact-method updates.

Reads only from the local SQLite store — run 'sync run' first.`,
		Example: strings.Trim(`
  servicetitan-crm-pp-cli customers timeline 22448259 --json
  servicetitan-crm-pp-cli customers timeline 22448259 --json --select events.kind,events.at,events.summary
  servicetitan-crm-pp-cli customers timeline 22448259 --limit 100 --verbose --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:novel": "customer-timeline"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			customerID, err := strconv.Atoi(strings.TrimSpace(args[0]))
			if err != nil {
				return usageErr(fmt.Errorf("customer-id must be an integer: %w", err))
			}
			if dbPath == "" {
				dbPath = defaultDBPath("servicetitan-crm-pp-cli")
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			events := []timelineEvent{}
			events = append(events, collectCustomerCreationEvent(cmd.Context(), db, customerID)...)
			events = append(events, collectLocationEvents(cmd.Context(), db, customerID)...)
			events = append(events, collectBookingEvents(cmd.Context(), db, customerID)...)
			events = append(events, collectCustomerTagEvents(cmd.Context(), db, customerID)...)
			events = append(events, collectContactMethodEvents(cmd.Context(), db, customerID)...)

			sort.SliceStable(events, func(i, j int) bool { return events[i].At > events[j].At })
			if limit > 0 && len(events) > limit {
				events = events[:limit]
			}
			if !verbose {
				for i := range events {
					events[i].Data = nil
				}
			}

			out := map[string]any{
				"customer_id": customerID,
				"events":      events,
				"event_count": len(events),
			}
			if flags.asJSON || (!isTerminal(cmd.OutOrStdout())) {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			renderTimelineText(cmd.OutOrStdout(), customerID, events)
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "SQLite database path")
	cmd.Flags().IntVar(&limit, "limit", 200, "Max events to return")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Include raw row data per event")
	return cmd
}

// Helpers below pull events from each source table. Each returns []timelineEvent
// or empty on miss/error; the joined view never errors out on a single missing
// sub-resource.

func collectCustomerCreationEvent(ctx context.Context, db *store.Store, customerID int) []timelineEvent {
	row, err := db.Get("customers", strconv.Itoa(customerID))
	if err != nil || len(row) == 0 {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(row, &data); err != nil {
		return nil
	}
	created, _ := data["createdOn"].(string)
	if created == "" {
		return nil
	}
	return []timelineEvent{{
		At:      created,
		Kind:    "customer_created",
		Summary: fmt.Sprintf("Customer %v created (%v)", data["name"], data["type"]),
		Source:  "customers",
		ItemID:  customerID,
		Data:    data,
	}}
}

func collectLocationEvents(ctx context.Context, db *store.Store, customerID int) []timelineEvent {
	rows, err := db.DB().QueryContext(ctx, `
		SELECT data FROM resources
		WHERE resource_type = 'locations'
		  AND CAST(json_extract(data, '$.customerId') AS INTEGER) = ?
		LIMIT 200
	`, customerID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []timelineEvent{}
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			continue
		}
		created, _ := data["createdOn"].(string)
		if created == "" {
			continue
		}
		addr, _ := data["address"].(map[string]any)
		street, _ := addr["street"].(string)
		out = append(out, timelineEvent{
			At:      created,
			Kind:    "location_added",
			Summary: fmt.Sprintf("Location %v added (%s)", data["id"], street),
			Source:  "locations",
			ItemID:  data["id"],
			Data:    data,
		})
	}
	return out
}

func collectBookingEvents(ctx context.Context, db *store.Store, customerID int) []timelineEvent {
	rows, err := db.DB().QueryContext(ctx, `
		SELECT data FROM resources
		WHERE resource_type = 'bookings'
		  AND CAST(json_extract(data, '$.customerId') AS INTEGER) = ?
		LIMIT 500
	`, customerID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []timelineEvent{}
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			continue
		}
		when, _ := data["start"].(string)
		if when == "" {
			when, _ = data["createdOn"].(string)
		}
		if when == "" {
			continue
		}
		out = append(out, timelineEvent{
			At:      when,
			Kind:    "booking",
			Summary: fmt.Sprintf("Booking %v (%v)", data["id"], data["status"]),
			Source:  "bookings",
			ItemID:  data["id"],
			Data:    data,
		})
	}
	return out
}

func collectCustomerTagEvents(ctx context.Context, db *store.Store, customerID int) []timelineEvent {
	rows, err := db.DB().QueryContext(ctx, `
		SELECT data FROM customers_tags
		WHERE CAST(json_extract(data, '$.customerId') AS INTEGER) = ?
		LIMIT 200
	`, customerID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []timelineEvent{}
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			continue
		}
		when, _ := data["createdOn"].(string)
		if when == "" {
			continue
		}
		out = append(out, timelineEvent{
			At:      when,
			Kind:    "tag_added",
			Summary: fmt.Sprintf("Tag %v added", firstNonEmpty(data, "tagName", "name", "tagId")),
			Source:  "customers_tags",
			ItemID:  data["id"],
			Data:    data,
		})
	}
	return out
}

func collectContactMethodEvents(ctx context.Context, db *store.Store, customerID int) []timelineEvent {
	rows, err := db.DB().QueryContext(ctx, `
		SELECT data FROM contact_methods
		WHERE CAST(json_extract(data, '$.customerId') AS INTEGER) = ?
		LIMIT 200
	`, customerID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []timelineEvent{}
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			continue
		}
		when, _ := data["modifiedOn"].(string)
		if when == "" {
			when, _ = data["createdOn"].(string)
		}
		if when == "" {
			continue
		}
		out = append(out, timelineEvent{
			At:      when,
			Kind:    "contact_method_updated",
			Summary: fmt.Sprintf("Contact method (%v) %v", data["type"], data["value"]),
			Source:  "contact_methods",
			ItemID:  data["id"],
			Data:    data,
		})
	}
	return out
}

func renderTimelineText(w interface{ Write(p []byte) (int, error) }, customerID int, events []timelineEvent) {
	if len(events) == 0 {
		fmt.Fprintf(w, "No events for customer %d. Run 'sync run' first.\n", customerID)
		return
	}
	fmt.Fprintf(w, "Customer %d timeline (%d events):\n", customerID, len(events))
	for _, e := range events {
		fmt.Fprintf(w, "  %s  %-22s  %s\n", e.At, e.Kind, e.Summary)
	}
}

// firstNonEmpty returns the first non-empty string value among the named keys.
func firstNonEmpty(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
			if v != nil {
				return fmt.Sprint(v)
			}
		}
	}
	return ""
}
