package devenv_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"

	// Register every ecosystem module with the DefaultRegistry via init().
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
)

// gitHooksBuiltInIDs is the authoritative set of built-in hook identifiers
// shipped by git-hooks.nix (github:cachix/git-hooks.nix). A hook declared with
// BuiltIn:true is rendered as "<id>.enable = true" in devenv.nix, which only
// evaluates when git-hooks.nix defines a built-in hook of that exact ID. Any
// BuiltIn:true hook whose ID is absent here breaks `devenv` evaluation for that
// ecosystem, so this list guards against that whole class of defect.
//
// Source: the config.hooks attribute set in modules/hooks.nix of git-hooks.nix
// at rev 61ab0e80d9c7ab14c256b5b453d8b3fb0189ba0a (the rev pinned in
// devenv.lock). To regenerate after bumping the git-hooks input:
//
//	src=$(nix eval --impure --raw --expr 'toString (builtins.fetchTree {
//	  type = "github"; owner = "cachix"; repo = "git-hooks.nix";
//	  rev = "<rev-from-devenv.lock>"; })')
//	sed -n '/config.hooks = mapAttrs/,$p' "$src/modules/hooks.nix" \
//	  | grep -oE '^      [A-Za-z0-9_.-]+ =$|^      [A-Za-z0-9_.-]+ =\{' \
//	  | sed -E 's/ =\{//; s/ =$//; s/^      //' | sort -u
var gitHooksBuiltInIDs = []string{
	"actionlint",
	"action-validator",
	"alejandra",
	"annex",
	"ansible-lint",
	"autoflake",
	"bats",
	"beautysh",
	"biome",
	"black",
	"cabal2nix",
	"cabal-fmt",
	"cabal-gild",
	"cargo-check",
	"cargo-sort",
	"chart-testing",
	"check-added-large-files",
	"check-builtin-literals",
	"check-case-conflicts",
	"check-docstring-first",
	"check-executables-have-shebangs",
	"check-json",
	"check-merge-conflicts",
	"check-python",
	"check-shebang-scripts-are-executable",
	"check-symlinks",
	"check-toml",
	"check-vcs-permalinks",
	"check-xml",
	"check-yaml",
	"chktex",
	"circleci",
	"clang-format",
	"clippy",
	"cljfmt",
	"cmake-format",
	"commitizen",
	"cspell",
	"deadnix",
	"denofmt",
	"denolint",
	"detect-aws-credentials",
	"detect-private-keys",
	"eclint",
	"editorconfig-checker",
	"elm-format",
	"elm-review",
	"elm-test",
	"end-of-file-fixer",
	"eslint",
	"fix-byte-order-marker",
	"fix-encoding-pragma",
	"flake8",
	"flynt",
	"forbid-new-submodules",
	"fourmolu",
	"gofmt",
	"golines",
	"govet",
	"hadolint",
	"headache",
	"hindent",
	"hledger-fmt",
	"hlint",
	"hpack",
	"html-tidy",
	"hunspell",
	"isort",
	"juliaformatter",
	"keep-sorted",
	"lacheck",
	"latexindent",
	"luacheck",
	"lua-ls",
	"markdownlint",
	"mdl",
	"mdsh",
	"mypy",
	"name-tests-test",
	"nbstripout",
	"nil",
	"nixf-diagnose",
	"nixfmt",
	"nixfmt-classic",
	"nixfmt-rfc-style",
	"nixpkgs-fmt",
	"no-commit-to-branch",
	"nufmt",
	"ocp-indent",
	"opam-lint",
	"openapi-spec-validator",
	"ormolu",
	"oxfmt",
	"oxlint",
	"panache-format",
	"panache-lint",
	"phpcbf",
	"phpcs",
	"php-cs-fixer",
	"phpstan",
	"prettier",
	"pretty-format-json",
	"promtool-rules",
	"proselint",
	"psalm",
	"purs-tidy",
	"pylint",
	"pyright",
	"python-debug-statements",
	"pyupgrade",
	"regal",
	"reuse",
	"revive",
	"ripsecrets",
	"rome",
	"ruff",
	"ruff-format",
	"rumdl",
	"rustfmt",
	"shellcheck",
	"shfmt",
	"single-quoted-strings",
	"sort-file-contents",
	"sort-requirements-txt",
	"sort-simple-yaml",
	"sqlfluff",
	"staticcheck",
	"statix",
	"stylish-haskell",
	"stylua",
	"tagref",
	"taplo",
	"terraform-format",
	"terraform-validate",
	"tflint",
	"topiary",
	"treefmt",
	"trim-trailing-whitespace",
	"trufflehog",
	"typos",
	"yamlfmt",
	"yamllint",
	"zprint",
}

