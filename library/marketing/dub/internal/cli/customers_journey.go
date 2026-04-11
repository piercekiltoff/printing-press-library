package cli

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/mvanhorn/printing-press-library/library/marketing/dub/internal/store"
	"github.com/spf13/cobra"
)

func newCustomersJourneyCmd(flags *rootFlags) *cobra.Command {
	var dbPath string

	cmd := &cobra.Command{
		Use:   "journey <customer-id>",
		Short: "Customer journey timeline — every interaction from click to purchase",
		Long: `Shows the full timeline of a customer's interactions: which links they
clicked, when they became a lead, and when they purchased. Joins customers,
events, and links to assemble a view no single API call provides.`,
		Example: `  # Show journey for a customer
  dub-pp-cli customers journey cust_abc123

  # As JSON
  dub-pp-cli customers journey cust_abc123 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			customerID := args[0]

			if dbPath == "" {
				dbPath = defaultDBPath("dub-pp-cli")
			}
			s, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("opening store: %w", err)
			}
			defer s.Close()

			// Get customer info
			customerData, err := s.Get("customers", customerID)
			if err != nil {
				return fmt.Errorf("fetching customer: %w", err)
			}

			type customerInfo struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Email     string `json:"email"`
				CreatedAt string `json:"createdAt"`
			}
			var cust customerInfo
			if customerData != nil {
				json.Unmarshal(customerData, &cust)
			} else {
				cust.ID = customerID
			}

			// Get events for this customer
			query := `
				SELECT e.data
				FROM events e
				WHERE json_extract(e.data, '$.customer.id') = ?
				   OR json_extract(e.data, '$.customerId') = ?
				ORDER BY json_extract(e.data, '$.timestamp') ASC
			`

			rows, err := s.Query(query, customerID, customerID)
			if err != nil {
				return fmt.Errorf("querying events: %w", err)
			}
			defer rows.Close()

			type journeyEvent struct {
				Timestamp string  `json:"timestamp"`
				Event     string  `json:"event"`
				Link      string  `json:"link"`
				LinkURL   string  `json:"link_url"`
				Amount    float64 `json:"amount,omitempty"`
			}

			var events []journeyEvent
			for rows.Next() {
				var rawData string
				if err := rows.Scan(&rawData); err != nil {
					return fmt.Errorf("scanning event: %w", err)
				}
				var obj map[string]any
				json.Unmarshal([]byte(rawData), &obj)

				je := journeyEvent{
					Timestamp: fmt.Sprintf("%v", obj["timestamp"]),
					Event:     fmt.Sprintf("%v", obj["event"]),
				}
				if link, ok := obj["link"].(map[string]any); ok {
					if sl, ok := link["shortLink"].(string); ok {
						je.Link = sl
					}
					if u, ok := link["url"].(string); ok {
						je.LinkURL = u
					}
				}
				if sale, ok := obj["sale"].(map[string]any); ok {
					if amt, ok := sale["amount"].(float64); ok {
						je.Amount = amt
					}
				}
				events = append(events, je)
			}
			if err := rows.Err(); err != nil {
				return err
			}

			type journeyResult struct {
				Customer customerInfo   `json:"customer"`
				Events   []journeyEvent `json:"events"`
				Summary  struct {
					TotalClicks int     `json:"total_clicks"`
					TotalLeads  int     `json:"total_leads"`
					TotalSales  int     `json:"total_sales"`
					TotalSpent  float64 `json:"total_spent"`
				} `json:"summary"`
			}

			result := journeyResult{Customer: cust, Events: events}
			for _, e := range events {
				switch e.Event {
				case "click":
					result.Summary.TotalClicks++
				case "lead":
					result.Summary.TotalLeads++
				case "sale":
					result.Summary.TotalSales++
					result.Summary.TotalSpent += e.Amount
				}
			}

			if flags.asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			if len(events) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No events found for customer %s. Run 'sync --full' to populate.\n", customerID)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Customer: %s (%s)\n", cust.Name, cust.Email)
			fmt.Fprintf(cmd.OutOrStdout(), "Summary: %d clicks → %d leads → %d sales ($%.2f)\n\n",
				result.Summary.TotalClicks, result.Summary.TotalLeads,
				result.Summary.TotalSales, result.Summary.TotalSpent/100)

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "TIMESTAMP\tEVENT\tLINK\tAMOUNT")
			for _, e := range events {
				amt := ""
				if e.Amount > 0 {
					amt = fmt.Sprintf("$%.2f", e.Amount/100)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Timestamp, e.Event, e.Link, amt)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&dbPath, "db", "", "Database path")

	return cmd
}
