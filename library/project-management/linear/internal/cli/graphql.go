package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func newGraphQLCmd(flags *rootFlags) *cobra.Command {
	var queryStr string
	var queryFile string
	var varsStr string
	var varsFile string

	cmd := &cobra.Command{
		Use:   "graphql",
		Short: "Execute a raw GraphQL query or mutation",
		Long: `Send an arbitrary GraphQL query or mutation to the Linear API.
Useful for operations not covered by built-in commands.`,
		Example: `  # Inline query
  linear-pp-cli graphql --query '{ viewer { id name } }'

  # Query from file
  linear-pp-cli graphql --file query.graphql

  # With variables
  linear-pp-cli graphql --query 'query($id: String!) { issue(id: $id) { title } }' \
    --variables '{"id": "abc-123"}'

  # Pipe query from stdin
  echo '{ teams { nodes { id name } } }' | linear-pp-cli graphql`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			// Resolve query
			query := queryStr
			if queryFile != "" {
				data, err := os.ReadFile(queryFile)
				if err != nil {
					return fmt.Errorf("reading query file: %w", err)
				}
				query = string(data)
			}
			if query == "" {
				// Try stdin
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) == 0 {
					data, err := io.ReadAll(os.Stdin)
					if err != nil {
						return fmt.Errorf("reading stdin: %w", err)
					}
					query = string(data)
				}
			}
			if query == "" {
				return usageErr(fmt.Errorf("provide a query via --query, --file, or stdin"))
			}

			// Resolve variables
			var variables map[string]any
			if varsFile != "" {
				data, err := os.ReadFile(varsFile)
				if err != nil {
					return fmt.Errorf("reading variables file: %w", err)
				}
				if json.Unmarshal(data, &variables) != nil {
					return fmt.Errorf("parsing variables file as JSON")
				}
			} else if varsStr != "" {
				if json.Unmarshal([]byte(varsStr), &variables) != nil {
					return fmt.Errorf("parsing --variables as JSON")
				}
			}

			data, err := c.GraphQL(query, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
		},
	}

	cmd.Flags().StringVar(&queryStr, "query", "", "GraphQL query string")
	cmd.Flags().StringVar(&queryFile, "file", "", "Path to .graphql file")
	cmd.Flags().StringVar(&varsStr, "variables", "", "Variables as inline JSON")
	cmd.Flags().StringVar(&varsFile, "vars-file", "", "Path to variables JSON file")
	return cmd
}
