// Copyright 2026 dinakar-sarbada. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// tweetsEngagementRow is the row returned by tweets engagement.
type tweetsEngagementRow struct {
	TweetID        string `json:"tweet_id"`
	AuthorHandle   string `json:"author_handle"`
	Text           string `json:"text"`
	Lang           string `json:"lang,omitempty"`
	HasMedia       bool   `json:"has_media,omitempty"`
	LikeCount      int64  `json:"like_count"`
	RetweetCount   int64  `json:"retweet_count"`
	ReplyCount     int64  `json:"reply_count"`
	QuoteCount     int64  `json:"quote_count,omitempty"`
	ViewCount      int64  `json:"view_count,omitempty"`
	EngagementSum  int64  `json:"engagement_score"`
	CreatedAt      string `json:"created_at,omitempty"`
}

// newTweetsEngagementCmd will be called from the existing 'tweets' parent command.
// (We don't have a tweets parent yet — we'll add it during root wiring.)
func newTweetsEngagementCmd(flags *rootFlags) *cobra.Command {
	var top int
	var since, user, lang string
	var hasMedia bool
	cmd := &cobra.Command{
		Use:   "engagement",
		Short: "Local SQL leaderboard: tweets ranked by weighted engagement",
		Long: strings.Trim(`
Rank synced tweets by weighted engagement: 1*likes + 2*retweets + 3*replies.
Filter by user, language, media presence, or time window.

Reads from local store. Run 'x-twitter sync tweets --user me' first to populate.
`, "\n"),
		Example: strings.Trim(`
  x-twitter-pp-cli tweets engagement --top 10 --since 30d --json
  x-twitter-pp-cli tweets engagement --user me --top 20 --json --select tweet_id,text,like_count
  x-twitter-pp-cli tweets engagement --user me --has-media --top 10
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			db, err := openXStore(flags)
			if err != nil {
				return err
			}
			defer db.Close()

			conds := []string{"1=1"}
			argsList := []any{}
			if user != "" {
				conds = append(conds, "author_handle = ?")
				argsList = append(argsList, normalizeHandle(user))
			}
			if lang != "" {
				conds = append(conds, "lang = ?")
				argsList = append(argsList, lang)
			}
			if hasMedia {
				conds = append(conds, "has_media = 1")
			}
			if since != "" {
				cutoff, err := parseRelativeDuration(since)
				if err != nil {
					return err
				}
				conds = append(conds, "created_at >= ?")
				argsList = append(argsList, cutoff.UTC().Format(time.RFC3339))
			}
			query := fmt.Sprintf(`
				SELECT tweet_id, COALESCE(author_handle, ''), full_text, COALESCE(lang, ''),
				       has_media, like_count, retweet_count, reply_count,
				       COALESCE(quote_count, 0), COALESCE(view_count, 0),
				       (like_count + 2*retweet_count + 3*reply_count) AS score,
				       COALESCE(strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', created_at), '')
				FROM x_tweets
				WHERE %s
				ORDER BY score DESC
				LIMIT %d
			`, strings.Join(conds, " AND "), top)

			rows, err := db.DB().Query(query, argsList...)
			if err != nil {
				return fmt.Errorf("engagement query: %w", err)
			}
			defer rows.Close()
			var out []tweetsEngagementRow
			for rows.Next() {
				var r tweetsEngagementRow
				var hasMediaInt int
				if err := rows.Scan(&r.TweetID, &r.AuthorHandle, &r.Text, &r.Lang,
					&hasMediaInt, &r.LikeCount, &r.RetweetCount, &r.ReplyCount,
					&r.QuoteCount, &r.ViewCount, &r.EngagementSum, &r.CreatedAt); err != nil {
					return fmt.Errorf("scanning engagement row: %w", err)
				}
				r.HasMedia = hasMediaInt == 1
				out = append(out, r)
			}
			if out == nil {
				out = []tweetsEngagementRow{}
			}
			w := cmd.OutOrStdout()
			if flags.asJSON || !isTerminal(w) {
				if len(out) == 0 {
					emitEmptyStoreHint(cmd, "tweets")
				}
				return printJSONFiltered(w, out, flags)
			}
			if len(out) == 0 {
				fmt.Fprintln(w, "(no tweets in local store — run 'sync tweets --user me' first)")
				return nil
			}
			for _, r := range out {
				snip := r.Text
				if len(snip) > 100 {
					snip = snip[:97] + "..."
				}
				fmt.Fprintf(w, "[%d] @%s — likes=%d rt=%d re=%d\n    %s\n",
					r.EngagementSum, r.AuthorHandle, r.LikeCount, r.RetweetCount, r.ReplyCount, snip)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&top, "top", 10, "How many tweets to return")
	cmd.Flags().StringVar(&since, "since", "", "Only include tweets within window (e.g. 7d, 30d, 24h)")
	cmd.Flags().StringVar(&user, "user", "", "Filter to a specific author handle (without @)")
	cmd.Flags().StringVar(&lang, "lang", "", "Filter to language code (e.g. en, ja)")
	cmd.Flags().BoolVar(&hasMedia, "has-media", false, "Only tweets that include media attachments")
	return cmd
}

// newTweetsParentCmd is a small parent grouping that wires `tweets engagement`.
// The existing graphql_create-tweet etc are unaffected; this is a new top-level group.
func newTweetsParentCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tweets",
		Short: "Tweet analytics over your local synced store",
		Long: strings.Trim(`
Local SQL analytics over synced tweets. Use 'graphql_*' commands for live API
mutation; this group is for offline queries that wouldn't fit a single endpoint.
`, "\n"),
	}
	cmd.AddCommand(newTweetsEngagementCmd(flags))
	return cmd
}
