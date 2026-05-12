// Copyright 2026 dstevens. Licensed under Apache-2.0. See LICENSE.

package granola

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mvanhorn/printing-press-library/library/productivity/granola/internal/cliutil"
)

// WorkOSClientID is the Granola desktop client's WorkOS application client
// id. It is hardcoded across the community ecosystem (getprobo, granola.py,
// granola-mcp) because Granola does not document a per-user OAuth app for
// the internal API; this value is the only client_id WorkOS will accept on
// the refresh-token endpoint for Granola's tokens.
const WorkOSClientID = "client_01HJK46TGGY2DFQ2NX9P9XYJZN"

// WorkOSAuthEndpoint is the refresh endpoint. POST a JSON body of
// {client_id, grant_type:"refresh_token", refresh_token} and you receive
// a new access_token plus a NEW refresh_token (single-use rotation).
const WorkOSAuthEndpoint = "https://api.workos.com/user_management/authenticate"

// granolaSupportDir is the macOS support directory for Granola.
func granolaSupportDir() string {
	if v := os.Getenv("GRANOLA_SUPPORT_DIR"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "Granola")
}

// supabaseJSONPath returns the path to the supabase.json token file.
func supabaseJSONPath() string {
	return filepath.Join(granolaSupportDir(), "supabase.json")
}

// storedAccountsPath returns the path to the stored-accounts.json fallback.
func storedAccountsPath() string {
	return filepath.Join(granolaSupportDir(), "stored-accounts.json")
}

// workosTokens is the inner shape of the (stringified) workos_tokens blob.
type workosTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ObtainedAt   int64  `json:"obtained_at"` // millis since epoch
	ExpiresIn    int    `json:"expires_in"`  // seconds
	TokenType    string `json:"token_type"`
	SignInMethod string `json:"sign_in_method"`
	ExternalID   string `json:"external_id"`
	SessionID    string `json:"session_id"`
}

// supabaseFile is the top-level shape of supabase.json. workos_tokens is a
// JSON-stringified blob, hence the json.RawMessage indirection.
type supabaseFile struct {
	SessionID    string          `json:"session_id"`
	UserInfo     json.RawMessage `json:"user_info"`
	WorkOSTokens json.RawMessage `json:"workos_tokens"`
}

// storedAccountsFile is the top-level shape of stored-accounts.json.
type storedAccountsFile struct {
	Accounts json.RawMessage `json:"accounts"`
}

// storedAccount is one entry in the (stringified) accounts array.
type storedAccount struct {
	Email    string          `json:"email"`
	UserID   string          `json:"userId"`
	SavedAt  string          `json:"savedAt"`
	Tokens   json.RawMessage `json:"tokens"`
	UserInfo json.RawMessage `json:"userInfo"`
}

// LoadAccessToken returns the current access token + its expiry time.
// It returns the in-memory cached token if RefreshAccessToken has been
// called this session and minted a newer one; otherwise it reads
// supabase.json, then stored-accounts.json, then env GRANOLA_WORKOS_TOKEN.
//
// The returned expiry is computed from obtained_at + expires_in. Callers
// SHOULD compare against time.Now() and call RefreshAccessToken if it has
// passed. The reverse path (network call) is what the InternalClient does
// on 401 from the API itself; both paths share the in-process cache.
func LoadAccessToken() (string, time.Time, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	if cachedAccess != "" && !cachedExpiry.IsZero() {
		return cachedAccess, cachedExpiry, nil
	}
	tok, err := loadTokensRaw()
	if err != nil {
		return "", time.Time{}, err
	}
	cachedAccess = tok.AccessToken
	cachedRefresh = tok.RefreshToken
	cachedExpiry = tok.expiry()
	return cachedAccess, cachedExpiry, nil
}

// LoadRefreshToken returns the refresh token (read-only).
func LoadRefreshToken() (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	if cachedRefresh != "" {
		return cachedRefresh, nil
	}
	tok, err := loadTokensRaw()
	if err != nil {
		return "", err
	}
	cachedAccess = tok.AccessToken
	cachedRefresh = tok.RefreshToken
	cachedExpiry = tok.expiry()
	return cachedRefresh, nil
}

// in-process token cache. Survives across multiple calls in one CLI run.
var (
	tokenMu       sync.Mutex
	cachedAccess  string
	cachedRefresh string
	cachedExpiry  time.Time
	// refreshClient is the HTTP client used for refresh calls. Tests may
	// override it with a transport that mocks WorkOS responses.
	refreshClient = &http.Client{Timeout: 15 * time.Second}
)

// SetRefreshHTTPClient swaps the HTTP client used for WorkOS refreshes.
// Tests use this to inject mocked transports.
func SetRefreshHTTPClient(c *http.Client) {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	refreshClient = c
}

// ResetTokenCache clears the in-process token cache. Tests call this to
// force re-reading the on-disk source.
func ResetTokenCache() {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	cachedAccess = ""
	cachedRefresh = ""
	cachedExpiry = time.Time{}
}

func (t workosTokens) expiry() time.Time {
	if t.ObtainedAt == 0 || t.ExpiresIn == 0 {
		return time.Time{}
	}
	obtained := time.UnixMilli(t.ObtainedAt)
	return obtained.Add(time.Duration(t.ExpiresIn) * time.Second)
}

