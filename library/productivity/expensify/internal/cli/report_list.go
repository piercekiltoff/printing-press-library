// Copyright 2026 matt-van-horn. Licensed under Apache-2.0.
// List reports from the local store (populated by `sync`). With --live (or
// --data-source=live), issue a /Search call first and upsert the returned
// rows into the local store as a side effect.

package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/config"
	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/store"
)

func newReportListCmd(flags *rootFlags) *cobra.Command {
	var policyID string
	var status string
	var limit int
	var live bool
	var owner string
	var allVisible bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your reports from the local cache (or --live to query Expensify)",
		Example: `  expensify-pp-cli report list
  expensify-pp-cli report list --status open
  expensify-pp-cli report list --live
  expensify-pp-cli report list --live --owner teammate@example.com
  expensify-pp-cli report list --live --all-visible
  expensify-pp-cli report list --policy-id POLICY_ID_HERE --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := store.Open(store.DefaultPath())
			if err != nil {
				return apiErr(fmt.Errorf("opening local store: %w", err))
			}
			defer db.Close()
			if err := db.Migrate(); err != nil {
				return apiErr(fmt.Errorf("migrating local store: %w", err))
			}

			if liveModeEnabled(live, flags) {
				cfg, err := config.Load(flags.configPath)
				if err != nil {
					return configErr(err)
				}
				if err := runLiveReportSearch(cmd, flags, db, cfg, owner, allVisible, policyID, status); err != nil {
					return err
				}
			}

			filters := map[string]string{}
			if policyID != "" {
				filters["policy_id"] = policyID
			}
			if status != "" {
				filters["status"] = status
			}
			items, err := db.ListReports(filters)
			if err != nil {
				return apiErr(fmt.Errorf("listing reports: %w", err))
			}
			if limit > 0 && len(items) > limit {
				items = items[:limit]
			}

			if len(items) == 0 {
				fmt.Fprintln(os.Stderr, "No reports in local cache. Run `expensify-pp-cli sync` to fetch from Expensify.")
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				data, _ := json.Marshal(items)
				return printOutput(cmd.OutOrStdout(), data, true)
			}

			w := newTabWriter(cmd.OutOrStdout())
			fmt.Fprintln(w, "REPORT_ID\tSTATUS\tTITLE\tTOTAL\tEXPENSES\tLAST_UPDATED\tOWNER")
			for _, r := range items {
				title := r.Title
				if title == "" {
					title = "(untitled)"
				}
				ownerCol := formatOwnerFromRaw(db, r.RawJSON)
				fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%d\t%s\t%s\n",
					r.ReportID, r.Status, truncate(title, 30),
					float64(r.Total)/100, r.ExpenseCount, r.LastUpdated, ownerCol)
			}
			w.Flush()
			fmt.Fprintf(cmd.OutOrStdout(), "\n%d report(s).\n", len(items))
			return nil
		},
	}

	cmd.Flags().StringVar(&policyID, "policy-id", "", "Filter to a specific workspace")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (open, submitted, approved, reimbursed)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max reports to return (0 = unlimited)")
	cmd.Flags().BoolVar(&live, "live", false, "Issue a live /Search call and upsert results into the local store before reading")
	cmd.Flags().StringVar(&owner, "owner", "", "Filter live results to a specific person's email/login (requires `sync` to have populated the people cache)")
	cmd.Flags().BoolVar(&allVisible, "all-visible", false, "Disable the implicit owner-filter-to-me for --live (return every report the API chooses to surface)")

	return cmd
}

// runLiveReportSearch issues a /Search call with type=expense-report and
// upserts the response rows into the local store.
func runLiveReportSearch(
	cmd *cobra.Command,
	flags *rootFlags,
	db *store.Store,
	cfg *config.Config,
	owner string,
	allVisible bool,
	policyID string,
	status string,
) error {
	filter, err := buildSearchFilterFromFlags(db, cfg, owner, allVisible, "expense-report", policyID, status)
	if err != nil {
		return usageErr(err)
	}
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	q := newSearchQuery("expense-report", filter)
	resp, err := c.Search(q)
	if err != nil {
		return classifyAPIError(err)
	}
	nR, nE := ingestSearchResponse(db, resp)
	if flags != nil && !flags.asJSON && isTerminal(cmd.OutOrStdout()) {
		fmt.Fprintf(os.Stderr, "live: upserted %d report(s), %d expense(s).\n", nR, nE)
	}
	return nil
}
