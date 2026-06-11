package bugreport

import (
	"strings"
	"testing"
)

func TestEnvironmentFormatTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		env        Environment
		wantParts  []string
		wantAbsent []string
	}{
		{
			name: "full environment",
			env: Environment{
				QsdevVersion:    "0.8.0",
				Commit:          "abc123",
				GoVersion:       "go1.22.0",
				OS:              "linux",
				Arch:            "amd64",
				Family:          "nixos",
				Shell:           "zsh",
				HasNix:          true,
				DevenvVer:       "1.3.0",
				Ecosystems:      []string{"go", "node"},
				ActiveToolCount: 5,
				SecurityProfile: "strict",
			},
			wantParts: []string{
				"| Field | Value |",
				"|-------|-------|",
				"0.8.0",
				"abc123",
				"go1.22.0",
				"linux/amd64",
				"nixos",
				"zsh",
				"installed",
				"1.3.0",
				"go, node",
				"5",
				"strict",
			},
		},
		{
			name: "no nix no devenv no ecosystems",
			env: Environment{
				QsdevVersion: "0.7.3",
				Commit:       "def456",
				GoVersion:    "go1.21.5",
				OS:           "darwin",
				Arch:         "arm64",
				Family:       "macos",
				Shell:        "zsh",
				HasNix:       false,
			},
			wantParts: []string{
				"not found",
				"darwin/arm64",
				"macos",
			},
			wantAbsent: []string{
				"devenv",
				"Ecosystems",
				"Active tools",
				"Security profile",
			},
		},
		{
			name: "with devenv but no ecosystems",
			env: Environment{
				QsdevVersion: "0.8.0",
				Commit:       "111222",
				GoVersion:    "go1.22.0",
				OS:           "linux",
				Arch:         "amd64",
				Family:       "ubuntu",
				Shell:        "bash",
				HasNix:       true,
				DevenvVer:    "1.2.0",
			},
			wantParts: []string{
				"1.2.0",
				"installed",
			},
			wantAbsent: []string{
				"Ecosystems",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			table := tt.env.FormatTable()

			for _, part := range tt.wantParts {
				if !strings.Contains(table, part) {
					t.Errorf("FormatTable() missing %q in output:\n%s", part, table)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(table, absent) {
					t.Errorf("FormatTable() should not contain %q in output:\n%s", absent, table)
				}
			}
		})
	}
}

func TestEnvironmentFormatTableMarkdownStructure(t *testing.T) {
	t.Parallel()

	env := Environment{
		QsdevVersion: "1.0.0",
		Commit:       "aaa111",
		GoVersion:    "go1.22.0",
		OS:           "linux",
		Arch:         "amd64",
		Family:       "nixos",
		Shell:        "zsh",
		HasNix:       true,
	}

	table := env.FormatTable()
	lines := strings.Split(strings.TrimRight(table, "\n"), "\n")

	// First line is header, second is separator.
	if len(lines) < 2 {
		t.Fatalf("table has %d lines, want at least 2", len(lines))
	}

	if !strings.HasPrefix(lines[0], "| Field") {
		t.Errorf("first line = %q, want header starting with '| Field'", lines[0])
	}
	if !strings.HasPrefix(lines[1], "|---") {
		t.Errorf("second line = %q, want separator starting with '|---'", lines[1])
	}

	// Every data line should start and end with pipe.
	for i, line := range lines {
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			t.Errorf("line %d = %q, should start and end with '|'", i, line)
		}
	}
}

func TestBoolStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input bool
		want  string
	}{
		{true, "installed"},
		{false, "not found"},
	}

	for _, tt := range tests {
		got := boolStr(tt.input)
		if got != tt.want {
			t.Errorf("boolStr(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBrowserURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		title    string
		body     string
		wantSubs []string
	}{
		{
			name:  "short body",
			title: "Bug title",
			body:  "Short body text.",
			wantSubs: []string{
				"https://github.com/",
				"/issues/new?",
				"title=",
				"labels=bug",
				"body=",
				"Short+body+text",
			},
		},
		{
			name:  "special characters in title",
			title: "Crash: init fails with & in path",
			body:  "Details here.",
			wantSubs: []string{
				"Crash",
				"init+fails",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BrowserURL(tt.title, tt.body)

			for _, sub := range tt.wantSubs {
				if !strings.Contains(got, sub) {
					t.Errorf("BrowserURL(%q, %q) = %q, missing %q", tt.title, tt.body, got, sub)
				}
			}
		})
	}
}

func TestBrowserURLTruncation(t *testing.T) {
	t.Parallel()

	// Body exceeding browserMaxLen (8000) should be truncated.
	longBody := strings.Repeat("x", 9000)
	got := BrowserURL("title", longBody)

	if !strings.Contains(got, "truncated") {
		t.Error("long body should include truncation notice")
	}
}

func TestBrowserURLShortBodyNotTruncated(t *testing.T) {
	t.Parallel()

	shortBody := strings.Repeat("y", 100)
	got := BrowserURL("title", shortBody)

	if strings.Contains(got, "truncated") {
		t.Error("short body should not be truncated")
	}
}

func TestSaveToFile(t *testing.T) {
	// Not parallel: t.Setenv is incompatible with t.Parallel.
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path, err := SaveToFile("Test Bug", "Body of bug report")
	if err != nil {
		t.Fatalf("SaveToFile error: %v", err)
	}

	if path == "" {
		t.Fatal("SaveToFile returned empty path")
	}
	if !strings.Contains(path, "bug-report-") {
		t.Errorf("path = %q, want containing 'bug-report-'", path)
	}
	if !strings.HasSuffix(path, ".md") {
		t.Errorf("path = %q, want .md extension", path)
	}
}

func TestFormatTableWithEcosystems(t *testing.T) {
	t.Parallel()

	env := Environment{
		QsdevVersion:    "1.0.0",
		Commit:          "aaa",
		GoVersion:       "go1.22",
		OS:              "linux",
		Arch:            "amd64",
		Family:          "nixos",
		Shell:           "zsh",
		HasNix:          true,
		Ecosystems:      []string{"go", "python", "rust"},
		ActiveToolCount: 12,
		SecurityProfile: "hardened",
	}

	table := env.FormatTable()

	if !strings.Contains(table, "go, python, rust") {
		t.Error("ecosystems not joined with comma-space")
	}
	if !strings.Contains(table, "| Active tools | 12 |") {
		t.Error("active tool count not formatted correctly")
	}
	if !strings.Contains(table, "| Security profile | hardened |") {
		t.Error("security profile not formatted correctly")
	}
}

func TestFormatTableGhVersion(t *testing.T) {
	t.Parallel()

	// GhVer is collected but not displayed in FormatTable (by design).
	// Verify it doesn't cause errors.
	env := Environment{
		QsdevVersion: "1.0.0",
		Commit:       "aaa",
		GoVersion:    "go1.22",
		OS:           "linux",
		Arch:         "amd64",
		Family:       "nixos",
		Shell:        "zsh",
		HasNix:       true,
		GhVer:        "gh version 2.30.0",
	}

	table := env.FormatTable()
	if table == "" {
		t.Error("FormatTable returned empty string")
	}
}
