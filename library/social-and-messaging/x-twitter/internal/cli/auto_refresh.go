package cli

import (
	"fmt"
	"time"

	"github.com/mvanhorn/printing-press-library/library/social-and-messaging/x-twitter/internal/cliutil"

	"github.com/spf13/cobra"
)

// defaultStaleAfter is the local-store age past which read commands print a
// hint to re-sync. Configurable per-invocation via flags / env in the future;
// hardcoded for now.
const defaultStaleAfter = 24 * time.Hour

// autoRefreshIfStale prints a one-line stderr hint when the local store is
// older than defaultStaleAfter. Does NOT auto-execute sync - the user controls
// when API calls happen. Designed to be called from PersistentPreRunE.
//
// Returns nil unless something unexpected went wrong reading the store path.
func autoRefreshIfStale(cmd *cobra.Command, dbPath string) error {
	if dbPath == "" {
		return nil
	}
	stale, age, err := cliutil.EnsureFresh(dbPath, defaultStaleAfter)
	if err != nil {
		return nil // non-fatal
	}
	if !stale {
		return nil
	}
	// Soft hint: stderr only, never blocks the command.
	if age == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "hint: local store is empty - run 'x-twitter-pp-cli sync followers' to populate")
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "hint: local store last synced %s - re-run 'sync' for fresher data\n", cliutil.FormatAge(age))
	return nil
}
