package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newTrendsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trends <region-query>",
		Short: "Market trends and regional statistics",
		Long: `View market trends for a region. Resolves the region by name via autocomplete,
then fetches aggregate trend data.`,
		Example: `  # Market snapshot for a city
  redfin-pp-cli trends "San Francisco, CA"

  # Compare two regions side by side
  redfin-pp-cli trends compare "San Francisco, CA" "Oakland, CA"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			regionQuery := args[0]

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Resolve region via autocomplete
			fmt.Fprintf(os.Stderr, "Resolving region: %s\n", regionQuery)
			acData, err := c.Get("/stingray/do/location-autocomplete", map[string]string{
				"location": regionQuery,
				"v":        "2",
			})
			if err != nil {
				return classifyAPIError(err)
			}

			regionID, regionType, regionName, err := extractRegionFromAutocomplete(acData)
			if err != nil {
				return fmt.Errorf("could not resolve region %q: %w", regionQuery, err)
			}
			fmt.Fprintf(os.Stderr, "Resolved to: %s (region_id=%s, region_type=%s)\n", regionName, regionID, regionType)

			// Fetch aggregate trends
			rtCode := mapRegionTypeCode(regionType)
			path := fmt.Sprintf("/stingray/api/region/%s/%s/%s/aggregate-trends", rtCode, regionID, rtCode)
			data, err := c.Get(path, map[string]string{})
			if err != nil {
				// Fallback: try the trends endpoint
				path = fmt.Sprintf("/stingray/api/region/%s/%s/%s/trends", rtCode, regionID, rtCode)
				data, err = c.Get(path, map[string]string{})
				if err != nil {
					return classifyAPIError(err)
				}
			}

			return printPropertyOutput(cmd, data, flags)
		},
	}

	cmd.AddCommand(newTrendsCompareCmd(flags))

	return cmd
}

func newTrendsCompareCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare <region1> <region2>",
		Short: "Compare market trends between two regions",
		Example: `  redfin-pp-cli trends compare "San Francisco, CA" "Oakland, CA"
  redfin-pp-cli trends compare "Seattle, WA" "Portland, OR" --json`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			type regionResult struct {
				Name string          `json:"region_name"`
				Data json.RawMessage `json:"trends"`
			}

			var results []regionResult

			for _, query := range args {
				fmt.Fprintf(os.Stderr, "Resolving region: %s\n", query)
				acData, acErr := c.Get("/stingray/do/location-autocomplete", map[string]string{
					"location": query,
					"v":        "2",
				})
				if acErr != nil {
					return classifyAPIError(acErr)
				}

				regionID, regionType, regionName, rErr := extractRegionFromAutocomplete(acData)
				if rErr != nil {
					return fmt.Errorf("could not resolve region %q: %w", query, rErr)
				}
				fmt.Fprintf(os.Stderr, "Resolved to: %s\n", regionName)

				rtCode := mapRegionTypeCode(regionType)
				path := fmt.Sprintf("/stingray/api/region/%s/%s/%s/aggregate-trends", rtCode, regionID, rtCode)
				data, tErr := c.Get(path, map[string]string{})
				if tErr != nil {
					path = fmt.Sprintf("/stingray/api/region/%s/%s/%s/trends", rtCode, regionID, rtCode)
					data, tErr = c.Get(path, map[string]string{})
					if tErr != nil {
						return classifyAPIError(tErr)
					}
				}

				results = append(results, regionResult{
					Name: regionName,
					Data: data,
				})
			}

			combined, _ := json.Marshal(map[string]any{
				"comparison": results,
			})
			return printPropertyOutput(cmd, json.RawMessage(combined), flags)
		},
	}

	return cmd
}

// mapRegionTypeCode converts a numeric region type to the string code used in API paths.
// Known types: 2=county, 5=zip, 6=city, 8=neighborhood
func mapRegionTypeCode(rt string) string {
	switch rt {
	case "2":
		return "county"
	case "5":
		return "zip"
	case "6":
		return "city"
	case "8":
		return "neighborhood"
	default:
		// If it's already a string code, pass through
		return rt
	}
}
