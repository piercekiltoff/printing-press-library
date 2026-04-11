// Detection for archive.today snapshots that are silently served as
// redirects to the site homepage (bot-wall captures).
//
// The NYT + DataDome case: someone archived a specific article URL, but
// NYT's DataDome returned an HTTP 403 "Not allowed" page that redirected
// browsers back to the nytimes.com homepage. Archive.today stored that
// redirect chain keyed to the article URL. When the user clicks the memento
// URL, they land on the homepage snapshot instead of the article. The
// ritualized behavior confuses users — they asked for an article and got a
// homepage.
//
// This file adds a post-lookup HEAD check. After `read` returns a memento URL
// via timegate, we HEAD the memento URL, follow redirects, and compare the
// final path to the originally requested path. If the final path is
// significantly shorter or matches the site root, we warn the user.
//
// Heuristic: the simplest signal that works.
//   - Final path is "/" or ""           → bot wall (redirected to root)
//   - Final path is shorter than original (fewer /-separated segments)
//     AND the first segment matches → bot wall (redirected to a parent)
//   - Otherwise → no warning
//
// Unit 8 of the polish plan.

package cli

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// detectSilentRedirect inspects a memento URL to see if archive.today is
// silently serving it as a redirect to a shorter path (typically the
// homepage), which usually indicates the original capture hit a bot wall.
//
// Returns a non-empty warning string when the redirect is detected.
// Returns empty string when the snapshot looks like a faithful capture or
// when the check itself fails (network, timeout).
//
// The check is a HEAD request with redirect-following enabled. Latency adds
// up to ~500ms to the overall `read` flow, which is acceptable for the UX
// signal it provides.
func detectSilentRedirect(origURL, mementoURL string) string {
	// Extract the path from the memento URL. Memento URLs look like
	//   https://archive.md/20260410174409/https://www.nytimes.com/2026/04/10/example
	// so we need to parse past the timestamp segment.
	origPath := originalPathFromMemento(mementoURL)
	if origPath == "" {
		// Unknown format — can't compare.
		return ""
	}

	requestedPath := pathFromURL(origURL)
	if requestedPath == "" || requestedPath == "/" {
		// User requested the root — no silent redirect is possible.
		return ""
	}

	// If the memento URL's embedded path matches the requested path, archive
	// stored exactly what was requested. Good.
	if normalizePath(origPath) == normalizePath(requestedPath) {
		return ""
	}

	// The memento URL itself has a different path than requested. Could be
	// two things:
	//   (a) User went through timegate and archive.today has a snapshot of a
	//       different URL that it thinks is a good match. In that case the
	//       snapshot is whatever it is — not necessarily a bot wall.
	//   (b) Someone captured the specific URL but DataDome redirected to
	//       homepage, so the stored path is shorter.
	// Distinguishing is hard without fetching. We use the shorter-path
	// heuristic as the strongest signal: if the stored path is the root or
	// is significantly shorter than requested, warn.
	if isRootOrNearRoot(origPath) && !isRootOrNearRoot(requestedPath) {
		return silentRedirectWarning(origURL, origPath)
	}

	// Also follow HEAD to see if archive.today's resolver does a runtime
	// redirect. If the resolved URL is shorter than the memento URL's
	// embedded path, warn.
	finalPath := headFollowRedirect(mementoURL, 10*time.Second)
	if finalPath != "" && finalPath != origPath && isRootOrNearRoot(finalPath) && !isRootOrNearRoot(requestedPath) {
		return silentRedirectWarning(origURL, finalPath)
	}
	return ""
}

// originalPathFromMemento extracts the original URL's path from a memento
// URL. Returns empty string if the URL is not in memento format.
//
// Memento format: https://archive.md/<14-digit-timestamp>/<original-url>
// Example: https://archive.md/20260410174409/https://www.nytimes.com/2026/04/10/article
// Returns: /2026/04/10/article
func originalPathFromMemento(mementoURL string) string {
	u, err := url.Parse(mementoURL)
	if err != nil {
		return ""
	}
	// Path looks like /20260410174409/https://www.nytimes.com/... — we need to
	// skip the timestamp segment and re-parse the rest as a URL.
	parts := strings.SplitN(strings.TrimPrefix(u.Path, "/"), "/", 2)
	if len(parts) < 2 {
		return ""
	}
	// First part should be a 14-digit timestamp
	if len(parts[0]) != 14 {
		return ""
	}
	remainder := parts[1]
	// The remainder starts with http: or https: — parse it as a URL
	inner, err := url.Parse(remainder)
	if err != nil {
		return ""
	}
	return inner.Path
}

// pathFromURL returns the path component of a URL string, or empty string on
// parse failure.
func pathFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Path
}

// normalizePath strips trailing slashes and lowercases the path for
// comparison. "/articles/foo/" and "/articles/foo" are equivalent.
func normalizePath(p string) string {
	p = strings.ToLower(p)
	p = strings.TrimRight(p, "/")
	if p == "" {
		return "/"
	}
	return p
}

// isRootOrNearRoot reports whether a path is the site root or has only one
// segment. Homepages like "/" and section pages like "/business" are both
// "near root" signals for the silent-redirect heuristic.
func isRootOrNearRoot(p string) bool {
	p = normalizePath(p)
	if p == "" || p == "/" {
		return true
	}
	segments := strings.Split(strings.Trim(p, "/"), "/")
	return len(segments) <= 1
}

// headFollowRedirect sends a HEAD request to the given URL with redirects
// enabled and returns the path of the final URL. Returns empty string on
// error. The client explicitly enables redirect following (the default
// in newArchiveHTTPClient does NOT follow — we override here).
func headFollowRedirect(rawURL string, timeout time.Duration) string {
	client := &http.Client{
		Timeout: timeout,
		// Follow redirects (default behavior, explicit for clarity)
	}
	req, err := http.NewRequest("HEAD", rawURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	// resp.Request.URL has the final URL after redirects
	if resp.Request == nil || resp.Request.URL == nil {
		return ""
	}
	// Extract the path from the archive-side memento URL (not the original
	// URL embedded inside). We want to know "where did archive.today resolve
	// us to", not "what's the embedded path".
	finalMementoURL := resp.Request.URL.String()
	return originalPathFromMemento(finalMementoURL)
}

// silentRedirectWarning returns the stderr warning text for a detected
// silent redirect.
func silentRedirectWarning(requestedURL, observedPath string) string {
	return fmt.Sprintf("\nNote: archive.today has a snapshot for this URL, but it's being served as a redirect\n"+
		"to a different path (%q). This usually means the original capture hit a bot\n"+
		"wall (e.g., DataDome) and stored an error page instead of the article.\n\n"+
		"For a fresh capture that strips JavaScript before the bot wall fires, try:\n"+
		"  archive-is-pp-cli save %s --force\n\n",
		observedPath, requestedURL)
}
