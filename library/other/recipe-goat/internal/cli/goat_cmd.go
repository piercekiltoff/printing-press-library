package cli

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mvanhorn/printing-press-library/library/other/recipe-goat/internal/recipes"

	"github.com/spf13/cobra"
)

// goatEntry is an internal ranking record. Exposed via JSON output.
type goatEntry struct {
	Rank        int     `json:"rank"`
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Site        string  `json:"site"`
	Author      string  `json:"author,omitempty"`
	Rating      float64 `json:"rating,omitempty"`
	ReviewCount int     `json:"reviewCount,omitempty"`
	TotalTimeS  int     `json:"totalTimeSeconds,omitempty"`
	Score       float64 `json:"score"`
}

func newGoatCmd(flags *rootFlags) *cobra.Command {
	var (
		limit    int
		sitesCSV string
		saveAll  bool
	)
	cmd := &cobra.Command{
		Use:   "goat <query>",
		Short: "Cross-site recipe ranker — fetch and rank the best version of any dish",
		Long: `Search across 15 trusted recipe sites, fetch each candidate, then rank by
a weighted score of rating, review volume, author trust, site trust, and recency.

Ranking weights:
  0.45 rating_normalized + 0.20 log(reviews+1)/log(1000) + 0.20 author_trust
  + 0.10 site_trust + 0.05 recency_norm`,
		Example: "  recipe-goat-pp-cli goat \"chicken tikka masala\" --limit 5",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			ctx, cancel := context.WithTimeout(context.Background(), 2*flags.timeout)
			defer cancel()

			sites := siteHostsFromCSV(sitesCSV)
			if len(sites) == 0 {
				return usageErr(fmt.Errorf("no sites selected — check --sites"))
			}
			client := httpClientForSites(flags.timeout)

			// 1. Search each site concurrently.
			type searchOut struct {
				results []recipes.SearchResult
				site    recipes.Site
			}
			searchCh := make(chan searchOut, len(sites))
			var swg sync.WaitGroup
			sem := make(chan struct{}, 4)
			for _, s := range sites {
				s := s
				swg.Add(1)
				go func() {
					defer swg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					res, err := recipes.SearchSite(ctx, client, s, query, 3)
					if err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "search %s: %v\n", s.Hostname, err)
						return
					}
					searchCh <- searchOut{results: res, site: s}
				}()
			}
			go func() { swg.Wait(); close(searchCh) }()

			var candidates []recipes.SearchResult
			for so := range searchCh {
				candidates = append(candidates, so.results...)
			}
			if len(candidates) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "no candidates found (sites may be blocking unauthenticated scrapers — run `doctor` to check reachability)")
				if flags.asJSON {
					return flags.printJSON(cmd, []goatEntry{})
				}
				return nil
			}

			// 2. Fetch each candidate (4 workers) to get ratings/reviews/etc.
			// Only recipes that (a) parse successfully, (b) have ingredients
			// and a name, and (c) whose name matches at least one query token
			// make it into the ranking. This keeps editorial pages and
			// round-ups that slipped past the URL filter out of the results.
			type fetchOut struct {
				r    *recipes.Recipe
				src  recipes.SearchResult
				kept bool
			}
			fetchCh := make(chan fetchOut, len(candidates))
			var fwg sync.WaitGroup
			fsem := make(chan struct{}, 4)
			for _, c := range candidates {
				c := c
				fwg.Add(1)
				go func() {
					defer fwg.Done()
					fsem <- struct{}{}
					defer func() { <-fsem }()
					fctx, fcancel := context.WithTimeout(ctx, flags.timeout)
					defer fcancel()
					r, err := recipes.Fetch(fctx, client, c.URL)
					if err != nil || r == nil {
						fetchCh <- fetchOut{src: c, kept: false}
						return
					}
					// JSON-LD must include a name and at least one ingredient.
					if strings.TrimSpace(r.Name) == "" || len(r.RecipeIngredient) == 0 {
						fetchCh <- fetchOut{src: c, kept: false}
						return
					}
					// Title must match at least one query token.
					if !recipes.MatchesQueryPublic(r.Name, c.URL, query) {
						fetchCh <- fetchOut{src: c, kept: false}
						return
					}
					fetchCh <- fetchOut{r: r, src: c, kept: true}
				}()
			}
			go func() { fwg.Wait(); close(fetchCh) }()

			fetched := []*recipes.Recipe{}
			totalCandidates := 0
			for fo := range fetchCh {
				totalCandidates++
				if fo.kept && fo.r != nil {
					fetched = append(fetched, fo.r)
				}
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "filtered: kept %d of %d candidates\n", len(fetched), totalCandidates)

			// 3. Score.
			entries := make([]goatEntry, 0, len(fetched))
			for _, r := range fetched {
				site := recipes.FindSite(r.Site)
				score := goatScore(r, site)
				entries = append(entries, goatEntry{
					Title:       r.Name,
					URL:         r.URL,
					Site:        r.Site,
					Author:      r.Author,
					Rating:      r.AggregateRating.Value,
					ReviewCount: r.AggregateRating.Count,
					TotalTimeS:  r.TotalTime,
					Score:       score,
				})
			}
			sort.SliceStable(entries, func(i, j int) bool { return entries[i].Score > entries[j].Score })
			if limit > 0 && len(entries) > limit {
				entries = entries[:limit]
			}
			for i := range entries {
				entries[i].Rank = i + 1
			}

			// 4. Optionally save all.
			if saveAll && !flags.dryRun {
				st, err := openRecipeStore()
				if err == nil {
					defer st.Close()
					for _, r := range fetched {
						// Only save results with real JSON-LD (have ingredients).
						if len(r.RecipeIngredient) == 0 {
							continue
						}
						if _, err := st.SaveRecipe(recipeToStored(r)); err != nil {
							fmt.Fprintf(cmd.ErrOrStderr(), "save %s: %v\n", r.URL, err)
						}
					}
				}
			}

			if flags.asJSON {
				return flags.printJSON(cmd, entries)
			}
			headers := []string{"#", "TITLE", "SITE", "AUTHOR", "RATING", "REVIEWS", "TIME", "SCORE", "URL"}
			rows := make([][]string, 0, len(entries))
			for _, e := range entries {
				rating := "—"
				if e.Rating > 0 {
					rating = fmt.Sprintf("%.2f", e.Rating)
				}
				rows = append(rows, []string{
					strconv.Itoa(e.Rank),
					truncate(e.Title, 48),
					e.Site,
					truncate(e.Author, 18),
					rating,
					strconv.Itoa(e.ReviewCount),
					formatDuration(e.TotalTimeS),
					fmt.Sprintf("%.3f", e.Score),
					e.URL,
				})
			}
			return flags.printTable(cmd, headers, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 5, "Top N results to return")
	cmd.Flags().StringVar(&sitesCSV, "sites", "all", "Sites to query (CSV of hostnames, or 'all')")
	cmd.Flags().BoolVar(&saveAll, "save-all", false, "Save every fetched result to the cookbook")
	return cmd
}

