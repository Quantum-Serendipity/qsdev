package sectools_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
	"github.com/Quantum-Serendipity/qsdev/rules"
)

// TestGenerateOpengrepFiles_NoConfigFile is the F217 regression: OpenGrep has
// no project config file, and the invented .opengrep/config.yaml (top-level
// exclude/severity/timeout, a directory under rules:) fails its parser. The
// rule library is passed straight to `opengrep scan --config`, so nothing but
// the rules and the package derivation may be delivered.
func TestGenerateOpengrepFiles_NoConfigFile(t *testing.T) {
	t.Parallel()

	files, err := sectools.GenerateOpengrepFiles(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepFiles() error: %v", err)
	}
	for _, f := range files {
		switch {
		case f.Path == ".opengrep/nix/default.nix":
		case strings.HasPrefix(f.Path, rules.ProjectCoreDir+"/"):
		default:
			t.Errorf("unexpected generated file %s (only the rule library and the nix derivation are delivered)", f.Path)
		}
	}
}

func TestGenerateOpengrepFiles_DeliversRules(t *testing.T) {
	t.Parallel()

	files, err := sectools.GenerateOpengrepFiles(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepFiles() error: %v", err)
	}

	var ruleFiles []types.GeneratedFile
	for i := range files {
		if strings.HasPrefix(files[i].Path, ".opengrep/rules/core/") {
			ruleFiles = append(ruleFiles, files[i])
		}
	}
	if len(ruleFiles) == 0 {
		t.Fatal("expected embedded rule files to be delivered, got none (rules never reach the user's project)")
	}

	for _, rf := range ruleFiles {
		if rf.Owner != "opengrep" {
			t.Errorf("rule file %s owner = %q, want opengrep", rf.Path, rf.Owner)
		}
		if len(rf.Content) == 0 {
			t.Errorf("rule file %s has empty content", rf.Path)
		}
		// Intentionally-vulnerable test fixtures must never ship to users.
		if strings.Contains(rf.Path, "/testdata/") {
			t.Errorf("testdata fixture must not be delivered: %s", rf.Path)
		}
		if !strings.HasSuffix(rf.Path, ".yaml") && !strings.HasSuffix(rf.Path, ".yml") {
			t.Errorf("non-rule file delivered: %s", rf.Path)
		}
	}

	var found bool
	for _, rf := range ruleFiles {
		if strings.Contains(string(rf.Content), "qsdev.core.") {
			found = true
			break
		}
	}
	if !found {
		t.Error("delivered rule files should contain qsdev.core.* rule IDs")
	}
}

// TestGenerateOpengrepFiles_DeliversNixDerivation checks that the OpenGrep
// package derivation the devenv.nix package list imports
// (./.opengrep/nix) is written into the project, pinned to real release
// asset hashes rather than placeholders.
func TestGenerateOpengrepFiles_DeliversNixDerivation(t *testing.T) {
	t.Parallel()

	files, err := sectools.GenerateOpengrepFiles(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepFiles() error: %v", err)
	}
	var drv *types.GeneratedFile
	for i := range files {
		if files[i].Path == ".opengrep/nix/default.nix" {
			drv = &files[i]
		}
	}
	if drv == nil {
		t.Fatal("expected .opengrep/nix/default.nix among generated files")
	}
	if drv.Owner != "opengrep" || drv.Mode != 0o644 || drv.Strategy != types.Overwrite {
		t.Errorf("derivation file = owner %q mode %#o strategy %v, want opengrep 0644 Overwrite",
			drv.Owner, drv.Mode, drv.Strategy)
	}

	content := string(drv.Content)
	tests := []struct {
		name, want string
	}{
		{"fetches a release asset", "https://github.com/opengrep/opengrep/releases/download/v${version}/${src.asset}"},
		{"x86_64-linux asset", `asset = "opengrep_manylinux_x86";`},
		{"aarch64-linux asset", `asset = "opengrep_manylinux_aarch64";`},
		{"aarch64-darwin asset", `asset = "opengrep_osx_arm64";`},
		{"x86_64-darwin asset", `asset = "opengrep_osx_x86";`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(content, tt.want) {
				t.Errorf("derivation should contain %q", tt.want)
			}
		})
	}
	if strings.Contains(content, "fakeHash") || strings.Contains(content, "lib.fakeSha256") {
		t.Error("derivation must pin real hashes, not placeholders")
	}
	if got, want := strings.Count(content, `hash = "sha256-`), 4; got != want {
		t.Errorf("derivation has %d SRI sha256 hashes, want %d (one per platform)", got, want)
	}
}
