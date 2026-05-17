package tmpl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

type nixTestData struct {
	Packages      []string
	BareItems     []string
	StringItems   []string
	Greeting      string
	EnableFeature bool
	ShellScript   string
	EnvVars       map[string]string
	IndentedBlock string
}

func TestRenderNixTemplate(t *testing.T) {
	fsys := os.DirFS(".")
	r, err := NewNixRenderer(fsys, "testdata")
	if err != nil {
		t.Fatalf("NewNixRenderer: %v", err)
	}

	data := nixTestData{
		Packages:      []string{"git", "curl"},
		BareItems:     []string{"x86_64-linux", "aarch64-linux"},
		StringItems:   []string{"${HOME}/bin", "plain"},
		Greeting:      "hello ${world}",
		EnableFeature: true,
		ShellScript:   "echo ${var}",
		EnvVars:       map[string]string{"FOO": "bar", "BAZ": "qux"},
		IndentedBlock: "# indented\nvalue = true;",
	}

	got, err := r.RenderString("test.nix", data)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}

	want := `{ pkgs, lib, ... }:
{
  packages = [ pkgs.git pkgs.curl ];
  bareList = [ x86_64-linux aarch64-linux ];
  stringList = [ "\${HOME}/bin" "plain" ];
  greeting = "hello \${world}";
  enableFeature = true;
  shellScript = ''
    echo ''${var}
  '';
  env = { BAZ = "qux"; FOO = "bar"; };
  # indented
  value = true;
}
`
	if got != want {
		t.Errorf("rendered Nix template mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderMarkdownTemplate(t *testing.T) {
	// Read the markdown template from testdata and use an in-memory FS
	// so the Markdown renderer doesn't try to parse Nix templates (which
	// use functions not in MarkdownFuncMap).
	mdContent, err := os.ReadFile("testdata/test.md.tmpl")
	if err != nil {
		t.Fatalf("reading test.md.tmpl: %v", err)
	}
	fsys := fstest.MapFS{
		"templates/test.md.tmpl": &fstest.MapFile{Data: mdContent},
	}

	r, err := NewMarkdownRenderer(fsys, "templates")
	if err != nil {
		t.Fatalf("NewMarkdownRenderer: %v", err)
	}

	data := struct {
		Header       string
		Title        string
		Subtitle     string
		Tags         []string
		Version      string
		Name         string
		MaybeEmpty   string
		Block        string
		TrailingText string
	}{
		Header:       "This is a header\nSecond line",
		Title:        "My Project",
		Subtitle:     "A SUBTITLE",
		Tags:         []string{"go", "nix"},
		Version:      "1.0",
		Name:         "test",
		MaybeEmpty:   "",
		Block:        "line1\nline2",
		TrailingText: "text\n\n",
	}

	got, err := r.RenderString("test.md", data)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}

	// Verify key features are present.
	if !strings.Contains(got, "## This is a header") {
		t.Error("expected comment prefix applied to header")
	}
	if !strings.Contains(got, "MY PROJECT") {
		t.Error("expected upper-cased title")
	}
	if !strings.Contains(got, "a subtitle") {
		t.Error("expected lower-cased subtitle")
	}
	if !strings.Contains(got, "Tags: go, nix") {
		t.Error("expected joined tags")
	}
	if !strings.Contains(got, "This is a Go project.") {
		t.Error("expected contains check for 'go' tag")
	}
	if !strings.Contains(got, "Version: 1.0, Name: test") {
		t.Error("expected dict values")
	}
	if !strings.Contains(got, "Default value: fallback") {
		t.Error("expected default fallback for empty string")
	}
	if !strings.Contains(got, "    line1\n    line2") {
		t.Error("expected nindented block")
	}
	// trimTrailingNewline should have removed trailing newlines from "text\n\n".
	if strings.Contains(got, "text\n\n\n") {
		t.Error("expected trimTrailingNewline to strip trailing newlines")
	}
}

