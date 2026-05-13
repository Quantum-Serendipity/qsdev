package doctor

import "testing"

// findCheck looks up a ToolCheck by name from the default registry.
func findCheck(t *testing.T, name string) ToolCheck {
	t.Helper()
	for _, tc := range DefaultChecks() {
		if tc.Name == name {
			return tc
		}
	}
	t.Fatalf("no check found with name %q", name)
	return ToolCheck{} // unreachable
}

func TestParseGitVersion(t *testing.T) {
	tc := findCheck(t, "git")
	tests := []struct {
		raw, want string
	}{
		{"git version 2.43.0", "2.43.0"},
		{"git version 2.47.1", "2.47.1"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("git ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseGoVersion(t *testing.T) {
	tc := findCheck(t, "go")
	tests := []struct {
		raw, want string
	}{
		{"go version go1.22.3 linux/amd64", "1.22.3"},
		{"go version go1.21.6 darwin/arm64", "1.21.6"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("go ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseNodeVersion(t *testing.T) {
	tc := findCheck(t, "node")
	tests := []struct {
		raw, want string
	}{
		{"v20.11.0", "20.11.0"},
		{"v22.1.0", "22.1.0"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("node ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseNpmVersion(t *testing.T) {
	tc := findCheck(t, "npm")
	tests := []struct {
		raw, want string
	}{
		{"10.2.3", "10.2.3"},
		{"9.8.1", "9.8.1"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("npm ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseNixVersion(t *testing.T) {
	tc := findCheck(t, "nix")
	tests := []struct {
		raw, want string
	}{
		{"nix (Nix) 2.19.3", "2.19.3"},
		{"nix (Nix) 2.24.6", "2.24.6"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("nix ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseJqVersion(t *testing.T) {
	tc := findCheck(t, "jq")
	tests := []struct {
		raw, want string
	}{
		{"jq-1.7.1", "1.7.1"},
		{"jq-1.6", "1.6"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("jq ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseCurlVersion(t *testing.T) {
	tc := findCheck(t, "curl")
	tests := []struct {
		raw, want string
	}{
		{"curl 8.5.0 (x86_64-pc-linux-gnu) libcurl/8.5.0", "8.5.0"},
		{"curl 7.88.1 (aarch64-apple-darwin22.0) libcurl/7.88.1", "7.88.1"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("curl ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParsePythonVersion(t *testing.T) {
	tc := findCheck(t, "python3")
	tests := []struct {
		raw, want string
	}{
		{"Python 3.11.7", "3.11.7"},
		{"Python 3.12.0", "3.12.0"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("python3 ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseShellcheckVersion(t *testing.T) {
	tc := findCheck(t, "shellcheck")
	tests := []struct {
		raw, want string
	}{
		{"ShellCheck - shell script analysis tool\nversion: 0.10.0\nlicense: GNU General Public License, version 3", "0.10.0"},
		{"ShellCheck - shell script analysis tool\nversion: 0.9.0\n", "0.9.0"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("shellcheck ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseHadolintVersion(t *testing.T) {
	tc := findCheck(t, "hadolint")
	tests := []struct {
		raw, want string
	}{
		{"Haskell Dockerfile Linter 2.12.0-no-git", "2.12.0"},
		{"Haskell Dockerfile Linter 2.10.0", "2.10.0"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("hadolint ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParsePrecommitVersion(t *testing.T) {
	tc := findCheck(t, "pre-commit")
	tests := []struct {
		raw, want string
	}{
		{"pre-commit 3.7.0", "3.7.0"},
		{"pre-commit 3.6.2", "3.6.2"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("pre-commit ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseShfmtVersion(t *testing.T) {
	tc := findCheck(t, "shfmt")
	tests := []struct {
		raw, want string
	}{
		{"v3.8.0", "3.8.0"},
		{"v3.7.0", "3.7.0"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("shfmt ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseDevenvVersion(t *testing.T) {
	tc := findCheck(t, "devenv")
	tests := []struct {
		raw, want string
	}{
		{"devenv 1.4.1", "1.4.1"},
		{"1.3.0", "1.3.0"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("devenv ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestParseDirenvVersion(t *testing.T) {
	tc := findCheck(t, "direnv")
	tests := []struct {
		raw, want string
	}{
		{"2.34.0", "2.34.0"},
		{"2.33.0", "2.33.0"},
		{"", ""},
	}
	for _, tt := range tests {
		got := tc.ParseVersion(tt.raw)
		if got != tt.want {
			t.Errorf("direnv ParseVersion(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestDefaultChecksCount(t *testing.T) {
	checks := DefaultChecks()
	if len(checks) != 15 {
		t.Errorf("DefaultChecks() returned %d checks, want 15", len(checks))
	}
}
