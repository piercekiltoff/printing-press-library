package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func newNewsCmd(flags *rootFlags) *cobra.Command {
	var team string

	cmd := &cobra.Command{
		Use:   "news [league]",
		Short: "Show ESPN news across one or more leagues",
		Example: `  espn-pp-cli news
  espn-pp-cli news nfl
  espn-pp-cli news nba --team lakers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				if flags.dryRun {
					args = []string{"nfl"}
				}
			}
			client := newESPNClient(flags)
			leagues := majorLeagueKeys()
			if len(args) == 1 {
				leagues = []string{args[0]}
			}

			var articles []json.RawMessage
			for _, key := range leagues {
				spec, err := resolveLeagueSpec(key)
				if err != nil {
					return err
				}
				data, err := client.News(spec.Sport, spec.League)
				if err != nil {
					return classifyAPIError(err)
				}
				items := extractNewsPayloads(data)
				if team != "" {
					items = filterRawItemsByTerms(items, team)
				}
				articles = append(articles, items...)
			}

			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(articles), flags)
		},
	}

	cmd.Flags().StringVar(&team, "team", "", "Filter league news by team name")
	return cmd
}