// TestBuiltInHookIDsAreRealGitHooksBuiltIns asserts that every pre-commit hook
// declared with BuiltIn:true across all registered ecosystem modules names an
// ID that git-hooks.nix actually provides as a built-in. This catches the class
// of defect where a module emits "<id>.enable = true" for an ID that does not
// exist, which breaks `devenv` evaluation for that ecosystem.
func TestBuiltInHookIDsAreRealGitHooksBuiltIns(t *testing.T) {
	t.Parallel()

	builtInSet := make(map[string]struct{}, len(gitHooksBuiltInIDs))
	for _, id := range gitHooksBuiltInIDs {
		builtInSet[id] = struct{}{}
	}

	mods := ecosystem.DefaultRegistry().All()
	if len(mods) == 0 {
		t.Fatal("no ecosystem modules registered; expected the modules package import to register them")
	}

	for _, mod := range mods {
		mod := mod
		t.Run(mod.Name(), func(t *testing.T) {
			t.Parallel()
			for _, hook := range mod.PreCommitHooks(ecosystem.ModuleConfig{}) {
				if !hook.BuiltIn {
					continue
				}
				if _, ok := builtInSet[hook.ID]; !ok {
					t.Errorf("module %q declares BuiltIn:true hook %q, but %q is not a git-hooks.nix built-in hook ID; "+
						"convert it to a custom hook (BuiltIn:false, Language:\"system\", NixPackage:\"<pkg>\")",
						mod.Name(), hook.ID, hook.ID)
				}
			}
		})
	}
}

// TestCustomHooksProvisionTheirBinary is the symmetric guard to
// TestBuiltInHookIDsAreRealGitHooksBuiltIns. A custom hook (BuiltIn:false) is
// rendered by collectLanguageFragmentsAndHooks in devenv_nix_data.go as an
// explicit `entry`. Unless the hook declares a NixPackage — which makes the
// generator rewrite the entry to "${pkgs.<pkg>}/bin/<binary>" AND add <pkg> to
// the environment — or a LanguagePackage — which rewrites it to
// "${config.languages.<name>.package}/bin/<binary>", the toolchain the devenv
// language already installs — the entry is emitted as a bare binary name that
// nothing provisions, so the hook fails with command-not-found at commit time
// (BL-P1-8).
//
// The single legitimate alternative is for the owning module to ship the hook's
// binary through DevenvPackages (e.g. container/hadolint, bazel/buildifier),
// which puts it on PATH via the environment rather than the hook entry. This
// test requires every custom hook across every registered module to satisfy one
// of those two conditions, guarding the whole class of unprovisioned-hook defect
// and preventing regressions when new custom hooks are added.
func TestCustomHooksProvisionTheirBinary(t *testing.T) {
	t.Parallel()

	mods := ecosystem.DefaultRegistry().All()
	if len(mods) == 0 {
		t.Fatal("no ecosystem modules registered; expected the modules package import to register them")
	}

	for _, mod := range mods {
		mod := mod
		t.Run(mod.Name(), func(t *testing.T) {
			t.Parallel()

			cfg := ecosystem.ModuleConfig{}

			// DevenvPackages is contributed via the optional PackageProvider
			// interface; modules that don't implement it ship no extra binaries.
			var devenvPkgs []string
			if pp, ok := mod.(ecosystem.PackageProvider); ok {
				devenvPkgs = pp.DevenvPackages(cfg)
			}

			for _, hook := range mod.PreCommitHooks(cfg) {
				if hook.BuiltIn {
					continue
				}
				if hook.NixPackage != "" || hook.LanguagePackage != "" {
					continue
				}

				binary := hookBinary(hook.Entry)
				if binary != "" && packageProvidesBinary(devenvPkgs, binary) {
					continue
				}

				t.Errorf("module %q declares custom hook %q (BuiltIn:false) with entry %q but no NixPackage or LanguagePackage, "+
					"and its binary %q is not shipped via DevenvPackages; the generated hook entry would be a bare "+
					"binary name that nothing provisions, failing with command-not-found at commit time. Set "+
					"NixPackage to the nixpkgs attribute (or LanguagePackage to the devenv language) that provides %q, "+
					"or ship it through DevenvPackages.",
					mod.Name(), hook.ID, hook.Entry, binary, binary)
			}
		})
	}
}

