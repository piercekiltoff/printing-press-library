// Copyright 2026 dstevens. Licensed under Apache-2.0. See LICENSE.

package granola

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRefreshAccessToken_RotatesRefresh(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`))
	}))
	defer srv.Close()

	// Re-point the refresh endpoint at the test server by overriding
	// the HTTP client to rewrite the URL.
	origClient := refreshClient
	SetRefreshHTTPClient(&http.Client{
		Transport: &rewriteTransport{target: srv.URL},
	})
	defer SetRefreshHTTPClient(origClient)

	resp, err := RefreshAccessToken("old-refresh")
	if err != nil {
		t.Fatalf("RefreshAccessToken: %v", err)
	}
	if resp.AccessToken != "new-access" {
		t.Errorf("expected new-access, got %q", resp.AccessToken)
	}
	if resp.RefreshToken != "new-refresh" {
		t.Errorf("expected new-refresh, got %q", resp.RefreshToken)
	}
	if gotBody["grant_type"] != "refresh_token" {
		t.Errorf("expected grant_type=refresh_token, got %q", gotBody["grant_type"])
	}
	if gotBody["refresh_token"] != "old-refresh" {
		t.Errorf("expected refresh_token=old-refresh, got %q", gotBody["refresh_token"])
	}
	if gotBody["client_id"] != WorkOSClientID {
		t.Errorf("expected client_id=%q, got %q", WorkOSClientID, gotBody["client_id"])
	}

	// Verify cache holds the new pair.
	ResetTokenCache()
}

// rewriteTransport replaces the request URL with target+path.
type rewriteTransport struct {
	target string
}

func (rt *rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Strip scheme+host and replace with target.
	r.URL.Scheme = ""
	r.URL.Host = ""
	new := strings.TrimRight(rt.target, "/") + r.URL.RequestURI()
	r2, err := http.NewRequest(r.Method, new, r.Body)
	if err != nil {
		return nil, err
	}
	r2.Header = r.Header
	return http.DefaultTransport.RoundTrip(r2)
}

func TestRefreshAccessToken_RejectsNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()
	origClient := refreshClient
	SetRefreshHTTPClient(&http.Client{Transport: &rewriteTransport{target: srv.URL}})
	defer SetRefreshHTTPClient(origClient)
	_, err := RefreshAccessToken("bad")
	if err == nil {
		t.Fatalf("expected error on 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected error to mention 401, got %v", err)
	}
}
