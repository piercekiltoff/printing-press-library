package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newScheduleCmd(flags *rootFlags) *cobra.Command {
	var team string
	var dates string

	cmd := &cobra.Command{
		Use:   "schedule <league>",
		Short: "Show ESPN schedule data for a league",
		Example: `  espn-pp-cli schedule nfl
  espn-pp-cli schedule nfl --team dal
  espn-pp-cli schedule nba --dates 20260401`,
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
			data, err := client.Schedule(spec.Sport, spec.League, dates)
			if err != nil {
				return classifyAPIError(err)
			}
			if team == "" {
				return printOutputWithFlags(cmd.OutOrStdout(), normalizeOutput(data), flags)
			}

			db, err := openStoreIfExists("")
			if err != nil {
				return err
			}
			if db != nil {
				defer db.Close()
			}
			resolved, err := resolveTeam(client, db, spec, team)
			if err != nil {
				return err
			}
			if resolved == nil {
				return notFoundErr(fmt.Errorf("team %q not found in %s", team, spec.Key))
			}

			items := filterRawItemsByTerms(
				extractEventPayloads(data),
				resolved.ID,
				resolved.Abbreviation,
				resolved.Name,
				resolved.DisplayName,
				team,
			)
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(items), flags)
		},
	}

	cmd.Flags().StringVar(&team, "team", "", "Filter schedule to a single team")
	cmd.Flags().StringVar(&dates, "dates", "", "Schedule dates in YYYYMMDD format")
	return cmd
}
