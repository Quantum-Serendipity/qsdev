package devinit

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldCmd_HasCorrectFlags(t *testing.T) {
	cmd := scaffoldCmd()

	if cmd.Use != "scaffold-instance <appname>" {
		t.Errorf("Use = %q, want %q", cmd.Use, "scaffold-instance <appname>")
	}

	expectedFlags := []string{"github-owner", "github-repo", "output-dir", "module"}
	for _, name := range expectedFlags {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("expected flag %q not found", name)
		}
	}

	if cmd.Flags().ShorthandLookup("o") == nil {
		t.Error("expected shorthand -o for --github-owner")
	}
	if cmd.Flags().ShorthandLookup("r") == nil {
		t.Error("expected shorthand -r for --github-repo")
	}
	if cmd.Flags().ShorthandLookup("d") == nil {
		t.Error("expected shorthand -d for --output-dir")
	}
}

func TestScaffoldCmd_ValidatesAppName(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"starts with number", []string{"1bad"}, "invalid app name"},
		{"has uppercase", []string{"BadName"}, "invalid app name"},
		{"has underscore", []string{"bad_name"}, "invalid app name"},
		{"has space", []string{"bad name"}, "invalid app name"},
		{"empty", []string{""}, "invalid app name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := scaffoldCmd()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(append(tt.args, "--github-owner", "test"))

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestScaffoldCmd_RequiresGitHubOwner(t *testing.T) {
	cmd := scaffoldCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"myapp"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error without --github-owner, got nil")
	}
	if !strings.Contains(err.Error(), "--github-owner is required") {
		t.Errorf("error = %q, want it to contain '--github-owner is required'", err.Error())
	}
}

func TestScaffoldCmd_GeneratesFiles(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "testapp")

	cmd := scaffoldCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"testapp", "--github-owner", "acme-corp", "--output-dir", outputDir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("scaffold failed: %v\nOutput: %s", err, buf.String())
	}

	expectedFiles := []string{
		"cmd/testapp/main.go",
		"go.mod",
		"Makefile",
		".goreleaser.yaml",
		"README.md",
		".gitignore",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(outputDir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %q not found: %v", f, err)
		}
	}
}

func TestScaffoldCmd_MainGoIsValidSyntax(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "validapp")

	cmd := scaffoldCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"validapp", "--github-owner", "test-org", "--output-dir", outputDir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	mainPath := filepath.Join(outputDir, "cmd", "validapp", "main.go")
	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, mainPath, nil, parser.AllErrors)
	if parseErr != nil {
		content, _ := os.ReadFile(mainPath)
		t.Fatalf("generated main.go has syntax errors: %v\n\nContent:\n%s", parseErr, content)
	}
}

func TestScaffoldCmd_GoModHasCorrectModule(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "modapp")

	cmd := scaffoldCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"modapp", "--github-owner", "my-org", "--github-repo", "my-tool", "--output-dir", outputDir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	goModPath := filepath.Join(outputDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}

	if !strings.Contains(string(content), "module github.com/my-org/my-tool") {
		t.Errorf("go.mod does not contain expected module path:\n%s", content)
	}
}

func TestScaffoldCmd_FailsWhenOutputExists(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "existing")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cmd := scaffoldCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"existing", "--github-owner", "test", "--output-dir", outputDir})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when output exists, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want 'already exists'", err.Error())
	}
}

func TestScaffoldCmd_BrandingInMainGo(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "brandtest")

	cmd := scaffoldCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"brandtest", "--github-owner", "cool-co", "--output-dir", outputDir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	mainPath := filepath.Join(outputDir, "cmd", "brandtest", "main.go")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}

	checks := []string{
		`AppName:       "brandtest"`,
		`ConfigFile:    ".brandtest.yaml"`,
		`GitHubOwner:   "cool-co"`,
		`GitHubRepo:    "brandtest"`,
		`EnvPrefix:     "BRANDTEST_"`,
	}
	for _, check := range checks {
		if !strings.Contains(string(content), check) {
			t.Errorf("main.go missing %q", check)
		}
	}
}

func TestValidAppName(t *testing.T) {
	valid := []string{"myapp", "my-app", "a1", "tool123", "x"}
	for _, name := range valid {
		if !validAppName.MatchString(name) {
			t.Errorf("%q should be valid", name)
		}
	}

	invalid := []string{"", "1app", "MyApp", "my_app", "my app", "-app", "APP"}
	for _, name := range invalid {
		if validAppName.MatchString(name) {
			t.Errorf("%q should be invalid", name)
		}
	}
}
