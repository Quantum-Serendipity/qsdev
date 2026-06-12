//go:build !windows

package sysinfo

import "testing"

func TestCleanShellName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain name", "bash", "bash"},
		{"login dash prefix", "-bash", "bash"},
		{"full path", "/bin/bash", "bash"},
		{"full path with dash", "/usr/bin/-zsh", "zsh"},
		{"nix store path", "/nix/store/abc123-bash-5.2/bin/bash", "bash"},
		{"empty string", "", "."},
		{"just dash", "-", ""},
		{"double dash prefix", "--fish", "-fish"},
		{"name with dots", "bash.exe", "bash.exe"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := cleanShellName(tc.input)
			if got != tc.want {
				t.Errorf("cleanShellName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
