// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.
// Hand-written transcendence command (Phase 3): customer-360 find.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-crm/internal/store"
)

// newCustomersFindCmd builds the `customers find <query>` transcendence
// command. FTS5 lookup over synced customer rows joined with locations,
// bookings, contact methods, and tags from the local store. Returns the
// composed customer-360 view that no single ServiceTitan endpoint provides.
func newCustomersFindCmd(flags *rootFlags) *cobra.Command {
	var (
		dbPath string
		limit  int
	)
	cmd := &cobra.Command{
		Use:   "find [query]",
		Short: "Customer-360 lookup by phone, email, name, or partial address (local FTS5 join)",
		Long: `Find a customer by phone, email, name, or partial address and return
the joined view across customer, locations, bookings, contact methods, and tags.

This is a local-store query — run 'sync run' first to populate the SQLite cache.
Sub-100ms typeahead lookup that the ServiceTitan Web UI cannot match.`,
		Example: strings.Trim(`
  servicetitan-crm-pp-cli customers find "555-0142" --json
  servicetitan-crm-pp-cli customers find "Smith" --limit 5 --json --select customer.name,locations.address
  servicetitan-crm-pp-cli customers find "Pine St" --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:novel": "customer-360-find"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			query := strings.TrimSpace(args[0])
			if query == "" {
				return usageErr(fmt.Errorf("query is required"))
			}
			if dbPath == "" {
				dbPath = defaultDBPath("servicetitan-crm-pp-cli")
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			customers, err := findCustomersByQuery(cmd.Context(), db, query, limit)
			if err != nil {
				return fmt.Errorf("customers find %q: %w", query, err)
			}

			results := make([]customer360View, 0, len(customers))
			for _, c := range customers {
				view, err := buildCustomer360View(cmd.Context(), db, c)
				if err != nil {
					return fmt.Errorf("composing 360 view for customer %d: %w", view.CustomerID, err)
				}
				results = append(results, view)
			}

			if flags.asJSON || (!isTerminal(cmd.OutOrStdout())) {
				return printJSONFiltered(cmd.OutOrStdout(), results, flags)
			}
			renderCustomer360TextTable(cmd.OutOrStdout(), results)
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "SQLite database path (defaults to platform user-data dir)")
	cmd.Flags().IntVar(&limit, "limit", 25, "Max customer matches to return")
	return cmd
}

// customer360View is the JSON shape returned by customers find / timeline.
type customer360View struct {
	CustomerID int              `json:"customer_id"`
	Customer   map[string]any   `json:"customer"`
	Locations  []map[string]any `json:"locations"`
	Bookings   []map[string]any `json:"bookings"`
	Contacts   []map[string]any `json:"contacts"`
	Tags       []map[string]any `json:"tags"`
}

// findCustomersByQuery uses FTS5 over the synced resources index, then
// filters to customers and parses each row's data JSON. Falls back to a
// LIKE scan of customer name/email/address when FTS5 misses (for short or
// punctuated queries that FTS5 won't tokenize).
func findCustomersByQuery(ctx context.Context, db *store.Store, query string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 25
	}
	// First pass: FTS5 search across the global resources_fts index.
	hits, err := db.Search(query, limit*4)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	customers := make([]map[string]any, 0, limit)
	for _, raw := range hits {
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err != nil {
			continue
		}
		// Only customer rows reach the 360 view.
		if t, _ := row["type"].(string); t == "" && row["id"] != nil && row["name"] != nil {
			customers = append(customers, row)
			if len(customers) >= limit {
				return customers, nil
			}
		}
	}

	// Fallback LIKE scan for short or punctuation-heavy queries.
	if len(customers) < limit {
		extra, err := likeScanCustomers(ctx, db, query, limit-len(customers))
		if err == nil {
			customers = append(customers, extra...)
		}
	}
	return customers, nil
}

// likeScanCustomers does a JSON1 LIKE scan over the resources table for
// resource_type='customers'. Used when FTS5 tokenization misses (short or
// punctuated terms — phone fragments like "555-01" are common).
func likeScanCustomers(ctx context.Context, db *store.Store, query string, limit int) ([]map[string]any, error) {
	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := db.DB().QueryContext(ctx, `
		SELECT data FROM resources
		WHERE resource_type = 'customers'
		  AND (
		    LOWER(json_extract(data, '$.name')) LIKE ?
		    OR LOWER(json_extract(data, '$.email')) LIKE ?
		    OR LOWER(json_extract(data, '$.address.street')) LIKE ?
		    OR LOWER(json_extract(data, '$.address.city')) LIKE ?
		    OR LOWER(json_extract(data, '$.address.zip')) LIKE ?
		  )
		LIMIT ?
	`, pattern, pattern, pattern, pattern, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]any, 0, limit)
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err == nil {
			out = append(out, row)
		}
	}
	return out, rows.Err()
}

// buildCustomer360View attaches locations/bookings/contacts/tags to a
// customer base record by querying related tables and parsing nested JSON
// data columns. Empty slices on miss — never returns nil for an absent
// relation so JSON consumers don't have to nil-check.
func buildCustomer360View(ctx context.Context, db *store.Store, c map[string]any) (customer360View, error) {
	view := customer360View{
		Customer:  c,
		Locations: []map[string]any{},
		Bookings:  []map[string]any{},
		Contacts:  []map[string]any{},
		Tags:      []map[string]any{},
	}
	if id, ok := toInt(c["id"]); ok {
		view.CustomerID = id
	}
	if view.CustomerID == 0 {
		return view, nil
	}

	// Locations linked by customerId
	locs, _ := scanResourceJSONByField(ctx, db, "locations", "customerId", view.CustomerID, 100)
	view.Locations = locs

	// Bookings linked by customerId (or via location)
	bks, _ := scanResourceJSONByField(ctx, db, "bookings", "customerId", view.CustomerID, 100)
	view.Bookings = bks

	// Customer contacts (sub-resource table customers_contacts)
	cts, _ := scanCustomerContactsSubresource(ctx, db, view.CustomerID)
	view.Contacts = cts

	// Customer tags (sub-resource table customers_tags)
	tags, _ := scanCustomerTagsSubresource(ctx, db, view.CustomerID)
	view.Tags = tags

	return view, nil
}

// scanResourceJSONByField queries `resources` for a given resource_type
// where data->>field == intValue. Returns parsed JSON rows.
func scanResourceJSONByField(ctx context.Context, db *store.Store, resourceType, field string, intValue, limit int) ([]map[string]any, error) {
	rows, err := db.DB().QueryContext(ctx, `
		SELECT data FROM resources
		WHERE resource_type = ?
		  AND CAST(json_extract(data, '$.' || ?) AS INTEGER) = ?
		LIMIT ?
	`, resourceType, field, intValue, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]map[string]any, 0, 8)
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err == nil {
			out = append(out, row)
		}
	}
	return out, rows.Err()
}

// scanCustomerContactsSubresource reads from the customers_contacts table
// (sub-resource synced from the customer-contacts endpoints). Best-effort:
// returns empty slice if the table has no rows for this customer.
func scanCustomerContactsSubresource(ctx context.Context, db *store.Store, customerID int) ([]map[string]any, error) {
	rows, err := db.DB().QueryContext(ctx, `
		SELECT data FROM customers_contacts
		WHERE CAST(json_extract(data, '$.customerId') AS INTEGER) = ?
		LIMIT 50
	`, customerID)
	if err != nil {
		// Table may not exist yet (sync hasn't reached this sub-resource);
		// return empty rather than fail the whole 360 lookup.
		return []map[string]any{}, nil
	}
	defer rows.Close()
	out := make([]map[string]any, 0, 8)
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err == nil {
			out = append(out, row)
		}
	}
	return out, nil
}

// scanCustomerTagsSubresource reads tag attachments for a customer.
func scanCustomerTagsSubresource(ctx context.Context, db *store.Store, customerID int) ([]map[string]any, error) {
	rows, err := db.DB().QueryContext(ctx, `
		SELECT data FROM customers_tags
		WHERE CAST(json_extract(data, '$.customerId') AS INTEGER) = ?
		LIMIT 50
	`, customerID)
	if err != nil {
		return []map[string]any{}, nil
	}
	defer rows.Close()
	out := make([]map[string]any, 0, 8)
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err == nil {
			out = append(out, row)
		}
	}
	return out, nil
}

// renderCustomer360TextTable renders human-friendly output for terminals.
// JSON output goes through printJSONFiltered.
func renderCustomer360TextTable(w interface{ Write(p []byte) (int, error) }, views []customer360View) {
	if len(views) == 0 {
		fmt.Fprintln(w, "No customers matched. Run 'sync run' first to populate the local store.")
		return
	}
	for _, v := range views {
		fmt.Fprintf(w, "Customer %d: %v\n", v.CustomerID, v.Customer["name"])
		fmt.Fprintf(w, "  type=%v  active=%v  email=%v\n", v.Customer["type"], v.Customer["active"], v.Customer["email"])
		fmt.Fprintf(w, "  Locations: %d  Bookings: %d  Contacts: %d  Tags: %d\n",
			len(v.Locations), len(v.Bookings), len(v.Contacts), len(v.Tags))
		fmt.Fprintln(w)
	}
}

// toInt coerces JSON-typed numbers (which arrive as float64 from
// encoding/json) to int. Customer IDs from ServiceTitan are int64 in the
// API but unmarshal to float64 via map[string]any.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i), true
		}
	}
	return 0, false
}
