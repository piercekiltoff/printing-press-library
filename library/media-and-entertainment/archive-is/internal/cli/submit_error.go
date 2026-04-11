// Structured error reporting for archive.today submit failures.
//
// The old error message `submit failed: https://archive.vn: rate limited` was
// misleading — it implied only the last mirror failed, when in practice all
// six were tried. This file defines a structured error type that collects
// per-mirror outcomes and renders them as a multi-line report with concrete
// remediation guidance.
//
// Unit 3 of the polish plan.

package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// MirrorResult captures the outcome of a single mirror's submit attempt.
type MirrorResult struct {
	URL      string `json:"url"`
	HTTPCode int    `json:"http_code,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
	Err      error  `json:"-"` // don't serialize raw error; see ErrMessage
}

// MarshalJSON renders the error as a string field so --json output stays
// parseable. The Err field holds the full Go error which may not round-trip.
func (r MirrorResult) MarshalJSON() ([]byte, error) {
	type alias MirrorResult
	wrapper := struct {
		alias
		ErrMessage string `json:"error,omitempty"`
	}{
		alias: alias(r),
	}
	if r.Err != nil {
		wrapper.ErrMessage = r.Err.Error()
	}
	return json.Marshal(wrapper)
}

// SubmitFailureError is returned when archive.today submit fails across all
// mirrors. It carries per-mirror results and, when rate-limited, the cooldown
// window so callers can render actionable guidance.
//
// BudgetExhausted is true when the per-invocation --submit-timeout budget
// expired mid-submit. This is a different failure mode from rate-limit: the
// mirrors did not refuse us, we just ran out of time. Budget-exhausted failures
// classify as apiErr (exit 5) rather than rateLimitErr (exit 7) so users aren't
// told to wait for a non-existent cooldown.
type SubmitFailureError struct {
	Attempts        []MirrorResult
	Cooldown        *time.Duration
	CooldownUntil   *time.Time
	BudgetExhausted bool
	Budget          time.Duration
}

// Error renders the failure as a multi-line string. The first line is a
// one-sentence summary; subsequent lines list each mirror's outcome; a final
// paragraph shows the cooldown window and remediation options.
//
// Output shape (human-readable):
//
//	submit failed: archive.today rate-limited this IP across all mirrors
//	  archive.ph:  HTTP 429  (attempts: 4)
//	  archive.md:  HTTP 429  (attempts: 4)
//	  ...
//
//	Cooldown: retry in 58 minutes (until 15:04).
//	Alternative: submit manually at https://archive.ph/ in your browser.
func (e *SubmitFailureError) Error() string {
	var b strings.Builder

	// Summary line
	switch {
	case e.BudgetExhausted:
		if e.Budget > 0 {
			b.WriteString(fmt.Sprintf("submit failed: budget exhausted after %s (archive.today did not respond in time)", formatBudget(e.Budget)))
		} else {
			b.WriteString("submit failed: budget exhausted (archive.today did not respond in time)")
		}
	case e.isAllRateLimited():
		b.WriteString("submit failed: archive.today rate-limited this IP across all mirrors")
	case len(e.Attempts) > 0:
		b.WriteString(fmt.Sprintf("submit failed: %d mirrors attempted, none succeeded", len(e.Attempts)))
	default:
		b.WriteString("submit failed: no mirrors attempted")
	}
	b.WriteString("\n")

	// Per-mirror lines
	for _, a := range e.Attempts {
		host := archiveHostFromURL(a.URL)
		b.WriteString(fmt.Sprintf("  %-14s ", host+":"))
		if a.HTTPCode > 0 {
			b.WriteString(fmt.Sprintf("HTTP %d", a.HTTPCode))
		} else if a.Err != nil {
			msg := a.Err.Error()
			if len(msg) > 60 {
				msg = msg[:57] + "..."
			}
			b.WriteString(msg)
		}
		if a.Attempts > 1 {
			b.WriteString(fmt.Sprintf("  (attempts: %d)", a.Attempts))
		}
		b.WriteString("\n")
	}

	// Remediation section
	if e.BudgetExhausted {
		b.WriteString("\n")
		b.WriteString("Retry with a longer budget:  archive-is-pp-cli read <url> --submit-timeout 20m\n")
		b.WriteString("Or try these alternatives:\n")
		b.WriteString("  - Pass --submit-timeout 0 to wait as long as archive.today takes.\n")
		b.WriteString("  - Submit manually at https://archive.ph/ in your browser (same backend, no client budget).\n")
		b.WriteString("  - The URL may require auth or a login-walled capture; try Wayback at https://web.archive.org/save/ instead.")
	} else if e.Cooldown != nil {
		mins := int(e.Cooldown.Minutes())
		if mins < 1 {
			mins = 1
		}
		b.WriteString("\n")
		if e.CooldownUntil != nil {
			b.WriteString(fmt.Sprintf("Cooldown: retry in %d minutes (until %s).\n",
				mins, e.CooldownUntil.Format("15:04")))
		} else {
			b.WriteString(fmt.Sprintf("Cooldown: retry in %d minutes.\n", mins))
		}
		b.WriteString("Alternative: submit manually at https://archive.ph/ in your browser.\n")
		b.WriteString("Browser traffic has a separate rate-limit budget from programmatic clients.")
	}

	return b.String()
}

// formatBudget renders a time.Duration as a short human phrase (e.g. "10m",
// "1h30m", "45s"). Used in error messages so users see the budget they set.
func formatBudget(d time.Duration) string {
	if d <= 0 {
		return "unbounded"
	}
	// time.Duration.String() handles most cases fine ("10m0s" → trim 0s).
	s := d.String()
	s = strings.TrimSuffix(s, "0s")
	if s == "" {
		s = "0s"
	}
	return s
}

// isAllRateLimited reports whether every mirror attempt returned HTTP 429.
func (e *SubmitFailureError) isAllRateLimited() bool {
	if len(e.Attempts) == 0 {
		return false
	}
	for _, a := range e.Attempts {
		if a.HTTPCode != 429 {
			return false
		}
	}
	return true
}

// MarshalJSON produces a structured JSON representation for --json output.
func (e *SubmitFailureError) MarshalJSON() ([]byte, error) {
	out := map[string]any{
		"error":            "submit_failed",
		"summary":          firstLine(e.Error()),
		"attempts":         e.Attempts,
		"all_rate_limited": e.isAllRateLimited(),
		"budget_exhausted": e.BudgetExhausted,
	}
	if e.Budget > 0 {
		out["budget_seconds"] = int(e.Budget.Seconds())
	}
	if e.Cooldown != nil {
		out["cooldown_seconds"] = int(e.Cooldown.Seconds())
	}
	if e.CooldownUntil != nil {
		out["cooldown_until"] = e.CooldownUntil.Format(time.RFC3339)
	}
	return json.Marshal(out)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
