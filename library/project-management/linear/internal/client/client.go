// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package client

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/project-management/linear/internal/config"
)

type Client struct {
	BaseURL    string
	Config     *config.Config
	HTTPClient *http.Client
	DryRun     bool
	NoCache    bool
	cacheDir   string
}

// APIError carries HTTP status information for structured exit codes.
type APIError struct {
	Method     string
	Path       string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s %s returned HTTP %d: %s", e.Method, e.Path, e.StatusCode, e.Body)
}

// GraphQLError represents an error returned by the GraphQL API.
type GraphQLError struct {
	Message    string
	Extensions map[string]any
}

func (e *GraphQLError) Error() string {
	return fmt.Sprintf("GraphQL error: %s", e.Message)
}

func New(cfg *config.Config, timeout time.Duration) *Client {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".cache", "linear-pp-cli")
	return &Client{
		BaseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		Config:     cfg,
		HTTPClient: &http.Client{Timeout: timeout},
		cacheDir:   cacheDir,
	}
}

// graphqlRequest is the JSON body sent to the GraphQL endpoint.
type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphqlResponse is the JSON envelope returned by the GraphQL endpoint.
type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message    string         `json:"message"`
		Extensions map[string]any `json:"extensions,omitempty"`
	} `json:"errors,omitempty"`
}

// GraphQL sends a GraphQL query or mutation to the Linear API and returns the "data" field.
// Queries (not mutations) are cached unless NoCache is set.
func (c *Client) GraphQL(query string, variables map[string]any) (json.RawMessage, error) {
	isMutation := isMutation(query)

	// Check cache for queries (not mutations)
	if !isMutation && !c.NoCache && c.cacheDir != "" {
		if cached, ok := c.readGraphQLCache(query, variables); ok {
			return cached, nil
		}
	}

	reqBody := graphqlRequest{
		Query:     query,
		Variables: variables,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling GraphQL request: %w", err)
	}

	// Dry run mode
	if c.DryRun {
		return c.graphqlDryRun(query, variables)
	}

	endpoint := c.BaseURL + "/graphql"

	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(bodyBytes)))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		authHeader, err := c.authHeader()
		if err != nil {
			return nil, err
		}
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "linear-pp-cli/1.0.0")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("POST /graphql: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		// Rate limited - wait and retry
		if resp.StatusCode == 429 && attempt < maxRetries {
			wait := retryAfter(resp)
			fmt.Fprintf(os.Stderr, "rate limited, waiting %s (attempt %d/%d)\n", wait, attempt+1, maxRetries)
			time.Sleep(wait)
			lastErr = &APIError{Method: "POST", Path: "/graphql", StatusCode: 429, Body: truncateBody(respBody)}
			continue
		}

		// Server error - retry with backoff
		if resp.StatusCode >= 500 && attempt < maxRetries {
			wait := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			fmt.Fprintf(os.Stderr, "server error %d, retrying in %s (attempt %d/%d)\n", resp.StatusCode, wait, attempt+1, maxRetries)
			time.Sleep(wait)
			lastErr = &APIError{Method: "POST", Path: "/graphql", StatusCode: resp.StatusCode, Body: truncateBody(respBody)}
			continue
		}

		// Non-200 HTTP error
		if resp.StatusCode >= 400 {
			return nil, &APIError{
				Method:     "POST",
				Path:       "/graphql",
				StatusCode: resp.StatusCode,
				Body:       truncateBody(respBody),
			}
		}

		// Parse the GraphQL response envelope
		var gqlResp graphqlResponse
		if err := json.Unmarshal(respBody, &gqlResp); err != nil {
			return nil, fmt.Errorf("parsing GraphQL response: %w", err)
		}

		// If there are errors and data is null, return an error
		if len(gqlResp.Errors) > 0 {
			dataIsNull := len(gqlResp.Data) == 0 || string(gqlResp.Data) == "null"
			if dataIsNull {
				return nil, &GraphQLError{
					Message:    gqlResp.Errors[0].Message,
					Extensions: gqlResp.Errors[0].Extensions,
				}
			}
			// If data is present alongside errors, log warnings but return data
			for _, e := range gqlResp.Errors {
				fmt.Fprintf(os.Stderr, "GraphQL warning: %s\n", e.Message)
			}
		}

		// Cache query results (not mutations)
		if !isMutation && !c.NoCache && c.cacheDir != "" {
			c.writeGraphQLCache(query, variables, gqlResp.Data)
		}

		return gqlResp.Data, nil
	}

	return nil, lastErr
}

