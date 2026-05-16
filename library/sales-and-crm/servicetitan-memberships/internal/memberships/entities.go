// Package memberships holds the transcendence-feature data layer for
// servicetitan-memberships-pp-cli: typed views over the synced ServiceTitan
// Memberships entities plus the cross-entity audits, membership-status
// snapshots, bill-walk math, revenue rollups, and fuzzy-match find that the
// novel commands expose. Nothing here talks to the ServiceTitan API — it
// reads the local SQLite store that `sync` populates. The one exception
// is the complete-event path's snapshot refresh, which only reads.
package memberships

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/store"
)

// Store resource_type keys. MUST match the strings the generated list
// commands and the patched sync registry write, so every layer reads
// and writes the same rows.
const (
	ResMemberships           = "memberships"
	ResMembershipTypes       = "membership-types"
	ResRecurringServices     = "recurring-services"
	ResRecurringServiceTypes = "recurring-service-types"
	ResRecurringEvents       = "recurring-service-events"
	ResInvoiceTemplates      = "invoice-templates"
)

// Standard ServiceTitan event-status strings that the audits compare against.
// The API serializes status as a string enum; "Completed" is the only one
// these commands need to recognize as terminal.
const (
	EventStatusCompleted = "Completed"
)

// CustomField mirrors Memberships.V2.CustomFieldResponse — the per-membership
// freeform metadata Find searches across.
type CustomField struct {
	TypeID   int64  `json:"typeId"`
	TypeName string `json:"name"`
	Value    string `json:"value"`
}

// Membership mirrors the subset of Memberships.V2.CustomerMembershipResponse
// the audit commands need. ServiceTitan returns optional dates as null
// strings; the *string preserves "missing" semantics that distinguish a
// not-yet-cancelled membership from one whose to-date is unset.
type Membership struct {
	ID                      int64         `json:"id"`
	CustomerID              int64         `json:"customerId"`
	MembershipTypeID        int64         `json:"membershipTypeId"`
	LocationID              *int64        `json:"locationId"`
	BusinessUnitID          int64         `json:"businessUnitId"`
	PaymentMethodID         *int64        `json:"paymentMethodId"`
	SoldByID                *int64        `json:"soldById"`
	RecurringLocationID     *int64        `json:"recurringLocationId"`
	Active                  bool          `json:"active"`
	Status                  string        `json:"status"`
	FollowUpStatus          string        `json:"followUpStatus"`
	BillingFrequency        string        `json:"billingFrequency"`
	RenewalBillingFrequency string        `json:"renewalBillingFrequency"`
	Duration                *int          `json:"duration"`
	From                    *string       `json:"from"`
	To                      *string       `json:"to"`
	NextScheduledBillDate   *string       `json:"nextScheduledBillDate"`
	CancellationDate        *string       `json:"cancellationDate"`
	ImportID                string        `json:"importId"`
	Memo                    string        `json:"memo"`
	CustomFields            []CustomField `json:"customFields"`
	ModifiedOn              string        `json:"modifiedOn"`
}

// MembershipTypeRecurringServiceEntry is the recurring-service template
// row attached to a membership-type: "every active membership of this type
// should have a recurring-service of recurringServiceTypeId N".
type MembershipTypeRecurringServiceEntry struct {
	MembershipTypeID       int64   `json:"membershipTypeId"`
	RecurringServiceTypeID int64   `json:"recurringServiceTypeId"`
	Offset                 int     `json:"offset"`
	OffsetType             string  `json:"offsetType"`
	Allocation             float64 `json:"allocation"`
}

// MembershipTypeDurationBillingEntry is one row of a membership-type's
// duration → price ladder. ServiceTitan resolves a membership's bill amount
// by looking up its duration against this ladder.
type MembershipTypeDurationBillingEntry struct {
	Duration         int     `json:"duration"`
	BillingFrequency string  `json:"billingFrequency"`
	SalePrice        float64 `json:"salePrice"`
	BillingPrice     float64 `json:"billingPrice"`
	RenewalPrice     float64 `json:"renewalPrice"`
}

// MembershipTypeDiscount is one discount row on a membership-type. Carried
// for completeness — find/audit do not score discount text today.
type MembershipTypeDiscount struct {
	ID          int64   `json:"id"`
	SkuType     string  `json:"skuType"`
	Discount    float64 `json:"discount"`
	BusinessRef string  `json:"applicableTo"`
}

