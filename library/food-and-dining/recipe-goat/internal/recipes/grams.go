package recipes

import "strings"

// CupToGrams maps canonical ingredient names (lowercased, singular) to grams
// per 1 cup (US customary). For ingredients counted in pieces, see
// PiecesToGrams.
var CupToGrams = map[string]float64{
	"all-purpose flour":   120,
	"flour":               120,
	"whole wheat flour":   113,
	"bread flour":         120,
	"cake flour":          114,
	"sugar":               200,
	"granulated sugar":    200,
	"brown sugar":         220,
	"powdered sugar":      120,
	"confectioners sugar": 120,
	"rice":                185,
	"white rice":          185,
	"brown rice":          190,
	"basmati rice":        185,
	"oats":                90,
	"rolled oats":         90,
	"quinoa":              170,
	"milk":                245,
	"whole milk":          245,
	"skim milk":           245,
	"2% milk":             245,
	"buttermilk":          240,
	"cream":               240,
	"heavy cream":         240,
	"half and half":       240,
	"yogurt":              245,
	"greek yogurt":        245,
	"sour cream":          230,
	"water":               240,
	"chicken broth":       240,
	"beef broth":          240,
	"vegetable broth":     240,
	"chicken stock":       240,
	"olive oil":           216,
	"vegetable oil":       218,
	"canola oil":          218,
	"coconut oil":         218,
	"melted butter":       227,
	"honey":               340,
	"maple syrup":         322,
	"molasses":            337,
	"peanut butter":       258,
	"almond butter":       258,
	"mayonnaise":          220,
	"ketchup":             240,
	"soy sauce":           255,
	"tomato sauce":        245,
	"tomato paste":        262,
	"salsa":               258,
	"cheese":              113, // grated/shredded
	"shredded cheese":     113,
	"cheddar":             113,
	"parmesan":            100, // grated
	"mozzarella":          113,
	"cottage cheese":      225,
	"ricotta":             250,
	"chopped onion":       160,
	"diced onion":         160,
	"chopped tomato":      180,
	"diced tomato":        180,
	"cherry tomatoes":     150,
	"sliced mushrooms":    70,
	"mushrooms":           96,
	"chopped bell pepper": 150,
	"frozen peas":         134,
	"frozen corn":         165,
	"cooked pasta":        140,
	"dried pasta":         100,
	"cooked rice":         158,
	"cocoa powder":        85,
	"chocolate chips":     170,
	"raisins":             150,
	"almonds":             143,
	"chopped nuts":        120,
	"walnuts":             120,
	"pecans":              110,
	"coconut flakes":      60,
	"breadcrumbs":         108,
	"panko":               56,
	"salt":                273,
	"kosher salt":         215,
	"pepper":              115,
	"cornstarch":          120,
	"baking powder":       240,
	"baking soda":         220,
	"yeast":               136,
	"cinnamon":            108,
	"vanilla extract":     208,
	"vinegar":             240,
	"white vinegar":       240,
	"apple cider vinegar": 240,
	"balsamic vinegar":    240,
	"lemon juice":         240,
	"lime juice":          240,
	"orange juice":        248,
	"wine":                240,
	"white wine":          240,
	"red wine":            240,
	"beer":                240,
}

// TbspToGrams overrides CupToGrams/16 for ingredients where the tablespoon
// conversion diverges meaningfully from simple division. Missing keys fall
// back to CupToGrams[x] / 16 via IngredientGramsPerTbsp.
var TbspToGrams = map[string]float64{
	"butter":        14,
	"olive oil":     13.5,
	"vegetable oil": 14,
	"coconut oil":   14,
	"honey":         21,
	"maple syrup":   20,
	"sugar":         12.5,
	"brown sugar":   13.75,
	"flour":         7.5,
	"salt":          18,
	"kosher salt":   13,
	"soy sauce":     16,
	"cornstarch":    7.5,
	"baking powder": 15,
	"cocoa powder":  5.3,
}

