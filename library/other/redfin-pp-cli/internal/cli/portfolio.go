package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mvanhorn/printing-press-library/library/other/redfin-pp-cli/internal/store"

	"github.com/spf13/cobra"
)

func newPortfolioCmd(flags *rootFlags) *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "portfolio",
		Short: "Track and monitor properties in your portfolio",
		Long: `Manage a portfolio of tracked properties. Add properties to watch,
set price alerts, and refresh data for all tracked properties.`,
		Example: `  # List all tracked properties
  redfin-pp-cli portfolio

  # Add a property to track
  redfin-pp-cli portfolio add 12345678 --label watching --notes "nice yard"

  # Set price alert
  redfin-pp-cli portfolio add 12345678 --alert-below 500000

  # Refresh all portfolio data
  redfin-pp-cli portfolio refresh

  # Check price alerts
  redfin-pp-cli portfolio alerts`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openPortfolioDB(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			items, err := db.List("portfolio", 200)
			if err != nil {
				return fmt.Errorf("listing portfolio: %w", err)
			}

			if len(items) == 0 {
				fmt.Fprintf(os.Stderr, "No properties in portfolio. Use 'portfolio add <property-id>' to track a property.\n")
				return nil
			}

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				headers := []string{"PROPERTY ID", "LABEL", "PRICE", "ALERT BELOW", "NOTES", "ADDED"}
				var rows [][]string
				for _, item := range items {
					var entry portfolioEntry
					if json.Unmarshal(item, &entry) != nil {
						continue
					}
					priceStr := ""
					if entry.Price > 0 {
						priceStr = "$" + formatCompact(int64(entry.Price))
					}
					alertStr := ""
					if entry.AlertBelow > 0 {
						alertStr = "$" + formatCompact(int64(entry.AlertBelow))
					}
					addedStr := ""
					if entry.AddedAt != "" {
						if t, err := time.Parse(time.RFC3339, entry.AddedAt); err == nil {
							addedStr = t.Format("2006-01-02")
						} else {
							addedStr = entry.AddedAt
						}
					}
					rows = append(rows, []string{
						entry.PropertyID,
						entry.Label,
						priceStr,
						alertStr,
						truncate(entry.Notes, 30),
						addedStr,
					})
				}
				return flags.printTable(cmd, headers, rows)
			}

			combined, _ := json.Marshal(items)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(combined), flags)
		},
	}

	cmd.PersistentFlags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/redfin-pp-cli/redfin.db)")

	cmd.AddCommand(newPortfolioAddCmd(flags, &dbPath))
	cmd.AddCommand(newPortfolioRemoveCmd(flags, &dbPath))
	cmd.AddCommand(newPortfolioRefreshCmd(flags, &dbPath))
	cmd.AddCommand(newPortfolioAlertsCmd(flags, &dbPath))

	return cmd
}

type portfolioEntry struct {
	PropertyID string  `json:"id"`
	Label      string  `json:"label"`
	Price      float64 `json:"price"`
	Address    string  `json:"address"`
	AlertBelow float64 `json:"alert_below"`
	Notes      string  `json:"notes"`
	AddedAt    string  `json:"added_at"`
	UpdatedAt  string  `json:"updated_at"`
}

