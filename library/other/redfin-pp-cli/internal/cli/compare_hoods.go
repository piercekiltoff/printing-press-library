package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newCompareHoodsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare-hoods <neighborhood1> <neighborhood2>",
		Short: "Side-by-side neighborhood comparison",
		Long: `Compare two neighborhoods side by side. Resolves both via autocomplete,
fetches neighborhood stats and trends, then displays a comparison table
with walk/bike/transit scores, median price, DOM, and price trend.`,
		Example: `  # Compare two SF neighborhoods
  redfin-pp-cli compare-hoods "Mission District, SF" "Castro, SF"

  # JSON output
  redfin-pp-cli compare-hoods "Capitol Hill, Seattle" "Ballard, Seattle" --json`,
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
				for _, query := range args {
					_, _ = c.Get("/stingray/do/location-autocomplete", map[string]string{
						"location": query,
						"v":        "2",
					})
				}
				return nil
			}

			type hoodData struct {
				Name   string         `json:"name"`
				Stats  map[string]any `json:"stats"`
				Trends map[string]any `json:"trends"`
			}

			var hoods []hoodData

			for _, query := range args {
				fmt.Fprintf(os.Stderr, "Resolving neighborhood: %s\n", query)
				acData, acErr := c.Get("/stingray/do/location-autocomplete", map[string]string{
					"location": query,
					"v":        "2",
				})
				if acErr != nil {
					return classifyAPIError(acErr)
				}

				regionID, regionType, regionName, rErr := extractRegionFromAutocomplete(acData)
				if rErr != nil {
					return fmt.Errorf("could not resolve neighborhood %q: %w", query, rErr)
				}
				fmt.Fprintf(os.Stderr, "Resolved to: %s (region_id=%s)\n", regionName, regionID)

				hood := hoodData{
					Name:   regionName,
					Stats:  map[string]any{},
					Trends: map[string]any{},
				}

				// Fetch neighborhood stats (walk/bike/transit scores)
				// The neighborhood stats endpoint uses propertyId, but we can try region-based
				// For neighborhoods, try the region trends endpoint
				rtCode := mapRegionTypeCode(regionType)

				// Fetch trends
				trendPath := fmt.Sprintf("/stingray/api/region/%s/%s/%s/aggregate-trends", rtCode, regionID, rtCode)
				trendsData, tErr := c.Get(trendPath, map[string]string{})
				if tErr != nil {
					trendPath = fmt.Sprintf("/stingray/api/region/%s/%s/%s/trends", rtCode, regionID, rtCode)
					trendsData, tErr = c.Get(trendPath, map[string]string{})
					if tErr != nil {
						fmt.Fprintf(os.Stderr, "warning: could not fetch trends for %s: %v\n", regionName, tErr)
					}
				}
				if trendsData != nil {
					hood.Trends = parseTrendsPayload(trendsData)
				}

				// Try to get stats via region info
				statsPath := fmt.Sprintf("/stingray/api/region/%s/%s/stats", rtCode, regionID)
				statsData, sErr := c.Get(statsPath, map[string]string{})
				if sErr == nil && statsData != nil {
					var statsMap map[string]any
					if json.Unmarshal(statsData, &statsMap) == nil {
						hood.Stats = statsMap
					}
				}

				hoods = append(hoods, hood)
			}

			if flags.asJSON {
				return flags.printJSON(cmd, map[string]any{
					"comparison": hoods,
				})
			}

			// Human-friendly comparison table
			headers := []string{"METRIC", hoods[0].Name, hoods[1].Name}
			rows := [][]string{
				{"Walk Score", extractScoreStr(hoods[0].Stats, "walkScore", "walk_score"), extractScoreStr(hoods[1].Stats, "walkScore", "walk_score")},
				{"Bike Score", extractScoreStr(hoods[0].Stats, "bikeScore", "bike_score"), extractScoreStr(hoods[1].Stats, "bikeScore", "bike_score")},
				{"Transit Score", extractScoreStr(hoods[0].Stats, "transitScore", "transit_score"), extractScoreStr(hoods[1].Stats, "transitScore", "transit_score")},
				{"Median Price", extractTrendPrice(hoods[0].Trends), extractTrendPrice(hoods[1].Trends)},
				{"Median DOM", extractTrendDOM(hoods[0].Trends), extractTrendDOM(hoods[1].Trends)},
				{"Price Trend", extractPriceTrend(hoods[0].Trends), extractPriceTrend(hoods[1].Trends)},
			}

			return flags.printTable(cmd, headers, rows)
		},
	}

	return cmd
}

func extractScoreStr(stats map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := stats[k]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
	}
	// Try nested: payload -> walkScore etc.
	if payload, ok := stats["payload"]; ok {
		if pm, ok := payload.(map[string]any); ok {
			for _, k := range keys {
				if v, ok := pm[k]; ok && v != nil {
					return fmt.Sprintf("%v", v)
				}
			}
		}
	}
	return "N/A"
}

func extractTrendPrice(trends map[string]any) string {
	for _, key := range []string{"medianSalePrice", "median_sale_price", "medianPrice"} {
		if v, ok := trends[key]; ok {
			if n, ok := v.(float64); ok {
				return "$" + formatCompact(int64(n))
			}
		}
	}
	return "N/A"
}

func extractTrendDOM(trends map[string]any) string {
	for _, key := range []string{"medianDom", "median_dom", "medianDaysOnMarket"} {
		if v, ok := trends[key]; ok {
			if n, ok := v.(float64); ok {
				return fmt.Sprintf("%.0f days", n)
			}
		}
	}
	return "N/A"
}

func extractPriceTrend(trends map[string]any) string {
	for _, key := range []string{"yearOverYear", "year_over_year", "yoyChange", "monthOverMonth", "month_over_month"} {
		if v, ok := trends[key]; ok {
			if n, ok := v.(float64); ok {
				if n > 0 {
					return fmt.Sprintf("+%.1f%%", n*100)
				}
				return fmt.Sprintf("%.1f%%", n*100)
			}
		}
	}
	return "N/A"
}
