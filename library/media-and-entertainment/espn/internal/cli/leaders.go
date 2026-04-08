package cli

import (
	"github.com/spf13/cobra"
)

func newLeadersCmd(flags *rootFlags) *cobra.Command {
	var category string

	cmd := &cobra.Command{
		Use:   "leaders <league>",
		Short: "Show ESPN stat leaders",
		Example: `  espn-pp-cli leaders nfl
  espn-pp-cli leaders nba --category assists`,
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
			data, err := client.Leaders(spec.Sport, spec.League, category)
			if err != nil {
				return classifyAPIError(err)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), normalizeOutput(data), flags)
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Leader category, such as assists or passingYards")
	return cmd
}
