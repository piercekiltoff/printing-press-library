// Rate-limit state persistence for archive.today submit throttling.
//
// Archive.today imposes a per-IP quota on /submit/. When you hit it, the
// server returns HTTP 429 and sets a qki= cookie with Max-Age=3600. The
// cooldown is real and the CLI used to hammer the wall repeatedly within that
// hour, making things worse.
//
// This file remembers the cooldown across CLI invocations. Before any submit,
// we check the state file. If we are in a known cooldown window, the call
// returns a CooldownError immediately without making an HTTP request.

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Default cooldown when the 429 response lacks a usable Max-Age hint.
// Archive.today's qki cookie usually has Max-Age=3600 so this fallback is
// rarely used in practice.
const defaultCooldownDuration = 1 * time.Hour

// rateLimitStateFile is the filename under stateDir() where we persist the
// cooldown window. Single JSON object, trivially self-healing on corruption.
const rateLimitStateFile = "rate-limit.json"

// rateLimitState is the on-disk schema.
type rateLimitState struct {
	Last429At     time.Time `json:"last_429_at"`
	CooldownUntil time.Time `json:"cooldown_until"`
}

// CooldownError is returned by submitCapture when we know the caller is in a
// rate-limit cooldown window. The caller can format the remaining time for
// the user.
type CooldownError struct {
	Remaining time.Duration
	Until     time.Time
}

func (e *CooldownError) Error() string {
	mins := int(e.Remaining.Minutes())
	if mins < 1 {
		mins = 1
	}
	return fmt.Sprintf("archive.today rate-limited this IP. Cooldown: retry in %d minutes (until %s).", mins, e.Until.Format("15:04"))
}

// rateLimitPath returns the absolute path to the state file, or empty string
// if stateDir() fails. An empty path signals "persistence disabled".
func rateLimitPath() string {
	dir, err := stateDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, rateLimitStateFile)
}

// readCooldown reads the persisted cooldown state. Returns nil on any error
// or when the file does not exist — the caller treats nil as "no cooldown".
// Malformed files are treated as "no cooldown" (self-heal on next write).
func readCooldown() *rateLimitState {
	path := rateLimitPath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // missing file or unreadable
	}
	var s rateLimitState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil // malformed
	}
	// Sanity check: a cooldown more than 24h in the future is stale (clock skew,
	// old state from a different archive.today policy). Treat as no cooldown.
	if time.Until(s.CooldownUntil) > 24*time.Hour {
		return nil
	}
	return &s
}

// writeCooldown persists a new cooldown window. Uses atomic rename to prevent
// partial-write corruption on crash. Silently drops errors — state persistence
// is best-effort; losing a cooldown record only means we might hit the wall
// once more.
func writeCooldown(duration time.Duration) {
	path := rateLimitPath()
	if path == "" {
		return
	}
	now := time.Now()
	state := rateLimitState{
		Last429At:     now,
		CooldownUntil: now.Add(duration),
	}
	data, err := json.MarshalIndent(&state, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// isInCooldown reports whether the CLI is currently in a persisted rate-limit
// cooldown. Returns (true, remaining) when in cooldown, (false, 0) otherwise.
func isInCooldown() (bool, time.Duration) {
	s := readCooldown()
	if s == nil {
		return false, 0
	}
	remaining := time.Until(s.CooldownUntil)
	if remaining <= 0 {
		return false, 0
	}
	return true, remaining
}

// cooldownError returns a CooldownError if we are currently in cooldown, or
// nil otherwise. Convenience for pre-submit guards.
func cooldownError() error {
	in, remaining := isInCooldown()
	if !in {
		return nil
	}
	return &CooldownError{
		Remaining: remaining,
		Until:     time.Now().Add(remaining),
	}
}

// IsCooldownError reports whether err is a CooldownError (including wrapped).
func IsCooldownError(err error) bool {
	var ce *CooldownError
	return errors.As(err, &ce)
}

// clearCooldown removes the state file. Useful for tests and for a future
// `doctor --reset-cooldown` command.
func clearCooldown() {
	path := rateLimitPath()
	if path == "" {
		return
	}
	_ = os.Remove(path)
}

// classifySubmitError wraps an error from submitCapture with the right typed
// exit code.
//
//   - CooldownError → exit code 7 (rate-limit)
//   - SubmitFailureError with BudgetExhausted → exit code 5 (api error)
//   - SubmitFailureError with all-429 attempts → exit code 7 (rate-limit)
//   - Any other SubmitFailureError → exit code 5 (api error)
//   - Other errors → exit code 5 (api error)
//
// Budget-exhausted failures are intentionally classified as apiErr, not
// rateLimitErr: we ran out of the caller's time budget, not archive.today's
// quota. Telling the user to "wait for cooldown" would be wrong — what they
// actually need is a longer --submit-timeout.
//
// Caller should `return classifySubmitError(err)` on any submit failure.
func classifySubmitError(err error) error {
	if err == nil {
		return nil
	}
	if IsCooldownError(err) {
		return rateLimitErr(err)
	}
	if sfe, ok := err.(*SubmitFailureError); ok {
		if sfe.BudgetExhausted {
			return apiErr(sfe)
		}
		if sfe.isAllRateLimited() || sfe.Cooldown != nil {
			return rateLimitErr(sfe)
		}
		return apiErr(sfe)
	}
	return apiErr(fmt.Errorf("submit failed: %w", err))
}
