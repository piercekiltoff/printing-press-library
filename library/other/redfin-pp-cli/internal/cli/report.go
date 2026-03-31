package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newReportCmd(flags *rootFlags) *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "report <region>",
		Short: "Generate a market report for a region",
		Long: `Generate a comprehensive market report for a region including market overview,
price analysis, inventory analysis, market velocity, and outlook.

Output as plain text (default) or markdown.`,
		Example: `  # Text report
  redfin-pp-cli report "Austin, TX"

  # Markdown report
  redfin-pp-cli report "Austin, TX" --format markdown

  # JSON data
  redfin-pp-cli report "Austin, TX" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			regionQuery := args[0]

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Step 1: Resolve region
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
			fmt.Fprintf(os.Stderr, "Resolved to: %s\n", regionName)

			// Step 2: Fetch trends
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

			// Step 3: Fetch active listings count
			searchData, _ := c.Get("/stingray/api/gis", map[string]string{
				"region_id":   regionID,
				"region_type": regionType,
				"al":          "1",
				"v":           "8",
				"num_homes":   "5",
				"status":      "1",
			})
			activeCount := 0
			if searchData != nil {
				activeCount = extractTotalCount(searchData)
			}

			// Step 4: Fetch recent sold
			soldData, _ := c.Get("/stingray/api/gis", map[string]string{
				"region_id":   regionID,
				"region_type": regionType,
				"al":          "1",
				"v":           "8",
				"num_homes":   "10",
				"status":      "9",
				"ord":         "days-on-redfin-asc",
			})
			recentSold := 0
			if soldData != nil {
				homes := extractHomesFromGIS(soldData)
				recentSold = len(homes)
			}

			// Parse trends
			trends := parseTrendsPayload(trendsData)
			dash := buildPulseDashboard(regionName, trends, activeCount)

			// Build report data
			reportData := map[string]any{
				"region":          regionName,
				"generated_at":    time.Now().Format("2006-01-02"),
				"median_price":    dash["median_price"],
				"price_change_mom": dash["price_change_mom"],
				"price_change_yoy": dash["price_change_yoy"],
				"active_listings": dash["active_listings"],
				"recent_sold":     recentSold,
				"median_dom":      dash["median_dom"],
				"inventory_dir":   dash["inventory_direction"],
				"temperature":     dash["temperature"],
			}

			if flags.asJSON {
				return flags.printJSON(cmd, reportData)
			}

			out := cmd.OutOrStdout()

			if format == "markdown" {
				return renderMarkdownReport(out, reportData)
			}
			return renderTextReport(out, reportData)
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "Output format: text or markdown")

	return cmd
}

func renderTextReport(w interface{ Write([]byte) (int, error) }, data map[string]any) error {
	region := fmt.Sprintf("%v", data["region"])
	date := fmt.Sprintf("%v", data["generated_at"])

	lines := []string{
		fmt.Sprintf("MARKET REPORT: %s", strings.ToUpper(region)),
		fmt.Sprintf("Generated: %s", date),
		"",
		"== Market Overview ==",
		fmt.Sprintf("  Temperature:     %v", data["temperature"]),
		fmt.Sprintf("  Median Price:    %v", data["median_price"]),
		fmt.Sprintf("  Active Listings: %v", data["active_listings"]),
		"",
		"== Price Analysis ==",
		fmt.Sprintf("  Median Price:     %v", data["median_price"]),
		fmt.Sprintf("  Month-over-Month: %v", data["price_change_mom"]),
		fmt.Sprintf("  Year-over-Year:   %v", data["price_change_yoy"]),
		"",
		"== Inventory Analysis ==",
		fmt.Sprintf("  Active Listings: %v", data["active_listings"]),
		fmt.Sprintf("  Recent Sales:    %v", data["recent_sold"]),
		fmt.Sprintf("  Direction:       %v", data["inventory_dir"]),
		"",
		"== Market Velocity ==",
		fmt.Sprintf("  Median DOM: %v", data["median_dom"]),
		"",
		"== Outlook ==",
		fmt.Sprintf("  Market temperature is %v.", data["temperature"]),
	}

	temp := fmt.Sprintf("%v", data["temperature"])
	switch temp {
	case "Hot":
		lines = append(lines, "  Expect competitive conditions with multiple offers and fast closings.")
	case "Warm":
		lines = append(lines, "  Market favors sellers but buyers have some negotiating room.")
	case "Cool":
		lines = append(lines, "  Buyers have leverage with moderate inventory and longer selling times.")
	case "Cold":
		lines = append(lines, "  Buyer's market with ample inventory and significant negotiating power.")
	}

	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return nil
}

func renderMarkdownReport(w interface{ Write([]byte) (int, error) }, data map[string]any) error {
	region := fmt.Sprintf("%v", data["region"])
	date := fmt.Sprintf("%v", data["generated_at"])

	lines := []string{
		fmt.Sprintf("# Market Report: %s", region),
		fmt.Sprintf("*Generated: %s*", date),
		"",
		"## Market Overview",
		fmt.Sprintf("- **Temperature:** %v", data["temperature"]),
		fmt.Sprintf("- **Median Price:** %v", data["median_price"]),
		fmt.Sprintf("- **Active Listings:** %v", data["active_listings"]),
		"",
		"## Price Analysis",
		fmt.Sprintf("| Metric | Value |"),
		fmt.Sprintf("|--------|-------|"),
		fmt.Sprintf("| Median Price | %v |", data["median_price"]),
		fmt.Sprintf("| Month-over-Month | %v |", data["price_change_mom"]),
		fmt.Sprintf("| Year-over-Year | %v |", data["price_change_yoy"]),
		"",
		"## Inventory Analysis",
		fmt.Sprintf("| Metric | Value |"),
		fmt.Sprintf("|--------|-------|"),
		fmt.Sprintf("| Active Listings | %v |", data["active_listings"]),
		fmt.Sprintf("| Recent Sales | %v |", data["recent_sold"]),
		fmt.Sprintf("| Direction | %v |", data["inventory_dir"]),
		"",
		"## Market Velocity",
		fmt.Sprintf("- **Median DOM:** %v", data["median_dom"]),
		"",
		"## Outlook",
		fmt.Sprintf("Market temperature is **%v**.", data["temperature"]),
	}

	temp := fmt.Sprintf("%v", data["temperature"])
	switch temp {
	case "Hot":
		lines = append(lines, "Expect competitive conditions with multiple offers and fast closings.")
	case "Warm":
		lines = append(lines, "Market favors sellers but buyers have some negotiating room.")
	case "Cool":
		lines = append(lines, "Buyers have leverage with moderate inventory and longer selling times.")
	case "Cold":
		lines = append(lines, "Buyer's market with ample inventory and significant negotiating power.")
	}

	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return nil
}
