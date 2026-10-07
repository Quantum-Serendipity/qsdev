package toolcheck_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"3.11.7 > 3.11", "3.11.7", "3.11", 1},
		{"3.10 < 3.11", "3.10", "3.11", -1},
		{"3.11 == 3.11", "3.11", "3.11", 0},
		{"22 > 20", "22", "20", 1},
		{"empty < 3.11", "", "3.11", -1},
		{"3.11.0 == 3.11", "3.11.0", "3.11", 0},
		{"1.22.3 > 1.21", "1.22.3", "1.21", 1},
		{"2.43.0 > 2.42.0", "2.43.0", "2.42.0", 1},
		{"equal", "10.2.3", "10.2.3", 0},
		{"patch difference", "10.2.3", "10.2.4", -1},
		{"both empty", "", "", 0},
		{"single vs multi", "20", "20.0.0", 0},
		{"abc graceful", "abc", "3.11", -1},
		{"both non-numeric", "abc", "xyz", 0},
		{"devenv 2.1.2 > 2.1", "2.1.2", "2.1", 1},
		{"devenv 1.4.1 < 2.1", "1.4.1", "2.1", -1},
		{"nix 2.3.0 < 2.4", "2.3.0", "2.4", -1},
		{"nix pre-release segment keeps its leading digits", "2.4pre20211001", "2.4", 0},
		{"build metadata segment keeps its leading digits", "2.1.2+abc", "2.1", 1},
		{"python 3.8.10 < 3.9", "3.8.10", "3.9", -1},
		{"multi-digit leading run", "2.19rc1", "2.4", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toolcheck.CompareVersions(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMeetsMinimum(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		version, minVer string
		want            bool
	}{
		{"3.11.7 >= 3.11", "3.11.7", "3.11", true},
		{"3.10 < 3.11", "3.10", "3.11", false},
		{"3.11 >= 3.11", "3.11", "3.11", true},
		{"20.11.0 >= 18", "20.11.0", "18", true},
		{"empty < 3.11", "", "3.11", false},
		{"1.22.3 >= 1.21", "1.22.3", "1.21", true},
		{"both empty", "", "", true},
		{"no floor accepts any version", "0.0.1", "", true},
		{"2.1.2 >= 2.1", "2.1.2", "2.1", true},
		{"1.4.1 < 2.1", "1.4.1", "2.1", false},
		{"2.3.0 < 2.4", "2.3.0", "2.4", false},
		{"2.4pre20211001 >= 2.4", "2.4pre20211001", "2.4", true},
		{"2.1.2+abc >= 2.1", "2.1.2+abc", "2.1", true},
		{"3.8.10 < 3.9", "3.8.10", "3.9", false},
		{"empty never meets a zero floor", "", "0", false},
		{"empty never meets a floor", "", "2.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toolcheck.MeetsMinimum(tt.version, tt.minVer)
			if got != tt.want {
				t.Errorf("MeetsMinimum(%q, %q) = %v, want %v", tt.version, tt.minVer, got, tt.want)
			}
		})
	}
}
