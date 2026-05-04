// Copyright 2026 dinakar-sarbada. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/social-and-messaging/x-twitter/internal/store"

	"github.com/spf13/cobra"
)

// xUserRow is the shared shape returned by every relationships subcommand.
type xUserRow struct {
	UserID         string `json:"user_id"`
	Handle         string `json:"handle"`
	DisplayName    string `json:"name,omitempty"`
	Bio            string `json:"bio,omitempty"`
	FollowersCount int64  `json:"followers_count,omitempty"`
	FollowingCount int64  `json:"following_count,omitempty"`
	TweetCount     int64  `json:"tweet_count,omitempty"`
	LastTweetAt    string `json:"last_tweet_at,omitempty"`
	AccountCreated string `json:"account_created_at,omitempty"`
	ScrapedAt      string `json:"scraped_at,omitempty"`
}

func newRelationshipsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relationships",
		Short: "Asymmetric relationship analytics over your synced followers/following",
		Long: strings.Trim(`
Local-store-backed relationship analytics: who follows whom.

All commands here read from the local SQLite store. Run 'x-twitter sync followers'
and 'x-twitter sync following' first (and re-run periodically to refresh data).

Snapshot-based commands (unfollowed-me, new-followers) require multiple syncs over
time so a diff can be computed.
`, "\n"),
	}
	cmd.AddCommand(newRelationshipsNotFollowingBackCmd(flags))
	cmd.AddCommand(newRelationshipsMutualsCmd(flags))
	cmd.AddCommand(newRelationshipsFansCmd(flags))
	cmd.AddCommand(newRelationshipsGhostFollowersCmd(flags))
	cmd.AddCommand(newRelationshipsUnfollowedMeCmd(flags))
	cmd.AddCommand(newRelationshipsNewFollowersCmd(flags))
	cmd.AddCommand(newRelationshipsOverlapCmd(flags))
	return cmd
}

func newRelationshipsNotFollowingBackCmd(flags *rootFlags) *cobra.Command {
	var account string
	var limit int
	cmd := &cobra.Command{
		Use:   "not-following-back",
		Short: "Accounts you follow that don't follow you back",
		Long: strings.Trim(`
List the asymmetric edges in your follow graph: accounts you follow that
don't follow you back. Useful for periodic audience hygiene.

Requires 'sync followers' and 'sync following' to have run for your account.
`, "\n"),
		Example: strings.Trim(`
  x-twitter-pp-cli relationships not-following-back --json
  x-twitter-pp-cli relationships not-following-back --account me --limit 100 --json --select handle,name,last_tweet_at
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
			rows, err := queryAsymmetric(db, handle, "following", "followers", limit)
			if err != nil {
				return err
			}
			return emitUserRows(cmd, flags, rows)
		},
	}
	cmd.Flags().StringVar(&account, "account", "me", "Account perspective to query (handle without @)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows to return (0 = unlimited)")
	return cmd
}

func newRelationshipsFansCmd(flags *rootFlags) *cobra.Command {
	var account, since string
	var limit int
	cmd := &cobra.Command{
		Use:   "fans",
		Short: "Followers you don't follow back (the inverse of not-following-back)",
		Example: strings.Trim(`
  x-twitter-pp-cli relationships fans --json
  x-twitter-pp-cli relationships fans --since 30d --json
  x-twitter-pp-cli relationships fans --limit 50 --json --select handle,name,followers_count
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
			rows, err := queryAsymmetricSince(db, handle, "followers", "following", since, limit)
			if err != nil {
				return err
			}
			return emitUserRows(cmd, flags, rows)
		},
	}
	cmd.Flags().StringVar(&account, "account", "me", "Account perspective to query (handle without @)")
	cmd.Flags().StringVar(&since, "since", "", "Filter to follows scraped within window (e.g. 7d, 24h)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows to return (0 = unlimited)")
	return cmd
}

