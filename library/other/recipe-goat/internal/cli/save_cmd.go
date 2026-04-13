package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/other/recipe-goat/internal/recipes"

	"github.com/spf13/cobra"
)

func newSaveCmd(flags *rootFlags) *cobra.Command {
	var (
		tagsCSV   string
		fromStdin bool
	)
	cmd := &cobra.Command{
		Use:     "save <url>",
		Short:   "Save a recipe to the local cookbook",
		Example: "  recipe-goat-pp-cli save https://www.food52.com/recipes/chicken-paprikash --tags weeknight",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var urls []string
			if fromStdin {
				sc := bufio.NewScanner(os.Stdin)
				for sc.Scan() {
					if l := strings.TrimSpace(sc.Text()); l != "" {
						urls = append(urls, l)
					}
				}
			} else {
				if len(args) == 0 {
					return usageErr(fmt.Errorf("provide a URL or use --stdin"))
				}
				urls = []string{args[0]}
			}
			if len(urls) == 0 {
				return usageErr(fmt.Errorf("no URLs provided"))
			}

			tags := []string{}
			for _, t := range strings.Split(tagsCSV, ",") {
				if t = strings.TrimSpace(t); t != "" {
					tags = append(tags, t)
				}
			}

			st, err := openRecipeStore()
			if err != nil {
				return err
			}
			defer st.Close()

			client := httpClientForSites(flags.timeout)
			results := []map[string]any{}
			failures := 0
			successes := 0
			for _, u := range urls {
				ctx, cancel := flags.withContext()
				r, err := recipes.Fetch(ctx, client, u)
				cancel()
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "fetch %s: %v\n", u, err)
					results = append(results, map[string]any{"url": u, "error": err.Error()})
					failures++
					continue
				}
				if flags.dryRun {
					fmt.Fprintf(cmd.OutOrStdout(), "would save: %s (%s)\n", r.Name, r.URL)
					successes++
					continue
				}
				id, err := st.SaveRecipe(recipeToStored(r))
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "save %s: %v\n", u, err)
					results = append(results, map[string]any{"url": u, "error": err.Error()})
					failures++
					continue
				}
				if len(tags) > 0 {
					if err := st.TagRecipe(id, tags); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "tag: %v\n", err)
					}
				}
				fmt.Fprintf(cmd.OutOrStdout(), "saved: %d %s\n", id, r.Name)
				results = append(results, map[string]any{"id": id, "title": r.Name, "url": r.URL})
				successes++
				// Small pacing between stdin entries so we don't hammer sites.
				if fromStdin && len(urls) > 1 {
					time.Sleep(500 * time.Millisecond)
				}
			}
			if flags.asJSON {
				if err := flags.printJSON(cmd, results); err != nil {
					return err
				}
			}
			// Non-zero exit if nothing saved successfully — but not in dry-run
			// (fetch failures on placeholder URLs are expected there).
			if !flags.dryRun && successes == 0 && failures > 0 {
				return apiErr(fmt.Errorf("all %d save(s) failed", failures))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tagsCSV, "tags", "", "Comma-separated tags to attach")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "Read URLs from stdin, one per line")
	return cmd
}
