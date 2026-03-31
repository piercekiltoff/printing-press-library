package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/other/redfin-pp-cli/internal/store"

	"github.com/spf13/cobra"
)

func newSearchCmd(flags *rootFlags) *cobra.Command {
	var minPrice, maxPrice int
	var beds, baths int
	var minSqft, maxSqft int
	var propType string
	var status string
	var sortOrder string
	var limit int
	var save bool
	var offline bool
	var dbPath string

	cmd := &cobra.Command{
		Use:   "search <location>",
		Short: "Search properties by location with filters",
		Long: `Search for properties by location name. Resolves the location via autocomplete,
then searches for matching properties. Results are displayed in a human-friendly table.

Use --save to persist results to local SQLite for offline access.
Use --offline to search only the local database (requires prior --save).`,
		Example: `  # Search active listings in San Francisco
  redfin-pp-cli search "San Francisco, CA"

  # Filter by price and bedrooms
  redfin-pp-cli search "Seattle, WA" --min-price 500000 --max-price 1000000 --beds 3

  # Search condos sorted by newest
  redfin-pp-cli search "Austin, TX" --type condo --sort newest --limit 20

  # Search sold homes and save to local database
  redfin-pp-cli search "Denver, CO" --status sold --save

  # Search local database offline
  redfin-pp-cli search "Portland" --offline`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			locationQuery := args[0]

			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".local", "share", "redfin-pp-cli", "redfin.db")
			}

			// Offline mode: search local SQLite only
			if offline {
				db, err := store.Open(dbPath)
				if err != nil {
					return fmt.Errorf("opening local database: %w", err)
				}
				defer db.Close()

				results, err := db.Search(locationQuery, limit)
				if err != nil {
					return fmt.Errorf("searching local database: %w", err)
				}
				if len(results) == 0 {
					fmt.Fprintf(os.Stderr, "No results found in local database for %q. Run a search with --save first.\n", locationQuery)
					return nil
				}
				combined, _ := json.Marshal(results)
				return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(combined), flags)
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Step 1: Resolve location via autocomplete
			fmt.Fprintf(os.Stderr, "Resolving location: %s\n", locationQuery)
			acData, err := c.Get("/stingray/do/location-autocomplete", map[string]string{
				"location": locationQuery,
				"v":        "2",
			})
			if err != nil {
				return classifyAPIError(err)
			}

			regionID, regionType, regionName, err := extractRegionFromAutocomplete(acData)
			if err != nil {
				return fmt.Errorf("could not resolve location %q: %w", locationQuery, err)
			}
			fmt.Fprintf(os.Stderr, "Resolved to: %s (region_id=%s, region_type=%s)\n", regionName, regionID, regionType)

			// Step 2: Build search params
			params := map[string]string{
				"region_id":   regionID,
				"region_type": regionType,
				"al":          "1",
				"v":           "8",
			}

			if limit > 0 {
				params["num_homes"] = strconv.Itoa(limit)
			} else {
				params["num_homes"] = "350"
			}

			if minPrice > 0 {
				params["min_price"] = strconv.Itoa(minPrice)
			}
			if maxPrice > 0 {
				params["max_price"] = strconv.Itoa(maxPrice)
			}
			if beds > 0 {
				params["min_beds"] = strconv.Itoa(beds)
			}
			if baths > 0 {
				params["min_baths"] = strconv.Itoa(baths)
			}
			if minSqft > 0 {
				params["min_sqft"] = strconv.Itoa(minSqft)
			}
			if maxSqft > 0 {
				params["max_sqft"] = strconv.Itoa(maxSqft)
			}

			// Map property type to uipt codes
			if propType != "" {
				switch strings.ToLower(propType) {
				case "house":
					params["uipt"] = "1"
				case "condo":
					params["uipt"] = "2"
				case "townhouse":
					params["uipt"] = "3"
				case "land":
					params["uipt"] = "5"
				default:
					params["uipt"] = propType
				}
			}

			// Map status
			if status != "" {
				switch strings.ToLower(status) {
				case "active":
					params["status"] = "1"
				case "sold":
					params["status"] = "9"
				default:
					params["status"] = status
				}
			}

			// Map sort order
			if sortOrder != "" {
				switch strings.ToLower(sortOrder) {
				case "price":
					params["ord"] = "price-asc"
				case "newest":
					params["ord"] = "days-on-redfin-asc"
				case "dom":
					params["ord"] = "days-on-redfin-desc"
				default:
					params["ord"] = sortOrder
				}
			}

			// Step 3: Execute search
			data, err := c.Get("/stingray/api/gis", params)
			if err != nil {
				return classifyAPIError(err)
			}

			// Step 4: Parse and display results
			homes := extractHomesFromGIS(data)

			if len(homes) == 0 {
				fmt.Fprintf(os.Stderr, "No properties found matching your criteria.\n")
				return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
			}

			fmt.Fprintf(os.Stderr, "Found %d properties\n", len(homes))

			// Step 5: Optionally save to SQLite
			if save {
				db, err := store.Open(dbPath)
				if err != nil {
					return fmt.Errorf("opening local database: %w", err)
				}
				defer db.Close()

				for _, home := range homes {
					raw, _ := json.Marshal(home)
					id := ""
					if pid, ok := home["propertyId"]; ok {
						id = fmt.Sprintf("%v", pid)
					} else if mlsid, ok := home["mlsId"]; ok {
						id = fmt.Sprintf("%v", mlsid)
					}
					if id != "" {
						if err := db.Upsert("property", id, json.RawMessage(raw)); err != nil {
							fmt.Fprintf(os.Stderr, "warning: failed to save property %s: %v\n", id, err)
						}
					}
				}
				fmt.Fprintf(os.Stderr, "Saved %d properties to local database\n", len(homes))
			}

			// Human-friendly table output
			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				headers := []string{"ADDRESS", "PRICE", "BEDS", "BATHS", "SQFT", "DOM", "STATUS"}
				var rows [][]string
				for _, h := range homes {
					rows = append(rows, []string{
						extractStr(h, "streetAddress", "address"),
						formatPrice(h),
						extractNumStr(h, "beds", "numBeds"),
						extractNumStr(h, "baths", "numBaths"),
						extractNumStr(h, "sqFt", "sqft", "sqftTotal"),
						extractNumStr(h, "dom", "daysOnMarket", "timeOnRedfin"),
						extractStr(h, "listingStatus", "status", "marketStatus"),
					})
				}
				return flags.printTable(cmd, headers, rows)
			}

			// JSON/machine output
			combined, _ := json.Marshal(homes)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(combined), flags)
		},
	}

	cmd.Flags().IntVar(&minPrice, "min-price", 0, "Minimum price")
	cmd.Flags().IntVar(&maxPrice, "max-price", 0, "Maximum price")
	cmd.Flags().IntVar(&beds, "beds", 0, "Minimum bedrooms")
	cmd.Flags().IntVar(&baths, "baths", 0, "Minimum bathrooms")
	cmd.Flags().IntVar(&minSqft, "min-sqft", 0, "Minimum square footage")
	cmd.Flags().IntVar(&maxSqft, "max-sqft", 0, "Maximum square footage")
	cmd.Flags().StringVar(&propType, "type", "", "Property type (house/condo/townhouse/land)")
	cmd.Flags().StringVar(&status, "status", "", "Listing status (active/sold)")
	cmd.Flags().StringVar(&sortOrder, "sort", "", "Sort order (price/newest/dom)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().BoolVar(&save, "save", false, "Persist results to local SQLite")
	cmd.Flags().BoolVar(&offline, "offline", false, "Search local SQLite only (FTS)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/redfin-pp-cli/redfin.db)")

	return cmd
}

