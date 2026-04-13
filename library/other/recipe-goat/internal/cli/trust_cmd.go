package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/other/recipe-goat/internal/recipes"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

func newTrustCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "trust",
		Short:   "View and override site/author trust scores (ranking integration wip)",
		Example: "  recipe-goat-pp-cli trust list",
	}
	cmd.AddCommand(newTrustListCmd(flags))
	cmd.AddCommand(newTrustSetCmd(flags))
	return cmd
}

func newTrustListCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List site trust scores + curated authors",
		Example: "  recipe-goat-pp-cli trust list",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.asJSON {
				payload := map[string]any{
					"sites":   recipes.Sites,
					"authors": recipes.CuratedAuthors(),
				}
				return flags.printJSON(cmd, payload)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "SITES")
			headers := []string{"SITE", "TIER", "TRUST"}
			rows := make([][]string, 0, len(recipes.Sites))
			for _, s := range recipes.Sites {
				rows = append(rows, []string{s.Hostname, strconv.Itoa(s.Tier), fmt.Sprintf("%.2f", s.Trust)})
			}
			if err := flags.printTable(cmd, headers, rows); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "\nCURATED AUTHORS (trust 1.00)")
			for _, a := range recipes.CuratedAuthors() {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", a)
			}
			return nil
		},
	}
}

// trustOverrides persisted at ~/.config/recipe-goat-pp-cli/trust.toml.
// Shape: [authors] "name" = 0.9   [sites] "host" = 0.8
type trustOverrides struct {
	Authors map[string]float64 `toml:"authors"`
	Sites   map[string]float64 `toml:"sites"`
}

func trustPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "recipe-goat-pp-cli", "trust.toml")
}

func loadTrust() (trustOverrides, error) {
	out := trustOverrides{Authors: map[string]float64{}, Sites: map[string]float64{}}
	data, err := os.ReadFile(trustPath())
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return out, err
	}
	if err := toml.Unmarshal(data, &out); err != nil {
		return out, err
	}
	if out.Authors == nil {
		out.Authors = map[string]float64{}
	}
	if out.Sites == nil {
		out.Sites = map[string]float64{}
	}
	return out, nil
}

func saveTrust(t trustOverrides) error {
	p := trustPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := toml.Marshal(t)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func newTrustSetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "set <author|site> <delta>",
		Short:   "Persist a user trust override (ranking integration wip)",
		Example: "  recipe-goat-pp-cli trust set kenji +2",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			delta, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				return usageErr(fmt.Errorf("delta must be a number, got %q", args[1]))
			}
			overrides, err := loadTrust()
			if err != nil {
				return err
			}
			// Heuristic: if target looks like a hostname (contains a dot and no space), treat as site.
			if strings.Contains(target, ".") && !strings.Contains(target, " ") {
				overrides.Sites[strings.ToLower(target)] = delta
			} else {
				overrides.Authors[strings.ToLower(target)] = delta
			}
			if flags.dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "would save trust override for %s → %.2f\n", target, delta)
				return nil
			}
			if err := saveTrust(overrides); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved: %s → %.2f (ranking integration wip — override stored but not yet applied)\n", target, delta)
			return nil
		},
	}
}
