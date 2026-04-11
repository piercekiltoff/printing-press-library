// Multi-option post-action menu for archive-is-pp-cli.
//
// Extends unit 5's promptYesNo with a richer menu after read/save success:
//
//   Open in browser? [o/t/r/q]
//     o  open in browser (default)
//     t  tl;dr — summarize with Claude
//     r  read full text here
//     q  quit
//
// Single-keystroke input via raw terminal mode. Falls back to line-mode with
// Enter if raw mode cannot be established. Respects isInteractive() —
// non-interactive contexts never see the menu.
//
// Unit 7 of the polish plan.

package cli

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// menuAction represents a user selection from the post-action menu.
type menuAction int

const (
	menuActionNone menuAction = iota
	menuActionOpen
	menuActionTldr
	menuActionReadHere
	menuActionQuit
)

// promptMenu renders the multi-option menu and reads a single keystroke.
// Returns the selected action. Defaults to Open on Enter.
//
// Keymap:
//   - 'o', Enter   → open in browser
//   - 't'          → tl;dr
//   - 'r'          → read here (full text to stdout)
//   - 'q', ESC     → quit
//   - Ctrl-C       → exit 130 immediately
//
// Falls back to line-mode prompt if raw mode fails.
func promptMenu(w io.Writer) menuAction {
	fmt.Fprintln(w, "\nWhat now?")
	fmt.Fprintln(w, "  [o] open in browser (default)")
	fmt.Fprintln(w, "  [t] tl;dr — summarize with Claude")
	fmt.Fprintln(w, "  [r] read full text here")
	fmt.Fprintln(w, "  [q] quit")
	fmt.Fprint(w, "> ")

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Line-mode fallback
		return readMenuLineMode(w)
	}
	defer term.Restore(fd, oldState)

	buf := make([]byte, 1)
	for attempt := 0; attempt < 2; attempt++ {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			fmt.Fprintln(w)
			return menuActionOpen
		}
		switch buf[0] {
		case 'o', 'O':
			fmt.Fprintln(w, "o")
			return menuActionOpen
		case 't', 'T':
			fmt.Fprintln(w, "t")
			return menuActionTldr
		case 'r', 'R':
			fmt.Fprintln(w, "r")
			return menuActionReadHere
		case 'q', 'Q':
			fmt.Fprintln(w, "q")
			return menuActionQuit
		case '\r', '\n':
			fmt.Fprintln(w, "o")
			return menuActionOpen
		case 0x1b: // ESC
			fmt.Fprintln(w, "esc")
			return menuActionQuit
		case 0x03: // Ctrl-C
			fmt.Fprintln(w, "^C")
			term.Restore(fd, oldState)
			os.Exit(130)
		}
		if attempt == 0 {
			fmt.Fprint(w, "\n> ")
		}
	}
	fmt.Fprintln(w)
	return menuActionOpen
}

// readMenuLineMode is the fallback for environments where raw mode fails.
// Reads a line and interprets the first character.
func readMenuLineMode(w io.Writer) menuAction {
	fmt.Fprint(w, "(press letter + Enter) ")
	var input string
	_, err := fmt.Fscanln(os.Stdin, &input)
	if err != nil || input == "" {
		return menuActionOpen
	}
	switch input[0] {
	case 'o', 'O':
		return menuActionOpen
	case 't', 'T':
		return menuActionTldr
	case 'r', 'R':
		return menuActionReadHere
	case 'q', 'Q':
		return menuActionQuit
	}
	return menuActionOpen
}
