package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/mvanhorn/printing-press-library/library/other/redfin-pp-cli/internal/store"

	"github.com/spf13/cobra"
)

func newDealsCmd(flags *rootFlags) *cobra.Command {
	var maxPrice int
	var minDiscount float64
	var limit int
	var save bool
	var dbPath string

	cmd := &cobra.Command{
		Use:   "deals <location>",
		Short: "Find properties priced below their AVM estimate",
		Long: `Search for deal properties where the listing price is significantly below
the automated valuation model (AVM) estimate. Fetches properties in the region,
compares each listing price to its Redfin estimate, and ranks by discount.`,
		Example: `  # Find deals in San Francisco with at least 10% discount
  redfin-pp-cli deals "San Francisco, CA" --min-discount 10

  # Find deals under $800K
  redfin-pp-cli deals "San Francisco, CA" --max-price 800000 --min-discount 10

  # Save deals to local database
  redfin-pp-cli deals "Austin, TX" --save --limit 50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			locationQuery := args[0]

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Step 1: Resolve location
			fmt.Fprintf(os.Stderr, "Resolving location: %s\n", locationQuery)
			acData, err := c.Get("/stingray/do/location-autocomplete", map[string]string{
				"location": locationQuery,
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
				return fmt.Errorf("could not resolve location %q: %w", locationQuery, err)
			}
			fmt.Fprintf(os.Stderr, "Resolved to: %s (region_id=%s, region_type=%s)\n", regionName, regionID, regionType)

			// Step 2: Search properties
			params := map[string]string{
				"region_id":   regionID,
				"region_type": regionType,
				"al":          "1",
				"v":           "8",
				"num_homes":   "350",
			}
			if maxPrice > 0 {
				params["max_price"] = strconv.Itoa(maxPrice)
			}

			data, err := c.Get("/stingray/api/gis", params)
			if err != nil {
				return classifyAPIError(err)
			}

			homes := extractHomesFromGIS(data)
			if len(homes) == 0 {
				fmt.Fprintf(os.Stderr, "No properties found in %s.\n", regionName)
				return nil
			}
			fmt.Fprintf(os.Stderr, "Found %d properties, checking AVM estimates...\n", len(homes))

			// Step 3: For each property, fetch AVM estimate and calculate discount
			type deal struct {
				Address  string  `json:"address"`
				Price    float64 `json:"price"`
				Estimate float64 `json:"estimate"`
				Discount float64 `json:"discount_pct"`
				DOM      string  `json:"days_on_market"`
				Status   string  `json:"status"`
				ID       string  `json:"property_id"`
			}

			var deals []deal
			for _, home := range homes {
				price := 0.0
				for _, k := range []string{"price", "listPrice", "listingPrice"} {
					if v, ok := home[k]; ok {
						if f, fok := v.(float64); fok && f > 0 {
							price = f
							break
						}
					}
				}
				if price == 0 {
					continue
				}

				// Try to get estimate from the home data first
				estimate := 0.0
				for _, k := range []string{"estimate", "avmValue", "redfinEstimate", "predictedValue", "avm"} {
					if v, ok := home[k]; ok {
						if f, fok := v.(float64); fok && f > 0 {
							estimate = f
							break
						}
					}
				}

				// If no estimate in home data, try fetching from API
				if estimate == 0 {
					pid := ""
					for _, k := range []string{"propertyId", "mlsId", "listingId"} {
						if v, ok := home[k]; ok {
							pid = fmt.Sprintf("%v", v)
							break
						}
					}
					if pid != "" {
						avmData, avmErr := c.Get("/stingray/api/home/details/avm", map[string]string{
							"propertyId": pid,
						})
						if avmErr == nil {
							var avmResp map[string]any
							if json.Unmarshal(avmData, &avmResp) == nil {
								estimate = findNestedFloat(avmResp, "predictedValue", "estimate", "avmValue", "redfinEstimate")
							}
						}
					}
				}

				if estimate <= 0 {
					continue
				}

				discount := (estimate - price) / estimate * 100
				if discount < minDiscount {
					continue
				}

				d := deal{
					Address:  extractStr(home, "streetAddress", "address"),
					Price:    price,
					Estimate: estimate,
					Discount: discount,
					DOM:      extractNumStr(home, "dom", "daysOnMarket", "timeOnRedfin"),
					Status:   extractStr(home, "listingStatus", "status", "marketStatus"),
					ID:       extractStr(home, "propertyId", "mlsId"),
				}
				deals = append(deals, d)
			}

			// Step 4: Sort by discount descending
			sort.Slice(deals, func(i, j int) bool {
				return deals[i].Discount > deals[j].Discount
			})

			// Step 5: Apply limit
			if limit > 0 && len(deals) > limit {
				deals = deals[:limit]
			}

			if len(deals) == 0 {
				fmt.Fprintf(os.Stderr, "No deals found with at least %.1f%% discount.\n", minDiscount)
				return nil
			}

			fmt.Fprintf(os.Stderr, "Found %d deals\n", len(deals))

			// Step 6: Optionally save
			if save {
				if dbPath == "" {
					home, _ := os.UserHomeDir()
					dbPath = filepath.Join(home, ".local", "share", "redfin-pp-cli", "redfin.db")
				}
				db, err := store.Open(dbPath)
				if err != nil {
					return fmt.Errorf("opening local database: %w", err)
				}
				defer db.Close()
				for _, d := range deals {
					raw, _ := json.Marshal(d)
					if d.ID != "" {
						_ = db.Upsert("deal", d.ID, json.RawMessage(raw))
					}
				}
				fmt.Fprintf(os.Stderr, "Saved %d deals to local database\n", len(deals))
			}

			// Display
			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				headers := []string{"ADDRESS", "PRICE", "ESTIMATE", "DISCOUNT", "DOM", "STATUS"}
				var rows [][]string
				for _, d := range deals {
					rows = append(rows, []string{
						truncate(d.Address, 35),
						fmt.Sprintf("$%s", formatCompact(int64(d.Price))),
						fmt.Sprintf("$%s", formatCompact(int64(d.Estimate))),
						fmt.Sprintf("%.1f%%", d.Discount),
						d.DOM,
						d.Status,
					})
				}
				return flags.printTable(cmd, headers, rows)
			}

			combined, _ := json.Marshal(deals)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(combined), flags)
		},
	}

	cmd.Flags().IntVar(&maxPrice, "max-price", 0, "Maximum listing price")
	cmd.Flags().Float64Var(&minDiscount, "min-discount", 5, "Minimum discount percentage (estimate vs price)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of results")
	cmd.Flags().BoolVar(&save, "save", false, "Persist deals to local SQLite")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/redfin-pp-cli/redfin.db)")

	return cmd
}
