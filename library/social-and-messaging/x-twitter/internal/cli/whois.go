// Copyright 2026 dinakar-sarbada. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// whoisOutput is the aggregated agent-friendly profile view.
type whoisOutput struct {
	Profile struct {
		Handle             string `json:"handle"`
		DisplayName        string `json:"name,omitempty"`
		Bio                string `json:"bio,omitempty"`
		ProfileImageURL    string `json:"profile_image_url,omitempty"`
		FollowersCount     int64  `json:"followers_count"`
		FollowingCount     int64  `json:"following_count"`
		TweetCount         int64  `json:"tweet_count"`
		AccountCreatedAt   string `json:"account_created_at,omitempty"`
		LastTweetAt        string `json:"last_tweet_at,omitempty"`
		AccountAgeDays     int    `json:"account_age_days,omitempty"`
	} `json:"profile"`
	Engagement struct {
		TweetsPerDay         float64 `json:"tweets_per_day,omitempty"`
		AvgLikesPerTweet     float64 `json:"avg_likes,omitempty"`
		AvgRetweetsPerTweet  float64 `json:"avg_retweets,omitempty"`
		EngagementRate       float64 `json:"engagement_rate,omitempty"`
	} `json:"engagement"`
	Relationships struct {
		MutualsWithMeCount    int  `json:"mutuals_with_me_count"`
		FollowsMe             bool `json:"follows_me"`
		FollowedByMe          bool `json:"followed_by_me"`
	} `json:"relationships_with_me"`
	TopTweets []whoisTweet `json:"top_tweets,omitempty"`
	DataSource string       `json:"data_source"`
	SyncedAt   string       `json:"synced_at,omitempty"`
}

type whoisTweet struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	LikeCount int64  `json:"like_count"`
	CreatedAt string `json:"created_at,omitempty"`
}

