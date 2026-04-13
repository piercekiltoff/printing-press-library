package recipes

import (
	"net/url"
	"regexp"
	"strings"
)

// Site describes a recipe source we search against.
type Site struct {
	Name      string
	Hostname  string
	Tier      int
	SearchURL string  // template with {q}
	Trust     float64 // 0..1

	// RecipeURLPattern, if non-nil, is matched against the *path portion* of
	// candidate URLs when deciding whether a link looks like a recipe permalink.
	// Populated in init() below — sourced from each site's real URL structure.
	RecipeURLPattern *regexp.Regexp
}

// Sites is the built-in registry of recipe sources.
var Sites = []Site{
	// Tier 1 (trust 0.9)
	{Name: "King Arthur Baking", Hostname: "kingarthurbaking.com", Tier: 1, SearchURL: "https://www.kingarthurbaking.com/search?f%5B0%5D=content_type%3Arecipe&q={q}", Trust: 0.9},
	{Name: "Budget Bytes", Hostname: "budgetbytes.com", Tier: 1, SearchURL: "https://www.budgetbytes.com/?s={q}", Trust: 0.9},
	{Name: "Smitten Kitchen", Hostname: "smittenkitchen.com", Tier: 1, SearchURL: "https://smittenkitchen.com/?s={q}", Trust: 0.9},
	{Name: "Food52", Hostname: "food52.com", Tier: 1, SearchURL: "https://food52.com/recipes/search?q={q}", Trust: 0.9},
	{Name: "BBC Good Food", Hostname: "bbcgoodfood.com", Tier: 1, SearchURL: "https://www.bbcgoodfood.com/search?q={q}", Trust: 0.9},
	{Name: "Minimalist Baker", Hostname: "minimalistbaker.com", Tier: 1, SearchURL: "https://minimalistbaker.com/?s={q}", Trust: 0.9},
	{Name: "Skinnytaste", Hostname: "skinnytaste.com", Tier: 1, SearchURL: "https://www.skinnytaste.com/?s={q}", Trust: 0.9},
	{Name: "The Kitchn", Hostname: "thekitchn.com", Tier: 1, SearchURL: "https://www.thekitchn.com/search?q={q}", Trust: 0.9},
	{Name: "Food Network", Hostname: "foodnetwork.com", Tier: 1, SearchURL: "https://www.foodnetwork.com/search/{q}-", Trust: 0.9},

	// Tier 2 (trust 0.8)
	{Name: "Bon Appétit", Hostname: "bonappetit.com", Tier: 2, SearchURL: "https://www.bonappetit.com/search?q={q}", Trust: 0.8},
	{Name: "Epicurious", Hostname: "epicurious.com", Tier: 2, SearchURL: "https://www.epicurious.com/search/{q}", Trust: 0.8},

	// Tier 3 (trust 0.95 content, low reachability)
	{Name: "Allrecipes", Hostname: "allrecipes.com", Tier: 3, SearchURL: "https://www.allrecipes.com/search?q={q}", Trust: 0.95},
	{Name: "Simply Recipes", Hostname: "simplyrecipes.com", Tier: 3, SearchURL: "https://www.simplyrecipes.com/search?q={q}", Trust: 0.95},
	{Name: "EatingWell", Hostname: "eatingwell.com", Tier: 3, SearchURL: "https://www.eatingwell.com/search?q={q}", Trust: 0.95},
	{Name: "Serious Eats", Hostname: "seriouseats.com", Tier: 3, SearchURL: "https://www.seriouseats.com/search?q={q}", Trust: 0.95},
}

