// Copyright 2026 dinakar-sarbada. Licensed under Apache-2.0. See LICENSE.

package store

import (
	"context"
	"fmt"
)

// UpsertXTweet inserts or replaces a tweet row in x_tweets. Used by archive
// import and live sync paths. Centralizing the SQL keeps the schema's column
// order coupled to one place — adding a new column means updating only this
// method, not every caller.
func (s *Store) UpsertXTweet(ctx context.Context, tweetID, authorID, authorHandle, fullText, lang, createdAtIso string, likeCount, retweetCount, replyCount int64) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO x_tweets
		  (tweet_id, author_id, author_handle, full_text, lang, like_count, retweet_count, reply_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tweetID, authorID, authorHandle, fullText, lang, likeCount, retweetCount, replyCount, createdAtIso)
	if err != nil {
		return fmt.Errorf("upsert x_tweet %s: %w", tweetID, err)
	}
	return nil
}

// UpsertXUser inserts or replaces a user record in x_users.
func (s *Store) UpsertXUser(ctx context.Context, userID, handle, displayName, accountCreatedAtIso string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO x_users
		  (user_id, handle, display_name, account_created_at)
		VALUES (?, ?, ?, ?)
	`, userID, handle, displayName, accountCreatedAtIso)
	if err != nil {
		return fmt.Errorf("upsert x_user %s: %w", userID, err)
	}
	return nil
}

// UpsertXFollow records a single follow edge for an account in a given direction.
func (s *Store) UpsertXFollow(ctx context.Context, accountHandle, direction, userID, handle string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO x_follows (account_handle, direction, user_id, handle)
		VALUES (?, ?, ?, ?)
	`, accountHandle, direction, userID, handle)
	if err != nil {
		return fmt.Errorf("upsert x_follow %s/%s/%s: %w", accountHandle, direction, userID, err)
	}
	return nil
}

// SearchXTweets runs a full-text search against the x_tweets_fts index and
// returns the rows ranked by weighted engagement. Limit controls max results.
func (s *Store) SearchXTweets(ctx context.Context, query string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.tweet_id, COALESCE(t.author_handle, ''), t.full_text,
		       t.like_count, t.retweet_count, t.reply_count
		FROM x_tweets_fts
		JOIN x_tweets t ON t.rowid = x_tweets_fts.rowid
		WHERE x_tweets_fts MATCH ?
		ORDER BY (t.like_count + 2*t.retweet_count + 3*t.reply_count) DESC
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search x_tweets: %w", err)
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, handle, text string
		var likes, retweets, replies int64
		if err := rows.Scan(&id, &handle, &text, &likes, &retweets, &replies); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"tweet_id":     id,
			"author":       handle,
			"text":         text,
			"like_count":   likes,
			"retweet_count": retweets,
			"reply_count":  replies,
		})
	}
	return out, nil
}

// SearchXUsers runs a substring search over x_users.handle and display_name.
func (s *Store) SearchXUsers(ctx context.Context, query string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 50
	}
	pattern := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, handle, COALESCE(display_name, ''), COALESCE(bio, '')
		FROM x_users
		WHERE handle LIKE ? OR display_name LIKE ?
		ORDER BY handle
		LIMIT ?
	`, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("search x_users: %w", err)
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var userID, handle, displayName, bio string
		if err := rows.Scan(&userID, &handle, &displayName, &bio); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"user_id": userID,
			"handle":  handle,
			"name":    displayName,
			"bio":     bio,
		})
	}
	return out, nil
}
