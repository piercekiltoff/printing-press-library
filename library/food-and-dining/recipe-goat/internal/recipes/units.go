package recipes

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// ConvertIngredients takes a list of raw ingredient strings and a direction
// ("metric" or "us") and returns converted strings. Preserves unrecognized
// lines verbatim. The conversion targets the common 90% of cooking units —
// cups, tbsp, tsp, lb/oz, fl oz, and Fahrenheit → Celsius for temperatures
// that sneak into ingredient annotations ("water heated to 110°F").
func ConvertIngredients(ings []string, target string) []string {
	out := make([]string, len(ings))
	for i, line := range ings {
		switch strings.ToLower(target) {
		case "metric":
			out[i] = toMetric(line)
		case "us":
			out[i] = toUS(line)
		default:
			out[i] = line
		}
	}
	return out
}

// ConvertInstructionsTemps applies Fahrenheit→Celsius to every line when
// target=="metric", and Celsius→Fahrenheit when target=="us". Only temperature
// mentions are touched — bodies of instructions otherwise pass through.
func ConvertInstructionsTemps(steps []string, target string) []string {
	out := make([]string, len(steps))
	for i, s := range steps {
		switch strings.ToLower(target) {
		case "metric":
			out[i] = rewriteTempsFToC(s)
		case "us":
			out[i] = rewriteTempsCToF(s)
		default:
			out[i] = s
		}
	}
	return out
}

// --- metric conversion ---------------------------------------------------

// metricUnitRe captures: <qty> <unit>. qty matches the same forms the scaler
// accepts ("2", "1 1/2", "1/2", "1.5"). Unit capture is case-insensitive.
var metricUnitRe = regexp.MustCompile(`(?i)^\s*((?:\d+\s+\d+/\d+)|(?:\d+/\d+)|(?:\d*\.\d+)|(?:\d+))\s*(cups?|c\.?|tablespoons?|tbsps?|tbsp\.?|tbs\.?|T\.?|teaspoons?|tsps?|tsp\.?|t\.?|pounds?|lbs?\.?|ounces?|oz\.?|fluid\s+ounces?|fl\.?\s*oz\.?)\b`)

// fToCRe matches Fahrenheit temperatures in text, e.g. "350°F", "350 F",
// "350 degrees F", "350°Fahrenheit".
var fToCRe = regexp.MustCompile(`(?i)(\d{2,3})\s*(?:°\s*F\b|°F\b|\s+degrees?\s+F(?:ahrenheit)?\b|\s*F\b(?:ahrenheit)?)`)

// cToFRe matches Celsius temperatures.
var cToFRe = regexp.MustCompile(`(?i)(\d{2,3})\s*(?:°\s*C\b|°C\b|\s+degrees?\s+C(?:elsius)?\b)`)

func toMetric(line string) string {
	m := metricUnitRe.FindStringSubmatchIndex(line)
	if m == nil {
		return rewriteTempsFToC(line)
	}
	qtyStr := line[m[2]:m[3]]
	unitStr := strings.ToLower(strings.TrimSpace(line[m[4]:m[5]]))
	rest := line[m[1]:]
	qty, ok := parseQtyFloat(qtyStr)
	if !ok {
		return rewriteTempsFToC(line)
	}

	// Dispatch by normalized unit.
	switch {
	case strings.HasPrefix(unitStr, "cup") || unitStr == "c" || unitStr == "c.":
		// Flour-ish vs sugar-ish vs water-ish.
		ingLower := strings.ToLower(rest)
		grams := 0.0
		isWeight := false
		switch {
		case strings.Contains(ingLower, "flour"):
			grams = 120 * qty
			isWeight = true
		case strings.Contains(ingLower, "sugar") && !strings.Contains(ingLower, "powdered"):
			grams = 200 * qty
			isWeight = true
		case strings.Contains(ingLower, "powdered sugar") || strings.Contains(ingLower, "confectioners"):
			grams = 120 * qty
			isWeight = true
		case strings.Contains(ingLower, "cocoa"):
			grams = 85 * qty
			isWeight = true
		case strings.Contains(ingLower, "butter"):
			grams = 227 * qty
			isWeight = true
		}
		if isWeight {
			return fmt.Sprintf("%s g%s", formatMetric(grams), rest)
		}
		ml := 240 * qty
		return fmt.Sprintf("%s ml%s", formatMetric(ml), rest)
	case strings.HasPrefix(unitStr, "tablespoon") || strings.HasPrefix(unitStr, "tbsp") || strings.HasPrefix(unitStr, "tbs"):
		// Note: uppercase "T" as a capital-T shorthand is handled by the regex
		// (case-insensitive), so unitStr lowercases to "t". We disambiguate
		// capital-T tablespoons vs lowercase-t teaspoons by requiring the
		// explicit "tbsp"/"tablespoon" spellings here — ambiguous single-letter
		// units fall through to the teaspoon case, which is the safer default.
		ml := 15 * qty
		return fmt.Sprintf("%s ml%s", formatMetric(ml), rest)
	case strings.HasPrefix(unitStr, "teaspoon") || strings.HasPrefix(unitStr, "tsp") || unitStr == "t" || unitStr == "t.":
		ml := 5 * qty
		return fmt.Sprintf("%s ml%s", formatMetric(ml), rest)
	case strings.HasPrefix(unitStr, "pound") || strings.HasPrefix(unitStr, "lb"):
		g := 454 * qty
		return fmt.Sprintf("%s g%s", formatMetric(g), rest)
	case strings.Contains(unitStr, "fl") && strings.Contains(unitStr, "oz"):
		ml := 30 * qty
		return fmt.Sprintf("%s ml%s", formatMetric(ml), rest)
	case strings.HasPrefix(unitStr, "ounce") || unitStr == "oz" || unitStr == "oz.":
		g := 28 * qty
		return fmt.Sprintf("%s g%s", formatMetric(g), rest)
	}
	return rewriteTempsFToC(line)
}

