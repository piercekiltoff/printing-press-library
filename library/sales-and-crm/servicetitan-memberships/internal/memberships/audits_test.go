package memberships

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/store"
)

// newTestStore opens an isolated SQLite database for a single test. The
// store is created fresh in a t.TempDir-scoped path so each test owns its
// own file and there is no cross-test contamination via the migration
// lock or the shared connection pool.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := store.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_ = os.RemoveAll(dir)
	})
	return db
}

// upsertJSON writes one JSON object as a row under the given resource type.
// Bypasses Upsert's id-extraction so tests don't need to spell out the exact
// id field — we pass the id straight through.
func upsertJSON(t *testing.T, db *store.Store, resourceType string, obj any) {
	t.Helper()
	raw, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marshal %s: %v", resourceType, err)
	}
	if err := db.Upsert(resourceType, fmt.Sprintf("%v", obj.(map[string]any)["id"]), raw); err != nil {
		t.Fatalf("upsert %s: %v", resourceType, err)
	}
}

func TestRenewals(t *testing.T) {
	t.Run("empty store returns no rows", func(t *testing.T) {
		db := newTestStore(t)
		rows, err := Renewals(db, 30, false)
		if err != nil {
			t.Fatalf("Renewals: %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("want 0 rows, got %d", len(rows))
		}
	})

	t.Run("active membership within window matches", func(t *testing.T) {
		db := newTestStore(t)
		soon := time.Now().UTC().AddDate(0, 0, 10).Format(time.RFC3339)
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 1, "customerId": 7, "membershipTypeId": 3,
			"active": true, "status": "Active", "to": soon,
			"businessUnitId": 1,
		})
		rows, err := Renewals(db, 30, false)
		if err != nil {
			t.Fatalf("Renewals: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("want 1 row, got %d", len(rows))
		}
		if rows[0].ID != 1 || rows[0].CustomerID != 7 {
			t.Fatalf("wrong row: %+v", rows[0])
		}
		if rows[0].DaysUntil < 9 || rows[0].DaysUntil > 11 {
			t.Fatalf("DaysUntil should be ~10, got %d", rows[0].DaysUntil)
		}
	})

	t.Run("inactive membership skipped unless all=true", func(t *testing.T) {
		db := newTestStore(t)
		soon := time.Now().UTC().AddDate(0, 0, 10).Format(time.RFC3339)
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 2, "customerId": 8, "membershipTypeId": 3,
			"active": false, "status": "Cancelled", "to": soon,
			"businessUnitId": 1,
		})
		rows, err := Renewals(db, 30, false)
		if err != nil {
			t.Fatalf("Renewals: %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("active filter should drop inactive, got %d rows", len(rows))
		}
		rowsAll, err := Renewals(db, 30, true)
		if err != nil {
			t.Fatalf("Renewals all=true: %v", err)
		}
		if len(rowsAll) != 1 {
			t.Fatalf("all=true should include inactive, got %d rows", len(rowsAll))
		}
	})

	t.Run("to-date outside window does not match", func(t *testing.T) {
		db := newTestStore(t)
		far := time.Now().UTC().AddDate(0, 0, 120).Format(time.RFC3339)
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 3, "customerId": 9, "membershipTypeId": 3,
			"active": true, "status": "Active", "to": far,
			"businessUnitId": 1,
		})
		rows, err := Renewals(db, 30, false)
		if err != nil {
			t.Fatalf("Renewals: %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("out-of-window row should not match, got %d", len(rows))
		}
	})
}

