package doctor

import "testing"

func TestCompareVersions(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareVersions(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestMeetsMinimum(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MeetsMinimum(tt.version, tt.minVer)
			if got != tt.want {
				t.Errorf("MeetsMinimum(%q, %q) = %v, want %v", tt.version, tt.minVer, got, tt.want)
			}
		})
	}
}
