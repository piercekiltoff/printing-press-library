// Copyright 2026 matt-van-horn. Licensed under Apache-2.0.
//
// Shared helpers for list commands that issue a /Search query in --live
// mode. The filter builder resolves owner-filter-by-default, --owner email
// lookups, and --all-visible opt-out; the onyxData walker upserts results
// into the local store as a side effect so the fall-through local read sees
// fresh rows.

package cli

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/config"
	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/expensifysearch"
	"github.com/mvanhorn/printing-press-library/library/productivity/expensify/internal/store"
)

// buildSearchFilterFromFlags assembles a *expensifysearch.Filter tree for the
// expense/report list commands. Composition order: type -> owner -> policy ->
// status. Any zero-value leaf is skipped; the caller receives nil when no
// filters were requested.
//
// Owner-filter behavior:
//   - allVisible=true       : no "from" filter is added
//   - ownerFlag non-empty   : GetPersonByLogin lookup resolves to accountID
//   - otherwise             : cfg.ExpensifyAccountID is used; missing => error
func buildSearchFilterFromFlags(
	st *store.Store,
	cfg *config.Config,
	ownerFlag string,
	allVisible bool,
	typ string,
	policyID string,
	status string,
) (*expensifysearch.Filter, error) {
	var leaves []*expensifysearch.Filter

	if typ != "" {
		leaves = append(leaves, expensifysearch.Eq("type", typ))
	}

	if !allVisible {
		if ownerFlag != "" {
			if st == nil {
				return nil, fmt.Errorf("internal: cannot resolve --owner without a local store")
			}
			p, err := st.GetPersonByLogin(ownerFlag)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, fmt.Errorf("unknown owner %q — run `expensify-pp-cli sync` to refresh the people cache (or pass --all-visible)", ownerFlag)
				}
				return nil, fmt.Errorf("looking up --owner %q: %w", ownerFlag, err)
			}
			leaves = append(leaves, expensifysearch.Eq("from", strconv.FormatInt(p.AccountID, 10)))
		} else {
			if cfg == nil || cfg.ExpensifyAccountID == 0 {
				return nil, fmt.Errorf("cannot default owner filter to you: unknown accountID. Run `expensify-pp-cli sync` to identify your account (or pass --all-visible)")
			}
			leaves = append(leaves, expensifysearch.Eq("from", strconv.FormatInt(cfg.ExpensifyAccountID, 10)))
		}
	}

	if policyID != "" {
		leaves = append(leaves, expensifysearch.Eq("policyID", policyID))
	}
	if status != "" {
		leaves = append(leaves, expensifysearch.Eq("status", status))
	}

	return andChain(leaves), nil
}

// andChain folds a slice of filter leaves into a left-deep And tree. Empty
// slice returns nil (Search's Filters is *Filter — nil means "no filter").
func andChain(leaves []*expensifysearch.Filter) *expensifysearch.Filter {
	if len(leaves) == 0 {
		return nil
	}
	cur := leaves[0]
	for i := 1; i < len(leaves); i++ {
		cur = expensifysearch.And(cur, leaves[i])
	}
	return cur
}

// ingestSearchResponse walks a /Search response's onyxData and upserts every
// report_* / transactions_* row it finds into the local store. Returns the
// count of (reports, expenses) upserted. Mirrors sync.go's ingest pattern but
// ignores policies / personalDetails since /Search does not return those.
func ingestSearchResponse(st *store.Store, resp *expensifysearch.Response) (nReports, nExpenses int) {
	if st == nil || resp == nil {
		return 0, 0
	}
	for _, entry := range resp.OnyxData {
		key := entry.Key
		if key == "" {
			continue
		}
		// Parse entry.Value into generic map/slice. Value can be an object
		// (key -> row) or, rarely, a single row; walkMaps handles both.
		var val any
		if len(entry.Value) == 0 {
			continue
		}
		if err := json.Unmarshal(entry.Value, &val); err != nil {
			continue
		}
		// Some /Search entries nest under value.data.* — peek inside.
		inner := unwrapSnapshotData(val)
		switch {
		case strings.HasPrefix(key, "snapshot_"):
			// snapshot_<hash> wraps { data: { report_X: {...}, transactions_Y: {...} } }
			walkSnapshotData(st, inner, &nReports, &nExpenses)
		case strings.HasPrefix(key, "transactions") || strings.HasPrefix(key, "transaction_"):
			nExpenses += upsertTransactions(st, inner, "", "")
		case strings.HasPrefix(key, "reports") || strings.HasPrefix(key, "report_"):
			nReports += upsertReports(st, inner, "")
		}
	}
	return nReports, nExpenses
}

// unwrapSnapshotData returns the `.data` child of a snapshot entry when the
// caller passed the full snapshot value; otherwise returns val unchanged.
func unwrapSnapshotData(val any) any {
	m, ok := val.(map[string]any)
	if !ok {
		return val
	}
	if data, ok := m["data"]; ok {
		return data
	}
	return val
}

// walkSnapshotData iterates the inner `data` map of a snapshot_<hash> entry
// and routes each report_*/transactions_* child to the right upserter.
func walkSnapshotData(st *store.Store, inner any, nReports, nExpenses *int) {
	m, ok := inner.(map[string]any)
	if !ok {
		return
	}
	for k, child := range m {
		switch {
		case strings.HasPrefix(k, "report_"):
			// A single report row.
			if row, ok := child.(map[string]any); ok {
				*nReports += upsertReports(st, row, "")
			}
		case strings.HasPrefix(k, "transactions_") || strings.HasPrefix(k, "transaction_"):
			// A single transaction row.
			if row, ok := child.(map[string]any); ok {
				*nExpenses += upsertTransactions(st, row, "", "")
			}
		}
	}
}

// liveModeEnabled reports whether the command should issue a /Search in
// addition to (or instead of) reading the local store. Both an explicit
// --live bool and the persistent --data-source=live flag enable it.
func liveModeEnabled(live bool, flags *rootFlags) bool {
	if live {
		return true
	}
	if flags != nil && flags.dataSource == "live" {
		return true
	}
	return false
}

// newSearchQuery wraps a filter tree in a Query with type set and sane
// defaults. Kept as a helper so call sites read as one line.
func newSearchQuery(typ string, filter *expensifysearch.Filter) expensifysearch.Query {
	return expensifysearch.Query{
		Type:    typ,
		Filters: filter,
	}
}
