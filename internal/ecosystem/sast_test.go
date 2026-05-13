package ecosystem_test

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"

	// Import all Tier 1 modules to ensure they are registered.
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/docker"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/dotnet"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/golang"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/java"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/javascript"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/python"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/rust"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/terraform"
)

func TestTier1ModulesImplementSASTModule(t *testing.T) {
	tier1Names := []string{
		"go", "javascript", "python", "rust",
		"java", "dotnet", "docker", "terraform",
	}

	reg := ecosystem.DefaultRegistry()

	for _, name := range tier1Names {
		mod, ok := reg.ByName(name)
		if !ok {
			t.Errorf("module %q not registered in default registry", name)
			continue
		}
		sast, ok := mod.(ecosystem.SASTModule)
		if !ok {
			t.Errorf("module %q does not implement SASTModule", name)
			continue
		}
		rules := sast.SemgrepRuleSets()
		if len(rules) == 0 {
			t.Errorf("module %q returned empty SemgrepRuleSets", name)
		}
	}
}

func TestSASTModuleRuleSetsContent(t *testing.T) {
	tests := []struct {
		moduleName string
		wantRules  []string
	}{
		{"go", []string{"p/golang", "p/owasp-top-ten"}},
		{"javascript", []string{"p/typescript", "p/javascript", "p/react", "p/nextjs", "p/owasp-top-ten", "p/xss"}},
		{"python", []string{"p/python", "p/django", "p/flask", "p/owasp-top-ten"}},
		{"rust", []string{"p/rust", "p/owasp-top-ten"}},
		{"java", []string{"p/java", "p/kotlin", "p/spring", "p/owasp-top-ten"}},
		{"dotnet", []string{"p/csharp", "p/owasp-top-ten"}},
		{"docker", []string{"p/dockerfile"}},
		{"terraform", []string{"p/terraform", "p/terraform-aws"}},
	}

	reg := ecosystem.DefaultRegistry()

	for _, tt := range tests {
		t.Run(tt.moduleName, func(t *testing.T) {
			mod, ok := reg.ByName(tt.moduleName)
			if !ok {
				t.Fatalf("module %q not registered", tt.moduleName)
			}
			sast := mod.(ecosystem.SASTModule)
			got := sast.SemgrepRuleSets()
			if len(got) != len(tt.wantRules) {
				t.Fatalf("SemgrepRuleSets() returned %d rules, want %d", len(got), len(tt.wantRules))
			}
			for i, want := range tt.wantRules {
				if got[i] != want {
					t.Errorf("SemgrepRuleSets()[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}
