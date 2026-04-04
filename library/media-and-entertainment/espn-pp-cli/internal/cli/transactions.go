package cli

import (
	"github.com/spf13/cobra"
)

func newTransactionsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transactions <league>",
		Short: "Show recent league transactions from ESPN",
		Example: `  espn-pp-cli transactions nfl
  espn-pp-cli transactions nba`,
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
			data, err := client.Transactions(spec.Sport, spec.League)
			if err != nil {
				return classifyAPIError(err)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), normalizeOutput(data), flags)
		},
	}
	return cmd
}
