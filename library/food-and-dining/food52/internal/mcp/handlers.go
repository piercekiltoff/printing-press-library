// Copyright 2026 trevin-chow. Licensed under Apache-2.0. See LICENSE.

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/food-and-dining/food52/internal/client"
	"github.com/mvanhorn/printing-press-library/library/food-and-dining/food52/internal/config"
	"github.com/mvanhorn/printing-press-library/library/food-and-dining/food52/internal/store"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// makeAPIHandler creates a generic MCP tool handler for an API endpoint.
func makeAPIHandler(method, pathTemplate string, positionalParams []string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		c, err := newMCPClient()
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}

		// mcp-go v0.47+ made CallToolParams.Arguments an `any` to support
		// non-map payloads; GetArguments() returns the map[string]any shape
		// we rely on here (or an empty map when the payload is something else).
		args := req.GetArguments()

		// Build path by substituting positional params
		path := pathTemplate
		for _, p := range positionalParams {
			if v, ok := args[p]; ok {
				path = strings.Replace(path, "{"+p+"}", fmt.Sprintf("%v", v), 1)
			}
		}

		// Collect non-positional params as query params
		params := make(map[string]string)
		for k, v := range args {
			isPositional := false
			for _, p := range positionalParams {
				if k == p {
					isPositional = true
					break
				}
			}
			if !isPositional {
				params[k] = fmt.Sprintf("%v", v)
			}
		}

		var data json.RawMessage
		switch method {
		case "GET":
			data, err = c.Get(path, params)
		case "POST":
			body, _ := json.Marshal(args)
			data, _, err = c.Post(path, body)
		case "PUT":
			body, _ := json.Marshal(args)
			data, _, err = c.Put(path, body)
		case "PATCH":
			body, _ := json.Marshal(args)
			data, _, err = c.Patch(path, body)
		case "DELETE":
			data, _, err = c.Delete(path)
		default:
			return mcplib.NewToolResultError("unsupported method: " + method), nil
		}

		if err != nil {
			msg := err.Error()
			switch {
			case strings.Contains(msg, "HTTP 409"):
				return mcplib.NewToolResultText("already exists (no-op)"), nil
			case strings.Contains(msg, "HTTP 401"):
				return mcplib.NewToolResultError("authentication failed: " + msg +
					"\nhint: check your API credentials." +
					"\n      Run 'food52-pp-cli doctor' to check auth status."), nil
			case strings.Contains(msg, "HTTP 403"):
				return mcplib.NewToolResultError("permission denied: " + msg +
					"\nhint: your credentials are valid but lack access to this resource." +
					"\n      Run 'food52-pp-cli doctor' to check auth status."), nil
			case strings.Contains(msg, "HTTP 404"):
				if method == "DELETE" {
					return mcplib.NewToolResultText("already deleted (no-op)"), nil
				}
				return mcplib.NewToolResultError("not found: " + msg), nil
			case strings.Contains(msg, "HTTP 429"):
				return mcplib.NewToolResultError("rate limited: " + msg), nil
			default:
				return mcplib.NewToolResultError(msg), nil
			}
		}

		// For GET responses, wrap bare arrays with count metadata
		if method == "GET" {
			trimmed := strings.TrimSpace(string(data))
			if len(trimmed) > 0 && trimmed[0] == '[' {
				var items []json.RawMessage
				if json.Unmarshal(data, &items) == nil {
					wrapped := map[string]any{
						"count": len(items),
						"items": items,
					}
					out, _ := json.Marshal(wrapped)
					return mcplib.NewToolResultText(string(out)), nil
				}
			}
		}
		return mcplib.NewToolResultText(string(data)), nil
	}
}

func newMCPClient() (*client.Client, error) {
	home, _ := os.UserHomeDir()
	cfgPath := filepath.Join(home, ".config", "food52-pp-cli", "config.toml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return client.New(cfg, 30*time.Second, 2), nil
}

func dbPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "food52-pp-cli", "data.db")
}

// Note: MCP tools use their own dbPath() because they are in a separate package (main, not cli).
// The CLI's defaultDBPath() in the cli package uses the same canonical path.

func handleSync(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	return mcplib.NewToolResultText("sync not yet implemented via MCP - use the CLI: food52-pp-cli sync"), nil
}

func handleSQL(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	args := req.GetArguments()
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return mcplib.NewToolResultError("query is required"), nil
	}

	// Block write operations
	upper := strings.ToUpper(strings.TrimSpace(query))
	for _, prefix := range []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE"} {
		if strings.HasPrefix(upper, prefix) {
			return mcplib.NewToolResultError("only SELECT queries are allowed"), nil
		}
	}

	db, err := store.Open(dbPath())
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("opening database: %v", err)), nil
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return mcplib.NewToolResultError(fmt.Sprintf("query failed: %v", err)), nil
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []map[string]any
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		rows.Scan(ptrs...)
		row := make(map[string]any)
		for i, col := range cols {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	data, _ := json.MarshalIndent(results, "", "  ")
	return mcplib.NewToolResultText(string(data)), nil
}

func handleContext(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	ctx := map[string]any{
		"api":        "food52",
		"archetype":  "content",
		"tool_count": 7,
		"resources": []map[string]any{
			{"name": "articles", "endpoints": []string{"browse", "get"}, "searchable": true},
			{"name": "recipes", "endpoints": []string{"browse", "get"}, "searchable": true},
		},
		"query_tips": []string{
			"Use sync first; then prefer the sql or search tools over re-hitting the API.",
			"Pagination is cursor-based via the after parameter; tune page size with limit.",
		},
		"unique_capabilities": []map[string]string{
			{"command": "pantry match", "use_when": "what can I make with what I have"},
			{"command": "search", "use_when": "offline FTS across synced recipes and articles"},
			{"command": "sync recipes", "use_when": "seed the local store with one or more tags"},
			{"command": "recipes top", "use_when": "Test-Kitchen approved + rating-floored browse"},
			{"command": "scale", "use_when": "resize ingredients via JSON-LD recipeYield"},
			{"command": "print", "use_when": "clean ingredients + numbered steps for cooking"},
			{"command": "articles for-recipe", "use_when": "editorial context for a recipe"},
		},
	}
	data, _ := json.MarshalIndent(ctx, "", "  ")
	return mcplib.NewToolResultText(string(data)), nil
}
