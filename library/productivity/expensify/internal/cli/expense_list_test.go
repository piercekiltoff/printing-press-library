// Copyright 2026 matt-van-horn. Licensed under Apache-2.0.
//
// Tests for the --live mode filter builder and onyxData->upsert walk used
// by `expense list`. We don't hit the real API here — we unit-test the
// filter tree and feed canned *expensifysearch.Response values through the
// ingest helper to verify the store side effects.

package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/config"
	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/expensifysearch"
	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/store"
)

// filterHasEq recursively searches the filter tree for an eq(field, value)
// leaf. Returns true when found; false otherwise. Used to assert the
// presence (or absence) of specific leaves without pinning to a particular
// tree shape.
func filterHasEq(f *expensifysearch.Filter, field, value string) bool {
	if f == nil {
		return false
	}
	if f.Operator == "eq" {
		lf, _ := f.Left.(string)
		rv := ""
		switch v := f.Right.(type) {
		case string:
			rv = v
		}
		if lf == field && rv == value {
			return true
		}
	}
	if l, ok := f.Left.(*expensifysearch.Filter); ok {
		if filterHasEq(l, field, value) {
			return true
		}
	}
	if r, ok := f.Right.(*expensifysearch.Filter); ok {
		if filterHasEq(r, field, value) {
			return true
		}
	}
	return false
}

// filterHasField returns true when the tree has any eq leaf with the named
// field regardless of value.
func filterHasField(f *expensifysearch.Filter, field string) bool {
	if f == nil {
		return false
	}
	if f.Operator == "eq" {
		lf, _ := f.Left.(string)
		if lf == field {
			return true
		}
	}
	if l, ok := f.Left.(*expensifysearch.Filter); ok {
		if filterHasField(l, field) {
			return true
		}
	}
	if r, ok := f.Right.(*expensifysearch.Filter); ok {
		if filterHasField(r, field) {
			return true
		}
	}
	return false
}

// TestBuildFilterForExpenseList_OwnerDefault: cfg.ExpensifyAccountID=42 and
// neither --owner nor --all-visible is set, so the builder should default to
// filtering on the session user (eq "from" 42).
func TestBuildFilterForExpenseList_OwnerDefault(t *testing.T) {
	st := openTestStore(t)
	cfg := &config.Config{ExpensifyAccountID: 42}

	f, err := buildSearchFilterFromFlags(st, cfg, "", false, "expense", "", "")
	if err != nil {
		t.Fatalf("builder: %v", err)
	}
	if !filterHasEq(f, "from", "42") {
		t.Fatalf("expected eq(from, 42) in filter, got %+v", f)
	}
	if !filterHasEq(f, "type", "expense") {
		t.Fatalf("expected eq(type, expense) in filter, got %+v", f)
	}
}

// TestBuildFilterForExpenseList_OwnerByEmail: when --owner <email> is set
// and a matching person exists, the builder resolves to that accountID.
func TestBuildFilterForExpenseList_OwnerByEmail(t *testing.T) {
	st := openTestStore(t)
	if err := st.UpsertPerson(store.Person{
		AccountID:   20647491,
		DisplayName: "Myk Melez",
		Login:       "myk@example.com",
	}); err != nil {
		t.Fatalf("UpsertPerson: %v", err)
	}
	cfg := &config.Config{ExpensifyAccountID: 42}

	f, err := buildSearchFilterFromFlags(st, cfg, "myk@example.com", false, "expense", "", "")
	if err != nil {
		t.Fatalf("builder: %v", err)
	}
	if !filterHasEq(f, "from", "20647491") {
		t.Fatalf("expected eq(from, 20647491), got %+v", f)
	}
	// cfg.ExpensifyAccountID must NOT leak in as a second from-filter
	if filterHasEq(f, "from", "42") {
		t.Fatalf("cfg.ExpensifyAccountID leaked into filter despite --owner")
	}
}

