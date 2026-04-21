// Copyright 2026 matt-van-horn. Licensed under Apache-2.0.
//
// Package credentials wraps github.com/zalando/go-keyring to store Expensify
// email+password pairs in the OS-native keychain (macOS Keychain, Windows
// Credential Manager, Linux Secret Service). Passwords never touch the TOML
// config file.
package credentials

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// ServiceKey is the keychain service identifier. All Expensify secrets written
// by this CLI land under this key, so `security dump-keychain` or `seahorse`
// shows them grouped.
const ServiceKey = "expensify-pp-cli"

// ErrNotFound is returned by Get when no password is stored for the given
// email. Callers can use errors.Is to detect it without depending on the
// underlying keyring package.
var ErrNotFound = errors.New("credentials: not found")

// ErrEmptyEmail is returned when Set/Get/Delete is called with an empty email.
var ErrEmptyEmail = errors.New("credentials: email is required")

// ErrEmptyPassword is returned when Set is called with an empty password.
var ErrEmptyPassword = errors.New("credentials: password is required")

// Set writes the password into the OS keychain under (ServiceKey, email).
// Returns ErrEmptyEmail / ErrEmptyPassword for empty arguments, or a wrapped
// keychain error on failure.
func Set(email, password string) error {
	if email == "" {
		return ErrEmptyEmail
	}
	if password == "" {
		return ErrEmptyPassword
	}
	if err := keyring.Set(ServiceKey, email, password); err != nil {
		return fmt.Errorf("credentials: set keychain entry: %w", err)
	}
	return nil
}

// Get reads the password stored for email. Returns ErrNotFound (wrapped if
// there's additional context) when the entry is missing.
func Get(email string) (string, error) {
	if email == "" {
		return "", ErrEmptyEmail
	}
	pw, err := keyring.Get(ServiceKey, email)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("credentials: get keychain entry: %w", err)
	}
	return pw, nil
}

// Delete removes the stored password. Deleting a non-existent entry returns
// ErrNotFound so callers can distinguish "already gone" from "real failure".
func Delete(email string) error {
	if email == "" {
		return ErrEmptyEmail
	}
	if err := keyring.Delete(ServiceKey, email); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("credentials: delete keychain entry: %w", err)
	}
	return nil
}

// Has reports whether a password is configured for email. Empty email always
// returns false. Any error other than "not found" also returns false — callers
// who care about the distinction should use Get directly.
func Has(email string) bool {
	if email == "" {
		return false
	}
	_, err := Get(email)
	return err == nil
}
