package cli

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/mvanhorn/printing-press-library/library/other/redfin-pp-cli/internal/store"

	"github.com/spf13/cobra"
)

type investResult struct {
	PropertyID        string  `json:"property_id"`
	Address           string  `json:"address"`
	Price             float64 `json:"price"`
	MonthlyRent       float64 `json:"monthly_rent"`
	DownPayment       float64 `json:"down_payment"`
	LoanAmount        float64 `json:"loan_amount"`
	MonthlyMortgage   float64 `json:"monthly_mortgage"`
	MonthlyTaxes      float64 `json:"monthly_taxes"`
	MonthlyInsurance  float64 `json:"monthly_insurance"`
	MonthlyHOA        float64 `json:"monthly_hoa"`
	MonthlyVacancy    float64 `json:"monthly_vacancy"`
	MonthlyManagement float64 `json:"monthly_management"`
	MonthlyMaint      float64 `json:"monthly_maintenance"`
	MonthlyCashFlow   float64 `json:"monthly_cash_flow"`
	AnnualCashFlow    float64 `json:"annual_cash_flow"`
	CapRate           float64 `json:"cap_rate_pct"`
	CashOnCash        float64 `json:"cash_on_cash_pct"`
	GrossRentMult     float64 `json:"gross_rent_multiplier"`
}

