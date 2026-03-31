package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

func newPulseCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pulse <region>",
		Short: "Market pulse snapshot for a region",
		Long: `Show a concise market dashboard for a region including median price,
price changes, active listings, days on market, inventory direction,
and market temperature.`,
		Example: `  # Market pulse for San Francisco
  redfin-pp-cli pulse "San Francisco, CA"

  # Compact JSON output
  redfin-pp-cli pulse "Seattle, WA" --json --compact`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			regionQuery := args[0]

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Step 1: Resolve region via autocomplete
			fmt.Fprintf(os.Stderr, "Resolving region: %s\n", regionQuery)
			acData, err := c.Get("/stingray/do/location-autocomplete", map[string]string{
				"location": regionQuery,
				"v":        "2",
			})
			if err != nil {
				return classifyAPIError(err)
			}

			if flags.dryRun {
				return nil
			}

			regionID, regionType, regionName, err := extractRegionFromAutocomplete(acData)
			if err != nil {
				return fmt.Errorf("could not resolve region %q: %w", regionQuery, err)
			}
			fmt.Fprintf(os.Stderr, "Resolved to: %s (region_id=%s, region_type=%s)\n", regionName, regionID, regionType)

			// Step 2: Fetch aggregate trends
			rtCode := mapRegionTypeCode(regionType)
			path := fmt.Sprintf("/stingray/api/region/%s/%s/%s/aggregate-trends", rtCode, regionID, rtCode)
			trendsData, err := c.Get(path, map[string]string{})
			if err != nil {
				path = fmt.Sprintf("/stingray/api/region/%s/%s/%s/trends", rtCode, regionID, rtCode)
				trendsData, err = c.Get(path, map[string]string{})
				if err != nil {
					return classifyAPIError(err)
				}
			}

			// Step 3: Fetch active listing count via search
			searchData, err := c.Get("/stingray/api/gis", map[string]string{
				"region_id":   regionID,
				"region_type": regionType,
				"al":          "1",
				"v":           "8",
				"num_homes":   "1",
				"status":      "1",
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not fetch active listings: %v\n", err)
			}

			// Parse trends
			trends := parseTrendsPayload(trendsData)

			// Parse active listings count
			activeListings := 0
			if searchData != nil {
				activeListings = extractTotalCount(searchData)
			}

			// Build dashboard
			dashboard := buildPulseDashboard(regionName, trends, activeListings)

			if flags.asJSON {
				return flags.printJSON(cmd, dashboard)
			}

			// Human-friendly display
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Market Pulse: %s\n", regionName)
			fmt.Fprintf(out, "============================\n\n")
			fmt.Fprintf(out, "  Median Price:     %s\n", dashboard["median_price"])
			fmt.Fprintf(out, "  Price Change MoM: %s\n", dashboard["price_change_mom"])
			fmt.Fprintf(out, "  Price Change YoY: %s\n", dashboard["price_change_yoy"])
			fmt.Fprintf(out, "  Active Listings:  %s\n", dashboard["active_listings"])
			fmt.Fprintf(out, "  Median DOM:       %s\n", dashboard["median_dom"])
			fmt.Fprintf(out, "  Inventory:        %s\n", dashboard["inventory_direction"])
			fmt.Fprintf(out, "  Temperature:      %s\n", dashboard["temperature"])

			return nil
		},
	}

	return cmd
}

// parseTrendsPayload extracts key metrics from a trends API response.
func parseTrendsPayload(data json.RawMessage) map[string]any {
	result := map[string]any{}
	if data == nil {
		return result
	}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return result
	}

	// Walk the structure looking for known metric keys
	walkForMetrics(raw, result)
	return result
}

