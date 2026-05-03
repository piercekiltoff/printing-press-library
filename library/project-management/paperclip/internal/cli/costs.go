package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newCostsCmd(flags *rootFlags) *cobra.Command {
	var companyID string

	cmd := &cobra.Command{
		Use:   "costs",
		Short: "Cost reports for a company",
		Long:  "View spending summaries, per-agent costs, budget utilization, and anomalies.",
	}

	getCompanyID := func(flag string) string {
		if flag != "" {
			return flag
		}
		return os.Getenv("PAPERCLIP_COMPANY_ID")
	}

	requireCompanyID := func(cid string) error {
		if cid == "" {
			return usageErr(fmt.Errorf("company ID required: use --company-id or set PAPERCLIP_COMPANY_ID"))
		}
		return nil
	}

	fetchAndPrint := func(cmd *cobra.Command, flags *rootFlags, path string) error {
		c, err := flags.newClient()
		if err != nil {
			return err
		}
		data, err := c.Get(path, nil)
		if err != nil {
			return classifyAPIError(err)
		}
		if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}
		var items []map[string]any
		if json.Unmarshal(data, &items) == nil && len(items) > 0 {
			return printAutoTable(cmd.OutOrStdout(), items)
		}
		var obj map[string]any
		if json.Unmarshal(data, &obj) == nil {
			return printAutoTable(cmd.OutOrStdout(), []map[string]any{obj})
		}
		return printOutputWithFlags(cmd.OutOrStdout(), data, flags)
	}

	// summary
	summaryCmd := &cobra.Command{
		Use:     "summary",
		Short:   "Overall spend vs budget for the company",
		Example: "  paperclip-pp-cli costs summary --company-id <id>",
		RunE: func(cmd *cobra.Command, args []string) error {
			cid := getCompanyID(companyID)
			if err := requireCompanyID(cid); err != nil {
				return err
			}
			return fetchAndPrint(cmd, flags, "/api/companies/"+cid+"/costs/summary")
		},
	}

	// by-agent
	byAgentCmd := &cobra.Command{
		Use:     "by-agent",
		Short:   "Cost breakdown by agent for this month",
		Example: "  paperclip-pp-cli costs by-agent --company-id <id> --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			cid := getCompanyID(companyID)
			if err := requireCompanyID(cid); err != nil {
				return err
			}
			return fetchAndPrint(cmd, flags, "/api/companies/"+cid+"/costs/by-agent")
		},
	}

	// by-project
	byProjectCmd := &cobra.Command{
		Use:     "by-project",
		Short:   "Cost breakdown by project for this month",
		Example: "  paperclip-pp-cli costs by-project --company-id <id>",
		RunE: func(cmd *cobra.Command, args []string) error {
			cid := getCompanyID(companyID)
			if err := requireCompanyID(cid); err != nil {
				return err
			}
			return fetchAndPrint(cmd, flags, "/api/companies/"+cid+"/costs/by-project")
		},
	}

	// by-provider
	byProviderCmd := &cobra.Command{
		Use:     "by-provider",
		Short:   "Cost breakdown by LLM provider for this month",
		Example: "  paperclip-pp-cli costs by-provider --company-id <id>",
		RunE: func(cmd *cobra.Command, args []string) error {
			cid := getCompanyID(companyID)
			if err := requireCompanyID(cid); err != nil {
				return err
			}
			return fetchAndPrint(cmd, flags, "/api/companies/"+cid+"/costs/by-provider")
		},
	}

	// by-biller
	byBillerCmd := &cobra.Command{
		Use:     "by-biller",
		Short:   "Cost breakdown by billing entity",
		Example: "  paperclip-pp-cli costs by-biller --company-id <id>",
		RunE: func(cmd *cobra.Command, args []string) error {
			cid := getCompanyID(companyID)
			if err := requireCompanyID(cid); err != nil {
				return err
			}
			return fetchAndPrint(cmd, flags, "/api/companies/"+cid+"/costs/by-biller")
		},
	}

	// budget
	budgetCmd := &cobra.Command{
		Use:     "budget",
		Short:   "Budget overview: utilization across the company",
		Example: "  paperclip-pp-cli costs budget --company-id <id>",
		RunE: func(cmd *cobra.Command, args []string) error {
			cid := getCompanyID(companyID)
			if err := requireCompanyID(cid); err != nil {
				return err
			}
			return fetchAndPrint(cmd, flags, "/api/companies/"+cid+"/budgets/overview")
		},
	}

	// anomalies
	var threshold float64
	anomaliesCmd := &cobra.Command{
		Use:   "anomalies",
		Short: "Flag agents spending above their budget threshold or in the top 3 spenders",
		Long: `Compares each agent's monthly spend to their budget. Agents spending more than
--threshold × their budget are flagged. If no budgets are set, the top 3 spenders are shown.`,
		Example: `  paperclip-pp-cli costs anomalies --company-id <id>
  paperclip-pp-cli costs anomalies --company-id <id> --threshold 0.5 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cid := getCompanyID(companyID)
			if err := requireCompanyID(cid); err != nil {
				return err
			}

			c, err := flags.newClient()
			if err != nil {
				return err
			}

			data, err := c.Get("/api/companies/"+cid+"/costs/by-agent", nil)
			if err != nil {
				return classifyAPIError(err)
			}
			var agents []map[string]any
			if err := json.Unmarshal(data, &agents); err != nil {
				return fmt.Errorf("parsing costs: %w", err)
			}

			type anomaly struct {
				AgentName      string  `json:"agentName"`
				SpentCents     int     `json:"spentMonthlyCents"`
				BudgetCents    int     `json:"budgetMonthlyCents"`
				UtilizationPct float64 `json:"utilizationPct"`
				Reason         string  `json:"reason"`
			}

			var flagged []anomaly
			hasBudgets := false

			for _, a := range agents {
				name, _ := a["agentName"].(string)
				spent, _ := a["costCents"].(float64)
				budget, _ := a["budgetMonthlyCents"].(float64)
				if budget > 0 {
					hasBudgets = true
					util := spent / budget
					if util >= threshold {
						flagged = append(flagged, anomaly{
							AgentName:      name,
							SpentCents:     int(spent),
							BudgetCents:    int(budget),
							UtilizationPct: util * 100,
							Reason:         fmt.Sprintf("%.0f%% of budget", util*100),
						})
					}
				}
			}

			if !hasBudgets {
				// Fall back to top 3 spenders
				sort.Slice(agents, func(i, j int) bool {
					si, _ := agents[i]["costCents"].(float64)
					sj, _ := agents[j]["costCents"].(float64)
					return si > sj
				})
				top := 3
				if len(agents) < top {
					top = len(agents)
				}
				for _, a := range agents[:top] {
					name, _ := a["agentName"].(string)
					spent, _ := a["costCents"].(float64)
					flagged = append(flagged, anomaly{
						AgentName:  name,
						SpentCents: int(spent),
						Reason:     "top spender (no budget set)",
					})
				}
			}

			if flags.asJSON || !isTerminal(cmd.OutOrStdout()) {
				out, _ := json.MarshalIndent(flagged, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}

			if len(flagged) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No anomalies detected.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "AGENT\tSPEND\tBUDGET\tUTILIZATION\tREASON")
			for _, f := range flagged {
				spend := fmt.Sprintf("$%.2f", float64(f.SpentCents)/100)
				budget := "-"
				if f.BudgetCents > 0 {
					budget = fmt.Sprintf("$%.2f", float64(f.BudgetCents)/100)
				}
				util := "-"
				if f.UtilizationPct > 0 {
					util = fmt.Sprintf("%.0f%%", f.UtilizationPct)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					truncate(f.AgentName, 30), spend, budget, util, f.Reason)
			}
			w.Flush()
			return nil
		},
	}
	anomaliesCmd.Flags().Float64Var(&threshold, "threshold", 0.8, "Utilization threshold (0.0–1.0) to flag as anomaly")

	cmd.PersistentFlags().StringVar(&companyID, "company-id", "", "Company ID (defaults to PAPERCLIP_COMPANY_ID env var)")
	cmd.AddCommand(summaryCmd, byAgentCmd, byProjectCmd, byProviderCmd, byBillerCmd, budgetCmd, anomaliesCmd)
	return cmd
}
