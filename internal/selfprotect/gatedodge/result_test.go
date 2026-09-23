package gatedodge

import "testing"

func TestResultRuleFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{"/p/.npmrc", "GD-004"},
		{".npmrc", "GD-004"},
		{"/p/.NPMRC", "GD-004"},
		{"/p/.Pre-Commit-Config.yaml", "GD-003"},
		{"/p/.pre-commit-config.yaml", "GD-003"},
		{"/p/x.npmrc", ""},
		{"/p/.qsdev.yaml", ""},
		{"/p/main.go", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got := ""
			if r := ResultRuleFor(tt.path); r != nil {
				got = r.ID
			}
			if got != tt.want {
				t.Errorf("ResultRuleFor(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestResultRule_Npmrc verifies that deleting or neutralizing ignore-scripts —
// which Detect cannot see, since it only scans introduced text — is blocked.
func TestResultRule_Npmrc(t *testing.T) {
	t.Parallel()

	const guarded = "registry=https://r\nignore-scripts=true\n"
	tests := []struct {
		name    string
		before  string
		after   string
		blocked bool
	}{
		{"line deleted", guarded, "registry=https://r\n", true},
		{"file emptied", guarded, "", true},
		{"set to 0", guarded, "ignore-scripts=0\n", true},
		{"set to false with spaces", "", "ignore-scripts = false\n", true},
		{"new file disables", "", "ignore-scripts=0\n", true},
		{"commented out", guarded, "; ignore-scripts=true\n", true},
		{"later assignment wins", guarded, "ignore-scripts=true\nignore-scripts=no\n", true},

		{"unchanged", guarded, guarded, false},
		{"other line edited", guarded, "registry=https://other\nignore-scripts=true\n", false},
		{"quoted true", guarded, "ignore-scripts=\"true\"\n", false},
		{"numeric true", guarded, "ignore-scripts=1\n", false},
		{"bare key is true", guarded, "ignore-scripts\n", false},
		{"never guarded", "registry=https://r\n", "", false},
		{"adds the guard", "", guarded, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, reason := ResultRuleFor(".npmrc").Check(tt.before, tt.after)
			if blocked != tt.blocked {
				t.Errorf("Check(%q -> %q) = (%v, %q), want blocked %v", tt.before, tt.after, blocked, reason, tt.blocked)
			}
		})
	}
}

// TestResultRule_Precommit verifies that removing configured hooks is blocked
// however it is expressed (emptying repos, deleting one hook, emptying the
// file), while edits that keep every hook pass.
func TestResultRule_Precommit(t *testing.T) {
	t.Parallel()

	const guarded = `repos:
  - repo: https://github.com/sirwart/ripsecrets
    rev: v0.1.8
    hooks:
      - id: ripsecrets
  - repo: local
    hooks:
      - id: golangci-lint
        entry: golangci-lint run
`
	tests := []struct {
		name    string
		before  string
		after   string
		blocked bool
	}{
		{"repos emptied", guarded, "repos: []\n", true},
		{"file emptied", guarded, "", true},
		{"one hook removed", guarded, `repos:
  - repo: https://github.com/sirwart/ripsecrets
    rev: v0.1.8
    hooks:
      - id: ripsecrets
`, true},
		{"hook moved to a dummy local repo", guarded, `repos:
  - repo: local
    hooks:
      - id: ripsecrets
      - id: golangci-lint
`, true},
		{"unparseable result", guarded, "repos: [\n", true},

		{"unchanged", guarded, guarded, false},
		{"rev bumped and hook added", guarded, `repos:
  - repo: https://github.com/sirwart/ripsecrets
    rev: v0.1.9
    hooks:
      - id: ripsecrets
  - repo: local
    hooks:
      - id: golangci-lint
        entry: golangci-lint run ./...
      - id: go-vet
`, false},
		{"new config", "", guarded, false},
		{"nothing to remove", "repos: []\n", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, reason := ResultRuleFor(".pre-commit-config.yaml").Check(tt.before, tt.after)
			if blocked != tt.blocked {
				t.Errorf("Check = (%v, %q), want blocked %v", blocked, reason, tt.blocked)
			}
		})
	}
}
