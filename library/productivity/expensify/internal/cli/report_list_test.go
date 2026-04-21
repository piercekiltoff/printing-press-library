// Copyright 2026 matt-van-horn. Licensed under Apache-2.0.
//
// Tests for the --live filter builder and onyxData walk used by `report
// list`. We verify that the report list sends type=expense-report and that
// the owner-filter default lands on cfg.ExpensifyAccountID.

package cli

import (
	"encoding/json"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/config"
	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/expensifysearch"
)

// TestBuildFilterForReportList_TypeFilter: the type passed to the builder
// ("expense-report") must appear in the filter tree as an eq leaf.
func TestBuildFilterForReportList_TypeFilter(t *testing.T) {
	st := openTestStore(t)
	cfg := &config.Config{ExpensifyAccountID: 20631946}

	f, err := buildSearchFilterFromFlags(st, cfg, "", false, "expense-report", "", "")
	if err != nil {
		t.Fatalf("builder: %v", err)
	}
	if !filterHasEq(f, "type", "expense-report") {
		t.Fatalf("expected eq(type, expense-report), got %+v", f)
	}
}

// TestReportList_LiveOwnerFilterApplied verifies that the builder threads
// cfg.ExpensifyAccountID into the Query we'd send to /Search. We don't
// intercept the HTTP call — we verify the wire-shaped query the CLI would
// marshal is correct.
func TestReportList_LiveOwnerFilterApplied(t *testing.T) {
	st := openTestStore(t)
	cfg := &config.Config{ExpensifyAccountID: 20631946}

	f, err := buildSearchFilterFromFlags(st, cfg, "", false, "expense-report", "", "")
	if err != nil {
		t.Fatalf("builder: %v", err)
	}
	q := newSearchQuery("expense-report", f)
	if q.Type != "expense-report" {
		t.Fatalf("q.Type = %q, want expense-report", q.Type)
	}
	if !filterHasEq(q.Filters, "from", "20631946") {
		t.Fatalf("expected from=20631946 in query, got %+v", q.Filters)
	}

	// Confirm the query marshals to the expected wire shape.
	b, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("marshal query: %v", err)
	}
	s := string(b)
	// The filter tree should contain both eq leaves.
	for _, want := range []string{`"type":"expense-report"`, `"operator":"eq"`, `"20631946"`} {
		if !containsSubstring(s, want) {
			t.Fatalf("marshaled query missing %q: %s", want, s)
		}
	}
}

// TestReportList_LiveUpsertsReports: canned response with 2 reports should
// upsert 2 rows into the store.
func TestReportList_LiveUpsertsReports(t *testing.T) {
	st := openTestStore(t)

	data := map[string]any{
		"report_100": map[string]any{
			"reportID":   "100",
			"policyID":   "POL",
			"reportName": "Trip A",
			"total":      int64(10000),
		},
		"report_200": map[string]any{
			"reportID":   "200",
			"policyID":   "POL",
			"reportName": "Trip B",
			"total":      int64(20000),
		},
	}
	snapshotValue := map[string]any{"data": data}
	snapshotBytes, err := json.Marshal(snapshotValue)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp := &expensifysearch.Response{
		JSONCode: 200,
		OnyxData: []expensifysearch.OnyxEntry{
			{Key: "snapshot_zzz", OnyxMethod: "set", Value: snapshotBytes},
		},
	}
	nR, _ := ingestSearchResponse(st, resp)
	if nR != 2 {
		t.Fatalf("expected 2 reports upserted, got %d", nR)
	}
	items, err := st.ListReports(nil)
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("ListReports = %d rows, want 2", len(items))
	}
}

// containsSubstring is a tiny local helper so the test file doesn't pull in
// strings just for one call.
func containsSubstring(haystack, needle string) bool {
	return len(needle) == 0 || indexOf(haystack, needle) >= 0
}

func indexOf(s, sub string) int {
	n := len(sub)
	for i := 0; i+n <= len(s); i++ {
		if s[i:i+n] == sub {
			return i
		}
	}
	return -1
}
