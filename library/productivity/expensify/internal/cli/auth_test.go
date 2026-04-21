// Copyright 2026 matt-van-horn. Licensed under Apache-2.0.
//
// Tests for the `auth store-credentials` subcommand and the email/credentials
// extensions to `auth status`. The keychain is mocked via keyring.MockInit()
// so these tests don't prompt the OS keychain on developer machines or
// hard-fail on CI Linux without Secret Service.

package cli

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/config"
	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/credentials"

	"github.com/zalando/go-keyring"
)

// init installs the go-keyring mock so every test in the cli package uses a
// process-local, in-memory keychain. Cheaper and safer than TestMain because
// the cli package may pick up additional *_test.go files that need the same
// guarantee.
func init() {
	keyring.MockInit()
}

// newAuthTestFlags returns a rootFlags whose configPath points at a fresh temp
// file so each test starts from a clean slate.
func newAuthTestFlags(t *testing.T) *rootFlags {
	t.Helper()
	dir := t.TempDir()
	return &rootFlags{configPath: filepath.Join(dir, "config.toml")}
}

// uniqueEmail returns a per-test email so the mocked keychain doesn't see
// cross-test collisions (tests can leave state in the mock map across
// invocations within one process).
func uniqueEmail(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test-%d@expensify-pp-cli.test", time.Now().UnixNano())
}

// runAuthCmd invokes the "auth" subtree against the given flags + argv. It
// returns stdout, stderr (combined), and the error cobra returned.
func runAuthCmd(t *testing.T, flags *rootFlags, argv ...string) (string, error) {
	t.Helper()
	root := newAuthCmd(flags)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(argv)
	err := root.Execute()
	return buf.String(), err
}

func TestAuthStoreCredentials_WithFlags(t *testing.T) {
	flags := newAuthTestFlags(t)
	email := uniqueEmail(t)
	t.Cleanup(func() { _ = credentials.Delete(email) })

	out, err := runAuthCmd(t, flags, "store-credentials", "--email", email, "--password", "pw")
	if err != nil {
		t.Fatalf("store-credentials: err = %v\nout: %s", err, out)
	}
	if !credentials.Has(email) {
		t.Fatalf("credentials.Has(%q) = false after store-credentials\nout: %s", email, out)
	}

	cfg, err := config.Load(flags.configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.ExpensifyEmail != email {
		t.Fatalf("cfg.ExpensifyEmail = %q, want %q", cfg.ExpensifyEmail, email)
	}
	if !strings.Contains(out, fmt.Sprintf("Credentials stored for %s", email)) {
		t.Fatalf("output missing confirmation line; got:\n%s", out)
	}
}

func TestAuthStoreCredentials_NoInput_MissingPassword(t *testing.T) {
	flags := newAuthTestFlags(t)
	flags.noInput = true
	email := uniqueEmail(t)

	out, err := runAuthCmd(t, flags, "store-credentials", "--email", email)
	if err == nil {
		t.Fatalf("expected usage error with --no-input and no password; got out: %s", out)
	}
	var ce *cliError
	if !errors.As(err, &ce) {
		t.Fatalf("err = %v (%T), want *cliError", err, err)
	}
	if ce.code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage)", ce.code)
	}
}

func TestAuthStoreCredentials_FromEnv(t *testing.T) {
	flags := newAuthTestFlags(t)
	email := uniqueEmail(t)
	t.Cleanup(func() { _ = credentials.Delete(email) })

	t.Setenv("EXPENSIFY_EMAIL", email)
	t.Setenv("EXPENSIFY_PASSWORD", "pw")

	out, err := runAuthCmd(t, flags, "store-credentials", "--from-env")
	if err != nil {
		t.Fatalf("store-credentials --from-env: err = %v\nout: %s", err, out)
	}
	if !credentials.Has(email) {
		t.Fatalf("credentials.Has(%q) = false\nout: %s", email, out)
	}
	got, err := credentials.Get(email)
	if err != nil || got != "pw" {
		t.Fatalf("credentials.Get = (%q, %v), want (%q, nil)", got, err, "pw")
	}
	cfg, err := config.Load(flags.configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.ExpensifyEmail != email {
		t.Fatalf("cfg.ExpensifyEmail = %q, want %q", cfg.ExpensifyEmail, email)
	}
}

func TestAuthStatus_WithCredentials(t *testing.T) {
	flags := newAuthTestFlags(t)
	email := uniqueEmail(t)
	t.Cleanup(func() { _ = credentials.Delete(email) })

	// Seed via the real command so we exercise the persistence path.
	if out, err := runAuthCmd(t, flags, "store-credentials", "--email", email, "--password", "pw"); err != nil {
		t.Fatalf("seed store-credentials: %v\n%s", err, out)
	}

	out, err := runAuthCmd(t, flags, "status")
	// `auth status` returns authErr when neither session nor partner auth are
	// set, even if headless credentials are configured — that's the intended
	// behaviour (credentials alone can't call the API; you still need a token).
	// We just assert the output lines are present.
	_ = err
	if !strings.Contains(out, fmt.Sprintf("Email: %s", email)) {
		t.Fatalf("status output missing %q line; got:\n%s", fmt.Sprintf("Email: %s", email), out)
	}
	if !strings.Contains(out, "Headless credentials: configured") {
		t.Fatalf("status output missing %q line; got:\n%s", "Headless credentials: configured", out)
	}
}

func TestAuthStatus_NoCredentials(t *testing.T) {
	flags := newAuthTestFlags(t)

	out, _ := runAuthCmd(t, flags, "status")
	if !strings.Contains(out, "Email: not configured") {
		t.Fatalf("status output missing %q; got:\n%s", "Email: not configured", out)
	}
	if !strings.Contains(out, "Headless credentials: not configured") {
		t.Fatalf("status output missing %q; got:\n%s", "Headless credentials: not configured", out)
	}
}
