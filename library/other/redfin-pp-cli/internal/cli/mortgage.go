package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mvanhorn/printing-press-library/library/other/redfin-pp-cli/internal/store"

	"github.com/spf13/cobra"
)

func newMortgageCmd(flags *rootFlags) *cobra.Command {
	var downPct float64
	var downAbs int
	var rate float64
	var term int
	var taxes float64
	var insurance float64
	var compare bool
	var propertyID string
	var dbPath string

	cmd := &cobra.Command{
		Use:   "mortgage [price]",
		Short: "Calculate monthly mortgage payments",
		Long: `Calculate monthly mortgage payments with principal, interest, taxes,
insurance, and PMI estimates. Supports comparing 15yr vs 30yr terms
and fetching property price from the API or local store.`,
		Example: `  # Basic calculation
  redfin-pp-cli mortgage 750000 --down 20 --rate 6.5 --term 30

  # Compare 15yr vs 30yr
  redfin-pp-cli mortgage 750000 --compare

  # Use a property's current price
  redfin-pp-cli mortgage --property 12345`,
		RunE: func(cmd *cobra.Command, args []string) error {
			purchasePrice := 0.0

			// Determine price source
			if propertyID != "" {
				if flags.dryRun {
					// In --property mode, show what API call would be made
					c, cErr := flags.newClient()
					if cErr != nil {
						return cErr
					}
					_, _ = c.Get("/stingray/api/home/details/aboveTheFold", map[string]string{
						"propertyId":  propertyID,
						"accessLevel": "3",
					})
					return nil
				}
				p, err := fetchPropertyPrice(flags, propertyID, dbPath)
				if err != nil {
					return err
				}
				purchasePrice = p
				fmt.Fprintf(os.Stderr, "Using property %s price: $%s\n", propertyID, formatCompact(int64(purchasePrice)))
			} else if len(args) > 0 {
				p, err := strconv.ParseFloat(args[0], 64)
				if err != nil || p <= 0 {
					return fmt.Errorf("invalid price: %s", args[0])
				}
				purchasePrice = p
			} else {
				return cmd.Help()
			}

			// Calculate down payment
			downPayment := 0.0
			if downAbs > 0 {
				downPayment = float64(downAbs)
			} else {
				downPayment = purchasePrice * downPct / 100
			}
			downPctActual := downPayment / purchasePrice * 100

			// Estimate taxes and insurance if not provided
			annualTaxes := taxes
			if annualTaxes == 0 {
				annualTaxes = purchasePrice * 0.011 // 1.1% default
			}
			annualInsurance := insurance
			if annualInsurance == 0 {
				annualInsurance = purchasePrice * 0.0035 // 0.35% default
			}

			if compare {
				return mortgageCompare(cmd, flags, purchasePrice, downPayment, downPctActual, rate, annualTaxes, annualInsurance)
			}

			result := calculateMortgage(purchasePrice, downPayment, downPctActual, rate, term, annualTaxes, annualInsurance)

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printMortgageHuman(cmd, flags, result)
			}

			raw, _ := json.Marshal(result)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(raw), flags)
		},
	}

	cmd.Flags().Float64Var(&downPct, "down", 20, "Down payment percentage")
	cmd.Flags().IntVar(&downAbs, "down-amount", 0, "Down payment absolute amount (overrides --down percentage)")
	cmd.Flags().Float64Var(&rate, "rate", 6.5, "Annual interest rate (percent)")
	cmd.Flags().IntVar(&term, "term", 30, "Loan term in years")
	cmd.Flags().Float64Var(&taxes, "taxes", 0, "Annual property taxes (estimate 1.1% if not provided)")
	cmd.Flags().Float64Var(&insurance, "insurance", 0, "Annual insurance (estimate 0.35% if not provided)")
	cmd.Flags().BoolVar(&compare, "compare", false, "Compare 15yr vs 30yr side-by-side")
	cmd.Flags().StringVar(&propertyID, "property", "", "Fetch price from a property ID")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/redfin-pp-cli/redfin.db)")

	return cmd
}