func TestRenderMissingTemplate(t *testing.T) {
	fsys := os.DirFS(".")
	r, err := NewNixRenderer(fsys, "testdata")
	if err != nil {
		t.Fatalf("NewNixRenderer: %v", err)
	}

	_, err = r.Render("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention missing template name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "available templates") {
		t.Errorf("error should list available templates, got: %v", err)
	}
	if !strings.Contains(err.Error(), "test.nix") {
		t.Errorf("error should include 'test.nix' in available templates, got: %v", err)
	}
}

func TestRenderNixValidation(t *testing.T) {
	nixInst, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Skip("nix-instantiate not on PATH, skipping Nix validation test")
	}

	// Use a purpose-built template that produces standalone-valid Nix.
	// The main test.nix.tmpl uses nixList (bare identifiers) which are
	// only valid inside certain Nix contexts, so we use a simpler template
	// that only produces quoted strings, lists, and attr sets.
	tmplContent := `{ pkgs, ... }:
{
  packages = {{ nixPkgList .Packages }};
  stringList = {{ nixStringList .StringItems }};
  greeting = {{ nixString .Greeting }};
  enableFeature = {{ nixBool .EnableFeature }};
  shellScript = ''
    {{ nixMultiline .ShellScript }}
  '';
  env = {{ nixAttrSet .EnvVars }};
{{ indent 2 .IndentedBlock }}
}
`
	fsys := fstest.MapFS{
		"nix/valid.nix.tmpl": &fstest.MapFile{Data: []byte(tmplContent)},
	}
	r, err := NewNixRenderer(fsys, "nix")
	if err != nil {
		t.Fatalf("NewNixRenderer: %v", err)
	}

	data := struct {
		Packages      []string
		StringItems   []string
		Greeting      string
		EnableFeature bool
		ShellScript   string
		EnvVars       map[string]string
		IndentedBlock string
	}{
		Packages:      []string{"git", "curl"},
		StringItems:   []string{"plain"},
		Greeting:      "hello world",
		EnableFeature: false,
		ShellScript:   "echo hello",
		EnvVars:       map[string]string{"KEY": "value"},
		IndentedBlock: "# comment",
	}

	rendered, err := r.Render("valid.nix", data)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// Write to a temp file because nix-instantiate --parse /dev/stdin
	// doesn't work reliably in all environments.
	tmpFile := filepath.Join(t.TempDir(), "test.nix")
	if err := os.WriteFile(tmpFile, rendered, 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	cmd := exec.Command(nixInst, "--parse", tmpFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("nix-instantiate --parse failed:\n%s\nrendered output:\n%s", output, rendered)
	}
}

func TestFuncMapIsolation(t *testing.T) {
	m1 := NixFuncMap()
	m2 := NixFuncMap()

	// Mutate m1 and verify m2 is unaffected.
	m1["customKey"] = func() string { return "custom" }

	if _, ok := m2["customKey"]; ok {
		t.Error("mutating one NixFuncMap should not affect another")
	}
}

func TestNewRendererNoTemplates(t *testing.T) {
	fsys := fstest.MapFS{
		"empty/readme.txt": &fstest.MapFile{Data: []byte("not a template")},
	}
	_, err := NewNixRenderer(fsys, "empty")
	if err == nil {
		t.Fatal("expected error when no .tmpl files found")
	}
	if !strings.Contains(err.Error(), "no *.tmpl files found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAvailableTemplates(t *testing.T) {
	fsys := os.DirFS(".")
	r, err := NewNixRenderer(fsys, "testdata")
	if err != nil {
		t.Fatalf("NewNixRenderer: %v", err)
	}

	templates := r.AvailableTemplates()
	if len(templates) == 0 {
		t.Fatal("expected at least one template")
	}

	found := false
	for _, name := range templates {
		if name == "test.nix" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'test.nix' in available templates: %v", templates)
	}
}
