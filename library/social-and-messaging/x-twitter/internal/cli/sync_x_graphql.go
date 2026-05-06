// Copyright 2026 dinakar-sarbada. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/social-and-messaging/x-twitter/internal/store"
)

// xGraphQLOp maps a sync resource name to its GraphQL operation metadata.
// Defaults are baked in and will go stale when X rotates query IDs. To refresh:
// open x.com in DevTools, find a /i/api/graphql/<id>/<Op> request, and copy <id>.
var xGraphQLOps = map[string]struct {
	QueryID      string
	Operation    string
	FieldToggles string
	// VariablesTemplate has placeholders {USER_ID} and {COUNT} substituted at request time.
	VariablesTemplate string
	// FeaturesJSON is the boolean feature-flag bundle X requires. Empty means xWebFeaturesJSON.
	FeaturesJSON string
	// IsFollowDir is "followers" or "following", written into x_follows.direction.
	IsFollowDir string
}{
	"followers": {
		QueryID:           "f_mHnjGiLxcNKbvKG5VQZg",
		Operation:         "Followers",
		FieldToggles:      xWebFieldTogglesJSON,
		VariablesTemplate: `{"userId":"{USER_ID}","count":{COUNT},"includePromotedContent":false}`,
		IsFollowDir:       "followers",
	},
	"following": {
		QueryID:           "BdLNz9uyjufSJAveij_WZw",
		Operation:         "Following",
		FieldToggles:      xWebFieldTogglesJSON,
		VariablesTemplate: `{"userId":"{USER_ID}","count":{COUNT},"includePromotedContent":false}`,
		IsFollowDir:       "following",
	},
}

// xWebFeaturesJSON is the feature bundle X's web client currently sends for
// Followers/Following calls. The exact set rotates, but these defaults match
// the live X web bundle as of 2026-05-06.
const xWebFeaturesJSON = `{"rweb_video_screen_enabled":false,"rweb_cashtags_enabled":true,"profile_label_improvements_pcf_label_in_post_enabled":true,"responsive_web_profile_redirect_enabled":true,"rweb_tipjar_consumption_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"premium_content_api_read_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"responsive_web_grok_analyze_button_fetch_trends_enabled":false,"responsive_web_grok_analyze_post_followups_enabled":true,"rweb_cashtags_composer_attachment_enabled":true,"responsive_web_jetfuel_frame":false,"responsive_web_grok_share_attachment_enabled":true,"responsive_web_grok_annotations_enabled":true,"articles_preview_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"content_disclosure_indicator_enabled":true,"content_disclosure_ai_generated_indicator_enabled":true,"responsive_web_grok_show_grok_translated_post":false,"responsive_web_grok_analysis_button_from_backend":false,"post_ctas_fetch_enabled":true,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_grok_image_annotation_enabled":true,"responsive_web_grok_imagine_annotation_enabled":true,"responsive_web_grok_community_note_auto_translation_is_enabled":false,"responsive_web_enhance_cards_enabled":false}`

const xWebFieldTogglesJSON = `{"withPayments":true,"withAuxiliaryUserLabels":true,"withArticleRichContentState":true,"withArticlePlainText":true,"withArticleSummaryText":true,"withArticleVoiceOver":true,"withGrokAnalyze":true,"withDisallowedReplyControls":true}`

const xViewerFeaturesJSON = `{"subscriptions_upsells_api_enabled":true,"profile_label_improvements_pcf_label_in_post_enabled":true,"responsive_web_profile_redirect_enabled":true,"rweb_tipjar_consumption_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`

const xViewerFieldTogglesJSON = `{"isDelegate":false,"withPayments":true,"withAuxiliaryUserLabels":true}`

const xUserByScreenNameFeaturesJSON = `{"hidden_profile_subscriptions_enabled":true,"profile_label_improvements_pcf_label_in_post_enabled":true,"responsive_web_profile_redirect_enabled":true,"rweb_tipjar_consumption_enabled":true,"verified_phone_label_enabled":false,"subscriptions_verification_info_is_identity_verified_enabled":true,"subscriptions_verification_info_verified_since_enabled":true,"highlights_tweets_tab_ui_enabled":true,"responsive_web_twitter_article_notes_tab_enabled":true,"subscriptions_feature_can_gift_premium":true,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`

