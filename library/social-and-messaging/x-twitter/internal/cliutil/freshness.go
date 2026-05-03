package cliutil

import (
	"os"
	"time"
)

// EnsureFresh returns (isStale, age, err) for the file at path. A file is
// considered stale when its mtime is older than maxAge. Missing file = stale.
// Used by the auto-refresh hook to decide whether to nudge the user to
// re-sync before running a read command.
func EnsureFresh(path string, maxAge time.Duration) (bool, time.Duration, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, 0, nil
		}
		return false, 0, err
	}
	age := time.Since(info.ModTime())
	return age > maxAge, age, nil
}

// FormatAge returns a short human-friendly age (e.g. "2h ago", "3d ago").
func FormatAge(age time.Duration) string {
	switch {
	case age < time.Minute:
		return "just now"
	case age < time.Hour:
		return age.Truncate(time.Minute).String() + " ago"
	case age < 24*time.Hour:
		return age.Truncate(time.Hour).String() + " ago"
	default:
		days := int(age / (24 * time.Hour))
		if days == 1 {
			return "1 day ago"
		}
		return formatDays(days) + " days ago"
	}
}

func formatDays(d int) string {
	// Avoid strconv import for one call site.
	if d < 10 {
		return string(rune('0' + d))
	}
	// Two digits: assume <100 days reasonable
	tens := d / 10
	ones := d % 10
	return string([]rune{rune('0' + tens), rune('0' + ones)})
}
