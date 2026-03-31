package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/mvanhorn/printing-press-library/library/other/redfin-pp-cli/internal/store"

	"github.com/spf13/cobra"
)

type scoringProfile struct {
	Name              string `json:"name"`
	ValueWeight       int    `json:"value_weight"`
	NeighborhoodWeight int   `json:"neighborhood_weight"`
	DOMWeight         int    `json:"dom_weight"`
	PriceTrendWeight  int    `json:"price_trend_weight"`
}

type scoreResult struct {
	PropertyID       string  `json:"property_id"`
	Address          string  `json:"address"`
	OverallScore     float64 `json:"overall_score"`
	ValueScore       float64 `json:"value_score"`
	NeighborhoodScore float64 `json:"neighborhood_score"`
	DOMScore         float64 `json:"dom_score"`
	PriceTrendScore  float64 `json:"price_trend_score"`
	Profile          string  `json:"profile"`
}

func newScoreCmd(flags *rootFlags) *cobra.Command {
	var profileName string
	var dbPath string

	cmd := &cobra.Command{
		Use:   "score <property-id>",
		Short: "Score a property against a weighted profile",
		Long: `Calculate a smart score (0-100) for a property based on weighted dimensions:
value (price vs estimate), neighborhood (walk/bike/transit scores),
days on market (negotiation potential), and price trends.`,
		Example: `  # Score a property with default profile
  redfin-pp-cli score 12345

  # Score with a custom profile
  redfin-pp-cli score 12345 --profile myprofile

  # Create a scoring profile
  redfin-pp-cli score create-profile myprofile --value-weight 30 --neighborhood-weight 25 --dom-weight 25 --price-trend-weight 20

  # List profiles
  redfin-pp-cli score list-profiles`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
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

			// Load profile
			profile := scoringProfile{
				Name:               "default",
				ValueWeight:        30,
				NeighborhoodWeight: 30,
				DOMWeight:          20,
				PriceTrendWeight:   20,
			}
			if profileName != "" && profileName != "default" {
				raw, err := db.GetScoringProfile(profileName)
				if err != nil {
					return fmt.Errorf("loading profile: %w", err)
				}
				if raw == nil {
					return fmt.Errorf("profile %q not found. Use 'score create-profile' to create one", profileName)
				}
				if err := json.Unmarshal(raw, &profile); err != nil {
					return fmt.Errorf("parsing profile: %w", err)
				}
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Fetch property data
			propData, err := c.Get("/stingray/api/home/details/aboveTheFold", map[string]string{
				"propertyId":  propertyID,
				"accessLevel": "3",
			})
			if err != nil {
				return classifyAPIError(err)
			}

			if flags.dryRun {
				return nil
			}

			var propResp map[string]any
			if err := json.Unmarshal(propData, &propResp); err != nil {
				return fmt.Errorf("parsing property data: %w", err)
			}

			price := findNestedFloat(propResp, "price", "listPrice")
			address := findNestedStr(propResp, "streetAddress", "address")

			// Value score: price vs AVM estimate
			valueScore := 50.0 // neutral default
			avmData, avmErr := c.Get("/stingray/api/home/details/avm", map[string]string{
				"propertyId": propertyID,
			})
			if avmErr == nil {
				var avmResp map[string]any
				if json.Unmarshal(avmData, &avmResp) == nil {
					estimate := findNestedFloat(avmResp, "predictedValue", "estimate", "avmValue")
					if estimate > 0 && price > 0 {
						discount := (estimate - price) / estimate * 100
						// Map discount to 0-100: -20% overpriced=0, 0%=50, +20% underpriced=100
						valueScore = math.Min(100, math.Max(0, 50+(discount*2.5)))
					}
				}
			}

			// Neighborhood score: walk/bike/transit scores
			neighborhoodScore := 50.0
			nsData, nsErr := c.Get("/stingray/api/home/details/neighborhoodStats", map[string]string{
				"propertyId": propertyID,
			})
			if nsErr == nil {
				var nsResp map[string]any
				if json.Unmarshal(nsData, &nsResp) == nil {
					walkScore := findNestedFloat(nsResp, "walkScore", "walk_score")
					bikeScore := findNestedFloat(nsResp, "bikeScore", "bike_score")
					transitScore := findNestedFloat(nsResp, "transitScore", "transit_score")
					count := 0.0
					total := 0.0
					if walkScore > 0 {
						total += walkScore
						count++
					}
					if bikeScore > 0 {
						total += bikeScore
						count++
					}
					if transitScore > 0 {
						total += transitScore
						count++
					}
					if count > 0 {
						neighborhoodScore = total / count
					}
				}
			}

			// DOM score: longer DOM = more negotiation potential
			domScore := 50.0
			dom := findNestedFloat(propResp, "dom", "daysOnMarket", "timeOnRedfin")
			if dom > 0 {
				// 0 days=20, 30 days=50, 90+ days=100
				domScore = math.Min(100, math.Max(0, 20+(dom*0.89)))
			}

			// Price trend score: declining market = buyer advantage
			priceTrendScore := 50.0
			trendData, trendErr := c.Get("/stingray/api/home/details/propertyStats", map[string]string{
				"propertyId": propertyID,
			})
			if trendErr == nil {
				var trendResp map[string]any
				if json.Unmarshal(trendData, &trendResp) == nil {
					medianChange := findNestedFloat(trendResp, "medianPriceChange", "priceChange", "yoyChange")
					if medianChange != 0 {
						// Negative change = declining = higher score for buyers
						// -10% = 100, 0% = 50, +10% = 0
						priceTrendScore = math.Min(100, math.Max(0, 50-(medianChange*5)))
					}
				}
			}

			// Calculate overall score
			totalWeight := float64(profile.ValueWeight + profile.NeighborhoodWeight + profile.DOMWeight + profile.PriceTrendWeight)
			overall := 0.0
			if totalWeight > 0 {
				overall = (valueScore*float64(profile.ValueWeight) +
					neighborhoodScore*float64(profile.NeighborhoodWeight) +
					domScore*float64(profile.DOMWeight) +
					priceTrendScore*float64(profile.PriceTrendWeight)) / totalWeight
			}

			result := scoreResult{
				PropertyID:        propertyID,
				Address:           address,
				OverallScore:      math.Round(overall*10) / 10,
				ValueScore:        math.Round(valueScore*10) / 10,
				NeighborhoodScore: math.Round(neighborhoodScore*10) / 10,
				DOMScore:          math.Round(domScore*10) / 10,
				PriceTrendScore:   math.Round(priceTrendScore*10) / 10,
				Profile:           profile.Name,
			}

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				headers := []string{"DIMENSION", "SCORE", "WEIGHT"}
				rows := [][]string{
					{"Value (price vs estimate)", fmt.Sprintf("%.1f", result.ValueScore), fmt.Sprintf("%d%%", profile.ValueWeight)},
					{"Neighborhood", fmt.Sprintf("%.1f", result.NeighborhoodScore), fmt.Sprintf("%d%%", profile.NeighborhoodWeight)},
					{"Days on Market", fmt.Sprintf("%.1f", result.DOMScore), fmt.Sprintf("%d%%", profile.DOMWeight)},
					{"Price Trend", fmt.Sprintf("%.1f", result.PriceTrendScore), fmt.Sprintf("%d%%", profile.PriceTrendWeight)},
					{"", "", ""},
					{"OVERALL SCORE", fmt.Sprintf("%.1f / 100", result.OverallScore), ""},
				}
				return flags.printTable(cmd, headers, rows)
			}

			raw, _ := json.Marshal(result)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(raw), flags)
		},
	}

	cmd.Flags().StringVar(&profileName, "profile", "default", "Scoring profile to use")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/redfin-pp-cli/redfin.db)")

	cmd.AddCommand(newScoreCreateProfileCmd(flags, &dbPath))
	cmd.AddCommand(newScoreListProfilesCmd(flags, &dbPath))

	return cmd
}

