package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

type timelineEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Kind      string    `json:"kind"`
	Summary   string    `json:"summary"`
}

func newAgentsTimelineCmd(flags *rootFlags) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "timeline <agentId>",
		Short: "Chronological view of an agent's task sessions and config revisions",
		Long: `Merges task sessions and config revisions for a single agent into a
chronological activity stream. Useful for understanding what an agent has been
doing and why issues may have stalled.`,
		Example: `  paperclip-pp-cli agents timeline <agentId>
  paperclip-pp-cli agents timeline <agentId> --limit 50 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			var events []timelineEvent

			// Fetch task sessions
			sessData, err := c.Get("/api/agents/"+agentID+"/task-sessions", nil)
			if err == nil {
				var sessions []map[string]any
				if json.Unmarshal(sessData, &sessions) == nil {
					for _, s := range sessions {
						ts := parseTime(s, "createdAt", "startedAt")
						summary := fmt.Sprintf("task session")
						if sid, ok := s["id"].(string); ok {
							summary = "task session " + truncate(sid, 8)
						}
						if status, ok := s["status"].(string); ok {
							summary += " [" + status + "]"
						}
						if !ts.IsZero() {
							events = append(events, timelineEvent{Timestamp: ts, Kind: "session", Summary: summary})
						}
					}
				}
			}

			// Fetch config revisions
			revData, err := c.Get("/api/agents/"+agentID+"/config-revisions", nil)
			if err == nil {
				var revisions []map[string]any
				if json.Unmarshal(revData, &revisions) == nil {
					for _, r := range revisions {
						ts := parseTime(r, "createdAt")
						summary := "config revision"
						if rid, ok := r["id"].(string); ok {
							summary = "config revision " + truncate(rid, 8)
						}
						if !ts.IsZero() {
							events = append(events, timelineEvent{Timestamp: ts, Kind: "config", Summary: summary})
						}
					}
				}
			}

			// Sort descending
			sort.Slice(events, func(i, j int) bool {
				return events[i].Timestamp.After(events[j].Timestamp)
			})

			if limit > 0 && len(events) > limit {
				events = events[:limit]
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				out, _ := json.MarshalIndent(events, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}

			if len(events) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No timeline events found for this agent.")
				return nil
			}

			tw := newTabWriter(cmd.OutOrStdout())
			fmt.Fprintln(tw, "TIME\tTYPE\tSUMMARY")
			for _, e := range events {
				fmt.Fprintf(tw, "%s\t%s\t%s\n",
					e.Timestamp.Local().Format("2006-01-02 15:04"), e.Kind, e.Summary)
			}
			tw.Flush()
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum events to show")
	return cmd
}

// parseTime tries common timestamp field names and returns zero time if none found.
func parseTime(obj map[string]any, fields ...string) time.Time {
	for _, f := range fields {
		v, ok := obj[f].(string)
		if !ok || v == "" {
			continue
		}
		for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05.000Z"} {
			if t, err := time.Parse(layout, v); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}
