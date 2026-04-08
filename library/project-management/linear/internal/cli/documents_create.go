// Copyright 2026 user. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newDocumentsCreateCmd(flags *rootFlags) *cobra.Command {
	var bodyTitle string
	var bodyContent string
	var bodyProjectId string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a document",
		Example: `  linear-pp-cli documents create --title "Design Doc" --content "# Overview"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if bodyTitle == "" {
				return usageErr(fmt.Errorf("--title is required"))
			}
			if bodyContent == "" {
				return usageErr(fmt.Errorf("--content is required"))
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			mutation := `mutation($input: DocumentCreateInput!) {
				documentCreate(input: $input) {
					success
					document { id title }
				}
			}`

			input := map[string]any{
				"title":   bodyTitle,
				"content": bodyContent,
			}
			if bodyProjectId != "" {
				input["projectId"] = bodyProjectId
			}

			variables := map[string]any{
				"input": input,
			}

			data, err := c.GraphQL(mutation, variables)
			if err != nil {
				return classifyAPIError(err)
			}

			var resp struct {
				DocumentCreate struct {
					Success  bool            `json:"success"`
					Document json.RawMessage `json:"document"`
				} `json:"documentCreate"`
			}
			if err := json.Unmarshal(data, &resp); err != nil {
				return fmt.Errorf("parsing response: %w", err)
			}

			if !resp.DocumentCreate.Success {
				return apiErr(fmt.Errorf("document creation failed"))
			}

			return printOutputWithFlags(cmd.OutOrStdout(), resp.DocumentCreate.Document, flags)
		},
	}
	cmd.Flags().StringVar(&bodyTitle, "title", "", "Document title (required)")
	cmd.Flags().StringVar(&bodyContent, "content", "", "Document content in markdown (required)")
	cmd.Flags().StringVar(&bodyProjectId, "projectid", "", "Associated project ID")

	return cmd
}
