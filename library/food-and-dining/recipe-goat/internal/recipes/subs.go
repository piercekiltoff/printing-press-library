package recipes

import (
	"regexp"
	"strings"
)

// Sub is one suggested substitution for an ingredient.
type Sub struct {
	Ingredient string  `json:"ingredient"`
	Substitute string  `json:"substitute"`
	Ratio      string  `json:"ratio"`
	Context    string  `json:"context"` // "baking" | "marinade" | "sauce" | "any"
	Source     string  `json:"source"`
	Trust      float64 `json:"trust"`
}

// subTable is a curated list of common substitutions. Sources are recognizable
// baking/cooking references; trust is loosely calibrated to how confident the
// substitution is across the linked use-case.
var subTable = []Sub{
	// Dairy
	{Ingredient: "buttermilk", Substitute: "milk + lemon juice", Ratio: "1 cup = 1 cup milk + 1 Tbsp lemon juice", Context: "baking", Source: "King Arthur", Trust: 0.95},
	{Ingredient: "buttermilk", Substitute: "milk + vinegar", Ratio: "1 cup = 1 cup milk + 1 Tbsp white vinegar", Context: "baking", Source: "King Arthur", Trust: 0.95},
	{Ingredient: "buttermilk", Substitute: "plain yogurt (thinned)", Ratio: "1 cup = 3/4 cup yogurt + 1/4 cup milk", Context: "any", Source: "Serious Eats", Trust: 0.9},
	{Ingredient: "heavy cream", Substitute: "milk + butter", Ratio: "1 cup = 3/4 cup milk + 1/4 cup melted butter", Context: "baking", Source: "King Arthur", Trust: 0.85},
	{Ingredient: "half-and-half", Substitute: "milk + cream", Ratio: "1 cup = 3/4 cup milk + 1/4 cup heavy cream", Context: "any", Source: "AR community", Trust: 0.9},
	{Ingredient: "sour cream", Substitute: "Greek yogurt", Ratio: "1:1", Context: "any", Source: "Serious Eats", Trust: 0.95},
	{Ingredient: "sour cream", Substitute: "buttermilk + butter", Ratio: "1 cup = 3/4 cup buttermilk + 1/3 cup butter", Context: "baking", Source: "King Arthur", Trust: 0.8},
	{Ingredient: "milk", Substitute: "evaporated milk + water", Ratio: "1 cup = 1/2 cup evap + 1/2 cup water", Context: "any", Source: "AR community", Trust: 0.9},
	{Ingredient: "milk", Substitute: "oat milk", Ratio: "1:1", Context: "baking", Source: "Minimalist Baker", Trust: 0.9},
	{Ingredient: "yogurt", Substitute: "sour cream", Ratio: "1:1", Context: "any", Source: "Budget Bytes", Trust: 0.9},
	{Ingredient: "ricotta", Substitute: "cottage cheese (blended)", Ratio: "1:1", Context: "any", Source: "Serious Eats", Trust: 0.85},
	{Ingredient: "mozzarella", Substitute: "provolone", Ratio: "1:1", Context: "any", Source: "AR community", Trust: 0.85},
	{Ingredient: "parmesan", Substitute: "pecorino romano", Ratio: "1:1 (saltier)", Context: "any", Source: "Serious Eats", Trust: 0.9},

	// Fats
	{Ingredient: "butter", Substitute: "oil", Ratio: "1 cup butter = 3/4 cup oil", Context: "baking", Source: "King Arthur", Trust: 0.8},
	{Ingredient: "butter", Substitute: "coconut oil", Ratio: "1:1", Context: "baking", Source: "Minimalist Baker", Trust: 0.85},
	{Ingredient: "olive oil", Substitute: "avocado oil", Ratio: "1:1", Context: "any", Source: "Serious Eats", Trust: 0.95},

	// Sugar
	{Ingredient: "sugar", Substitute: "honey", Ratio: "1 cup sugar = 3/4 cup honey; reduce liquid by 1/4 cup", Context: "baking", Source: "King Arthur", Trust: 0.85},
	{Ingredient: "brown sugar", Substitute: "white sugar + molasses", Ratio: "1 cup = 1 cup sugar + 1 Tbsp molasses", Context: "baking", Source: "King Arthur", Trust: 0.95},
	{Ingredient: "powdered sugar", Substitute: "granulated sugar (blended)", Ratio: "1 cup = 1 cup sugar + 1 Tbsp cornstarch, blended", Context: "baking", Source: "King Arthur", Trust: 0.85},

	// Leavening
	{Ingredient: "baking powder", Substitute: "baking soda + cream of tartar", Ratio: "1 tsp = 1/4 tsp baking soda + 1/2 tsp cream of tartar", Context: "baking", Source: "King Arthur", Trust: 0.95},
	{Ingredient: "baking soda", Substitute: "baking powder (triple)", Ratio: "1 tsp baking soda = 3 tsp baking powder", Context: "baking", Source: "Serious Eats", Trust: 0.85},
	{Ingredient: "yeast", Substitute: "active dry ↔ instant", Ratio: "instant = 0.75x of active dry", Context: "baking", Source: "King Arthur", Trust: 0.95},

	// Eggs
	{Ingredient: "eggs", Substitute: "flax egg", Ratio: "1 egg = 1 Tbsp ground flax + 3 Tbsp water", Context: "baking", Source: "Minimalist Baker", Trust: 0.9},
	{Ingredient: "eggs", Substitute: "applesauce", Ratio: "1 egg = 1/4 cup unsweetened applesauce", Context: "baking", Source: "AR community", Trust: 0.8},

	// Flour
	{Ingredient: "flour", Substitute: "gluten-free 1:1 blend", Ratio: "1:1", Context: "baking", Source: "King Arthur", Trust: 0.85},
	{Ingredient: "flour", Substitute: "cake flour (thicken)", Ratio: "1 cup AP = 1 cup cake flour - 2 Tbsp + 2 Tbsp cornstarch", Context: "baking", Source: "King Arthur", Trust: 0.9},

	// Pantry / seasonings
	{Ingredient: "vanilla", Substitute: "vanilla bean paste", Ratio: "1:1", Context: "baking", Source: "King Arthur", Trust: 0.95},
	{Ingredient: "cornstarch", Substitute: "arrowroot", Ratio: "1:1", Context: "sauce", Source: "Serious Eats", Trust: 0.9},
	{Ingredient: "cornstarch", Substitute: "flour", Ratio: "1 Tbsp cornstarch = 2 Tbsp flour", Context: "sauce", Source: "AR community", Trust: 0.85},
	{Ingredient: "lemon juice", Substitute: "white vinegar", Ratio: "1:1 (flavor differs)", Context: "any", Source: "Serious Eats", Trust: 0.85},
	{Ingredient: "vinegar", Substitute: "lemon juice", Ratio: "1:1", Context: "any", Source: "Serious Eats", Trust: 0.85},
	{Ingredient: "wine", Substitute: "broth + vinegar", Ratio: "1 cup wine = 1 cup broth + 1 Tbsp vinegar", Context: "sauce", Source: "Serious Eats", Trust: 0.85},
	{Ingredient: "wine", Substitute: "grape juice + vinegar", Ratio: "1 cup = 1 cup juice + 1 Tbsp vinegar", Context: "marinade", Source: "AR community", Trust: 0.8},
	{Ingredient: "soy sauce", Substitute: "tamari", Ratio: "1:1 (GF)", Context: "any", Source: "Serious Eats", Trust: 0.95},
	{Ingredient: "soy sauce", Substitute: "coconut aminos", Ratio: "1:1 (sweeter)", Context: "marinade", Source: "Minimalist Baker", Trust: 0.85},
	{Ingredient: "chicken broth", Substitute: "vegetable broth", Ratio: "1:1", Context: "any", Source: "Budget Bytes", Trust: 0.95},
	{Ingredient: "beef broth", Substitute: "mushroom broth", Ratio: "1:1", Context: "any", Source: "Serious Eats", Trust: 0.9},
	{Ingredient: "garlic", Substitute: "garlic powder", Ratio: "1 clove = 1/8 tsp powder", Context: "any", Source: "AR community", Trust: 0.8},
	{Ingredient: "onion", Substitute: "onion powder", Ratio: "1 medium onion = 1 Tbsp powder", Context: "any", Source: "AR community", Trust: 0.8},
	{Ingredient: "honey", Substitute: "maple syrup", Ratio: "1:1", Context: "baking", Source: "Minimalist Baker", Trust: 0.9},
}

