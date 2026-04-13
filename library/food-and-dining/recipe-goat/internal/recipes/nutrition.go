package recipes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// BackfillNutrition returns a best-effort nutrition map computed from USDA
// FoodData Central search matches per ingredient. Requires USDA_FDC_API_KEY
// in env. The second return value is a provenance string — "usda-computed"
// when at least one API call produced usable data, "unavailable" otherwise.
//
// When no API key is configured the function returns (recipe.Nutrition, "site",
// nil) — callers can rely on the provenance string to decide how to surface
// the numbers.
func BackfillNutrition(ctx context.Context, client *http.Client, recipe *Recipe) (map[string]string, string, error) {
	if recipe == nil {
		return nil, "unavailable", nil
	}
	// Already well-populated: trust the source.
	if len(recipe.Nutrition) >= 3 && recipe.Nutrition["calories"] != "" {
		return recipe.Nutrition, "site", nil
	}
	apiKey := strings.TrimSpace(os.Getenv("USDA_FDC_API_KEY"))
	if apiKey == "" {
		if len(recipe.Nutrition) > 0 {
			return recipe.Nutrition, "site", nil
		}
		return nil, "unavailable", nil
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	servings := ParseYield(recipe.RecipeYield)
	if servings <= 0 {
		servings = 4
	}

	type per100 struct {
		calories float64
		protein  float64
		carbs    float64
		fat      float64
	}
	totalKcal := 0.0
	totalProt := 0.0
	totalCarb := 0.0
	totalFat := 0.0
	successful := 0

	for _, raw := range recipe.RecipeIngredient {
		name, grams := parseIngredientForFDC(raw)
		if name == "" {
			continue
		}
		if grams == 0 {
			// Last-resort fallback when no qty/unit was parseable.
			grams = 80
		}

		p, ok := fdcLookupPer100g(ctx, client, apiKey, name)
		if !ok {
			// Respect the 3 req/sec cap even on miss.
			time.Sleep(334 * time.Millisecond)
			continue
		}
		successful++
		factor := grams / 100.0
		totalKcal += p.calories * factor
		totalProt += p.protein * factor
		totalCarb += p.carbs * factor
		totalFat += p.fat * factor
		time.Sleep(334 * time.Millisecond)
	}

	if successful == 0 {
		if len(recipe.Nutrition) > 0 {
			return recipe.Nutrition, "site", nil
		}
		return nil, "unavailable", nil
	}

	sv := float64(servings)
	out := map[string]string{
		"calories":            fmt.Sprintf("%d kcal", int(totalKcal/sv)),
		"proteinContent":      fmt.Sprintf("%.1f g", totalProt/sv),
		"carbohydrateContent": fmt.Sprintf("%.1f g", totalCarb/sv),
		"fatContent":          fmt.Sprintf("%.1f g", totalFat/sv),
	}
	return out, "usda-computed", nil
}

// fdcSearchResp is the subset of the FDC /foods/search response we need.
type fdcSearchResp struct {
	Foods []struct {
		Description   string `json:"description"`
		FoodNutrients []struct {
			NutrientID   int     `json:"nutrientId"`
			NutrientName string  `json:"nutrientName"`
			Value        float64 `json:"value"`
			UnitName     string  `json:"unitName"`
		} `json:"foodNutrients"`
	} `json:"foods"`
}

type per100gMacros struct {
	calories float64
	protein  float64
	carbs    float64
	fat      float64
}

func fdcLookupPer100g(ctx context.Context, client *http.Client, key, query string) (per100gMacros, bool) {
	endpoint := fmt.Sprintf(
		"https://api.nal.usda.gov/fdc/v1/foods/search?query=%s&pageSize=1&dataType=Foundation,SR%%20Legacy&api_key=%s",
		url.QueryEscape(query), url.QueryEscape(key),
	)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return per100gMacros{}, false
	}
	resp, err := client.Do(req)
	if err != nil {
		return per100gMacros{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return per100gMacros{}, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return per100gMacros{}, false
	}
	var parsed fdcSearchResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return per100gMacros{}, false
	}
	if len(parsed.Foods) == 0 {
		return per100gMacros{}, false
	}
	m := per100gMacros{}
	got := false
	for _, n := range parsed.Foods[0].FoodNutrients {
		switch n.NutrientID {
		case 1008: // Energy (kcal)
			m.calories = n.Value
			got = true
		case 1003: // Protein
			m.protein = n.Value
			got = true
		case 1005: // Carbohydrates
			m.carbs = n.Value
			got = true
		case 1004: // Total lipid (fat)
			m.fat = n.Value
			got = true
		}
		// Some entries only expose nutrientName, not id.
		nm := strings.ToLower(n.NutrientName)
		switch {
		case m.calories == 0 && strings.Contains(nm, "energy") && strings.EqualFold(n.UnitName, "KCAL"):
			m.calories = n.Value
			got = true
		case m.protein == 0 && nm == "protein":
			m.protein = n.Value
			got = true
		case m.carbs == 0 && strings.HasPrefix(nm, "carbohydrate"):
			m.carbs = n.Value
			got = true
		case m.fat == 0 && (nm == "total lipid (fat)" || nm == "fat"):
			m.fat = n.Value
			got = true
		}
	}
	return m, got
}