// hookBinary extracts the executable name from a hook entry (its first
// whitespace-separated token), e.g. "cppcheck --error-exitcode=1" -> "cppcheck".
func hookBinary(entry string) string {
	fields := strings.Fields(entry)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// packageProvidesBinary reports whether any package in pkgs plausibly provides
// the given binary. It matches a top-level attribute exactly (e.g. "hadolint")
// as well as a nested attribute whose final path segment is the binary
// (e.g. "lua54Packages.luacheck" provides "luacheck").
func packageProvidesBinary(pkgs []string, binary string) bool {
	for _, p := range pkgs {
		if p == binary {
			return true
		}
		if idx := strings.LastIndex(p, "."); idx >= 0 && p[idx+1:] == binary {
			return true
		}
	}
	return false
}

// TestCustomHooksSharingBuiltInIDPinPackage guards W170: a custom hook whose
// ID is also a git-hooks.nix built-in merges with that definition, so unless
// it sets `package` (rendered from NixPackage or LanguagePackage) upstream's
// default package is evaluated and installed even though qsdev's entry names
// another binary. When that default is a removed nixpkgs attribute
// (phpPackages.phpstan) the whole devenv fails to evaluate.
func TestCustomHooksSharingBuiltInIDPinPackage(t *testing.T) {
	t.Parallel()
	builtIn := make(map[string]bool, len(gitHooksBuiltInIDs))
	for _, id := range gitHooksBuiltInIDs {
		builtIn[id] = true
	}
	for _, mod := range ecosystem.DefaultRegistry().All() {
		for _, hook := range mod.PreCommitHooks(ecosystem.ModuleConfig{}) {
			if !hook.BuiltIn && builtIn[hook.ID] && hook.NixPackage == "" && hook.LanguagePackage == "" {
				t.Errorf("module %q custom hook %q shares a git-hooks.nix built-in ID but sets neither NixPackage "+
					"nor LanguagePackage; upstream's default package would be evaluated", mod.Name(), hook.ID)
			}
		}
	}
}

// hookPackageCase is one devenv.nix render of TestCustomHooksDoNotRedefineBuiltinPackage.
type hookPackageCase struct {
	name    string
	answers types.WizardAnswers
}

// hookPackageCases renders every registered module alone at every hook tier
// (and with no tier), plus version configs that change a hook's package.
func hookPackageCases(t *testing.T) []hookPackageCase {
	t.Helper()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("loading catalog: %v", err)
	}
	tiers := append([]string{""}, cat.HookTierOrder()...)
	var cases []hookPackageCase
	for _, name := range ecosystem.DefaultRegistry().Names() {
		for _, tier := range tiers {
			cases = append(cases, hookPackageCase{
				name:    name + "/tier=" + tier,
				answers: types.WizardAnswers{ProjectName: "hooks", HookTier: tier, Languages: []types.LanguageChoice{{Name: name}}},
			})
		}
	}
	for _, lang := range []types.LanguageChoice{{Name: "zig", Version: "0.13.0"}, {Name: "dotnet", Version: "8"}} {
		cases = append(cases, hookPackageCase{
			name:    lang.Name + "/version=" + lang.Version,
			answers: types.WizardAnswers{ProjectName: "hooks", Languages: []types.LanguageChoice{lang}},
		})
	}
	return cases
}

