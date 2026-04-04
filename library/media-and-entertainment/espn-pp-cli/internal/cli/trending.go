package cli

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newTrendingCmd(flags *rootFlags) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "trending <league>",
		Short: "Find trending players from recent news and leaderboards",
		Example: `  espn-pp-cli trending nba
  espn-pp-cli trending nfl --limit 10
  espn-pp-cli trending mlb --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if !flags.dryRun {
					return cmd.Help()
				}
				args = []string{"nfl"}
			}
			spec, err := resolveLeagueSpec(args[0])
			if err != nil {
				return err
			}
			client := newESPNClient(flags)

			newsData, err := client.News(spec.Sport, spec.League)
			if err != nil {
				return classifyAPIError(err)
			}
			leadersData, err := client.Leaders(spec.Sport, spec.League, "")
			if err != nil {
				return classifyAPIError(err)
			}

			type trend struct {
				ID      string         `json:"id,omitempty"`
				Name    string         `json:"name"`
				Score   int            `json:"score"`
				Sources []string       `json:"sources"`
				Context map[string]any `json:"context,omitempty"`
			}

			acc := map[string]*trend{}
			add := func(id, name string, score int, source string) {
				key := strings.ToLower(strings.TrimSpace(firstNonEmpty(id, name)))
				if key == "" || name == "" {
					return
				}
				if acc[key] == nil {
					acc[key] = &trend{
						ID:      id,
						Name:    name,
						Context: map[string]any{},
					}
				}
				acc[key].Score += score
				if !containsString(acc[key].Sources, source) {
					acc[key].Sources = append(acc[key].Sources, source)
				}
			}

			for _, athlete := range extractAthleteCandidates(leadersData) {
				add(athlete.ID, firstNonEmpty(athlete.DisplayName, athlete.Name), 3, "leaders")
			}
			for _, raw := range extractNewsPayloads(newsData) {
				var article map[string]any
				if err := json.Unmarshal(raw, &article); err != nil {
					continue
				}
				for _, athlete := range extractAthleteCandidates(raw) {
					add(athlete.ID, firstNonEmpty(athlete.DisplayName, athlete.Name), 2, "news")
					entry := acc[strings.ToLower(firstNonEmpty(athlete.ID, athlete.DisplayName, athlete.Name))]
					if entry != nil && entry.Context["headline"] == nil {
						entry.Context["headline"] = bestString(article, "headline", "title")
					}
				}
			}

			var players []trend
			for _, item := range acc {
				players = append(players, *item)
			}
			sort.Slice(players, func(i, j int) bool {
				if players[i].Score != players[j].Score {
					return players[i].Score > players[j].Score
				}
				return players[i].Name < players[j].Name
			})
			if limit > 0 && len(players) > limit {
				players = players[:limit]
			}

			payload := map[string]any{
				"league":  spec.Key,
				"players": players,
				"news":    parseJSON(normalizeOutput(newsData)),
				"leaders": parseJSON(normalizeOutput(leadersData)),
			}
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(payload), flags)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum number of trending players to return")
	return cmd
}
