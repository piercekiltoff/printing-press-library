package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func newSOSCmd(flags *rootFlags) *cobra.Command {
	var league string

	cmd := &cobra.Command{
		Use:   "sos <team>",
		Short: "Analyze strength of schedule from opponent records",
		Example: `  espn-pp-cli sos "Dallas Cowboys" --league nfl
  espn-pp-cli sos "Los Angeles Lakers" --league nba
  espn-pp-cli sos dal --league nfl --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if !flags.dryRun {
					return cmd.Help()
				}
				args = []string{"dal"}
			}
			spec, err := resolveLeagueSpec(league)
			if err != nil {
				return err
			}
			client := newESPNClient(flags)
			db, err := openStoreIfExists("")
			if err != nil {
				return err
			}
			if db != nil {
				defer db.Close()
			}

			team, err := resolveTeam(client, db, spec, args[0])
			if err != nil {
				return err
			}
			if team == nil || team.ID == "" {
				return notFoundErr(fmt.Errorf("team %q not found in %s", args[0], spec.Key))
			}

			scheduleData, err := client.Schedule(spec.Sport, spec.League, "")
			if err != nil {
				return classifyAPIError(err)
			}
			standingsData, err := client.Standings(spec.Sport, spec.League)
			if err != nil {
				return classifyAPIError(err)
			}

			standings := extractStandingsIndex(standingsData)
			terms := teamTerms(team, args[0])
			events := extractEventSnapshots(scheduleData)

			var opponents []map[string]any
			var total float64
			for _, event := range events {
				var opponent eventTeamSnapshot
				switch {
				case containsString(terms, event.Home.ID) || containsString(terms, event.Home.Abbreviation) || containsString(terms, event.Home.Name):
					opponent = event.Away
				case containsString(terms, event.Away.ID) || containsString(terms, event.Away.Abbreviation) || containsString(terms, event.Away.Name):
					opponent = event.Home
				default:
					continue
				}

				standing := standings[opponent.ID]
				total += standing.WinPct
				opponents = append(opponents, map[string]any{
					"id":      opponent.ID,
					"name":    firstNonEmpty(opponent.Name, opponent.Abbreviation),
					"record":  standing.Record,
					"win_pct": standing.WinPct,
					"event":   event,
				})
			}

			sort.Slice(opponents, func(i, j int) bool {
				left, _ := opponents[i]["win_pct"].(float64)
				right, _ := opponents[j]["win_pct"].(float64)
				return left > right
			})

			average := 0.0
			if len(opponents) > 0 {
				average = total / float64(len(opponents))
			}
			payload := map[string]any{
				"league":            spec.Key,
				"team":              team,
				"games_analyzed":    len(opponents),
				"opponent_win_pct":  average,
				"difficulty_rating": int(clamp01(average)*100 + 0.5),
				"difficulty_tier":   sosDifficultyTier(average),
				"opponents":         opponents,
			}
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(payload), flags)
		},
	}

	cmd.Flags().StringVar(&league, "league", "nfl", "League key")
	return cmd
}

func sosDifficultyTier(v float64) string {
	switch {
	case v >= 0.6:
		return "very-hard"
	case v >= 0.55:
		return "hard"
	case v >= 0.5:
		return "medium"
	default:
		return "light"
	}
}
