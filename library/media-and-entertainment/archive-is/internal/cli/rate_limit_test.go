package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCooldownRoundTrip(t *testing.T) {
	// Use a temp dir for isolation
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("HOME", tmpDir) // fallback path

	// Ensure no cooldown initially
	if in, _ := isInCooldown(); in {
		t.Fatalf("expected no cooldown on empty state, got in-cooldown")
	}

	// Write a cooldown and read it back
	writeCooldown(30 * time.Minute)
	in, remaining := isInCooldown()
	if !in {
		t.Fatalf("expected in-cooldown after writeCooldown")
	}
	if remaining < 29*time.Minute || remaining > 31*time.Minute {
		t.Fatalf("unexpected remaining %s", remaining)
	}

	// cooldownError should wrap it
	err := cooldownError()
	if err == nil {
		t.Fatalf("expected cooldownError to return non-nil")
	}
	if !IsCooldownError(err) {
		t.Fatalf("expected IsCooldownError to return true")
	}

	// Clear and verify empty
	clearCooldown()
	if in, _ := isInCooldown(); in {
		t.Fatalf("expected no cooldown after clear")
	}
}

func TestCooldownExpires(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Write a cooldown in the past
	path := rateLimitPath()
	if path == "" {
		t.Fatalf("rateLimitPath returned empty")
	}
	past := time.Now().Add(-1 * time.Hour)
	content := `{"last_429_at":"` + past.Format(time.RFC3339) + `","cooldown_until":"` + past.Format(time.RFC3339) + `"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if in, _ := isInCooldown(); in {
		t.Fatalf("expected expired cooldown to return false")
	}
}

func TestCooldownMalformed(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	dir, _ := stateDir()
	if err := os.WriteFile(filepath.Join(dir, rateLimitStateFile), []byte("not json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if in, _ := isInCooldown(); in {
		t.Fatalf("expected malformed file to be treated as no cooldown")
	}
}

func TestCooldownClockSkew(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)

	// Cooldown 48h in the future — should be treated as stale/skew
	path := rateLimitPath()
	future := time.Now().Add(48 * time.Hour)
	content := `{"last_429_at":"` + time.Now().Format(time.RFC3339) + `","cooldown_until":"` + future.Format(time.RFC3339) + `"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if in, _ := isInCooldown(); in {
		t.Fatalf("expected 48h-future cooldown to be treated as stale")
	}
}

func TestStateDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	dir, err := stateDir()
	if err != nil {
		t.Fatalf("stateDir error: %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("stateDir did not create directory: %s", dir)
	}
}
