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

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/espn-pp-cli/internal/store"
)

type eventTeamSnapshot struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Abbreviation string `json:"abbreviation,omitempty"`
	HomeAway     string `json:"home_away,omitempty"`
	Score        string `json:"score,omitempty"`
	Record       string `json:"record,omitempty"`
}

type eventSnapshot struct {
	ID     string            `json:"id,omitempty"`
	Name   string            `json:"name,omitempty"`
	Date   string            `json:"date,omitempty"`
	Status string            `json:"status,omitempty"`
	Home   eventTeamSnapshot `json:"home"`
	Away   eventTeamSnapshot `json:"away"`
}

type standingSnapshot struct {
	TeamID   string  `json:"team_id,omitempty"`
	TeamName string  `json:"team_name,omitempty"`
	Wins     int     `json:"wins"`
	Losses   int     `json:"losses"`
	Ties     int     `json:"ties"`
	WinPct   float64 `json:"win_pct"`
	Record   string  `json:"record,omitempty"`
	Rank     string  `json:"rank,omitempty"`
}

type watchItem struct {
	League       string `json:"league"`
	Sport        string `json:"sport"`
	TeamID       string `json:"team_id"`
	Name         string `json:"name"`
	DisplayName  string `json:"display_name,omitempty"`
	Abbreviation string `json:"abbreviation,omitempty"`
	AddedAt      string `json:"added_at"`
}

type watchlistFile struct {
	Items []watchItem `json:"items"`
}

func toJSONObject(data json.RawMessage) map[string]any {
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil
	}
	return obj
}

func extractEventSnapshots(data json.RawMessage) []eventSnapshot {
	rawEvents := extractEventPayloads(data)
	snapshots := make([]eventSnapshot, 0, len(rawEvents))
	for _, raw := range rawEvents {
		if snapshot, ok := extractEventSnapshotFromRaw(raw); ok {
			snapshots = append(snapshots, snapshot)
		}
	}
	return snapshots
}

func extractEventSnapshotFromRaw(data json.RawMessage) (eventSnapshot, bool) {
	obj := toJSONObject(data)
	if obj == nil {
		return eventSnapshot{}, false
	}
	snapshot := eventSnapshot{
		ID:     bestString(obj, "id", "uid"),
		Name:   bestString(obj, "shortName", "name", "displayName"),
		Date:   bestString(obj, "date"),
		Status: bestString(obj, "status.type.description", "status.type.name", "status.description"),
	}
	competitors, _ := digCompetitors(obj)
	for _, competitor := range competitors {
		item := eventTeamSnapshot{
			ID:           bestString(competitor, "team.id", "id"),
			Name:         bestString(competitor, "team.displayName", "team.shortDisplayName", "team.name", "displayName", "name"),
			Abbreviation: bestString(competitor, "team.abbreviation", "abbreviation"),
			HomeAway:     bestString(competitor, "homeAway"),
			Score:        bestString(competitor, "score"),
			Record:       competitorRecord(competitor),
		}
		switch item.HomeAway {
		case "home":
			snapshot.Home = item
		case "away":
			snapshot.Away = item
		default:
			if snapshot.Away.ID == "" {
				item.HomeAway = "away"
				snapshot.Away = item
			} else if snapshot.Home.ID == "" {
				item.HomeAway = "home"
				snapshot.Home = item
			}
		}
	}
	if snapshot.ID == "" || (snapshot.Home.ID == "" && snapshot.Away.ID == "") {
		return eventSnapshot{}, false
	}
	return snapshot, true
}

func digCompetitors(obj map[string]any) ([]map[string]any, bool) {
	candidates := []string{"competitions.0.competitors", "header.competitions.0.competitors", "competitors"}
	for _, path := range candidates {
		value, ok := digAnyValue(obj, path)
		if !ok {
			continue
		}
		items, ok := value.([]any)
		if !ok {
			continue
		}
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if competitor, ok := item.(map[string]any); ok {
				out = append(out, competitor)
			}
		}
		if len(out) > 0 {
			return out, true
		}
	}
	return nil, false
}

