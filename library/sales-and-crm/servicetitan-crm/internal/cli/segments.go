// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.
// Hand-written transcendence command (Phase 3): segments parent.

package cli

import "github.com/spf13/cobra"

// newSegmentsCmd is the parent command for segment composition + export.
// The transcendence value is in the `export` child which resolves boolean
// tag expressions across the local store. Group parent surfaces help only.
func newSegmentsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "segments",
		Short: "Compose and export customer/location segments by tag expression (local store)",
		Long: `Build customer or location segments from boolean combinations of tags
and filter predicates (recency, zone, etc.). All resolution runs against
the local SQLite store — run 'sync run' first to populate it.`,
	}
	cmd.AddCommand(newSegmentsExportCmd(flags))
	return cmd
}
