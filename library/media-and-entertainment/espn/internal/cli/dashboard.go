package cli

import (
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newDashboardCmd(flags *rootFlags) *cobra.Command {
	var leaguesCSV string
	var date string

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Show a quick scoreboard overview across major leagues",
		Example: `  espn-pp-cli dashboard
  espn-pp-cli dashboard --leagues nfl,nba
  espn-pp-cli dashboard --date 20260329 --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newESPNClient(flags)
			leagueKeys := majorLeagueKeys()
			if strings.TrimSpace(leaguesCSV) != "" {
				leagueKeys = nil
				for _, part := range strings.Split(leaguesCSV, ",") {
					part = strings.TrimSpace(part)
					if part != "" {
						leagueKeys = append(leagueKeys, part)
					}
				}
			}

			leagues := make([]map[string]any, 0, len(leagueKeys))
			for _, leagueKey := range leagueKeys {
				rows, err := scoreRowsForLeague(client, leagueKey, date)
				if err != nil {
					return classifyAPIError(err)
				}
				if len(rows) == 0 {
					continue
				}
				leagues = append(leagues, map[string]any{
					"league": leagueKey,
					"games":  len(rows),
					"events": rows,
				})
			}

			payload := map[string]any{
				"generated_at": time.Now().Format(time.RFC3339),
				"leagues":      leagues,
			}
			return printOutputWithFlags(cmd.OutOrStdout(), marshalRaw(payload), flags)
		},
	}

	cmd.Flags().StringVar(&leaguesCSV, "leagues", "", "Comma-separated league keys to include")
	cmd.Flags().StringVar(&date, "date", "", "Scoreboard date in YYYYMMDD format")
	return cmd
}
