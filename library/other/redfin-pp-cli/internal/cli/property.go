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

func newPropertyCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "property",
		Short: "Property details, comparables, and analysis",
		Long:  `View detailed property information, comparables, neighborhood stats, and more.`,
	}

	cmd.AddCommand(newPropertyInfoCmd(flags))
	cmd.AddCommand(newPropertyDetailsCmd(flags))
	cmd.AddCommand(newPropertyValueCmd(flags))
	cmd.AddCommand(newPropertyCompsCmd(flags))
	cmd.AddCommand(newPropertyNearbyCmd(flags))
	cmd.AddCommand(newPropertyNeighborhoodCmd(flags))
	cmd.AddCommand(newPropertyCostsCmd(flags))
	cmd.AddCommand(newPropertyHistoryCmd(flags))
	cmd.AddCommand(newPropertyCommuteCmd(flags))

	return cmd
}

func newPropertyInfoCmd(flags *rootFlags) *cobra.Command {
	var save bool
	var dbPath string

	cmd := &cobra.Command{
		Use:   "info <url-or-property-id>",
		Short: "Get property overview (price, beds, baths, photos)",
		Example: `  # By Redfin URL path
  redfin-pp-cli property info /CA/San-Francisco/123-Main-St-94102/home/12345

  # By property ID
  redfin-pp-cli property info 12345678`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			input := args[0]
			propertyID, listingID, err := resolvePropertyInput(c, input)
			if err != nil {
				return err
			}

			params := map[string]string{
				"propertyId":  propertyID,
				"accessLevel": "3",
			}
			if listingID != "" {
				params["listingId"] = listingID
			}

			data, err := c.Get("/stingray/api/home/details/aboveTheFold", params)
			if err != nil {
				return classifyAPIError(err)
			}

			if save {
				savePropertyToStore(dbPath, propertyID, data)
			}

			return printPropertyOutput(cmd, data, flags)
		},
	}

	cmd.Flags().BoolVar(&save, "save", false, "Persist to local SQLite")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func newPropertyDetailsCmd(flags *rootFlags) *cobra.Command {
	var save bool
	var dbPath string

	cmd := &cobra.Command{
		Use:   "details <property-id>",
		Short: "Get full property details (amenities, history, characteristics)",
		Example: `  redfin-pp-cli property details 12345678
  redfin-pp-cli property details 12345678 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			propertyID := args[0]
			params := map[string]string{
				"propertyId":  propertyID,
				"accessLevel": "3",
			}

			data, err := c.Get("/stingray/api/home/details/belowTheFold", params)
			if err != nil {
				return classifyAPIError(err)
			}

			if save {
				savePropertyToStore(dbPath, propertyID, data)
			}

			return printPropertyOutput(cmd, data, flags)
		},
	}

	cmd.Flags().BoolVar(&save, "save", false, "Persist to local SQLite")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func newPropertyValueCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "value <property-id>",
		Short: "Get Redfin valuation estimate and AVM history",
		Example: `  redfin-pp-cli property value 12345678
  redfin-pp-cli property value 12345678 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			propertyID := args[0]

			// Fetch both estimate and AVM history
			estimateData, err := c.Get("/stingray/api/home/details/owner-estimate", map[string]string{
				"propertyId": propertyID,
			})
			if err != nil {
				return classifyAPIError(err)
			}

			avmData, err := c.Get("/stingray/api/home/details/avmHistoricalData", map[string]string{
				"propertyId": propertyID,
			})
			if err != nil {
				// AVM history might not be available for all properties
				fmt.Fprintf(os.Stderr, "warning: AVM history unavailable: %v\n", err)
				return printPropertyOutput(cmd, estimateData, flags)
			}

			// Combine both results
			combined := map[string]json.RawMessage{
				"estimate":   estimateData,
				"avmHistory": avmData,
			}
			result, _ := json.Marshal(combined)
			return printPropertyOutput(cmd, json.RawMessage(result), flags)
		},
	}

	return cmd
}

func newPropertyCompsCmd(flags *rootFlags) *cobra.Command {
	var sold bool

	cmd := &cobra.Command{
		Use:   "comps <property-id>",
		Short: "Get comparable properties (active or sold)",
		Example: `  # Active comparables
  redfin-pp-cli property comps 12345678

  # Recently sold comparables
  redfin-pp-cli property comps 12345678 --sold`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			propertyID := args[0]
			params := map[string]string{
				"propertyId": propertyID,
			}

			path := "/stingray/api/home/details/similar-listings"
			if sold {
				path = "/stingray/api/home/details/similar-sold"
			}

			data, err := c.Get(path, params)
			if err != nil {
				return classifyAPIError(err)
			}

			return printPropertyOutput(cmd, data, flags)
		},
	}

	cmd.Flags().BoolVar(&sold, "sold", false, "Show recently sold comparables instead of active listings")
	return cmd
}

func newPropertyNearbyCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nearby <property-id>",
		Short: "Get nearby properties",
		Example: `  redfin-pp-cli property nearby 12345678`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			data, err := c.Get("/stingray/api/home/details/nearby-homes", map[string]string{
				"propertyId": args[0],
			})
			if err != nil {
				return classifyAPIError(err)
			}

			return printPropertyOutput(cmd, data, flags)
		},
	}

	return cmd
}

func newPropertyNeighborhoodCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "neighborhood <property-id>",
		Short: "Get neighborhood stats (walk score, transit, bike score)",
		Example: `  redfin-pp-cli property neighborhood 12345678`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			data, err := c.Get("/stingray/api/home/details/neighborhoodStats/statsInfo", map[string]string{
				"propertyId": args[0],
			})
			if err != nil {
				return classifyAPIError(err)
			}

			return printPropertyOutput(cmd, data, flags)
		},
	}

	return cmd
}

func newPropertyCostsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "costs <property-id>",
		Short: "Get cost of ownership breakdown",
		Example: `  redfin-pp-cli property costs 12345678`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			data, err := c.Get("/stingray/api/home/details/cost-of-home-ownership", map[string]string{
				"propertyId": args[0],
			})
			if err != nil {
				return classifyAPIError(err)
			}

			return printPropertyOutput(cmd, data, flags)
		},
	}

	return cmd
}

func newPropertyHistoryCmd(flags *rootFlags) *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "history <property-id>",
		Short: "Get price history from local database",
		Long:  `Shows price history for a property from the local SQLite database. Run sync first to populate data.`,
		Example: `  redfin-pp-cli property history 12345678`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			propertyID := args[0]

			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".local", "share", "redfin-pp-cli", "redfin.db")
			}

			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer db.Close()

			// Try to find price history in the local store
			data, err := db.Get("price_history", propertyID)
			if err != nil {
				return fmt.Errorf("querying price history: %w", err)
			}

			if data == nil {
				// Fallback: try fetching from AVM history API
				c, cErr := flags.newClient()
				if cErr != nil {
					return fmt.Errorf("no local price history found for %s and cannot connect to API: %w", propertyID, cErr)
				}

				fmt.Fprintf(os.Stderr, "No local history found, fetching from API...\n")
				apiData, apiErr := c.Get("/stingray/api/home/details/avmHistoricalData", map[string]string{
					"propertyId": propertyID,
				})
				if apiErr != nil {
					return classifyAPIError(apiErr)
				}

				return printPropertyOutput(cmd, apiData, flags)
			}

			return printPropertyOutput(cmd, data, flags)
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")
	return cmd
}

func newPropertyCommuteCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commute <property-id>",
		Short: "Get commute information",
		Example: `  redfin-pp-cli property commute 12345678`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			data, err := c.Get("/stingray/api/home/details/commute/commuteInfo", map[string]string{
				"propertyId": args[0],
			})
			if err != nil {
				return classifyAPIError(err)
			}

			return printPropertyOutput(cmd, data, flags)
		},
	}

	return cmd
}

// resolvePropertyInput accepts either a Redfin URL path or a property ID.
// If given a URL path, calls initialInfo to resolve property and listing IDs.
func resolvePropertyInput(c interface {
	Get(path string, params map[string]string) (json.RawMessage, error)
}, input string) (propertyID, listingID string, err error) {
	// If input looks like a numeric ID, use it directly
	if _, numErr := strconv.Atoi(input); numErr == nil {
		return input, "", nil
	}

	// If input looks like a URL path, call initialInfo
	if strings.HasPrefix(input, "/") || strings.Contains(input, "/home/") {
		fmt.Fprintf(os.Stderr, "Resolving property URL: %s\n", input)
		data, apiErr := c.Get("/stingray/api/home/details/initialInfo", map[string]string{
			"path": input,
		})
		if apiErr != nil {
			return "", "", classifyAPIError(apiErr)
		}

		// Extract propertyId and listingId from response
		var resp map[string]any
		if json.Unmarshal(data, &resp) == nil {
			pid, lid := extractPropertyIDs(resp)
			if pid != "" {
				fmt.Fprintf(os.Stderr, "Resolved property ID: %s\n", pid)
				return pid, lid, nil
			}
		}

		return "", "", fmt.Errorf("could not resolve property from URL %q", input)
	}

	// Assume it's a property ID string
	return input, "", nil
}

// extractPropertyIDs recursively searches for propertyId and listingId in a response.
func extractPropertyIDs(obj map[string]any) (propertyID, listingID string) {
	// Direct fields
	for _, key := range []string{"propertyId", "property_id", "rpid"} {
		if v, ok := obj[key]; ok {
			propertyID = fmt.Sprintf("%v", v)
			break
		}
	}
	for _, key := range []string{"listingId", "listing_id"} {
		if v, ok := obj[key]; ok {
			listingID = fmt.Sprintf("%v", v)
			break
		}
	}

	if propertyID != "" {
		return propertyID, listingID
	}

	// Search nested maps
	for _, v := range obj {
		if nested, ok := v.(map[string]any); ok {
			pid, lid := extractPropertyIDs(nested)
			if pid != "" {
				return pid, lid
			}
		}
	}

	return "", ""
}

// printPropertyOutput handles output for property commands with human-friendly formatting.
func printPropertyOutput(cmd *cobra.Command, data json.RawMessage, flags *rootFlags) error {
	if wantsHumanTable(cmd.OutOrStdout(), flags) {
		var items []map[string]any
		if json.Unmarshal(data, &items) == nil && len(items) > 0 {
			if err := printAutoTable(cmd.OutOrStdout(), items); err != nil {
				return err
			}
			return nil
		}
	}
	return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
}

// savePropertyToStore persists property data to the local SQLite store.
func savePropertyToStore(dbPath, propertyID string, data json.RawMessage) {
	if dbPath == "" {
		home, _ := os.UserHomeDir()
		dbPath = filepath.Join(home, ".local", "share", "redfin-pp-cli", "redfin.db")
	}
	db, err := store.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not open database: %v\n", err)
		return
	}
	defer db.Close()

	if err := db.Upsert("property", propertyID, data); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to save property %s: %v\n", propertyID, err)
	} else {
		fmt.Fprintf(os.Stderr, "Saved property %s to local database\n", propertyID)
	}
}
