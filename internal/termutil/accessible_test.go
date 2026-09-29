package termutil

import "testing"

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
			// NO_COLOR (https://no-color.org/) only asks for colorless output;
			// lipgloss/termenv honor it, so the full TUI stays available.
			name:    "false when only NO_COLOR set",
			noColor: "1",
			term:    "xterm-256color",
			want:    false,
		},
		{
			name:       "true when ACCESSIBLE and NO_COLOR set",
			accessible: "1",
			noColor:    "1",
			want:       true,
		},
		{
			name:    "true when TERM is dumb even with NO_COLOR set",
			noColor: "1",
			term:    "dumb",
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
			// t.Setenv restores each variable's original set-or-unset state;
			// IsAccessible treats an empty value as unset.
			t.Setenv("ACCESSIBLE", tt.accessible)
			t.Setenv("NO_COLOR", tt.noColor)
			t.Setenv("TERM", tt.term)

			if got := IsAccessible(); got != tt.want {
				t.Errorf("IsAccessible() = %v, want %v", got, tt.want)
			}
		})
	}
}