const xUserByScreenNameFieldTogglesJSON = `{"withPayments":true,"withAuxiliaryUserLabels":true}`

// isXGraphQLResource reports whether a resource name should be dispatched
// through the GraphQL handler instead of generic sync.
func isXGraphQLResource(resource string) bool {
	_, ok := xGraphQLOps[resource]
	return ok
}

// syncXGraphQLResource handles one paginated sync of an X GraphQL-shaped
// resource. It discovers the authenticated user's rest_id, paginates through
// the GraphQL timeline, and writes follow edges into the local store.
func syncXGraphQLResource(c interface {
	Get(string, map[string]string) (json.RawMessage, error)
}, db *store.Store, resource string, maxPages int) syncResult {
	started := time.Now()
	op, ok := xGraphQLOps[resource]
	if !ok {
		return xGraphQLErrorResult(resource, started, 0, fmt.Errorf("not a GraphQL resource: %s", resource))
	}

	userID, screenName, err := discoverXSelf(c)
	if err != nil {
		return xGraphQLErrorResult(resource, started, 0, fmt.Errorf("discovering self: %w", err))
	}

	totalCount := 0
	cursor := ""
	count := 100
	pageCap := maxPages
	if pageCap <= 0 {
		pageCap = 100
	}

	for page := 0; page < pageCap; page++ {
		vars, err := xGraphQLVariables(op.VariablesTemplate, userID, count, cursor)
		if err != nil {
			return xGraphQLErrorResult(resource, started, totalCount, fmt.Errorf("page %d variables: %w", page, err))
		}

		features := op.FeaturesJSON
		if features == "" {
			features = xWebFeaturesJSON
		}
		path := fmt.Sprintf("/graphql/%s/%s", op.QueryID, op.Operation)
		params := map[string]string{
			"variables": vars,
			"features":  features,
		}
		if op.FieldToggles != "" {
			params["fieldToggles"] = op.FieldToggles
		}
		data, err := c.Get(path, params)
		if err != nil {
			if page == 0 && op.IsFollowDir != "" {
				return syncXRESTFollowResource(c, db, resource, userID, screenName, op.IsFollowDir, maxPages, started, err)
			}
			return xGraphQLErrorResult(resource, started, totalCount, fmt.Errorf("page %d: %w", page, err))
		}

		pageCount, nextCursor, err := parseXFollowsResponse(data, db, screenName, op.IsFollowDir)
		if err != nil {
			return xGraphQLErrorResult(resource, started, totalCount, fmt.Errorf("page %d parse: %w", page, err))
		}
		totalCount += pageCount

		if pageCount == 0 || nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
	}

	if !humanFriendly {
		fmt.Fprintf(os.Stdout, `{"event":"sync_complete","resource":"%s","total":%d,"duration_ms":%d}`+"\n", resource, totalCount, time.Since(started).Milliseconds())
	}
	return syncResult{Resource: resource, Count: totalCount, Duration: time.Since(started)}
}

