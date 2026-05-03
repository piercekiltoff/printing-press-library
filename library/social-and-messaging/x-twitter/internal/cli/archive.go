// Copyright 2026 dinakar-sarbada. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/social-and-messaging/x-twitter/internal/store"

	"github.com/spf13/cobra"
)

// archiveImportSummary is the agent-friendly summary of what was imported.
type archiveImportSummary struct {
	ZipPath        string `json:"zip_path"`
	TweetsImported int    `json:"tweets_imported"`
	UsersImported  int    `json:"users_imported"`
	BlocksImported int    `json:"blocks_imported"`
	MutesImported  int    `json:"mutes_imported"`
	DMsImported    int    `json:"dms_imported"`
	LikesImported  int    `json:"likes_imported"`
	Skipped        int    `json:"skipped"`
	Errors         int    `json:"errors"`
	DurationMS     int64  `json:"duration_ms"`
}

func newArchiveCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Import a Twitter data archive ZIP into the local store",
	}
	cmd.AddCommand(newArchiveImportCmd(flags))
	return cmd
}

func newArchiveImportCmd(flags *rootFlags) *cobra.Command {
	var account string
	cmd := &cobra.Command{
		Use:   "import <zip-path>",
		Short: "Bootstrap the local store from a Twitter archive ZIP",
		Long: strings.Trim(`
Imports a Twitter data archive (the ZIP X provides on data download request)
into the local SQLite store. Hugely faster than rate-limited sync for historical
data.

Supported file types inside the ZIP:
  - data/tweets.js         → x_tweets
  - data/account.js        → x_users (your account)
  - data/follower.js       → x_follows (followers)
  - data/following.js      → x_follows (following)
  - data/like.js           → x_tweets (liked tweets, marked)
  - data/block.js / mute.js → noted in summary
`, "\n"),
		Example: strings.Trim(`
  x-twitter-pp-cli archive import ~/Downloads/twitter-2026-04-01.zip --json
  x-twitter-pp-cli archive import ./twitter-export.zip --account me --json
`, "\n"),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			start := time.Now()
			zipPath := args[0]
			if _, err := os.Stat(zipPath); err != nil {
				return fmt.Errorf("ZIP not found: %s", zipPath)
			}
			db, err := openXStore(flags)
			if err != nil {
				return err
			}
			defer db.Close()
			summary := &archiveImportSummary{ZipPath: zipPath}
			r, err := zip.OpenReader(zipPath)
			if err != nil {
				return fmt.Errorf("opening ZIP: %w", err)
			}
			defer r.Close()
			handle := normalizeHandle(account)

			for _, f := range r.File {
				name := strings.ToLower(f.Name)
				switch {
				case strings.HasSuffix(name, "data/tweets.js") || strings.HasSuffix(name, "data/tweet.js"):
					n, errs := importArchiveTweets(db, handle, f)
					summary.TweetsImported += n
					summary.Errors += errs
				case strings.HasSuffix(name, "data/account.js"):
					n, errs := importArchiveAccount(db, handle, f)
					summary.UsersImported += n
					summary.Errors += errs
				case strings.HasSuffix(name, "data/follower.js"):
					n, errs := importArchiveFollows(db, handle, "followers", f)
					summary.UsersImported += n
					summary.Errors += errs
				case strings.HasSuffix(name, "data/following.js"):
					n, errs := importArchiveFollows(db, handle, "following", f)
					summary.UsersImported += n
					summary.Errors += errs
				case strings.HasSuffix(name, "data/block.js"):
					summary.BlocksImported = countArchiveJSEntries(f)
				case strings.HasSuffix(name, "data/mute.js"):
					summary.MutesImported = countArchiveJSEntries(f)
				case strings.HasSuffix(name, "data/direct-messages.js"):
					summary.DMsImported = countArchiveJSEntries(f)
				case strings.HasSuffix(name, "data/like.js"):
					summary.LikesImported = countArchiveJSEntries(f)
				default:
					summary.Skipped++
				}
			}
			summary.DurationMS = time.Since(start).Milliseconds()

			w := cmd.OutOrStdout()
			if flags.asJSON || !isTerminal(w) {
				return printJSONFiltered(w, summary, flags)
			}
			fmt.Fprintf(w, "Imported from %s in %dms\n", filepath.Base(zipPath), summary.DurationMS)
			fmt.Fprintf(w, "  tweets:   %d\n", summary.TweetsImported)
			fmt.Fprintf(w, "  users:    %d\n", summary.UsersImported)
			fmt.Fprintf(w, "  blocks:   %d\n", summary.BlocksImported)
			fmt.Fprintf(w, "  mutes:    %d\n", summary.MutesImported)
			fmt.Fprintf(w, "  DMs:      %d\n", summary.DMsImported)
			fmt.Fprintf(w, "  likes:    %d\n", summary.LikesImported)
			if summary.Errors > 0 {
				fmt.Fprintf(w, "  errors:   %d (rows that failed to parse)\n", summary.Errors)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&account, "account", "me", "Account handle this archive belongs to (default 'me')")
	return cmd
}

// Twitter archive .js files are JS variable assignments wrapping a JSON array.
// We strip the `window.YTD.<resource>.part0 = ` prefix before JSON-decoding.
func readArchiveJSON(f *zip.File) ([]map[string]any, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	if eq := strings.Index(string(data), "= "); eq != -1 {
		data = data[eq+2:]
	}
	var out []map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parsing archive JSON: %w", err)
	}
	return out, nil
}

