package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const algoliaBaseURL = "https://hn.algolia.com/api/v1"

// algoliaGet makes a direct HTTP GET to the Algolia HN API.
// path should start with "/" (e.g., "/search", "/search_by_date").
// params are added as query parameters.
func algoliaGet(path string, params map[string]string) (json.RawMessage, error) {
	u, err := url.Parse(algoliaBaseURL + path)
	if err != nil {
		return nil, fmt.Errorf("parsing algolia URL: %w", err)
	}
	q := u.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("algolia request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading algolia response: %w", err)
	}

	if resp.StatusCode >= 400 {
		errBody := string(body)
		if len(errBody) > 200 {
			errBody = errBody[:200] + "..."
		}
		return nil, fmt.Errorf("algolia HTTP %d: %s", resp.StatusCode, errBody)
	}

	return json.RawMessage(body), nil
}

// algoliaHits extracts the "hits" array from an Algolia search response.
func algoliaHits(data json.RawMessage) ([]map[string]any, error) {
	var envelope struct {
		Hits []map[string]any `json:"hits"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("parsing algolia hits: %w", err)
	}
	return envelope.Hits, nil
}

// parseDuration parses human-friendly durations like "1h", "24h", "7d", "30d".
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	re := regexp.MustCompile(`^(\d+)\s*(h|d|w|m)$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid duration %q: use format like 1h, 24h, 7d, 30d", s)
	}

	n, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid number in duration: %w", err)
	}

	switch matches[2] {
	case "h":
		return time.Duration(n) * time.Hour, nil
	case "d":
		return time.Duration(n) * 24 * time.Hour, nil
	case "w":
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	case "m":
		return time.Duration(n) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit %q in duration", matches[2])
	}
}

// sinceTimestamp computes the Unix timestamp for "now minus duration".
func sinceTimestamp(dur time.Duration) int64 {
	return time.Now().Add(-dur).Unix()
}

// timeAgo returns a human-friendly relative time string.
func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}

// extractDomain pulls the domain from a URL, stripping "www." prefix.
func extractDomain(rawURL string) string {
	if rawURL == "" {
		return "(self)"
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := u.Hostname()
	host = strings.TrimPrefix(host, "www.")
	return host
}

// velocity computes points per hour since creation.
func velocity(points int, createdAt time.Time) float64 {
	hours := time.Since(createdAt).Hours()
	if hours < 0.1 {
		hours = 0.1
	}
	return float64(points) / hours
}

// fetchFirebaseItem fetches a single item from the Firebase HN API.
func fetchFirebaseItem(flags *rootFlags, itemID int) (map[string]any, error) {
	c, err := flags.newClient()
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/item/%d.json", itemID)
	data, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}
	var item map[string]any
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	return item, nil
}

// fetchFirebaseItems fetches multiple items from Firebase concurrently.
// Returns items in order, skipping any that fail.
func fetchFirebaseItems(flags *rootFlags, ids []int, limit int) ([]map[string]any, error) {
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}

	type result struct {
		index int
		item  map[string]any
		err   error
	}

	results := make(chan result, len(ids))
	// Limit concurrency
	sem := make(chan struct{}, 10)

	for i, id := range ids {
		sem <- struct{}{}
		go func(idx, itemID int) {
			defer func() { <-sem }()
			item, err := fetchFirebaseItem(flags, itemID)
			results <- result{index: idx, item: item, err: err}
		}(i, id)
	}

	items := make([]map[string]any, len(ids))
	for range ids {
		r := <-results
		if r.err == nil && r.item != nil {
			items[r.index] = r.item
		}
	}

	// Filter out nil entries
	var out []map[string]any
	for _, item := range items {
		if item != nil {
			out = append(out, item)
		}
	}
	return out, nil
}

// getInt safely extracts an int from a map value (handles float64 from JSON).
func getInt(m map[string]any, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case json.Number:
		n, _ := val.Int64()
		return int(n)
	default:
		return 0
	}
}

// getString safely extracts a string from a map value.
func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// getFloat safely extracts a float64 from a map value.
func getFloat(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	f, _ := v.(float64)
	return f
}

// getIntSlice safely extracts an int slice from a map value (kids array).
func getIntSlice(m map[string]any, key string) []int {
	v, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []int
	for _, item := range arr {
		if f, ok := item.(float64); ok {
			out = append(out, int(f))
		}
	}
	return out
}
