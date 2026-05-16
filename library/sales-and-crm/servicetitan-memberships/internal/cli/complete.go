package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/mvanhorn/printing-press-library/library/sales-and-crm/servicetitan-memberships/internal/memberships"
)

func newCompleteCmd(flags *rootFlags) *cobra.Command {
	var jobID int64
	var dbPath string

	cmd := &cobra.Command{
		Use:   "complete <event-id>",
		Short: "Mark a recurring-service event complete with a required job link",
		Long: "Thin wrapper over the recurring-service-events mark-complete endpoint.\n" +
			"Required --job links the visit to the ServiceTitan job that fulfilled\n" +
			"the recurrence. After the API call succeeds, the local membership-\n" +
			"status snapshot is refreshed so subsequent overdue-events / risk\n" +
			"queries see the latest state in the same session.",
		Example: strings.Trim(`
  servicetitan-memberships-pp-cli complete 8801234 --job 7700321
  servicetitan-memberships-pp-cli complete 8801234 --job 7700321 --dry-run
`, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			eventID, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
			if err != nil {
				return usageErr(fmt.Errorf("event-id must be an integer: %w", err))
			}
			if dryRunOK(flags) {
				return flags.printJSON(cmd, map[string]any{
					"action":   "mark-complete",
					"event_id": eventID,
					"job_id":   jobID,
					"dry_run":  true,
				})
			}
			if jobID == 0 {
				return usageErr(fmt.Errorf("--job is required (the job that fulfilled the recurrence)"))
			}

			tenant := strings.TrimSpace(os.Getenv("ST_TENANT_ID"))
			if tenant == "" {
				return configErr(fmt.Errorf("ST_TENANT_ID is not set; complete requires a tenant id to build the API path"))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			path := strings.ReplaceAll("/tenant/{tenant}/recurring-service-events/{id}/mark-complete", "{tenant}", tenant)
			path = strings.ReplaceAll(path, "{id}", strconv.FormatInt(eventID, 10))
			data, statusCode, err := c.Post(path, map[string]any{"jobId": jobID})
			if err != nil {
				return classifyAPIError(err, flags)
			}

			// Best-effort local snapshot refresh — if the store isn't synced
			// yet, that's fine; the API call already succeeded and the user
			// gets the response. The snapshot is for keeping subsequent
			// audit queries in the same session honest.
			var snapshotted int
			if db, derr := openMembershipsStore(cmd, dbPath); derr == nil {
				defer db.Close()
				if w, _, serr := memberships.SnapshotMembershipStatus(db); serr == nil {
					snapshotted = w
				}
			}

			envelope := map[string]any{
				"action":              "mark-complete",
				"event_id":            eventID,
				"job_id":              jobID,
				"status":              statusCode,
				"success":             statusCode >= 200 && statusCode < 300,
				"snapshot_rows_added": snapshotted,
			}
			if len(data) > 0 {
				envelope["data"] = string(data)
			}
			return flags.printJSON(cmd, envelope)
		},
	}
	cmd.Flags().Int64Var(&jobID, "job", 0, "Job ID that fulfilled the recurrence (required for real runs)")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/servicetitan-memberships-pp-cli/data.db)")
	return cmd
}
