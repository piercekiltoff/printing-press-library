package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newTeamCmd(flags *rootFlags) *cobra.Command {
	var roster bool
	var stats bool
	var league string

	cmd := &cobra.Command{
		Use:   "team <name-or-abbreviation>",
		Short: "Show team details, roster, or stats",
		Example: `  espn-pp-cli team "Dallas Cowboys"
  espn-pp-cli team dal --roster
  espn-pp-cli team dal --stats`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if !flags.dryRun {
					return cmd.Help()
				}
				args = []string{"dal"}
			}
			if roster && stats {
				return usageErr(fmt.Errorf("use only one of --roster or --stats"))
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

			var data []byte
			if roster {
				data, err = client.TeamRoster(spec.Sport, spec.League, team.ID)
			} else {
				data, err = client.Team(spec.Sport, spec.League, team.ID)
			}
			if err != nil {
				return classifyAPIError(err)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), normalizeOutput(data), flags)
		},
	}

	cmd.Flags().BoolVar(&roster, "roster", false, "Return team roster")
	cmd.Flags().BoolVar(&stats, "stats", false, "Return team overview and stats")
	cmd.Flags().StringVar(&league, "league", "nfl", "League key")
	return cmd
}