// extractRegionFromAutocomplete parses the autocomplete response to find
// the best matching region_id and region_type.
func extractRegionFromAutocomplete(data json.RawMessage) (regionID, regionType, name string, err error) {
	// The autocomplete response can have various structures.
	// Try to find a payload with regions/suggestions.
	var raw map[string]json.RawMessage
	if jsonErr := json.Unmarshal(data, &raw); jsonErr == nil {
		// Try payload.sections or payload.exactMatch
		if payload, ok := raw["payload"]; ok {
			return extractRegionFromPayload(payload)
		}
		// Try direct fields
		if sections, ok := raw["sections"]; ok {
			return extractRegionFromSections(sections)
		}
	}

	// Try as array of suggestions
	var suggestions []map[string]any
	if jsonErr := json.Unmarshal(data, &suggestions); jsonErr == nil && len(suggestions) > 0 {
		return extractRegionFromSuggestion(suggestions[0])
	}

	// Try to walk arbitrary structure for common fields
	var generic any
	if json.Unmarshal(data, &generic) == nil {
		rid, rt, n := walkForRegion(generic)
		if rid != "" {
			return rid, rt, n, nil
		}
	}

	return "", "", "", fmt.Errorf("no region found in autocomplete response")
}

func extractRegionFromPayload(data json.RawMessage) (string, string, string, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", "", "", err
	}

	// Try exactMatch first
	if em, ok := payload["exactMatch"]; ok {
		var match map[string]any
		if json.Unmarshal(em, &match) == nil {
			return extractRegionFromSuggestion(match)
		}
	}

	// Try sections
	if sections, ok := payload["sections"]; ok {
		return extractRegionFromSections(sections)
	}

	return "", "", "", fmt.Errorf("no region in payload")
}

