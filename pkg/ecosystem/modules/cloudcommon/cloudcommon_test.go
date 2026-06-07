package cloudcommon

import (
	"sort"
	"testing"
)

func TestReadDenyPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider CloudProvider
		want     int
	}{
		{name: "AWS", provider: AWS, want: 3},
		{name: "GCP", provider: GCP, want: 4},
		{name: "Azure", provider: Azure, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			paths := ReadDenyPaths(tt.provider)
			if len(paths) != tt.want {
				t.Errorf("ReadDenyPaths(%s) returned %d paths, want %d", tt.provider, len(paths), tt.want)
			}
		})
	}
}

func TestBashDenyRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider CloudProvider
		want     int
	}{
		{name: "AWS", provider: AWS, want: 5},
		{name: "GCP", provider: GCP, want: 5},
		{name: "Azure", provider: Azure, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rules := BashDenyRules(tt.provider)
			if len(rules) != tt.want {
				t.Errorf("BashDenyRules(%s) returned %d rules, want %d", tt.provider, len(rules), tt.want)
			}
		})
	}
}

func TestAllBashDenyRules_MultiProvider(t *testing.T) {
	t.Parallel()

	rules := AllBashDenyRules([]CloudProvider{AWS, GCP, Azure})

	// Verify no duplicates.
	seen := make(map[string]bool)
	for _, r := range rules {
		if seen[r] {
			t.Errorf("duplicate rule: %s", r)
		}
		seen[r] = true
	}

	// Verify sorted.
	if !sort.StringsAreSorted(rules) {
		t.Error("AllBashDenyRules result is not sorted")
	}

	// Total should be 5 + 5 + 4 = 14 (no overlaps between providers).
	if len(rules) != 14 {
		t.Errorf("AllBashDenyRules returned %d rules, want 14", len(rules))
	}
}

func TestAllReadDenyPaths_MultiProvider(t *testing.T) {
	t.Parallel()

	paths := AllReadDenyPaths([]CloudProvider{AWS, GCP, Azure})

	// Verify no duplicates.
	seen := make(map[string]bool)
	for _, p := range paths {
		if seen[p] {
			t.Errorf("duplicate path: %s", p)
		}
		seen[p] = true
	}

	// Verify sorted.
	if !sort.StringsAreSorted(paths) {
		t.Error("AllReadDenyPaths result is not sorted")
	}

	// Total should be 3 + 4 + 4 = 11 (no overlaps between providers).
	if len(paths) != 11 {
		t.Errorf("AllReadDenyPaths returned %d paths, want 11", len(paths))
	}
}

func TestEnvVarForProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider CloudProvider
		want     string
	}{
		{name: "AWS", provider: AWS, want: "AWS_PROFILE"},
		{name: "GCP", provider: GCP, want: "CLOUDSDK_ACTIVE_CONFIG_NAME"},
		{name: "Azure", provider: Azure, want: "ARM_SUBSCRIPTION_ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := EnvVarForProvider(tt.provider)
			if got != tt.want {
				t.Errorf("EnvVarForProvider(%s) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestEnvVarForProvider_Unknown(t *testing.T) {
	t.Parallel()

	got := EnvVarForProvider(CloudProvider("unknown"))
	if got != "" {
		t.Errorf("EnvVarForProvider(unknown) = %q, want empty string", got)
	}
}
