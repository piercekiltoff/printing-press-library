package cli

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newSearchCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search ESPN live data and the local sync store",
		Example: `  espn-pp-cli search "LeBron James"
  espn-pp-cli search "Dallas Cowboys"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			client := newESPNClient(flags)
			results := make([]map[string]any, 0)

			liveData, err := client.Search(query)
			if err != nil {
				return classifyAPIError(err)
			}

			// ESPN search returns {results: [{type: "player", contents: [...]}, {type: "team", contents: [...]}]}
			results = append(results, parseESPNSearchResults(liveData)...)

			db, err := openStoreIfExists("")
			if err != nil {
				return err
			}
			if db != nil {
				defer db.Close()
				teamRows, _ := db.SearchTeams(query, 10)
				athleteRows, _ := db.SearchAthletes(query, 10)
				newsRows, _ := db.SearchNews(query, 10)

				for _, team := range extractTeamResults(teamRows) {
					results = append(results, map[string]any{
						"source":       "local",
						"type":         "team",
						"id":           team.ID,
						"name":         firstNonEmpty(team.DisplayName, team.Name),
						"abbreviation": team.Abbreviation,
					})
				}
				for _, athlete := range extractAthleteResults(athleteRows) {
					results = append(results, map[string]any{
						"source":  "local",
						"type":    "athlete",
						"id":      athlete.ID,
						"name":    firstNonEmpty(athlete.DisplayName, athlete.Name),
						"team_id": athlete.TeamID,
					})
				}
				for _, article := range newsRows {
					var obj map[string]any
					if err := json.Unmarshal(article, &obj); err != nil {
						continue
					}
					results = append(results, map[string]any{
						"source": "local",
						"type":   "news",
						"id":     bestString(obj, "id", "guid"),
						"name":   bestString(obj, "headline", "title"),
					})
				}
			}

			sort.SliceStable(results, func(i, j int) bool {
				left := results[i]["source"].(string) + ":" + results[i]["type"].(string)
				right := results[j]["source"].(string) + ":" + results[j]["type"].(string)
				return left < right
			})

			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(results), flags)
		},
	}
	return cmd
}

// parseESPNSearchResults extracts structured results from ESPN's search response.
// Format: {results: [{type: "player"|"team"|"article", contents: [{id, uid, displayName, ...}]}]}
func parseESPNSearchResults(data json.RawMessage) []map[string]any {
	var resp struct {
		Results []struct {
			Type     string `json:"type"`
			Contents []struct {
				ID          string `json:"id"`
				UID         string `json:"uid"`
				DisplayName string `json:"displayName"`
				Description string `json:"description"`
			} `json:"contents"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil
	}

	var out []map[string]any
	for _, group := range resp.Results {
		for _, item := range group.Contents {
			// Extract numeric athlete ID from uid like "s:40~l:46~a:1966"
			athleteID := ""
			if item.UID != "" {
				for _, part := range strings.Split(item.UID, "~") {
					if strings.HasPrefix(part, "a:") {
						athleteID = strings.TrimPrefix(part, "a:")
					}
				}
			}

			entry := map[string]any{
				"source":      "live",
				"type":        group.Type,
				"id":          firstNonEmpty(athleteID, item.ID),
				"name":        item.DisplayName,
				"description": item.Description,
			}
			if athleteID != "" {
				entry["athlete_id"] = athleteID
			}
			out = append(out, entry)
		}
	}
	return out
}
