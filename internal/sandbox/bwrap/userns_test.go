package bwrap

import (
	"slices"
	"testing"
)

func TestParseBwrapVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		out  string
		want []int
	}{
		{"release", "bubblewrap 0.11.2\n", []int{0, 11, 2}},
		{"two components", "bubblewrap 0.8", []int{0, 8}},
		{"not bubblewrap", "bwrap-compat 1.0.0", nil},
		{"garbage version", "bubblewrap 0.x.1", nil},
		{"empty", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseBwrapVersion(tt.out); !slices.Equal(got, tt.want) {
				t.Errorf("parseBwrapVersion(%q) = %v, want %v", tt.out, got, tt.want)
			}
		})
	}
}

func TestVersionAtLeast(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version []int
		want    bool
	}{
		{"exact minimum", []int{0, 8, 0}, true},
		{"newer minor", []int{0, 11, 2}, true},
		{"newer major", []int{1, 0}, true},
		{"missing patch equals minimum", []int{0, 8}, true},
		{"older minor", []int{0, 7, 9}, false},
		{"unparsed", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := versionAtLeast(tt.version, disableUserNSMinVersion); got != tt.want {
				t.Errorf("versionAtLeast(%v) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
