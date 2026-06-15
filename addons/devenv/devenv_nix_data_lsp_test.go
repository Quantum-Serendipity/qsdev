package devenv

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// lspFragment returns the NixFragment of the synthetic "LSP servers" language
// fragment, or "" if none was emitted. It also reports whether the fragment was
// present.
func lspFragment(t *testing.T, data *DevenvNixTemplateData) (string, bool) {
	t.Helper()
	for _, f := range data.LanguageFragments {
		if f.DisplayName == lspDisplayName {
			return f.NixFragment, true
		}
	}
	return "", false
}

// registerMockLanguages registers an empty-fragment MockModule per language so
// BuildDevenvNixData does not error on unknown modules. The centralized LSP
// section is driven by the lsp registry, not these modules.
func registerMockLanguages(t *testing.T, reg *ecosystem.Registry, names ...string) {
	t.Helper()
	for _, name := range names {
		if err := reg.Register(&ecosystem.MockModule{
			NameVal:        name,
			DisplayNameVal: name,
			TierVal:        1,
		}); err != nil {
			t.Fatalf("registering mock module %q: %v", name, err)
		}
	}
}

func TestCollectLSPSection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		languages []string
		// wantInFragment are substrings that must appear in the LSP fragment.
		wantInFragment []string
		// wantNotInFragment are substrings that must NOT appear in the fragment.
		wantNotInFragment []string
		// wantPackages are packages that must be present in data.Packages.
		wantPackages []string
		// wantNotPackages are packages that must NOT be present.
		wantNotPackages []string
	}{
		{
			name:      "empty languages still provisions nixd",
			languages: nil,
			wantInFragment: []string{
				"languages.nix.enable = true;",
				"languages.nix.lsp.enable = true;",
			},
		},
		{
			name:      "default-on go yields enable line",
			languages: []string{"go"},
			wantInFragment: []string{
				"languages.go.lsp.enable = true;",
				"languages.nix.lsp.enable = true;",
			},
		},
		{
			name:      "default-off haskell yields disable line",
			languages: []string{"haskell"},
			wantInFragment: []string{
				"languages.haskell.lsp.enable = false;",
				// The opt-in hint is commented out, never active.
				"# languages.haskell.lsp.enable = true;",
			},
			wantNotInFragment: []string{
				// No active (uncommented) enable line for the disabled server.
				"  languages.haskell.lsp.enable = true;",
			},
		},
		{
			name:      "override ruby yields package override",
			languages: []string{"ruby"},
			wantInFragment: []string{
				"languages.ruby.lsp.enable = true;",
				"languages.ruby.lsp.package = pkgs.ruby-lsp;",
			},
		},
		{
			name:      "package-list container adds package, emits no fragment line",
			languages: []string{"container"},
			wantNotInFragment: []string{
				"languages.container",
				"docker-language-server.lsp",
			},
			wantPackages: []string{"docker-language-server"},
		},
		{
			name:      "selecting nix does not duplicate the always-on enable",
			languages: []string{"nix"},
			wantInFragment: []string{
				"languages.nix.lsp.enable = true;",
			},
		},
		{
			name:      "polyglot go+ruby+haskell+container",
			languages: []string{"go", "ruby", "haskell", "container"},
			wantInFragment: []string{
				"languages.go.lsp.enable = true;",
				"languages.ruby.lsp.package = pkgs.ruby-lsp;",
				"languages.haskell.lsp.enable = false;",
				"languages.nix.lsp.enable = true;",
			},
			wantPackages: []string{"docker-language-server"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg := ecosystem.NewRegistry()
			registerMockLanguages(t, reg, tt.languages...)

			answers := types.WizardAnswers{}
			for _, name := range tt.languages {
				answers.Languages = append(answers.Languages, types.LanguageChoice{Name: name})
			}

			data, err := BuildDevenvNixData(answers, reg)
			if err != nil {
				t.Fatalf("BuildDevenvNixData: %v", err)
			}

			frag, ok := lspFragment(t, data)
			if !ok {
				t.Fatalf("no %q language fragment emitted", lspDisplayName)
			}

			for _, want := range tt.wantInFragment {
				if !strings.Contains(frag, want) {
					t.Errorf("LSP fragment missing %q\ngot:\n%s", want, frag)
				}
			}
			for _, notWant := range tt.wantNotInFragment {
				if strings.Contains(frag, notWant) {
					t.Errorf("LSP fragment unexpectedly contains %q\ngot:\n%s", notWant, frag)
				}
			}
			for _, want := range tt.wantPackages {
				if !slices.Contains(data.Packages, want) {
					t.Errorf("data.Packages missing %q; got %v", want, data.Packages)
				}
			}
			for _, notWant := range tt.wantNotPackages {
				if slices.Contains(data.Packages, notWant) {
					t.Errorf("data.Packages unexpectedly contains %q; got %v", notWant, data.Packages)
				}
			}
		})
	}
}

func TestBuildDevenvNixData_NixdAlwaysOn(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()

	data, err := BuildDevenvNixData(types.WizardAnswers{}, reg)
	if err != nil {
		t.Fatalf("BuildDevenvNixData: %v", err)
	}

	frag, ok := lspFragment(t, data)
	if !ok {
		t.Fatal("nixd LSP fragment not emitted for empty languages")
	}
	// languages.nix.enable is required because devenv guards
	// languages.nix.lsp.package behind lib.mkIf languages.nix.enable.
	for _, want := range []string{"languages.nix.enable = true;", "languages.nix.lsp.enable = true;"} {
		if !strings.Contains(frag, want) {
			t.Errorf("nixd fragment missing %q\ngot:\n%s", want, frag)
		}
	}
}

func TestBuildDevenvNixData_JqInPackages(t *testing.T) {
	t.Parallel()
	reg := ecosystem.NewRegistry()

	data, err := BuildDevenvNixData(types.WizardAnswers{}, reg)
	if err != nil {
		t.Fatalf("BuildDevenvNixData: %v", err)
	}
	if !slices.Contains(data.Packages, "jq") {
		t.Errorf("data.Packages missing %q (lsp-first-guard hook dependency); got %v", "jq", data.Packages)
	}
}

func TestBuildDevenvNixData_LSPEnforcementEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		enforcement string
		want        string
	}{
		{name: "unset defaults to block", enforcement: "", want: "block"},
		{name: "explicit warn", enforcement: "warn", want: "warn"},
		{name: "explicit off", enforcement: "off", want: "off"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := ecosystem.NewRegistry()
			answers := types.WizardAnswers{
				LSP: types.LSPSettings{Enforcement: tt.enforcement},
			}
			data, err := BuildDevenvNixData(answers, reg)
			if err != nil {
				t.Fatalf("BuildDevenvNixData: %v", err)
			}
			got, ok := data.EnvVars["QSDEV_LSP_ENFORCEMENT"]
			if !ok {
				t.Fatalf("EnvVars missing QSDEV_LSP_ENFORCEMENT; got %v", data.EnvVars)
			}
			if got != tt.want {
				t.Errorf("QSDEV_LSP_ENFORCEMENT = %q, want %q", got, tt.want)
			}
		})
	}
}
