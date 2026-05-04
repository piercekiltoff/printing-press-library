// Copyright 2026 dinakar-sarbada. Licensed under Apache-2.0. See LICENSE.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// DefaultBearerToken is the public bearer token used by the X web client.
// It is not user-secret; it identifies the X web app to the GraphQL backend.
// Override via env X_TWITTER_BEARER_TOKEN if X rotates it.
const DefaultBearerToken = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3DbjLTQ4fjfwpTk0EqOZJEqiWVNfdz3Mxh99gHLs5Z9SuRf0CXhd"

type Config struct {
	BaseURL string `toml:"base_url"`

	// Cookie-based auth (primary; captured via `auth login --chrome`)
	AuthToken string `toml:"auth_token"` // X session cookie
	CSRFToken string `toml:"ct0"`        // CSRF cookie; mirrored into x-csrf-token header
	GuestID   string `toml:"guest_id"`   // X guest tracking cookie

	// Bearer token (overridable; defaults to DefaultBearerToken)
	BearerToken string `toml:"bearer_token"`

	// Multi-account support: which account is currently active
	ActiveAccount string `toml:"active_account"`

	// Cookie capture metadata
	CapturedAt    time.Time `toml:"captured_at"`
	CapturedFrom  string    `toml:"captured_from"` // "chrome", "manual", "env"

	// AuthSource and Path are not persisted
	AuthSource string `toml:"-"`
	Path       string `toml:"-"`

	// Legacy fields kept for compatibility with generated code that still references them
	AuthHeaderVal string    `toml:"auth_header"`
	AccessToken   string    `toml:"access_token"`
	RefreshToken  string    `toml:"refresh_token"`
	TokenExpiry   time.Time `toml:"token_expiry"`
	ClientID      string    `toml:"client_id"`
	ClientSecret  string    `toml:"client_secret"`
	TwitterAccept string    `toml:"accept"`
}

func Load(configPath string) (*Config, error) {
	cfg := &Config{
		BaseURL: "https://x.com/i/api",
	}

	// Resolve config path
	path := configPath
	if path == "" {
		path = os.Getenv("X_TWITTER_CONFIG")
	}
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".config", "x-twitter-pp-cli", "config.toml")
	}
	cfg.Path = path

	// Try to load config file
	data, err := os.ReadFile(path)
	if err == nil {
		if err := toml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config %s: %w", path, err)
		}
		if cfg.AuthToken != "" {
			cfg.AuthSource = "config:" + path
		}
	}

	// Env var overrides for cookie auth
	if v := os.Getenv("X_TWITTER_AUTH_TOKEN"); v != "" {
		cfg.AuthToken = v
		cfg.AuthSource = "env:X_TWITTER_AUTH_TOKEN"
	}
	if v := os.Getenv("X_TWITTER_CT0"); v != "" {
		cfg.CSRFToken = v
	}
	if v := os.Getenv("X_TWITTER_GUEST_ID"); v != "" {
		cfg.GuestID = v
	}
	if v := os.Getenv("X_TWITTER_BEARER_TOKEN"); v != "" {
		cfg.BearerToken = v
	}

	// Default bearer token if not set
	if cfg.BearerToken == "" {
		cfg.BearerToken = DefaultBearerToken
	}

	// Base URL override (used by printing-press verify to point at mock/test servers)
	if v := os.Getenv("X_TWITTER_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	return cfg, nil
}

// AuthHeader returns the Authorization header value (Bearer token).
// Cookie auth happens via the cookie jar; CSRF goes through a separate header.
func (c *Config) AuthHeader() string {
	if c.AuthHeaderVal != "" {
		return c.AuthHeaderVal
	}
	if c.BearerToken != "" {
		return "Bearer " + c.BearerToken
	}
	if c.TwitterAccept != "" {
		return "Bearer " + c.TwitterAccept
	}
	return ""
}

// HasCookieAuth reports whether the cookie chain (auth_token + ct0) is configured.
func (c *Config) HasCookieAuth() bool {
	return c.AuthToken != "" && c.CSRFToken != ""
}

// CookieHeader returns a Cookie header value for the X session.
func (c *Config) CookieHeader() string {
	parts := []string{}
	if c.AuthToken != "" {
		parts = append(parts, "auth_token="+c.AuthToken)
	}
	if c.CSRFToken != "" {
		parts = append(parts, "ct0="+c.CSRFToken)
	}
	if c.GuestID != "" {
		parts = append(parts, "guest_id="+c.GuestID)
	}
	return strings.Join(parts, "; ")
}

func applyAuthFormat(format string, replacements map[string]string) string {
	if format == "" {
		return ""
	}
	for key, value := range replacements {
		format = strings.ReplaceAll(format, "{"+key+"}", value)
	}
	if strings.Contains(format, "{") {
		return ""
	}
	return format
}

// SaveCookies persists the cookie chain captured from the browser.
func (c *Config) SaveCookies(authToken, csrfToken, guestID, source string) error {
	c.AuthToken = authToken
	c.CSRFToken = csrfToken
	c.GuestID = guestID
	c.CapturedAt = time.Now().UTC()
	c.CapturedFrom = source
	return c.save()
}

func (c *Config) SaveTokens(clientID, clientSecret, accessToken, refreshToken string, expiry time.Time) error {
	c.ClientID = clientID
	c.ClientSecret = clientSecret
	c.AccessToken = accessToken
	c.RefreshToken = refreshToken
	c.TokenExpiry = expiry
	return c.save()
}

func (c *Config) ClearTokens() error {
	c.AccessToken = ""
	c.RefreshToken = ""
	c.TokenExpiry = time.Time{}
	c.AuthToken = ""
	c.CSRFToken = ""
	c.GuestID = ""
	return c.save()
}

func (c *Config) save() error {
	dir := filepath.Dir(c.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(c.Path, data, 0o600)
}

// applyAuthFormat is retained for compatibility with the generator's
// templated auth-header substitution; reference it here so go vet doesn't
// flag it as unused on CLIs whose auth is fully cookie-based.
var _ = applyAuthFormat
