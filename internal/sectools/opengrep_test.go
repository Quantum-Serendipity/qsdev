package sectools_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateOpengrepConfigYaml_Structure(t *testing.T) {
	t.Parallel()

	f, err := sectools.GenerateOpengrepConfigYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepConfigYaml() error: %v", err)
	}
	if f.Path != ".opengrep/config.yaml" {
		t.Errorf("Path = %q, want %q", f.Path, ".opengrep/config.yaml")
	}
	if f.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0o644)
	}
	if f.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", f.Strategy)
	}
	if f.Owner != "opengrep" {
		t.Errorf("Owner = %q, want %q", f.Owner, "opengrep")
	}
}

func TestGenerateOpengrepConfigYaml_YAMLContent(t *testing.T) {
	t.Parallel()

	f, err := sectools.GenerateOpengrepConfigYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepConfigYaml() error: %v", err)
	}
	content := string(f.Content)

	for _, key := range []string{"rules:", "exclude:", "severity:", "timeout:"} {
		if !strings.Contains(content, key) {
			t.Errorf("content should contain %q key", key)
		}
	}
}

func TestGenerateOpengrepConfigYaml_RulePaths(t *testing.T) {
	t.Parallel()

	f, err := sectools.GenerateOpengrepConfigYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepConfigYaml() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "rules/core") {
		t.Error("content should reference rules/core path")
	}
}

func TestGenerateOpengrepConfigYaml_PathExclusions(t *testing.T) {
	t.Parallel()

	f, err := sectools.GenerateOpengrepConfigYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepConfigYaml() error: %v", err)
	}
	content := string(f.Content)

	for _, path := range []string{"vendor/", "node_modules/", "dist/", ".devenv/", "__pycache__/"} {
		if !strings.Contains(content, path) {
			t.Errorf("content should exclude path %q", path)
		}
	}
}

func TestGenerateOpengrepFiles_DeliversRules(t *testing.T) {
	t.Parallel()

	files, err := sectools.GenerateOpengrepFiles(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepFiles() error: %v", err)
	}

	var config *types.GeneratedFile
	var ruleFiles []types.GeneratedFile
	for i := range files {
		switch {
		case files[i].Path == ".opengrep/config.yaml":
			config = &files[i]
		case strings.HasPrefix(files[i].Path, ".opengrep/rules/core/"):
			ruleFiles = append(ruleFiles, files[i])
		}
	}

	if config == nil {
		t.Fatal("expected a .opengrep/config.yaml among generated files")
	}
	if len(ruleFiles) == 0 {
		t.Fatal("expected embedded rule files to be delivered, got none (rules never reach the user's project)")
	}

	// The path the config references must be the delivered location, not the
	// source-only rules/core path that never exists in a user project.
	cfg := string(config.Content)
	if !strings.Contains(cfg, ".opengrep/rules/core") {
		t.Errorf("config should point rules at the delivered .opengrep/rules/core; got:\n%s", cfg)
	}
	if strings.Contains(cfg, "  - rules/core\n") {
		t.Error("config must not point at the source-only rules/core path")
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

func TestGenerateOpengrepConfigYaml_Defaults(t *testing.T) {
	t.Parallel()

	f, err := sectools.GenerateOpengrepConfigYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepConfigYaml() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "severity: warning") {
		t.Error("default severity should be warning")
	}
	if !strings.Contains(content, "timeout: 300") {
		t.Error("default timeout should be 300")
	}
}
