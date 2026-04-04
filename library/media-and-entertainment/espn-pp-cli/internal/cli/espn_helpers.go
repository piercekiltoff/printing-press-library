package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/espn-pp-cli/internal/espn"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/espn-pp-cli/internal/store"
)

type leagueSpec struct {
	Key    string
	Sport  string
	League string
}

type teamCandidate struct {
	ID           string
	Name         string
	DisplayName  string
	Abbreviation string
	Data         json.RawMessage
}

type athleteCandidate struct {
	ID          string
	Name        string
	DisplayName string
	TeamID      string
	Data        json.RawMessage
}

func newESPNClient(flags *rootFlags) *espn.ESPN {
	client := espn.NewWithTimeout(flags.timeout)
	client.SetDryRun(flags.dryRun)
	client.SetNoCache(flags.noCache)
	return client
}

func resolveLeagueSpec(input string) (leagueSpec, error) {
	sport, league, err := espn.ResolveSportLeague(input)
	if err != nil {
		return leagueSpec{}, usageErr(err)
	}
	return leagueSpec{Key: strings.ToLower(strings.TrimSpace(input)), Sport: sport, League: league}, nil
}

func defaultStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "espn-pp-cli", "data.db")
}

func openStoreIfExists(path string) (*store.Store, error) {
	if path == "" {
		path = defaultStorePath()
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return store.Open(path)
}

func normalizeOutput(data json.RawMessage) json.RawMessage {
	if len(data) == 0 {
		return json.RawMessage("null")
	}
	return data
}

func marshalRaw(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

func parseJSON(data json.RawMessage) any {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil
	}
	return v
}

func majorLeagueKeys() []string {
	return []string{"nfl", "nba", "mlb", "nhl"}
}

func allLeagueKeys() []string {
	keys := make([]string, 0, len(espn.DefaultLeagues))
	for key := range espn.DefaultLeagues {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isNumericID(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return false
	}
	_, err := strconv.ParseInt(input, 10, 64)
	return err == nil
}

func extractMapString(obj map[string]any, path string) string {
	var cur any = obj
	for _, part := range strings.Split(path, ".") {
		switch typed := cur.(type) {
		case map[string]any:
			value, ok := typed[part]
			if !ok {
				return ""
			}
			cur = value
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(typed) {
				return ""
			}
			cur = typed[idx]
		default:
			return ""
		}
	}
	switch value := cur.(type) {
	case string:
		return strings.TrimSpace(value)
	case float64:
		return strconv.FormatInt(int64(value), 10)
	case json.Number:
		return value.String()
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "<nil>" {
			return ""
		}
		return text
	}
}

func bestString(obj map[string]any, fields ...string) string {
	for _, field := range fields {
		if value := extractMapString(obj, field); value != "" {
			return value
		}
	}
	return ""
}

func walkJSON(node any, visit func(map[string]any)) {
	switch typed := node.(type) {
	case map[string]any:
		visit(typed)
		for _, value := range typed {
			walkJSON(value, visit)
		}
	case []any:
		for _, item := range typed {
			walkJSON(item, visit)
		}
	}
}

func extractTeamCandidates(data json.RawMessage) []teamCandidate {
	root := parseJSON(data)
	if root == nil {
		return nil
	}

	seen := map[string]bool{}
	var teams []teamCandidate
	walkJSON(root, func(obj map[string]any) {
		id := bestString(obj, "id", "uid")
		name := bestString(obj, "displayName", "shortDisplayName", "name", "nickname")
		abbr := bestString(obj, "abbreviation")
		if id == "" || (name == "" && abbr == "") {
			return
		}
		key := id
		if key == "" {
			key = strings.ToLower(name)
		}
		if seen[key] {
			return
		}
		seen[key] = true
		teams = append(teams, teamCandidate{
			ID:           id,
			Name:         bestString(obj, "name", "shortDisplayName", "displayName"),
			DisplayName:  name,
			Abbreviation: abbr,
			Data:         marshalRaw(obj),
		})
	})
	return teams
}

func extractAthleteCandidates(data json.RawMessage) []athleteCandidate {
	root := parseJSON(data)
	if root == nil {
		return nil
	}

	seen := map[string]bool{}
	var athletes []athleteCandidate
	walkJSON(root, func(obj map[string]any) {
		id := bestString(obj, "id", "uid")
		name := bestString(obj, "displayName", "fullName", "name", "shortName")
		if id == "" || name == "" {
			return
		}
		if bestString(obj, "headline", "title") != "" {
			return
		}
		if seen[id] {
			return
		}
		seen[id] = true
		athletes = append(athletes, athleteCandidate{
			ID:          id,
			Name:        bestString(obj, "fullName", "name", "displayName"),
			DisplayName: name,
			TeamID:      bestString(obj, "team.id"),
			Data:        marshalRaw(obj),
		})
	})
	return athletes
}

func extractEventPayloads(data json.RawMessage) []json.RawMessage {
	root := parseJSON(data)
	if root == nil {
		return nil
	}
	seen := map[string]bool{}
	var events []json.RawMessage
	walkJSON(root, func(obj map[string]any) {
		id := bestString(obj, "id", "uid")
		date := bestString(obj, "date")
		name := bestString(obj, "name", "shortName", "displayName")
		if id == "" || (date == "" && name == "") {
			return
		}
		if _, ok := obj["competitions"]; !ok && bestString(obj, "status.type.name", "status.type.description") == "" {
			return
		}
		if seen[id] {
			return
		}
		seen[id] = true
		events = append(events, marshalRaw(obj))
	})
	return events
}

func extractStandingsPayloads(data json.RawMessage) []json.RawMessage {
	root := parseJSON(data)
	if root == nil {
		return nil
	}
	seen := map[string]bool{}
	var standings []json.RawMessage
	walkJSON(root, func(obj map[string]any) {
		teamID := bestString(obj, "team.id")
		if teamID == "" {
			return
		}
		if _, hasStats := obj["stats"]; !hasStats {
			if _, hasRecords := obj["records"]; !hasRecords {
				return
			}
		}
		key := teamID + ":" + bestString(obj, "id")
		if seen[key] {
			return
		}
		seen[key] = true
		standings = append(standings, marshalRaw(obj))
	})
	return standings
}

func extractNewsPayloads(data json.RawMessage) []json.RawMessage {
	root := parseJSON(data)
	if root == nil {
		return nil
	}
	seen := map[string]bool{}
	var articles []json.RawMessage
	walkJSON(root, func(obj map[string]any) {
		headline := bestString(obj, "headline", "title")
		id := bestString(obj, "id", "guid", "linkText")
		if headline == "" || id == "" {
			return
		}
		if seen[id] {
			return
		}
		seen[id] = true
		articles = append(articles, marshalRaw(obj))
	})
	return articles
}

func extractScoreRows(data json.RawMessage, leagueKey string) []map[string]any {
	rows := make([]map[string]any, 0)
	for _, raw := range extractEventPayloads(data) {
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}
		row := map[string]any{
			"league": leagueKey,
			"id":     bestString(obj, "id", "uid"),
			"name":   bestString(obj, "shortName", "name", "displayName"),
			"date":   bestString(obj, "date"),
			"status": bestString(obj, "status.type.description", "status.type.name", "status.description"),
		}
		if competitors, ok := obj["competitions"].([]any); ok && len(competitors) > 0 {
			if comp, ok := competitors[0].(map[string]any); ok {
				if teams, ok := comp["competitors"].([]any); ok {
					for _, item := range teams {
						team, ok := item.(map[string]any)
						if !ok {
							continue
						}
						side := bestString(team, "homeAway")
						prefix := side
						if prefix == "" {
							prefix = "team"
						}
						row[prefix+"TeamID"] = bestString(team, "team.id", "id")
						row[prefix+"Team"] = bestString(team, "team.abbreviation", "team.displayName", "team.name")
						row[prefix+"Score"] = bestString(team, "score")
					}
				}
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func scoreRowsForLeague(client *espn.ESPN, leagueKey, date string) ([]map[string]any, error) {
	spec, err := resolveLeagueSpec(leagueKey)
	if err != nil {
		return nil, err
	}
	data, err := client.Scoreboard(spec.Sport, spec.League, date)
	if err != nil {
		return nil, err
	}
	return extractScoreRows(normalizeOutput(data), spec.Key), nil
}

func findTeamInStore(db *store.Store, spec leagueSpec, input string) (*teamCandidate, error) {
	if db == nil {
		return nil, nil
	}
	if byAbbrev, err := db.GetTeamByAbbreviation(spec.Sport, spec.League, input); err == nil && len(byAbbrev) > 0 {
		candidates := extractTeamCandidates(byAbbrev)
		if len(candidates) > 0 {
			return &candidates[0], nil
		}
	}
	results, err := db.SearchTeams(input, 10)
	if err != nil {
		return nil, err
	}
	return pickBestTeam(input, extractTeamResults(results)), nil
}

func findTeamLive(client *espn.ESPN, spec leagueSpec, input string) (*teamCandidate, error) {
	data, err := client.Teams(spec.Sport, spec.League)
	if err != nil {
		return nil, err
	}
	return pickBestTeam(input, extractTeamCandidates(data)), nil
}

func resolveTeam(client *espn.ESPN, db *store.Store, spec leagueSpec, input string) (*teamCandidate, error) {
	if isNumericID(input) {
		return &teamCandidate{ID: input}, nil
	}
	if team, err := findTeamInStore(db, spec, input); err != nil {
		return nil, err
	} else if team != nil {
		return team, nil
	}
	return findTeamLive(client, spec, input)
}

func extractTeamResults(items []json.RawMessage) []teamCandidate {
	var out []teamCandidate
	for _, item := range items {
		out = append(out, extractTeamCandidates(item)...)
	}
	return out
}

func pickBestTeam(input string, candidates []teamCandidate) *teamCandidate {
	if len(candidates) == 0 {
		return nil
	}
	input = strings.ToLower(strings.TrimSpace(input))
	var contains *teamCandidate
	for i := range candidates {
		candidate := &candidates[i]
		names := []string{
			strings.ToLower(candidate.ID),
			strings.ToLower(candidate.Abbreviation),
			strings.ToLower(candidate.Name),
			strings.ToLower(candidate.DisplayName),
		}
		for _, name := range names {
			if name == "" {
				continue
			}
			if name == input {
				return candidate
			}
			if contains == nil && strings.Contains(name, input) {
				contains = candidate
			}
		}
	}
	return contains
}

func findAthleteInStore(db *store.Store, input string) (*athleteCandidate, error) {
	if db == nil {
		return nil, nil
	}
	results, err := db.SearchAthletes(input, 10)
	if err != nil {
		return nil, err
	}
	return pickBestAthlete(input, extractAthleteResults(results)), nil
}

func findAthleteLive(client *espn.ESPN, input string) (*athleteCandidate, error) {
	data, err := client.Search(input)
	if err != nil {
		return nil, err
	}
	// Parse ESPN search response format: {results: [{type: "player", contents: [{uid: "s:40~l:46~a:1966", displayName: "..."}]}]}
	candidates := extractAthletesFromSearch(data)
	if len(candidates) > 0 {
		return pickBestAthlete(input, candidates), nil
	}
	// Fallback to generic walk
	return pickBestAthlete(input, extractAthleteCandidates(data)), nil
}

// extractAthletesFromSearch parses ESPN's search API response into athlete candidates.
func extractAthletesFromSearch(data json.RawMessage) []athleteCandidate {
	var resp struct {
		Results []struct {
			Type     string `json:"type"`
			Contents []struct {
				UID         string `json:"uid"`
				DisplayName string `json:"displayName"`
				Description string `json:"description"`
			} `json:"contents"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil
	}

	var out []athleteCandidate
	for _, group := range resp.Results {
		if group.Type != "player" {
			continue
		}
		for _, item := range group.Contents {
			athleteID := ""
			if item.UID != "" {
				for _, part := range strings.Split(item.UID, "~") {
					if strings.HasPrefix(part, "a:") {
						athleteID = strings.TrimPrefix(part, "a:")
					}
				}
			}
			if athleteID != "" {
				out = append(out, athleteCandidate{
					ID:          athleteID,
					DisplayName: item.DisplayName,
					Name:        item.DisplayName,
				})
			}
		}
	}
	return out
}

func resolveAthlete(client *espn.ESPN, db *store.Store, input string) (*athleteCandidate, error) {
	if isNumericID(input) {
		return &athleteCandidate{ID: input}, nil
	}
	if athlete, err := findAthleteInStore(db, input); err != nil {
		return nil, err
	} else if athlete != nil {
		return athlete, nil
	}
	return findAthleteLive(client, input)
}

func extractAthleteResults(items []json.RawMessage) []athleteCandidate {
	var out []athleteCandidate
	for _, item := range items {
		out = append(out, extractAthleteCandidates(item)...)
	}
	return out
}

func pickBestAthlete(input string, candidates []athleteCandidate) *athleteCandidate {
	if len(candidates) == 0 {
		return nil
	}
	input = strings.ToLower(strings.TrimSpace(input))
	var contains *athleteCandidate
	for i := range candidates {
		candidate := &candidates[i]
		names := []string{
			strings.ToLower(candidate.ID),
			strings.ToLower(candidate.Name),
			strings.ToLower(candidate.DisplayName),
		}
		for _, name := range names {
			if name == "" {
				continue
			}
			if name == input {
				return candidate
			}
			if contains == nil && strings.Contains(name, input) {
				contains = candidate
			}
		}
	}
	return contains
}

func extractArrayAt(data json.RawMessage, key string) []json.RawMessage {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil
	}
	raw, ok := obj[key]
	if !ok {
		return nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	return items
}

func filterRawItemsByTerms(items []json.RawMessage, terms ...string) []json.RawMessage {
	cleanTerms := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term != "" {
			cleanTerms = append(cleanTerms, term)
		}
	}
	if len(cleanTerms) == 0 {
		return items
	}
	var filtered []json.RawMessage
	for _, item := range items {
		text := strings.ToLower(string(item))
		for _, term := range cleanTerms {
			if strings.Contains(text, term) {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

func seasonLabel(data json.RawMessage) string {
	root := parseJSON(data)
	if root == nil {
		return time.Now().Format("2006")
	}
	var season string
	walkJSON(root, func(obj map[string]any) {
		if season != "" {
			return
		}
		season = bestString(obj, "season.year", "season.displayName", "season.name")
	})
	if season == "" {
		season = time.Now().Format("2006")
	}
	return season
}