// PiecesToGrams handles ingredients counted in pieces.
var PiecesToGrams = map[string]float64{
	"egg":                     50,
	"large egg":               50,
	"extra large egg":         56,
	"medium egg":              44,
	"small egg":               38,
	"apple":                   182,
	"medium apple":            182,
	"large apple":             223,
	"banana":                  118,
	"medium banana":           118,
	"large banana":            136,
	"lemon":                   65,
	"lime":                    67,
	"orange":                  131,
	"onion":                   110,
	"small onion":             70,
	"medium onion":            110,
	"large onion":             150,
	"garlic clove":            3,
	"clove garlic":            3,
	"cloves garlic":           3,
	"potato":                  213,
	"medium potato":           213,
	"tomato":                  123,
	"medium tomato":           123,
	"bell pepper":             119,
	"carrot":                  61,
	"medium carrot":           61,
	"celery stalk":            40,
	"stalk celery":            40,
	"zucchini":                196,
	"cucumber":                301,
	"avocado":                 150,
	"chicken breast":          174,
	"boneless chicken breast": 174,
	"chicken thigh":           94,
	"boneless chicken thigh":  94,
	"bacon strip":             28,
	"slice bacon":             28,
	"strip bacon":             28,
	"sausage":                 75,
	"bun":                     55,
	"slice bread":             28,
	"tortilla":                40,
	"flour tortilla":          40,
	"corn tortilla":           24,
	"pita":                    60,
	"bagel":                   105,
	"english muffin":          58,
	"slice cheese":            21,
	"string cheese":           28,
}

// leading descriptors to strip when normalizing for lookup.
var gramsLookupDescriptors = []string{
	"fresh", "dried", "cooked", "raw", "chopped", "diced", "sliced",
	"minced", "grated", "shredded", "crushed", "ground", "peeled",
	"finely", "coarsely", "fine", "coarse",
}

// normalizeGramsName lowercases and strips leading descriptors.
func normalizeGramsName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	for {
		trimmed := n
		for _, d := range gramsLookupDescriptors {
			if strings.HasPrefix(trimmed, d+" ") {
				trimmed = strings.TrimSpace(trimmed[len(d):])
			}
		}
		if trimmed == n {
			break
		}
		n = trimmed
	}
	return n
}

// trimPlural strips a trailing "s" unless the word ends in "ss".
func trimPlural(s string) string {
	if len(s) >= 2 && strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss") {
		return s[:len(s)-1]
	}
	return s
}

// lookupInTable tries exact, plural-trimmed, and longest-substring matches.
func lookupInTable(name string, table map[string]float64) (float64, bool) {
	n := normalizeGramsName(name)
	if n == "" {
		return 0, false
	}
	if v, ok := table[n]; ok {
		return v, true
	}
	// Try trimmed plural on the whole name.
	if tp := trimPlural(n); tp != n {
		if v, ok := table[tp]; ok {
			return v, true
		}
	}
	// Longest-substring match: iterate all keys, pick longest key that
	// appears as a whole-word substring of the normalized name, or where
	// the name is a substring of the key (rarer, for e.g. "egg" → "large egg").
	var bestKey string
	var bestVal float64
	for k, v := range table {
		if len(k) <= len(bestKey) {
			continue
		}
		if strings.Contains(n, k) || strings.Contains(k, n) {
			bestKey = k
			bestVal = v
		}
	}
	if bestKey != "" {
		return bestVal, true
	}
	return 0, false
}

// IngredientGramsPerCup returns grams-per-cup for a parsed ingredient name.
// Returns (grams, true) on match, (0, false) otherwise.
func IngredientGramsPerCup(name string) (float64, bool) {
	return lookupInTable(name, CupToGrams)
}

// IngredientGramsPerTbsp returns grams-per-tablespoon. If the ingredient is
// not in TbspToGrams but is in CupToGrams, returns cupValue/16.
func IngredientGramsPerTbsp(name string) (float64, bool) {
	if v, ok := lookupInTable(name, TbspToGrams); ok {
		return v, true
	}
	if v, ok := lookupInTable(name, CupToGrams); ok {
		return v / 16.0, true
	}
	return 0, false
}

// IngredientGramsPerPiece looks up piece-based ingredients.
func IngredientGramsPerPiece(name string) (float64, bool) {
	return lookupInTable(name, PiecesToGrams)
}