func TestDrift(t *testing.T) {
	t.Run("template-extra and template-missing", func(t *testing.T) {
		db := newTestStore(t)
		// Membership-type 100 expects recurring-service-type 11.
		upsertJSON(t, db, ResMembershipTypes, map[string]any{
			"id": 100, "name": "Gold", "active": true,
			"recurringServices": []map[string]any{
				{"membershipTypeId": 100, "recurringServiceTypeId": 11, "allocation": 1.0},
			},
		})
		// Active membership 1 has the template service (no drift).
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 1, "customerId": 1, "membershipTypeId": 100,
			"active": true, "status": "Active", "businessUnitId": 1,
		})
		upsertJSON(t, db, ResRecurringServices, map[string]any{
			"id": 1001, "membershipId": 1, "recurringServiceTypeId": 11, "active": true,
		})
		// Active membership 2 has an unexpected extra service (type 22).
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 2, "customerId": 2, "membershipTypeId": 100,
			"active": true, "status": "Active", "businessUnitId": 1,
		})
		upsertJSON(t, db, ResRecurringServices, map[string]any{
			"id": 1002, "membershipId": 2, "recurringServiceTypeId": 11, "active": true,
		})
		upsertJSON(t, db, ResRecurringServices, map[string]any{
			"id": 1003, "membershipId": 2, "recurringServiceTypeId": 22, "active": true,
		})
		// Active membership 3 is missing the template service.
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 3, "customerId": 3, "membershipTypeId": 100,
			"active": true, "status": "Active", "businessUnitId": 1,
		})

		rows, err := Drift(db)
		if err != nil {
			t.Fatalf("Drift: %v", err)
		}
		if len(rows) != 2 {
			t.Fatalf("want 2 drift rows (mem 2 extra + mem 3 missing), got %d: %+v", len(rows), rows)
		}
		// Sorted by membership id asc; first is 2 (extra), second is 3 (missing).
		if rows[0].MembershipID != 2 || rows[0].Reason != "template-extra" || len(rows[0].Extra) != 1 || rows[0].Extra[0] != 22 {
			t.Fatalf("row 0 wrong: %+v", rows[0])
		}
		if rows[1].MembershipID != 3 || rows[1].Reason != "template-missing" || len(rows[1].Missing) != 1 || rows[1].Missing[0] != 11 {
			t.Fatalf("row 1 wrong: %+v", rows[1])
		}
	})
}

func TestRisk(t *testing.T) {
	t.Run("multiple rule contributions accumulate", func(t *testing.T) {
		db := newTestStore(t)
		// follow-up active 0.3 + no payment-method 0.2 + bill past due 0.2 = 0.7
		yesterday := time.Now().UTC().AddDate(0, 0, -2).Format(time.RFC3339)
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 1, "customerId": 1, "membershipTypeId": 1,
			"active": true, "status": "Active", "followUpStatus": "Open",
			"paymentMethodId": nil, "nextScheduledBillDate": yesterday,
			"businessUnitId": 1,
		})
		// Clean — no risk contribution.
		future := time.Now().UTC().AddDate(0, 1, 0).Format(time.RFC3339)
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 2, "customerId": 2, "membershipTypeId": 1,
			"active": true, "status": "Active", "followUpStatus": "None",
			"paymentMethodId": 99, "nextScheduledBillDate": future,
			"businessUnitId": 1,
		})
		// Membership 2 has a recent completed event so the "no completed in 180d" rule does not fire.
		recent := time.Now().UTC().AddDate(0, 0, -10).Format(time.RFC3339)
		upsertJSON(t, db, ResRecurringEvents, map[string]any{
			"id": 5001, "membershipId": 2, "locationRecurringServiceId": 1,
			"status": "Completed", "date": recent,
		})
		rows, err := Risk(db, 0.5)
		if err != nil {
			t.Fatalf("Risk: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("want 1 risky membership, got %d: %+v", len(rows), rows)
		}
		if rows[0].MembershipID != 1 {
			t.Fatalf("wrong membership flagged: %+v", rows[0])
		}
		if rows[0].Score < 0.7 {
			t.Fatalf("expected score >= 0.7, got %v", rows[0].Score)
		}
		// Reasons must include follow-up, no payment, past-bill, stale events
		want := map[string]bool{
			"no-payment-method":       false,
			"next-bill-past-due":      false,
			"no-completed-event-180d": false,
		}
		for _, r := range rows[0].Reasons {
			for k := range want {
				if r == k {
					want[k] = true
				}
			}
		}
		for k, ok := range want {
			if !ok {
				t.Errorf("expected reason %q in %+v", k, rows[0].Reasons)
			}
		}
	})

	t.Run("filter below minScore", func(t *testing.T) {
		db := newTestStore(t)
		// Only "to-date within 30 days" + "no completed event in 180d" fires.
		// 0.1 + 0.2 = 0.3, below default 0.5.
		soon := time.Now().UTC().AddDate(0, 0, 10).Format(time.RFC3339)
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 1, "customerId": 1, "membershipTypeId": 1,
			"active": true, "status": "Active", "followUpStatus": "None",
			"paymentMethodId": 99, "to": soon,
			"businessUnitId": 1,
		})
		rows, err := Risk(db, 0.5)
		if err != nil {
			t.Fatalf("Risk: %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("score 0.3 should be filtered below 0.5; got %+v", rows)
		}
		rowsLow, err := Risk(db, 0.1)
		if err != nil {
			t.Fatalf("Risk low: %v", err)
		}
		if len(rowsLow) != 1 {
			t.Fatalf("score 0.3 should pass at minScore 0.1; got %+v", rowsLow)
		}
	})
}

