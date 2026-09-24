package sectools_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"

	// Import modules so they register with the default registry.
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/container"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cpp"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/dotnet"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/golang"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/java"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/javascript"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/python"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/rust"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/terraform"
)

func TestGenerateSemgrepIgnore_Metadata(t *testing.T) {
	t.Parallel()
	f, err := sectools.GenerateSemgrepIgnore(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateSemgrepIgnore() error: %v", err)
	}
	if f.Path != ".semgrepignore" {
		t.Errorf("Path = %q, want %q", f.Path, ".semgrepignore")
	}
	if f.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0o644)
	}
	if f.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", f.Strategy)
	}
	if f.Owner != "semgrep" {
		t.Errorf("Owner = %q, want %q", f.Owner, "semgrep")
	}
}

// TestGenerateSemgrepIgnore_Content is the F202 regression: the generated file
// is a gitignore-syntax ignore list, one pattern per line, not the rejected
// .semgrep.yml shape (registry refs under rules: and a paths: key).
func TestGenerateSemgrepIgnore_Content(t *testing.T) {
	t.Parallel()
	f, err := sectools.GenerateSemgrepIgnore(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateSemgrepIgnore() error: %v", err)
	}
	var patterns []string
	for _, line := range strings.Split(strings.TrimRight(string(f.Content), "\n"), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.TrimSpace(line) != line || line == "" {
			t.Errorf("pattern line %q has surrounding whitespace or is empty", line)
		}
		patterns = append(patterns, line)
	}

	tests := []struct {
		pattern string
		present bool
	}{
		{"vendor/", true},
		{"node_modules/", true},
		{"dist/", true},
		{".devenv/", true},
		{"testdata/", true},
		// Python build metadata directories are named <pkg>.egg-info.
		{"*.egg-info/", true},
		// The project's own rules are --config inputs, not scan targets.
		{".semgrep/", true},
		{".egg-info/", false},
		{"rules:", false},
		{"paths:", false},
		{"exclude:", false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			t.Parallel()
			if got := slices.Contains(patterns, tt.pattern); got != tt.present {
				t.Errorf("pattern %q present = %v, want %v (patterns: %q)", tt.pattern, got, tt.present, patterns)
			}
		})
	}
	for _, p := range patterns {
		if strings.HasPrefix(p, "p/") || strings.HasPrefix(p, "- ") {
			t.Errorf("pattern %q looks like a rule pack or YAML list item; the security-scan task passes rule packs", p)
		}
	}
}

// registryRuleSetRe matches a Semgrep registry rule pack reference. The
// security-scan task splices these unquoted into a shell command line, so they
// must be plain registry refs.
var registryRuleSetRe = regexp.MustCompile(`^p/[a-z0-9][a-z0-9-]*$`)

// TestSemgrepRuleSets_AreShellSafeRegistryRefs guards the rule packs every
// SASTModule declares: each is passed as `semgrep --config <ref>`.
func TestSemgrepRuleSets_AreShellSafeRegistryRefs(t *testing.T) {
	t.Parallel()
	var checked int
	for _, mod := range ecosystem.DefaultRegistry().All() {
		sast, ok := mod.(ecosystem.SASTModule)
		if !ok {
			continue
		}
		t.Run(mod.Name(), func(t *testing.T) {
			t.Parallel()
			sets := sast.SemgrepRuleSets()
			if len(sets) == 0 {
				t.Error("SASTModule declares no rule sets")
			}
			for _, rs := range sets {
				if !registryRuleSetRe.MatchString(rs) {
					t.Errorf("rule set %q is not a registry ref matching %s", rs, registryRuleSetRe)
				}
			}
		})
		checked++
	}
	if checked == 0 {
		t.Fatal("no SASTModule registered")
	}
}
