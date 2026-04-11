package cli

import (
	"strings"
	"testing"
	"time"
)

func TestSubmitFailureError_AllRateLimited(t *testing.T) {
	d := 55 * time.Minute
	until := time.Now().Add(d)
	e := &SubmitFailureError{
		Attempts: []MirrorResult{
			{URL: "https://archive.ph", HTTPCode: 429, Attempts: 4},
			{URL: "https://archive.md", HTTPCode: 429, Attempts: 4},
			{URL: "https://archive.is", HTTPCode: 429, Attempts: 4},
		},
		Cooldown:      &d,
		CooldownUntil: &until,
	}
	msg := e.Error()
	if !strings.Contains(msg, "rate-limited this IP across all mirrors") {
		t.Errorf("missing rate-limit summary: %q", msg)
	}
	for _, host := range []string{"archive.ph", "archive.md", "archive.is"} {
		if !strings.Contains(msg, host) {
			t.Errorf("missing mirror %s in report", host)
		}
	}
	if !strings.Contains(msg, "Cooldown: retry in") {
		t.Errorf("missing cooldown line: %q", msg)
	}
	if !strings.Contains(msg, "Alternative: submit manually") {
		t.Errorf("missing remediation line: %q", msg)
	}
	if !strings.Contains(msg, "attempts: 4") {
		t.Errorf("missing attempts count: %q", msg)
	}
}

func TestSubmitFailureError_Mixed(t *testing.T) {
	e := &SubmitFailureError{
		Attempts: []MirrorResult{
			{URL: "https://archive.ph", HTTPCode: 500},
			{URL: "https://archive.md", HTTPCode: 429, Attempts: 4},
		},
	}
	msg := e.Error()
	if strings.Contains(msg, "across all mirrors") {
		t.Errorf("should not claim all-rate-limited when mixed: %q", msg)
	}
	if !strings.Contains(msg, "HTTP 500") {
		t.Errorf("missing HTTP 500 line: %q", msg)
	}
	if !strings.Contains(msg, "HTTP 429") {
		t.Errorf("missing HTTP 429 line: %q", msg)
	}
}

func TestSubmitFailureError_Empty(t *testing.T) {
	e := &SubmitFailureError{}
	msg := e.Error()
	if !strings.Contains(msg, "no mirrors attempted") {
		t.Errorf("empty error should say no mirrors attempted: %q", msg)
	}
}

func TestClassifySubmitError(t *testing.T) {
	d := 1 * time.Hour
	allRL := &SubmitFailureError{
		Attempts: []MirrorResult{{HTTPCode: 429}},
		Cooldown: &d,
	}
	if err := classifySubmitError(allRL); err == nil {
		t.Fatal("expected non-nil")
	} else {
		ce, ok := err.(*cliError)
		if !ok {
			t.Fatalf("not a cliError: %T", err)
		}
		if ce.code != 7 {
			t.Errorf("expected exit code 7 for rate limit, got %d", ce.code)
		}
	}

	mixed := &SubmitFailureError{
		Attempts: []MirrorResult{{HTTPCode: 500}},
	}
	if err := classifySubmitError(mixed); err == nil {
		t.Fatal("expected non-nil")
	} else {
		ce, ok := err.(*cliError)
		if !ok {
			t.Fatalf("not a cliError: %T", err)
		}
		if ce.code != 5 {
			t.Errorf("expected exit code 5 for api error, got %d", ce.code)
		}
	}

	cooldownErr := &CooldownError{Remaining: 30 * time.Minute, Until: time.Now().Add(30 * time.Minute)}
	if err := classifySubmitError(cooldownErr); err == nil {
		t.Fatal("expected non-nil")
	} else {
		ce, ok := err.(*cliError)
		if !ok {
			t.Fatalf("not a cliError: %T", err)
		}
		if ce.code != 7 {
			t.Errorf("expected exit code 7 for cooldown, got %d", ce.code)
		}
	}

	// Budget-exhausted must route to apiErr (exit 5), not rateLimitErr (exit 7).
	// Telling a user "wait for cooldown" when they actually need a longer
	// --submit-timeout is a bad instruction.
	budgetErr := &SubmitFailureError{
		BudgetExhausted: true,
		Budget:          10 * time.Minute,
		Attempts: []MirrorResult{
			{URL: "https://archive.ph", Attempts: 1},
		},
	}
	if err := classifySubmitError(budgetErr); err == nil {
		t.Fatal("expected non-nil")
	} else {
		ce, ok := err.(*cliError)
		if !ok {
			t.Fatalf("not a cliError: %T", err)
		}
		if ce.code != 5 {
			t.Errorf("expected exit code 5 for budget exhausted, got %d", ce.code)
		}
	}
}

func TestSubmitFailureError_BudgetExhausted(t *testing.T) {
	e := &SubmitFailureError{
		BudgetExhausted: true,
		Budget:          10 * time.Minute,
		Attempts: []MirrorResult{
			{URL: "https://archive.ph", Attempts: 1},
		},
	}
	msg := e.Error()
	if !strings.Contains(msg, "budget exhausted after 10m") {
		t.Errorf("summary missing budget phrase: %q", msg)
	}
	if !strings.Contains(msg, "--submit-timeout 20m") {
		t.Errorf("remediation missing retry hint: %q", msg)
	}
	if !strings.Contains(msg, "--submit-timeout 0") {
		t.Errorf("remediation missing unbounded option: %q", msg)
	}
	if !strings.Contains(msg, "archive.ph") {
		t.Errorf("remediation missing manual fallback: %q", msg)
	}
	// Must NOT include the rate-limit cooldown guidance.
	if strings.Contains(msg, "Cooldown: retry in") {
		t.Errorf("budget-exhausted should not mention cooldown: %q", msg)
	}
}