func newRelationshipsMutualsCmd(flags *rootFlags) *cobra.Command {
	var account, since string
	var limit int
	cmd := &cobra.Command{
		Use:   "mutuals",
		Short: "Accounts you follow that also follow you back (two-way edges)",
		Example: strings.Trim(`
  x-twitter-pp-cli relationships mutuals --json
  x-twitter-pp-cli relationships mutuals --since 90d --json
  x-twitter-pp-cli relationships mutuals --limit 100 --json --select handle,name
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
			conds := []string{
				"f1.account_handle = ?",
				"f1.direction = 'following'",
				"f2.direction = 'followers'",
			}
			argsList := []any{handle}
			if since != "" {
				cutoff, err := parseRelativeDuration(since)
				if err != nil {
					return err
				}
				conds = append(conds, "f1.scraped_at >= ?")
				argsList = append(argsList, cutoff.UTC().Format(time.RFC3339))
			}
			query := fmt.Sprintf(`
				SELECT u.user_id, u.handle, COALESCE(u.display_name, ''), COALESCE(u.bio, ''),
				       COALESCE(u.followers_count, 0), COALESCE(u.following_count, 0),
				       COALESCE(u.tweet_count, 0), COALESCE(strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', u.last_tweet_at), ''),
				       COALESCE(strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', u.account_created_at), ''),
				       strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', f1.scraped_at)
				FROM x_follows f1
				JOIN x_follows f2
				  ON f1.user_id = f2.user_id
				 AND f1.account_handle = f2.account_handle
				LEFT JOIN x_users u ON u.user_id = f1.user_id
				WHERE %s
				ORDER BY u.followers_count DESC
			`, strings.Join(conds, " AND "))
			if limit > 0 {
				query += fmt.Sprintf(" LIMIT %d", limit)
			}
			rows, err := queryUserRows(db, query, argsList...)
			if err != nil {
				return err
			}
			return emitUserRows(cmd, flags, rows)
		},
	}
	cmd.Flags().StringVar(&account, "account", "me", "Account perspective to query (handle without @)")
	cmd.Flags().StringVar(&since, "since", "", "Filter to follows scraped within window (e.g. 7d, 24h)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows to return (0 = unlimited)")
	return cmd
}

func newRelationshipsGhostFollowersCmd(flags *rootFlags) *cobra.Command {
	var account string
	var days int
	var limit int
	cmd := &cobra.Command{
		Use:   "ghost-followers",
		Short: "Followers who haven't tweeted in N days (likely-inactive accounts)",
		Example: strings.Trim(`
  x-twitter-pp-cli relationships ghost-followers --days 90 --json
  x-twitter-pp-cli relationships ghost-followers --days 365 --limit 200 --json
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
				  AND f.direction = 'followers'
				  AND (u.last_tweet_at IS NULL OR u.last_tweet_at < ?)
				ORDER BY u.last_tweet_at ASC NULLS FIRST, u.followers_count DESC
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

func newRelationshipsUnfollowedMeCmd(flags *rootFlags) *cobra.Command {
	var account, since string
	var limit int
	cmd := &cobra.Command{
		Use:   "unfollowed-me",
		Short: "Diff snapshots to find followers you've lost recently",
		Long: strings.Trim(`
Compares the most recent follower snapshot against an older one to find user_ids
present in the older snapshot but missing from the newer.

Requires multiple 'sync followers' invocations over time — at least one before the
--since cutoff and at least one after.
`, "\n"),
		Example: strings.Trim(`
  x-twitter-pp-cli relationships unfollowed-me --since 7d --json
  x-twitter-pp-cli relationships unfollowed-me --since 30d --json --select handle,name
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
			cutoff, err := parseRelativeDuration(since)
			if err != nil {
				return err
			}
			rows, err := querySnapshotDiff(db, handle, "followers", cutoff, "removed", limit)
			if err != nil {
				return err
			}
			return emitUserRows(cmd, flags, rows)
		},
	}
	cmd.Flags().StringVar(&account, "account", "me", "Account perspective to query (handle without @)")
	cmd.Flags().StringVar(&since, "since", "7d", "Time window (e.g. 1h, 7d, 30d)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows to return (0 = unlimited)")
	return cmd
}

func newRelationshipsNewFollowersCmd(flags *rootFlags) *cobra.Command {
	var account, since string
	var limit int
	cmd := &cobra.Command{
		Use:   "new-followers",
		Short: "Followers gained in the last N days (snapshot diff)",
		Example: strings.Trim(`
  x-twitter-pp-cli relationships new-followers --since 7d --json
  x-twitter-pp-cli relationships new-followers --since 24h --json --select handle,name,followers_count
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
			cutoff, err := parseRelativeDuration(since)
			if err != nil {
				return err
			}
			rows, err := querySnapshotDiff(db, handle, "followers", cutoff, "added", limit)
			if err != nil {
				return err
			}
			return emitUserRows(cmd, flags, rows)
		},
	}
	cmd.Flags().StringVar(&account, "account", "me", "Account perspective to query (handle without @)")
	cmd.Flags().StringVar(&since, "since", "7d", "Time window (e.g. 1h, 7d, 30d)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows to return (0 = unlimited)")
	return cmd
}

func newRelationshipsOverlapCmd(flags *rootFlags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "overlap <user_a> <user_b>",
		Short: "Find accounts that follow BOTH user A and user B",
		Long: strings.Trim(`
INTERSECT the followers of two users to discover their shared audience.
Useful for warm-intro discovery, partnership outreach, audience analysis.

Requires 'sync followers --user <handle>' to have run for BOTH users.
`, "\n"),
		Example: strings.Trim(`
  x-twitter-pp-cli relationships overlap @paulg @sama --json --limit 50
  x-twitter-pp-cli relationships overlap @vercel @nextjs --json --select handle,name
`, "\n"),
		Args: cobra.MaximumNArgs(2),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			db, err := openXStore(flags)
			if err != nil {
				return err
			}
			defer db.Close()
			a := normalizeHandle(args[0])
			b := normalizeHandle(args[1])
			query := `
				SELECT u.user_id, u.handle, COALESCE(u.display_name, ''), COALESCE(u.bio, ''),
				       COALESCE(u.followers_count, 0), COALESCE(u.following_count, 0),
				       COALESCE(u.tweet_count, 0), COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.last_tweet_at), ''),
				       COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.account_created_at), ''),
				       strftime('%Y-%m-%dT%H:%M:%SZ', fa.scraped_at)
				FROM x_follows fa
				JOIN x_follows fb
				  ON fa.user_id = fb.user_id
				LEFT JOIN x_users u ON u.user_id = fa.user_id
				WHERE fa.account_handle = ?
				  AND fb.account_handle = ?
				  AND fa.direction = 'followers'
				  AND fb.direction = 'followers'
				ORDER BY u.followers_count DESC
			`
			if limit > 0 {
				query += fmt.Sprintf(" LIMIT %d", limit)
			}
			rows, err := queryUserRows(db, query, a, b)
			if err != nil {
				return err
			}
			// Wrap with the queried handles so consumers (and the live-check
			// probe) can see what was queried even when the result set is empty
			// (e.g., before sync has populated the store).
			w := cmd.OutOrStdout()
			if flags.asJSON || !isTerminal(w) {
				if len(rows) == 0 {
					emitEmptyStoreHint(cmd, "followers,following")
				}
				envelope := map[string]any{
					"user_a":  "@" + a,
					"user_b":  "@" + b,
					"overlap": rows,
				}
				return printJSONFiltered(w, envelope, flags)
			}
			return emitUserRows(cmd, flags, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "Max rows to return (0 = unlimited)")
	return cmd
}

// queryAsymmetricSince adds an optional --since filter on scraped_at.
func queryAsymmetricSince(db *store.Store, handle, direction, inverse, since string, limit int) ([]xUserRow, error) {
	conds := []string{"f.account_handle = ?", "f.direction = ?",
		"f.user_id NOT IN (SELECT user_id FROM x_follows WHERE account_handle = ? AND direction = ?)"}
	argsList := []any{handle, direction, handle, inverse}
	if since != "" {
		cutoff, err := parseRelativeDuration(since)
		if err != nil {
			return nil, err
		}
		conds = append(conds, "f.scraped_at >= ?")
		argsList = append(argsList, cutoff.UTC().Format(time.RFC3339))
	}
	query := fmt.Sprintf(`
		SELECT u.user_id, u.handle, COALESCE(u.display_name, ''), COALESCE(u.bio, ''),
		       COALESCE(u.followers_count, 0), COALESCE(u.following_count, 0),
		       COALESCE(u.tweet_count, 0), COALESCE(strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', u.last_tweet_at), ''),
		       COALESCE(strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', u.account_created_at), ''),
		       strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', f.scraped_at)
		FROM x_follows f
		LEFT JOIN x_users u ON u.user_id = f.user_id
		WHERE %s
		ORDER BY u.followers_count DESC
	`, strings.Join(conds, " AND "))
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	return queryUserRows(db, query, argsList...)
}

// queryAsymmetric returns rows in `direction` for `handle` that are NOT in `inverse`.
func queryAsymmetric(db *store.Store, handle, direction, inverse string, limit int) ([]xUserRow, error) {
	query := `
		SELECT u.user_id, u.handle, COALESCE(u.display_name, ''), COALESCE(u.bio, ''),
		       COALESCE(u.followers_count, 0), COALESCE(u.following_count, 0),
		       COALESCE(u.tweet_count, 0), COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.last_tweet_at), ''),
		       COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.account_created_at), ''),
		       strftime('%Y-%m-%dT%H:%M:%SZ', f.scraped_at)
		FROM x_follows f
		LEFT JOIN x_users u ON u.user_id = f.user_id
		WHERE f.account_handle = ?
		  AND f.direction = ?
		  AND f.user_id NOT IN (
		      SELECT user_id FROM x_follows
		      WHERE account_handle = ? AND direction = ?
		  )
		ORDER BY u.followers_count DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	return queryUserRows(db, query, handle, direction, handle, inverse)
}

// querySnapshotDiff finds user_ids that appeared (added) or disappeared (removed)
// between the most recent snapshot and the most recent snapshot before `cutoff`.
func querySnapshotDiff(db *store.Store, handle, direction string, cutoff time.Time, mode string, limit int) ([]xUserRow, error) {
	cutoffStr := cutoff.UTC().Format(time.RFC3339)
	var query string
	switch mode {
	case "removed":
		// In old snapshot, missing from new snapshot
		query = `
			WITH latest AS (
				SELECT MAX(snapshot_at) AS at FROM x_follow_snapshots
				WHERE account_handle = ? AND direction = ?
			), prev AS (
				SELECT MAX(snapshot_at) AS at FROM x_follow_snapshots
				WHERE account_handle = ? AND direction = ? AND snapshot_at <= ?
			)
			SELECT s.user_id, s.handle, COALESCE(u.display_name, ''), COALESCE(u.bio, ''),
			       COALESCE(u.followers_count, 0), COALESCE(u.following_count, 0),
			       COALESCE(u.tweet_count, 0), COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.last_tweet_at), ''),
			       COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.account_created_at), ''),
			       strftime('%Y-%m-%dT%H:%M:%SZ', s.snapshot_at)
			FROM x_follow_snapshots s
			LEFT JOIN x_users u ON u.user_id = s.user_id
			WHERE s.account_handle = ?
			  AND s.direction = ?
			  AND s.snapshot_at = (SELECT at FROM prev)
			  AND s.user_id NOT IN (
			      SELECT user_id FROM x_follow_snapshots
			      WHERE account_handle = ? AND direction = ? AND snapshot_at = (SELECT at FROM latest)
			  )
			ORDER BY u.followers_count DESC
		`
	case "added":
		query = `
			WITH latest AS (
				SELECT MAX(snapshot_at) AS at FROM x_follow_snapshots
				WHERE account_handle = ? AND direction = ?
			), prev AS (
				SELECT MAX(snapshot_at) AS at FROM x_follow_snapshots
				WHERE account_handle = ? AND direction = ? AND snapshot_at <= ?
			)
			SELECT s.user_id, s.handle, COALESCE(u.display_name, ''), COALESCE(u.bio, ''),
			       COALESCE(u.followers_count, 0), COALESCE(u.following_count, 0),
			       COALESCE(u.tweet_count, 0), COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.last_tweet_at), ''),
			       COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', u.account_created_at), ''),
			       strftime('%Y-%m-%dT%H:%M:%SZ', s.snapshot_at)
			FROM x_follow_snapshots s
			LEFT JOIN x_users u ON u.user_id = s.user_id
			WHERE s.account_handle = ?
			  AND s.direction = ?
			  AND s.snapshot_at = (SELECT at FROM latest)
			  AND s.user_id NOT IN (
			      SELECT user_id FROM x_follow_snapshots
			      WHERE account_handle = ? AND direction = ? AND snapshot_at = (SELECT at FROM prev)
			  )
			ORDER BY u.followers_count DESC
		`
	default:
		return nil, fmt.Errorf("invalid snapshot diff mode %q", mode)
	}
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	return queryUserRows(db, query, handle, direction, handle, direction, cutoffStr, handle, direction, handle, direction)
}

func queryUserRows(db *store.Store, query string, args ...any) ([]xUserRow, error) {
	rows, err := db.DB().Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("relationships query: %w", err)
	}
	defer rows.Close()
	var out []xUserRow
	for rows.Next() {
		var r xUserRow
		if err := rows.Scan(&r.UserID, &r.Handle, &r.DisplayName, &r.Bio,
			&r.FollowersCount, &r.FollowingCount, &r.TweetCount,
			&r.LastTweetAt, &r.AccountCreated, &r.ScrapedAt); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []xUserRow{}
	}
	return out, nil
}

func emitUserRows(cmd *cobra.Command, flags *rootFlags, rows []xUserRow) error {
	w := cmd.OutOrStdout()
	if flags.asJSON || !isTerminal(w) {
		if len(rows) == 0 {
			emitEmptyStoreHint(cmd, "followers,following")
		}
		return printJSONFiltered(w, rows, flags)
	}
	if len(rows) == 0 {
		fmt.Fprintln(w, "(no results — sync followers/following first, or widen the time window)")
		return nil
	}
	for _, r := range rows {
		name := r.DisplayName
		if name == "" {
			name = "(no name)"
		}
		bio := r.Bio
		if len(bio) > 80 {
			bio = bio[:77] + "..."
		}
		fmt.Fprintf(w, "@%s — %s [followers=%d, last_tweet=%s]\n", r.Handle, name, r.FollowersCount, r.LastTweetAt)
		if bio != "" {
			fmt.Fprintf(w, "    %s\n", bio)
		}
	}
	return nil
}

func emitEmptyStoreHint(cmd *cobra.Command, resources string) {
	fmt.Fprintf(cmd.ErrOrStderr(), "hint: local store is empty — run 'x-twitter-pp-cli sync --resources %s' to populate\n", resources)
}

// openXStore is a small wrapper that opens the default DB path.
func openXStore(flags *rootFlags) (*store.Store, error) {
	path := defaultDBPath("x-twitter-pp-cli")
	s, err := store.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening local store: %w", err)
	}
	return s, nil
}

// normalizeHandle strips '@' and lowercases. "me" is preserved as the special account name.
func normalizeHandle(h string) string {
	h = strings.TrimSpace(h)
	h = strings.TrimPrefix(h, "@")
	return strings.ToLower(h)
}

// parseRelativeDuration handles "7d", "24h", "30m", or RFC3339 timestamps.
func parseRelativeDuration(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty duration")
	}
	// Days suffix
	if strings.HasSuffix(s, "d") {
		n := 0
		if _, err := fmt.Sscanf(s, "%dd", &n); err != nil {
			return time.Time{}, fmt.Errorf("parsing %q: %w", s, err)
		}
		return time.Now().AddDate(0, 0, -n).UTC(), nil
	}
	// Standard duration
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d).UTC(), nil
	}
	// RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unrecognized duration %q (expected 7d, 24h, 30m, or RFC3339)", s)
}

// hasXFollowsData is a small helper used by doctor / dry-run paths.
func hasXFollowsData(db *sql.DB, handle string) bool {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM x_follows WHERE account_handle = ?`, handle).Scan(&n)
	return n > 0
}
