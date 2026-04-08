package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newUsersMeCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "me",
		Short:   "Get the authenticated user",
		Example: "  linear-pp-cli users me",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := flags.newClient()
			if err != nil {
				return err
			}

			query := `{ viewer { id name email displayName active admin } }`

			data, err := c.GraphQL(query, nil)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				Viewer json.RawMessage `json:"viewer"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.Viewer, flags)
		},
	}

	return cmd
}
