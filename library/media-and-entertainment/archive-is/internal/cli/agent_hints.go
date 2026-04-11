// Agent hints: machine-readable next-step suggestions for non-interactive CLI callers.
//
// When archive-is-pp-cli runs inside an agent loop (Claude Code calling it via
// a Bash tool), the interactive terminal prompt never fires — the calling
// agent sees only bare stdout and has to guess what to do with the result.
//
// This file adds a short list of suggested next-step commands that the CLI
// emits on stderr (or includes in --json payloads) when running in
// non-interactive mode. The interactive human flow is untouched.
//
// Format on stderr (tab-separated for easy parsing):
//
//	NEXT: open in browser          open "<memento_url>"
//	NEXT: summarize with LLM       archive-is-pp-cli tldr "<orig_url>"
//	NEXT: get article text         archive-is-pp-cli get "<orig_url>"
//	NEXT: list historical snapshots archive-is-pp-cli history "<orig_url>"
//
// Unit 1 of the agent-hints plan.

package cli

import (
	"fmt"
	"io"
	"runtime"
)

// hintAction is one suggested next-step command the CLI wants the caller to
// know about. Used both for stderr rendering and for the JSON next_actions
// field in Unit 3.
type hintAction struct {
	Action      string `json:"action"`      // machine-readable tag (open_browser, summarize, etc.)
	Command     string `json:"command"`     // full shell command line to run
	Description string `json:"description"` // short human-readable label
}

// agentHintsFor returns the standard set of next-step actions for a memento
// result. Always returns the same four actions in the same order: open, tldr,
// get, history. Returns nil if the memento has no memento URL (nothing to
// act on).
//
// The open command is built from runtime.GOOS. The tldr/get/history commands
// use the memento's OriginalURL so the caller targets the source, not the
// archive. If OriginalURL is empty, falls back to MementoURL so the hints
// are still runnable (just less ideal).
func agentHintsFor(m *memento) []hintAction {
	if m == nil || m.MementoURL == "" {
		return nil
	}

	source := m.OriginalURL
	if source == "" {
		source = m.MementoURL
	}

	return []hintAction{
		{
			Action:      "open_browser",
			Command:     platformOpenCommand(m.MementoURL),
			Description: "open in browser",
		},
		{
			Action:      "summarize",
			Command:     fmt.Sprintf(`archive-is-pp-cli tldr %q`, source),
			Description: "summarize with LLM",
		},
		{
			Action:      "extract_text",
			Command:     fmt.Sprintf(`archive-is-pp-cli get %q`, source),
			Description: "get article text",
		},
		{
			Action:      "list_history",
			Command:     fmt.Sprintf(`archive-is-pp-cli history %q`, source),
			Description: "list historical snapshots",
		},
	}
}

// platformOpenCommand returns the shell command the caller should run to open
// the given URL in the OS default browser. Mirrors the dispatch in
// openInBrowser() in interactive.go, but returns the command string instead of
// launching it (the agent is the one who will run it).
func platformOpenCommand(url string) string {
	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf(`open %q`, url)
	case "linux":
		return fmt.Sprintf(`xdg-open %q`, url)
	case "windows":
		return fmt.Sprintf(`cmd /c start "" %q`, url)
	default:
		return fmt.Sprintf(`open %q`, url) // best-effort
	}
}

// writeAgentHints renders the hint list as tab-separated NEXT: lines on the
// given writer. Prefixed with a leading blank line for visual separation from
// the result output.
//
// Output format per line:
//
//	NEXT: <description><tab><command>
//
// The NEXT: prefix is greppable for agents that want to filter stderr. The
// tab separator lets callers split fields cleanly.
func writeAgentHints(w io.Writer, actions []hintAction) {
	if len(actions) == 0 {
		return
	}
	fmt.Fprintln(w)
	for _, a := range actions {
		fmt.Fprintf(w, "NEXT: %-26s\t%s\n", a.Description, a.Command)
	}
}
