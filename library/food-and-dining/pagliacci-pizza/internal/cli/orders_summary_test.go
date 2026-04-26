// Copyright 2026 trevin-chow. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"testing"
	"time"
)

func TestComputeOrdersSummary_HappyPath(t *testing.T) {
	now := time.Now()
	recent := now.Add(-1 * time.Hour).Format(time.RFC3339)
	older := now.Add(-90 * 24 * time.Hour).Format(time.RFC3339)

	orders := []json.RawMessage{
		json.RawMessage(`{"Date":"` + recent + `","StoreID":492,"Total":25.50,"Items":[{"Name":"Original Cheese"},{"Name":"Pepperoni"}]}`),
		json.RawMessage(`{"Date":"` + recent + `","StoreID":492,"Total":15.00,"Items":[{"Name":"Original Cheese"}]}`),
		json.RawMessage(`{"Date":"` + recent + `","StoreID":500,"Total":35.00,"Items":[{"Name":"Pepperoni"}]}`),
		// This older order should be excluded by since=30d
		json.RawMessage(`{"Date":"` + older + `","StoreID":500,"Total":100.00,"Items":[{"Name":"Pepperoni"}]}`),
	}

	cutoff := now.Add(-30 * 24 * time.Hour)
	storeNames := map[int]string{492: "Ballard", 500: "Capitol Hill"}
	got := computeOrdersSummary(orders, cutoff, storeNames)

	if got.OrderCount != 3 {
		t.Errorf("order_count: want 3, got %d", got.OrderCount)
	}
	if got.TotalSpend != 75.50 {
		t.Errorf("total_spend: want 75.50, got %v", got.TotalSpend)
	}
	if len(got.TopItems) == 0 || got.TopItems[0].Name != "Original Cheese" {
		t.Errorf("top_items[0]: want Original Cheese (count 2), got %+v", got.TopItems)
	}
	if got.TopItems[0].Count != 2 {
		t.Errorf("top item count: want 2, got %d", got.TopItems[0].Count)
	}
	if len(got.ByStore) != 2 {
		t.Errorf("by_store: want 2 entries, got %d", len(got.ByStore))
	}
	// Highest-total store first: 492 has 25.50+15.00=40.50, 500 has 35.00.
	if got.ByStore[0].StoreID != 492 {
		t.Errorf("by_store[0]: want store 492 (highest total), got %d", got.ByStore[0].StoreID)
	}
	if got.ByStore[0].StoreName != "Ballard" {
		t.Errorf("by_store[0] name: want Ballard, got %q", got.ByStore[0].StoreName)
	}
	if got.AvgOrderValue < 25.16 || got.AvgOrderValue > 25.17 {
		t.Errorf("avg_order_value: ~25.166, got %v", got.AvgOrderValue)
	}
}

func TestComputeOrdersSummary_Empty(t *testing.T) {
	got := computeOrdersSummary(nil, time.Now().Add(-30*24*time.Hour), nil)
	if got.OrderCount != 0 || got.TotalSpend != 0 {
		t.Errorf("empty input should yield zero values, got %+v", got)
	}
	if len(got.TopItems) != 0 {
		t.Errorf("empty input top_items: want 0, got %d", len(got.TopItems))
	}
}

func TestParseSinceForSummary(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
		hint string
	}{
		{"30d", true, "30 days"},
		{"24h", true, "24 hours"},
		{"1y", true, "1 year"},
		{"2w", true, "2 weeks"},
		{"", true, "default 30d"},
		{"abc", false, "garbage"},
		{"-5d", false, "negative"},
		{"5x", false, "unknown unit"},
	}
	for _, c := range cases {
		_, err := parseSinceForSummary(c.in)
		if c.ok && err != nil {
			t.Errorf("parseSinceForSummary(%q): want ok, got %v (%s)", c.in, err, c.hint)
		}
		if !c.ok && err == nil {
			t.Errorf("parseSinceForSummary(%q): want error, got nil (%s)", c.in, c.hint)
		}
	}
}
