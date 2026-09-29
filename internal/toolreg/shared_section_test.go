package toolreg

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestSharedContent_KeyedPerFile is the F027 regression at the registry
// level: catalog section_content belongs to the file that declares it
// (CLAUDE.md), never to another file that reuses the section ID (devenv.nix).
func TestSharedContent_KeyedPerFile(t *testing.T) {
	t.Parallel()
	reg := DefaultRegistry()
	tests := []struct {
		tool    string
		present []SharedSection
		absent  []SharedSection
	}{
		{
			tool:    "gitleaks",
			present: []SharedSection{{Path: "CLAUDE.md", SectionID: "gitleaks"}},
			absent:  []SharedSection{{Path: DevenvNixFile, SectionID: "gitleaks"}},
		},
		{
			tool:    "opengrep",
			present: []SharedSection{{Path: "CLAUDE.md", SectionID: "opengrep"}},
			absent:  []SharedSection{{Path: DevenvNixFile, SectionID: "opengrep"}},
		},
		{
			tool:    "starship-integration",
			present: []SharedSection{{Path: DevenvNixFile, SectionID: "starship"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			t.Parallel()
			tool, ok := reg.ByName(tt.tool)
			if !ok {
				t.Fatalf("%s not registered", tt.tool)
			}
			for _, k := range tt.present {
				if _, ok := tool.SharedContent[k]; !ok {
					t.Errorf("missing content for %v", k)
				}
			}
			for _, k := range tt.absent {
				if _, ok := tool.SharedContent[k]; ok {
					t.Errorf("unexpected content for %v", k)
				}
			}
		})
	}
}

func TestSharedSectionsFor_DevenvNix(t *testing.T) {
	t.Parallel()
	reg := DefaultRegistry()
	answers := types.WizardAnswers{EnabledTools: map[string]bool{
		"starship-integration": true,
		"gitleaks":             true, // declares a devenv.nix section but has no Nix content
		"commit-ticket":        false,
	}}
	sections, err := reg.SharedSectionsFor(DevenvNixFile, answers)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range sections {
		got = append(got, s.Tool.Name)
		if strings.HasPrefix(strings.TrimSpace(string(s.Content)), "- ") {
			t.Errorf("%s: Markdown content in devenv.nix: %q", s.Tool.Name, s.Content)
		}
	}
	if strings.Join(got, ",") != "starship-integration" {
		t.Errorf("sections from %v, want only starship-integration", got)
	}
}

func TestTool_OwnsExclusively(t *testing.T) {
	t.Parallel()
	tool := &Tool{OwnedFiles: []FileOwnership{
		{Path: ".opengrep/nix/default.nix", Ownership: Exclusive},
		{Path: ".opengrep/rules/core", Ownership: Exclusive},
		{Path: "CLAUDE.md", Ownership: Shared, SectionID: "x"},
	}}
	tests := []struct {
		path string
		want bool
	}{
		{".opengrep/nix/default.nix", true},
		{".opengrep/rules/core", true},
		{".opengrep/rules/core/auth/jwt.yaml", true},
		{".opengrep/rules/corex/a.yaml", false},
		{".opengrep/other.yaml", false},
		{"CLAUDE.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if got := tool.OwnsExclusively(tt.path); got != tt.want {
				t.Errorf("OwnsExclusively(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
