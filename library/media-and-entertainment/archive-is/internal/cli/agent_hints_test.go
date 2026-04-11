package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestAgentHintsFor_FullMemento(t *testing.T) {
	m := &memento{
		OriginalURL: "https://www.nytimes.com/2026/04/11/article",
		MementoURL:  "https://archive.md/20260411/https://www.nytimes.com/2026/04/11/article",
	}
	hints := agentHintsFor(m)
	if len(hints) != 4 {
		t.Fatalf("expected 4 hints, got %d", len(hints))
	}
	wantActions := []string{"open_browser", "summarize", "extract_text", "list_history"}
	for i, want := range wantActions {
		if hints[i].Action != want {
			t.Errorf("hint %d action = %q, want %q", i, hints[i].Action, want)
		}
	}
	if !strings.Contains(hints[0].Command, "archive.md") {
		t.Errorf("open command should contain memento URL: %q", hints[0].Command)
	}
	if !strings.Contains(hints[1].Command, "nytimes.com") {
		t.Errorf("tldr command should contain original URL: %q", hints[1].Command)
	}
	if !strings.Contains(hints[1].Command, "archive-is-pp-cli tldr") {
		t.Errorf("tldr command should invoke tldr: %q", hints[1].Command)
	}
}

func TestAgentHintsFor_EmptyOriginal_FallsBackToMemento(t *testing.T) {
	m := &memento{
		MementoURL: "https://archive.md/20260411/https://example.com/",
	}
	hints := agentHintsFor(m)
	if len(hints) != 4 {
		t.Fatalf("expected 4 hints, got %d", len(hints))
	}
	// tldr should still have a runnable command, pointing at the memento URL
	if !strings.Contains(hints[1].Command, "archive.md") {
		t.Errorf("expected fallback to memento URL in tldr command: %q", hints[1].Command)
	}
}

func TestAgentHintsFor_EmptyMemento_ReturnsNil(t *testing.T) {
	m := &memento{OriginalURL: "https://example.com/"}
	hints := agentHintsFor(m)
	if hints != nil {
		t.Errorf("expected nil hints for memento with no MementoURL, got %d", len(hints))
	}
}

func TestAgentHintsFor_NilMemento(t *testing.T) {
	if agentHintsFor(nil) != nil {
		t.Error("expected nil hints for nil memento")
	}
}

func TestWriteAgentHints_Format(t *testing.T) {
	m := &memento{
		OriginalURL: "https://example.com/article",
		MementoURL:  "https://archive.md/20260411/https://example.com/article",
	}
	hints := agentHintsFor(m)
	var buf bytes.Buffer
	writeAgentHints(&buf, hints)
	out := buf.String()
	// Should have exactly 4 NEXT: lines
	nextCount := strings.Count(out, "NEXT:")
	if nextCount != 4 {
		t.Errorf("expected 4 NEXT: lines, got %d: %q", nextCount, out)
	}
	// Each line should have a tab separator
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "NEXT:") && !strings.Contains(line, "\t") {
			t.Errorf("NEXT: line missing tab separator: %q", line)
		}
	}
}

func TestWriteAgentHints_Empty(t *testing.T) {
	var buf bytes.Buffer
	writeAgentHints(&buf, nil)
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty hints, got %q", buf.String())
	}
}

func TestPlatformOpenCommand(t *testing.T) {
	// Can't override runtime.GOOS cleanly, but we can assert the command contains the URL.
	cmd := platformOpenCommand("https://example.com/article")
	if !strings.Contains(cmd, "example.com") {
		t.Errorf("expected open command to contain URL: %q", cmd)
	}
}
