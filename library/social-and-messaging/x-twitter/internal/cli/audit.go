// Copyright 2026 dinakar-sarbada. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newAuditCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audience hygiene heuristics over your synced follow data",
		Long: strings.Trim(`
Heuristics that surface accounts worth reviewing: silent (inactive) accounts you
follow, and likely-bot followers of yours.

All reads come from the local store. Run 'x-twitter sync followers' and 'sync
following' before running these.
`, "\n"),
	}
	cmd.AddCommand(newAuditInactiveCmd(flags))
	cmd.AddCommand(newAuditSuspiciousFollowersCmd(flags))
	return cmd
}

func newAuditInactiveCmd(flags *rootFlags) *cobra.Command {
	var account string
	var days int
	var limit int
	cmd := &cobra.Command{
		Use:   "inactive",
		Short: "Accounts you follow that haven't tweeted in N days",
		Example: strings.Trim(`
  x-twitter-pp-cli audit inactive --days 90 --json
  x-twitter-pp-cli audit inactive --days 365 --limit 200 --json --select handle,name,last_tweet_at
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
			handle := normalizeHandle(account)
			cutoff := time.Now().AddDate(0, 0, -days).UTC().Format(time.RFC3339)
			query := `
				SELECT u.user_id, u.handle, COALESCE(u.display_name, ''), COALESCE(u.bio, ''),
				       COALESCE(u.followers_count, 0), COALESCE(u.following_count, 0),
				       COALESCE(u.tweet_count, 0), COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.last_tweet_at), ''),
				       COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.account_created_at), ''),
				       strftime('%Y-%m-%dT%H:%M:%SZ', f.scraped_at)
				FROM x_follows f
				JOIN x_users u ON u.user_id = f.user_id
				WHERE f.account_handle = ?
				  AND f.direction = 'following'
				  AND (u.last_tweet_at IS NULL OR u.last_tweet_at < ?)
				ORDER BY u.last_tweet_at ASC NULLS FIRST
			`
			if limit > 0 {
				query += fmt.Sprintf(" LIMIT %d", limit)
			}
			rows, err := queryUserRows(db, query, handle, cutoff)
			if err != nil {
				return err
			}
			return emitUserRows(cmd, flags, rows)
		},
	}
	cmd.Flags().StringVar(&account, "account", "me", "Account perspective to query (handle without @)")
	cmd.Flags().IntVar(&days, "days", 90, "Inactivity threshold in days")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows to return (0 = unlimited)")
	return cmd
}

// suspiciousFollowerRow is the auditable row with score breakdown.
type suspiciousFollowerRow struct {
	xUserRow
	Score   float64  `json:"score"`
	Signals []string `json:"signals"`
}

func newAuditSuspiciousFollowersCmd(flags *rootFlags) *cobra.Command {
	var account string
	var threshold float64
	var limit int
	cmd := &cobra.Command{
		Use:   "suspicious-followers",
		Short: "Heuristic-based bot detection on your synced followers",
		Long: strings.Trim(`
Score each of your followers on a 0-1 risk scale based on heuristics that
correlate with automated/spam accounts:

  default profile image          (+0.30)
  default name pattern           (+0.20)
  zero original tweets           (+0.20)
  high follow ratio (>5x)        (+0.15)
  account < 30 days old          (+0.15)

Higher scores are likelier bots. Inspect signals to validate.
`, "\n"),
		Example: strings.Trim(`
  x-twitter-pp-cli audit suspicious-followers --threshold 0.5 --json
  x-twitter-pp-cli audit suspicious-followers --threshold 0.7 --limit 100 --json
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
			handle := normalizeHandle(account)
			query := `
				SELECT u.user_id, u.handle, COALESCE(u.display_name, ''), COALESCE(u.bio, ''),
				       COALESCE(u.followers_count, 0), COALESCE(u.following_count, 0),
				       COALESCE(u.tweet_count, 0), COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.last_tweet_at), ''),
				       COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.account_created_at), ''),
				       strftime('%Y-%m-%dT%H:%M:%SZ', f.scraped_at),
				       COALESCE(u.profile_image_url, '')
				FROM x_follows f
				JOIN x_users u ON u.user_id = f.user_id
				WHERE f.account_handle = ? AND f.direction = 'followers'
			`
			rows, err := db.DB().Query(query, handle)
			if err != nil {
				return fmt.Errorf("audit query: %w", err)
			}
			defer rows.Close()
			var scored []suspiciousFollowerRow
			for rows.Next() {
				var r xUserRow
				var profileImg string
				if err := rows.Scan(&r.UserID, &r.Handle, &r.DisplayName, &r.Bio,
					&r.FollowersCount, &r.FollowingCount, &r.TweetCount,
					&r.LastTweetAt, &r.AccountCreated, &r.ScrapedAt, &profileImg); err != nil {
					continue
				}
				score, signals := suspiciousScore(r, profileImg)
				if score >= threshold {
					scored = append(scored, suspiciousFollowerRow{xUserRow: r, Score: round2(score), Signals: signals})
				}
			}
			if scored == nil {
				scored = []suspiciousFollowerRow{}
			}
			// Highest score first
			sortByScoreDesc(scored)
			if limit > 0 && len(scored) > limit {
				scored = scored[:limit]
			}
			w := cmd.OutOrStdout()
			if flags.asJSON || !isTerminal(w) {
				if len(scored) == 0 {
					emitEmptyStoreHint(cmd, "followers,following")
				}
				return printJSONFiltered(w, scored, flags)
			}
			if len(scored) == 0 {
				fmt.Fprintln(w, "(no suspicious followers above threshold)")
				return nil
			}
			for _, r := range scored {
				fmt.Fprintf(w, "@%s [score=%.2f] %s\n    signals: %s\n", r.Handle, r.Score, r.DisplayName, strings.Join(r.Signals, ", "))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&account, "account", "me", "Account perspective to query (handle without @)")
	cmd.Flags().Float64Var(&threshold, "threshold", 0.5, "Minimum suspicion score (0-1) to include")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows to return (0 = unlimited)")
	return cmd
}

func suspiciousScore(r xUserRow, profileImg string) (float64, []string) {
	var score float64
	var signals []string
	if profileImg == "" || strings.Contains(profileImg, "default_profile_images") {
		score += 0.30
		signals = append(signals, "default profile image")
	}
	if looksLikeDefaultName(r.Handle) {
		score += 0.20
		signals = append(signals, "default-pattern handle")
	}
	if r.TweetCount == 0 {
		score += 0.20
		signals = append(signals, "zero tweets")
	}
	if r.FollowersCount > 0 && r.FollowingCount > 5*r.FollowersCount {
		score += 0.15
		signals = append(signals, "high follow ratio")
	}
	if r.AccountCreated != "" {
		if t, err := time.Parse(time.RFC3339, r.AccountCreated); err == nil {
			if time.Since(t) < 30*24*time.Hour {
				score += 0.15
				signals = append(signals, "account < 30 days old")
			}
		}
	}
	return score, signals
}

// looksLikeDefaultName flags handles like 'user12345678', '12345', 'name_xxxxxxxx'.
func looksLikeDefaultName(handle string) bool {
	if handle == "" {
		return false
	}
	digits := 0
	for _, r := range handle {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	// >50% digits is a strong default-name signal
	return float64(digits) > 0.5*float64(len(handle))
}

func round2(f float64) float64 {
	return float64(int(f*100)) / 100
}

// sortByScoreDesc sorts the slice in-place, highest score first.
func sortByScoreDesc(rows []suspiciousFollowerRow) {
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].Score > rows[i].Score {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}