// MembershipType mirrors Memberships.V2.MembershipTypeResponse. The inline
// arrays (durationBilling, recurringServices, discounts) are what bill-preview
// and drift walk. ServiceTitan also exposes them as sub-resource tables
// (duration_billing_items, recurring_service_items, discounts) — this struct
// reads them from the inline arrays so bill-preview works with one query
// per type instead of a JOIN per type.
type MembershipType struct {
	ID                int64                                 `json:"id"`
	Name              string                                `json:"name"`
	DisplayName       string                                `json:"displayName"`
	Active            bool                                  `json:"active"`
	BillingTemplateID *int64                                `json:"billingTemplateId"`
	DurationBilling   []MembershipTypeDurationBillingEntry  `json:"durationBilling"`
	RecurringServices []MembershipTypeRecurringServiceEntry `json:"recurringServices"`
	Discounts         []MembershipTypeDiscount              `json:"discounts"`
	ImportID          string                                `json:"importId"`
	ModifiedOn        string                                `json:"modifiedOn"`
}

// RecurringService mirrors Memberships.V2.LocationRecurringServiceResponse —
// one actual recurring-service instance attached to a specific membership.
type RecurringService struct {
	ID                     int64   `json:"id"`
	Name                   string  `json:"name"`
	Active                 bool    `json:"active"`
	MembershipID           int64   `json:"membershipId"`
	LocationID             *int64  `json:"locationId"`
	RecurringServiceTypeID int64   `json:"recurringServiceTypeId"`
	DurationType           string  `json:"durationType"`
	DurationLength         int     `json:"durationLength"`
	From                   *string `json:"from"`
	To                     *string `json:"to"`
	RecurrenceType         string  `json:"recurrenceType"`
	RecurrenceInterval     int     `json:"recurrenceInterval"`
	BusinessUnitID         *int64  `json:"businessUnitId"`
	JobTypeID              *int64  `json:"jobTypeId"`
	InvoiceTemplateID      *int64  `json:"invoiceTemplateId"`
	FirstVisitComplete     bool    `json:"firstVisitComplete"`
	ModifiedOn             string  `json:"modifiedOn"`
}

// RecurringServiceType mirrors Memberships.V2.RecurringServiceTypeResponse —
// the catalog row that recurring-services point at.
type RecurringServiceType struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Active             bool   `json:"active"`
	RecurrenceType     string `json:"recurrenceType"`
	RecurrenceInterval int    `json:"recurrenceInterval"`
	BusinessUnitID     *int64 `json:"businessUnitId"`
	JobTypeID          *int64 `json:"jobTypeId"`
	DurationType       string `json:"durationType"`
	DurationLength     int    `json:"durationLength"`
	InvoiceTemplateID  *int64 `json:"invoiceTemplateId"`
}

// RecurringServiceEvent mirrors Memberships.V2.LocationRecurringServiceEventResponse —
// one scheduled visit. The audits compare Date against today and Status
// against "Completed".
type RecurringServiceEvent struct {
	ID                           int64  `json:"id"`
	LocationRecurringServiceID   int64  `json:"locationRecurringServiceId"`
	LocationRecurringServiceName string `json:"locationRecurringServiceName"`
	MembershipID                 int64  `json:"membershipId"`
	MembershipName               string `json:"membershipName"`
	Status                       string `json:"status"`
	Date                         string `json:"date"`
	JobID                        *int64 `json:"jobId"`
	CreatedOn                    string `json:"createdOn"`
	ModifiedOn                   string `json:"modifiedOn"`
}

