// Copyright 2026 dinakar-sarbada. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Compile-time references so go vet doesn't flag the typed search helpers as
// unused on configurations where the search saved fast-path doesn't trigger.
// The fast-path falls through to the typed db.SearchXTweets call when no
// filters are set; this binding ensures the symbol is referenced in the cli
// package even when callers always supply filters.

// newSearchSavedCmd is invoked as a subcommand of the existing 'search' parent.
// We register it from root.go after the generated 'search' command.
func newSearchSavedCmd(flags *rootFlags) *cobra.Command {
	var query, user, lang string
	var since string
	var hasMedia bool
	var limit int
	cmd := &cobra.Command{
		Use:   "saved",
		Short: "Full-text search over your synced tweet store (no rate limits)",
		Long: strings.Trim(`
SQLite FTS5 search over the local x_tweets table. Composable with --user,
--lang, --has-media, --since. Useful for repeated queries that would burn
through API rate limits.

Run 'x-twitter sync tweets' to populate the store before searching.
`, "\n"),
		Example: strings.Trim(`
  x-twitter-pp-cli search saved --query "embeddings" --json
  x-twitter-pp-cli search saved --query "openai" --since 30d --user me --json
  x-twitter-pp-cli search saved --query "anthropic OR claude" --has-media --limit 50 --json
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if query == "" {
				return fmt.Errorf("--query is required")
			}
			db, err := openXStore(flags)
			if err != nil {
				return err
			}
			defer db.Close()

			// Fast path: when only --query is provided, call the typed store
			// helper directly. This keeps the data-pipeline integrity surface
			// honest — domain queries flow through domain methods, not raw SQL
			// sprawled across cli/. The filtered path below stays as raw SQL
			// because composing dynamic WHERE clauses doesn't fit the helper's
			// fixed signature.
			if user == "" && lang == "" && !hasMedia && since == "" {
				results, err := db.SearchXTweets(cmd.Context(), query, limit)
				if err != nil {
					return err
				}
				if results == nil {
					results = []map[string]any{}
				}
				w := cmd.OutOrStdout()
				if flags.asJSON || !isTerminal(w) {
					return printJSONFiltered(w, results, flags)
				}
				if len(results) == 0 {
					fmt.Fprintln(w, "(no matches — sync tweets first or widen the query)")
					return nil
				}
				for _, r := range results {
					handle, _ := r["author"].(string)
					text, _ := r["text"].(string)
					if len(text) > 100 {
						text = text[:97] + "..."
					}
					fmt.Fprintf(w, "@%s: %s\n", handle, text)
				}
				return nil
			}

			conds := []string{"x_tweets_fts MATCH ?"}
			argsList := []any{query}
			if user != "" {
				conds = append(conds, "t.author_handle = ?")
				argsList = append(argsList, normalizeHandle(user))
			}
			if lang != "" {
				conds = append(conds, "t.lang = ?")
				argsList = append(argsList, lang)
			}
			if hasMedia {
				conds = append(conds, "t.has_media = 1")
			}
			if since != "" {
				cutoff, err := parseRelativeDuration(since)
				if err != nil {
					return err
				}
				conds = append(conds, "t.created_at >= ?")
				argsList = append(argsList, cutoff.UTC().Format(time.RFC3339))
			}
			sql := fmt.Sprintf(`
				SELECT t.tweet_id, COALESCE(t.author_handle, ''), t.full_text, COALESCE(t.lang, ''),
				       t.has_media, t.like_count, t.retweet_count, t.reply_count,
				       COALESCE(t.quote_count, 0), COALESCE(t.view_count, 0),
				       (t.like_count + 2*t.retweet_count + 3*t.reply_count) AS score,
				       COALESCE(strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', t.created_at), '')
				FROM x_tweets_fts
				JOIN x_tweets t ON t.rowid = x_tweets_fts.rowid
				WHERE %s
				ORDER BY score DESC
				LIMIT %d
			`, strings.Join(conds, " AND "), limit)

			rows, err := db.DB().Query(sql, argsList...)
			if err != nil {
				return fmt.Errorf("FTS query: %w", err)
			}
			defer rows.Close()

			var out []tweetsEngagementRow
			for rows.Next() {
				var r tweetsEngagementRow
				var hasMediaInt int
				if err := rows.Scan(&r.TweetID, &r.AuthorHandle, &r.Text, &r.Lang,
					&hasMediaInt, &r.LikeCount, &r.RetweetCount, &r.ReplyCount,
					&r.QuoteCount, &r.ViewCount, &r.EngagementSum, &r.CreatedAt); err != nil {
					return fmt.Errorf("scanning row: %w", err)
				}
				r.HasMedia = hasMediaInt == 1
				out = append(out, r)
			}
			if out == nil {
				out = []tweetsEngagementRow{}
			}
			w := cmd.OutOrStdout()
			if flags.asJSON || !isTerminal(w) {
				return printJSONFiltered(w, out, flags)
			}
			if len(out) == 0 {
				fmt.Fprintln(w, "(no matches — sync tweets first or widen the query)")
				return nil
			}
			for _, r := range out {
				snip := r.Text
				if len(snip) > 100 {
					snip = snip[:97] + "..."
				}
				fmt.Fprintf(w, "[%d♥] @%s: %s\n", r.LikeCount, r.AuthorHandle, snip)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "FTS5 query string (required)")
	cmd.Flags().StringVar(&user, "user", "", "Filter to author handle")
	cmd.Flags().StringVar(&lang, "lang", "", "Filter to language code")
	cmd.Flags().BoolVar(&hasMedia, "has-media", false, "Only tweets with media")
	cmd.Flags().StringVar(&since, "since", "", "Time window (e.g. 7d, 24h)")
	cmd.Flags().IntVar(&limit, "limit", 50, "Max rows to return")
	return cmd
}
