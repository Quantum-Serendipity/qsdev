package sectools_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateGitleaksToml_Structure(t *testing.T) {
	f, err := sectools.GenerateGitleaksToml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateGitleaksToml() error: %v", err)
	}
	if f.Path != ".gitleaks.toml" {
		t.Errorf("Path = %q, want %q", f.Path, ".gitleaks.toml")
	}
	if f.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0o644)
	}
	if f.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", f.Strategy)
	}
	if f.Owner != "gitleaks" {
		t.Errorf("Owner = %q, want %q", f.Owner, "gitleaks")
	}
}

func TestGenerateGitleaksToml_TOMLContent(t *testing.T) {
	f, err := sectools.GenerateGitleaksToml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateGitleaksToml() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "title = \"gitleaks config\"") {
		t.Error("content should contain title")
	}
	if !strings.Contains(content, "[allowlist]") {
		t.Error("content should contain [allowlist] section")
	}
	if !strings.Contains(content, "regexes = []") {
		t.Error("content should contain empty regexes list")
	}
}

func TestGenerateGitleaksToml_AllowlistPaths(t *testing.T) {
	f, err := sectools.GenerateGitleaksToml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateGitleaksToml() error: %v", err)
	}
	content := string(f.Content)

	// Allowlist paths are regexes: each must be anchored to a path-segment
	// boundary and have its metacharacters escaped.
	for _, path := range []string{`(^|/)vendor/`, `(^|/)node_modules/`, `(^|/)\.devenv/`, `(^|/)testdata/`} {
		if !strings.Contains(content, "'''"+path+"'''") {
			t.Errorf("content should contain allowlisted path regex %q", path)
		}
	}

	// docs/ should NOT be in the allowlist — it may contain legitimate secrets
	// documentation that should still be scanned.
	if strings.Contains(content, `"docs/"`) {
		t.Error("content should NOT contain allowlisted path \"docs/\"")
	}
}

// TestGenerateGitleaksToml_ExtendsDefaultRules guards against the generated
// config replacing Gitleaks' built-in rules with an empty rule set: a repo-root
// .gitleaks.toml without [extend] useDefault = true detects nothing.
func TestGenerateGitleaksToml_ExtendsDefaultRules(t *testing.T) {
	t.Parallel()
	f, err := sectools.GenerateGitleaksToml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateGitleaksToml() error: %v", err)
	}
	var cfg struct {
		Extend struct {
			UseDefault bool `toml:"useDefault"`
		} `toml:"extend"`
		Allowlist struct {
			Paths []string `toml:"paths"`
		} `toml:"allowlist"`
	}
	if _, err := toml.Decode(string(f.Content), &cfg); err != nil {
		t.Fatalf("generated .gitleaks.toml is not valid TOML: %v", err)
	}
	if !cfg.Extend.UseDefault {
		t.Error("[extend] useDefault must be true so the built-in rules stay active")
	}
	for _, p := range cfg.Allowlist.Paths {
		re, err := regexp.Compile(p)
		if err != nil {
			t.Errorf("allowlist path %q is not a valid regex: %v", p, err)
			continue
		}
		if re.MatchString("src/myvendor/x.go") || re.MatchString("src/xdevenv/x.go") {
			t.Errorf("allowlist path %q matches outside its directory", p)
		}
	}
}

// fakeGitHubPAT builds a GitHub PAT-shaped token at runtime so this test file
// never contains a literal credential (which pre-commit secret scanners flag).
func fakeGitHubPAT() string {
	const alphabet = "Zq8Rk2Lm7Xv4Tn9Pb3Wc6Hy5Jd1Gf0Sa"
	var b strings.Builder
	b.WriteString("gh" + "p_")
	for i := range 36 {
		b.WriteByte(alphabet[(i*7+3)%len(alphabet)])
	}
	return b.String()
}

// TestGenerateGitleaksToml_DetectsWithBinary runs the real gitleaks binary with
// the generated config and asserts a known token is reported, while a token
// under an allowlisted directory is not.
func TestGenerateGitleaksToml_DetectsWithBinary(t *testing.T) {
	t.Parallel()
	bin, err := exec.LookPath("gitleaks")
	if err != nil {
		t.Skip("gitleaks not installed")
	}
	f, err := sectools.GenerateGitleaksToml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateGitleaksToml() error: %v", err)
	}

	tests := []struct {
		name      string
		file      string
		wantLeaks bool
	}{
		{name: "source file is scanned", file: "main.go", wantLeaks: true},
		{name: "look-alike directory is scanned", file: "myvendor/main.go", wantLeaks: true},
		{name: "vendor directory is allowlisted", file: "vendor/lib/main.go", wantLeaks: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, ".gitleaks.toml")
			if err := os.WriteFile(cfgPath, f.Content, 0o644); err != nil {
				t.Fatal(err)
			}
			src := filepath.Join(dir, filepath.FromSlash(tt.file))
			if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
				t.Fatal(err)
			}
			body := "package main\n\nconst token = \"" + fakeGitHubPAT() + "\"\n"
			if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(bin, "dir", "--no-banner", "--config", cfgPath, "--exit-code", "3", dir)
			out, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			gotLeaks := errors.As(err, &exitErr) && exitErr.ExitCode() == 3
			if err != nil && !gotLeaks {
				t.Fatalf("gitleaks failed: %v\n%s", err, out)
			}
			if gotLeaks != tt.wantLeaks {
				t.Errorf("leaks found = %v, want %v\n%s", gotLeaks, tt.wantLeaks, out)
			}
		})
	}
}
