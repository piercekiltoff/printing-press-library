// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mvanhorn/printing-press-library/library/project-management/linear/internal/store"
	"github.com/spf13/cobra"
)

func newWorkflowCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Compound workflows that combine multiple API operations",
	}

	cmd.AddCommand(newWorkflowArchiveCmd(flags))
	cmd.AddCommand(newWorkflowStatusCmd(flags))

	return cmd
}

func newWorkflowArchiveCmd(flags *rootFlags) *cobra.Command {
	var dbPath string
	var full bool

	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Sync all resources to local store for offline access and search",
		Long: `Archive fetches all syncable resources from the API and stores them in a
local SQLite database. Supports incremental sync (only new data since last run)
and full resync. After archiving, use 'search' for instant full-text search.`,
		Example: `  # Archive all resources
  linear-pp-cli workflow archive

  # Full re-archive (ignore previous sync state)
  linear-pp-cli workflow archive --full`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			c.NoCache = true

			if dbPath == "" {
				dbPath = defaultDBPath("linear-pp-cli")
			}
			s, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer s.Close()

			if full {
				s.ClearSyncCursors()
			}

			type syncResource struct {
				name     string
				query    string
				dataPath string
				upsert   func(s *store.Store, items []json.RawMessage) error
			}

			resources := []syncResource{
				{
					name:     "teams",
					query:    `{ teams(first: 250) { nodes { id name key description } pageInfo { hasNextPage endCursor } } }`,
					dataPath: "teams",
					upsert: func(s *store.Store, items []json.RawMessage) error {
						return s.UpsertBatch("teams", items)
					},
				},
				{
					name:     "users",
					query:    `{ users(first: 250) { nodes { id name email displayName active admin } pageInfo { hasNextPage endCursor } } }`,
					dataPath: "users",
					upsert: func(s *store.Store, items []json.RawMessage) error {
						return s.UpsertBatch("users", items)
					},
				},
				{
					name:     "workflow_states",
					query:    `{ workflowStates(first: 250) { nodes { id name color type team { id } position } pageInfo { hasNextPage endCursor } } }`,
					dataPath: "workflowStates",
					upsert: func(s *store.Store, items []json.RawMessage) error {
						return s.UpsertBatch("workflow_states", items)
					},
				},
				{
					name:     "labels",
					query:    `{ issueLabels(first: 250) { nodes { id name color } pageInfo { hasNextPage endCursor } } }`,
					dataPath: "issueLabels",
					upsert: func(s *store.Store, items []json.RawMessage) error {
						return s.UpsertBatch("labels", items)
					},
				},
				{
					name:     "cycles",
					query:    `{ cycles(first: 100) { nodes { id number name startsAt endsAt completedAt team { id } } pageInfo { hasNextPage endCursor } } }`,
					dataPath: "cycles",
					upsert: func(s *store.Store, items []json.RawMessage) error {
						return s.UpsertBatch("cycles", items)
					},
				},
				{
					name:     "projects",
					query:    `{ projects(first: 100) { nodes { id name description state slugId startDate targetDate lead { id } } pageInfo { hasNextPage endCursor } } }`,
					dataPath: "projects",
					upsert: func(s *store.Store, items []json.RawMessage) error {
						for _, item := range items {
							if err := s.UpsertProjects(item); err != nil {
								return err
							}
						}
						return nil
					},
				},
				{
					name: "issues",
					query: `query($after: String) { issues(first: 100, after: $after) { nodes { id identifier title description priority estimate dueDate createdAt updatedAt completedAt canceledAt team { id } assignee { id } state { id name } project { id } cycle { id } parent { id } labels { nodes { id name } } } pageInfo { hasNextPage endCursor } } }`,
					dataPath: "issues",
					upsert: func(s *store.Store, items []json.RawMessage) error {
						for _, item := range items {
							if err := s.UpsertIssues(item); err != nil {
								return err
							}
						}
						return nil
					},
				},
				{
					name:     "comments",
					query:    `{ comments(first: 100) { nodes { id body createdAt updatedAt issue { id } user { id } } pageInfo { hasNextPage endCursor } } }`,
					dataPath: "comments",
					upsert: func(s *store.Store, items []json.RawMessage) error {
						return s.UpsertBatch("comments", items)
					},
				},
				{
					name:     "documents",
					query:    `{ documents(first: 100) { nodes { id title content slugId createdAt updatedAt project { id } } pageInfo { hasNextPage endCursor } } }`,
					dataPath: "documents",
					upsert: func(s *store.Store, items []json.RawMessage) error {
						return s.UpsertBatch("documents", items)
					},
				},
			}

			totalSynced := 0

			for _, res := range resources {
				fmt.Fprintf(cmd.ErrOrStderr(), "Syncing %s...", res.name)

				items, fetchErr := c.GraphQLPaginated(res.query, nil, res.dataPath)
				if fetchErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), " error: %v\n", fetchErr)
					continue
				}

				fmt.Fprintf(cmd.ErrOrStderr(), " %d items\n", len(items))

				if len(items) > 0 {
					if err := res.upsert(s, items); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "  warning: store %s: %v\n", res.name, err)
					}
					cursor := ""
					if len(items) > 0 {
						var last struct {
							ID string `json:"id"`
						}
						json.Unmarshal(items[len(items)-1], &last)
						cursor = last.ID
					}
					s.SaveSyncState(res.name, cursor, len(items))
				}

				totalSynced += len(items)
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"resources_synced": len(resources),
					"total_items":     totalSynced,
					"store_path":      dbPath,
					"timestamp":       time.Now().UTC().Format(time.RFC3339),
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Archived %d items across %d resources to %s\n", totalSynced, len(resources), dbPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.config/linear-pp-cli/store.db)")
	cmd.Flags().BoolVar(&full, "full", false, "Full re-archive (ignore previous sync state)")

	return cmd
}

func newWorkflowStatusCmd(flags *rootFlags) *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show local archive status and sync state for all resources",
		Example: `  # Show archive status
  linear-pp-cli workflow status

  # Show status as JSON
  linear-pp-cli workflow status --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				dbPath = defaultDBPath("linear-pp-cli")
			}
			s, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer s.Close()

			status, err := s.Status()
			if err != nil {
				return err
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(status)
			}

			if len(status) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No archived data. Run 'workflow archive' to sync.")
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Archive Status:")
			total := 0
			for resource, count := range status {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %d items\n", resource, count)
				total += count
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n  Total: %d items\n", total)
			fmt.Fprintf(cmd.OutOrStdout(), "  Store: %s\n", dbPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")

	return cmd
}

func defaultDBPath(name string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", name, "store.db")
}