func newPortfolioAddCmd(flags *rootFlags, dbPath *string) *cobra.Command {
	var label string
	var alertBelow int
	var notes string

	cmd := &cobra.Command{
		Use:   "add <property-id>",
		Short: "Add a property to your portfolio",
		Example: `  redfin-pp-cli portfolio add 12345678
  redfin-pp-cli portfolio add 12345678 --label watching --notes "great school district"
  redfin-pp-cli portfolio add 12345678 --alert-below 500000`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			propertyID := args[0]

			db, err := openPortfolioDB(*dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			if label == "" {
				label = "watching"
			}

			entry := portfolioEntry{
				PropertyID: propertyID,
				Label:      label,
				AlertBelow: float64(alertBelow),
				Notes:      notes,
				AddedAt:    time.Now().Format(time.RFC3339),
				UpdatedAt:  time.Now().Format(time.RFC3339),
			}

			// Try to fetch current property data to populate price/address
			c, cErr := flags.newClient()
			if cErr == nil {
				data, apiErr := c.Get("/stingray/api/home/details/aboveTheFold", map[string]string{
					"propertyId":  propertyID,
					"accessLevel": "3",
				})
				if apiErr == nil {
					var resp map[string]any
					if json.Unmarshal(data, &resp) == nil {
						if p := findNestedFloat(resp, "price", "listPrice"); p > 0 {
							entry.Price = p
						}
						if a := findNestedStr(resp, "streetAddress", "address"); a != "" {
							entry.Address = a
						}
					}
				}
			}

			raw, _ := json.Marshal(entry)
			if err := db.Upsert("portfolio", propertyID, json.RawMessage(raw)); err != nil {
				return fmt.Errorf("adding to portfolio: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Added property %s to portfolio (label: %s)\n", propertyID, label)
			return nil
		},
	}

	cmd.Flags().StringVar(&label, "label", "watching", "Label for the property (watching/owned)")
	cmd.Flags().IntVar(&alertBelow, "alert-below", 0, "Alert when price drops below this value")
	cmd.Flags().StringVar(&notes, "notes", "", "Notes about this property")

	return cmd
}

func newPortfolioRemoveCmd(flags *rootFlags, dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <property-id>",
		Short: "Remove a property from your portfolio",
		Example: `  redfin-pp-cli portfolio remove 12345678`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			propertyID := args[0]

			db, err := openPortfolioDB(*dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			// Use a direct query to delete since store doesn't have a Delete method
			rows, qErr := db.Query("DELETE FROM resources WHERE resource_type = 'portfolio' AND id = ?", propertyID)
			if qErr != nil {
				return fmt.Errorf("removing from portfolio: %w", qErr)
			}
			rows.Close()

			fmt.Fprintf(os.Stderr, "Removed property %s from portfolio\n", propertyID)
			return nil
		},
	}

	return cmd
}