func newScoreCreateProfileCmd(flags *rootFlags, dbPath *string) *cobra.Command {
	var valueWeight, neighborhoodWeight, domWeight, priceTrendWeight int

	cmd := &cobra.Command{
		Use:   "create-profile <name>",
		Short: "Create a scoring profile with custom weights",
		Example: `  redfin-pp-cli score create-profile myprofile --value-weight 30 --neighborhood-weight 25 --dom-weight 25 --price-trend-weight 20`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if *dbPath == "" {
				home, _ := os.UserHomeDir()
				*dbPath = filepath.Join(home, ".local", "share", "redfin-pp-cli", "redfin.db")
			}

			db, err := store.Open(*dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer db.Close()

			profile := scoringProfile{
				Name:               name,
				ValueWeight:        valueWeight,
				NeighborhoodWeight: neighborhoodWeight,
				DOMWeight:          domWeight,
				PriceTrendWeight:   priceTrendWeight,
			}

			raw, _ := json.Marshal(profile)
			if err := db.UpsertScoringProfile(name, json.RawMessage(raw)); err != nil {
				return fmt.Errorf("saving profile: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Created scoring profile %q (value=%d, neighborhood=%d, dom=%d, price-trend=%d)\n",
				name, valueWeight, neighborhoodWeight, domWeight, priceTrendWeight)
			return nil
		},
	}

	cmd.Flags().IntVar(&valueWeight, "value-weight", 30, "Weight for value score (price vs estimate)")
	cmd.Flags().IntVar(&neighborhoodWeight, "neighborhood-weight", 30, "Weight for neighborhood score")
	cmd.Flags().IntVar(&domWeight, "dom-weight", 20, "Weight for days on market score")
	cmd.Flags().IntVar(&priceTrendWeight, "price-trend-weight", 20, "Weight for price trend score")

	return cmd
}

func newScoreListProfilesCmd(flags *rootFlags, dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-profiles",
		Short: "List all scoring profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			if *dbPath == "" {
				home, _ := os.UserHomeDir()
				*dbPath = filepath.Join(home, ".local", "share", "redfin-pp-cli", "redfin.db")
			}

			db, err := store.Open(*dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer db.Close()

			rows, err := db.Query("SELECT name, config FROM scoring_profiles ORDER BY name")
			if err != nil {
				return fmt.Errorf("listing profiles: %w", err)
			}
			defer rows.Close()

			type profileRow struct {
				Name   string          `json:"name"`
				Config json.RawMessage `json:"config"`
			}
			var profiles []profileRow
			for rows.Next() {
				var name, config string
				if err := rows.Scan(&name, &config); err != nil {
					return err
				}
				profiles = append(profiles, profileRow{Name: name, Config: json.RawMessage(config)})
			}

			if len(profiles) == 0 {
				fmt.Fprintf(os.Stderr, "No scoring profiles. Use 'score create-profile' to create one.\n")
				fmt.Fprintf(os.Stderr, "The 'default' profile is always available with weights: value=30, neighborhood=30, dom=20, price-trend=20\n")
				return nil
			}

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				headers := []string{"NAME", "VALUE", "NEIGHBORHOOD", "DOM", "PRICE TREND"}
				var tableRows [][]string
				for _, p := range profiles {
					var sp scoringProfile
					if json.Unmarshal(p.Config, &sp) != nil {
						continue
					}
					tableRows = append(tableRows, []string{
						p.Name,
						fmt.Sprintf("%d", sp.ValueWeight),
						fmt.Sprintf("%d", sp.NeighborhoodWeight),
						fmt.Sprintf("%d", sp.DOMWeight),
						fmt.Sprintf("%d", sp.PriceTrendWeight),
					})
				}
				return flags.printTable(cmd, headers, tableRows)
			}

			raw, _ := json.Marshal(profiles)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(raw), flags)
		},
	}

	return cmd
}
