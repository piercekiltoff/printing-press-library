// Copyright 2026 dstevens. Licensed under Apache-2.0. See LICENSE.

package granola

import (
	"os"
	"testing"
)

// TestLoadCache_RealFile loads the actual Granola cache when present. We
// don't pin counts here because the live cache grows; we just assert the
// invariants: load succeeds, doc count > 0, version > 0.
func TestLoadCache_RealFile(t *testing.T) {
	path := DefaultCachePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("no live cache at " + path)
	}
	c, err := LoadCache(path)
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}
	if c.Version <= 0 {
		t.Errorf("expected version > 0, got %d", c.Version)
	}
	if len(c.Documents) == 0 {
		t.Errorf("expected documents > 0, got 0")
	}
	t.Logf("loaded cache v%d: %d documents, %d transcripts, %d folders, %d panels, %d recipes",
		c.Version, len(c.Documents), len(c.Transcripts), len(c.DocumentListsMetadata), len(c.PanelTemplates), len(c.RecipesAll()))
}

// TestLoadCache_Synthetic tests the v3 unwrap path with a hand-built blob.
func TestLoadCache_Synthetic(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/cache.json"
	// v6-shaped (dict).
	if err := os.WriteFile(path, []byte(`{"cache":{"version":6,"state":{"documents":{"a":{"id":"a","title":"T"}}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadCache(path)
	if err != nil {
		t.Fatalf("LoadCache v6: %v", err)
	}
	if c.DocumentByID("a") == nil || c.DocumentByID("a").Title != "T" {
		t.Errorf("missing doc 'a'")
	}

	// v3-shaped (stringified).
	if err := os.WriteFile(path, []byte(`{"cache":"{\"version\":3,\"state\":{\"documents\":{\"b\":{\"id\":\"b\",\"title\":\"U\"}}}}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err = LoadCache(path)
	if err != nil {
		t.Fatalf("LoadCache v3: %v", err)
	}
	if c.DocumentByID("b") == nil || c.DocumentByID("b").Title != "U" {
		t.Errorf("missing doc 'b'")
	}
}
