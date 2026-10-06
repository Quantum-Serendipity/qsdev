package termutil

import "testing"

func TestSafe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "github", "github"},
		{"space and unicode letters", "my server é", "my server é"},
		{"ansi escape", "evil\u001b[2K\rok", `"evil\x1b[2K\rok"`},
		{"carriage return", "a\rb", `"a\rb"`},
		{"conceal", "ok \u001b[8m", `"ok \x1b[8m"`},
		{"bidi override", "a\u202eb", `"a\u202eb"`},
		{"newline", "a\nb", `"a\nb"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Safe(tt.in); got != tt.want {
				t.Errorf("Safe(%q) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}