// ingredientStripRe peels off a leading quantity + unit so we can pass just
// the food name to the FDC search. Handles the same qty forms as the scaler,
// plus piece-style units (cloves, slices, strips) for grams lookup.
// Alternation order matters: RE2 uses leftmost-first, so longer/more-specific
// tokens must come before their shorter abbreviations (e.g. "cloves" before
// "c", "tablespoons" before "t"). A trailing word-boundary-ish guard (\b)
// keeps us from eating a single letter that belongs to the ingredient name.
var ingredientStripRe = regexp.MustCompile(`(?i)^\s*((?:\d+\s+\d+/\d+)|(?:\d+/\d+)|(?:\d*\.\d+)|(?:\d+))\s*(tablespoons?|teaspoons?|kilograms?|milliliters?|pounds?|ounces?|cloves?|slices?|strips?|stalks?|sprigs?|pieces?|liters?|grams?|cups?|tbsps?|tbsp\.?|tbs\.?|tsps?|tsp\.?|fluid\s+ounces?|fl\.?\s*oz\.?|lbs?\.?|oz\.?|kg|ml|c\.|t\.|T\.?|\bc\b|\bt\b|\bl\b|\bg\b)?\s+`)

// parseIngredientForFDC returns (foodName, grams). grams==0 means we couldn't
// derive a mass; caller should substitute a stub.
func parseIngredientForFDC(line string) (string, float64) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", 0
	}
	// Approximate grams from the leading quantity when possible.
	grams := 0.0
	m := ingredientStripRe.FindStringSubmatchIndex(line)
	name := line
	qtyOK := false
	var qty float64
	unitStr := ""
	if m != nil {
		qtyStr := ""
		if m[2] >= 0 {
			qtyStr = line[m[2]:m[3]]
		}
		if m[4] >= 0 {
			unitStr = strings.ToLower(strings.TrimSpace(line[m[4]:m[5]]))
		}
		if qtyStr != "" {
			if q, ok := parseQtyFloat(qtyStr); ok {
				qty = q
				qtyOK = true
			}
		}
		name = strings.TrimSpace(line[m[1]:])
	}
	// Strip common parenthetical notes, commas, "chopped", etc., and punctuation.
	if i := strings.Index(name, ","); i > 0 {
		name = name[:i]
	}
	if i := strings.Index(name, "("); i > 0 {
		name = name[:i]
	}
	name = strings.TrimSpace(name)
	// Drop leading "of " from "1 cup of flour" style phrasing.
	name = strings.TrimPrefix(strings.ToLower(name), "of ")
	// Truncate at common modifier keywords.
	for _, kw := range []string{" chopped", " diced", " minced", " sliced", " grated", " shredded", " melted", " softened", " crushed", " cubed", " peeled"} {
		if i := strings.Index(name, kw); i > 0 {
			name = name[:i]
			break
		}
	}
	name = strings.TrimSpace(name)

	// Canonicalize piece-style phrasing so the PiecesToGrams lookup hits.
	// "3 cloves garlic" (unit=cloves, name=garlic) → name="garlic clove"
	pieceCanonical := ""
	if unitStr != "" {
		uBase := strings.TrimSuffix(unitStr, "s")
		switch uBase {
		case "clove":
			pieceCanonical = name + " clove"
		case "slice":
			pieceCanonical = "slice " + name
		case "strip":
			pieceCanonical = name + " strip"
		case "stalk":
			pieceCanonical = "stalk " + name
		case "piece":
			pieceCanonical = name
		}
	}

	if qtyOK {
		grams = gramsForName(qty, unitStr, name, pieceCanonical)
	}

	// For the FDC search, prefer the short food name (e.g. "garlic" not "garlic clove").
	return name, grams
}

// gramsForName converts a qty+unit+canonical-name into grams using the lookup
// tables. Returns 0 when we can't estimate.
func gramsForName(qty float64, unit, name, pieceCanonical string) float64 {
	uBase := strings.TrimSuffix(strings.TrimSuffix(unit, "."), "s")
	switch {
	case unit == "":
		// No unit word — check piece table directly (e.g. "1 large egg").
		if g, ok := IngredientGramsPerPiece(name); ok {
			return qty * g
		}
		return 0
	case strings.HasPrefix(unit, "gram") || unit == "g":
		return qty
	case strings.HasPrefix(unit, "kilogram") || unit == "kg":
		return qty * 1000
	case strings.HasPrefix(unit, "pound") || strings.HasPrefix(unit, "lb"):
		return qty * 454
	case strings.Contains(unit, "fl") && strings.Contains(unit, "oz"):
		return qty * 30
	case strings.HasPrefix(unit, "ounce") || unit == "oz" || unit == "oz.":
		return qty * 28.35
	case strings.HasPrefix(unit, "cup") || unit == "c" || unit == "c.":
		if g, ok := IngredientGramsPerCup(name); ok {
			return qty * g
		}
		return qty * 240 // water-density fallback
	case strings.HasPrefix(unit, "tablespoon") || strings.HasPrefix(unit, "tbsp") || strings.HasPrefix(unit, "tbs"):
		if g, ok := IngredientGramsPerTbsp(name); ok {
			return qty * g
		}
		return qty * 15
	case strings.HasPrefix(unit, "teaspoon") || strings.HasPrefix(unit, "tsp") || unit == "t" || unit == "t.":
		if g, ok := IngredientGramsPerTbsp(name); ok {
			return qty * (g / 3.0)
		}
		return qty * 5
	case strings.HasPrefix(unit, "milliliter") || unit == "ml":
		return qty
	case strings.HasPrefix(unit, "liter") || unit == "l":
		return qty * 1000
	case uBase == "clove" || uBase == "slice" || uBase == "strip" || uBase == "stalk" || uBase == "sprig" || uBase == "piece":
		if pieceCanonical != "" {
			if g, ok := IngredientGramsPerPiece(pieceCanonical); ok {
				return qty * g
			}
		}
		if g, ok := IngredientGramsPerPiece(name); ok {
			return qty * g
		}
		return qty * 80
	}
	// Unknown unit — fall back to piece lookup or a generic 80g.
	if g, ok := IngredientGramsPerPiece(name); ok {
		return qty * g
	}
	return qty * 80
}
