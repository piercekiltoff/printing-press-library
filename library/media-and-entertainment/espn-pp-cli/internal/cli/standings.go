package cli

import (
	"github.com/spf13/cobra"
)

func newStandingsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "standings <league>",
		Short: "Show league standings from ESPN",
		Example: `  espn-pp-cli standings nfl
  espn-pp-cli standings nba --json`,
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
			data, err := client.Standings(spec.Sport, spec.League)
			if err != nil {
				return classifyAPIError(err)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), normalizeOutput(data), flags)
		},
	}
	return cmd
}
