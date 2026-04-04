package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newH2HCmd(flags *rootFlags) *cobra.Command {
	var league string
	var limit int

	cmd := &cobra.Command{
		Use:   "h2h <team-1> <team-2>",
		Short: "Show head-to-head history between two teams",
		Example: `  espn-pp-cli h2h "Lakers" "Celtics" --league nba
  espn-pp-cli h2h "Chiefs" "Bills" --league nfl
  espn-pp-cli h2h "Dodgers" "Giants" --league mlb --limit 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				if !flags.dryRun {
					return cmd.Help()
				}
				args = []string{"dal", "phi"}
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

			left, err := resolveTeam(client, db, spec, args[0])
			if err != nil {
				return err
			}
			right, err := resolveTeam(client, db, spec, args[1])
			if err != nil {
				return err
			}
			if left == nil || left.ID == "" {
				return notFoundErr(fmt.Errorf("team %q not found in %s", args[0], spec.Key))
			}
			if right == nil || right.ID == "" {
				return notFoundErr(fmt.Errorf("team %q not found in %s", args[1], spec.Key))
			}

			matchups, err := extractStoreH2HEvents(db, spec, left.ID, right.ID)
			if err != nil {
				return err
			}
			if len(matchups) == 0 {
				scheduleData, err := client.Schedule(spec.Sport, spec.League, "")
				if err != nil {
					return classifyAPIError(err)
				}
				matchups = filterEventsForTeams(extractEventSnapshots(scheduleData), left.ID, right.ID)
			}
			matchups = filterPastEvents(matchups)
			if limit > 0 && len(matchups) > limit {
				matchups = matchups[:limit]
			}

			payload := map[string]any{
				"league": spec.Key,
				"team1":  left,
				"team2":  right,
				"games":  matchups,
				"count":  len(matchups),
			}
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(payload), flags)
		},
	}

	cmd.Flags().StringVar(&league, "league", "nfl", "League key")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of past matchups to return")
	return cmd
}
