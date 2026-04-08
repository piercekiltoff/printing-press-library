package cli

import (
	"github.com/spf13/cobra"
)

func newOddsCmd(flags *rootFlags) *cobra.Command {
	var event string

	cmd := &cobra.Command{
		Use:   "odds <league>",
		Short: "Show betting odds from ESPN",
		Example: `  espn-pp-cli odds nfl
  espn-pp-cli odds nba --event 401671793`,
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
			data, err := client.Odds(spec.Sport, spec.League, event)
			if err != nil {
				return classifyAPIError(err)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), normalizeOutput(data), flags)
		},
	}

	cmd.Flags().StringVar(&event, "event", "", "Optional event ID")
	return cmd
}