type mortgageResult struct {
	PurchasePrice    float64 `json:"purchase_price"`
	DownPayment      float64 `json:"down_payment"`
	DownPaymentPct   float64 `json:"down_payment_pct"`
	LoanAmount       float64 `json:"loan_amount"`
	Rate             float64 `json:"rate"`
	Term             int     `json:"term_years"`
	MonthlyPI        float64 `json:"monthly_pi"`
	MonthlyTaxes     float64 `json:"monthly_taxes"`
	MonthlyInsurance float64 `json:"monthly_insurance"`
	MonthlyPMI       float64 `json:"monthly_pmi"`
	TotalMonthly     float64 `json:"total_monthly"`
	TotalOverLife     float64 `json:"total_over_life"`
	TotalInterest    float64 `json:"total_interest"`
}

func calculateMortgage(purchasePrice, downPayment, downPctActual, annualRate float64, termYears int, annualTaxes, annualInsurance float64) mortgageResult {
	loanAmount := purchasePrice - downPayment
	monthlyRate := annualRate / 100 / 12
	totalPayments := float64(termYears * 12)

	// M = P[r(1+r)^n]/[(1+r)^n-1]
	var monthlyPI float64
	if monthlyRate == 0 {
		monthlyPI = loanAmount / totalPayments
	} else {
		factor := math.Pow(1+monthlyRate, totalPayments)
		monthlyPI = loanAmount * (monthlyRate * factor) / (factor - 1)
	}

	monthlyTaxes := annualTaxes / 12
	monthlyInsurance := annualInsurance / 12

	// PMI if less than 20% down
	monthlyPMI := 0.0
	if downPctActual < 20 {
		monthlyPMI = loanAmount * 0.005 / 12 // ~0.5% annual PMI rate
	}

	totalMonthly := monthlyPI + monthlyTaxes + monthlyInsurance + monthlyPMI
	totalOverLife := totalMonthly * totalPayments
	totalInterest := (monthlyPI * totalPayments) - loanAmount

	return mortgageResult{
		PurchasePrice:    purchasePrice,
		DownPayment:      downPayment,
		DownPaymentPct:   downPctActual,
		LoanAmount:       loanAmount,
		Rate:             annualRate,
		Term:             termYears,
		MonthlyPI:        math.Round(monthlyPI*100) / 100,
		MonthlyTaxes:     math.Round(monthlyTaxes*100) / 100,
		MonthlyInsurance: math.Round(monthlyInsurance*100) / 100,
		MonthlyPMI:       math.Round(monthlyPMI*100) / 100,
		TotalMonthly:     math.Round(totalMonthly*100) / 100,
		TotalOverLife:     math.Round(totalOverLife*100) / 100,
		TotalInterest:    math.Round(totalInterest*100) / 100,
	}
}

func printMortgageHuman(cmd *cobra.Command, flags *rootFlags, r mortgageResult) error {
	headers := []string{"ITEM", "VALUE"}
	rows := [][]string{
		{"Purchase Price", fmt.Sprintf("$%s", formatCompact(int64(r.PurchasePrice)))},
		{"Down Payment", fmt.Sprintf("$%s (%.0f%%)", formatCompact(int64(r.DownPayment)), r.DownPaymentPct)},
		{"Loan Amount", fmt.Sprintf("$%s", formatCompact(int64(r.LoanAmount)))},
		{"Interest Rate", fmt.Sprintf("%.2f%%", r.Rate)},
		{"Loan Term", fmt.Sprintf("%d years", r.Term)},
		{"", ""},
		{"Monthly P&I", fmt.Sprintf("$%.2f", r.MonthlyPI)},
		{"Monthly Taxes", fmt.Sprintf("$%.2f", r.MonthlyTaxes)},
		{"Monthly Insurance", fmt.Sprintf("$%.2f", r.MonthlyInsurance)},
	}
	if r.MonthlyPMI > 0 {
		rows = append(rows, []string{"Monthly PMI", fmt.Sprintf("$%.2f", r.MonthlyPMI)})
	}
	rows = append(rows,
		[]string{"", ""},
		[]string{"Total Monthly", fmt.Sprintf("$%.2f", r.TotalMonthly)},
		[]string{"Total Over Life", fmt.Sprintf("$%s", formatCompact(int64(r.TotalOverLife)))},
		[]string{"Total Interest", fmt.Sprintf("$%s", formatCompact(int64(r.TotalInterest)))},
	)
	return flags.printTable(cmd, headers, rows)
}