func walkForMetrics(v any, out map[string]any) {
	switch val := v.(type) {
	case map[string]any:
		for _, key := range []string{
			"medianSalePrice", "median_sale_price", "medianPrice",
			"medianDom", "median_dom", "medianDaysOnMarket",
			"monthOverMonth", "month_over_month", "momChange",
			"yearOverYear", "year_over_year", "yoyChange",
			"inventory", "activeListings", "newListings",
			"inventoryChange", "inventory_change",
		} {
			if vv, ok := val[key]; ok && vv != nil {
				out[key] = vv
			}
		}
		for _, child := range val {
			walkForMetrics(child, out)
		}
	case []any:
		for _, item := range val {
			walkForMetrics(item, out)
		}
	}
}

// extractTotalCount tries to find a total count from a GIS search response.
func extractTotalCount(data json.RawMessage) int {
	var wrapper map[string]any
	if json.Unmarshal(data, &wrapper) == nil {
		for _, key := range []string{"totalResultCount", "resultsCount", "total", "count"} {
			if v, ok := wrapper[key]; ok {
				if n, ok := v.(float64); ok {
					return int(n)
				}
			}
		}
		// Try payload.totalResultCount
		if payload, ok := wrapper["payload"]; ok {
			if pm, ok := payload.(map[string]any); ok {
				for _, key := range []string{"totalResultCount", "resultsCount", "total"} {
					if v, ok := pm[key]; ok {
						if n, ok := v.(float64); ok {
							return int(n)
						}
					}
				}
			}
		}
	}

	// Fallback: count homes array
	homes := extractHomesFromGIS(data)
	return len(homes)
}

func buildPulseDashboard(regionName string, trends map[string]any, activeListings int) map[string]string {
	dash := map[string]string{
		"region":              regionName,
		"median_price":        "N/A",
		"price_change_mom":    "N/A",
		"price_change_yoy":    "N/A",
		"active_listings":     "N/A",
		"median_dom":          "N/A",
		"inventory_direction": "N/A",
		"temperature":         "N/A",
	}

	// Median price
	for _, key := range []string{"medianSalePrice", "median_sale_price", "medianPrice"} {
		if v, ok := trends[key]; ok {
			if n, ok := v.(float64); ok {
				dash["median_price"] = "$" + formatCompact(int64(n))
				break
			}
		}
	}

	// MoM change
	for _, key := range []string{"monthOverMonth", "month_over_month", "momChange"} {
		if v, ok := trends[key]; ok {
			if n, ok := v.(float64); ok {
				dash["price_change_mom"] = fmt.Sprintf("%.1f%%", n*100)
				break
			}
		}
	}

	// YoY change
	for _, key := range []string{"yearOverYear", "year_over_year", "yoyChange"} {
		if v, ok := trends[key]; ok {
			if n, ok := v.(float64); ok {
				dash["price_change_yoy"] = fmt.Sprintf("%.1f%%", n*100)
				break
			}
		}
	}

	// Active listings
	if activeListings > 0 {
		dash["active_listings"] = strconv.Itoa(activeListings)
	}

	// Median DOM
	medianDOM := 0.0
	for _, key := range []string{"medianDom", "median_dom", "medianDaysOnMarket"} {
		if v, ok := trends[key]; ok {
			if n, ok := v.(float64); ok {
				medianDOM = n
				dash["median_dom"] = fmt.Sprintf("%.0f days", n)
				break
			}
		}
	}

	// Inventory direction
	for _, key := range []string{"inventoryChange", "inventory_change"} {
		if v, ok := trends[key]; ok {
			if n, ok := v.(float64); ok {
				switch {
				case n > 0.05:
					dash["inventory_direction"] = "Rising"
				case n < -0.05:
					dash["inventory_direction"] = "Falling"
				default:
					dash["inventory_direction"] = "Stable"
				}
				break
			}
		}
	}

	// Temperature based on DOM + inventory
	dash["temperature"] = computeTemperature(medianDOM, dash["inventory_direction"])

	return dash
}

func computeTemperature(dom float64, inventoryDir string) string {
	if dom == 0 {
		return "N/A"
	}
	switch {
	case dom <= 14 && inventoryDir != "Rising":
		return "Hot"
	case dom <= 30:
		return "Warm"
	case dom <= 60:
		return "Cool"
	default:
		return "Cold"
	}
}
