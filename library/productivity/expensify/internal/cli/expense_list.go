// Copyright 2026 matt-van-horn. Licensed under Apache-2.0.
// List expenses from the local store (populated by `sync`). Expensify's
// /Search dispatcher uses an undocumented filter DSL; the local store
// gives us reliable cross-year queries instead.
//
// With --live (or --data-source=live) the command issues a /Search call
// against the live API, upserts the returned rows into the local store as
// a side effect, then falls through to the local read so the user sees
// freshly-synced data.

package cli

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/config"
	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/store"
)

func newExpenseListCmd(flags *rootFlags) *cobra.Command {
	var policyID string
	var status string
	var limit int
	var live bool
	var owner string
	var allVisible bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your expenses from the local cache (or --live to query Expensify)",
		Example: `  expensify-pp-cli expense list
  expensify-pp-cli expense list --policy-id POLICY_ID_HERE --limit 20
  expensify-pp-cli expense list --live
  expensify-pp-cli expense list --live --owner teammate@example.com
  expensify-pp-cli expense list --live --all-visible
  expensify-pp-cli expense list --json`,
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
				if err := runLiveExpenseSearch(cmd, flags, db, cfg, owner, allVisible, policyID, status); err != nil {
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

			items, err := db.ListExpenses(filters)
			if err != nil {
				return apiErr(fmt.Errorf("listing expenses: %w", err))
			}
			if limit > 0 && len(items) > limit {
				items = items[:limit]
			}

			// Optionally narrow the displayed rows to the requested owner when
			// --live ran against the server (which already filtered), so the
			// local read doesn't accidentally include stale sibling rows.
			if liveModeEnabled(live, flags) && !allVisible {
				accountID, _ := resolveOwnerAccountID(db, nil, owner)
				_ = accountID // reserved for future local-side owner filtering
			}

			if len(items) == 0 {
				fmt.Fprintln(os.Stderr, "No expenses in local cache. Run `expensify-pp-cli sync` to fetch from Expensify.")
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				data, _ := json.Marshal(items)
				return printOutput(cmd.OutOrStdout(), data, true)
			}

			// Human table
			w := newTabWriter(cmd.OutOrStdout())
			fmt.Fprintln(w, "TX_ID\tDATE\tMERCHANT\tAMOUNT\tCATEGORY\tREPORT\tOWNER")
			for _, e := range items {
				merch := e.Merchant
				if merch == "" {
					merch = "(none)"
				}
				ownerCol := formatOwnerFromRaw(db, e.RawJSON)
				fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%s\t%s\t%s\n",
					e.TransactionID, e.Date, truncate(merch, 30),
					float64(e.Amount)/100, e.Category, e.ReportID, ownerCol)
			}
			w.Flush()
			fmt.Fprintf(cmd.OutOrStdout(), "\n%d expense(s).\n", len(items))
			return nil
		},
	}

	cmd.Flags().StringVar(&policyID, "policy-id", "", "Filter to a specific workspace")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (draft, submitted, approved, paid)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Max expenses to return (0 = unlimited)")
	cmd.Flags().BoolVar(&live, "live", false, "Issue a live /Search call and upsert results into the local store before reading")
	cmd.Flags().StringVar(&owner, "owner", "", "Filter live results to a specific person's email/login (requires `sync` to have populated the people cache)")
	cmd.Flags().BoolVar(&allVisible, "all-visible", false, "Disable the implicit owner-filter-to-me for --live (return every expense the API chooses to surface)")

	return cmd
}

// runLiveExpenseSearch issues a /Search call with type=expense, walks the
// response onyxData, and upserts every transaction row into the local
// store. Auth / API errors are classified so the CLI exits with the
// expected status code.
func runLiveExpenseSearch(
	cmd *cobra.Command,
	flags *rootFlags,
	db *store.Store,
	cfg *config.Config,
	owner string,
	allVisible bool,
	policyID string,
	status string,
) error {
	filter, err := buildSearchFilterFromFlags(db, cfg, owner, allVisible, "expense", policyID, status)
	if err != nil {
		return usageErr(err)
	}
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	q := newSearchQuery("expense", filter)
	resp, err := c.Search(q)
	if err != nil {
		return classifyAPIError(err)
	}
	nR, nE := ingestSearchResponse(db, resp)
	if flags != nil && !flags.asJSON && isTerminal(cmd.OutOrStdout()) {
		fmt.Fprintf(os.Stderr, "live: upserted %d expense(s), %d report(s).\n", nE, nR)
	}
	return nil
}

// resolveOwnerAccountID resolves the owner flag to an accountID, falling back
// to cfg.ExpensifyAccountID when the flag is empty. Returns 0 on any miss.
func resolveOwnerAccountID(db *store.Store, cfg *config.Config, owner string) (int64, error) {
	if owner != "" && db != nil {
		p, err := db.GetPersonByLogin(owner)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, fmt.Errorf("unknown owner %q", owner)
			}
			return 0, err
		}
		return p.AccountID, nil
	}
	if cfg != nil {
		return cfg.ExpensifyAccountID, nil
	}
	return 0, nil
}

// formatOwnerFromRaw extracts a likely ownerAccountID from an expense's raw
// JSON and renders it as "<display_name> <<login>>" when the people cache has
// a matching row; falls back to "accountID:N" otherwise, or empty string when
// no owner field is present.
func formatOwnerFromRaw(db *store.Store, raw string) string {
	if db == nil || raw == "" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ""
	}
	id := firstInt64(m, "accountID", "ownerAccountID", "submitterAccountID", "createdByAccountID")
	if id == 0 {
		return ""
	}
	return formatAccountID(db, id)
}

// formatAccountID renders an accountID via the people cache. Returns
// "<display_name> <<login>>" on hit; "accountID:N" on miss.
func formatAccountID(db *store.Store, id int64) string {
	if db == nil || id == 0 {
		return ""
	}
	p, err := db.GetPersonByAccountID(id)
	if err != nil || p == nil {
		return fmt.Sprintf("accountID:%d", id)
	}
	name := p.DisplayName
	if name == "" {
		name = p.Login
	}
	if p.Login != "" && name != p.Login {
		return fmt.Sprintf("%s <%s>", name, p.Login)
	}
	return name
}
