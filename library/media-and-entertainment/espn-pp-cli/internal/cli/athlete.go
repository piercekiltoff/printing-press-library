package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAthleteCmd(flags *rootFlags) *cobra.Command {
	var gamelog bool
	var splits bool
	var stats bool
	var league string

	cmd := &cobra.Command{
		Use:   "athlete <name-or-id>",
		Short: "Show athlete profile, gamelog, splits, or stats",
		Example: `  espn-pp-cli athlete "Patrick Mahomes"
  espn-pp-cli athlete 3139477 --gamelog
  espn-pp-cli athlete "LeBron James" --splits --league nba
  espn-pp-cli athlete "LeBron James" --stats`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if !flags.dryRun {
					return cmd.Help()
				}
				league = "nba"
				args = []string{"LeBron James"}
			}
			modes := 0
			if gamelog {
				modes++
			}
			if splits {
				modes++
			}
			if stats {
				modes++
			}
			if modes > 1 {
				return usageErr(fmt.Errorf("use at most one of --gamelog, --splits, or --stats"))
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

			athlete, err := resolveAthlete(client, db, args[0])
			if err != nil {
				return err
			}
			if athlete == nil || athlete.ID == "" {
				return notFoundErr(fmt.Errorf("athlete %q not found", args[0]))
			}

			var data []byte
			switch {
			case gamelog:
				data, err = client.AthleteGamelog(spec.Sport, spec.League, athlete.ID)
			case splits:
				data, err = client.AthleteSplits(spec.Sport, spec.League, athlete.ID)
			case stats:
				data, err = client.AthleteStats(spec.Sport, spec.League, athlete.ID)
			default:
				data, err = client.Athlete(spec.Sport, spec.League, athlete.ID)
			}
			if err != nil {
				return classifyAPIError(err)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), normalizeOutput(data), flags)
		},
	}

	cmd.Flags().BoolVar(&gamelog, "gamelog", false, "Return athlete game log")
	cmd.Flags().BoolVar(&splits, "splits", false, "Return athlete splits")
	cmd.Flags().BoolVar(&stats, "stats", false, "Return athlete stats")
	cmd.Flags().StringVar(&league, "league", "nfl", "League key")
	return cmd
}
