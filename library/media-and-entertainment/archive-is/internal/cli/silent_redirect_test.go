package cli

import (
	"strings"
	"testing"
)

func TestOriginalPathFromMemento(t *testing.T) {
	cases := []struct {
		mem  string
		want string
	}{
		{"https://archive.md/20260410174409/https://www.nytimes.com/2026/04/10/article", "/2026/04/10/article"},
		{"https://archive.md/20260410174409/https://www.nytimes.com/", "/"},
		{"https://archive.md/20260410174409/https://www.nytimes.com", ""},
		{"https://archive.md/notaTS/https://www.nytimes.com/", ""},
		{"https://archive.md/", ""},
		{"not a url", ""},
	}
	for _, tc := range cases {
		got := originalPathFromMemento(tc.mem)
		if got != tc.want {
			t.Errorf("originalPathFromMemento(%q) = %q, want %q", tc.mem, got, tc.want)
		}
	}
}

func TestIsRootOrNearRoot(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"", true},
		{"/business", true},
		{"/business/", true},
		{"/2026/04/10/article", false},
		{"/articles/foo/bar", false},
	}
	for _, tc := range cases {
		got := isRootOrNearRoot(tc.path)
		if got != tc.want {
			t.Errorf("isRootOrNearRoot(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestDetectSilentRedirect_Homepage(t *testing.T) {
	origURL := "https://www.nytimes.com/2026/04/10/business/iran-oil.html"
	// Memento URL that stores the homepage instead of the article
	mementoURL := "https://archive.md/20260410221519/https://www.nytimes.com/"
	warning := detectSilentRedirect(origURL, mementoURL)
	if warning == "" {
		t.Error("expected warning for homepage silent redirect")
	}
	if !strings.Contains(warning, "bot wall") {
		t.Errorf("warning should mention bot wall: %q", warning)
	}
	if !strings.Contains(warning, "--force") {
		t.Errorf("warning should suggest --force: %q", warning)
	}
}

func TestDetectSilentRedirect_Faithful(t *testing.T) {
	origURL := "https://www.nytimes.com/2026/04/10/business/iran-oil.html"
	// Faithful capture — stored path matches requested path
	mementoURL := "https://archive.md/20260410221519/https://www.nytimes.com/2026/04/10/business/iran-oil.html"
	warning := detectSilentRedirect(origURL, mementoURL)
	if warning != "" {
		t.Errorf("expected no warning for faithful capture, got: %q", warning)
	}
}

func TestDetectSilentRedirect_RootRequest(t *testing.T) {
	// User requested the root — no silent redirect possible
	origURL := "https://www.nytimes.com/"
	mementoURL := "https://archive.md/20260410221519/https://www.nytimes.com/"
	warning := detectSilentRedirect(origURL, mementoURL)
	if warning != "" {
		t.Errorf("expected no warning for root request, got: %q", warning)
	}
}
