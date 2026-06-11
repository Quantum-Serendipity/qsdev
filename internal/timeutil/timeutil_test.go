package timeutil

import (
	"strings"
	"testing"
	"time"
)

func TestRelativeTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		when time.Time
		want string
	}{
		{"zero value", time.Time{}, "never"},
		{"just now", time.Now().Add(-10 * time.Second), "just now"},
		{"1 minute ago", time.Now().Add(-1 * time.Minute), "1 minute ago"},
		{"5 minutes ago", time.Now().Add(-5 * time.Minute), "5 minutes ago"},
		{"1 hour ago", time.Now().Add(-1 * time.Hour), "1 hour ago"},
		{"3 hours ago", time.Now().Add(-3 * time.Hour), "3 hours ago"},
		{"1 day ago", time.Now().Add(-25 * time.Hour), "1 day ago"},
		{"4 days ago", time.Now().Add(-4 * 24 * time.Hour), "4 days ago"},
		{"1 week ago", time.Now().Add(-8 * 24 * time.Hour), "1 week ago"},
		{"3 weeks ago", time.Now().Add(-21 * 24 * time.Hour), "3 weeks ago"},
		{"1 month ago", time.Now().Add(-31 * 24 * time.Hour), "1 month ago"},
		{"6 months ago", time.Now().Add(-180 * 24 * time.Hour), "6 months ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RelativeTime(tt.when)
			if got != tt.want {
				t.Errorf("RelativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRelativeTimeFuture(t *testing.T) {
	t.Parallel()

	got := RelativeTime(time.Now().Add(1 * time.Hour))
	if got != "in the future" {
		t.Errorf("RelativeTime(future) = %q, want %q", got, "in the future")
	}
}

func TestRelativeTimeShort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		when     time.Time
		contains string
	}{
		{"just now", time.Now().UTC().Add(-10 * time.Second), "just now"},
		{"minutes", time.Now().UTC().Add(-5 * time.Minute), "min ago"},
		{"hours", time.Now().UTC().Add(-3 * time.Hour), "h ago"},
		{"days", time.Now().UTC().Add(-4 * 24 * time.Hour), "d ago"},
		{"months", time.Now().UTC().Add(-60 * 24 * time.Hour), "mo ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RelativeTimeShort(tt.when)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("RelativeTimeShort() = %q, want to contain %q", got, tt.contains)
			}
		})
	}
}

func TestRelativeTimeShortFuture(t *testing.T) {
	t.Parallel()

	got := RelativeTimeShort(time.Now().UTC().Add(1 * time.Hour))
	if got != "in the future" {
		t.Errorf("RelativeTimeShort(future) = %q, want %q", got, "in the future")
	}
}

func TestRelativeTimeShort1Minute(t *testing.T) {
	t.Parallel()

	got := RelativeTimeShort(time.Now().UTC().Add(-61 * time.Second))
	if got != "1 min ago" {
		t.Errorf("RelativeTimeShort(~1min) = %q, want %q", got, "1 min ago")
	}
}

func TestRelativeTimeShort1Hour(t *testing.T) {
	t.Parallel()

	got := RelativeTimeShort(time.Now().UTC().Add(-61 * time.Minute))
	if got != "1h ago" {
		t.Errorf("RelativeTimeShort(~1hr) = %q, want %q", got, "1h ago")
	}
}

func TestRelativeTimeShort1Day(t *testing.T) {
	t.Parallel()

	got := RelativeTimeShort(time.Now().UTC().Add(-25 * time.Hour))
	if got != "1d ago" {
		t.Errorf("RelativeTimeShort(~1d) = %q, want %q", got, "1d ago")
	}
}

func TestRelativeTimeShort1Month(t *testing.T) {
	t.Parallel()

	got := RelativeTimeShort(time.Now().UTC().Add(-31 * 24 * time.Hour))
	if got != "1mo ago" {
		t.Errorf("RelativeTimeShort(~1mo) = %q, want %q", got, "1mo ago")
	}
}
