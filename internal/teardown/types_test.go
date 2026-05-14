package teardown

import "testing"

func TestProfileConstants(t *testing.T) {
	profiles := []Profile{ProfileQuick, ProfileDefault, ProfileCompliance}
	seen := make(map[Profile]bool)

	for _, p := range profiles {
		if p == "" {
			t.Errorf("profile constant should not be empty")
		}
		if seen[p] {
			t.Errorf("duplicate profile constant: %s", p)
		}
		seen[p] = true
	}

	if len(seen) != 3 {
		t.Errorf("expected 3 distinct profiles, got %d", len(seen))
	}
}

func TestProfileValues(t *testing.T) {
	tests := []struct {
		profile Profile
		want    string
	}{
		{ProfileQuick, "quick"},
		{ProfileDefault, "default"},
		{ProfileCompliance, "compliance"},
	}

	for _, tt := range tests {
		if string(tt.profile) != tt.want {
			t.Errorf("Profile(%q) = %q, want %q", tt.profile, string(tt.profile), tt.want)
		}
	}
}
