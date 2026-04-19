package cli

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mvanhorn/printing-press-library/library/commerce/instacart/internal/config"
	"github.com/mvanhorn/printing-press-library/library/commerce/instacart/internal/store"
)

// newTestApp builds an AppContext pointed at a fresh per-test store.
// Session is left nil since the history-first resolver never needs it.
func newTestApp(t *testing.T) *AppContext {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir) // config.Dir() fallback

	s, err := store.OpenAt(filepath.Join(dir, "instacart.db"))
	if err != nil {
		t.Fatalf("store open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	return &AppContext{
		Ctx:   context.Background(),
		Cfg:   &config.Config{},
		Store: s,
	}
}

func seedPurchase(t *testing.T, s *store.Store, retailer, itemID, name string, ageDays int, inStock bool) {
	t.Helper()
	when := time.Now().Add(-time.Duration(ageDays) * 24 * time.Hour)
	if err := s.UpsertPurchasedItem(store.PurchasedItem{
		ItemID:           itemID,
		RetailerSlug:     retailer,
		Name:             name,
		Brand:            "Test Brand",
		Size:             "1 ct",
		Category:         "Frozen",
		LastPurchasedAt:  when,
		FirstPurchasedAt: when,
		PurchaseCount:    1,
		LastInStock:      inStock,
	}, false); err != nil {
		t.Fatalf("seed purchase: %v", err)
	}
}

func TestResolveFromHistory_ExactNameHit(t *testing.T) {
	app := newTestApp(t)
	seedPurchase(t, app.Store, "pcc-community-markets", "items_91435-64527285",
		"Alden's Organic Limoncello Sorbet Bars", 7, true)

	got := resolveFromHistory(app, "pcc-community-markets", "limoncello sorbet")
	if got == nil {
		t.Fatal("expected history hit, got nil")
	}
	if got.ItemID != "items_91435-64527285" {
		t.Errorf("unexpected item_id: %s", got.ItemID)
	}
}

func TestResolveFromHistory_WrongRetailer(t *testing.T) {
	app := newTestApp(t)
	seedPurchase(t, app.Store, "pcc-community-markets", "items_91435-64527285",
		"Alden's Organic Limoncello Sorbet Bars", 7, true)

	got := resolveFromHistory(app, "safeway", "limoncello sorbet")
	if got != nil {
		t.Errorf("expected nil for wrong retailer, got %+v", got)
	}
}

func TestResolveFromHistory_StaleFallsThrough(t *testing.T) {
	app := newTestApp(t)
	// Purchase 400 days ago exceeds the 365-day staleness threshold.
	seedPurchase(t, app.Store, "qfc", "items_404-old",
		"Old Sorbet", historyMaxAgeDays+30, true)

	got := resolveFromHistory(app, "qfc", "sorbet")
	if got != nil {
		t.Errorf("stale history entry should fall through; got %+v", got)
	}
}

func TestResolveFromHistory_OutOfStockFallsThrough(t *testing.T) {
	app := newTestApp(t)
	seedPurchase(t, app.Store, "qfc", "items_404-oos",
		"Out of stock item", 10, false)

	got := resolveFromHistory(app, "qfc", "Out of stock item")
	if got != nil {
		t.Errorf("out-of-stock history entry should fall through; got %+v", got)
	}
}

func TestResolveFromHistory_NoMatch(t *testing.T) {
	app := newTestApp(t)
	seedPurchase(t, app.Store, "qfc", "items_404-a", "Whole Milk", 1, true)

	got := resolveFromHistory(app, "qfc", "completely unrelated thing")
	if got != nil {
		t.Errorf("expected nil for non-matching query; got %+v", got)
	}
}

func TestResolveFromHistory_EmptyQuery(t *testing.T) {
	app := newTestApp(t)
	seedPurchase(t, app.Store, "qfc", "items_404-a", "Whole Milk", 1, true)

	got := resolveFromHistory(app, "qfc", "")
	if got != nil {
		t.Errorf("expected nil for empty query; got %+v", got)
	}
}

func TestWriteBackPurchasedItem_Increments(t *testing.T) {
	app := newTestApp(t)
	pick := SearchResult{
		Name:      "Whole Milk",
		ItemID:    "items_1-milk",
		ProductID: "milk",
		Retailer:  "qfc",
	}

	// Two write-backs in a row should bump purchase_count to 2.
	if err := writeBackPurchasedItem(app, "qfc", pick); err != nil {
		t.Fatalf("first write-back: %v", err)
	}
	if err := writeBackPurchasedItem(app, "qfc", pick); err != nil {
		t.Fatalf("second write-back: %v", err)
	}
	rows, _ := app.Store.ListPurchasedItems("qfc", 5)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].PurchaseCount != 2 {
		t.Errorf("expected count=2 after two writebacks; got %d", rows[0].PurchaseCount)
	}
}