func newInvestCmd(flags *rootFlags) *cobra.Command {
	var rent float64
	var vacancy float64
	var downPct float64
	var rate float64
	var management float64
	var maintenance float64
	var dbPath string

	cmd := &cobra.Command{
		Use:   "invest <property-id>",
		Short: "Calculate investment metrics for a rental property",
		Long: `Calculate key investment metrics including cap rate, cash-on-cash return,
gross rent multiplier, and monthly cash flow. Fetches property data from the
API or local store and applies mortgage, tax, insurance, and expense estimates.`,
		Example: `  # Basic investment analysis
  redfin-pp-cli invest 12345 --rent 3500

  # With custom assumptions
  redfin-pp-cli invest 12345 --rent 3500 --vacancy 8 --down 25 --rate 7.0

  # With property management
  redfin-pp-cli invest 12345 --rent 4000 --management 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			propertyID := args[0]

			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".local", "share", "redfin-pp-cli", "redfin.db")
			}

			// Fetch property data
			price := 0.0
			address := ""
			hoa := 0.0

			// Try local store first
			db, dbErr := store.Open(dbPath)
			if dbErr == nil {
				defer db.Close()
				prop, err := db.GetProperty(propertyID)
				if err == nil && prop != nil {
					if prop.Price > 0 {
						price = float64(prop.Price)
					}
					address = prop.Address
					hoa = float64(prop.HOA)
				}
			}

			// Try API if no local data
			if price == 0 {
				c, err := flags.newClient()
				if err != nil {
					return err
				}
				data, err := c.Get("/stingray/api/home/details/aboveTheFold", map[string]string{
					"propertyId":  propertyID,
					"accessLevel": "3",
				})
				if err != nil {
					return classifyAPIError(err)
				}
				if flags.dryRun {
					return nil
				}
				var resp map[string]any
				if json.Unmarshal(data, &resp) == nil {
					if p := findNestedFloat(resp, "price", "listPrice"); p > 0 {
						price = p
					}
					if a := findNestedStr(resp, "streetAddress", "address"); a != "" {
						address = a
					}
					if h := findNestedFloat(resp, "hoa", "hoaDues"); h > 0 {
						hoa = h
					}
				}
			}

			if price == 0 {
				return fmt.Errorf("could not determine price for property %s", propertyID)
			}
			if rent == 0 {
				return fmt.Errorf("--rent is required (monthly rent amount)")
			}

			fmt.Fprintf(os.Stderr, "Analyzing property %s: %s ($%s)\n", propertyID, address, formatCompact(int64(price)))

			// Calculate mortgage
			downPayment := price * downPct / 100
			loanAmount := price - downPayment
			monthlyRate := rate / 100 / 12
			totalPayments := 30.0 * 12 // 30yr fixed

			var monthlyMortgage float64
			if monthlyRate == 0 {
				monthlyMortgage = loanAmount / totalPayments
			} else {
				factor := math.Pow(1+monthlyRate, totalPayments)
				monthlyMortgage = loanAmount * (monthlyRate * factor) / (factor - 1)
			}

			// Expenses
			monthlyTaxes := price * 0.011 / 12        // 1.1% annual
			monthlyInsurance := price * 0.0035 / 12    // 0.35% annual
			monthlyVacancy := rent * vacancy / 100
			monthlyManagement := rent * management / 100
			monthlyMaint := price * maintenance / 100 / 12

			// Cash flow
			totalExpenses := monthlyMortgage + monthlyTaxes + monthlyInsurance + hoa + monthlyVacancy + monthlyManagement + monthlyMaint
			monthlyCashFlow := rent - totalExpenses
			annualCashFlow := monthlyCashFlow * 12

			// NOI for cap rate (excludes mortgage, includes operating expenses)
			annualRent := rent * 12
			annualOperatingExpenses := (monthlyTaxes + monthlyInsurance + hoa + monthlyVacancy + monthlyManagement + monthlyMaint) * 12
			noi := annualRent - annualOperatingExpenses

			capRate := 0.0
			if price > 0 {
				capRate = noi / price * 100
			}

			cashOnCash := 0.0
			if downPayment > 0 {
				cashOnCash = annualCashFlow / downPayment * 100
			}

			grm := 0.0
			if annualRent > 0 {
				grm = price / annualRent
			}

			result := investResult{
				PropertyID:        propertyID,
				Address:           address,
				Price:             price,
				MonthlyRent:       rent,
				DownPayment:       downPayment,
				LoanAmount:        loanAmount,
				MonthlyMortgage:   math.Round(monthlyMortgage*100) / 100,
				MonthlyTaxes:      math.Round(monthlyTaxes*100) / 100,
				MonthlyInsurance:  math.Round(monthlyInsurance*100) / 100,
				MonthlyHOA:        hoa,
				MonthlyVacancy:    math.Round(monthlyVacancy*100) / 100,
				MonthlyManagement: math.Round(monthlyManagement*100) / 100,
				MonthlyMaint:      math.Round(monthlyMaint*100) / 100,
				MonthlyCashFlow:   math.Round(monthlyCashFlow*100) / 100,
				AnnualCashFlow:    math.Round(annualCashFlow*100) / 100,
				CapRate:           math.Round(capRate*100) / 100,
				CashOnCash:        math.Round(cashOnCash*100) / 100,
				GrossRentMult:     math.Round(grm*100) / 100,
			}

			if wantsHumanTable(cmd.OutOrStdout(), flags) {
				headers := []string{"METRIC", "VALUE"}
				rows := [][]string{
					{"Property", fmt.Sprintf("%s (%s)", address, propertyID)},
					{"Price", fmt.Sprintf("$%s", formatCompact(int64(price)))},
					{"Monthly Rent", fmt.Sprintf("$%.2f", rent)},
					{"Down Payment", fmt.Sprintf("$%s (%.0f%%)", formatCompact(int64(downPayment)), downPct)},
					{"Loan Amount", fmt.Sprintf("$%s", formatCompact(int64(loanAmount)))},
					{"", ""},
					{"Monthly Mortgage", fmt.Sprintf("$%.2f", result.MonthlyMortgage)},
					{"Monthly Taxes", fmt.Sprintf("$%.2f", result.MonthlyTaxes)},
					{"Monthly Insurance", fmt.Sprintf("$%.2f", result.MonthlyInsurance)},
					{"Monthly HOA", fmt.Sprintf("$%.2f", result.MonthlyHOA)},
					{"Monthly Vacancy", fmt.Sprintf("$%.2f (%.0f%%)", result.MonthlyVacancy, vacancy)},
					{"Monthly Management", fmt.Sprintf("$%.2f (%.0f%%)", result.MonthlyManagement, management)},
					{"Monthly Maintenance", fmt.Sprintf("$%.2f", result.MonthlyMaint)},
					{"", ""},
					{"Monthly Cash Flow", fmt.Sprintf("$%.2f", result.MonthlyCashFlow)},
					{"Annual Cash Flow", fmt.Sprintf("$%.2f", result.AnnualCashFlow)},
					{"Cap Rate", fmt.Sprintf("%.2f%%", result.CapRate)},
					{"Cash-on-Cash Return", fmt.Sprintf("%.2f%%", result.CashOnCash)},
					{"Gross Rent Multiplier", fmt.Sprintf("%.2f", result.GrossRentMult)},
				}
				return flags.printTable(cmd, headers, rows)
			}

			raw, _ := json.Marshal(result)
			return printOutputWithFlags(cmd.OutOrStdout(), json.RawMessage(raw), flags)
		},
	}

	cmd.Flags().Float64Var(&rent, "rent", 0, "Monthly rental income")
	cmd.Flags().Float64Var(&vacancy, "vacancy", 5, "Vacancy rate percentage")
	cmd.Flags().Float64Var(&downPct, "down", 20, "Down payment percentage")
	cmd.Flags().Float64Var(&rate, "rate", 6.5, "Annual interest rate (percent)")
	cmd.Flags().Float64Var(&management, "management", 0, "Property management fee percentage of rent")
	cmd.Flags().Float64Var(&maintenance, "maintenance", 1, "Annual maintenance as percentage of property value")
	cmd.Flags().StringVar(&dbPath, "db", "", "Database path (default: ~/.local/share/redfin-pp-cli/redfin.db)")

	return cmd
}
