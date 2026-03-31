package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dataCenterDatasets = map[string]struct {
	Description string
	URL         string
}{
	"weekly": {
		Description: "Weekly Housing Market Data (most recent)",
		URL:         "https://redfin-public-data.s3.us-west-2.amazonaws.com/redfin_market_tracker/weekly_housing_market_data_most_recent.tsv000.gz",
	},
	"monthly": {
		Description: "Monthly Housing Market Data",
		URL:         "https://redfin-public-data.s3.us-west-2.amazonaws.com/redfin_market_tracker/redfin_monthly_market_tracker.tsv000.gz",
	},
	"city-monthly": {
		Description: "City-level Monthly Market Tracker",
		URL:         "https://redfin-public-data.s3.us-west-2.amazonaws.com/redfin_market_tracker/city_market_tracker.tsv000.gz",
	},
	"county-monthly": {
		Description: "County-level Monthly Market Tracker",
		URL:         "https://redfin-public-data.s3.us-west-2.amazonaws.com/redfin_market_tracker/county_market_tracker.tsv000.gz",
	},
	"zip-monthly": {
		Description: "Zip-code-level Monthly Market Tracker",
		URL:         "https://redfin-public-data.s3.us-west-2.amazonaws.com/redfin_market_tracker/zip_code_market_tracker.tsv000.gz",
	},
	"neighborhood-monthly": {
		Description: "Neighborhood-level Monthly Market Tracker",
		URL:         "https://redfin-public-data.s3.us-west-2.amazonaws.com/redfin_market_tracker/neighborhood_market_tracker.tsv000.gz",
	},
}

func newDataCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data",
		Short: "Redfin Data Center datasets",
		Long:  `Download and manage Redfin Data Center public datasets (TSV/gzip).`,
	}

	cmd.AddCommand(newDataListCmd(flags))
	cmd.AddCommand(newDataDownloadCmd(flags))

	return cmd
}

func newDataListCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available datasets",
		Example: `  redfin-pp-cli data list
  redfin-pp-cli data list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.asJSON {
				type dsEntry struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					URL         string `json:"url"`
				}
				var entries []dsEntry
				for name, ds := range dataCenterDatasets {
					entries = append(entries, dsEntry{
						Name:        name,
						Description: ds.Description,
						URL:         ds.URL,
					})
				}
				return flags.printJSON(cmd, entries)
			}

			headers := []string{"DATASET", "DESCRIPTION", "URL"}
			var rows [][]string
			for name, ds := range dataCenterDatasets {
				rows = append(rows, []string{name, ds.Description, ds.URL})
			}

			return flags.printTable(cmd, headers, rows)
		},
	}
}

func newDataDownloadCmd(flags *rootFlags) *cobra.Command {
	var region string

	cmd := &cobra.Command{
		Use:   "download <dataset>",
		Short: "Download a dataset from Redfin Data Center",
		Long: `Download a TSV dataset from Redfin Data Center. Available datasets:
  weekly, monthly, city-monthly, county-monthly, zip-monthly, neighborhood-monthly

Files are saved to the current directory as .tsv.gz files.`,
		Example: `  redfin-pp-cli data download weekly
  redfin-pp-cli data download monthly --region national`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dataset := args[0]

			ds, ok := dataCenterDatasets[dataset]
			if !ok {
				return fmt.Errorf("unknown dataset %q. Available: weekly, monthly, city-monthly, county-monthly, zip-monthly, neighborhood-monthly", dataset)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Dataset:     %s\n", ds.Description)
			fmt.Fprintf(out, "Source URL:  %s\n", ds.URL)
			fmt.Fprintf(out, "\n")
			fmt.Fprintf(out, "[Stub] Download not yet implemented.\n")
			fmt.Fprintf(out, "To download manually:\n")
			fmt.Fprintf(out, "  curl -o %s.tsv.gz %q\n", dataset, ds.URL)
			fmt.Fprintf(out, "  gunzip %s.tsv.gz\n", dataset)

			if region != "" {
				fmt.Fprintf(out, "\nRegion filter: %s (will be applied after download)\n", region)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&region, "region", "", "Filter by region (e.g., national, state:CA)")

	return cmd
}
