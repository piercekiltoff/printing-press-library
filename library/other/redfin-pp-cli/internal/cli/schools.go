package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newSchoolsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schools <property-id>",
		Short: "Display school district information for a property",
		Long: `Fetch and display school information from neighborhood stats for a property.
Shows nearby schools, ratings, and district details.`,
		Example: `  # View schools near a property
  redfin-pp-cli schools 12345

  # Compare schools between two properties
  redfin-pp-cli schools compare 12345 67890`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			propertyID := args[0]

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "Fetching school data for property %s...\n", propertyID)

			// Fetch neighborhood stats which include school data
			data, err := c.Get("/stingray/api/home/details/neighborhoodStats", map[string]string{
				"propertyId": propertyID,
			})
			if err != nil {
				return classifyAPIError(err)
			}

			if flags.dryRun {
				return nil
			}

			var resp map[string]any
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			schools := extractSchoolData(resp)
			if len(schools) == 0 {
				fmt.Fprintf(os.Stderr, "No school data found for property %s.\n", propertyID)
				// Still output the raw neighborhood data
				return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
			}

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				headers := []string{"SCHOOL", "TYPE", "RATING", "GRADES", "DISTANCE"}
				var rows [][]string
				for _, s := range schools {
					rows = append(rows, []string{
						truncate(s.Name, 35),
						s.Type,
						s.Rating,
						s.Grades,
						s.Distance,
					})
				}

				// Also show summary scores
				walkScore := findNestedFloat(resp, "walkScore", "walk_score")
				bikeScore := findNestedFloat(resp, "bikeScore", "bike_score")
				transitScore := findNestedFloat(resp, "transitScore", "transit_score")

				if err := flags.printTable(cmd, headers, rows); err != nil {
					return err
				}

				if walkScore > 0 || bikeScore > 0 || transitScore > 0 {
					fmt.Fprintf(os.Stderr, "\nNeighborhood Scores: Walk=%.0f  Bike=%.0f  Transit=%.0f\n",
						walkScore, bikeScore, transitScore)
				}
				return nil
			}

			type schoolOutput struct {
				Schools      []schoolInfo `json:"schools"`
				WalkScore    float64      `json:"walk_score,omitempty"`
				BikeScore    float64      `json:"bike_score,omitempty"`
				TransitScore float64      `json:"transit_score,omitempty"`
			}
			out := schoolOutput{
				Schools:      schools,
				WalkScore:    findNestedFloat(resp, "walkScore", "walk_score"),
				BikeScore:    findNestedFloat(resp, "bikeScore", "bike_score"),
				TransitScore: findNestedFloat(resp, "transitScore", "transit_score"),
			}
			raw, _ := json.Marshal(out)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(raw), flags)
		},
	}

	cmd.AddCommand(newSchoolsCompareCmd(flags))

	return cmd
}

func newSchoolsCompareCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare <property-id-1> <property-id-2>",
		Short: "Compare school data between two properties",
		Example: `  redfin-pp-cli schools compare 12345 67890`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			propA := args[0]
			propB := args[1]

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "Fetching school data for properties %s and %s...\n", propA, propB)

			dataA, errA := c.Get("/stingray/api/home/details/neighborhoodStats", map[string]string{
				"propertyId": propA,
			})
			if errA != nil {
				return classifyAPIError(errA)
			}

			dataB, errB := c.Get("/stingray/api/home/details/neighborhoodStats", map[string]string{
				"propertyId": propB,
			})
			if errB != nil {
				return classifyAPIError(errB)
			}

			if flags.dryRun {
				return nil
			}

			var respA, respB map[string]any
			json.Unmarshal(dataA, &respA)
			json.Unmarshal(dataB, &respB)

			schoolsA := extractSchoolData(respA)
			schoolsB := extractSchoolData(respB)

			type compareOutput struct {
				PropertyA struct {
					ID           string       `json:"property_id"`
					Schools      []schoolInfo `json:"schools"`
					WalkScore    float64      `json:"walk_score"`
					BikeScore    float64      `json:"bike_score"`
					TransitScore float64      `json:"transit_score"`
				} `json:"property_a"`
				PropertyB struct {
					ID           string       `json:"property_id"`
					Schools      []schoolInfo `json:"schools"`
					WalkScore    float64      `json:"walk_score"`
					BikeScore    float64      `json:"bike_score"`
					TransitScore float64      `json:"transit_score"`
				} `json:"property_b"`
			}

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				// Summary comparison table
				headers := []string{"METRIC", propA, propB}

				walkA := findNestedFloat(respA, "walkScore", "walk_score")
				walkB := findNestedFloat(respB, "walkScore", "walk_score")
				bikeA := findNestedFloat(respA, "bikeScore", "bike_score")
				bikeB := findNestedFloat(respB, "bikeScore", "bike_score")
				transitA := findNestedFloat(respA, "transitScore", "transit_score")
				transitB := findNestedFloat(respB, "transitScore", "transit_score")

				rows := [][]string{
					{"Walk Score", fmt.Sprintf("%.0f", walkA), fmt.Sprintf("%.0f", walkB)},
					{"Bike Score", fmt.Sprintf("%.0f", bikeA), fmt.Sprintf("%.0f", bikeB)},
					{"Transit Score", fmt.Sprintf("%.0f", transitA), fmt.Sprintf("%.0f", transitB)},
					{"Schools Nearby", fmt.Sprintf("%d", len(schoolsA)), fmt.Sprintf("%d", len(schoolsB))},
				}

				if err := flags.printTable(cmd, headers, rows); err != nil {
					return err
				}

				// School details for property A
				if len(schoolsA) > 0 {
					fmt.Fprintf(os.Stderr, "\nSchools near %s:\n", propA)
					for _, s := range schoolsA {
						fmt.Fprintf(os.Stderr, "  %s (%s) - Rating: %s, Grades: %s\n", s.Name, s.Type, s.Rating, s.Grades)
					}
				}

				// School details for property B
				if len(schoolsB) > 0 {
					fmt.Fprintf(os.Stderr, "\nSchools near %s:\n", propB)
					for _, s := range schoolsB {
						fmt.Fprintf(os.Stderr, "  %s (%s) - Rating: %s, Grades: %s\n", s.Name, s.Type, s.Rating, s.Grades)
					}
				}

				return nil
			}

			out := compareOutput{}
			out.PropertyA.ID = propA
			out.PropertyA.Schools = schoolsA
			out.PropertyA.WalkScore = findNestedFloat(respA, "walkScore", "walk_score")
			out.PropertyA.BikeScore = findNestedFloat(respA, "bikeScore", "bike_score")
			out.PropertyA.TransitScore = findNestedFloat(respA, "transitScore", "transit_score")
			out.PropertyB.ID = propB
			out.PropertyB.Schools = schoolsB
			out.PropertyB.WalkScore = findNestedFloat(respB, "walkScore", "walk_score")
			out.PropertyB.BikeScore = findNestedFloat(respB, "bikeScore", "bike_score")
			out.PropertyB.TransitScore = findNestedFloat(respB, "transitScore", "transit_score")

			raw, _ := json.Marshal(out)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(raw), flags)
		},
	}

	return cmd
}

type schoolInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Rating   string `json:"rating"`
	Grades   string `json:"grades"`
	Distance string `json:"distance"`
}

// extractSchoolData pulls school information from neighborhood stats response.
func extractSchoolData(resp map[string]any) []schoolInfo {
	if resp == nil {
		return nil
	}

	var schools []schoolInfo

	// Try common structures for school data
	for _, key := range []string{"schools", "nearbySchools", "schoolData", "servingSchools"} {
		if v, ok := resp[key]; ok {
			if arr, aok := v.([]any); aok {
				for _, item := range arr {
					if m, mok := item.(map[string]any); mok {
						s := schoolInfo{
							Name:   findSchoolStr(m, "name", "schoolName"),
							Type:   findSchoolStr(m, "type", "schoolType", "level"),
							Rating: findSchoolStr(m, "rating", "greatSchoolsRating", "parentRating"),
							Grades: findSchoolStr(m, "grades", "gradeRange", "gradeLevels"),
						}
						dist := findSchoolFloat(m, "distance", "distanceMiles")
						if dist > 0 {
							s.Distance = fmt.Sprintf("%.1f mi", dist)
						}
						if s.Name != "" {
							schools = append(schools, s)
						}
					}
				}
				if len(schools) > 0 {
					return schools
				}
			}
		}
	}

	// Try nested payload
	if payload, ok := resp["payload"]; ok {
		if pm, pok := payload.(map[string]any); pok {
			return extractSchoolData(pm)
		}
	}

	return schools
}

func findSchoolStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

func findSchoolFloat(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if f, fok := v.(float64); fok {
				return f
			}
		}
	}
	return 0
}
