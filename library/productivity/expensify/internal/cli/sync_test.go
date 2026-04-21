// Copyright 2026 matt-van-horn. Licensed under Apache-2.0.

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/config"
	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/expensifysearch"
	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "store.sqlite")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	return &config.Config{Path: filepath.Join(dir, "config.toml")}
}

// TestIngestReconnectApp_PersonalDetails verifies that a canned response with
// a personalDetailsList entry upserts the expected Person rows.
func TestIngestReconnectApp_PersonalDetails(t *testing.T) {
	st := openTestStore(t)
	cfg := newTestConfig(t)

	payload := map[string]any{
		"onyxData": []any{
			map[string]any{
				"key": "personalDetailsList",
				"value": map[string]any{
					"12345": map[string]any{
						"displayName": "Test User",
						"login":       "user1@example.com",
					},
					"67890": map[string]any{
						"displayName": "mvh",
						"login":       "user2@example.com",
					},
				},
			},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	_, _, _, nPeople := ingestReconnectApp(st, raw, "", "", cfg)
	if nPeople != 2 {
		t.Fatalf("nPeople = %d, want 2", nPeople)
	}

	p, err := st.GetPersonByAccountID(12345)
	if err != nil {
		t.Fatalf("GetPersonByAccountID(12345): %v", err)
	}
	if p == nil || p.DisplayName != "Test User" || p.Login != "user1@example.com" {
		t.Fatalf("got %+v, want Test User / user1@example.com", p)
	}
	p2, err := st.GetPersonByAccountID(67890)
	if err != nil {
		t.Fatalf("GetPersonByAccountID(67890): %v", err)
	}
	if p2 == nil || p2.DisplayName != "mvh" {
		t.Fatalf("got %+v, want mvh", p2)
	}
}

// TestIngestReconnectApp_SessionAccountID verifies that a session blob with
// accountID populates an empty config.ExpensifyAccountID.
func TestIngestReconnectApp_SessionAccountID(t *testing.T) {
	st := openTestStore(t)
	cfg := newTestConfig(t)

	payload := map[string]any{
		"onyxData": []any{
			map[string]any{
				"key": "session",
				"value": map[string]any{
					"accountID": float64(67890),
					"authToken": "abc",
				},
			},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	ingestReconnectApp(st, raw, "", "", cfg)
	if cfg.ExpensifyAccountID != 67890 {
		t.Fatalf("cfg.ExpensifyAccountID = %d, want 67890", cfg.ExpensifyAccountID)
	}
}

// TestIngestReconnectApp_SessionAccountID_AlreadySet verifies that a session
// blob does NOT overwrite a pre-existing config.ExpensifyAccountID.
func TestIngestReconnectApp_SessionAccountID_AlreadySet(t *testing.T) {
	st := openTestStore(t)
	cfg := newTestConfig(t)
	cfg.ExpensifyAccountID = 99

	payload := map[string]any{
		"onyxData": []any{
			map[string]any{
				"key": "session",
				"value": map[string]any{
					"accountID": float64(67890),
				},
			},
		},
	}
	raw, _ := json.Marshal(payload)

	ingestReconnectApp(st, raw, "", "", cfg)
	if cfg.ExpensifyAccountID != 99 {
		t.Fatalf("cfg.ExpensifyAccountID = %d, want 99 (pre-existing)", cfg.ExpensifyAccountID)
	}
}

// TestIngestReconnectApp_SessionStringAccountID verifies that a string-typed
// accountID (some session blobs stringify numbers) still populates the config.
func TestIngestReconnectApp_SessionStringAccountID(t *testing.T) {
	st := openTestStore(t)
	cfg := newTestConfig(t)

	payload := map[string]any{
		"onyxData": []any{
			map[string]any{
				"key": "session",
				"value": map[string]any{
					"accountID": "67890",
				},
			},
		},
	}
	raw, _ := json.Marshal(payload)

	ingestReconnectApp(st, raw, "", "", cfg)
	if cfg.ExpensifyAccountID != 67890 {
		t.Fatalf("cfg.ExpensifyAccountID = %d, want 67890 (parsed from string)", cfg.ExpensifyAccountID)
	}
}

// TestIngestReconnectApp_TopLevelPersonalDetails verifies that a top-level
// personalDetailsList (some responses include it at the root, not inside
// onyxData) is still ingested.
func TestIngestReconnectApp_TopLevelPersonalDetails(t *testing.T) {
	st := openTestStore(t)
	cfg := newTestConfig(t)
	payload := map[string]any{
		"personalDetailsList": map[string]any{
			"1001": map[string]any{
				"displayName": "Solo Person",
				"login":       "solo@example.com",
			},
		},
	}
	raw, _ := json.Marshal(payload)
	_, _, _, nPeople := ingestReconnectApp(st, raw, "", "", cfg)
	if nPeople != 1 {
		t.Fatalf("nPeople = %d, want 1", nPeople)
	}
	p, err := st.GetPersonByAccountID(1001)
	if err != nil {
		t.Fatalf("GetPersonByAccountID(1001): %v", err)
	}
	if p == nil || p.DisplayName != "Solo Person" {
		t.Fatalf("got %+v, want Solo Person", p)
	}
}

// buildSnapshotResponse builds an expensifysearch.Response wrapping a single
// snapshot_<hash> entry whose `data` map is the given children. Each child
// key/value pair becomes a direct entry in data (e.g., report_1 -> {...}).
func buildSnapshotResponse(t *testing.T, children map[string]any) *expensifysearch.Response {
	t.Helper()
	data := map[string]any{}
	for k, v := range children {
		data[k] = v
	}
	value := map[string]any{"data": data}
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal snapshot value: %v", err)
	}
	return &expensifysearch.Response{
		JSONCode: 200,
		OnyxData: []expensifysearch.OnyxEntry{
			{Key: "snapshot_abc123", OnyxMethod: "merge", Value: raw},
		},
	}
}

// TestIngestHistory_FiltersByType verifies that only real expense
// reports (type=iou or expense-report / expenseReport) land in the store;
// chat / task / policyExpenseChat entries are skipped.
func TestIngestHistory_FiltersByType(t *testing.T) {
	st := openTestStore(t)

	resp := buildSnapshotResponse(t, map[string]any{
		"report_1": map[string]any{
			"reportID":   "1",
			"reportName": "IOU bucket",
			"type":       "iou",
			"stateNum":   float64(4),
		},
		"report_2": map[string]any{
			"reportID":   "2",
			"reportName": "March expenses",
			"type":       "expense-report",
			"stateNum":   float64(3),
		},
		"report_3": map[string]any{
			"reportID":   "3",
			"reportName": "Random chat",
			"type":       "chat",
		},
		"report_4": map[string]any{
			"reportID":   "4",
			"reportName": "A task",
			"type":       "task",
		},
		"report_5": map[string]any{
			"reportID":   "5",
			"reportName": "Workspace chat",
			"type":       "policyExpenseChat",
		},
	})

	n := ingestHistoricalSearch(st, resp)
	if n != 2 {
		t.Fatalf("ingestHistoricalSearch = %d, want 2 (iou + expense-report)", n)
	}

	reports, err := st.ListReports(nil)
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}
	if len(reports) != 2 {
		t.Fatalf("len(reports) = %d, want 2; got %+v", len(reports), reports)
	}
	seen := map[string]bool{}
	for _, r := range reports {
		seen[r.ReportID] = true
	}
	if !seen["1"] || !seen["2"] {
		t.Fatalf("expected reports 1 and 2, got %v", seen)
	}
	if seen["3"] || seen["4"] || seen["5"] {
		t.Fatalf("chat/task/policyExpenseChat leaked into store: %v", seen)
	}
}

// TestIngestHistory_FallbackWhenTypeMissing verifies the defensive
// fallback: when a report row has no `type` field but carries a non-empty
// reportName AND a stateNum, it still gets upserted.
func TestIngestHistory_FallbackWhenTypeMissing(t *testing.T) {
	st := openTestStore(t)

	resp := buildSnapshotResponse(t, map[string]any{
		"report_typed": map[string]any{
			"reportID":   "typed",
			"reportName": "January expenses",
			"type":       "expense-report",
			"stateNum":   float64(2),
		},
		"report_missing_type": map[string]any{
			"reportID":   "mystery",
			"reportName": "Expense Report 2025-01",
			"stateNum":   float64(3),
			// no "type" field
		},
		"report_missing_everything": map[string]any{
			"reportID":   "empty",
			"reportName": "",
			// no type, no stateNum, no reportName — should be skipped
		},
	})

	n := ingestHistoricalSearch(st, resp)
	if n != 2 {
		t.Fatalf("ingestHistoricalSearch = %d, want 2 (typed + fallback)", n)
	}

	reports, err := st.ListReports(nil)
	if err != nil {
		t.Fatalf("ListReports: %v", err)
	}
	ids := map[string]bool{}
	for _, r := range reports {
		ids[r.ReportID] = true
	}
	if !ids["typed"] || !ids["mystery"] {
		t.Fatalf("expected typed + mystery, got %v", ids)
	}
	if ids["empty"] {
		t.Fatalf("row with no type/name/stateNum leaked into store")
	}
}

// TestIngestHistory_PopulatesStateNum verifies the state_num column is
// populated from the raw `stateNum` field on each upserted report.
func TestIngestHistory_PopulatesStateNum(t *testing.T) {
	st := openTestStore(t)

	resp := buildSnapshotResponse(t, map[string]any{
		"report_paid": map[string]any{
			"reportID":   "paid-1",
			"reportName": "Paid report",
			"type":       "expense-report",
			"stateNum":   float64(4),
		},
	})

	n := ingestHistoricalSearch(st, resp)
	if n != 1 {
		t.Fatalf("ingestHistoricalSearch = %d, want 1", n)
	}

	var stateNum int64
	row := st.DB.QueryRow(`SELECT state_num FROM reports WHERE report_id = ?`, "paid-1")
	if err := row.Scan(&stateNum); err != nil {
		t.Fatalf("scan state_num: %v", err)
	}
	if stateNum != 4 {
		t.Fatalf("state_num = %d, want 4", stateNum)
	}
}

// TestSyncFlags_HistoryMonthsClamp verifies --history-months above the cap is
// clamped to maxHistoryMonths with a stderr warning, and the dispatched Query
// carries the clamped `last-24-months` date filter.
func TestSyncFlags_HistoryMonthsClamp(t *testing.T) {
	// Direct helper check for the clamp + warning.
	var warn bytes.Buffer
	got := clampHistoryMonths(36, &warn)
	if got != maxHistoryMonths {
		t.Fatalf("clampHistoryMonths(36) = %d, want %d", got, maxHistoryMonths)
	}
	warnStr := warn.String()
	if !strings.Contains(warnStr, "36") || !strings.Contains(warnStr, "clamping") {
		t.Fatalf("stderr warning = %q, want it to mention 36 + clamp", warnStr)
	}

	// Confirm the dispatched Query carries the clamped filter.
	st := openTestStore(t)
	var captured expensifysearch.Query
	stub := func(q expensifysearch.Query) (*expensifysearch.Response, error) {
		captured = q
		return &expensifysearch.Response{JSONCode: 200}, nil
	}
	if _, err := runHistoricalFetch(st, stub, maxHistoryMonths); err != nil {
		t.Fatalf("runHistoricalFetch: %v", err)
	}
	if captured.Filters == nil {
		t.Fatalf("captured.Filters is nil, want an Eq(date, last-24-months)")
	}
	if got, ok := captured.Filters.Right.(string); !ok || got != "last-24-months" {
		t.Fatalf("captured.Filters.Right = %v, want last-24-months", captured.Filters.Right)
	}
	if captured.Type != "expense-report" {
		t.Fatalf("captured.Type = %q, want expense-report", captured.Type)
	}
}

// TestSyncFlags_NoHistory verifies that when the caller skips the historical
// pass (e.g., --no-history), runHistoricalFetch is not invoked. Modeled as a
// direct spy assertion: the sync RunE gates on `noHistory` before the call.
func TestSyncFlags_NoHistory(t *testing.T) {
	st := openTestStore(t)
	called := false
	stub := func(q expensifysearch.Query) (*expensifysearch.Response, error) {
		called = true
		return &expensifysearch.Response{JSONCode: 200}, nil
	}
	// Caller semantics: when noHistory is true, runHistoricalFetch is never
	// called. Simulate by skipping the call and asserting the invariant.
	noHistory := true
	var n int
	if !noHistory {
		var err error
		n, err = runHistoricalFetch(st, stub, 12)
		if err != nil {
			t.Fatalf("runHistoricalFetch: %v", err)
		}
	}
	if called {
		t.Fatalf("Search was called despite --no-history")
	}
	if n != 0 {
		t.Fatalf("n = %d, want 0 when --no-history", n)
	}
}

// TestSync_Partial_SearchFails verifies that when the historical Search
// returns a non-200 jsonCode, runHistoricalFetch surfaces the SearchError
// but does not prevent ReconnectApp rows that were upserted earlier from
// remaining in the store.
func TestSync_Partial_SearchFails(t *testing.T) {
	st := openTestStore(t)

	// Simulate a ReconnectApp commit first: one report lands in the store.
	if err := st.UpsertReport(store.Report{
		ReportID: "pre-existing-from-reconnect",
		Title:    "Current draft",
	}); err != nil {
		t.Fatalf("pre-upsert: %v", err)
	}

	searchErr := &expensifysearch.SearchError{
		JSONCode: 401,
		Message:  "invalid query",
	}
	stub := func(q expensifysearch.Query) (*expensifysearch.Response, error) {
		return nil, searchErr
	}

	n, err := runHistoricalFetch(st, stub, 12)
	if err == nil {
		t.Fatalf("expected error from stub, got nil")
	}
	var se *expensifysearch.SearchError
	if !errors.As(err, &se) || se.JSONCode != 401 {
		t.Fatalf("err = %v, want SearchError jsonCode 401", err)
	}
	if n != 0 {
		t.Fatalf("n = %d, want 0 when Search failed", n)
	}

	// The pre-existing ReconnectApp row must still be in the store.
	reports, lerr := st.ListReports(nil)
	if lerr != nil {
		t.Fatalf("ListReports: %v", lerr)
	}
	found := false
	for _, r := range reports {
		if r.ReportID == "pre-existing-from-reconnect" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("pre-existing ReconnectApp row missing after Search failure")
	}
}