func syncXRESTFollowResource(c interface {
	Get(string, map[string]string) (json.RawMessage, error)
}, db *store.Store, resource, userID, screenName, direction string, maxPages int, started time.Time, graphQLErr error) syncResult {
	if !humanFriendly {
		fmt.Fprintf(os.Stdout, `{"event":"sync_warning","resource":"%s","reason":"graphql_fallback","message":"GraphQL first page failed (%s); falling back to X 1.1 follow list endpoint."}`+"\n",
			resource, strings.ReplaceAll(graphQLErr.Error(), `"`, `\"`))
	}

	path := "/1.1/followers/list.json"
	if direction == "following" {
		path = "/1.1/friends/following/list.json"
	}

	accountHandles := xAccountHandles(screenName)
	pageCap := maxPages
	if pageCap <= 0 {
		pageCap = 100
	}

	totalCount := 0
	cursor := "-1"
	for page := 0; page < pageCap; page++ {
		params := map[string]string{
			"user_id":               userID,
			"count":                 "200",
			"skip_status":           "true",
			"include_user_entities": "false",
		}
		if cursor != "" {
			params["cursor"] = cursor
		}
		data, err := c.Get(path, params)
		if err != nil {
			return xGraphQLErrorResult(resource, started, totalCount, fmt.Errorf("GraphQL first page failed (%v); REST fallback page %d: %w", graphQLErr, page, err))
		}
		pageCount, nextCursor, err := parseXRESTFollowsResponse(data, db, accountHandles, direction)
		if err != nil {
			return xGraphQLErrorResult(resource, started, totalCount, fmt.Errorf("REST fallback page %d parse: %w", page, err))
		}
		totalCount += pageCount
		if pageCount == 0 || nextCursor == "" || nextCursor == "0" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
	}

	if !humanFriendly {
		fmt.Fprintf(os.Stdout, `{"event":"sync_complete","resource":"%s","total":%d,"duration_ms":%d}`+"\n", resource, totalCount, time.Since(started).Milliseconds())
	}
	return syncResult{Resource: resource, Count: totalCount, Duration: time.Since(started)}
}

func xGraphQLErrorResult(resource string, started time.Time, count int, err error) syncResult {
	if !humanFriendly {
		fmt.Fprintf(os.Stdout, `{"event":"sync_error","resource":"%s","error":"%s"}`+"\n", resource, strings.ReplaceAll(err.Error(), `"`, `\"`))
	}
	return syncResult{Resource: resource, Count: count, Err: err, Duration: time.Since(started)}
}