func digAnyValue(obj map[string]any, path string) (any, bool) {
	var current any = obj
	for _, part := range strings.Split(path, ".") {
		switch typed := current.(type) {
		case map[string]any:
			value, ok := typed[part]
			if !ok {
				return nil, false
			}
			current = value
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func competitorRecord(obj map[string]any) string {
	if value := bestString(obj, "records.0.summary", "record.summary"); value != "" {
		return value
	}
	if records, ok := obj["records"].([]any); ok {
		for _, item := range records {
			record, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if summary := bestString(record, "summary", "displayValue", "value"); summary != "" {
				return summary
			}
		}
	}
	return ""
}

func teamTerms(team *teamCandidate, fallback string) []string {
	if team == nil {
		return []string{fallback}
	}
	return uniqueNonEmptyStrings(
		team.ID,
		team.Abbreviation,
		team.Name,
		team.DisplayName,
		fallback,
	)
}

func filterEventsForTeams(events []eventSnapshot, teamAID, teamBID string) []eventSnapshot {
	var matches []eventSnapshot
	for _, event := range events {
		ids := []string{event.Home.ID, event.Away.ID}
		if containsString(ids, teamAID) && containsString(ids, teamBID) {
			matches = append(matches, event)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Date > matches[j].Date
	})
	return matches
}

func filterPastEvents(events []eventSnapshot) []eventSnapshot {
	now := time.Now()
	var out []eventSnapshot
	for _, event := range events {
		ts, err := time.Parse(time.RFC3339, event.Date)
		if err != nil {
			out = append(out, event)
			continue
		}
		if !ts.After(now) {
			out = append(out, event)
		}
	}
	return out
}

func extractStandingsIndex(data json.RawMessage) map[string]standingSnapshot {
	index := map[string]standingSnapshot{}
	for _, raw := range extractStandingsPayloads(data) {
		obj := toJSONObject(raw)
		if obj == nil {
			continue
		}
		teamID := bestString(obj, "team.id")
		if teamID == "" {
			continue
		}
		snapshot := standingSnapshot{
			TeamID:   teamID,
			TeamName: bestString(obj, "team.displayName", "team.name"),
			Record:   bestString(obj, "records.0.summary"),
			Rank:     bestString(obj, "stats.0.displayValue", "stats.0.value", "team.rank"),
		}
		applyStandingStats(&snapshot, obj)
		if snapshot.Record == "" {
			record := formatRecord(snapshot.Wins, snapshot.Losses, snapshot.Ties)
			if record != "" {
				snapshot.Record = record
			}
		}
		index[teamID] = snapshot
	}
	return index
}

func applyStandingStats(snapshot *standingSnapshot, obj map[string]any) {
	applyStatList := func(items []any) {
		for _, item := range items {
			stat, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := strings.ToLower(bestString(stat, "name", "abbreviation", "displayName", "shortDisplayName", "type"))
			value := firstNonEmpty(bestString(stat, "value"), bestString(stat, "displayValue"))
			switch key {
			case "wins", "win":
				snapshot.Wins = atoiString(value)
			case "losses", "loss":
				snapshot.Losses = atoiString(value)
			case "ties", "tie":
				snapshot.Ties = atoiString(value)
			case "winpercent", "winpct", "pct":
				snapshot.WinPct = parseRatio(value)
			case "rank", "playoffseed", "seed":
				if snapshot.Rank == "" {
					snapshot.Rank = value
				}
			}
		}
	}

	if stats, ok := obj["stats"].([]any); ok {
		applyStatList(stats)
	}
	if records, ok := obj["records"].([]any); ok {
		for _, item := range records {
			record, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if snapshot.Record == "" {
				snapshot.Record = bestString(record, "summary")
			}
			if stats, ok := record["stats"].([]any); ok {
				applyStatList(stats)
			}
		}
	}
	if snapshot.WinPct == 0 {
		total := snapshot.Wins + snapshot.Losses + snapshot.Ties
		if total > 0 {
			snapshot.WinPct = (float64(snapshot.Wins) + 0.5*float64(snapshot.Ties)) / float64(total)
		}
	}
}

func formatRecord(wins, losses, ties int) string {
	if wins == 0 && losses == 0 && ties == 0 {
		return ""
	}
	if ties > 0 {
		return fmt.Sprintf("%d-%d-%d", wins, losses, ties)
	}
	return fmt.Sprintf("%d-%d", wins, losses)
}

func parseRatio(raw string) float64 {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "%"))
	if raw == "" {
		return 0
	}
	raw = strings.ReplaceAll(raw, ",", "")
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	if value > 1 {
		return value / 100
	}
	return value
}

func atoiString(raw string) int {
	value, _ := strconv.Atoi(strings.TrimSpace(raw))
	return value
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

func extractTeamInjuries(data json.RawMessage, terms ...string) []json.RawMessage {
	items := extractArrayAt(data, "injuries")
	if len(items) == 0 {
		items = extractArrayAt(data, "items")
	}
	if len(items) == 0 {
		return nil
	}
	return filterRawItemsByTerms(items, terms...)
}

func extractStatSummary(data json.RawMessage, limit int) []map[string]any {
	root := parseJSON(data)
	if root == nil {
		return nil
	}
	seen := map[string]bool{}
	out := make([]map[string]any, 0, limit)
	walkJSON(root, func(obj map[string]any) {
		if limit > 0 && len(out) >= limit {
			return
		}
		label := bestString(obj, "displayName", "shortDisplayName", "name", "label", "abbreviation")
		value := firstNonEmpty(bestString(obj, "displayValue"), bestString(obj, "formatted"), bestString(obj, "value"))
		if label == "" || value == "" {
			return
		}
		key := strings.ToLower(strings.TrimSpace(label))
		if seen[key] || len(key) < 2 {
			return
		}
		seen[key] = true
		entry := map[string]any{
			"name":  label,
			"value": value,
		}
		if numeric, ok := parseNumericValue(value); ok {
			entry["numeric_value"] = numeric
		}
		out = append(out, entry)
	})
	return out
}

func parseNumericValue(raw string) (float64, bool) {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "%"))
	raw = strings.ReplaceAll(raw, ",", "")
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func buildStatComparison(left, right []map[string]any) []map[string]any {
	rightIndex := map[string]map[string]any{}
	for _, item := range right {
		name, _ := item["name"].(string)
		if name == "" {
			continue
		}
		rightIndex[strings.ToLower(name)] = item
	}

	var out []map[string]any
	for _, item := range left {
		name, _ := item["name"].(string)
		if name == "" {
			continue
		}
		other, ok := rightIndex[strings.ToLower(name)]
		if !ok {
			continue
		}
		entry := map[string]any{
			"stat":    name,
			"player1": item["value"],
			"player2": other["value"],
		}
		leftNum, leftOK := item["numeric_value"].(float64)
		rightNum, rightOK := other["numeric_value"].(float64)
		if leftOK && rightOK {
			entry["delta"] = leftNum - rightNum
			switch {
			case leftNum > rightNum:
				entry["leader"] = "player1"
			case rightNum > leftNum:
				entry["leader"] = "player2"
			default:
				entry["leader"] = "tie"
			}
		}
		out = append(out, entry)
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func uniqueNonEmptyStrings(values ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func defaultWatchlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "espn-pp-cli", "watchlist.json")
}

func loadWatchlist() ([]watchItem, error) {
	path := defaultWatchlistPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var wrapped watchlistFile
	if err := json.Unmarshal(data, &wrapped); err == nil && wrapped.Items != nil {
		return wrapped.Items, nil
	}

	var items []watchItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parsing watchlist %s: %w", path, err)
	}
	return items, nil
}

func saveWatchlist(items []watchItem) error {
	path := defaultWatchlistPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].League != items[j].League {
			return items[i].League < items[j].League
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	data, err := json.MarshalIndent(watchlistFile{Items: items}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func filterWatchlistScores(items []watchItem, rows []map[string]any) []map[string]any {
	teamIDs := map[string]bool{}
	teamNames := map[string]bool{}
	for _, item := range items {
		teamIDs[item.TeamID] = true
		for _, name := range uniqueNonEmptyStrings(item.Name, item.DisplayName, item.Abbreviation) {
			teamNames[strings.ToLower(name)] = true
		}
	}

	var filtered []map[string]any
	for _, row := range rows {
		text := strings.ToLower(fmt.Sprintf("%v %v %v %v",
			row["homeTeam"], row["awayTeam"], row["name"], row["league"],
		))
		if teamIDs[fmt.Sprintf("%v", row["homeTeamID"])] || teamIDs[fmt.Sprintf("%v", row["awayTeamID"])] {
			filtered = append(filtered, row)
			continue
		}
		for name := range teamNames {
			if strings.Contains(text, name) {
				filtered = append(filtered, row)
				break
			}
		}
	}
	return filtered
}

func extractStoreH2HEvents(db *store.Store, spec leagueSpec, teamAID, teamBID string) ([]eventSnapshot, error) {
	if db == nil {
		return nil, nil
	}
	items, err := db.ListEvents(spec.Sport, spec.League, "")
	if err != nil {
		return nil, err
	}
	raw := marshalRaw(items)
	return filterEventsForTeams(extractEventSnapshots(raw), teamAID, teamBID), nil
}
