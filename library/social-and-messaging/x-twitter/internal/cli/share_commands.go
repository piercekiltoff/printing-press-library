package cli

import (
	"fmt"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/social-and-messaging/x-twitter/internal/share"

	"github.com/spf13/cobra"
)

// newShareCmd is the parent for share subcommands. Exporter for cross-device
// state sync via git-friendly JSONL bundles.
func newShareCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "share",
		Short: "Export/import local store as portable JSONL bundles for sharing across machines",
		Long: strings.Trim(`
Export the local SQLite store as a JSONL bundle that can be checked into
git or shared between devices/users. Useful when bootstrapping a new
machine without re-running rate-limited API sync.
`, "\n"),
	}
	cmd.AddCommand(newShareExportCmd(flags))
	cmd.AddCommand(newShareImportCmd(flags))
	return cmd
}

func newShareExportCmd(flags *rootFlags) *cobra.Command {
	var resource, outputDir string
	cmd := &cobra.Command{
		Use:         "export",
		Short:       "Export a local table as a portable JSONL share bundle",
		Example:     "  x-twitter-pp-cli share export --resource follows --output ./shared",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			db, err := openXStore(flags)
			if err != nil {
				return err
			}
			defer db.Close()
			var query string
			switch resource {
			case "follows":
				query = `SELECT account_handle, direction, user_id, handle FROM x_follows`
			case "users":
				query = `SELECT user_id, handle, display_name, bio FROM x_users`
			default:
				return fmt.Errorf("unsupported resource %q (use: follows, users)", resource)
			}
			rows, err := db.DB().Query(query)
			if err != nil {
				return err
			}
			defer rows.Close()
			cols, _ := rows.Columns()
			var out []map[string]any
			for rows.Next() {
				vals := make([]any, len(cols))
				ptrs := make([]any, len(cols))
				for i := range vals {
					ptrs[i] = &vals[i]
				}
				if err := rows.Scan(ptrs...); err != nil {
					continue
				}
				m := make(map[string]any, len(cols))
				for i, name := range cols {
					m[name] = vals[i]
				}
				out = append(out, m)
			}
			path, err := share.Export(outputDir, resource, out)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "exported %d rows to %s\n", len(out), path)
			return nil
		},
	}
	cmd.Flags().StringVar(&resource, "resource", "follows", "Resource to export (follows, users)")
	cmd.Flags().StringVar(&outputDir, "output", "./x-twitter-share", "Output directory")
	return cmd
}

func newShareImportCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "import <bundle.share.jsonl>",
		Short:   "Import a share bundle into the local store",
		Example: "  x-twitter-pp-cli share import ./shared/follows.share.jsonl",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRunOK(flags) {
				return nil
			}
			resource, rows, err := share.Import(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "loaded %d rows of %s from bundle (write-back not yet implemented)\n", len(rows), resource)
			return nil
		},
	}
	return cmd
}
