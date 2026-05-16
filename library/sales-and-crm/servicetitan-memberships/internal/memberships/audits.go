package memberships

import (
	"sort"
	"time"

	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/store"
)

// ----- renewals --------------------------------------------------------

// RenewalRow is one membership whose to-date falls inside the window.
type RenewalRow struct {
	ID               int64  `json:"id"`
	CustomerID       int64  `json:"customer_id"`
	MembershipTypeID int64  `json:"membership_type_id"`
	Status           string `json:"status"`
	FollowUpStatus   string `json:"follow_up_status"`
	To               string `json:"to"`
	DaysUntil        int    `json:"days_until"`
	Active           bool   `json:"active"`
}

// Renewals returns active memberships whose to-date falls within [today,
// today+withinDays]. When all is true, inactive memberships are included
// (lapse-recovery sweeps); the active filter is the default. Memberships
// whose to-date is null are skipped — without a to-date there is no
// renewal in the offing.
func Renewals(db *store.Store, withinDays int, all bool) ([]RenewalRow, error) {
	members, err := LoadMemberships(db)
	if err != nil {
		return nil, err
	}
	// PATCH: audits-day-aligned-today (Phase 4.95 native code review).
	// Truncate to start-of-day so a membership whose to-date is today
	// (renewal-day case) is not silently excluded by sub-day instants:
	// API dates parse as midnight UTC, while time.Now() is the current
	// instant, so t.Before(today) would otherwise drop today's renewals.
	today := time.Now().UTC().Truncate(24 * time.Hour)
	cutoff := today.AddDate(0, 0, withinDays)
	var out []RenewalRow
	for _, m := range members {
		if !all && !m.Active {
			continue
		}
		t, ok := parseTimestamp(m.To)
		if !ok {
			continue
		}
		if t.Before(today) || t.After(cutoff) {
			continue
		}
		out = append(out, RenewalRow{
			ID: m.ID, CustomerID: m.CustomerID, MembershipTypeID: m.MembershipTypeID,
			Status: m.Status, FollowUpStatus: m.FollowUpStatus, To: *m.To,
			DaysUntil: daysBetween(today, t), Active: m.Active,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DaysUntil != out[j].DaysUntil {
			return out[i].DaysUntil < out[j].DaysUntil
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

// ----- expiring --------------------------------------------------------

// Expiring returns every membership (active or not) whose to-date falls
// within [today, today+withinDays]. Differs from Renewals by including
// cancelled memberships — the lapse-recovery sweep.
func Expiring(db *store.Store, withinDays int) ([]RenewalRow, error) {
	return Renewals(db, withinDays, true)
}

// ----- overdue-events --------------------------------------------------

// OverdueEventRow is one recurring-service event past its scheduled date
// on a still-active membership.
type OverdueEventRow struct {
	EventID                    int64  `json:"event_id"`
	MembershipID               int64  `json:"membership_id"`
	MembershipName             string `json:"membership_name"`
	LocationRecurringServiceID int64  `json:"location_recurring_service_id"`
	ServiceName                string `json:"service_name"`
	Status                     string `json:"status"`
	Date                       string `json:"date"`
	DaysOverdue                int    `json:"days_overdue"`
}

// OverdueEvents returns events whose date is before today and whose status
// is not "Completed", for memberships that are still active. daysAgoMax
// caps how far back to look (memberships sit in the system for years;
// a 365-day default keeps the report focused on the actionable tail).
func OverdueEvents(db *store.Store, daysAgoMax int) ([]OverdueEventRow, error) {
	events, err := LoadRecurringServiceEvents(db)
	if err != nil {
		return nil, err
	}
	memberships, err := LoadMemberships(db)
	if err != nil {
		return nil, err
	}
	activeByID := make(map[int64]bool, len(memberships))
	for _, m := range memberships {
		if m.Active {
			activeByID[m.ID] = true
		}
	}
	// Day-aligned today (see Renewals for rationale).
	today := time.Now().UTC().Truncate(24 * time.Hour)
	earliest := today.AddDate(0, 0, -daysAgoMax)
	var out []OverdueEventRow
	for _, e := range events {
		if e.Status == EventStatusCompleted {
			continue
		}
		if !activeByID[e.MembershipID] {
			continue
		}
		dt, ok := parseTimestampStr(e.Date)
		if !ok || !dt.Before(today) {
			continue
		}
		if dt.Before(earliest) {
			continue
		}
		out = append(out, OverdueEventRow{
			EventID: e.ID, MembershipID: e.MembershipID, MembershipName: e.MembershipName,
			LocationRecurringServiceID: e.LocationRecurringServiceID, ServiceName: e.LocationRecurringServiceName,
			Status: e.Status, Date: e.Date, DaysOverdue: daysBetween(dt, today),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DaysOverdue != out[j].DaysOverdue {
			return out[i].DaysOverdue > out[j].DaysOverdue
		}
		return out[i].EventID < out[j].EventID
	})
	return out, nil
}

// ----- schedule --------------------------------------------------------

// ScheduleRow is one upcoming event grouped by date.
type ScheduleRow struct {
	Date           string `json:"date"`
	EventID        int64  `json:"event_id"`
	MembershipID   int64  `json:"membership_id"`
	MembershipName string `json:"membership_name"`
	ServiceName    string `json:"service_name"`
	Status         string `json:"status"`
}

// Schedule returns events whose date falls in [today, today+withinDays].
// Sorted by date ascending so the next visits lead.
func Schedule(db *store.Store, withinDays int) ([]ScheduleRow, error) {
	events, err := LoadRecurringServiceEvents(db)
	if err != nil {
		return nil, err
	}
	// Day-aligned today so events dated today are included in upcoming.
	today := time.Now().UTC().Truncate(24 * time.Hour)
	cutoff := today.AddDate(0, 0, withinDays)
	var out []ScheduleRow
	for _, e := range events {
		dt, ok := parseTimestampStr(e.Date)
		if !ok {
			continue
		}
		if dt.Before(today) || dt.After(cutoff) {
			continue
		}
		if e.Status == EventStatusCompleted {
			continue
		}
		out = append(out, ScheduleRow{
			Date: e.Date, EventID: e.ID, MembershipID: e.MembershipID,
			MembershipName: e.MembershipName, ServiceName: e.LocationRecurringServiceName,
			Status: e.Status,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return out[i].EventID < out[j].EventID
	})
	return out, nil
}

// ----- drift -----------------------------------------------------------

// DriftRow is one membership whose attached recurring-services do not match
// the membership-type template. Missing = type IDs in template but not
// attached; Extra = type IDs attached but not in template.
type DriftRow struct {
	MembershipID     int64   `json:"membership_id"`
	MembershipTypeID int64   `json:"membership_type_id"`
	Missing          []int64 `json:"missing"`
	Extra            []int64 `json:"extra"`
	Reason           string  `json:"reason"`
}

// Drift compares each active membership's attached recurring-services
// against its membership-type's recurringServices[] template. Reports
// memberships with at least one missing or extra service. Memberships
// pointing at a membership-type ID that no longer exists are flagged
// with a "missing-type" reason — useful when type IDs change between
// API versions or after a renaming.
func Drift(db *store.Store) ([]DriftRow, error) {
	memberships, err := LoadMemberships(db)
	if err != nil {
		return nil, err
	}
	types, err := LoadMembershipTypes(db)
	if err != nil {
		return nil, err
	}
	services, err := LoadRecurringServices(db)
	if err != nil {
		return nil, err
	}
	servicesByMembership := make(map[int64][]int64, len(memberships))
	for _, rs := range services {
		if !rs.Active {
			continue
		}
		servicesByMembership[rs.MembershipID] = append(servicesByMembership[rs.MembershipID], rs.RecurringServiceTypeID)
	}
	var out []DriftRow
	for _, m := range memberships {
		if !m.Active {
			continue
		}
		mt, ok := types[m.MembershipTypeID]
		if !ok {
			out = append(out, DriftRow{
				MembershipID: m.ID, MembershipTypeID: m.MembershipTypeID,
				Reason: "missing-type",
			})
			continue
		}
		templateSet := make(map[int64]struct{}, len(mt.RecurringServices))
		for _, entry := range mt.RecurringServices {
			templateSet[entry.RecurringServiceTypeID] = struct{}{}
		}
		actualSet := make(map[int64]struct{}, len(servicesByMembership[m.ID]))
		for _, sid := range servicesByMembership[m.ID] {
			actualSet[sid] = struct{}{}
		}
		var missing, extra []int64
		for tid := range templateSet {
			if _, ok := actualSet[tid]; !ok {
				missing = append(missing, tid)
			}
		}
		for aid := range actualSet {
			if _, ok := templateSet[aid]; !ok {
				extra = append(extra, aid)
			}
		}
		if len(missing) == 0 && len(extra) == 0 {
			continue
		}
		sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
		sort.Slice(extra, func(i, j int) bool { return extra[i] < extra[j] })
		reason := ""
		switch {
		case len(missing) > 0 && len(extra) > 0:
			reason = "template-extra-and-missing"
		case len(missing) > 0:
			reason = "template-missing"
		case len(extra) > 0:
			reason = "template-extra"
		}
		out = append(out, DriftRow{
			MembershipID: m.ID, MembershipTypeID: m.MembershipTypeID,
			Missing: missing, Extra: extra, Reason: reason,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].MembershipID < out[j].MembershipID })
	return out, nil
}

// ----- risk ------------------------------------------------------------

// RiskRow is one membership with a risk score and the contributing reasons.
type RiskRow struct {
	MembershipID     int64    `json:"membership_id"`
	MembershipTypeID int64    `json:"membership_type_id"`
	CustomerID       int64    `json:"customer_id"`
	Status           string   `json:"status"`
	FollowUpStatus   string   `json:"follow_up_status"`
	Score            float64  `json:"score"`
	Reasons          []string `json:"reasons"`
}

// Risk applies a rule engine over active memberships:
//
//   - followUpStatus != "None" / "" → +0.3
//   - paymentMethodId == nil       → +0.2
//   - nextScheduledBillDate < today → +0.2
//   - no completed event in 180d   → +0.2
//   - to-date within 30 days       → +0.1
//
// Rows scoring below minScore are dropped. The score is clipped to [0, 1].
// Returned sorted by score descending then membership ID asc for stability.
func Risk(db *store.Store, minScore float64) ([]RiskRow, error) {
	memberships, err := LoadMemberships(db)
	if err != nil {
		return nil, err
	}
	events, err := LoadRecurringServiceEvents(db)
	if err != nil {
		return nil, err
	}
	// Day-aligned today so the past-due and within-30 rules don't trip
	// on the sub-day distinction between time.Now() and midnight-parsed dates.
	today := time.Now().UTC().Truncate(24 * time.Hour)
	staleCutoff := today.AddDate(0, 0, -180)
	renewalCutoff := today.AddDate(0, 0, 30)
	lastCompletedByMembership := make(map[int64]time.Time, len(memberships))
	for _, e := range events {
		if e.Status != EventStatusCompleted {
			continue
		}
		dt, ok := parseTimestampStr(e.Date)
		if !ok {
			continue
		}
		if prev, ok := lastCompletedByMembership[e.MembershipID]; !ok || dt.After(prev) {
			lastCompletedByMembership[e.MembershipID] = dt
		}
	}
	var out []RiskRow
	for _, m := range memberships {
		if !m.Active {
			continue
		}
		var reasons []string
		score := 0.0
		if m.FollowUpStatus != "" && m.FollowUpStatus != "None" {
			score += 0.3
			reasons = append(reasons, "follow-up-active:"+m.FollowUpStatus)
		}
		if m.PaymentMethodID == nil {
			score += 0.2
			reasons = append(reasons, "no-payment-method")
		}
		if nbd, ok := parseTimestamp(m.NextScheduledBillDate); ok && nbd.Before(today) {
			score += 0.2
			reasons = append(reasons, "next-bill-past-due")
		}
		last, hadComplete := lastCompletedByMembership[m.ID]
		if !hadComplete || last.Before(staleCutoff) {
			score += 0.2
			reasons = append(reasons, "no-completed-event-180d")
		}
		if to, ok := parseTimestamp(m.To); ok && !to.Before(today) && to.Before(renewalCutoff) {
			score += 0.1
			reasons = append(reasons, "to-date-within-30d")
		}
		if score > 1.0 {
			score = 1.0
		}
		if score < minScore {
			continue
		}
		out = append(out, RiskRow{
			MembershipID: m.ID, MembershipTypeID: m.MembershipTypeID, CustomerID: m.CustomerID,
			Status: m.Status, FollowUpStatus: m.FollowUpStatus,
			Score: round2(score), Reasons: reasons,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].MembershipID < out[j].MembershipID
	})
	return out, nil
}

// ----- stale-services --------------------------------------------------

// StaleServiceRow is one active recurring-service with no completed event
// in the lookback window.
type StaleServiceRow struct {
	RecurringServiceID int64  `json:"recurring_service_id"`
	Name               string `json:"name"`
	MembershipID       int64  `json:"membership_id"`
	LastCompletedDate  string `json:"last_completed_date"`
	DaysSinceCompleted int    `json:"days_since_completed"`
}

// StaleServices returns active recurring-services attached to active
// memberships with no completed event in the past `months` months. A
// service that has never had a completed event is reported with an empty
// LastCompletedDate and DaysSinceCompleted = -1.
func StaleServices(db *store.Store, months int) ([]StaleServiceRow, error) {
	services, err := LoadRecurringServices(db)
	if err != nil {
		return nil, err
	}
	memberships, err := LoadMemberships(db)
	if err != nil {
		return nil, err
	}
	events, err := LoadRecurringServiceEvents(db)
	if err != nil {
		return nil, err
	}
	activeMembership := make(map[int64]bool, len(memberships))
	for _, m := range memberships {
		if m.Active {
			activeMembership[m.ID] = true
		}
	}
	lastByService := make(map[int64]time.Time, len(services))
	for _, e := range events {
		if e.Status != EventStatusCompleted {
			continue
		}
		dt, ok := parseTimestampStr(e.Date)
		if !ok {
			continue
		}
		if prev, ok := lastByService[e.LocationRecurringServiceID]; !ok || dt.After(prev) {
			lastByService[e.LocationRecurringServiceID] = dt
		}
	}
	// Day-aligned today (see Renewals for rationale).
	today := time.Now().UTC().Truncate(24 * time.Hour)
	cutoff := today.AddDate(0, -months, 0)
	var out []StaleServiceRow
	for _, rs := range services {
		if !rs.Active {
			continue
		}
		if !activeMembership[rs.MembershipID] {
			continue
		}
		last, ok := lastByService[rs.ID]
		if ok && !last.Before(cutoff) {
			continue
		}
		row := StaleServiceRow{
			RecurringServiceID: rs.ID, Name: rs.Name, MembershipID: rs.MembershipID,
		}
		if ok {
			row.LastCompletedDate = last.Format(time.RFC3339)
			row.DaysSinceCompleted = daysBetween(last, today)
		} else {
			row.DaysSinceCompleted = -1
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DaysSinceCompleted != out[j].DaysSinceCompleted {
			// -1 (never) sorts to the top, then biggest gaps first
			if out[i].DaysSinceCompleted == -1 {
				return true
			}
			if out[j].DaysSinceCompleted == -1 {
				return false
			}
			return out[i].DaysSinceCompleted > out[j].DaysSinceCompleted
		}
		return out[i].RecurringServiceID < out[j].RecurringServiceID
	})
	return out, nil
}

// daysBetween returns the integer count of days from a to b, rounded toward
// zero. Negative when a > b.
func daysBetween(a, b time.Time) int {
	d := b.Sub(a)
	return int(d / (24 * time.Hour))
}
