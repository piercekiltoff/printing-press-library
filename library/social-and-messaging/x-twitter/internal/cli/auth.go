// Copyright 2026 dinakar-sarbada. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/social-and-messaging/x-twitter/internal/config"

	"github.com/spf13/cobra"
)

func newAuthCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage X session cookies (auth_token, ct0, guest_id)",
		Long: strings.Trim(`
Manage X (Twitter) session credentials.

x-twitter uses cookie-based auth captured from a logged-in browser session.
The required cookies are:
  - auth_token (X session)
  - ct0 (CSRF token, mirrored into x-csrf-token header)
  - guest_id (X guest tracking)

Use 'auth login --chrome' to import them from your browser, or
'auth login --paste' to enter them manually.
`, "\n"),
	}

	cmd.AddCommand(newAuthStatusCmd(flags))
	cmd.AddCommand(newAuthLoginCmd(flags))
	cmd.AddCommand(newAuthSetTokenCmd(flags))
	cmd.AddCommand(newAuthLogoutCmd(flags))

	return cmd
}

func newAuthStatusCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Short:   "Show authentication status",
		Example: "  x-twitter-pp-cli auth status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}

			w := cmd.OutOrStdout()
			if !cfg.HasCookieAuth() {
				fmt.Fprintln(w, red("Not authenticated"))
				fmt.Fprintln(w, "")
				fmt.Fprintln(w, "Capture cookies from your logged-in browser:")
				fmt.Fprintln(w, "  x-twitter-pp-cli auth login --chrome")
				fmt.Fprintln(w, "")
				fmt.Fprintln(w, "Or paste them manually:")
				fmt.Fprintln(w, "  x-twitter-pp-cli auth login --paste")
				return authErr(fmt.Errorf("no cookies configured"))
			}

			fmt.Fprintln(w, green("Authenticated (cookie-based)"))
			fmt.Fprintf(w, "  auth_token: %s...%s\n", first(cfg.AuthToken, 6), last(cfg.AuthToken, 4))
			fmt.Fprintf(w, "  ct0:        %s...%s\n", first(cfg.CSRFToken, 6), last(cfg.CSRFToken, 4))
			if cfg.GuestID != "" {
				fmt.Fprintf(w, "  guest_id:   %s\n", cfg.GuestID)
			}
			if !cfg.CapturedAt.IsZero() {
				fmt.Fprintf(w, "  Captured:   %s ago (from %s)\n", time.Since(cfg.CapturedAt).Round(time.Minute), cfg.CapturedFrom)
			}
			fmt.Fprintf(w, "  Source:     %s\n", cfg.AuthSource)
			fmt.Fprintf(w, "  Config:     %s\n", cfg.Path)
			return nil
		},
	}
}

func newAuthLoginCmd(flags *rootFlags) *cobra.Command {
	var fromChrome, fromPaste bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Capture X session cookies (--chrome or --paste)",
		Example: strings.Trim(`
  x-twitter-pp-cli auth login --chrome
  x-twitter-pp-cli auth login --paste
`, "\n"),
		Long: strings.Trim(`
Capture X session cookies and persist them to the config file.

  --chrome   Read cookies from your local Chrome cookie store.
             Requires you to be logged in to x.com in Chrome on macOS.
             Note: Chrome cookies are encrypted; this command needs
             access to the macOS Keychain to decrypt them.
  --paste    Prompt interactively for cookie values. Use this when
             auto-capture fails or you're using a different browser.
             Find them via DevTools > Application > Cookies > x.com.
`, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !fromChrome && !fromPaste {
				return fmt.Errorf("specify --chrome or --paste")
			}
			if fromChrome && fromPaste {
				return fmt.Errorf("specify either --chrome OR --paste, not both")
			}

			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}

			if dryRunOK(flags) {
				fmt.Fprintln(cmd.OutOrStdout(), "would capture cookies (dry-run)")
				return nil
			}

			var authToken, csrfToken, guestID, source string
			if fromChrome {
				authToken, csrfToken, guestID, err = readChromeXCookies()
				if err != nil {
					return authErr(fmt.Errorf("reading Chrome cookies: %w\n\nFallback: use --paste and copy the cookies from DevTools manually.", err))
				}
				source = "chrome"
			} else {
				authToken, csrfToken, guestID, err = promptCookies(cmd.InOrStdin(), cmd.OutOrStdout())
				if err != nil {
					return fmt.Errorf("reading cookies: %w", err)
				}
				source = "paste"
			}

			if authToken == "" || csrfToken == "" {
				return fmt.Errorf("auth_token and ct0 are required")
			}

			if err := cfg.SaveCookies(authToken, csrfToken, guestID, source); err != nil {
				return configErr(fmt.Errorf("saving cookies: %w", err))
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s cookies saved to %s\n", green("OK"), cfg.Path)
			fmt.Fprintln(cmd.OutOrStdout(), "Run 'x-twitter-pp-cli doctor' to verify.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&fromChrome, "chrome", false, "Import cookies from local Chrome cookie store (macOS)")
	cmd.Flags().BoolVar(&fromPaste, "paste", false, "Prompt interactively for cookie values")
	return cmd
}

func newAuthSetTokenCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "set-token <auth_token>",
		Short:   "Set the auth_token cookie value (legacy; prefer auth login)",
		Example: "  x-twitter-pp-cli auth set-token abc123def456",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			cfg.AuthToken = args[0]
			cfg.CapturedFrom = "set-token"
			cfg.CapturedAt = time.Now().UTC()
			if err := cfg.SaveCookies(args[0], cfg.CSRFToken, cfg.GuestID, "set-token"); err != nil {
				return configErr(fmt.Errorf("saving auth_token: %w", err))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "auth_token saved to %s\n", cfg.Path)
			if cfg.CSRFToken == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Note: ct0 (CSRF token) is also required. Run 'auth login --paste' to set it.")
			}
			return nil
		},
	}
}

func newAuthLogoutCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "logout",
		Short:   "Clear stored cookies",
		Example: "  x-twitter-pp-cli auth logout",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(flags.configPath)
			if err != nil {
				return configErr(err)
			}
			if err := cfg.ClearTokens(); err != nil {
				return configErr(fmt.Errorf("clearing cookies: %w", err))
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out. Cookies cleared.")
			return nil
		},
	}
}

// promptCookies asks the user to paste auth_token, ct0, and (optional) guest_id.
func promptCookies(in io.Reader, out io.Writer) (authToken, csrfToken, guestID string, err error) {
	r := bufio.NewReader(in)
	fmt.Fprintln(out, "Paste your X session cookies (find them in DevTools > Application > Cookies > x.com):")
	fmt.Fprint(out, "  auth_token: ")
	authToken, err = readLine(r)
	if err != nil {
		return
	}
	fmt.Fprint(out, "  ct0: ")
	csrfToken, err = readLine(r)
	if err != nil {
		return
	}
	fmt.Fprint(out, "  guest_id (optional, press Enter to skip): ")
	guestID, err = readLine(r)
	if err != nil {
		return
	}
	return
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// readChromeXCookies attempts to read X session cookies from the Chrome cookie store.
// macOS-only path: ~/Library/Application Support/Google/Chrome/Default/Cookies (encrypted SQLite).
// Cookies are encrypted with AES-128-CBC using PBKDF2-derived key from the keychain "Chrome Safe Storage" password.
// To avoid bundling crypto + keychain dependencies in the generated CLI, we surface a helpful
// fallback message if extraction is non-trivial in the user's environment.
func readChromeXCookies() (string, string, string, error) {
	if runtime.GOOS != "darwin" {
		return "", "", "", fmt.Errorf("--chrome is currently macOS-only; use --paste on %s", runtime.GOOS)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", "", err
	}
	cookieDB := filepath.Join(home, "Library", "Application Support", "Google", "Chrome", "Default", "Cookies")
	if _, err := os.Stat(cookieDB); err != nil {
		return "", "", "", fmt.Errorf("Chrome cookie database not found at %s", cookieDB)
	}

	// Copy the DB to a temp file (Chrome holds an exclusive lock when running).
	tmp, err := os.CreateTemp("", "x-twitter-cookies-*.db")
	if err != nil {
		return "", "", "", err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	src, err := os.Open(cookieDB)
	if err != nil {
		return "", "", "", fmt.Errorf("opening Chrome cookie DB: %w (close Chrome and try again, or use --paste)", err)
	}
	defer src.Close()
	dst, err := os.OpenFile(tmp.Name(), os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", "", "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return "", "", "", err
	}
	dst.Close()

	db, err := sql.Open("sqlite", tmp.Name())
	if err != nil {
		return "", "", "", fmt.Errorf("opening cookie DB: %w", err)
	}
	defer db.Close()

	// Query for x.com / .twitter.com cookies. Encrypted_value is bytes; value is plaintext (often empty).
	rows, err := db.Query(`
		SELECT name, value, encrypted_value, host_key
		FROM cookies
		WHERE (host_key LIKE '%x.com' OR host_key LIKE '%twitter.com')
		  AND name IN ('auth_token', 'ct0', 'guest_id')
	`)
	if err != nil {
		return "", "", "", fmt.Errorf("querying cookies: %w", err)
	}
	defer rows.Close()

	var encryptedFound bool
	for rows.Next() {
		var name, value, host string
		var encrypted []byte
		if err := rows.Scan(&name, &value, &encrypted, &host); err != nil {
			continue
		}
		// Plaintext path (rare on modern Chrome but supported)
		if value != "" {
			switch name {
			case "auth_token":
				authToken = value
			case "ct0":
				csrfToken = value
			case "guest_id":
				guestID = value
			}
			continue
		}
		// Encrypted path: would need keychain access + AES decryption.
		if len(encrypted) > 0 {
			encryptedFound = true
		}
	}

	if authToken != "" && csrfToken != "" {
		return authToken, csrfToken, guestID, nil
	}
	if encryptedFound {
		return "", "", "", fmt.Errorf("Chrome cookies are encrypted (modern Chrome default).\n\nUse --paste instead:\n  1. Open Chrome, go to x.com, ensure you're logged in\n  2. Open DevTools (Cmd+Option+I) > Application > Cookies > https://x.com\n  3. Copy the values for auth_token, ct0, guest_id\n  4. Run: x-twitter-pp-cli auth login --paste")
	}
	return "", "", "", fmt.Errorf("no x.com cookies found in Chrome. Are you logged in to x.com?")
}

// authToken / csrfToken / guestID file-scope captures from extractor; tests can override.
var (
	authToken string
	csrfToken string
	guestID   string
)

// first returns the first n characters of s (safe for short strings).
func first(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// last returns the last n characters of s (safe for short strings).
func last(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
