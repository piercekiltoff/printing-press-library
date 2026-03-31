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

func newStaleCmd(flags *rootFlags) *cobra.Command {
	var minDays int
	var region string
	var priceDropsOnly bool
	var limit int
	var save bool
	var dbPath string

	cmd := &cobra.Command{
		Use:   "stale",
		Short: "Find stale listings with high days on market",
		Long: `Search for stale listings that have been on the market for an extended period.
Optionally filter to only those with price drops, which may indicate motivated sellers.`,
		Example: `  # Find listings on market 30+ days
  redfin-pp-cli stale --days 30 --region "Austin, TX"

  # Find 60+ day listings with price drops only
  redfin-pp-cli stale --days 60 --region "Denver, CO" --price-drops-only

  # Save results
  redfin-pp-cli stale --days 45 --region "Seattle, WA" --save`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if region == "" {
				return cmd.Help()
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Resolve location
			fmt.Fprintf(os.Stderr, "Resolving location: %s\n", region)
			acData, err := c.Get("/stingray/do/location-autocomplete", map[string]string{
				"location": region,
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
				return fmt.Errorf("could not resolve location %q: %w", region, err)
			}
			fmt.Fprintf(os.Stderr, "Resolved to: %s\n", regionName)

			// Search with DOM sort descending
			searchLimit := limit
			if searchLimit <= 0 {
				searchLimit = 50
			}
			// Fetch more than limit to allow filtering
			fetchLimit := searchLimit * 3
			if fetchLimit > 350 {
				fetchLimit = 350
			}

			params := map[string]string{
				"region_id":   regionID,
				"region_type": regionType,
				"al":          "1",
				"v":           "8",
				"num_homes":   strconv.Itoa(fetchLimit),
				"ord":         "days-on-redfin-desc",
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

			// Filter by minimum days on market
			type staleEntry struct {
				Address       string  `json:"address"`
				Price         float64 `json:"price"`
				DOM           int     `json:"days_on_market"`
				PriceChanges  int     `json:"price_changes"`
				OriginalPrice float64 `json:"original_price"`
				Status        string  `json:"status"`
				ID            string  `json:"property_id"`
				HasPriceDrop  bool    `json:"has_price_drop"`
			}

			var staleListings []staleEntry
			for _, home := range homes {
				domStr := extractNumStr(home, "dom", "daysOnMarket", "timeOnRedfin")
				dom := 0
				if domStr != "" {
					if d, err := strconv.Atoi(domStr); err == nil {
						dom = d
					}
				}

				if dom < minDays {
					continue
				}

				price := 0.0
				for _, k := range []string{"price", "listPrice", "listingPrice"} {
					if v, ok := home[k]; ok {
						if f, fok := v.(float64); fok && f > 0 {
							price = f
							break
						}
					}
				}

				entry := staleEntry{
					Address: extractStr(home, "streetAddress", "address"),
					Price:   price,
					DOM:     dom,
					Status:  extractStr(home, "listingStatus", "status", "marketStatus"),
					ID:      extractStr(home, "propertyId", "mlsId"),
				}

				// Check for price history / original price
				origPrice := 0.0
				for _, k := range []string{"originalPrice", "originalListPrice"} {
					if v, ok := home[k]; ok {
						if f, fok := v.(float64); fok && f > 0 {
							origPrice = f
							break
						}
					}
				}
				if origPrice > 0 && origPrice != price {
					entry.OriginalPrice = origPrice
					entry.HasPriceDrop = price < origPrice
					entry.PriceChanges = 1
				}

				// Check price drop count
				for _, k := range []string{"priceDropCount", "priceChanges", "numPriceDrops"} {
					if v, ok := home[k]; ok {
						if f, fok := v.(float64); fok && f > 0 {
							entry.PriceChanges = int(f)
							entry.HasPriceDrop = true
							break
						}
					}
				}

				if priceDropsOnly && !entry.HasPriceDrop {
					continue
				}

				staleListings = append(staleListings, entry)
			}

			// Sort by DOM descending
			sort.Slice(staleListings, func(i, j int) bool {
				return staleListings[i].DOM > staleListings[j].DOM
			})

			// Apply limit
			if limit > 0 && len(staleListings) > limit {
				staleListings = staleListings[:limit]
			}

			if len(staleListings) == 0 {
				fmt.Fprintf(os.Stderr, "No stale listings found with %d+ days on market.\n", minDays)
				return nil
			}

			fmt.Fprintf(os.Stderr, "Found %d stale listings\n", len(staleListings))

			// Optionally save
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
				for _, s := range staleListings {
					raw, _ := json.Marshal(s)
					if s.ID != "" {
						_ = db.Upsert("stale", s.ID, json.RawMessage(raw))
					}
				}
				fmt.Fprintf(os.Stderr, "Saved %d stale listings to local database\n", len(staleListings))
			}

			// Display
			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				headers := []string{"ADDRESS", "PRICE", "DOM", "PRICE CHANGES", "ORIGINAL PRICE", "STATUS"}
				var rows [][]string
				for _, s := range staleListings {
					origStr := ""
					if s.OriginalPrice > 0 {
						origStr = fmt.Sprintf("$%s", formatCompact(int64(s.OriginalPrice)))
					}
					rows = append(rows, []string{
						truncate(s.Address, 35),
						fmt.Sprintf("$%s", formatCompact(int64(s.Price))),
						strconv.Itoa(s.DOM),
						strconv.Itoa(s.PriceChanges),
						origStr,
						s.Status,
					})
				}
				return flags.printTable(cmd, headers, rows)
			}

			combined, _ := json.Marshal(staleListings)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(combined), flags)
		},
	}

	cmd.Flags().IntVar(&minDays, "days", 30, "Minimum days on market")
	cmd.Flags().StringVar(&region, "region", "", "Location to search (required)")
	cmd.Flags().BoolVar(&priceDropsOnly, "price-drops-only", false, "Show only listings with price drops")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of results")
	cmd.Flags().BoolVar(&save, "save", false, "Persist results to local SQLite")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/redfin-pp-cli/redfin.db)")

	return cmd
}
