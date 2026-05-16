package memberships

import (
	"fmt"
	"sort"
	"time"

	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/store"
)

// RevenueRow is one bucket of the recurring-revenue rollup. The grouping
// dimension is named in GroupKey and varies by `by`:
//
//   - "month"              → YYYY-MM bucket of nextScheduledBillDate
//   - "business-unit"      → membership.businessUnitId
//   - "billing-frequency"  → membership.billingFrequency value
type RevenueRow struct {
	GroupKey        string  `json:"group_key"`
	MembershipCount int     `json:"membership_count"`
	BillingTotal    float64 `json:"billing_total"`
	SaleTotal       float64 `json:"sale_total"`
	RenewalTotal    float64 `json:"renewal_total"`
}

// Revenue rolls up active memberships joined to their membership-type's
// durationBilling entries. For each membership we resolve its bill amount
// by walking the type's durationBilling[] looking for an entry whose
// (duration, billingFrequency) matches the membership's current state;
// when one matches, billingPrice/salePrice/renewalPrice contribute to the
// bucket. Memberships with no matching durationBilling entry contribute 0
// dollars but still count toward MembershipCount so the bucket reflects
// the real population. by selects the grouping dimension.
func Revenue(db *store.Store, by string) ([]RevenueRow, error) {
	switch by {
	case "month", "business-unit", "billing-frequency":
	default:
		return nil, fmt.Errorf("invalid --by %q: must be one of month, business-unit, billing-frequency", by)
	}
	memberships, err := LoadMemberships(db)
	if err != nil {
		return nil, err
	}
	types, err := LoadMembershipTypes(db)
	if err != nil {
		return nil, err
	}
	buckets := make(map[string]*RevenueRow)
	for _, m := range memberships {
		if !m.Active {
			continue
		}
		key := groupKeyFor(m, by)
		if key == "" {
			key = "(unknown)"
		}
		b, ok := buckets[key]
		if !ok {
			b = &RevenueRow{GroupKey: key}
			buckets[key] = b
		}
		b.MembershipCount++

		mt, ok := types[m.MembershipTypeID]
		if !ok {
			continue
		}
		entry, ok := lookupDurationBilling(mt.DurationBilling, m.Duration, m.BillingFrequency)
		if !ok {
			continue
		}
		// PATCH: revenue-accumulator-rounding (Phase 4.95 native code review).
		// Sum raw and round once at the end so sub-cent precision in
		// the inputs doesn't accumulate per-row rounding error across
		// large buckets.
		b.BillingTotal += entry.BillingPrice
		b.SaleTotal += entry.SalePrice
		b.RenewalTotal += entry.RenewalPrice
	}
	out := make([]RevenueRow, 0, len(buckets))
	for _, b := range buckets {
		b.BillingTotal = round2(b.BillingTotal)
		b.SaleTotal = round2(b.SaleTotal)
		b.RenewalTotal = round2(b.RenewalTotal)
		out = append(out, *b)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].GroupKey < out[j].GroupKey })
	return out, nil
}

func groupKeyFor(m Membership, by string) string {
	switch by {
	case "month":
		t, ok := parseTimestamp(m.NextScheduledBillDate)
		if !ok {
			return ""
		}
		return t.UTC().Format("2006-01")
	case "business-unit":
		return fmt.Sprintf("%d", m.BusinessUnitID)
	case "billing-frequency":
		return m.BillingFrequency
	}
	return ""
}

// lookupDurationBilling returns the membership-type duration-billing entry
// whose (duration, billingFrequency) matches the membership's current state.
// Duration is a pointer because ServiceTitan reports it as nullable; a nil
// duration falls back to matching billingFrequency alone, which mirrors how
// ServiceTitan's own renewal engine treats unset durations.
func lookupDurationBilling(entries []MembershipTypeDurationBillingEntry, duration *int, freq string) (MembershipTypeDurationBillingEntry, bool) {
	if len(entries) == 0 {
		return MembershipTypeDurationBillingEntry{}, false
	}
	var fallback *MembershipTypeDurationBillingEntry
	for i := range entries {
		e := entries[i]
		if duration != nil && e.Duration == *duration && e.BillingFrequency == freq {
			return e, true
		}
		if e.BillingFrequency == freq && fallback == nil {
			fallback = &entries[i]
		}
	}
	if fallback != nil {
		return *fallback, true
	}
	// PATCH: fix-revenue-lookup-fallback (Greptile PR #601 P1): when no exact
	// or frequency-only match exists, return ok=false so callers can honestly
	// surface "no matching duration-billing entry" instead of silently
	// billing the membership at the wrong tier from entries[0]. The !ok
	// branch in callers (bill-preview, revenue) is the canonical
	// missing-config path; promoting a non-matching entry to "ok" hid it.
	return MembershipTypeDurationBillingEntry{}, false
}

// AnnualizedRevenueAtRisk sums the BillingTotal across every bucket and
// returns it. Used by Health to give one "at-risk dollars" headline.
func AnnualizedRevenueAtRisk(db *store.Store) (float64, error) {
	// Simplification: sum BillingTotal across active memberships, treating
	// it as the per-cycle dollar value. A full annualization would multiply
	// by billing-frequency cycles per year; this number is the "what's
	// running through the system right now" figure that the agent priming
	// header needs, not an accounting-grade ARR.
	rows, err := Revenue(db, "month")
	if err != nil {
		return 0, err
	}
	var sum float64
	for _, r := range rows {
		sum += r.BillingTotal
	}
	return round2(sum), nil
}

// _ silences the unused import in builds that drop revenue temporarily.
var _ = time.Now