func toUS(line string) string {
	// Best-effort reverse conversion for ml → cups/tbsp/tsp and g → oz.
	reML := regexp.MustCompile(`(?i)^\s*((?:\d+\s+\d+/\d+)|(?:\d+/\d+)|(?:\d*\.\d+)|(?:\d+))\s*(ml|milliliters?|millilitres?|g|grams?)\b`)
	m := reML.FindStringSubmatchIndex(line)
	if m == nil {
		return rewriteTempsCToF(line)
	}
	qtyStr := line[m[2]:m[3]]
	unit := strings.ToLower(strings.TrimSpace(line[m[4]:m[5]]))
	rest := line[m[1]:]
	qty, ok := parseQtyFloat(qtyStr)
	if !ok {
		return rewriteTempsCToF(line)
	}
	switch {
	case unit == "ml" || strings.HasPrefix(unit, "milli"):
		// Prefer cup if ≥ 180ml; else tbsp if ≥ 12ml; else tsp.
		switch {
		case qty >= 180:
			return fmt.Sprintf("%s cups%s", formatUSFrac(qty/240.0), rest)
		case qty >= 12:
			return fmt.Sprintf("%s tbsp%s", formatUSFrac(qty/15.0), rest)
		default:
			return fmt.Sprintf("%s tsp%s", formatUSFrac(qty/5.0), rest)
		}
	case unit == "g" || strings.HasPrefix(unit, "gram"):
		// Prefer lb if ≥ 300g, else oz.
		if qty >= 300 {
			return fmt.Sprintf("%s lb%s", formatUSFrac(qty/454.0), rest)
		}
		return fmt.Sprintf("%s oz%s", formatUSFrac(qty/28.0), rest)
	}
	return rewriteTempsCToF(line)
}

// rewriteTempsFToC replaces Fahrenheit mentions with Celsius (rounded to the
// nearest 5°C — good enough for oven temperatures).
func rewriteTempsFToC(s string) string {
	return fToCRe.ReplaceAllStringFunc(s, func(match string) string {
		// Extract the integer portion.
		nm := regexp.MustCompile(`\d+`).FindString(match)
		if nm == "" {
			return match
		}
		f, err := strconv.Atoi(nm)
		if err != nil {
			return match
		}
		c := int(math.Round(float64(f-32)*5.0/9.0/5.0)) * 5
		return fmt.Sprintf("%d°C", c)
	})
}

func rewriteTempsCToF(s string) string {
	return cToFRe.ReplaceAllStringFunc(s, func(match string) string {
		nm := regexp.MustCompile(`\d+`).FindString(match)
		if nm == "" {
			return match
		}
		c, err := strconv.Atoi(nm)
		if err != nil {
			return match
		}
		f := int(math.Round(float64(c)*9.0/5.0 + 32))
		return fmt.Sprintf("%d°F", f)
	})
}

// parseQtyFloat parses the same forms as scaling.parseQty but returns a float
// so we can do simple metric arithmetic without pulling in big.Rat here.
func parseQtyFloat(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if parts := strings.Fields(s); len(parts) == 2 && strings.Contains(parts[1], "/") {
		whole, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, false
		}
		f, ok := parseFractionFloat(parts[1])
		if !ok {
			return 0, false
		}
		return float64(whole) + f, true
	}
	if strings.Contains(s, "/") {
		return parseFractionFloat(s)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, true
	}
	return 0, false
}

func parseFractionFloat(s string) (float64, bool) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, false
	}
	d, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || d == 0 {
		return 0, false
	}
	return float64(n) / float64(d), true
}

// formatMetric renders a metric quantity to an appropriate precision: integer
// for ≥ 10, one decimal for 1–10, two decimals below 1.
func formatMetric(v float64) string {
	switch {
	case v >= 100:
		// Round to nearest 5 for nicer oven-like numbers.
		return strconv.Itoa(int(math.Round(v/5) * 5))
	case v >= 10:
		return strconv.Itoa(int(math.Round(v)))
	case v >= 1:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", v), "0"), ".")
	default:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
	}
}

// formatUSFrac renders a US quantity rounded to the nearest 1/4.
func formatUSFrac(v float64) string {
	q := math.Round(v*4) / 4
	if q == math.Floor(q) {
		return strconv.Itoa(int(q))
	}
	whole := int(math.Floor(q))
	rem := q - float64(whole)
	// rem in {0.25, 0.5, 0.75}
	frac := ""
	switch {
	case math.Abs(rem-0.25) < 0.01:
		frac = "1/4"
	case math.Abs(rem-0.5) < 0.01:
		frac = "1/2"
	case math.Abs(rem-0.75) < 0.01:
		frac = "3/4"
	default:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".")
	}
	if whole == 0 {
		return frac
	}
	return fmt.Sprintf("%d %s", whole, frac)
}
