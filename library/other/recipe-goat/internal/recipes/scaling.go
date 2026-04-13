package recipes

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// qtyRe matches a leading quantity in an ingredient line. Supports:
//
//	"2"           → whole
//	"1 1/2"       → mixed
//	"1/2"         → fraction
//	"1.5"         → decimal
//
// The pattern captures the full prefix up to the first non-quantity char.
var qtyRe = regexp.MustCompile(`^\s*((?:\d+\s+\d+/\d+)|(?:\d+/\d+)|(?:\d*\.\d+)|(?:\d+))`)

// ScaleIngredients scales each ingredient line's leading quantity by
// toYield/fromYield, preserving the rest of the line verbatim. When a line
// has no quantity, it's returned unchanged.
func ScaleIngredients(ingredients []string, fromYield, toYield int) []string {
	if fromYield <= 0 || toYield <= 0 || fromYield == toYield {
		// Copy-return for API symmetry.
		out := make([]string, len(ingredients))
		copy(out, ingredients)
		return out
	}
	ratio := new(big.Rat).SetFrac(big.NewInt(int64(toYield)), big.NewInt(int64(fromYield)))
	out := make([]string, len(ingredients))
	for i, line := range ingredients {
		out[i] = scaleOneIngredient(line, ratio)
	}
	return out
}

func scaleOneIngredient(line string, ratio *big.Rat) string {
	m := qtyRe.FindStringSubmatchIndex(line)
	if m == nil {
		return line
	}
	qtyStr := line[m[2]:m[3]]
	rest := line[m[3]:]
	qty, ok := parseQty(qtyStr)
	if !ok {
		return line
	}
	scaled := new(big.Rat).Mul(qty, ratio)
	return formatQty(scaled) + rest
}

// parseQty parses "1", "1 1/2", "1/2", or "1.5" into a big.Rat.
func parseQty(s string) (*big.Rat, bool) {
	s = strings.TrimSpace(s)
	// mixed: "1 1/2"
	if parts := strings.Fields(s); len(parts) == 2 && strings.Contains(parts[1], "/") {
		whole, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, false
		}
		frac, ok := parseFraction(parts[1])
		if !ok {
			return nil, false
		}
		r := new(big.Rat).SetInt64(int64(whole))
		return r.Add(r, frac), true
	}
	if strings.Contains(s, "/") {
		return parseFraction(s)
	}
	if strings.Contains(s, ".") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, false
		}
		r := new(big.Rat)
		r.SetFloat64(f)
		return r, true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil, false
	}
	return new(big.Rat).SetInt64(int64(n)), true
}

func parseFraction(s string) (*big.Rat, bool) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return nil, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, false
	}
	d, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || d == 0 {
		return nil, false
	}
	return new(big.Rat).SetFrac(big.NewInt(int64(n)), big.NewInt(int64(d))), true
}

// formatQty renders a big.Rat rounded to the nearest 1/8, preferring "1 1/2"
// over "1.5" and stripping trailing ".0"/"0" where applicable.
func formatQty(r *big.Rat) string {
	// Round to nearest 1/8 via (round(8r))/8.
	eight := big.NewInt(8)
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt64(8))
	num := new(big.Int).Set(scaled.Num())
	den := new(big.Int).Set(scaled.Denom())
	// Round-half-up.
	half := new(big.Int).Quo(den, big.NewInt(2))
	if num.Sign() >= 0 {
		num.Add(num, half)
	} else {
		num.Sub(num, half)
	}
	q := new(big.Int).Quo(num, den)
	// rounded = q/8
	whole := new(big.Int).Quo(q, eight)
	remainder := new(big.Int).Rem(q, eight)
	if remainder.Sign() == 0 {
		return whole.String()
	}
	// Reduce remainder/8.
	frac := new(big.Rat).SetFrac(remainder, eight)
	numStr := frac.Num().String()
	denStr := frac.Denom().String()
	if whole.Sign() == 0 {
		return fmt.Sprintf("%s/%s", numStr, denStr)
	}
	return fmt.Sprintf("%s %s/%s", whole.String(), numStr, denStr)
}

// yieldRe extracts a leading integer from strings like "4 servings" or "6".
var yieldRe = regexp.MustCompile(`(\d+)`)

// ParseYield returns the first integer in s, or 0 when none found.
func ParseYield(s string) int {
	m := yieldRe.FindString(s)
	if m == "" {
		return 0
	}
	n, err := strconv.Atoi(m)
	if err != nil {
		return 0
	}
	return n
}
