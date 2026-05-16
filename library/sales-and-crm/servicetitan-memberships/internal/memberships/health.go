package memberships

import (
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/store"
)

// HealthThresholds are the knobs Health uses for the audits that take one.
type HealthThresholds struct {
	WithinDays   int     // renewals window (default 30)
	OverdueDays  int     // overdue-events lookback (default 365)
	ScheduleDays int     // upcoming-schedule window (default 14)
	StaleMonths  int     // stale-services threshold (default 6)
	RiskMinScore float64 // risk score floor (default 0.5)
	RevenueGroup string  // bucket dimension for revenue rollup (default "business-unit")
}

// HealthReport is the one-shot agent-priming rollup of every memberships
// audit, plus store/snapshot status. Mirrors pricebook's HealthReport shape:
// every count is a plain int so the table renderer is one line per metric.
type HealthReport struct {
	Memberships        int     `json:"memberships"`
	MembershipTypes    int     `json:"membership_types"`
	RecurringServices  int     `json:"recurring_services"`
	RecurringEvents    int     `json:"recurring_events"`
	InvoiceTemplates   int     `json:"invoice_templates"`
	ActiveMemberships  int     `json:"active_memberships"`
	Renewals           int     `json:"renewals"`
	Overdue            int     `json:"overdue_events"`
	Schedule           int     `json:"upcoming_schedule"`
	Drift              int     `json:"drift"`
	Risk               int     `json:"risk"`
	StaleServices      int     `json:"stale_services"`
	RevenueAtRisk      float64 `json:"revenue_at_risk"`
	StatusSnapshotRows int     `json:"status_snapshot_rows"`
}

// Health aggregates every audit into one compact rollup. It calls
// SnapshotMembershipStatus first so the status-snapshot count reflects
// the latest membership state captured before any diff runs.
func Health(db *store.Store, th HealthThresholds) (HealthReport, error) {
	if th.WithinDays == 0 {
		th.WithinDays = 30
	}
	if th.OverdueDays == 0 {
		th.OverdueDays = 365
	}
	if th.ScheduleDays == 0 {
		th.ScheduleDays = 14
	}
	if th.StaleMonths == 0 {
		th.StaleMonths = 6
	}
	if th.RiskMinScore == 0 {
		th.RiskMinScore = 0.5
	}
	if th.RevenueGroup == "" {
		th.RevenueGroup = "business-unit"
	}

	var h HealthReport
	members, err := LoadMemberships(db)
	if err != nil {
		return h, err
	}
	h.Memberships = len(members)
	for _, m := range members {
		if m.Active {
			h.ActiveMemberships++
		}
	}
	types, err := LoadMembershipTypes(db)
	if err != nil {
		return h, err
	}
	h.MembershipTypes = len(types)
	svcs, err := LoadRecurringServices(db)
	if err != nil {
		return h, err
	}
	h.RecurringServices = len(svcs)
	events, err := LoadRecurringServiceEvents(db)
	if err != nil {
		return h, err
	}
	h.RecurringEvents = len(events)
	templates, err := LoadInvoiceTemplates(db)
	if err != nil {
		return h, err
	}
	h.InvoiceTemplates = len(templates)

	r, err := Renewals(db, th.WithinDays, false)
	if err != nil {
		return h, err
	}
	h.Renewals = len(r)

	od, err := OverdueEvents(db, th.OverdueDays)
	if err != nil {
		return h, err
	}
	h.Overdue = len(od)

	sched, err := Schedule(db, th.ScheduleDays)
	if err != nil {
		return h, err
	}
	h.Schedule = len(sched)

	drift, err := Drift(db)
	if err != nil {
		return h, err
	}
	h.Drift = len(drift)

	risk, err := Risk(db, th.RiskMinScore)
	if err != nil {
		return h, err
	}
	h.Risk = len(risk)

	stale, err := StaleServices(db, th.StaleMonths)
	if err != nil {
		return h, err
	}
	h.StaleServices = len(stale)

	rar, err := AnnualizedRevenueAtRisk(db)
	if err != nil {
		return h, err
	}
	h.RevenueAtRisk = rar

	if _, _, err := SnapshotMembershipStatus(db); err != nil {
		return h, err
	}
	rows, err := StatusSnapshotRows(db)
	if err != nil {
		return h, err
	}
	h.StatusSnapshotRows = rows
	return h, nil
}
