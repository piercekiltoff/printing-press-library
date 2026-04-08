package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPreviewCmd(flags *rootFlags) *cobra.Command {
	var league string

	cmd := &cobra.Command{
		Use:   "preview <event-id>",
		Short: "Build a matchup preview with records, injuries, and odds",
		Example: `  espn-pp-cli preview 401810937 --league nba
  espn-pp-cli preview 401547665 --league nfl
  espn-pp-cli preview 401810937 --league nba --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if !flags.dryRun {
					return cmd.Help()
				}
				args = []string{"401671793"}
			}
			spec, err := resolveLeagueSpec(league)
			if err != nil {
				return err
			}
			client := newESPNClient(flags)

			summaryData, err := client.Summary(spec.Sport, spec.League, args[0])
			if err != nil {
				return classifyAPIError(err)
			}
			injuriesData, err := client.Injuries(spec.Sport, spec.League)
			if err != nil {
				return classifyAPIError(err)
			}
			oddsData, err := client.Odds(spec.Sport, spec.League, args[0])
			if err != nil {
				return classifyAPIError(err)
			}
			standingsData, err := client.Standings(spec.Sport, spec.League)
			if err != nil {
				return classifyAPIError(err)
			}

			event, ok := extractEventSnapshotFromRaw(normalizeOutput(summaryData))
			if !ok {
				return notFoundErr(fmt.Errorf("event %q not found", args[0]))
			}
			standings := extractStandingsIndex(standingsData)

			payload := map[string]any{
				"league": spec.Key,
				"event":  parseJSON(normalizeOutput(summaryData)),
				"matchup": map[string]any{
					"id":     event.ID,
					"name":   event.Name,
					"date":   event.Date,
					"status": event.Status,
					"home": map[string]any{
						"id":        event.Home.ID,
						"name":      event.Home.Name,
						"abbrev":    event.Home.Abbreviation,
						"record":    firstNonEmpty(event.Home.Record, standings[event.Home.ID].Record),
						"score":     event.Home.Score,
						"standings": standings[event.Home.ID],
						"injuries":  parseJSON(marshalRaw(extractTeamInjuries(injuriesData, event.Home.ID, event.Home.Name, event.Home.Abbreviation))),
					},
					"away": map[string]any{
						"id":        event.Away.ID,
						"name":      event.Away.Name,
						"abbrev":    event.Away.Abbreviation,
						"record":    firstNonEmpty(event.Away.Record, standings[event.Away.ID].Record),
						"score":     event.Away.Score,
						"standings": standings[event.Away.ID],
						"injuries":  parseJSON(marshalRaw(extractTeamInjuries(injuriesData, event.Away.ID, event.Away.Name, event.Away.Abbreviation))),
					},
				},
				"odds": parseJSON(normalizeOutput(oddsData)),
			}
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(payload), flags)
		},
	}

	cmd.Flags().StringVar(&league, "league", "nfl", "League key")
	return cmd
}
