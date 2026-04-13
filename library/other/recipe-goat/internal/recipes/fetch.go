package recipes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrBlocked indicates the target site actively refused the request (402/403
// or a Cloudflare-style gate). Callers typically recover by surfacing the
// site-level failure and continuing to other sources.
var ErrBlocked = errors.New("site blocked")

// chromeUA is a current-ish Chrome user agent. Many recipe sites 403 on Go's
// default UA, so sending a browser-looking UA is required for reach.
const chromeUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

// Fetch downloads recipeURL (honoring ctx / redirects) and extracts the
// JSON-LD Recipe from the response body. A 10s ctx deadline is applied if
// the caller didn't set one. Returns ErrBlocked for 402/403 and
// ErrNoJSONLD when the page has no Recipe node.
func Fetch(ctx context.Context, client *http.Client, recipeURL string) (*Recipe, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", recipeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("User-Agent", chromeUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", recipeURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusPaymentRequired || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: %s returned HTTP %d", ErrBlocked, recipeURL, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, recipeURL)
	}

	// Hard cap response body to 5 MiB — recipe pages are never larger in practice
	// and we don't want a misbehaving server to exhaust memory.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	r, err := ParseJSONLD(body, recipeURL)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// FetchHTML is a lower-level helper that returns raw HTML. Used by the search
// scrapers and the 'trending' command.
func FetchHTML(ctx context.Context, client *http.Client, target string) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", chromeUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusPaymentRequired || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w: %s returned HTTP %d", ErrBlocked, target, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, target)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}
