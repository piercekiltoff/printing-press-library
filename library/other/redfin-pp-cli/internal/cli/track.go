package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mvanhorn/printing-press-library/library/other/redfin-pp-cli/internal/store"

	"github.com/spf13/cobra"
)

func newTrackCmd(flags *rootFlags) *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "track <property-id>",
		Short: "Track property price trajectory over time",
		Long: `Show the price trajectory of a property over time using local valuation history.
Fetches the current AVM estimate, stores a snapshot, and displays the full trajectory.

Use "track list" to see all properties with valuation history.`,
		Example: `  # Track a property's valuation
  redfin-pp-cli track 12345

  # List all tracked properties
  redfin-pp-cli track list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			propertyID := args[0]

			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".local", "share", "redfin-pp-cli", "redfin.db")
			}

			// Handle "list" subcommand inline
			if propertyID == "list" {
				return trackList(cmd, flags, dbPath)
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Fetch current AVM
			fmt.Fprintf(os.Stderr, "Fetching valuation for property %s...\n", propertyID)
			pidInt, _ := strconv.Atoi(propertyID)
			avmData, err := c.Get("/stingray/api/home/details/avmHistoricalData", map[string]string{
				"propertyId": fmt.Sprintf("%d", pidInt),
			})
			if err != nil {
				return classifyAPIError(err)
			}

			if flags.dryRun {
				return nil
			}

			// Parse current estimate from AVM response
			estimate, low, high := extractCurrentAVM(avmData)

			// Store valuation snapshot
			db, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening local database: %w", err)
			}
			defer db.Close()

			if estimate > 0 {
				if err := db.AddValuation(propertyID, estimate, low, high); err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to store valuation: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "Stored valuation snapshot: $%s\n", formatCompact(int64(estimate)))
				}
			}

			// Fetch full history from local store
			history, err := db.GetValuationHistory(propertyID)
			if err != nil {
				return fmt.Errorf("fetching valuation history: %w", err)
			}

			if len(history) == 0 {
				fmt.Fprintf(os.Stderr, "No valuation history found for property %s\n", propertyID)
				return printPropertyOutput(cmd, avmData, flags)
			}

			if flags.asJSON {
				type valEntry struct {
					Date      string `json:"date"`
					Estimate  int    `json:"estimate"`
					Low       int    `json:"estimate_low"`
					High      int    `json:"estimate_high"`
					Change    string `json:"change"`
					Direction string `json:"direction"`
				}
				var entries []valEntry
				for i, v := range history {
					entry := valEntry{
						Date:     v.CapturedAt.Format("2006-01-02"),
						Estimate: v.Estimate,
						Low:      v.EstimateLow,
						High:     v.EstimateHigh,
					}
					if i < len(history)-1 {
						prev := history[i+1].Estimate
						if prev > 0 {
							pctChange := float64(v.Estimate-prev) / float64(prev) * 100
							entry.Change = fmt.Sprintf("%.1f%%", pctChange)
							if pctChange > 0 {
								entry.Direction = "up"
							} else if pctChange < 0 {
								entry.Direction = "down"
							} else {
								entry.Direction = "flat"
							}
						}
					}
					entries = append(entries, entry)
				}

				first := history[len(history)-1]
				last := history[0]
				totalChange := 0.0
				if first.Estimate > 0 {
					totalChange = float64(last.Estimate-first.Estimate) / float64(first.Estimate) * 100
				}

				return flags.printJSON(cmd, map[string]any{
					"property_id":  propertyID,
					"trajectory":   entries,
					"started_at":   "$" + formatCompact(int64(first.Estimate)),
					"current":      "$" + formatCompact(int64(last.Estimate)),
					"total_change": fmt.Sprintf("%.1f%%", totalChange),
				})
			}

			// Human-friendly table
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Price Trajectory: Property %s\n", propertyID)
			fmt.Fprintf(out, "============================\n\n")

			headers := []string{"DATE", "ESTIMATE", "CHANGE", "DIRECTION"}
			var rows [][]string
			for i, v := range history {
				change := "-"
				direction := "-"
				if i < len(history)-1 {
					prev := history[i+1].Estimate
					if prev > 0 {
						pctChange := float64(v.Estimate-prev) / float64(prev) * 100
						change = fmt.Sprintf("%.1f%%", pctChange)
						if pctChange > 0 {
							direction = "^"
						} else if pctChange < 0 {
							direction = "v"
						} else {
							direction = "="
						}
					}
				}
				rows = append(rows, []string{
					v.CapturedAt.Format("2006-01-02"),
					"$" + formatCompact(int64(v.Estimate)),
					change,
					direction,
				})
			}
			if err := flags.printTable(cmd, headers, rows); err != nil {
				return err
			}

			// Summary
			first := history[len(history)-1]
			last := history[0]
			totalChange := 0.0
			if first.Estimate > 0 {
				totalChange = float64(last.Estimate-first.Estimate) / float64(first.Estimate) * 100
			}
			fmt.Fprintf(out, "\nSummary: Started at $%s, currently $%s, total change %.1f%%\n",
				formatCompact(int64(first.Estimate)),
				formatCompact(int64(last.Estimate)),
				totalChange)

			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/redfin-pp-cli/redfin.db)")

	return cmd
}

func trackList(cmd *cobra.Command, flags *rootFlags, dbPath string) error {
	db, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening local database: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(
		`SELECT property_id, COUNT(*) as snapshots, MIN(captured_at) as first_seen, MAX(captured_at) as last_seen,
		 MIN(estimate) as low_est, MAX(estimate) as high_est
		 FROM valuations GROUP BY property_id ORDER BY last_seen DESC`,
	)
	if err != nil {
		return fmt.Errorf("querying valuations: %w", err)
	}
	defer rows.Close()

	type trackEntry struct {
		PropertyID string `json:"property_id"`
		Snapshots  int    `json:"snapshots"`
		FirstSeen  string `json:"first_seen"`
		LastSeen   string `json:"last_seen"`
		LowEst     int    `json:"low_estimate"`
		HighEst    int    `json:"high_estimate"`
	}

	var entries []trackEntry
	for rows.Next() {
		var e trackEntry
		if err := rows.Scan(&e.PropertyID, &e.Snapshots, &e.FirstSeen, &e.LastSeen, &e.LowEst, &e.HighEst); err != nil {
			return fmt.Errorf("scanning row: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "No tracked properties found. Use 'track <property-id>' to start tracking.")
		return nil
	}

	if flags.asJSON {
		return flags.printJSON(cmd, entries)
	}

	headers := []string{"PROPERTY", "SNAPSHOTS", "FIRST SEEN", "LAST SEEN", "LOW EST", "HIGH EST"}
	var tableRows [][]string
	for _, e := range entries {
		tableRows = append(tableRows, []string{
			e.PropertyID,
			strconv.Itoa(e.Snapshots),
			e.FirstSeen,
			e.LastSeen,
			"$" + formatCompact(int64(e.LowEst)),
			"$" + formatCompact(int64(e.HighEst)),
		})
	}

	return flags.printTable(cmd, headers, tableRows)
}

// extractCurrentAVM pulls the most recent AVM estimate from the response.
func extractCurrentAVM(data json.RawMessage) (estimate, low, high int) {
	// Try as object with estimate fields
	var obj map[string]any
	if json.Unmarshal(data, &obj) == nil {
		estimate = extractIntField(obj, "estimate", "avm", "value", "predictedValue")
		low = extractIntField(obj, "estimateLow", "avmLow", "lowValue")
		high = extractIntField(obj, "estimateHigh", "avmHigh", "highValue")
		if estimate > 0 {
			return
		}

		// Try payload wrapper
		if payload, ok := obj["payload"]; ok {
			if pm, ok := payload.(map[string]any); ok {
				estimate = extractIntField(pm, "estimate", "avm", "value", "predictedValue")
				low = extractIntField(pm, "estimateLow", "avmLow", "lowValue")
				high = extractIntField(pm, "estimateHigh", "avmHigh", "highValue")
				if estimate > 0 {
					return
				}
			}
		}
	}

	// Try as array (historical data) - take first entry
	var items []map[string]any
	if json.Unmarshal(data, &items) == nil && len(items) > 0 {
		first := items[0]
		estimate = extractIntField(first, "estimate", "avm", "value", "predictedValue", "price")
		low = extractIntField(first, "estimateLow", "avmLow", "lowValue")
		high = extractIntField(first, "estimateHigh", "avmHigh", "highValue")
	}

	return
}

func extractIntField(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch n := v.(type) {
			case float64:
				return int(n)
			case int:
				return n
			case string:
				if parsed, err := strconv.Atoi(n); err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}
