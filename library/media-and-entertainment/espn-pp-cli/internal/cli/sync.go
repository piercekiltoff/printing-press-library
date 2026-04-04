package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/espn-pp-cli/internal/espn"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/espn-pp-cli/internal/store"
)

type syncResult struct {
	Resource string
	Count    int
	Err      error
	Duration time.Duration
}

func newSyncCmd(flags *rootFlags) *cobra.Command {
	var resources []string
	var full bool
	var since string
	var concurrency int
	var dbPath string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync ESPN teams, athletes, events, standings, and news to SQLite",
		Example: `  espn-pp-cli sync
  espn-pp-cli sync --resources nfl,nba
  espn-pp-cli sync --full
  espn-pp-cli sync --since 7d
  espn-pp-cli sync --concurrency 4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newESPNClient(flags)

			if dbPath == "" {
				dbPath = defaultStorePath()
			}

			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer db.Close()

			if len(resources) == 0 {
				resources = defaultSyncResources()
			}
			for i := range resources {
				resources[i] = strings.ToLower(strings.TrimSpace(resources[i]))
			}

			if full {
				if err := db.ClearSyncCursors(); err != nil {
					return fmt.Errorf("clearing sync state: %w", err)
				}
			}

			sinceTS := ""
			if since != "" {
				t, err := parseSinceDuration(since)
				if err != nil {
					return err
				}
				sinceTS = t.Format(time.RFC3339)
			}

			if concurrency < 1 {
				concurrency = 1
			}

			started := time.Now()
			work := make(chan string, len(resources))
			results := make(chan syncResult, len(resources))

			var wg sync.WaitGroup
			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for resource := range work {
						results <- syncResource(client, db, resource, sinceTS, full)
					}
				}()
			}

			for _, resource := range resources {
				work <- resource
			}
			close(work)

			go func() {
				wg.Wait()
				close(results)
			}()

			total := 0
			failures := 0
			for result := range results {
				if result.Err != nil {
					fmt.Fprintf(os.Stderr, "%s: error: %v\n", result.Resource, result.Err)
					failures++
					continue
				}
				fmt.Fprintf(os.Stderr, "%s: %d records synced (%.1fs)\n", result.Resource, result.Count, result.Duration.Seconds())
				total += result.Count
			}

			fmt.Fprintf(os.Stderr, "Sync complete: %d records across %d league(s) in %.1fs\n", total, len(resources), time.Since(started).Seconds())
			if failures > 0 {
				return fmt.Errorf("%d league sync(s) failed", failures)
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&resources, "resources", nil, "League keys to sync, such as nfl,nba,mlb,nhl")
	cmd.Flags().BoolVar(&full, "full", false, "Clear previous sync state before syncing")
	cmd.Flags().StringVar(&since, "since", "", "Only sync data newer than this duration (for example 7d or 24h)")
	cmd.Flags().IntVar(&concurrency, "concurrency", 4, "Number of parallel sync workers")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/espn-pp-cli/data.db)")
	return cmd
}

func syncResource(client *espn.ESPN, db *store.Store, resource, sinceTS string, full bool) syncResult {
	started := time.Now()
	spec, err := resolveLeagueSpec(resource)
	if err != nil {
		return syncResult{Resource: resource, Err: err, Duration: time.Since(started)}
	}

	total := 0
	syncErr := func(step string, err error) syncResult {
		return syncResult{Resource: resource, Count: total, Err: fmt.Errorf("%s: %w", step, err), Duration: time.Since(started)}
	}

	teamsData, err := client.Teams(spec.Sport, spec.League)
	if err != nil {
		return syncErr("fetching teams", err)
	}
	for _, item := range extractTeamCandidates(teamsData) {
		if err := db.UpsertTeam(spec.Sport, spec.League, item.Data); err == nil {
			total++
		}
	}

	athletesData, err := client.Athletes(spec.Sport, spec.League, 1000)
	if err != nil {
		return syncErr("fetching athletes", err)
	}
	for _, item := range extractAthleteCandidates(athletesData) {
		if err := db.UpsertAthlete(spec.Sport, spec.League, item.Data); err == nil {
			total++
		}
	}

	dates := ""
	if sinceTS != "" {
		if t, err := time.Parse(time.RFC3339, sinceTS); err == nil {
			dates = t.Format("20060102")
		}
	}
	eventsData, err := client.Schedule(spec.Sport, spec.League, dates)
	if err != nil {
		return syncErr("fetching schedule", err)
	}
	events := extractEventPayloads(eventsData)
	if len(events) == 0 {
		if scoreboardData, scoreErr := client.Scoreboard(spec.Sport, spec.League, dates); scoreErr == nil {
			events = extractEventPayloads(scoreboardData)
		}
	}
	for _, item := range events {
		if err := db.UpsertEvent(spec.Sport, spec.League, item); err == nil {
			total++
		}
	}

	standingsData, err := client.Standings(spec.Sport, spec.League)
	if err != nil {
		return syncErr("fetching standings", err)
	}
	season := seasonLabel(standingsData)
	for _, item := range extractStandingsPayloads(standingsData) {
		if err := db.UpsertStandings(spec.Sport, spec.League, season, item); err == nil {
			total++
		}
	}

	newsData, err := client.News(spec.Sport, spec.League)
	if err != nil {
		return syncErr("fetching news", err)
	}
	for _, item := range extractNewsPayloads(newsData) {
		if err := db.UpsertNews(spec.Sport, spec.League, item); err == nil {
			total++
		}
	}

	if saveErr := db.SaveSyncState(resource, "", total); saveErr != nil && !full {
		return syncErr("saving sync state", saveErr)
	}

	return syncResult{Resource: resource, Count: total, Duration: time.Since(started)}
}

func parseSinceDuration(input string) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, fmt.Errorf("empty duration")
	}

	var value int
	var unit string
	if _, err := fmt.Sscanf(input, "%d%s", &value, &unit); err != nil {
		return time.Time{}, fmt.Errorf("invalid --since value %q", input)
	}

	switch unit {
	case "m":
		return time.Now().Add(-time.Duration(value) * time.Minute), nil
	case "h":
		return time.Now().Add(-time.Duration(value) * time.Hour), nil
	case "d":
		return time.Now().Add(-time.Duration(value) * 24 * time.Hour), nil
	case "w":
		return time.Now().Add(-time.Duration(value) * 7 * 24 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("invalid --since unit %q", unit)
	}
}

func defaultSyncResources() []string {
	return majorLeagueKeys()
}
