package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newAnalyzeZipsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze-zips <zip1> <zip2> [zip3] [zip4] [zip5]",
		Short: "Compare multiple zip codes side by side",
		Long: `Analyze and compare multiple zip codes. For each zip, resolves the region,
fetches trends, and displays a comparison table with median price, DOM,
active listings, price trend, and market temperature.`,
		Example: `  # Compare three SF zip codes
  redfin-pp-cli analyze-zips 94110 94114 94102

  # JSON output
  redfin-pp-cli analyze-zips 78701 78704 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return cmd.Help()
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			if flags.dryRun {
				// In dry-run mode, just trigger the API calls so they get logged
				for _, zip := range args {
					_, _ = c.Get("/stingray/do/location-autocomplete", map[string]string{
						"location": zip,
						"v":        "2",
					})
				}
				return nil
			}

			type zipResult struct {
				Zip            string `json:"zip"`
				RegionName     string `json:"region_name"`
				MedianPrice    string `json:"median_price"`
				MedianDOM      string `json:"median_dom"`
				ActiveListings string `json:"active_listings"`
				PriceTrend     string `json:"price_trend"`
				Temperature    string `json:"temperature"`
			}

			var results []zipResult

			for _, zip := range args {
				fmt.Fprintf(os.Stderr, "Resolving zip: %s\n", zip)
				acData, acErr := c.Get("/stingray/do/location-autocomplete", map[string]string{
					"location": zip,
					"v":        "2",
				})
				if acErr != nil {
					fmt.Fprintf(os.Stderr, "warning: could not resolve zip %s: %v\n", zip, acErr)
					results = append(results, zipResult{Zip: zip, RegionName: "Error", MedianPrice: "N/A", MedianDOM: "N/A", ActiveListings: "N/A", PriceTrend: "N/A", Temperature: "N/A"})
					continue
				}

				regionID, regionType, regionName, rErr := extractRegionFromAutocomplete(acData)
				if rErr != nil {
					fmt.Fprintf(os.Stderr, "warning: could not resolve zip %s: %v\n", zip, rErr)
					results = append(results, zipResult{Zip: zip, RegionName: "Error", MedianPrice: "N/A", MedianDOM: "N/A", ActiveListings: "N/A", PriceTrend: "N/A", Temperature: "N/A"})
					continue
				}
				fmt.Fprintf(os.Stderr, "Resolved to: %s\n", regionName)

				// Fetch trends
				rtCode := mapRegionTypeCode(regionType)
				trendPath := fmt.Sprintf("/stingray/api/region/%s/%s/%s/aggregate-trends", rtCode, regionID, rtCode)
				trendsData, tErr := c.Get(trendPath, map[string]string{})
				if tErr != nil {
					trendPath = fmt.Sprintf("/stingray/api/region/%s/%s/%s/trends", rtCode, regionID, rtCode)
					trendsData, tErr = c.Get(trendPath, map[string]string{})
				}

				// Fetch active listings
				searchData, _ := c.Get("/stingray/api/gis", map[string]string{
					"region_id":   regionID,
					"region_type": regionType,
					"al":          "1",
					"v":           "8",
					"num_homes":   "1",
					"status":      "1",
				})

				trends := parseTrendsPayload(trendsData)
				activeCount := 0
				if searchData != nil {
					activeCount = extractTotalCount(searchData)
				}

				dash := buildPulseDashboard(regionName, trends, activeCount)

				results = append(results, zipResult{
					Zip:            zip,
					RegionName:     regionName,
					MedianPrice:    dash["median_price"],
					MedianDOM:      dash["median_dom"],
					ActiveListings: dash["active_listings"],
					PriceTrend:     extractPriceTrend(trends),
					Temperature:    dash["temperature"],
				})
			}

			if flags.asJSON {
				return flags.printJSON(cmd, results)
			}

			headers := []string{"ZIP", "REGION", "MEDIAN PRICE", "DOM", "LISTINGS", "TREND", "TEMP"}
			var rows [][]string
			for _, r := range results {
				rows = append(rows, []string{
					r.Zip,
					r.RegionName,
					r.MedianPrice,
					r.MedianDOM,
					r.ActiveListings,
					r.PriceTrend,
					r.Temperature,
				})
			}

			return flags.printTable(cmd, headers, rows)
		},
	}

	return cmd
}