// InvoiceTemplateItem mirrors Memberships.V2.InvoiceTemplateItemResponse —
// one line item on an invoice template. bill-preview walks these to produce
// a per-line breakdown of the next bill.
type InvoiceTemplateItem struct {
	ID          int64   `json:"id"`
	SkuID       int64   `json:"skuId"`
	SkuType     string  `json:"skuType"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	IsAddOn     bool    `json:"isAddOn"`
	Description string  `json:"description"`
	Cost        float64 `json:"cost"`
	Hours       float64 `json:"hours"`
}

// InvoiceTemplate mirrors Memberships.V2.InvoiceTemplateResponse.
type InvoiceTemplate struct {
	ID         int64                 `json:"id"`
	Name       string                `json:"name"`
	Active     bool                  `json:"active"`
	Total      float64               `json:"total"`
	Items      []InvoiceTemplateItem `json:"items"`
	ImportID   string                `json:"importId"`
	ModifiedOn string                `json:"modifiedOn"`
}

// loadRaw returns every stored JSON blob for a resource type. Unlike
// store.List it does not cap at 200 rows — the audits need the whole set.
// ORDER BY id keeps output stable across runs so deterministic dogfood
// diffs and the agentic output review see the same rows.
func loadRaw(db *store.Store, resourceType string) ([]json.RawMessage, error) {
	rows, err := db.Query(`SELECT data FROM resources WHERE resource_type = ? ORDER BY id`, resourceType)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", resourceType, err)
	}
	defer rows.Close()
	var out []json.RawMessage
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("scan %s: %w", resourceType, err)
		}
		out = append(out, json.RawMessage(data))
	}
	return out, rows.Err()
}

// LoadMemberships returns every synced membership. A row that fails to
// unmarshal is skipped rather than failing the whole load — one malformed
// blob should not blind an audit to the rest of the set.
func LoadMemberships(db *store.Store) ([]Membership, error) {
	raw, err := loadRaw(db, ResMemberships)
	if err != nil {
		return nil, err
	}
	out := make([]Membership, 0, len(raw))
	for _, r := range raw {
		var m Membership
		if json.Unmarshal(r, &m) == nil && m.ID != 0 {
			out = append(out, m)
		}
	}
	return out, nil
}

// LoadMembershipTypes returns every synced membership-type, keyed by ID for
// the drift / bill-preview joins.
func LoadMembershipTypes(db *store.Store) (map[int64]MembershipType, error) {
	raw, err := loadRaw(db, ResMembershipTypes)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]MembershipType, len(raw))
	for _, r := range raw {
		var t MembershipType
		if json.Unmarshal(r, &t) == nil && t.ID != 0 {
			out[t.ID] = t
		}
	}
	return out, nil
}

// LoadRecurringServices returns every synced recurring-service.
func LoadRecurringServices(db *store.Store) ([]RecurringService, error) {
	raw, err := loadRaw(db, ResRecurringServices)
	if err != nil {
		return nil, err
	}
	out := make([]RecurringService, 0, len(raw))
	for _, r := range raw {
		var rs RecurringService
		if json.Unmarshal(r, &rs) == nil && rs.ID != 0 {
			out = append(out, rs)
		}
	}
	return out, nil
}

// LoadRecurringServiceTypes returns the recurring-service-type catalog keyed
// by ID for the drift join.
func LoadRecurringServiceTypes(db *store.Store) (map[int64]RecurringServiceType, error) {
	raw, err := loadRaw(db, ResRecurringServiceTypes)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]RecurringServiceType, len(raw))
	for _, r := range raw {
		var t RecurringServiceType
		if json.Unmarshal(r, &t) == nil && t.ID != 0 {
			out[t.ID] = t
		}
	}
	return out, nil
}

// LoadRecurringServiceEvents returns every synced event.
func LoadRecurringServiceEvents(db *store.Store) ([]RecurringServiceEvent, error) {
	raw, err := loadRaw(db, ResRecurringEvents)
	if err != nil {
		return nil, err
	}
	out := make([]RecurringServiceEvent, 0, len(raw))
	for _, r := range raw {
		var e RecurringServiceEvent
		if json.Unmarshal(r, &e) == nil && e.ID != 0 {
			out = append(out, e)
		}
	}
	return out, nil
}

// LoadInvoiceTemplates returns the invoice-template catalog keyed by ID.
func LoadInvoiceTemplates(db *store.Store) (map[int64]InvoiceTemplate, error) {
	raw, err := loadRaw(db, ResInvoiceTemplates)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]InvoiceTemplate, len(raw))
	for _, r := range raw {
		var t InvoiceTemplate
		if json.Unmarshal(r, &t) == nil && t.ID != 0 {
			out[t.ID] = t
		}
	}
	return out, nil
}

// StoreEmpty reports whether the local store has no memberships and no
// membership-types — the signal that `sync` has not run yet. The audit
// commands use it so empty results from an unsynced store don't look like
// a clean audit.
func StoreEmpty(db *store.Store) (bool, error) {
	for _, rt := range []string{ResMemberships, ResMembershipTypes} {
		var n int
		if err := db.DB().QueryRow(`SELECT COUNT(*) FROM resources WHERE resource_type = ?`, rt).Scan(&n); err != nil {
			return false, fmt.Errorf("counting %s: %w", rt, err)
		}
		if n > 0 {
			return false, nil
		}
	}
	return true, nil
}

// parseTimestamp parses an RFC3339 (or date-only) timestamp string. Returns
// the zero time when value is nil or empty.
func parseTimestamp(value *string) (time.Time, bool) {
	if value == nil {
		return time.Time{}, false
	}
	return parseTimestampStr(*value)
}

func parseTimestampStr(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// round2 rounds to cents. ServiceTitan prices are currency; trailing
// precision is just noise in tables and diffs.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
