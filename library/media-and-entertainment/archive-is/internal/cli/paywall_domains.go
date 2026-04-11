// Hard-paywall domain detection for archive-is-pp-cli.
//
// Wayback Machine captures pages AFTER their JavaScript has run, so for hard
// paywalls (WSJ, NYT, FT, Bloomberg, Economist) the Wayback snapshot often
// contains only the paywall teaser — the first few paragraphs that render
// before the JS overlay fires. Archive.today captures pages BEFORE the JS runs,
// which is why it bypasses these paywalls in practice.
//
// This file maintains a list of known hard-paywall domains. When the CLI falls
// back to Wayback for a URL on one of these domains, it prints a warning
// suggesting `save <url>` to force a fresh archive.today capture.
//
// Adding a new domain: append to hardPaywallDomains and rebuild. The match is
// suffix-based so `wsj.com` matches `www.wsj.com`, `markets.wsj.com`, etc.

package cli

import (
	"net/url"
	"strings"
)

// hardPaywallDomains is the curated list of sites whose Wayback snapshots are
// known to be teaser-only because the paywall runs as JavaScript. These are
// sites where archive.today is meaningfully better at paywall bypass.
//
// Grouped by category to make additions easier. Order within a group is
// alphabetical.
var hardPaywallDomains = map[string]bool{
	// Newspapers
	"wsj.com":            true,
	"nytimes.com":        true,
	"washingtonpost.com": true,
	"wapo.st":            true,
	"ft.com":             true,
	"latimes.com":        true,
	"bostonglobe.com":    true,
	"thetimes.co.uk":     true,
	"telegraph.co.uk":    true,
	"seattletimes.com":   true,
	"sfchronicle.com":    true,
	"chicagotribune.com": true,

	// Business & finance
	"bloomberg.com":       true,
	"barrons.com":         true,
	"marketwatch.com":     true,
	"businessinsider.com": true,
	"forbes.com":          true,
	"economist.com":       true,

	// Magazines & long-form
	"theatlantic.com":    true,
	"newyorker.com":      true,
	"wired.com":          true,
	"foreignaffairs.com": true,
	"foreignpolicy.com":  true,
	"harpers.org":        true,
	"nybooks.com":        true,
	"newstatesman.com":   true,
	"theintercept.com":   true,

	// Trade & research
	"hbr.org":                true,
	"nature.com":             true,
	"science.org":            true,
	"thelancet.com":          true,
	"scientificamerican.com": true,
}

// isHardPaywallDomain reports whether the given URL is on a known hard-paywall
// domain. Matches on the registerable host name so www.wsj.com, markets.wsj.com,
// and wsj.com all return true. Returns false on malformed URLs.
func isHardPaywallDomain(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Host)
	// Strip port if present
	if i := strings.Index(host, ":"); i != -1 {
		host = host[:i]
	}
	// Check exact match first, then walk up the domain hierarchy.
	if hardPaywallDomains[host] {
		return true
	}
	// Match suffixes on label boundaries: for www.wsj.com, try wsj.com.
	// Split on "." and check progressively shorter suffixes.
	parts := strings.Split(host, ".")
	for i := 1; i < len(parts)-1; i++ {
		candidate := strings.Join(parts[i:], ".")
		if hardPaywallDomains[candidate] {
			return true
		}
	}
	return false
}

// paywallWarning returns the stderr warning text for a hard-paywall domain.
// The warning suggests `save` as the next step.
func paywallWarning(rawURL string) string {
	u, err := url.Parse(rawURL)
	host := "this site"
	if err == nil && u.Host != "" {
		host = u.Host
	}
	return "\nNote: Wayback snapshots of " + host + " articles usually show only the paywall teaser.\n" +
		"For full article text, try:\n" +
		"  archive-is-pp-cli save " + rawURL + "\n" +
		"This forces a fresh archive.today capture which strips JavaScript and bypasses the paywall overlay.\n"
}
