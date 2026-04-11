// Shared state directory helpers for archive-is-pp-cli.
//
// All persistent CLI state (rate-limit cooldown, async request records, future
// caches) lives under a single XDG-compliant directory. This file provides the
// one canonical resolution helper so no unit invents its own path.

package cli

import (
	"os"
	"path/filepath"
	"runtime"
)

// stateDir returns the per-user state directory for archive-is-pp-cli.
// Creates the directory if it does not exist. Resolution order:
//
//  1. $XDG_STATE_HOME/archive-is-pp-cli
//  2. macOS: $HOME/Library/Application Support/archive-is-pp-cli
//  3. $HOME/.local/share/archive-is-pp-cli
//  4. Fallback: $TMPDIR/archive-is-pp-cli (only if $HOME is unset)
//
// Returns the resolved path and any mkdir error. Callers should treat a
// non-nil error as "persistence unavailable" and fall back to in-memory
// behavior rather than crashing.
func stateDir() (string, error) {
	var base string
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		base = filepath.Join(xdg, "archive-is-pp-cli")
	} else if home := os.Getenv("HOME"); home != "" {
		if runtime.GOOS == "darwin" {
			base = filepath.Join(home, "Library", "Application Support", "archive-is-pp-cli")
		} else {
			base = filepath.Join(home, ".local", "share", "archive-is-pp-cli")
		}
	} else {
		base = filepath.Join(os.TempDir(), "archive-is-pp-cli")
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	return base, nil
}
