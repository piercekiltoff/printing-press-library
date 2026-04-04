package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCompareCmd(flags *rootFlags) *cobra.Command {
	var league string

	cmd := &cobra.Command{
		Use:   "compare <athlete-1> <athlete-2>",
		Short: "Compare two athletes side by side",
		Example: `  espn-pp-cli compare "LeBron James" "Kevin Durant" --league nba
  espn-pp-cli compare "Patrick Mahomes" "Josh Allen" --league nfl
  espn-pp-cli compare 3139477 3918298 --league nfl --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				if !flags.dryRun {
					return cmd.Help()
				}
				args = []string{"LeBron James", "Kevin Durant"}
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

			left, err := resolveAthlete(client, db, args[0])
			if err != nil {
				return err
			}
			right, err := resolveAthlete(client, db, args[1])
			if err != nil {
				return err
			}
			if left == nil || left.ID == "" {
				return notFoundErr(fmt.Errorf("athlete %q not found", args[0]))
			}
			if right == nil || right.ID == "" {
				return notFoundErr(fmt.Errorf("athlete %q not found", args[1]))
			}

			leftOverview, err := client.Athlete(spec.Sport, spec.League, left.ID)
			if err != nil {
				return classifyAPIError(err)
			}
			leftStats, err := client.AthleteStats(spec.Sport, spec.League, left.ID)
			if err != nil {
				return classifyAPIError(err)
			}
			rightOverview, err := client.Athlete(spec.Sport, spec.League, right.ID)
			if err != nil {
				return classifyAPIError(err)
			}
			rightStats, err := client.AthleteStats(spec.Sport, spec.League, right.ID)
			if err != nil {
				return classifyAPIError(err)
			}

			leftSummary := extractStatSummary(leftStats, 16)
			rightSummary := extractStatSummary(rightStats, 16)
			payload := map[string]any{
				"league": spec.Key,
				"player1": map[string]any{
					"query":    args[0],
					"id":       left.ID,
					"name":     firstNonEmpty(left.DisplayName, left.Name, args[0]),
					"overview": parseJSON(normalizeOutput(leftOverview)),
					"stats":    parseJSON(normalizeOutput(leftStats)),
					"summary":  leftSummary,
				},
				"player2": map[string]any{
					"query":    args[1],
					"id":       right.ID,
					"name":     firstNonEmpty(right.DisplayName, right.Name, args[1]),
					"overview": parseJSON(normalizeOutput(rightOverview)),
					"stats":    parseJSON(normalizeOutput(rightStats)),
					"summary":  rightSummary,
				},
				"comparison": map[string]any{
					"shared_stats": buildStatComparison(leftSummary, rightSummary),
				},
			}
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(payload), flags)
		},
	}

	cmd.Flags().StringVar(&league, "league", "nfl", "League key")
	return cmd
}
