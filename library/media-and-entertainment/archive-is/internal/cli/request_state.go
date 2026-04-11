// Request state persistence for async submits.
//
// The `request <url>` command (without --wait) fires submit in a goroutine
// and returns PENDING immediately. But if the goroutine fails (rate-limit,
// network error), the error is lost — `request check` has no way to know
// whether the submit succeeded, failed, or is still running.
//
// This file persists the submit result to disk so `request check` can report
// terminal states. The store is a single JSON file at stateDir/requests.json
// keyed by original URL.
//
// Follow-up improvement (noted in the plan): true detached-process submit so
// the worker survives the parent process exit. For now we use an in-process
// goroutine that writes state on completion. The parent blocks briefly when
// using `request --wait`, but `request` without --wait exits before the
// goroutine finishes, so the writer must survive the parent exit... which it
// won't in a pure goroutine. For this pass, accept the limitation: bare
// `request` writes the initial PENDING record and spawns the goroutine, but
// the goroutine's success/failure is only captured reliably when the user runs
// with --wait. Rare loss is preferable to the complexity of detached child
// processes right now.

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const requestStateFile = "requests.json"

// Request state constants.
const (
	requestStatusPending = "pending"
	requestStatusReady   = "ready"
	requestStatusFailed  = "failed"
)

// requestRecord is one entry in the state file, keyed by original URL.
type requestRecord struct {
	OriginalURL string    `json:"original_url"`
	Status      string    `json:"status"`
	SubmittedAt time.Time `json:"submitted_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	MementoURL  string    `json:"memento_url,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// requestStateStore holds the full collection of records. The store is
// trivially small (one record per unique URL), so we load and save the entire
// file on each operation. No in-memory cache.
type requestStateStore struct {
	Records map[string]requestRecord `json:"records"`
}

// requestStatePath returns the absolute path to the state file, or empty
// string if stateDir() fails.
func requestStatePath() string {
	dir, err := stateDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, requestStateFile)
}

// requestStateLock serializes read-modify-write cycles to prevent concurrent
// writes from corrupting the file. Per-process only; multi-process safety
// relies on atomic rename at the filesystem level.
var requestStateLock sync.Mutex

// readRequestState loads the full store. Returns an empty store on any error
// (missing file, malformed JSON, unreadable). Callers treat empty as
// "no known requests".
func readRequestState() *requestStateStore {
	path := requestStatePath()
	if path == "" {
		return &requestStateStore{Records: make(map[string]requestRecord)}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return &requestStateStore{Records: make(map[string]requestRecord)}
	}
	var s requestStateStore
	if err := json.Unmarshal(data, &s); err != nil {
		return &requestStateStore{Records: make(map[string]requestRecord)}
	}
	if s.Records == nil {
		s.Records = make(map[string]requestRecord)
	}
	return &s
}

// writeRequestState persists the full store atomically. Silently ignores
// write errors — state persistence is best-effort.
func writeRequestState(s *requestStateStore) {
	path := requestStatePath()
	if path == "" {
		return
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// recordPending writes a PENDING record for the URL. Called immediately when
// `request <url>` starts the submit goroutine.
func recordPending(origURL string) {
	requestStateLock.Lock()
	defer requestStateLock.Unlock()
	s := readRequestState()
	s.Records[origURL] = requestRecord{
		OriginalURL: origURL,
		Status:      requestStatusPending,
		SubmittedAt: time.Now(),
	}
	gcOldRecordsLocked(s)
	writeRequestState(s)
}

// recordReady updates the record to READY with the memento URL. Called when
// the submit goroutine succeeds.
func recordReady(origURL, mementoURL string) {
	requestStateLock.Lock()
	defer requestStateLock.Unlock()
	s := readRequestState()
	s.Records[origURL] = requestRecord{
		OriginalURL: origURL,
		Status:      requestStatusReady,
		SubmittedAt: recordSubmittedAt(s, origURL),
		CompletedAt: time.Now(),
		MementoURL:  mementoURL,
	}
	writeRequestState(s)
}

// recordFailed updates the record to FAILED with the error message. Called
// when the submit goroutine errors.
func recordFailed(origURL string, err error) {
	requestStateLock.Lock()
	defer requestStateLock.Unlock()
	s := readRequestState()
	msg := "unknown error"
	if err != nil {
		msg = err.Error()
	}
	s.Records[origURL] = requestRecord{
		OriginalURL: origURL,
		Status:      requestStatusFailed,
		SubmittedAt: recordSubmittedAt(s, origURL),
		CompletedAt: time.Now(),
		Error:       msg,
	}
	writeRequestState(s)
}

// lookupRequest returns the record for the URL, or nil if missing.
func lookupRequest(origURL string) *requestRecord {
	requestStateLock.Lock()
	defer requestStateLock.Unlock()
	s := readRequestState()
	if r, ok := s.Records[origURL]; ok {
		return &r
	}
	return nil
}

// recordSubmittedAt returns the original submission time for a record if one
// exists, or time.Now() as a fallback. Used to preserve the timestamp across
// status updates.
func recordSubmittedAt(s *requestStateStore, origURL string) time.Time {
	if r, ok := s.Records[origURL]; ok && !r.SubmittedAt.IsZero() {
		return r.SubmittedAt
	}
	return time.Now()
}

// gcOldRecordsLocked removes records older than 7 days. Caller must hold
// requestStateLock.
func gcOldRecordsLocked(s *requestStateStore) {
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	for url, r := range s.Records {
		if !r.SubmittedAt.IsZero() && r.SubmittedAt.Before(cutoff) {
			delete(s.Records, url)
		}
	}
}

// isStalePending reports whether a record is in PENDING status but has been
// there for too long (5+ minutes). Used by `request check` to detect worker
// processes that died without writing their terminal state.
func isStalePending(r *requestRecord) bool {
	if r == nil || r.Status != requestStatusPending {
		return false
	}
	return time.Since(r.SubmittedAt) > 5*time.Minute
}

// ErrRequestStateUnavailable is returned when the state file is unreadable or
// unwritable. Callers treat this as "proceed without persistence".
var ErrRequestStateUnavailable = errors.New("request state unavailable")

// DebugDumpRequests returns a human-readable dump of all known requests.
// Useful for a future `request list` subcommand or debugging.
func DebugDumpRequests() string {
	s := readRequestState()
	if len(s.Records) == 0 {
		return "no requests tracked"
	}
	out := ""
	for _, r := range s.Records {
		out += fmt.Sprintf("%s  %s  %s\n", r.SubmittedAt.Format("2006-01-02 15:04"), r.Status, r.OriginalURL)
	}
	return out
}
