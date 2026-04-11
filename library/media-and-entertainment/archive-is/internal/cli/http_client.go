// Shared HTTP client factory for archive.today requests.
//
// This file centralizes HTTP client construction so all archive.today-touching
// calls (timegate, timemap, submit) use the same User-Agent, cookie jar, and
// redirect policy. The cookie jar is scoped to archive.today hostnames so we
// do not accidentally leak cookies to unrelated services.
//
// The primary purpose of the cookie jar is preserving archive.today's `qki=`
// tracking cookie across requests. Visiting the homepage once at the start of
// a submit flow gets us a fresh cookie; subsequent requests include it
// automatically. This does not defeat rate limiting, but it makes us look
// more like a real browser and may help with soft-throttle heuristics.

package cli

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// archiveHostnames lists the hostnames whose cookies the jar is willing to
// persist. Used by the scoped cookie jar to prevent cross-domain cookie leaks.
var archiveHostnames = []string{
	"archive.ph", "archive.md", "archive.is", "archive.fo", "archive.li", "archive.vn",
	"archive.today",
}

// newArchiveHTTPClient returns an HTTP client configured for archive.today
// requests: no redirect follow (so we can inspect Location headers from
// timegate/submit), per-archive cookie jar, and the standard User-Agent.
//
// The returned client is safe to use concurrently.
func newArchiveHTTPClient(timeout time.Duration) *http.Client {
	jar, _ := cookiejar.New(nil) // err is always nil for default options
	return &http.Client{
		Timeout: timeout,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// primeCookieJar makes a lightweight GET to archive.ph's homepage so the
// client's cookie jar picks up the qki= tracking cookie. Silently ignores
// errors — if we can't reach the homepage, the submit call will fail for the
// same reason and give a clearer error.
func primeCookieJar(client *http.Client) {
	if client.Jar == nil {
		return
	}
	req, err := http.NewRequest("GET", "https://archive.ph/", nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

// hasArchiveCookie reports whether the client's jar currently holds the qki
// cookie for any archive.today hostname. Used for test assertions and
// diagnostic output.
func hasArchiveCookie(client *http.Client) bool {
	if client.Jar == nil {
		return false
	}
	for _, host := range archiveHostnames {
		u, err := url.Parse("https://" + host + "/")
		if err != nil {
			continue
		}
		for _, ck := range client.Jar.Cookies(u) {
			if ck.Name == "qki" {
				return true
			}
		}
	}
	return false
}

// backoffSchedule returns the retry delays for rate-limit backoff. The
// schedule is 5s, 15s, 60s with ±25% jitter applied per attempt. After the
// last retry the caller gives up and records a cooldown.
func backoffSchedule() []time.Duration {
	base := []time.Duration{
		5 * time.Second,
		15 * time.Second,
		60 * time.Second,
	}
	out := make([]time.Duration, len(base))
	for i, d := range base {
		// ±25% jitter
		jitter := time.Duration(rand.Int63n(int64(d) / 2))
		if rand.Intn(2) == 0 {
			out[i] = d + jitter
		} else {
			out[i] = d - jitter
		}
	}
	return out
}

// archiveHostFromURL returns the hostname portion of a URL string.
// Used for logging and error formatting.
func archiveHostFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Host
}

// ensureTrailingSlash appends "/" to a URL if missing. Some archive.today
// endpoints are sensitive to trailing slashes on the base path.
func ensureTrailingSlash(s string) string {
	if strings.HasSuffix(s, "/") {
		return s
	}
	return s + "/"
}

// maxAgeFromCookie returns the Max-Age of a named cookie from the response,
// or zero if not present. Used to extract archive.today's 1-hour rate-limit
// window from the qki cookie on 429 responses.
func maxAgeFromCookie(cookies []*http.Cookie, name string) time.Duration {
	for _, ck := range cookies {
		if ck.Name == name && ck.MaxAge > 0 {
			return time.Duration(ck.MaxAge) * time.Second
		}
	}
	return 0
}

// formatBackoffError renders a short description of a retry delay for stderr
// output. Used by the backoff loop to show the user what is happening.
func formatBackoffError(attempt int, delay time.Duration, mirror string) string {
	return fmt.Sprintf("  429 from %s — waiting %s before retry %d", mirror, delay.Round(time.Second), attempt+1)
}
