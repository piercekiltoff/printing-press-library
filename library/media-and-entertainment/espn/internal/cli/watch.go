package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newWatchCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Manage a multi-sport team watchlist",
		Example: `  espn-pp-cli watch add "Lakers" --league nba
  espn-pp-cli watch add "Chiefs" --league nfl
  espn-pp-cli watch list
  espn-pp-cli watch scores
  espn-pp-cli watch remove "Lakers" --league nba`,
	}

	cmd.AddCommand(newWatchAddCmd(flags))
	cmd.AddCommand(newWatchRemoveCmd(flags))
	cmd.AddCommand(newWatchListCmd(flags))
	cmd.AddCommand(newWatchScoresCmd(flags))
	return cmd
}

func newWatchAddCmd(flags *rootFlags) *cobra.Command {
	var league string

	cmd := &cobra.Command{
		Use:   "add <team>",
		Short: "Add a team to the watchlist",
		Example: `  espn-pp-cli watch add "Lakers" --league nba
  espn-pp-cli watch add "Chiefs" --league nfl --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			items, err := loadWatchlist()
			if err != nil {
				return err
			}
			for _, item := range items {
				if item.League == spec.Key && item.TeamID == team.ID {
					return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(item), flags)
				}
			}

			entry := watchItem{
				League:       spec.Key,
				Sport:        spec.Sport,
				TeamID:       team.ID,
				Name:         firstNonEmpty(team.Name, team.DisplayName, args[0]),
				DisplayName:  team.DisplayName,
				Abbreviation: team.Abbreviation,
				AddedAt:      time.Now().Format(time.RFC3339),
			}
			if err := confirmMutation(flags, cmd.ErrOrStderr(), fmt.Sprintf("Add %q to the %s watchlist?", entry.Name, spec.Key)); err != nil {
				return err
			}
			items = append(items, entry)
			if !flags.dryRun {
				if err := saveWatchlist(items); err != nil {
					return err
				}
			}
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(entry), flags)
		},
	}
	cmd.Flags().StringVar(&league, "league", "nfl", "League key")
	return cmd
}

func newWatchRemoveCmd(flags *rootFlags) *cobra.Command {
	var league string

	cmd := &cobra.Command{
		Use:   "remove <team>",
		Short: "Remove a team from the watchlist",
		Example: `  espn-pp-cli watch remove "Lakers" --league nba
  espn-pp-cli watch remove "Chiefs" --league nfl`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := resolveLeagueSpec(league)
			if err != nil {
				return err
			}
			items, err := loadWatchlist()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return notFoundErr(fmt.Errorf("watchlist is empty"))
			}

			var (
				removed *watchItem
				next    []watchItem
			)
			target := strings.ToLower(strings.TrimSpace(args[0]))
			for i := range items {
				item := items[i]
				if item.League == spec.Key && (strings.ToLower(item.TeamID) == target ||
					strings.ToLower(item.Name) == target ||
					strings.ToLower(item.DisplayName) == target ||
					strings.ToLower(item.Abbreviation) == target) {
					copyItem := item
					removed = &copyItem
					continue
				}
				next = append(next, item)
			}
			if removed == nil {
				return notFoundErr(fmt.Errorf("team %q is not on the %s watchlist", args[0], spec.Key))
			}
			if err := confirmMutation(flags, cmd.ErrOrStderr(), fmt.Sprintf("Remove %q from the %s watchlist?", removed.Name, spec.Key)); err != nil {
				return err
			}
			if !flags.dryRun {
				if err := saveWatchlist(next); err != nil {
					return err
				}
			}
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(removed), flags)
		},
	}

	cmd.Flags().StringVar(&league, "league", "nfl", "League key")
	return cmd
}

func newWatchListCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show the current watchlist",
		Example: `  espn-pp-cli watch list
  espn-pp-cli watch list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := loadWatchlist()
			if err != nil {
				return err
			}
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(items), flags)
		},
	}
}

func newWatchScoresCmd(flags *rootFlags) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "scores",
		Short: "Show scores for all watched teams",
		Example: `  espn-pp-cli watch scores
  espn-pp-cli watch scores --date 20260329
  espn-pp-cli watch scores --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := loadWatchlist()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw([]map[string]any{}), flags)
			}
			client := newESPNClient(flags)
			leagues := map[string]bool{}
			for _, item := range items {
				leagues[item.League] = true
			}

			var rows []map[string]any
			for leagueKey := range leagues {
				leagueRows, err := scoreRowsForLeague(client, leagueKey, date)
				if err != nil {
					return classifyAPIError(err)
				}
				rows = append(rows, leagueRows...)
			}
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(filterWatchlistScores(items, rows)), flags)
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "Scoreboard date in YYYYMMDD format")
	return cmd
}
