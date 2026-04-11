// Interactive-mode helpers for archive-is-pp-cli.
//
// This file contains three things:
//   1. isInteractive() — decides whether the CLI should prompt the user
//   2. openInBrowser() — cross-platform browser launch helper
//   3. promptYesNo() — simple yes/no prompt with single-keystroke support
//
// The CLI never auto-opens a browser. Any browser launch must go through a
// prompt that the user explicitly approves. This was an explicit user
// correction during dogfooding: "offer to open in my browser" — never default.
//
// Unit 7 later extends promptYesNo into a richer multi-option menu
// (open / tl;dr / read-here / quit) via promptMenu() in menu.go.

package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/term"
)

// isInteractive reports whether the CLI is running in an interactive terminal
// context where showing a prompt makes sense. Returns false when any of the
// following is true:
//
//   - --json, --quiet, --agent, or --no-prompt flag is set
//   - stdin or stdout is not a TTY (piped, redirected, or daemonized)
//   - the process is running under CI (CI env var)
//
// All prompt- and browser-related features consult this helper so the rules
// for "should we prompt" match "should we open the browser" and so on.
func isInteractive(flags *rootFlags) bool {
	if flags == nil {
		return false
	}
	if flags.asJSON || flags.quiet || flags.agent || flags.noPrompt {
		return false
	}
	if os.Getenv("CI") != "" {
		return false
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	return true
}

// openInBrowser launches the given URL in the user's default browser. It does
// NOT wait for the browser process to exit. Errors from the OS command are
// returned so the caller can decide how to handle them (the CLI currently
// prints a stderr warning and continues).
//
// Platform mapping:
//   - macOS:   `open <url>`
//   - Linux:   `xdg-open <url>`
//   - Windows: `cmd /c start "" <url>`
//
// The empty title argument on Windows is intentional — `start` treats the first
// quoted argument as a window title, so without it a URL containing `&` gets
// misparsed as a new window title.
func openInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

// promptYesNo shows a simple yes/no prompt on the given writer (stderr in
// practice), reads a single keystroke from stdin, and returns true for yes,
// false for no/quit/cancel.
//
// Behavior:
//   - 'y' or 'Y'               → true
//   - 'n' or 'N'               → false
//   - 'q', 'Q', ESC, Ctrl-C   → false (explicit cancel)
//   - Enter (Return)          → defaultYes (accept the default)
//   - Any other key            → reprompt once, then default on second unknown key
//
// The prompt echoes the chosen letter on its own line so the user can see what
// they picked before the program continues.
//
// Raw terminal mode is used for single-keystroke input. If the raw-mode
// setup fails (unusual environment, Windows console edge cases), the function
// falls back to a line-mode prompt that requires Enter.
func promptYesNo(w io.Writer, question string, defaultYes bool) bool {
	suffix := " [y/N/q] "
	if defaultYes {
		suffix = " [Y/n/q] "
	}
	fmt.Fprint(w, question+suffix)

	// Try raw mode first for single-keystroke input.
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fall back to line-mode prompt. Print a hint so the user knows to press Enter.
		fmt.Fprint(w, "(press Enter) ")
		return readYesNoLineMode(w, defaultYes)
	}
	defer term.Restore(fd, oldState)

	buf := make([]byte, 1)
	for attempt := 0; attempt < 2; attempt++ {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			fmt.Fprintln(w)
			return defaultYes
		}
		switch buf[0] {
		case 'y', 'Y':
			fmt.Fprintln(w, "y")
			return true
		case 'n', 'N':
			fmt.Fprintln(w, "n")
			return false
		case 'q', 'Q':
			fmt.Fprintln(w, "q")
			return false
		case '\r', '\n': // Enter
			if defaultYes {
				fmt.Fprintln(w, "y")
				return true
			}
			fmt.Fprintln(w, "n")
			return false
		case 0x1b: // ESC
			fmt.Fprintln(w, "esc")
			return false
		case 0x03: // Ctrl-C
			fmt.Fprintln(w, "^C")
			// Restore terminal immediately so the parent shell doesn't get stuck
			term.Restore(fd, oldState)
			os.Exit(130)
		}
		// Unknown key — reprompt once.
		if attempt == 0 {
			fmt.Fprintf(w, "\r%s%s", question, suffix)
		}
	}
	// Two unknown keys in a row — take the default.
	fmt.Fprintln(w)
	return defaultYes
}

// readYesNoLineMode is the fallback path when raw mode cannot be established.
// Reads a full line from stdin and interprets the first character.
func readYesNoLineMode(w io.Writer, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return defaultYes
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultYes
	}
	switch strings.ToLower(line)[0] {
	case 'y':
		return true
	case 'n', 'q':
		return false
	}
	return defaultYes
}
