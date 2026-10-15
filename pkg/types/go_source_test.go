package types

import "testing"

func TestIsValidGoSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want bool
	}{
		{GoSourceGoOverlay, true},
		{GoSourceNixpkgs, true},
		{"", false},
		{"true", false},
		{"Nixpkgs", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := IsValidGoSource(tt.in); got != tt.want {
				t.Errorf("IsValidGoSource(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
