package termutil

import (
	"os"
	"testing"
)

func TestIsAccessible(t *testing.T) {
	tests := []struct {
		name       string
		accessible string
		noColor    string
		term       string
		want       bool
	}{
		{
			name: "false when no env vars set",
			want: false,
		},
		{
			name:       "true when ACCESSIBLE set",
			accessible: "1",
			want:       true,
		},
		{
			name:    "true when NO_COLOR set",
			noColor: "1",
			want:    true,
		},
		{
			name: "true when TERM is dumb",
			term: "dumb",
			want: true,
		},
		{
			name: "false when TERM is xterm",
			term: "xterm-256color",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origAccessible := os.Getenv("ACCESSIBLE")
			origNoColor := os.Getenv("NO_COLOR")
			origTerm := os.Getenv("TERM")
			t.Cleanup(func() {
				restoreEnv("ACCESSIBLE", origAccessible)
				restoreEnv("NO_COLOR", origNoColor)
				restoreEnv("TERM", origTerm)
			})

			os.Unsetenv("ACCESSIBLE")
			os.Unsetenv("NO_COLOR")
			os.Unsetenv("TERM")

			if tt.accessible != "" {
				os.Setenv("ACCESSIBLE", tt.accessible)
			}
			if tt.noColor != "" {
				os.Setenv("NO_COLOR", tt.noColor)
			}
			if tt.term != "" {
				os.Setenv("TERM", tt.term)
			}

			if got := IsAccessible(); got != tt.want {
				t.Errorf("IsAccessible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func restoreEnv(key, orig string) {
	if orig != "" {
		os.Setenv(key, orig)
	} else {
		os.Unsetenv(key)
	}
}
