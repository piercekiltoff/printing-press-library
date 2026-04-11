package cli

import (
	"fmt"
	"testing"
	"time"
)

func TestRequestState_PendingThenReady(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	url := "https://example.com/test"
	recordPending(url)

	r := lookupRequest(url)
	if r == nil {
		t.Fatal("expected pending record")
	}
	if r.Status != requestStatusPending {
		t.Errorf("got status %q, want pending", r.Status)
	}

	recordReady(url, "https://archive.ph/20260411/https://example.com/test")
	r = lookupRequest(url)
	if r == nil || r.Status != requestStatusReady {
		t.Errorf("expected ready, got %+v", r)
	}
	if r.MementoURL == "" {
		t.Error("expected memento URL")
	}
}

func TestRequestState_PendingThenFailed(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	url := "https://example.com/fail-test"
	recordPending(url)
	recordFailed(url, fmt.Errorf("rate limited"))

	r := lookupRequest(url)
	if r == nil || r.Status != requestStatusFailed {
		t.Errorf("expected failed, got %+v", r)
	}
	if r.Error == "" {
		t.Error("expected error message")
	}
}

func TestRequestState_StalePending(t *testing.T) {
	r := &requestRecord{
		Status:      requestStatusPending,
		SubmittedAt: time.Now().Add(-10 * time.Minute),
	}
	if !isStalePending(r) {
		t.Error("expected 10-minute old pending to be stale")
	}

	r2 := &requestRecord{
		Status:      requestStatusPending,
		SubmittedAt: time.Now().Add(-1 * time.Minute),
	}
	if isStalePending(r2) {
		t.Error("expected 1-minute old pending to be fresh")
	}

	r3 := &requestRecord{
		Status: requestStatusReady,
	}
	if isStalePending(r3) {
		t.Error("ready records are never stale-pending")
	}
}

func TestRequestState_GC(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Directly write a store with an old record
	s := &requestStateStore{Records: make(map[string]requestRecord)}
	s.Records["old.example.com"] = requestRecord{
		OriginalURL: "old.example.com",
		Status:      requestStatusReady,
		SubmittedAt: time.Now().Add(-10 * 24 * time.Hour),
	}
	s.Records["new.example.com"] = requestRecord{
		OriginalURL: "new.example.com",
		Status:      requestStatusReady,
		SubmittedAt: time.Now().Add(-1 * time.Hour),
	}
	writeRequestState(s)

	// recordPending triggers GC
	recordPending("https://trigger.example.com")

	s2 := readRequestState()
	if _, ok := s2.Records["old.example.com"]; ok {
		t.Error("old record should have been GC'd")
	}
	if _, ok := s2.Records["new.example.com"]; !ok {
		t.Error("new record should have been preserved")
	}
}
