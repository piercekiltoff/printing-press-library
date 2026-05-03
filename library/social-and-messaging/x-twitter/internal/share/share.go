// Package share provides export/import of the local x-twitter store as a
// portable JSONL bundle. Useful for syncing state across machines without
// re-running expensive API sync.
package share

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Bundle is the wire format for share files.
type Bundle struct {
	Version  int              `json:"version"`
	Resource string           `json:"resource"`
	Rows     []map[string]any `json:"rows"`
}

// Export writes a Bundle to outputDir as <resource>.share.jsonl.
func Export(outputDir, resource string, rows []map[string]any) (string, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("creating share dir: %w", err)
	}
	path := filepath.Join(outputDir, resource+".share.jsonl")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	header := map[string]any{"_share_version": 1, "_resource": resource}
	if err := enc.Encode(header); err != nil {
		return "", err
	}
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return "", err
		}
	}
	return path, nil
}

// Import reads a share bundle from path and returns the rows.
func Import(path string) (string, []map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var header map[string]any
	if err := dec.Decode(&header); err != nil {
		return "", nil, fmt.Errorf("reading share header: %w", err)
	}
	resource, _ := header["_resource"].(string)
	var rows []map[string]any
	for dec.More() {
		var row map[string]any
		if err := dec.Decode(&row); err != nil {
			return "", nil, err
		}
		rows = append(rows, row)
	}
	return resource, rows, nil
}