func extractRegionFromSections(data json.RawMessage) (string, string, string, error) {
	var sections []json.RawMessage
	if err := json.Unmarshal(data, &sections); err != nil {
		// Try as map
		var sectionsMap map[string]json.RawMessage
		if json.Unmarshal(data, &sectionsMap) == nil {
			for _, v := range sectionsMap {
				var items []map[string]any
				if json.Unmarshal(v, &items) == nil && len(items) > 0 {
					return extractRegionFromSuggestion(items[0])
				}
			}
		}
		return "", "", "", err
	}

	for _, section := range sections {
		var sec map[string]json.RawMessage
		if json.Unmarshal(section, &sec) == nil {
			if rows, ok := sec["rows"]; ok {
				var items []map[string]any
				if json.Unmarshal(rows, &items) == nil && len(items) > 0 {
					return extractRegionFromSuggestion(items[0])
				}
			}
		}
	}

	return "", "", "", fmt.Errorf("no region in sections")
}

func extractRegionFromSuggestion(s map[string]any) (string, string, string, error) {
	rid := ""
	rt := ""
	name := ""

	// Try various field names for region_id
	for _, key := range []string{"id", "regionId", "region_id", "tableId"} {
		if v, ok := s[key]; ok {
			rid = fmt.Sprintf("%v", v)
			break
		}
	}

	// Try various field names for region_type
	for _, key := range []string{"type", "regionType", "region_type", "tableType"} {
		if v, ok := s[key]; ok {
			rt = fmt.Sprintf("%v", v)
			break
		}
	}

	// Name
	for _, key := range []string{"name", "displayName", "display_name", "searchDisplay"} {
		if v, ok := s[key]; ok {
			name = fmt.Sprintf("%v", v)
			break
		}
	}

	if rid != "" && rt != "" {
		return rid, rt, name, nil
	}

	// Fallback: check for numeric conversion
	if rid == "" {
		return "", "", "", fmt.Errorf("no region_id found in suggestion")
	}
	return rid, rt, name, nil
}

// walkForRegion recursively searches a structure for region identifiers.
func walkForRegion(v any) (regionID, regionType, name string) {
	switch val := v.(type) {
	case map[string]any:
		// Check if this map has what we need
		rid, rok := val["id"]
		rt, tok := val["type"]
		if !rok {
			rid, rok = val["regionId"]
		}
		if !rok {
			rid, rok = val["region_id"]
		}
		if !tok {
			rt, tok = val["regionType"]
		}
		if !tok {
			rt, tok = val["region_type"]
		}
		if rok && tok {
			n := ""
			if nm, ok := val["name"]; ok {
				n = fmt.Sprintf("%v", nm)
			}
			return fmt.Sprintf("%v", rid), fmt.Sprintf("%v", rt), n
		}
		// Recurse
		for _, child := range val {
			rid, rt, n := walkForRegion(child)
			if rid != "" {
				return rid, rt, n
			}
		}
	case []any:
		for _, item := range val {
			rid, rt, n := walkForRegion(item)
			if rid != "" {
				return rid, rt, n
			}
		}
	}
	return "", "", ""
}

// extractHomesFromGIS pulls the homes array from a GIS search response.
func extractHomesFromGIS(data json.RawMessage) []map[string]any {
	// Try direct array
	var homes []map[string]any
	if json.Unmarshal(data, &homes) == nil && len(homes) > 0 {
		return homes
	}

	// Try wrapper object
	var wrapper map[string]json.RawMessage
	if json.Unmarshal(data, &wrapper) != nil {
		return nil
	}

	// Try common keys
	for _, key := range []string{"homes", "payload", "data", "results"} {
		if raw, ok := wrapper[key]; ok {
			if json.Unmarshal(raw, &homes) == nil && len(homes) > 0 {
				return homes
			}
			// Try nested: payload.homes
			var nested map[string]json.RawMessage
			if json.Unmarshal(raw, &nested) == nil {
				for _, nk := range []string{"homes", "searchResults", "results"} {
					if nr, ok := nested[nk]; ok {
						if json.Unmarshal(nr, &homes) == nil && len(homes) > 0 {
							return homes
						}
					}
				}
			}
		}
	}

	return nil
}

func extractStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func extractNumStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch n := v.(type) {
			case float64:
				if n == float64(int64(n)) {
					return strconv.FormatInt(int64(n), 10)
				}
				return fmt.Sprintf("%.1f", n)
			default:
				return fmt.Sprintf("%v", v)
			}
		}
	}
	return ""
}

func formatPrice(m map[string]any) string {
	for _, k := range []string{"price", "listPrice", "listingPrice", "soldPrice"} {
		if v, ok := m[k]; ok && v != nil {
			switch n := v.(type) {
			case float64:
				if n >= 1_000_000 {
					return fmt.Sprintf("$%.2fM", n/1_000_000)
				}
				return fmt.Sprintf("$%s", formatCompact(int64(n)))
			default:
				return fmt.Sprintf("$%v", v)
			}
		}
	}
	return ""
}