// LookupSubs returns substitutions for `ingredient` whose Context matches
// `contextFilter` (case-insensitive) or is "any". Pass "" or "any" to return
// all contexts.
//
// Matching is word-boundary aware: table ingredients are matched as whole
// tokens against the user's query, so "butter" will not match "buttermilk"
// (a previous bug). The needle itself is still allowed to be a phrase like
// "2 cups buttermilk" — we tokenize it and look for the table ingredient as a
// contiguous token sequence.
func LookupSubs(ingredient string, contextFilter string) []Sub {
	needle := strings.ToLower(strings.TrimSpace(ingredient))
	if needle == "" {
		return nil
	}
	cf := strings.ToLower(strings.TrimSpace(contextFilter))
	out := []Sub{}
	for _, s := range subTable {
		if !ingredientMatches(needle, strings.ToLower(s.Ingredient)) {
			continue
		}
		if cf != "" && cf != "any" && s.Context != "any" && s.Context != cf {
			continue
		}
		out = append(out, s)
	}
	return out
}

// ingredientTokenRe splits on non-alphanumerics so "buttermilk" stays one
// token and "brown sugar" becomes two.
var ingredientTokenRe = regexp.MustCompile(`[a-z0-9]+`)

// ingredientMatches returns true when the table ingredient `target` appears
// as a contiguous sequence of whole tokens inside the user's query `needle`.
// Both inputs are expected already lowercased.
func ingredientMatches(needle, target string) bool {
	needleToks := ingredientTokenRe.FindAllString(needle, -1)
	targetToks := ingredientTokenRe.FindAllString(target, -1)
	if len(targetToks) == 0 || len(needleToks) < len(targetToks) {
		return false
	}
	for i := 0; i <= len(needleToks)-len(targetToks); i++ {
		match := true
		for j, t := range targetToks {
			if needleToks[i+j] != t {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// AllSubs returns the built-in sub table (used by 'sub --list' or similar).
func AllSubs() []Sub {
	out := make([]Sub, len(subTable))
	copy(out, subTable)
	return out
}