func TestFind(t *testing.T) {
	t.Run("ranks by best matching field", func(t *testing.T) {
		db := newTestStore(t)
		upsertJSON(t, db, ResMembershipTypes, map[string]any{
			"id": 5, "name": "Platinum Comfort Plan", "displayName": "Platinum Comfort", "active": true,
		})
		upsertJSON(t, db, ResMembershipTypes, map[string]any{
			"id": 6, "name": "Basic Service", "displayName": "Basic", "active": true,
		})
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 1, "customerId": 7, "membershipTypeId": 5, "active": true,
			"importId": "PLAT-001", "memo": "Smith household",
			"businessUnitId": 1,
		})
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 2, "customerId": 8, "membershipTypeId": 6, "active": true,
			"importId": "BASIC-002", "memo": "Jones household",
			"businessUnitId": 1,
		})
		results, err := Find(db, "platinum comfort", 0.4, 15)
		if err != nil {
			t.Fatalf("Find: %v", err)
		}
		if len(results) == 0 {
			t.Fatalf("expected at least one hit for 'platinum comfort'")
		}
		if results[0].ID != 1 {
			t.Fatalf("expected platinum membership ranked first; got %+v", results[0])
		}
		if results[0].MatchedOn == "" {
			t.Fatalf("MatchedOn should be set; got %+v", results[0])
		}
	})

	t.Run("empty query returns nil", func(t *testing.T) {
		db := newTestStore(t)
		results, err := Find(db, "", 0.4, 15)
		if err != nil {
			t.Fatalf("Find: %v", err)
		}
		if results != nil {
			t.Fatalf("expected nil for empty query, got %+v", results)
		}
	})

	t.Run("nonsense query returns empty above minScore", func(t *testing.T) {
		db := newTestStore(t)
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 1, "customerId": 7, "membershipTypeId": 1, "active": true,
			"importId": "PLAT-001", "memo": "Smith household",
			"businessUnitId": 1,
		})
		results, err := Find(db, "xxyyzzqqqq-nomatch", 0.6, 15)
		if err != nil {
			t.Fatalf("Find: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected zero matches for nonsense; got %+v", results)
		}
	})
}

func TestSnapshotMembershipStatus(t *testing.T) {
	t.Run("idempotent on identical state", func(t *testing.T) {
		db := newTestStore(t)
		upsertJSON(t, db, ResMemberships, map[string]any{
			"id": 1, "customerId": 1, "membershipTypeId": 1, "active": true,
			"status": "Active", "followUpStatus": "None", "businessUnitId": 1,
		})
		w1, c1, err := SnapshotMembershipStatus(db)
		if err != nil {
			t.Fatalf("first snapshot: %v", err)
		}
		if w1 != 1 || c1 != 1 {
			t.Fatalf("want 1 written 1 considered on first call, got %d/%d", w1, c1)
		}
		w2, c2, err := SnapshotMembershipStatus(db)
		if err != nil {
			t.Fatalf("second snapshot: %v", err)
		}
		if w2 != 0 || c2 != 1 {
			t.Fatalf("identical state should write 0 rows, considered 1; got %d/%d", w2, c2)
		}
		rows, err := MembershipStatusHistory(db, 1)
		if err != nil {
			t.Fatalf("history: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("want 1 history row after two snapshots of identical state, got %d", len(rows))
		}
	})
}