func xGraphQLVariables(template, userID string, count int, cursor string) (string, error) {
	vars := strings.NewReplacer("{USER_ID}", userID, "{COUNT}", fmt.Sprintf("%d", count)).Replace(template)
	var body map[string]any
	if err := json.Unmarshal([]byte(vars), &body); err != nil {
		return "", err
	}
	if cursor != "" {
		body["cursor"] = cursor
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// discoverXSelf calls X to learn the authenticated user's rest_id and screen_name.
func discoverXSelf(c interface {
	Get(string, map[string]string) (json.RawMessage, error)
}) (userID, screenName string, err error) {
	userID, screenName, viewerErr := discoverXSelfViaViewer(c)
	if viewerErr == nil {
		return userID, screenName, nil
	}

	settings, err := c.Get("/1.1/account/settings.json", nil)
	if err != nil {
		return "", "", fmt.Errorf("%v; settings: %w", viewerErr, err)
	}
	var settingsBody struct {
		ScreenName string `json:"screen_name"`
	}
	if err := json.Unmarshal(settings, &settingsBody); err != nil {
		return "", "", fmt.Errorf("decode settings: %w", err)
	}
	if settingsBody.ScreenName == "" {
		return "", "", fmt.Errorf("settings response had no screen_name")
	}
	screenName = settingsBody.ScreenName

	const userByScreenNameQueryID = "IGgvgiOx4QZndDHuD3x9TQ"
	vars, err := json.Marshal(map[string]string{"screen_name": screenName})
	if err != nil {
		return "", "", fmt.Errorf("encode UserByScreenName variables: %w", err)
	}
	resp, err := c.Get(
		fmt.Sprintf("/graphql/%s/UserByScreenName", userByScreenNameQueryID),
		map[string]string{
			"variables":    string(vars),
			"features":     xUserByScreenNameFeaturesJSON,
			"fieldToggles": xUserByScreenNameFieldTogglesJSON,
		},
	)
	if err != nil {
		return "", "", fmt.Errorf("user-by-screen-name: %w", err)
	}

	var ubn struct {
		Data struct {
			User struct {
				Result struct {
					RestID string `json:"rest_id"`
				} `json:"result"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &ubn); err != nil {
		return "", "", fmt.Errorf("decode UserByScreenName: %w", err)
	}
	if ubn.Data.User.Result.RestID == "" {
		return "", "", fmt.Errorf("UserByScreenName had no rest_id (response: %s)", string(resp[:min(200, len(resp))]))
	}
	return ubn.Data.User.Result.RestID, screenName, nil
}

func discoverXSelfViaViewer(c interface {
	Get(string, map[string]string) (json.RawMessage, error)
}) (userID, screenName string, err error) {
	const viewerQueryID = "_8ClT24oZ8tpylf_OSuNdg"
	resp, err := c.Get(
		fmt.Sprintf("/graphql/%s/Viewer", viewerQueryID),
		map[string]string{
			"variables":    "{}",
			"features":     xViewerFeaturesJSON,
			"fieldToggles": xViewerFieldTogglesJSON,
		},
	)
	if err != nil {
		return "", "", fmt.Errorf("viewer: %w", err)
	}

	var viewer struct {
		Data struct {
			Viewer struct {
				UserResults struct {
					Result struct {
						RestID string `json:"rest_id"`
						Core   struct {
							ScreenName string `json:"screen_name"`
						} `json:"core"`
						Legacy struct {
							ScreenName string `json:"screen_name"`
						} `json:"legacy"`
					} `json:"result"`
				} `json:"user_results"`
			} `json:"viewer"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &viewer); err != nil {
		return "", "", fmt.Errorf("decode Viewer: %w", err)
	}
	result := viewer.Data.Viewer.UserResults.Result
	screenName = result.Core.ScreenName
	if screenName == "" {
		screenName = result.Legacy.ScreenName
	}
	if result.RestID == "" || screenName == "" {
		return "", "", fmt.Errorf("Viewer had no rest_id/screen_name (response: %s)", string(resp[:min(200, len(resp))]))
	}
	return result.RestID, screenName, nil
}

// parseXFollowsResponse walks a Followers/Following GraphQL response, writes
// each user into x_follows, and returns count_written plus the next bottom cursor.
func parseXFollowsResponse(data json.RawMessage, db *store.Store, accountHandle, direction string) (count int, nextCursor string, err error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return 0, "", fmt.Errorf("decode response: %w", err)
	}

	user, _ := mapPath(root, "data", "user", "result")
	timeline, _ := mapPath(user, "timeline", "timeline")
	if timeline == nil {
		timeline, _ = mapPath(user, "timeline_response", "timeline")
	}
	if timeline == nil {
		return 0, "", fmt.Errorf("response shape unexpected: no timeline node")
	}

	accountHandles := xAccountHandles(accountHandle)

	instructions, _ := timeline["instructions"].([]any)
	for _, ins := range instructions {
		insMap, _ := ins.(map[string]any)
		if insMap == nil {
			continue
		}
		insType, _ := insMap["type"].(string)
		if insType != "" && insType != "TimelineAddEntries" {
			continue
		}
		entries, _ := insMap["entries"].([]any)
		for _, e := range entries {
			em, _ := e.(map[string]any)
			if em == nil {
				continue
			}
			written, cursor, writeErr := parseXTimelineContent(em["content"], db, accountHandles, direction)
			if writeErr != nil {
				return count, nextCursor, writeErr
			}
			count += written
			if cursor != "" {
				nextCursor = cursor
			}
		}
	}
	return count, nextCursor, nil
}

func parseXRESTFollowsResponse(data json.RawMessage, db *store.Store, accountHandles []string, direction string) (count int, nextCursor string, err error) {
	var body struct {
		Users []struct {
			IDStr      string `json:"id_str"`
			ScreenName string `json:"screen_name"`
			Name       string `json:"name"`
		} `json:"users"`
		NextCursorStr string `json:"next_cursor_str"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return 0, "", fmt.Errorf("decode response: %w", err)
	}
	for _, user := range body.Users {
		if user.IDStr == "" {
			continue
		}
		if err := writeXFollowUser(db, accountHandles, direction, user.IDStr, user.ScreenName, user.Name); err != nil {
			return count, body.NextCursorStr, err
		}
		count++
	}
	return count, body.NextCursorStr, nil
}

func parseXTimelineContent(contentAny any, db *store.Store, accountHandles []string, direction string) (count int, nextCursor string, err error) {
	content, _ := contentAny.(map[string]any)
	if content == nil {
		return 0, "", nil
	}

	contentType, _ := content["__typename"].(string)
	if contentType == "" {
		contentType, _ = content["entryType"].(string)
	}
	switch contentType {
	case "TimelineTimelineItem":
		itemContent, _ := content["itemContent"].(map[string]any)
		written, err := writeXTimelineUser(itemContent, db, accountHandles, direction)
		return written, "", err
	case "TimelineTimelineCursor":
		cursorType, _ := content["cursorType"].(string)
		cursorValue, _ := content["value"].(string)
		if cursorType == "Bottom" {
			return 0, cursorValue, nil
		}
	case "TimelineTimelineModule":
		items, _ := content["items"].([]any)
		for _, item := range items {
			itemMap, _ := item.(map[string]any)
			if itemMap == nil {
				continue
			}
			inner, _ := itemMap["item"].(map[string]any)
			written, cursor, itemErr := parseXTimelineContent(inner["content"], db, accountHandles, direction)
			if itemErr != nil {
				return count, nextCursor, itemErr
			}
			count += written
			if cursor != "" {
				nextCursor = cursor
			}
		}
	}
	return count, nextCursor, nil
}

func writeXTimelineUser(itemContent map[string]any, db *store.Store, accountHandles []string, direction string) (int, error) {
	if itemContent == nil {
		return 0, nil
	}
	itemType, _ := itemContent["itemType"].(string)
	if itemType != "" && itemType != "TimelineUser" {
		return 0, nil
	}
	userResults, _ := itemContent["user_results"].(map[string]any)
	if userResults == nil {
		return 0, nil
	}
	result, _ := userResults["result"].(map[string]any)
	if result == nil {
		return 0, nil
	}
	restID, _ := result["rest_id"].(string)
	if restID == "" {
		return 0, nil
	}
	legacy, _ := result["legacy"].(map[string]any)
	screenName, _ := legacy["screen_name"].(string)
	displayName, _ := legacy["name"].(string)
	// X moved profile fields into a `core` block in newer GraphQL responses;
	// `legacy` still exists for backward compat but is empty for many users.
	// Fall back to `core` when legacy is missing values so x_users receives
	// full handles + display names that the relationship analytics surface.
	if screenName == "" || displayName == "" {
		core, _ := result["core"].(map[string]any)
		if core != nil {
			if screenName == "" {
				if v, ok := core["screen_name"].(string); ok {
					screenName = v
				}
			}
			if displayName == "" {
				if v, ok := core["name"].(string); ok {
					displayName = v
				}
			}
		}
	}

	if err := writeXFollowUser(db, accountHandles, direction, restID, screenName, displayName); err != nil {
		return 0, err
	}
	return 1, nil
}

func writeXFollowUser(db *store.Store, accountHandles []string, direction, userID, screenName, displayName string) error {
	ctx := context.Background()
	if err := db.UpsertXUser(ctx, userID, screenName, displayName, ""); err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}
	for _, accountHandle := range accountHandles {
		if err := db.UpsertXFollow(ctx, accountHandle, direction, userID, screenName); err != nil {
			return fmt.Errorf("upsert follow: %w", err)
		}
	}
	return nil
}

func xAccountHandles(accountHandle string) []string {
	accountHandles := []string{"me"}
	if normalized := normalizeHandle(accountHandle); normalized != "" && normalized != "me" {
		accountHandles = append(accountHandles, normalized)
	}
	return accountHandles
}

// mapPath walks a nested map[string]any tree by string keys.
func mapPath(m any, keys ...string) (map[string]any, bool) {
	cur, ok := m.(map[string]any)
	if !ok {
		return nil, false
	}
	for _, k := range keys {
		next, ok := cur[k].(map[string]any)
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