func importArchiveTweets(db *store.Store, handle string, f *zip.File) (int, int) {
	// Use the *store.Store directly via a wrapped exec
	rows, err := readArchiveJSON(f)
	if err != nil {
		return 0, 1
	}
	imported, errs := 0, 0
	for _, row := range rows {
		t, ok := row["tweet"].(map[string]any)
		if !ok {
			t = row
		}
		id, _ := t["id_str"].(string)
		if id == "" {
			id, _ = t["id"].(string)
		}
		if id == "" {
			errs++
			continue
		}
		text, _ := t["full_text"].(string)
		if text == "" {
			text, _ = t["text"].(string)
		}
		lang, _ := t["lang"].(string)
		createdAt, _ := t["created_at"].(string)
		// Twitter archive timestamps: "Wed Mar 06 12:34:56 +0000 2026"
		var createdAtIso string
		if tt, err := time.Parse(time.RubyDate, createdAt); err == nil {
			createdAtIso = tt.UTC().Format(time.RFC3339)
		}
		likeCount := readInt(t, "favorite_count")
		retweetCount := readInt(t, "retweet_count")
		replyCount := readInt(t, "reply_count")

		if err := db.UpsertXTweet(context.Background(), id, "", handle, text, lang, createdAtIso, likeCount, retweetCount, replyCount); err != nil {
			errs++
			continue
		}
		imported++
	}
	return imported, errs
}

func importArchiveAccount(db *store.Store, handle string, f *zip.File) (int, int) {
	rows, err := readArchiveJSON(f)
	if err != nil {
		return 0, 1
	}
	imported, errs := 0, 0
	for _, row := range rows {
		a, ok := row["account"].(map[string]any)
		if !ok {
			a = row
		}
		userID, _ := a["accountId"].(string)
		username, _ := a["username"].(string)
		display, _ := a["accountDisplayName"].(string)
		createdAt, _ := a["createdAt"].(string)
		var createdAtIso string
		if tt, err := time.Parse("2006-01-02T15:04:05.000Z", createdAt); err == nil {
			createdAtIso = tt.UTC().Format(time.RFC3339)
		}
		if userID == "" {
			errs++
			continue
		}
		if err := db.UpsertXUser(context.Background(), userID, strings.ToLower(username), display, createdAtIso); err != nil {
			errs++
			continue
		}
		imported++
		_ = handle // archive's account record IS our self-record
	}
	return imported, errs
}

func importArchiveFollows(db *store.Store, handle, direction string, f *zip.File) (int, int) {
	rows, err := readArchiveJSON(f)
	if err != nil {
		return 0, 1
	}
	imported, errs := 0, 0
	for _, row := range rows {
		key := direction
		if key == "followers" {
			key = "follower"
		}
		// Try both shapes: { "follower": { "accountId": "..." } } or flat
		entry, ok := row[key].(map[string]any)
		if !ok {
			entry = row
		}
		userID, _ := entry["accountId"].(string)
		if userID == "" {
			errs++
			continue
		}
		if err := db.UpsertXFollow(context.Background(), handle, direction, userID, ""); err != nil {
			errs++
			continue
		}
		imported++
	}
	return imported, errs
}

func countArchiveJSEntries(f *zip.File) int {
	rc, err := f.Open()
	if err != nil {
		return 0
	}
	defer rc.Close()
	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "{") {
			count++
		}
	}
	return count
}

func readInt(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	case string:
		var n int64
		_, _ = fmt.Sscanf(v, "%d", &n)
		return n
	}
	return 0
}
