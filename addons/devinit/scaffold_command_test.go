package devinit

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"

	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
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

// scaffoldInto runs scaffold-instance for appName into a temp directory and
// returns the output directory.
func scaffoldInto(t *testing.T, appName string) string {
	t.Helper()
	outputDir := filepath.Join(t.TempDir(), appName)
	cmd := scaffoldCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{appName, "--github-owner", "acme", "--output-dir", outputDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("scaffold failed: %v\nOutput: %s", err, buf.String())
	}
	return outputDir
}

// findReplace returns the replace directive for oldPath, or nil.
func findReplace(f *modfile.File, oldPath string) *modfile.Replace {
	for _, r := range f.Replace {
		if r.Old.Path == oldPath {
			return r
		}
	}
	return nil
}

// TestScaffoldCmd_GoModReplacesGdev is the F038 regression: qsdev satisfies
// its gdev requirement only through a replace directive, which Go ignores in
// dependencies, so the scaffold's go.mod must repeat the exact replacement
// from qsdev's own go.mod or module resolution fails.
func TestScaffoldCmd_GoModReplacesGdev(t *testing.T) {
	t.Parallel()
	qsdevGoMod, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("reading qsdev go.mod: %v", err)
	}
	qsdevMod, err := modfile.Parse("go.mod", qsdevGoMod, nil)
	if err != nil {
		t.Fatalf("parsing qsdev go.mod: %v", err)
	}
	want := findReplace(qsdevMod, gdevModulePath)
	if want == nil {
		t.Fatalf("qsdev go.mod has no replace for %s", gdevModulePath)
	}
	if got := want.New.Path + " " + want.New.Version; got != defaultGdevReplacement {
		t.Errorf("defaultGdevReplacement = %q, qsdev go.mod replaces with %q; keep them in sync", defaultGdevReplacement, got)
	}

	out := scaffoldInto(t, "modcheck")
	content, err := os.ReadFile(filepath.Join(out, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	scaffoldMod, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		t.Fatalf("scaffold go.mod does not parse: %v\n%s", err, content)
	}
	got := findReplace(scaffoldMod, gdevModulePath)
	if got == nil {
		t.Fatalf("scaffold go.mod lacks a replace for %s:\n%s", gdevModulePath, content)
	}
	if got.New != want.New {
		t.Errorf("scaffold replaces gdev with %v, qsdev uses %v", got.New, want.New)
	}
	for _, r := range scaffoldMod.Require {
		if r.Mod.Path == "github.com/Quantum-Serendipity/qsdev" && !semver.IsValid(r.Mod.Version) {
			t.Errorf("qsdev require version %q is not a valid semver", r.Mod.Version)
		}
	}
}

// TestScaffoldCmd_LdflagsStampFrameworkVersionPackage ensures the Makefile and
// goreleaser stamp the package the framework actually reads its version from;
// a -X on a nonexistent package is silently ignored and leaves "dev".
func TestScaffoldCmd_LdflagsStampFrameworkVersionPackage(t *testing.T) {
	t.Parallel()
	if got, want := reflect.TypeOf(version.BuildInfo{}).PkgPath(), instance.VersionPackage; got != want {
		t.Fatalf("instance.VersionPackage = %q, but the version package is %q", want, got)
	}
	out := scaffoldInto(t, "ldcheck")
	tests := []struct {
		file string
		want []string
	}{
		{"Makefile", []string{"VERSION_PKG := " + instance.VersionPackage, "-X $(VERSION_PKG).version=", "-X $(VERSION_PKG).commit="}},
		{".goreleaser.yaml", []string{"-X " + instance.VersionPackage + ".version=", "-X " + instance.VersionPackage + ".commit="}},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			t.Parallel()
			content, err := os.ReadFile(filepath.Join(out, tt.file))
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range tt.want {
				if !strings.Contains(string(content), w) {
					t.Errorf("%s missing %q:\n%s", tt.file, w, content)
				}
			}
			if strings.Contains(string(content), "github.com/acme/ldcheck/internal/version") {
				t.Errorf("%s stamps the scaffold's own (nonexistent) version package:\n%s", tt.file, content)
			}
		})
	}
}

// TestScaffoldCmd_MainGoCompiles type-checks the generated main.go against the
// current framework API (via a build overlay inside this module), so the
// scaffold cannot drift from the exported wiring cmd/qsdev uses.
func TestScaffoldCmd_MainGoCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a binary")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	out := scaffoldInto(t, "compilecheck")
	mainSrc := filepath.Join(out, "cmd", "compilecheck", "main.go")
	content, err := os.ReadFile(mainSrc)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"instance.RegisterFrameworkAdapters()", "instance.ApplyBuildVersion()", "instance.UseProjectDefaults()", "devinit.WithPlanPreview(true)"} {
		if !strings.Contains(string(content), w) {
			t.Errorf("scaffold main.go missing %q", w)
		}
	}
	// The overlay build below runs inside this module, where internal imports
	// resolve; a downstream module cannot import them, so reject them here.
	parsed, err := parser.ParseFile(token.NewFileSet(), mainSrc, content, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range parsed.Imports {
		if strings.Contains(imp.Path.Value, "/internal/") {
			t.Errorf("scaffold main.go imports %s, which a downstream module cannot import", imp.Path.Value)
		}
	}

	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	virtualMain := filepath.Join(moduleRoot, "internal", "scaffoldcompilecheck", "main.go")
	overlayJSON, err := json.Marshal(map[string]map[string]string{"Replace": {virtualMain: mainSrc}})
	if err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(t.TempDir(), "overlay.json")
	if err := os.WriteFile(overlayPath, overlayJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	build := exec.Command(goBin, "build", "-overlay", overlayPath,
		"-o", filepath.Join(t.TempDir(), "compilecheck"), "./internal/scaffoldcompilecheck")
	build.Dir = moduleRoot
	build.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=vendor")
	if outBytes, err := build.CombinedOutput(); err != nil {
		t.Fatalf("scaffolded main.go does not compile against the framework: %v\n%s", err, outBytes)
	}
}
