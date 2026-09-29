package devenv_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"

	// Register every ecosystem module with the DefaultRegistry via init().
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
)

// renderWithModules renders devenv.nix for the given languages using the real
// ecosystem modules.
func renderWithModules(t *testing.T, langs ...types.LanguageChoice) string {
	t.Helper()
	got, err := devenv.GenerateDevenvNix(types.WizardAnswers{Languages: langs}, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("GenerateDevenvNix: %v", err)
	}
	return string(got.Content)
}

// customHookBlock returns the rendered `<id> = { ... };` attribute set of a
// custom hook, or "" when absent.
func customHookBlock(content, id string) string {
	start := strings.Index(content, "\n    "+id+" = {\n")
	if start < 0 {
		return ""
	}
	end := strings.Index(content[start:], "\n    };")
	if end < 0 {
		return ""
	}
	return content[start : start+end]
}

// shellcheckScoped is the always-on shellcheck hook as rendered with its zsh
// exclusion (W122); the formatter folds the dotted override into the block.
const shellcheckScoped = "    shellcheck = {\n      enable = true;\n      exclude_types = [ \"zsh\" ];\n    };"

// TestGenerateDevenvNix_HookScoping verifies the rendered git-hooks scoping:
// custom hooks carry types_or/exclude_types and pin their package, and
// built-in hooks replace the git-hooks.nix file-type defaults.
func TestGenerateDevenvNix_HookScoping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		langs       []types.LanguageChoice
		contains    []string
		notContains []string
		blocks      map[string][]string // custom hook ID -> substrings of its block
	}{
		{
			// W103: types is an AND filter, so types = [ "c" "c++" ] only
			// matched headers. W117: clang-format must not reformat JSON/JS.
			name:     "cpp",
			langs:    []types.LanguageChoice{{Name: "cpp"}},
			contains: []string{"    clang-format = {\n      enable = true;\n      types_or = [ \"c\" \"c++\" \"cuda\" \"objective-c\" ];\n    };"},
			blocks: map[string][]string{
				"cppcheck": {`types_or = [ "c" "c++" ];`, "package = pkgs.cppcheck;"},
			},
			notContains: []string{`types = [ "c" "c++" ];`},
		},
		{
			// W122: the always-on shellcheck is scoped even without the
			// shell ecosystem, since zsh files carry the "shell" tag.
			name:     "security shellcheck without shell module",
			langs:    []types.LanguageChoice{{Name: "go"}},
			contains: []string{shellcheckScoped},
		},
		{
			name:  "shell module",
			langs: []types.LanguageChoice{{Name: "shell"}},
			contains: []string{
				"    shfmt = {\n      enable = true;\n      exclude_types = [ \"zsh\" ];\n    };",
				shellcheckScoped,
			},
		},
		{
			// W170: a custom hook sharing a git-hooks.nix built-in ID must
			// pin its package, or upstream's default (php84Packages.phpstan,
			// now a throw) is evaluated. W106: phpstan gets file operands and
			// phpcs runs PSR-12 when the project has no ruleset.
			name:  "php without phpcs ruleset",
			langs: []types.LanguageChoice{{Name: "php"}},
			blocks: map[string][]string{
				"phpstan": {"package = pkgs.phpstan;", "pass_filenames = true;"},
				"phpcs":   {"package = pkgs.phpPackages.php-codesniffer;", "--standard=PSR12"},
			},
			notContains: []string{"phpcs.enable = true;"},
		},
		{
			name:        "php with phpcs ruleset",
			langs:       []types.LanguageChoice{{Name: "php", Extras: []string{"phpcs_config=true"}}},
			contains:    []string{"phpcs.enable = true;"},
			notContains: []string{"--standard=PSR12"},
		},
		{
			// W105: the generated header names arguments it does not use.
			name:  "nix",
			langs: []types.LanguageChoice{{Name: "nix"}},
			blocks: map[string][]string{
				"deadnix": {"deadnix --fail --no-lambda-pattern-names", "package = pkgs.deadnix;"},
			},
		},
		{
			// W118: a bundled rubocop runs through Bundler.
			name:  "ruby bundling rubocop",
			langs: []types.LanguageChoice{{Name: "ruby", Extras: []string{"bundled_rubocop=true"}}},
			blocks: map[string][]string{
				"rubocop": {`entry = "bundle exec rubocop --autocorrect --force-exclusion";`},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content := renderWithModules(t, tt.langs...)
			for _, s := range tt.contains {
				requireContains(t, content, s)
			}
			for _, s := range tt.notContains {
				if strings.Contains(content, s) {
					t.Errorf("output unexpectedly contains %q\n\n%s", s, content)
				}
			}
			for id, subs := range tt.blocks {
				block := customHookBlock(content, id)
				if block == "" {
					t.Errorf("no custom hook block for %q\n\n%s", id, content)
					continue
				}
				for _, s := range subs {
					if !strings.Contains(block, s) {
						t.Errorf("hook %q block missing %q:\n%s", id, s, block)
					}
				}
			}
			if n := strings.Count(content, "\n    shellcheck = {") + strings.Count(content, "shellcheck.exclude_types"); n > 1 {
				t.Errorf("shellcheck scoping rendered %d times; a repeated attribute fails evaluation", n)
			}
		})
	}
}

// TestCustomHooks_TypesAreSingleTag guards against listing alternatives in
// Types: pre-commit/prek treat types as an AND filter, so a custom hook with
// types = [ "c" "c++" ] only ever sees files tagged with both (headers).
// Alternatives belong in TypesOr.
func TestCustomHooks_TypesAreSingleTag(t *testing.T) {
	t.Parallel()
	for _, mod := range ecosystem.DefaultRegistry().All() {
		for _, hook := range mod.PreCommitHooks(ecosystem.ModuleConfig{}) {
			if !hook.BuiltIn && len(hook.Types) > 1 {
				t.Errorf("module %q custom hook %q has Types %v; types is an AND filter, use TypesOr for alternatives",
					mod.Name(), hook.ID, hook.Types)
			}
		}
	}
}

// TestCustomHooks_BuiltInIDCollisionsPinPackage verifies every custom hook
// whose ID is also a git-hooks.nix built-in declares NixPackage. The two
// definitions merge, so without a rendered `package` upstream's default
// package is evaluated into the shell even though qsdev's entry names another
// binary; when that default is removed from nixpkgs (phpstan), the whole
// devenv fails to evaluate.
func TestCustomHooks_BuiltInIDCollisionsPinPackage(t *testing.T) {
	t.Parallel()
	for _, mod := range ecosystem.DefaultRegistry().All() {
		for _, hook := range mod.PreCommitHooks(ecosystem.ModuleConfig{}) {
			if hook.BuiltIn || !slices.Contains(gitHooksBuiltInIDs, hook.ID) {
				continue
			}
			if hook.NixPackage == "" {
				t.Errorf("module %q custom hook %q shares a git-hooks.nix built-in ID but declares no NixPackage; "+
					"upstream's default package would be evaluated", mod.Name(), hook.ID)
			}
		}
	}
}
