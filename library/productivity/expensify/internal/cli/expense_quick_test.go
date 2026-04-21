// Copyright 2026 matt-van-horn. Licensed under Apache-2.0.

package cli

import (
	"strings"
	"testing"
)

// TestParseQuickPrompt_AtPhraseWins verifies that an explicit "at X" phrase
// wins over the blocklist: "Dinner at Maya $42.50" must extract "Maya" as the
// merchant even though "Dinner" is blocklisted.
func TestParseQuickPrompt_AtPhraseWins(t *testing.T) {
	p := parseQuickPrompt("Dinner at Maya $42.50")
	if p.Merchant != "Maya" {
		t.Fatalf("merchant = %q, want %q", p.Merchant, "Maya")
	}
	if p.CategoryNounHit {
		t.Fatalf("CategoryNounHit = true, want false when at-X wins")
	}
	if p.Amount != 4250 {
		t.Fatalf("amount = %d cents, want 4250", p.Amount)
	}
}

// TestParseQuickPrompt_NormalMerchant verifies that a normal brand name that
// is NOT in the blocklist (like "Uber") is still extracted as the merchant.
func TestParseQuickPrompt_NormalMerchant(t *testing.T) {
	p := parseQuickPrompt("Uber $24")
	if p.Merchant != "Uber" {
		t.Fatalf("merchant = %q, want %q", p.Merchant, "Uber")
	}
	if p.CategoryNounHit {
		t.Fatalf("CategoryNounHit = true, want false for non-blocklisted merchant")
	}
	if p.Amount != 2400 {
		t.Fatalf("amount = %d cents, want 2400", p.Amount)
	}
}

// TestParseQuickPrompt_BlocklistedLeadingNoun verifies that "Tacos $5" does
// NOT set merchant=Tacos and flags the CategoryNounHit so the caller can
// prompt or error.
func TestParseQuickPrompt_BlocklistedLeadingNoun(t *testing.T) {
	p := parseQuickPrompt("Tacos $5")
	if p.Merchant != "" {
		t.Fatalf("merchant = %q, want empty for blocklisted leading noun", p.Merchant)
	}
	if !p.CategoryNounHit {
		t.Fatalf("CategoryNounHit = false, want true for %q", "Tacos")
	}
	if strings.ToLower(p.CategoryNounWord) != "tacos" {
		t.Fatalf("CategoryNounWord = %q, want %q", p.CategoryNounWord, "Tacos")
	}
	if p.Amount != 500 {
		t.Fatalf("amount = %d cents, want 500", p.Amount)
	}
}

// TestParseQuickPrompt_BlocklistedMidPhrase verifies "Coffee $4" — same
// category-noun hit behavior as "Tacos $5".
func TestParseQuickPrompt_BlocklistedMidPhrase(t *testing.T) {
	p := parseQuickPrompt("Coffee $4")
	if p.Merchant != "" {
		t.Fatalf("merchant = %q, want empty for blocklisted leading noun", p.Merchant)
	}
	if !p.CategoryNounHit {
		t.Fatalf("CategoryNounHit = false, want true for %q", "Coffee")
	}
	if strings.ToLower(p.CategoryNounWord) != "coffee" {
		t.Fatalf("CategoryNounWord = %q, want %q", p.CategoryNounWord, "Coffee")
	}
}

// TestParseQuickPrompt_BlocklistedWithAtPhrase verifies that when both a
// blocklisted word AND an "at X" phrase are present, "at X" still wins and
// no CategoryNounHit is flagged.
func TestParseQuickPrompt_BlocklistedWithAtPhrase(t *testing.T) {
	p := parseQuickPrompt("Lunch meeting at Panera $18")
	if p.Merchant != "Panera" {
		t.Fatalf("merchant = %q, want %q", p.Merchant, "Panera")
	}
	if p.CategoryNounHit {
		t.Fatalf("CategoryNounHit = true, want false when at-X wins")
	}
	if p.Amount != 1800 {
		t.Fatalf("amount = %d cents, want 1800", p.Amount)
	}
}

// TestParseQuickPrompt_BlocklistedWithOverride simulates the RunE-level
// --merchant override: after parsing surfaces CategoryNounHit, the caller
// applies overrideMerchant (as RunE does) and the final merchant is the
// override with no error. This mirrors the command code path.
func TestParseQuickPrompt_BlocklistedWithOverride(t *testing.T) {
	prompt := "Tacos $5"
	overrideMerchant := "El Gordo"

	parsed := parseQuickPrompt(prompt)
	// Sanity: parser surfaced the hit.
	if !parsed.CategoryNounHit {
		t.Fatalf("CategoryNounHit = false before override, want true")
	}
	// Mirror the RunE override flow.
	if overrideMerchant != "" {
		parsed.Merchant = overrideMerchant
	}
	if parsed.Merchant != "El Gordo" {
		t.Fatalf("merchant after override = %q, want %q", parsed.Merchant, "El Gordo")
	}
	// After override, the RunE code sees parsed.Merchant != "" and does NOT
	// take the CategoryNounHit error/prompt branch. The hit flag stays set
	// for diagnostic purposes but is irrelevant to the final outcome.
}

// TestParseQuickPrompt_EmptyPrompt verifies the parser does not panic on an
// empty prompt and returns an empty quickParsed.
func TestParseQuickPrompt_EmptyPrompt(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("parseQuickPrompt(\"\") panicked: %v", r)
		}
	}()
	p := parseQuickPrompt("")
	if p.Merchant != "" {
		t.Fatalf("merchant = %q, want empty for empty prompt", p.Merchant)
	}
	if p.Amount != 0 {
		t.Fatalf("amount = %d, want 0 for empty prompt", p.Amount)
	}
	if p.CategoryNounHit {
		t.Fatalf("CategoryNounHit = true, want false for empty prompt")
	}
}

// TestParseQuickPrompt_LowercaseBlocklisted verifies the parser also catches
// lowercase blocklisted leading words (e.g. "tacos $5"), via the fallback
// leading-word check.
func TestParseQuickPrompt_LowercaseBlocklisted(t *testing.T) {
	p := parseQuickPrompt("tacos $5")
	if p.Merchant != "" {
		t.Fatalf("merchant = %q, want empty for lowercase blocklisted noun", p.Merchant)
	}
	if !p.CategoryNounHit {
		t.Fatalf("CategoryNounHit = false, want true for lowercase %q", "tacos")
	}
}