// customHooksSection returns the rendered custom-hook definitions of
// git-hooks.hooks: from the "Specialized hooks" marker to the attribute set's
// closing brace.
func customHooksSection(t *testing.T, content string) string {
	t.Helper()
	const marker = "    # Specialized hooks (custom definitions)\n"
	start := strings.Index(content, marker)
	if start < 0 {
		t.Fatalf("no custom hooks section in:\n%s", content)
	}
	end := strings.Index(content[start:], "\n  };")
	if end < 0 {
		t.Fatalf("custom hooks section is not closed in:\n%s", content)
	}
	return content[start+len(marker) : start+end]
}

// customHookPackageLine is the prefix every custom hook package definition
// must carry. A devenv language module that also sets the hook's package
// (elixir: git-hooks.hooks.mix-format.package = cfg.package) defines it at
// normal priority (100), or 500 via mkOverrideDefault, and git-hooks.nix
// gives every built-in ID a mkDefault (1000) package. A second definition at
// either priority throws "is defined multiple times"; 999 sits strictly
// between, so the language's pin wins and upstream's default never does.
const customHookPackageLine = "package = lib.mkOverride 999 "

// TestCustomHooksDoNotRedefineBuiltinPackage guards U10-01: every custom hook
// package line in the rendered devenv.nix carries the lib.mkOverride 999
// priority, and every custom hook sharing a git-hooks.nix built-in ID still
// renders one, so upstream's default package is never evaluated.
func TestCustomHooksDoNotRedefineBuiltinPackage(t *testing.T) {
	t.Parallel()
	for _, tc := range hookPackageCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := devenv.GenerateDevenvNix(tc.answers, ecosystem.DefaultRegistry())
			if err != nil {
				t.Fatalf("GenerateDevenvNix: %v", err)
			}
			section := "\n" + customHooksSection(t, string(got.Content))
			for line := range strings.Lines(section) {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "package =") && !strings.HasPrefix(trimmed, customHookPackageLine) {
					t.Errorf("custom hook package line %q lacks the %q priority", trimmed, customHookPackageLine)
				}
			}
			for _, id := range gitHooksBuiltInIDs {
				block := customHookBlock(section, id)
				if block != "" && !strings.Contains(block, "\n      "+customHookPackageLine) {
					t.Errorf("custom hook %q shares a git-hooks.nix built-in ID but renders no package line:\n%s", id, block)
				}
			}
		})
	}
}

// TestLanguagePackageHooksEnableTheirLanguage guards the LanguagePackage
// binding: a hook bound to config.languages.<name>.package only evaluates when
// its own module enables languages.<name> (otherwise the option holds devenv's
// unpinned default, or the language does not exist), and no hook may set both
// NixPackage and LanguagePackage, which the renderer rejects.
func TestLanguagePackageHooksEnableTheirLanguage(t *testing.T) {
	t.Parallel()
	bound := 0
	for _, mod := range ecosystem.DefaultRegistry().All() {
		cfg := ecosystem.ModuleConfig{}
		for _, hook := range mod.PreCommitHooks(cfg) {
			if hook.NixPackage != "" && hook.LanguagePackage != "" {
				t.Errorf("module %q hook %q sets both NixPackage %q and LanguagePackage %q",
					mod.Name(), hook.ID, hook.NixPackage, hook.LanguagePackage)
			}
			if hook.LanguagePackage == "" {
				continue
			}
			bound++
			frag, err := mod.DevenvNixFragment(cfg)
			if err != nil {
				t.Fatalf("module %q DevenvNixFragment: %v", mod.Name(), err)
			}
			if !fragmentEnablesLanguage(frag, hook.LanguagePackage) {
				t.Errorf("module %q hook %q binds to languages.%s.package but its fragment does not enable "+
					"languages.%s:\n%s", mod.Name(), hook.ID, hook.LanguagePackage, hook.LanguagePackage, frag)
			}
		}
	}
	if bound == 0 {
		t.Fatal("no registered module binds a hook to its language package; the check above ran on nothing")
	}
}

// fragmentEnablesLanguage reports whether a module's Nix fragment enables
// languages.<name>, in the single-line or the block form that
// ecosystem.BuildLanguageFragment emits.
func fragmentEnablesLanguage(frag, name string) bool {
	path := "languages." + name
	return strings.Contains(frag, "  "+path+".enable = true;\n") ||
		strings.Contains(frag, "  "+path+" = {\n    enable = true;\n")
}
