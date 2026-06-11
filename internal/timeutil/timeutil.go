// Package timeutil provides shared time formatting helpers.
package timeutil

import (
	"fmt"
	"time"
)

// RelativeTime formats a time.Time as a human-readable relative duration
// using full English words (e.g. "5 minutes ago", "1 week ago").
// Returns "never" for the zero time.
func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	d := time.Since(t)
	if d < 0 {
		return "in the future"
	}

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case d < 30*24*time.Hour:
		weeks := int(d.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		months := int(d.Hours() / (24 * 30))
		if months == 0 {
			months = 1
		}
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}

// RelativeTimeShort formats a time.Time as a compact relative duration
// using abbreviations (e.g. "5 min ago", "1h ago", "3d ago").
func RelativeTimeShort(t time.Time) string {
	now := time.Now().UTC()
	d := now.Sub(t)

	if d < 0 {
		return "in the future"
	}

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		months := int(d.Hours() / (24 * 30))
		if months == 0 {
			months = 1
		}
		if months == 1 {
			return "1mo ago"
		}
		return fmt.Sprintf("%dmo ago", months)
	}
}