func newWhoisCmd(flags *rootFlags) *cobra.Command {
	var topN int
	cmd := &cobra.Command{
		Use:   "whois <handle>",
		Short: "Aggregated profile + relationship + engagement summary for one user",
		Long: strings.Trim(`
One-shot user lookup. Combines profile data, derived engagement stats,
relationship-with-me flags, and top recent tweets into a single agent-friendly
view. Saves agents from making 5-6 separate calls to assemble the same picture.

Reads from the local store. If the user hasn't been synced yet, the command
returns what it has and notes which fields are missing.
`, "\n"),
		Example: strings.Trim(`
  x-twitter-pp-cli whois @paulg --json
  x-twitter-pp-cli whois @vercel --json --select profile.handle,profile.bio,engagement.tweets_per_day
`, "\n"),
		Args: cobra.MaximumNArgs(1),
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
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
			handle := normalizeHandle(args[0])
			out := &whoisOutput{}
			out.DataSource = "local"
			out.Profile.Handle = handle

			row := db.DB().QueryRow(`
				SELECT user_id, handle, COALESCE(display_name, ''), COALESCE(bio, ''),
				       COALESCE(profile_image_url, ''),
				       COALESCE(followers_count, 0), COALESCE(following_count, 0),
				       COALESCE(tweet_count, 0),
				       COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', account_created_at), ''),
				       COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', last_tweet_at), ''),
				       strftime('%Y-%m-%dT%H:%M:%SZ', synced_at)
				FROM x_users WHERE handle = ? LIMIT 1
			`, handle)
			var userID, syncedAt string
			err = row.Scan(&userID, &out.Profile.Handle, &out.Profile.DisplayName, &out.Profile.Bio,
				&out.Profile.ProfileImageURL,
				&out.Profile.FollowersCount, &out.Profile.FollowingCount, &out.Profile.TweetCount,
				&out.Profile.AccountCreatedAt, &out.Profile.LastTweetAt, &syncedAt)
			// Always carry the @-prefixed queried handle in JSON output so consumers
			// (and live-check probes) can correlate request to response.
			out.Profile.Handle = "@" + handle
			if err != nil {
				w := cmd.OutOrStdout()
				if flags.asJSON || !isTerminal(w) {
					return printJSONFiltered(w, out, flags)
				}
				fmt.Fprintf(w, "@%s not found in local store. Sync first:\n  x-twitter-pp-cli users get %s\n", handle, handle)
				return nil
			}
			out.SyncedAt = syncedAt

			// Account age
			if out.Profile.AccountCreatedAt != "" {
				if t, err := time.Parse(time.RFC3339, out.Profile.AccountCreatedAt); err == nil {
					out.Profile.AccountAgeDays = int(time.Since(t).Hours() / 24)
				}
			}

			// Relationship with me ("me" account)
			meHandle := "me"
			db.DB().QueryRow(`
				SELECT EXISTS(SELECT 1 FROM x_follows WHERE account_handle = ? AND direction = 'following' AND user_id = ?),
				       EXISTS(SELECT 1 FROM x_follows WHERE account_handle = ? AND direction = 'followers' AND user_id = ?)
			`, meHandle, userID, meHandle, userID).Scan(&out.Relationships.FollowedByMe, &out.Relationships.FollowsMe)

			// Mutuals overlap (followers of this user that are also my followers)
			db.DB().QueryRow(`
				SELECT COUNT(*) FROM x_follows fa
				JOIN x_follows fb
				  ON fa.user_id = fb.user_id
				WHERE fa.account_handle = ?
				  AND fb.account_handle = ?
				  AND fa.direction = 'followers'
				  AND fb.direction = 'followers'
			`, handle, meHandle).Scan(&out.Relationships.MutualsWithMeCount)

			// Engagement: derive from synced tweets
			tweetRows, err := db.DB().Query(`
				SELECT tweet_id, full_text, like_count, COALESCE(retweet_count, 0),
				       COALESCE(strftime('%Y-%m-%dT%H:%M:%SZ', created_at), '')
				FROM x_tweets
				WHERE author_handle = ?
				ORDER BY (like_count + 2*retweet_count + 3*reply_count) DESC
				LIMIT ?
			`, handle, topN)
			if err == nil {
				defer tweetRows.Close()
				var totalLikes, totalRetweets, count int64
				for tweetRows.Next() {
					var t whoisTweet
					var rt int64
					if err := tweetRows.Scan(&t.ID, &t.Text, &t.LikeCount, &rt, &t.CreatedAt); err != nil {
						continue
					}
					out.TopTweets = append(out.TopTweets, t)
					totalLikes += t.LikeCount
					totalRetweets += rt
					count++
				}
				if count > 0 {
					out.Engagement.AvgLikesPerTweet = round2(float64(totalLikes) / float64(count))
					out.Engagement.AvgRetweetsPerTweet = round2(float64(totalRetweets) / float64(count))
					if out.Profile.FollowersCount > 0 {
						out.Engagement.EngagementRate = round2(float64(totalLikes+totalRetweets) / float64(count) / float64(out.Profile.FollowersCount))
					}
				}
				if out.Profile.AccountAgeDays > 0 && out.Profile.TweetCount > 0 {
					out.Engagement.TweetsPerDay = round2(float64(out.Profile.TweetCount) / float64(out.Profile.AccountAgeDays))
				}
			}

			w := cmd.OutOrStdout()
			if flags.asJSON || !isTerminal(w) {
				return printJSONFiltered(w, out, flags)
			}
			renderWhoisHuman(w, out)
			return nil
		},
	}
	cmd.Flags().IntVar(&topN, "top", 5, "How many top tweets to include")
	return cmd
}

func renderWhoisHuman(w interface{ Write([]byte) (int, error) }, out *whoisOutput) {
	fmt.Fprintf(w, "@%s — %s\n", out.Profile.Handle, out.Profile.DisplayName)
	if out.Profile.Bio != "" {
		fmt.Fprintf(w, "  %s\n", out.Profile.Bio)
	}
	fmt.Fprintf(w, "  followers=%d  following=%d  tweets=%d  age=%dd\n",
		out.Profile.FollowersCount, out.Profile.FollowingCount, out.Profile.TweetCount, out.Profile.AccountAgeDays)
	if out.Engagement.TweetsPerDay > 0 {
		fmt.Fprintf(w, "  tweets/day=%.2f  avg_likes=%.1f  avg_retweets=%.1f\n",
			out.Engagement.TweetsPerDay, out.Engagement.AvgLikesPerTweet, out.Engagement.AvgRetweetsPerTweet)
	}
	rel := []string{}
	if out.Relationships.FollowsMe {
		rel = append(rel, "follows you")
	}
	if out.Relationships.FollowedByMe {
		rel = append(rel, "you follow them")
	}
	if out.Relationships.MutualsWithMeCount > 0 {
		rel = append(rel, fmt.Sprintf("%d mutuals", out.Relationships.MutualsWithMeCount))
	}
	if len(rel) > 0 {
		fmt.Fprintf(w, "  with you: %s\n", strings.Join(rel, ", "))
	}
	if len(out.TopTweets) > 0 {
		fmt.Fprintln(w, "  top tweets:")
		for _, t := range out.TopTweets {
			snip := t.Text
			if len(snip) > 100 {
				snip = snip[:97] + "..."
			}
			fmt.Fprintf(w, "    [%d♥] %s\n", t.LikeCount, snip)
		}
	}
}
