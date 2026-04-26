// Copyright 2026 trevin-chow. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"testing"
)

func TestJoinSlicesAcrossStores_HappyPath(t *testing.T) {
	menu := json.RawMessage(`[
		{"ID": 1, "Name": "Original Cheese", "MenuID": 69, "Price": 4.25},
		{"ID": 2, "Name": "Pepperoni", "MenuID": 70, "Price": 4.75}
	]`)
	stores := json.RawMessage(`[
		{"ID": 492, "Name": "Ballard", "City": "Seattle", "Address": "3058 NW 54th",
		 "Slices": [{"ID": 69, "Name": "Original Cheese"}, {"ID": 70, "Name": "Pepperoni"}]},
		{"ID": 500, "Name": "Capitol Hill", "City": "Seattle", "Address": "426 Broadway E",
		 "Slices": [{"ID": 69, "Name": "Original Cheese"}]}
	]`)

	rows, err := joinSlicesAcrossStores(menu, stores, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// At least one row should have a populated store_name (the join worked).
	hasName := false
	for _, r := range rows {
		if r.StoreName != "" {
			hasName = true
			break
		}
	}
	if !hasName {
		t.Errorf("expected at least one row with store_name set; rows=%+v", rows)
	}
	// Prices should map from MenuID lookup.
	if rows[0].Price != 4.25 {
		t.Errorf("first row price: want 4.25, got %v", rows[0].Price)
	}
}

func TestJoinSlicesAcrossStores_StoreFilter(t *testing.T) {
	menu := json.RawMessage(`[{"ID": 1, "Name": "Cheese", "MenuID": 69, "Price": 4.25}]`)
	stores := json.RawMessage(`[
		{"ID": 492, "Name": "Ballard", "Slices": [{"ID": 69, "Name": "Cheese"}]},
		{"ID": 500, "Name": "Capitol Hill", "Slices": [{"ID": 69, "Name": "Cheese"}]}
	]`)
	rows, err := joinSlicesAcrossStores(menu, stores, 500, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].StoreID != 500 {
		t.Errorf("filter failed: got store_id=%d", rows[0].StoreID)
	}
}

func TestJoinSlicesAcrossStores_LimitCaps(t *testing.T) {
	menu := json.RawMessage(`[{"ID": 1, "Name": "Cheese", "MenuID": 69, "Price": 4.25}]`)
	stores := json.RawMessage(`[
		{"ID": 1, "Name": "A", "Slices": [{"ID": 69, "Name": "Cheese"}]},
		{"ID": 2, "Name": "B", "Slices": [{"ID": 69, "Name": "Cheese"}]},
		{"ID": 3, "Name": "C", "Slices": [{"ID": 69, "Name": "Cheese"}]}
	]`)
	rows, err := joinSlicesAcrossStores(menu, stores, 0, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("limit=2 should return 2 rows, got %d", len(rows))
	}
}

func TestJoinSlicesAcrossStores_EmptyStores(t *testing.T) {
	menu := json.RawMessage(`[{"ID": 1, "Name": "Cheese", "MenuID": 69, "Price": 4.25}]`)
	stores := json.RawMessage(`[]`)
	rows, err := joinSlicesAcrossStores(menu, stores, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for empty stores, got %d", len(rows))
	}
}