// TestBuildFilterForExpenseList_OwnerEmailNotFound: unknown owner email
// returns an error that mentions sync.
func TestBuildFilterForExpenseList_OwnerEmailNotFound(t *testing.T) {
	st := openTestStore(t)
	cfg := &config.Config{ExpensifyAccountID: 42}

	_, err := buildSearchFilterFromFlags(st, cfg, "ghost@example.com", false, "expense", "", "")
	if err == nil {
		t.Fatalf("expected error for unknown owner, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "sync") {
		t.Fatalf("error %q does not mention sync", err.Error())
	}
}

// TestBuildFilterForExpenseList_AllVisible: --all-visible disables the
// owner filter entirely.
func TestBuildFilterForExpenseList_AllVisible(t *testing.T) {
	st := openTestStore(t)
	cfg := &config.Config{ExpensifyAccountID: 42}

	f, err := buildSearchFilterFromFlags(st, cfg, "", true, "expense", "", "")
	if err != nil {
		t.Fatalf("builder: %v", err)
	}
	if filterHasField(f, "from") {
		t.Fatalf("--all-visible must not include a from-filter, got %+v", f)
	}
	if !filterHasEq(f, "type", "expense") {
		t.Fatalf("expected eq(type, expense), got %+v", f)
	}
}

// TestBuildFilterForExpenseList_AccountIDUnset: cfg.ExpensifyAccountID=0,
// no --owner, no --all-visible — must error and mention sync.
func TestBuildFilterForExpenseList_AccountIDUnset(t *testing.T) {
	st := openTestStore(t)
	cfg := &config.Config{}

	_, err := buildSearchFilterFromFlags(st, cfg, "", false, "expense", "", "")
	if err == nil {
		t.Fatalf("expected error when accountID unset, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "sync") {
		t.Fatalf("error %q does not mention sync", err.Error())
	}
}

// TestBuildFilterForExpenseList_PolicyAndStatus: both --policy-id and
// --status compose into the filter tree.
func TestBuildFilterForExpenseList_PolicyAndStatus(t *testing.T) {
	st := openTestStore(t)
	cfg := &config.Config{ExpensifyAccountID: 42}

	f, err := buildSearchFilterFromFlags(st, cfg, "", false, "expense", "ABCDEF", "submitted")
	if err != nil {
		t.Fatalf("builder: %v", err)
	}
	if !filterHasEq(f, "policyID", "ABCDEF") {
		t.Fatalf("expected eq(policyID, ABCDEF), got %+v", f)
	}
	if !filterHasEq(f, "status", "submitted") {
		t.Fatalf("expected eq(status, submitted), got %+v", f)
	}
}

// TestExpenseList_LiveUpsertsIntoStore: given a canned /Search response
// with 3 transactions and 1 report in a snapshot entry, ingestSearchResponse
// upserts all rows and a subsequent ListExpenses returns them.
func TestExpenseList_LiveUpsertsIntoStore(t *testing.T) {
	st := openTestStore(t)

	// Shape: { onyxData: [ { key: snapshot_xxx, value: { data: { report_1: {...}, transactions_t1: {...}, transactions_t2: {...}, transactions_t3: {...} } } } ] }
	data := map[string]any{
		"report_1": map[string]any{
			"reportID":   "1",
			"policyID":   "POL",
			"reportName": "Test Report",
			"total":      int64(12345),
		},
		"transactions_t1": map[string]any{
			"transactionID": "t1",
			"reportID":      "1",
			"merchant":      "Coffee A",
			"amount":        int64(500),
			"date":          "2026-04-01",
			"policyID":      "POL",
		},
		"transactions_t2": map[string]any{
			"transactionID": "t2",
			"reportID":      "1",
			"merchant":      "Coffee B",
			"amount":        int64(600),
			"date":          "2026-04-02",
			"policyID":      "POL",
		},
		"transactions_t3": map[string]any{
			"transactionID": "t3",
			"reportID":      "1",
			"merchant":      "Coffee C",
			"amount":        int64(700),
			"date":          "2026-04-03",
			"policyID":      "POL",
		},
	}
	snapshotValue := map[string]any{"data": data}
	snapshotBytes, err := json.Marshal(snapshotValue)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	resp := &expensifysearch.Response{
		JSONCode: 200,
		OnyxData: []expensifysearch.OnyxEntry{
			{Key: "snapshot_abc", OnyxMethod: "set", Value: snapshotBytes},
		},
	}

	nR, nE := ingestSearchResponse(st, resp)
	if nE != 3 {
		t.Fatalf("expected 3 expenses upserted, got %d", nE)
	}
	if nR != 1 {
		t.Fatalf("expected 1 report upserted, got %d", nR)
	}

	items, err := st.ListExpenses(nil)
	if err != nil {
		t.Fatalf("ListExpenses: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("ListExpenses returned %d rows, want 3", len(items))
	}

	reports, err := st.ListReports(nil)
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("ListReports returned %d rows, want 1", len(reports))
	}
}