func newPortfolioRefreshCmd(flags *rootFlags, dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh data for all portfolio properties",
		Example: `  redfin-pp-cli portfolio refresh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openPortfolioDB(*dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			c, cErr := flags.newClient()
			if cErr != nil {
				return cErr
			}
			c.NoCache = true

			items, err := db.List("portfolio", 200)
			if err != nil {
				return fmt.Errorf("listing portfolio: %w", err)
			}

			if len(items) == 0 {
				fmt.Fprintf(os.Stderr, "No properties in portfolio to refresh.\n")
				return nil
			}

			updated := 0
			for _, item := range items {
				var entry portfolioEntry
				if json.Unmarshal(item, &entry) != nil {
					continue
				}

				fmt.Fprintf(os.Stderr, "Refreshing %s...\n", entry.PropertyID)

				data, apiErr := c.Get("/stingray/api/home/details/aboveTheFold", map[string]string{
					"propertyId":  entry.PropertyID,
					"accessLevel": "3",
				})
				if apiErr != nil {
					fmt.Fprintf(os.Stderr, "  warning: failed to refresh %s: %v\n", entry.PropertyID, apiErr)
					continue
				}

				// Update price and address from fresh data
				var resp map[string]any
				if json.Unmarshal(data, &resp) == nil {
					if p := findNestedFloat(resp, "price", "listPrice"); p > 0 {
						entry.Price = p
					}
					if a := findNestedStr(resp, "streetAddress", "address"); a != "" {
						entry.Address = a
					}
				}
				entry.UpdatedAt = time.Now().Format(time.RFC3339)

				raw, _ := json.Marshal(entry)
				if uErr := db.Upsert("portfolio", entry.PropertyID, json.RawMessage(raw)); uErr != nil {
					fmt.Fprintf(os.Stderr, "  warning: failed to save %s: %v\n", entry.PropertyID, uErr)
					continue
				}

				// Also save the full property data
				if uErr := db.Upsert("property", entry.PropertyID, data); uErr != nil {
					fmt.Fprintf(os.Stderr, "  warning: failed to save property data for %s: %v\n", entry.PropertyID, uErr)
				}

				updated++
			}

			fmt.Fprintf(os.Stderr, "Refreshed %d/%d portfolio properties\n", updated, len(items))
			return nil
		},
	}

	return cmd
}

func newPortfolioAlertsCmd(flags *rootFlags, dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "Check price thresholds for portfolio properties",
		Example: `  redfin-pp-cli portfolio alerts`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openPortfolioDB(*dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			items, err := db.List("portfolio", 200)
			if err != nil {
				return fmt.Errorf("listing portfolio: %w", err)
			}

			type alertResult struct {
				PropertyID string  `json:"property_id"`
				Address    string  `json:"address"`
				Price      float64 `json:"price"`
				AlertBelow float64 `json:"alert_below"`
				Triggered  bool    `json:"triggered"`
			}

			var alerts []alertResult
			for _, item := range items {
				var entry portfolioEntry
				if json.Unmarshal(item, &entry) != nil {
					continue
				}
				if entry.AlertBelow <= 0 {
					continue
				}

				triggered := entry.Price > 0 && entry.Price <= entry.AlertBelow
				alerts = append(alerts, alertResult{
					PropertyID: entry.PropertyID,
					Address:    entry.Address,
					Price:      entry.Price,
					AlertBelow: entry.AlertBelow,
					Triggered:  triggered,
				})
			}

			if len(alerts) == 0 {
				fmt.Fprintf(os.Stderr, "No price alerts configured. Use 'portfolio add <id> --alert-below <price>' to set one.\n")
				return nil
			}

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				headers := []string{"PROPERTY ID", "ADDRESS", "CURRENT PRICE", "ALERT BELOW", "STATUS"}
				var rows [][]string
				for _, a := range alerts {
					status := "OK"
					if a.Triggered {
						status = "ALERT: BELOW THRESHOLD"
					} else if a.Price == 0 {
						status = "NO PRICE DATA"
					}
					rows = append(rows, []string{
						a.PropertyID,
						truncate(a.Address, 30),
						"$" + formatCompact(int64(a.Price)),
						"$" + formatCompact(int64(a.AlertBelow)),
						status,
					})
				}
				return flags.printTable(cmd, headers, rows)
			}

			raw, _ := json.Marshal(alerts)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(raw), flags)
		},
	}

	return cmd
}

func openPortfolioDB(dbPath string) (*store.Store, error) {
	if dbPath == "" {
		home, _ := os.UserHomeDir()
		dbPath = filepath.Join(home, ".local", "share", "redfin-pp-cli", "redfin.db")
	}
	db, err := store.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening local database: %w", err)
	}
	return db, nil
}

// findNestedFloat searches a nested map for a float64 value by trying multiple keys.
func findNestedFloat(obj map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := obj[k]; ok {
			switch n := v.(type) {
			case float64:
				return n
			case string:
				if f, err := strconv.ParseFloat(n, 64); err == nil {
					return f
				}
			}
		}
	}
	// Recurse into nested maps
	for _, v := range obj {
		if nested, ok := v.(map[string]any); ok {
			if f := findNestedFloat(nested, keys...); f > 0 {
				return f
			}
		}
	}
	return 0
}

// findNestedStr searches a nested map for a string value by trying multiple keys.
func findNestedStr(obj map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := obj[k]; ok {
			if s, sok := v.(string); sok && s != "" {
				return s
			}
		}
	}
	// Recurse into nested maps
	for _, v := range obj {
		if nested, ok := v.(map[string]any); ok {
			if s := findNestedStr(nested, keys...); s != "" {
				return s
			}
		}
	}
	return ""
}
