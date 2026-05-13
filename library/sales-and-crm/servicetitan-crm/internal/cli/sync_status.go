// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.
// Hand-written transcendence command (Phase 3): sync status.

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-crm/internal/store"
)

type syncStatusEntry struct {
	Resource     string `json:"resource"`
	RowCount     int    `json:"row_count"`
	LastSyncedAt string `json:"last_synced_at,omitempty"`
	Cursor       string `json:"cursor,omitempty"`
	StaleSeconds int64  `json:"stale_seconds,omitempty"`
}

// newSyncStatusCmd builds `sync-status` — reports per-resource sync
// freshness from the local store. Read-only; pure SQL over sync_state and
// resources tables.
func newSyncStatusCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:   "sync-status",
		Short: "Per-resource sync freshness and row counts from the local store",
		Long: `Reports the last-synced-at timestamp, row count, and remaining cursor
(if any) for each CRM resource family in the local SQLite store. Useful
to verify a sync ran end-to-end and to see how stale each cached resource
is before running a transcendence command that depends on it.

Read-only; runs against the local store. Run 'sync' first to populate.`,
		Example: strings.Trim(`
  servicetitan-crm-pp-cli sync-status --json
  servicetitan-crm-pp-cli sync-status --json --select entries.resource,entries.row_count
`, "\n"),
		Annotations: map[string]string{"mcp:read-only": "true", "pp:novel": "sync-status"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if dbPath == "" {
				dbPath = defaultDBPath("servicetitan-crm-pp-cli")
			}
			db, err := store.OpenWithContext(cmd.Context(), dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()

			now := time.Now().UTC()
			entries := []syncStatusEntry{}
			for _, resource := range defaultSyncResources() {
				e := syncStatusEntry{Resource: resource}
				// row count
				_ = db.DB().QueryRow(
					`SELECT COUNT(*) FROM resources WHERE resource_type = ?`, resource,
				).Scan(&e.RowCount)
				// sync_state cursor + timestamp
				row := db.DB().QueryRow(
					`SELECT cursor, last_sync_unix FROM sync_state WHERE resource_type = ?`, resource,
				)
				var unix int64
				_ = row.Scan(&e.Cursor, &unix)
				if unix > 0 {
					t := time.Unix(unix, 0).UTC()
					e.LastSyncedAt = t.Format(time.RFC3339)
					e.StaleSeconds = int64(now.Sub(t).Seconds())
				}
				entries = append(entries, e)
			}

			out := map[string]any{
				"db_path":    db.Path(),
				"checked_at": now.Format(time.RFC3339),
				"entries":    entries,
			}
			if flags.asJSON || (!isTerminal(cmd.OutOrStdout())) {
				return printJSONFiltered(cmd.OutOrStdout(), out, flags)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Sync status (%s):\n", db.Path())
			for _, e := range entries {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-22s rows=%-7d last=%-26s stale=%ds\n",
					e.Resource, e.RowCount, valueOrDash(e.LastSyncedAt), e.StaleSeconds)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "SQLite database path")
	return cmd
}

func valueOrDash(s string) string {
	if s == "" {
		return "(never)"
	}
	return s
}
