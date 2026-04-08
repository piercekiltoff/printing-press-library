package espn

import (
	"fmt"
	"strings"
)

var DefaultLeagues = map[string]struct{ Sport, League string }{
	"nfl":   {"football", "nfl"},
	"nba":   {"basketball", "nba"},
	"mlb":   {"baseball", "mlb"},
	"nhl":   {"hockey", "nhl"},
	"ncaaf": {"football", "college-football"},
	"ncaam": {"basketball", "mens-college-basketball"},
	"ncaaw": {"basketball", "womens-college-basketball"},
	"mls":   {"soccer", "usa.1"},
	"epl":   {"soccer", "eng.1"},
	"wnba":  {"basketball", "wnba"},
}

func ResolveSportLeague(input string) (sport, league string, err error) {
	key := strings.ToLower(strings.TrimSpace(input))
	spec, ok := DefaultLeagues[key]
	if !ok {
		return "", "", fmt.Errorf("unsupported league %q", input)
	}
	return spec.Sport, spec.League, nil
}