// goatScore implements the ranking formula. All components are in [0,1].
func goatScore(r *recipes.Recipe, site recipes.Site) float64 {
	ratingNorm := 0.0
	if r.AggregateRating.Value > 0 {
		ratingNorm = r.AggregateRating.Value / 5.0
		if ratingNorm > 1.0 {
			ratingNorm = 1.0
		}
	}
	reviewNorm := 0.0
	if r.AggregateRating.Count > 0 {
		reviewNorm = math.Log(float64(r.AggregateRating.Count+1)) / math.Log(1000)
		if reviewNorm > 1.0 {
			reviewNorm = 1.0
		}
	}
	authorTrust := recipes.AuthorTrust(r.Author)
	siteTrust := site.Trust
	if siteTrust == 0 {
		siteTrust = 0.5
	}
	// Recency: newer = higher. Treat "recent" as last 5 years on a linear
	// scale; everything older gets 0.
	recency := 0.0
	if !r.FetchedAt.IsZero() {
		// FetchedAt is "now" effectively — use DatePublished if parseable.
		if r.DatePublished != "" {
			if t, err := time.Parse("2006-01-02", r.DatePublished[:min(10, len(r.DatePublished))]); err == nil {
				ageYears := time.Since(t).Hours() / 24 / 365
				recency = 1.0 - ageYears/5.0
				if recency < 0 {
					recency = 0
				}
				if recency > 1 {
					recency = 1
				}
			}
		}
	}
	return 0.45*ratingNorm + 0.20*reviewNorm + 0.20*authorTrust + 0.10*siteTrust + 0.05*recency
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
