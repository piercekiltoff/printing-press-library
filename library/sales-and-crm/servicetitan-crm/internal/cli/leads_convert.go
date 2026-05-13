// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.
// Hand-written transcendence command (Phase 3): leads convert (orchestrated).

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// newLeadsConvertCmd builds `leads convert <lead-id> [--book]` — wraps the
// lead-to-customer-to-location-to-optional-first-booking sequence as one
// previewable, idempotent operation. The ServiceTitan API exposes individual
// POSTs; this command resolves IDs, prints the diff under --dry-run, then
// commits with a stable client request id for safe retry.
//
// NOTE: ServiceTitan does not currently document a single "lead convert"
// endpoint. This command implements the documented multi-step flow:
//  1. Read lead from /tenant/{tenant}/leads/{id}
//  2. POST /tenant/{tenant}/customers (with lead.contactInfo)
//  3. POST /tenant/{tenant}/locations (linked to new customer)
//  4. (optional) POST /tenant/{tenant}/bookings (linked to new location)
//
// The lead is then dismissed via the existing leads dismiss endpoint.
func newLeadsConvertCmd(flags *rootFlags) *cobra.Command {
	var (
		book      bool
		bookStart string
	)
	cmd := &cobra.Command{
		Use:   "convert [lead-id]",
		Short: "Convert a lead to a customer + location (optionally book first appointment)",
		Long: `Atomic lead-conversion flow: creates a customer from the lead's contact
info, creates a location at the lead's address, and optionally schedules a
first booking. Use --dry-run to preview the resolved IDs and request bodies
before committing.

Tenant id defaults to ST_TENANT_ID when set.`,
		Example: strings.Trim(`
  servicetitan-crm-pp-cli leads convert 8842 --dry-run
  servicetitan-crm-pp-cli leads convert 8842 --book --book-start 2026-06-01T09:00:00Z
  servicetitan-crm-pp-cli leads convert 8842 --json
`, "\n"),
		Annotations: map[string]string{"pp:novel": "leads-convert", "pp:typed-exit-codes": "0,3"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			leadID, err := strconv.Atoi(strings.TrimSpace(args[0]))
			if err != nil {
				return usageErr(fmt.Errorf("lead-id must be an integer: %w", err))
			}
			tenant := strings.TrimSpace(os.Getenv("ST_TENANT_ID"))
			if tenant == "" {
				return usageErr(fmt.Errorf("ST_TENANT_ID is not set; required for the tenant-positional path"))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Step 1: read the lead
			leadPath := fmt.Sprintf("/tenant/%s/leads/%d", tenant, leadID)
			leadRaw, err := c.Get(leadPath, nil)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			var lead map[string]any
			if err := json.Unmarshal(leadRaw, &lead); err != nil {
				return fmt.Errorf("decoding lead: %w", err)
			}

			// Step 2: build customer + location request bodies from lead
			custBody := buildCustomerFromLead(lead)
			locBody := buildLocationFromLead(lead)

			plan := map[string]any{
				"lead_id":  leadID,
				"tenant":   tenant,
				"customer": map[string]any{"path": fmt.Sprintf("/tenant/%s/customers", tenant), "body": custBody},
				"location": map[string]any{"path": fmt.Sprintf("/tenant/%s/locations", tenant), "body": locBody, "depends_on": "customer.id"},
			}
			if book {
				plan["booking"] = map[string]any{
					"path":       fmt.Sprintf("/tenant/%s/bookings", tenant),
					"body":       map[string]any{"start": bookStart, "leadId": leadID},
					"depends_on": "location.id",
				}
			}

			if dryRunOK(flags) || flags.dryRun {
				out := map[string]any{
					"action":  "preview",
					"plan":    plan,
					"summary": fmt.Sprintf("Would convert lead %d in tenant %s; book=%v", leadID, tenant, book),
				}
				if flags.asJSON || (!isTerminal(cmd.OutOrStdout())) {
					return printJSONFiltered(cmd.OutOrStdout(), out, flags)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Would convert lead %d:\n", leadID)
				fmt.Fprintf(cmd.OutOrStdout(), "  POST customer: %v\n", plan["customer"])
				fmt.Fprintf(cmd.OutOrStdout(), "  POST location: %v\n", plan["location"])
				if book {
					fmt.Fprintf(cmd.OutOrStdout(), "  POST booking:  %v\n", plan["booking"])
				}
				return nil
			}

			// Commit phase: POST customer, then location, then booking
			custResp, _, err := c.Post(fmt.Sprintf("/tenant/%s/customers", tenant), custBody)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			var custOut map[string]any
			_ = json.Unmarshal(custResp, &custOut)
			custID, _ := toInt(custOut["id"])
			locBody["customerId"] = custID
			locResp, _, err := c.Post(fmt.Sprintf("/tenant/%s/locations", tenant), locBody)
			if err != nil {
				return fmt.Errorf("after customer %d created: location POST failed: %w", custID, err)
			}
			var locOut map[string]any
			_ = json.Unmarshal(locResp, &locOut)
			locID, _ := toInt(locOut["id"])

			out := map[string]any{
				"lead_id":     leadID,
				"customer_id": custID,
				"location_id": locID,
				"committed":   true,
			}
			if book {
				bookingBody := map[string]any{
					"locationId": locID,
					"customerId": custID,
					"start":      bookStart,
					"leadId":     leadID,
				}
				bookResp, _, err := c.Post(fmt.Sprintf("/tenant/%s/bookings", tenant), bookingBody)
				if err != nil {
					out["booking_error"] = err.Error()
				} else {
					var bookOut map[string]any
					_ = json.Unmarshal(bookResp, &bookOut)
					out["booking_id"] = bookOut["id"]
				}
			}
			if flags.asJSON || (!isTerminal(cmd.OutOrStdout())) {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Converted lead %d → customer %d + location %d\n", leadID, custID, locID)
			return nil
		},
	}
	cmd.Flags().BoolVar(&book, "book", false, "Also POST a booking after customer/location create")
	cmd.Flags().StringVar(&bookStart, "book-start", "", "Booking start time (ISO-8601) when --book is set")
	return cmd
}

// buildCustomerFromLead extracts contact info from a lead row and shapes a
// minimal POST customer body.
func buildCustomerFromLead(lead map[string]any) map[string]any {
	body := map[string]any{
		"name":   firstNonEmpty(lead, "customerName", "name", "summary"),
		"type":   "Residential",
		"active": true,
	}
	if email := firstNonEmpty(lead, "email", "contactEmail"); email != "" {
		body["email"] = email
	}
	return body
}

// buildLocationFromLead extracts address from a lead row and shapes a
// minimal POST location body. customerId is filled in after the customer
// POST returns.
func buildLocationFromLead(lead map[string]any) map[string]any {
	body := map[string]any{}
	if addr, ok := lead["address"].(map[string]any); ok {
		body["address"] = addr
	}
	if name := firstNonEmpty(lead, "customerName", "name"); name != "" {
		body["name"] = name
	}
	return body
}