func mortgageCompare(cmd *cobra.Command, flags *rootFlags, purchasePrice, downPayment, downPctActual, rate, annualTaxes, annualInsurance float64) error {
	r15 := calculateMortgage(purchasePrice, downPayment, downPctActual, rate, 15, annualTaxes, annualInsurance)
	r30 := calculateMortgage(purchasePrice, downPayment, downPctActual, rate, 30, annualTaxes, annualInsurance)

	type compareResult struct {
		Term15 mortgageResult `json:"term_15yr"`
		Term30 mortgageResult `json:"term_30yr"`
	}

	if wantsHumanTable(cmd.OutOrStdout(), flags) {
		headers := []string{"ITEM", "15-YEAR", "30-YEAR"}
		rows := [][]string{
			{"Purchase Price", fmt.Sprintf("$%s", formatCompact(int64(purchasePrice))), fmt.Sprintf("$%s", formatCompact(int64(purchasePrice)))},
			{"Loan Amount", fmt.Sprintf("$%s", formatCompact(int64(r15.LoanAmount))), fmt.Sprintf("$%s", formatCompact(int64(r30.LoanAmount)))},
			{"Rate", fmt.Sprintf("%.2f%%", r15.Rate), fmt.Sprintf("%.2f%%", r30.Rate)},
			{"Monthly P&I", fmt.Sprintf("$%.2f", r15.MonthlyPI), fmt.Sprintf("$%.2f", r30.MonthlyPI)},
			{"Total Monthly", fmt.Sprintf("$%.2f", r15.TotalMonthly), fmt.Sprintf("$%.2f", r30.TotalMonthly)},
			{"Total Interest", fmt.Sprintf("$%s", formatCompact(int64(r15.TotalInterest))), fmt.Sprintf("$%s", formatCompact(int64(r30.TotalInterest)))},
			{"Total Over Life", fmt.Sprintf("$%s", formatCompact(int64(r15.TotalOverLife))), fmt.Sprintf("$%s", formatCompact(int64(r30.TotalOverLife)))},
		}
		return flags.printTable(cmd, headers, rows)
	}

	raw, _ := json.Marshal(compareResult{Term15: r15, Term30: r30})
	return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(raw), flags)
}

func fetchPropertyPrice(flags *rootFlags, propertyID, dbPath string) (float64, error) {
	// Try local store first
	if dbPath == "" {
		home, _ := os.UserHomeDir()
		dbPath = filepath.Join(home, ".local", "share", "redfin-pp-cli", "redfin.db")
	}
	db, err := store.Open(dbPath)
	if err == nil {
		defer db.Close()
		prop, err := db.GetProperty(propertyID)
		if err == nil && prop != nil && prop.Price > 0 {
			return float64(prop.Price), nil
		}
	}

	// Try API
	c, err := flags.newClient()
	if err != nil {
		return 0, err
	}
	data, err := c.Get("/stingray/api/home/details/aboveTheFold", map[string]string{
		"propertyId":  propertyID,
		"accessLevel": "3",
	})
	if err != nil {
		return 0, classifyAPIError(err)
	}

	var resp map[string]any
	if json.Unmarshal(data, &resp) == nil {
		if p := findNestedFloat(resp, "price", "listPrice"); p > 0 {
			return p, nil
		}
	}

	return 0, fmt.Errorf("could not determine price for property %s", propertyID)
}
