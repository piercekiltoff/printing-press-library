package store

import (
	"path/filepath"
	"testing"
	"time"
)

// openTempStore creates a fresh store in a temp dir for test isolation.
func openTempStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := openAt(filepath.Join(dir, "instacart.db"))
	if err != nil {
		t.Fatalf("openAt: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestMigrate_HistoryTablesExist(t *testing.T) {
	s := openTempStore(t)
	expected := []string{
		"orders",
		"order_items",
		"purchased_items",
		"purchased_items_fts",
		"search_history",
		"history_sync_meta",
	}
	for _, tbl := range expected {
		var name string
		err := s.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type IN ('table','virtual table') AND name = ?`,
			tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("expected table %q to exist after migrate, got err: %v", tbl, err)
		}
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	s := openTempStore(t)
	// migrate() already ran in openAt. Call it again.
	if err := s.migrate(); err != nil {
		t.Fatalf("second migrate failed: %v", err)
	}
}

func TestUpsertOrder_InsertAndUpdate(t *testing.T) {
	s := openTempStore(t)
	o := Order{
		OrderID:      "ord-1",
		RetailerSlug: "qfc",
		PlacedAt:     time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
		Status:       "DELIVERED",
		TotalCents:   4999,
		ItemCount:    12,
	}
	if err := s.UpsertOrder(o); err != nil {
		t.Fatalf("insert: %v", err)
	}
	count, err := s.CountOrders()
	if err != nil || count != 1 {
		t.Fatalf("expected 1 order, got %d err=%v", count, err)
	}

	// Same order_id with updated status should NOT create a duplicate row.
	o.Status = "REFUNDED"
	o.TotalCents = 2000
	if err := s.UpsertOrder(o); err != nil {
		t.Fatalf("update: %v", err)
	}
	count, _ = s.CountOrders()
	if count != 1 {
		t.Fatalf("expected 1 order after update, got %d", count)
	}
}

func TestUpsertPurchasedItem_FTSMirror(t *testing.T) {
	s := openTempStore(t)
	p := PurchasedItem{
		ItemID:           "items_91435-64527285",
		RetailerSlug:     "pcc-community-markets",
		Name:             "Alden's Organic Limoncello Sorbet Bars",
		Brand:            "Alden's Organic",
		Size:             "4 ct",
		Category:         "Frozen",
		LastPurchasedAt:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		FirstPurchasedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		PurchaseCount:    1,
		LastPriceCents:   699,
		LastInStock:      true,
	}
	if err := s.UpsertPurchasedItem(p, false); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// FTS search by partial name match should find the Alden's row.
	got, err := s.SearchPurchasedItems("limoncello", "pcc-community-markets", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 FTS match, got %d", len(got))
	}
	if got[0].Name != p.Name {
		t.Errorf("unexpected FTS result name: %s", got[0].Name)
	}

	// Wrong retailer returns no matches.
	got, _ = s.SearchPurchasedItems("limoncello", "qfc", 5)
	if len(got) != 0 {
		t.Fatalf("expected 0 matches at wrong retailer, got %d", len(got))
	}
}

func TestUpsertPurchasedItem_CountIncrement(t *testing.T) {
	s := openTempStore(t)
	p := PurchasedItem{
		ItemID:           "items_1388-1",
		RetailerSlug:     "safeway",
		Name:             "Whole Milk",
		LastPurchasedAt:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		FirstPurchasedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		PurchaseCount:    1,
	}
	// First insert sets count to 1 (max of seed 1 vs excluded).
	if err := s.UpsertPurchasedItem(p, false); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	// Second upsert with incrementCount=true: count should become 2.
	if err := s.UpsertPurchasedItem(p, true); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	got, err := s.ListPurchasedItems("safeway", 5)
	if err != nil || len(got) != 1 {
		t.Fatalf("list: got %d rows, err=%v", len(got), err)
	}
	if got[0].PurchaseCount != 2 {
		t.Errorf("expected purchase_count=2 after increment, got %d", got[0].PurchaseCount)
	}
}

func TestHistorySyncMeta_Roundtrip(t *testing.T) {
	s := openTempStore(t)
	m := HistorySyncMeta{
		RetailerSlug:       "qfc",
		LastSyncAt:         time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC),
		LastSyncStatus:     "ok",
		LastSyncOrderCount: 15,
		LastSyncItemCount:  87,
	}
	if err := s.UpsertHistorySyncMeta(m); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := s.GetHistorySyncMeta("qfc")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected row, got nil")
	}
	if got.LastSyncOrderCount != 15 || got.LastSyncItemCount != 87 {
		t.Errorf("roundtrip counts wrong: orders=%d items=%d", got.LastSyncOrderCount, got.LastSyncItemCount)
	}

	// Missing retailer returns (nil, nil).
	got2, err := s.GetHistorySyncMeta("nonexistent")
	if err != nil {
		t.Fatalf("missing get: %v", err)
	}
	if got2 != nil {
		t.Error("expected nil for missing retailer")
	}
}

func TestHistorySyncMeta_OptedOutSticky(t *testing.T) {
	s := openTempStore(t)
	// User opts out.
	_ = s.UpsertHistorySyncMeta(HistorySyncMeta{RetailerSlug: "safeway", OptedOut: true})
	// Subsequent update (e.g. from sync re-attempt) without OptedOut must not clear it.
	_ = s.UpsertHistorySyncMeta(HistorySyncMeta{
		RetailerSlug:   "safeway",
		LastSyncStatus: "skipped: opted out",
	})
	got, _ := s.GetHistorySyncMeta("safeway")
	if got == nil || !got.OptedOut {
		t.Errorf("opted_out must remain true; got %+v", got)
	}
}

func TestMostRecentOrderAt(t *testing.T) {
	s := openTempStore(t)
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	older := now.Add(-7 * 24 * time.Hour)

	_ = s.UpsertOrder(Order{OrderID: "a", RetailerSlug: "qfc", PlacedAt: now, Status: "ok"})
	_ = s.UpsertOrder(Order{OrderID: "b", RetailerSlug: "qfc", PlacedAt: older, Status: "ok"})
	_ = s.UpsertOrder(Order{OrderID: "c", RetailerSlug: "safeway", PlacedAt: now.Add(time.Hour), Status: "ok"})

	t1, err := s.MostRecentOrderAt("qfc")
	if err != nil {
		t.Fatalf("qfc: %v", err)
	}
	if !t1.Equal(now) {
		t.Errorf("expected qfc max at %v, got %v", now, t1)
	}

	t2, _ := s.MostRecentOrderAt("")
	if !t2.Equal(now.Add(time.Hour)) {
		t.Errorf("expected global max at %v, got %v", now.Add(time.Hour), t2)
	}

	t3, _ := s.MostRecentOrderAt("unknown")
	if !t3.IsZero() {
		t.Errorf("expected zero time for unknown retailer, got %v", t3)
	}
}
