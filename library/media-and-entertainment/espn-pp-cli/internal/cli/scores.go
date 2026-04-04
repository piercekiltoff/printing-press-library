package cli

import "github.com/spf13/cobra"

func newScoresCmd(flags *rootFlags) *cobra.Command {
	var date string
	var all bool

	cmd := &cobra.Command{
		Use:   "scores [league]",
		Short: "Show ESPN scores across one or more leagues",
		Example: `  espn-pp-cli scores
  espn-pp-cli scores nfl
  espn-pp-cli scores nba --date 20260328
  espn-pp-cli scores --all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newESPNClient(flags)

			var leagues []string
			switch {
			case len(args) == 1:
				leagues = []string{args[0]}
			case all:
				leagues = allLeagueKeys()
			default:
				leagues = majorLeagueKeys()
			}

			rows := make([]map[string]any, 0)
			for _, key := range leagues {
				leagueRows, err := scoreRowsForLeague(client, key, date)
				if err != nil {
					return classifyAPIError(err)
				}
				if len(args) == 0 && !all && len(leagueRows) == 0 {
					continue
				}
				rows = append(rows, leagueRows...)
			}

			if len(rows) == 0 && flags.dryRun {
				return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw([]map[string]any{}), flags)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(rows), flags)
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "Scoreboard date in YYYYMMDD format")
	cmd.Flags().BoolVar(&all, "all", false, "Query all supported leagues")
	return cmd
}