// loadTokensRaw reads the on-disk token, trying supabase.json first then
// stored-accounts.json. Returns the most-recent token by SavedAt.
func loadTokensRaw() (workosTokens, error) {
	// Env-var override path (mostly for tests / CI smoke).
	if v := os.Getenv("GRANOLA_WORKOS_TOKEN"); v != "" {
		// Synthesize an expiry far in the future.
		return workosTokens{
			AccessToken:  v,
			RefreshToken: os.Getenv("GRANOLA_WORKOS_REFRESH"),
			ObtainedAt:   time.Now().UnixMilli(),
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		}, nil
	}

	if tok, err := loadFromSupabaseJSON(); err == nil {
		return tok, nil
	}
	if tok, err := loadFromStoredAccountsJSON(); err == nil {
		return tok, nil
	}
	return workosTokens{}, fmt.Errorf("no Granola token found in supabase.json or stored-accounts.json; sign into the Granola desktop app or set GRANOLA_WORKOS_TOKEN")
}

func loadFromSupabaseJSON() (workosTokens, error) {
	data, err := os.ReadFile(supabaseJSONPath())
	if err != nil {
		return workosTokens{}, err
	}
	var top supabaseFile
	if err := json.Unmarshal(data, &top); err != nil {
		return workosTokens{}, err
	}
	if len(top.WorkOSTokens) == 0 {
		return workosTokens{}, fmt.Errorf("supabase.json: workos_tokens missing")
	}
	raw := unwrapStringifiedJSON(top.WorkOSTokens)
	var tok workosTokens
	if err := json.Unmarshal(raw, &tok); err != nil {
		return workosTokens{}, fmt.Errorf("supabase.json: parsing workos_tokens: %w", err)
	}
	if tok.AccessToken == "" {
		return workosTokens{}, fmt.Errorf("supabase.json: empty access_token")
	}
	return tok, nil
}

func loadFromStoredAccountsJSON() (workosTokens, error) {
	data, err := os.ReadFile(storedAccountsPath())
	if err != nil {
		return workosTokens{}, err
	}
	var top storedAccountsFile
	if err := json.Unmarshal(data, &top); err != nil {
		return workosTokens{}, err
	}
	if len(top.Accounts) == 0 {
		return workosTokens{}, fmt.Errorf("stored-accounts.json: accounts missing")
	}
	raw := unwrapStringifiedJSON(top.Accounts)
	var accts []storedAccount
	if err := json.Unmarshal(raw, &accts); err != nil {
		return workosTokens{}, fmt.Errorf("stored-accounts.json: parsing accounts: %w", err)
	}
	if len(accts) == 0 {
		return workosTokens{}, fmt.Errorf("stored-accounts.json: no accounts")
	}
	// Iterate every account; keep the newest by SavedAt with a parseable
	// tokens blob.
	var best workosTokens
	var bestSaved string
	for _, a := range accts {
		if len(a.Tokens) == 0 {
			continue
		}
		inner := unwrapStringifiedJSON(a.Tokens)
		var tok workosTokens
		if err := json.Unmarshal(inner, &tok); err != nil {
			continue
		}
		if tok.AccessToken == "" {
			continue
		}
		if a.SavedAt > bestSaved {
			best = tok
			bestSaved = a.SavedAt
		}
	}
	if best.AccessToken == "" {
		return workosTokens{}, fmt.Errorf("stored-accounts.json: no usable tokens")
	}
	return best, nil
}

// RefreshAccessTokenResponse is the parsed body from a WorkOS refresh call.
type RefreshAccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	// WorkOS returns the new expiry as expires_in (seconds) when present.
	ExpiresIn int `json:"expires_in"`
}

// workosLimiter paces WorkOS refresh-token calls. The endpoint is hit at most
// once per CLI invocation under normal conditions; the limiter is here for the
// pathological case where a caller burst-refreshes (and so the typed 429
// contract below is exercised by the AdaptiveLimiter as well).
var workosLimiter = cliutil.NewAdaptiveLimiter(2.0)

// RefreshAccessToken exchanges the current refresh token for a new
// access/refresh pair. WorkOS rotates refresh tokens single-use per
// getprobo's findings; the caller MUST persist the new refresh token if
// it intends to refresh again (we cache it in-process only — we do not
// write back to Granola's files).
func RefreshAccessToken(refreshToken string) (RefreshAccessTokenResponse, error) {
	body, _ := json.Marshal(map[string]string{
		"client_id":     WorkOSClientID,
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	})
	req, err := http.NewRequest("POST", WorkOSAuthEndpoint, bytes.NewReader(body))
	if err != nil {
		return RefreshAccessTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	workosLimiter.Wait()
	resp, err := refreshClient.Do(req)
	if err != nil {
		return RefreshAccessTokenResponse{}, fmt.Errorf("workos refresh: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	// Typed 429 handling — surface WorkOS throttling as cliutil.RateLimitError
	// so the caller can distinguish "rate limited" from "auth failed."
	if resp.StatusCode == http.StatusTooManyRequests {
		workosLimiter.OnRateLimit()
		wait := cliutil.RetryAfter(resp)
		return RefreshAccessTokenResponse{}, &cliutil.RateLimitError{
			URL:        WorkOSAuthEndpoint,
			RetryAfter: wait,
			Body:       string(respBody),
		}
	}
	workosLimiter.OnSuccess()
	if resp.StatusCode != http.StatusOK {
		return RefreshAccessTokenResponse{}, fmt.Errorf("workos refresh: status %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed RefreshAccessTokenResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return RefreshAccessTokenResponse{}, fmt.Errorf("workos refresh: parse response: %w", err)
	}
	if parsed.AccessToken == "" {
		return RefreshAccessTokenResponse{}, fmt.Errorf("workos refresh: empty access_token in response")
	}
	// Cache the new pair in-process.
	tokenMu.Lock()
	cachedAccess = parsed.AccessToken
	if parsed.RefreshToken != "" {
		cachedRefresh = parsed.RefreshToken
	}
	if parsed.ExpiresIn > 0 {
		cachedExpiry = time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second)
	}
	tokenMu.Unlock()
	return parsed, nil
}