// recipeURLPatterns is a lookup of compiled per-host regexes for candidate
// recipe-permalink paths. The Site.RecipeURLPattern field is populated from
// this map at init() time. Keys are the Site.Hostname (www-stripped).
var recipeURLPatterns = map[string]string{
	"kingarthurbaking.com": `^/recipes/[a-z0-9-]+-recipe$`,
	"budgetbytes.com":      `^/[a-z0-9-]{6,}/?$`,
	"smittenkitchen.com":   `^/\d{4}/\d{2}/[a-z0-9-]+/?$`,
	"food52.com":           `^/recipes/\d+-[a-z0-9-]+$`,
	"bbcgoodfood.com":      `^/recipes/[a-z0-9-]+$`,
	"minimalistbaker.com":  `^/[a-z0-9-]{6,}/?$`,
	"skinnytaste.com":      `^/[a-z0-9-]{6,}/?$`,
	"thekitchn.com":        `^/(?:.*-recipe-\d+|recipe-[a-z0-9-]+)$`,
	"foodnetwork.com":      `^/recipes/[a-z0-9-/]+-recipe-\d+$`,
	"bonappetit.com":       `^/recipe/[a-z0-9-]+$`,
	"epicurious.com":       `^/recipes/food/views/[a-z0-9-]+$`,
	"allrecipes.com":       `^/recipe/\d+/[a-z0-9-]+/?$`,
	"simplyrecipes.com":    `^/(?:recipes/[a-z0-9_-]+/?|[a-z0-9-]+-recipe-\d+)$`,
	"eatingwell.com":       `^/recipe/\d+/[a-z0-9-]+/?$`,
	"seriouseats.com":      `^/[a-z0-9-]+-recipe(-\d+)?$`,
}

func init() {
	// Compile patterns once and back-fill onto the Sites entries.
	compiled := make(map[string]*regexp.Regexp, len(recipeURLPatterns))
	for host, pat := range recipeURLPatterns {
		compiled[host] = regexp.MustCompile(pat)
	}
	for i := range Sites {
		if re, ok := compiled[Sites[i].Hostname]; ok {
			Sites[i].RecipeURLPattern = re
		}
	}
}

// FindSite returns the Site matching the given hostname (www.-stripped), or
// an empty Site with Trust=0.5 when we don't recognize the host.
func FindSite(host string) Site {
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	for _, s := range Sites {
		if s.Hostname == host {
			return s
		}
	}
	return Site{Hostname: host, Tier: 3, Trust: 0.5}
}

// siteByHost returns a pointer to the registered Site for the given host, or
// nil if the host is unknown. Kept separate from FindSite because callers that
// need the compiled regex benefit from sharing the struct verbatim.
func siteByHost(host string) *Site {
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	for i := range Sites {
		if Sites[i].Hostname == host {
			return &Sites[i]
		}
	}
	return nil
}

// BuildSearchURL applies url.QueryEscape to the query and substitutes into
// the site's SearchURL template. {q} is the only recognized placeholder.
func BuildSearchURL(site Site, query string) string {
	r := strings.NewReplacer("{q}", url.QueryEscape(query))
	return r.Replace(site.SearchURL)
}

// curatedAuthors is a lowercase set of hand-picked recipe authors we trust
// highly. Names are stored lowercased; lookup is case-insensitive.
var curatedAuthors = map[string]bool{
	"kenji lópez-alt":            true,
	"kenji lopez-alt":            true,
	"j. kenji lópez-alt":         true,
	"j. kenji lopez-alt":         true,
	"stella parks":               true,
	"deb perelman":               true,
	"samin nosrat":               true,
	"ina garten":                 true,
	"alton brown":                true,
	"claire saffitz":             true,
	"molly baz":                  true,
	"carla lalli music":          true,
	"budget bytes team":          true,
	"beth moncel":                true,
	"king arthur baking company": true,
	"brian lagerstrom":           true,
}

// AuthorTrust returns a trust score in [0,1] for the given author. Curated
// authors get 1.0; everyone else gets 0.5.
func AuthorTrust(author string) float64 {
	a := strings.ToLower(strings.TrimSpace(author))
	if a == "" {
		return 0.5
	}
	if curatedAuthors[a] {
		return 1.0
	}
	return 0.5
}

// CuratedAuthors returns the curated list (for 'trust list' output). Names are
// returned lowercased — callers can title-case them for display.
func CuratedAuthors() []string {
	out := make([]string, 0, len(curatedAuthors))
	for a := range curatedAuthors {
		out = append(out, a)
	}
	return out
}
