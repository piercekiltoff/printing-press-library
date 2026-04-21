// Copyright 2026 matt-van-horn. Licensed under Apache-2.0.

package credentials

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

// TestMain installs the go-keyring mock provider so these tests don't touch
// the real OS keychain. Without this, CI Linux machines lacking Secret Service
// would hard-fail, and macOS developer machines would get UI unlock prompts
// during test runs.
//
// To run against the real keychain, set EXPENSIFY_CREDENTIALS_TEST_REAL=1 —
// tests will then skip cleanly if a keychain probe fails.
func TestMain(m *testing.M) {
	if os.Getenv("EXPENSIFY_CREDENTIALS_TEST_REAL") == "" {
		keyring.MockInit()
	}
	os.Exit(m.Run())
}

// keychainAvailable probes the backend by writing and immediately deleting a
// dummy entry. Used to skip tests on machines where the real keychain isn't
// reachable (headless Linux without Secret Service, locked macOS CI).
func keychainAvailable(t *testing.T) bool {
	t.Helper()
	probeEmail := fmt.Sprintf("probe-%d@expensify-pp-cli.test", time.Now().UnixNano())
	if err := keyring.Set(ServiceKey, probeEmail, "probe"); err != nil {
		return false
	}
	_ = keyring.Delete(ServiceKey, probeEmail)
	return true
}

// testEmail returns a unique email scoped to this test invocation so parallel
// or serial test runs on a shared keychain don't collide.
func testEmail(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test-%d-%s@expensify-pp-cli.test", time.Now().UnixNano(), t.Name())
}

func TestSet_ThenGet(t *testing.T) {
	if !keychainAvailable(t) {
		t.Skip("keychain not available on this machine")
	}
	email := testEmail(t)
	t.Cleanup(func() { _ = Delete(email) })

	if err := Set(email, "pw1"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := Get(email)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "pw1" {
		t.Fatalf("Get returned %q, want %q", got, "pw1")
	}
}

func TestGet_NotFound(t *testing.T) {
	if !keychainAvailable(t) {
		t.Skip("keychain not available on this machine")
	}
	email := testEmail(t)
	_, err := Get(email)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get on missing entry: err = %v, want ErrNotFound", err)
	}
}

func TestDelete_ThenGet(t *testing.T) {
	if !keychainAvailable(t) {
		t.Skip("keychain not available on this machine")
	}
	email := testEmail(t)
	t.Cleanup(func() { _ = Delete(email) })

	if err := Set(email, "pw1"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := Delete(email); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := Get(email)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after Delete: err = %v, want ErrNotFound", err)
	}
}

func TestHas_True(t *testing.T) {
	if !keychainAvailable(t) {
		t.Skip("keychain not available on this machine")
	}
	email := testEmail(t)
	t.Cleanup(func() { _ = Delete(email) })

	if err := Set(email, "pw1"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !Has(email) {
		t.Fatalf("Has after Set = false, want true")
	}
}

func TestHas_False(t *testing.T) {
	if !keychainAvailable(t) {
		t.Skip("keychain not available on this machine")
	}
	email := testEmail(t)
	if Has(email) {
		t.Fatalf("Has on unknown email = true, want false")
	}
}

func TestSet_EmptyEmail(t *testing.T) {
	if err := Set("", "pw1"); !errors.Is(err, ErrEmptyEmail) {
		t.Fatalf("Set(empty email): err = %v, want ErrEmptyEmail", err)
	}
}

func TestSet_EmptyPassword(t *testing.T) {
	if err := Set("a@b.com", ""); !errors.Is(err, ErrEmptyPassword) {
		t.Fatalf("Set(empty password): err = %v, want ErrEmptyPassword", err)
	}
}