// GraphQLPaginated handles Linear's Relay-style cursor pagination.
// It fetches all pages by following pageInfo.endCursor and returns all nodes concatenated.
// dataPath is the top-level field name inside "data" (e.g., "issues" for data.issues).
func (c *Client) GraphQLPaginated(query string, variables map[string]any, dataPath string) ([]json.RawMessage, error) {
	if variables == nil {
		variables = make(map[string]any)
	}

	var allNodes []json.RawMessage

	for {
		data, err := c.GraphQL(query, variables)
		if err != nil {
			return nil, err
		}

		// Navigate into the data path: data.<dataPath>
		var dataMap map[string]json.RawMessage
		if err := json.Unmarshal(data, &dataMap); err != nil {
			return nil, fmt.Errorf("parsing data envelope: %w", err)
		}

		entityData, ok := dataMap[dataPath]
		if !ok {
			return nil, fmt.Errorf("data path %q not found in response", dataPath)
		}

		// Parse the entity's nodes and pageInfo
		var page struct {
			Nodes    []json.RawMessage `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		}
		if err := json.Unmarshal(entityData, &page); err != nil {
			return nil, fmt.Errorf("parsing paginated response for %q: %w", dataPath, err)
		}

		allNodes = append(allNodes, page.Nodes...)

		if !page.PageInfo.HasNextPage || page.PageInfo.EndCursor == "" {
			break
		}

		// Set the cursor for the next page
		variables["after"] = page.PageInfo.EndCursor
	}

	return allNodes, nil
}

// isMutation returns true if the query string starts with a mutation keyword.
func isMutation(query string) bool {
	trimmed := strings.TrimSpace(query)
	return strings.HasPrefix(trimmed, "mutation")
}

// graphqlDryRun prints the GraphQL query and variables to stderr without sending a request.
func (c *Client) graphqlDryRun(query string, variables map[string]any) (json.RawMessage, error) {
	fmt.Fprintf(os.Stderr, "POST %s/graphql\n", c.BaseURL)

	authHeader, err := c.authHeader()
	if err != nil {
		return nil, err
	}
	if authHeader != "" {
		auth := authHeader
		if len(auth) > 20 {
			auth = auth[:15] + "..."
		}
		fmt.Fprintf(os.Stderr, "  Authorization: %s\n", auth)
	}

	fmt.Fprintf(os.Stderr, "  Query:\n    %s\n", strings.ReplaceAll(strings.TrimSpace(query), "\n", "\n    "))

	if len(variables) > 0 {
		varsJSON, _ := json.MarshalIndent(variables, "    ", "  ")
		fmt.Fprintf(os.Stderr, "  Variables:\n    %s\n", string(varsJSON))
	}

	fmt.Fprintf(os.Stderr, "\n(dry run - no request sent)\n")
	return json.RawMessage(`{"dry_run": true}`), nil
}

// graphqlCacheKey generates a cache key from the query and variables.
func (c *Client) graphqlCacheKey(query string, variables map[string]any) string {
	key := query
	if len(variables) > 0 {
		varsJSON, _ := json.Marshal(variables)
		key += string(varsJSON)
	}
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:8])
}

// readGraphQLCache reads a cached GraphQL response if it exists and is fresh.
func (c *Client) readGraphQLCache(query string, variables map[string]any) (json.RawMessage, bool) {
	cacheFile := filepath.Join(c.cacheDir, c.graphqlCacheKey(query, variables)+".json")
	info, err := os.Stat(cacheFile)
	if err != nil || time.Since(info.ModTime()) > 5*time.Minute {
		return nil, false
	}
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false
	}
	return json.RawMessage(data), true
}

// writeGraphQLCache writes a GraphQL response to the cache.
func (c *Client) writeGraphQLCache(query string, variables map[string]any, data json.RawMessage) {
	os.MkdirAll(c.cacheDir, 0o755)
	cacheFile := filepath.Join(c.cacheDir, c.graphqlCacheKey(query, variables)+".json")
	os.WriteFile(cacheFile, []byte(data), 0o644)
}

// --- Backward-compatible REST methods (kept for compatibility) ---

func (c *Client) Get(path string, params map[string]string) (json.RawMessage, error) {
	if !c.NoCache && c.cacheDir != "" {
		if cached, ok := c.readCache(path, params); ok {
			return cached, nil
		}
	}
	result, _, err := c.do("GET", path, params, nil)
	if err == nil && !c.NoCache && c.cacheDir != "" {
		c.writeCache(path, params, result)
	}
	return result, err
}

func (c *Client) cacheKey(path string, params map[string]string) string {
	key := path
	for k, v := range params {
		key += k + "=" + v
	}
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:8])
}

func (c *Client) readCache(path string, params map[string]string) (json.RawMessage, bool) {
	cacheFile := filepath.Join(c.cacheDir, c.cacheKey(path, params)+".json")
	info, err := os.Stat(cacheFile)
	if err != nil || time.Since(info.ModTime()) > 5*time.Minute {
		return nil, false
	}
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, false
	}
	return json.RawMessage(data), true
}

func (c *Client) writeCache(path string, params map[string]string, data json.RawMessage) {
	os.MkdirAll(c.cacheDir, 0o755)
	cacheFile := filepath.Join(c.cacheDir, c.cacheKey(path, params)+".json")
	os.WriteFile(cacheFile, []byte(data), 0o644)
}

func (c *Client) Post(path string, body any) (json.RawMessage, int, error) {
	return c.do("POST", path, nil, body)
}

func (c *Client) Delete(path string) (json.RawMessage, int, error) {
	return c.do("DELETE", path, nil, nil)
}

func (c *Client) Put(path string, body any) (json.RawMessage, int, error) {
	return c.do("PUT", path, nil, body)
}

func (c *Client) Patch(path string, body any) (json.RawMessage, int, error) {
	return c.do("PATCH", path, nil, body)
}

func (c *Client) do(method, path string, params map[string]string, body any) (json.RawMessage, int, error) {
	url := c.BaseURL + path

	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling body: %w", err)
		}
		bodyBytes = b
	}

	if c.DryRun {
		return c.dryRun(method, url, params, bodyBytes)
	}

	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = strings.NewReader(string(bodyBytes))
		}

		req, err := http.NewRequest(method, url, bodyReader)
		if err != nil {
			return nil, 0, fmt.Errorf("creating request: %w", err)
		}

		authHeader, err := c.authHeader()
		if err != nil {
			return nil, 0, err
		}
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("User-Agent", "linear-pp-cli/1.0.0")

		if params != nil {
			q := req.URL.Query()
			for k, v := range params {
				if v != "" {
					q.Set(k, v)
				}
			}
			req.URL.RawQuery = q.Encode()
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%s %s: %w", method, path, err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, 0, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode < 400 {
			return json.RawMessage(respBody), resp.StatusCode, nil
		}

		apiErr := &APIError{
			Method:     method,
			Path:       path,
			StatusCode: resp.StatusCode,
			Body:       truncateBody(respBody),
		}

		if resp.StatusCode == 429 && attempt < maxRetries {
			wait := retryAfter(resp)
			fmt.Fprintf(os.Stderr, "rate limited, waiting %s (attempt %d/%d)\n", wait, attempt+1, maxRetries)
			time.Sleep(wait)
			lastErr = apiErr
			continue
		}

		if resp.StatusCode >= 500 && attempt < maxRetries {
			wait := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			fmt.Fprintf(os.Stderr, "server error %d, retrying in %s (attempt %d/%d)\n", resp.StatusCode, wait, attempt+1, maxRetries)
			time.Sleep(wait)
			lastErr = apiErr
			continue
		}

		return nil, resp.StatusCode, apiErr
	}

	return nil, 0, lastErr
}

func (c *Client) dryRun(method, url string, params map[string]string, body []byte) (json.RawMessage, int, error) {
	fmt.Fprintf(os.Stderr, "%s %s\n", method, url)
	if params != nil {
		for k, v := range params {
			if v != "" {
				fmt.Fprintf(os.Stderr, "  ?%s=%s\n", k, v)
			}
		}
	}
	authHeader, err := c.authHeader()
	if err != nil {
		return nil, 0, err
	}
	if authHeader != "" {
		auth := authHeader
		if len(auth) > 20 {
			auth = auth[:15] + "..."
		}
		fmt.Fprintf(os.Stderr, "  Authorization: %s\n", auth)
	}
	if body != nil {
		var pretty json.RawMessage
		if json.Unmarshal(body, &pretty) == nil {
			enc := json.NewEncoder(os.Stderr)
			enc.SetIndent("  ", "  ")
			fmt.Fprintf(os.Stderr, "  Body:\n")
			enc.Encode(pretty)
		}
	}
	fmt.Fprintf(os.Stderr, "\n(dry run - no request sent)\n")
	return json.RawMessage(`{"dry_run": true}`), 0, nil
}

func (c *Client) authHeader() (string, error) {
	if c.Config == nil {
		return "", nil
	}
	if c.Config.AccessToken != "" && !c.Config.TokenExpiry.IsZero() && time.Now().After(c.Config.TokenExpiry) && c.Config.RefreshToken != "" {
		if err := c.refreshAccessToken(); err != nil {
			return "", err
		}
	}
	return c.Config.AuthHeader(), nil
}

func (c *Client) refreshAccessToken() error {
	if c.Config == nil {
		return nil
	}
	if c.Config.RefreshToken == "" {
		return nil
	}

	tokenURL := ""
	if tokenURL == "" {
		return nil
	}

	params := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {c.Config.RefreshToken},
		"client_id":     {c.Config.ClientID},
	}
	if c.Config.ClientSecret != "" {
		params.Set("client_secret", c.Config.ClientSecret)
	}

	resp, err := c.HTTPClient.PostForm(tokenURL, params)
	if err != nil {
		return fmt.Errorf("refreshing access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refreshing access token: HTTP %d: %s", resp.StatusCode, truncateBody(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("parsing refresh response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return fmt.Errorf("refreshing access token: no access token in response")
	}

	refreshToken := c.Config.RefreshToken
	if tokenResp.RefreshToken != "" {
		refreshToken = tokenResp.RefreshToken
	}

	expiry := time.Time{}
	if tokenResp.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	if err := c.Config.SaveTokens(c.Config.ClientID, c.Config.ClientSecret, tokenResp.AccessToken, refreshToken, expiry); err != nil {
		return fmt.Errorf("saving refreshed token: %w", err)
	}

	return nil
}

func retryAfter(resp *http.Response) time.Duration {
	header := resp.Header.Get("Retry-After")
	if header == "" {
		return 5 * time.Second
	}
	if seconds, err := strconv.Atoi(header); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(header); err == nil {
		wait := time.Until(t)
		if wait > 0 {
			return wait
		}
	}
	return 5 * time.Second
}

func truncateBody(b []byte) string {
	s := string(b)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
