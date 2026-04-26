// Copyright 2026 trevin-chow. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"testing"
)

func TestFindLabel_HappyPath(t *testing.T) {
	items := []json.RawMessage{
		json.RawMessage(`{"ID": 1, "Label": "home", "Address": "123 Main"}`),
		json.RawMessage(`{"ID": 2, "Label": "work", "Address": "456 Office"}`),
	}
	got := findLabel(items, "home")
	if got == nil {
		t.Fatal("expected to find label home")
	}
	if got["Address"] != "123 Main" {
		t.Errorf("wrong address: %v", got["Address"])
	}
}

func TestFindLabel_CaseInsensitive(t *testing.T) {
	items := []json.RawMessage{
		json.RawMessage(`{"Label": "Home", "Address": "123"}`),
	}
	if findLabel(items, "home") == nil {
		t.Errorf("case-insensitive lookup failed")
	}
	if findLabel(items, "HOME") == nil {
		t.Errorf("case-insensitive lookup failed (uppercase query)")
	}
}

func TestFindLabel_AlternateField(t *testing.T) {
	// Some saved addresses use Name instead of Label.
	items := []json.RawMessage{
		json.RawMessage(`{"Name": "work", "Address": "456 Office"}`),
	}
	if findLabel(items, "work") == nil {
		t.Errorf("Name field fallback failed")
	}
}

func TestFindLabel_NoMatch(t *testing.T) {
	items := []json.RawMessage{
		json.RawMessage(`{"Label": "home"}`),
	}
	if findLabel(items, "missing") != nil {
		t.Errorf("expected no match for missing label")
	}
}

func TestAddressLine_Combines(t *testing.T) {
	o := map[string]any{
		"Address": "350 5th Ave",
		"City":    "Seattle",
		"State":   "WA",
		"Zip":     "98101",
	}
	got := addressLine(o)
	if got != "350 5th Ave, Seattle, WA 98101" {
		t.Errorf("unexpected address line: %q", got)
	}
}

func TestExtractSlotTimes_TopLevelArray(t *testing.T) {
	raw := json.RawMessage(`[{"Time":"5:00pm"},{"Time":"5:15pm"}]`)
	got := extractSlotTimes(raw)
	if len(got) != 2 || got[0] != "5:00pm" {
		t.Errorf("unexpected slots: %v", got)
	}
}

func TestExtractSlotTimes_ObjectWindows(t *testing.T) {
	raw := json.RawMessage(`{"Windows":[{"StartTime":"5:00pm"},{"StartTime":"5:15pm"}]}`)
	got := extractSlotTimes(raw)
	if len(got) != 2 || got[0] != "5:00pm" {
		t.Errorf("unexpected slots: %v", got)
	}
}

func TestExtractSlotTimes_Empty(t *testing.T) {
	if got := extractSlotTimes(json.RawMessage(`[]`)); len(got) != 0 {
		t.Errorf("expected empty slots, got %v", got)
	}
}
