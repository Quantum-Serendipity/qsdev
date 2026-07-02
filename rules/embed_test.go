package rules_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/rules"
)

func TestCoreRuleFiles(t *testing.T) {
	t.Parallel()

	files, err := rules.CoreRuleFiles()
	if err != nil {
		t.Fatalf("CoreRuleFiles() error: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected embedded core rule files, got none")
	}

	var sawRuleID bool
	for _, f := range files {
		if f.RelPath == "" {
			t.Error("rule file has empty RelPath")
		}
		// Intentionally-vulnerable fixtures and the index must be excluded.
		if strings.Contains(f.RelPath, "testdata/") {
			t.Errorf("testdata fixture must be excluded: %s", f.RelPath)
		}
		if f.RelPath == "manifest.yaml" {
			t.Error("manifest.yaml (index, not a loadable rule) must be excluded")
		}
		if !strings.HasSuffix(f.RelPath, ".yaml") && !strings.HasSuffix(f.RelPath, ".yml") {
			t.Errorf("non-rule file included: %s", f.RelPath)
		}
		if len(f.Content) == 0 {
			t.Errorf("rule file %s has empty content", f.RelPath)
		}
		if strings.Contains(string(f.Content), "qsdev.core.") {
			sawRuleID = true
		}
	}
	if !sawRuleID {
		t.Error("expected at least one qsdev.core.* rule ID among embedded files")
	}
}
