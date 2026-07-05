package devenv_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"

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
// the environment — the entry is emitted as a bare binary name that nothing
// provisions, so the hook fails with command-not-found at commit time (BL-P1-8).
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
				if hook.NixPackage != "" {
					continue
				}

				binary := hookBinary(hook.Entry)
				if binary != "" && packageProvidesBinary(devenvPkgs, binary) {
					continue
				}

				t.Errorf("module %q declares custom hook %q (BuiltIn:false) with entry %q but no NixPackage, "+
					"and its binary %q is not shipped via DevenvPackages; the generated hook entry would be a bare "+
					"binary name that nothing provisions, failing with command-not-found at commit time. Set "+
					"NixPackage to the nixpkgs attribute that provides %q (or ship it through DevenvPackages).",
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
