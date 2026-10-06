//go:build nixeval

package devenv_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// This file is the opt-in Nix evaluation smoke test (go test -tags nixeval).
// It needs Nix on PATH, so it is not part of the default test run.
//
//	command go test -tags nixeval -run 'TestDevenvNixEvaluates/hook_package_priorities' ./addons/devenv/
//
// The hook_package_priorities subtest is offline: it resolves <nixpkgs> once
// from the caller's NIX_PATH with the global flake registry disabled, then
// evaluates against that local store path.
//
// Without nix-instantiate or a locally resolvable nixpkgs it skips, or fails
// under testutil.RequireNix.
//
// The devenv_print_dev_env subtest fetches the devenv inputs and is meant for
// nightly and maintainer runs only: it runs only with QSDEV_NIXEVAL=1, and
// then fails when devenv or git is missing.

// nixEvalEnv is the opt-in variable that runs the network-fetching
// devenv_print_dev_env subtest.
const nixEvalEnv = "QSDEV_NIXEVAL"

// devenvHookDef is a git-hooks.hooks.<hook>.package definition that a devenv
// language module makes when the language is enabled.
type devenvHookDef struct {
	hook     string
	priority string // Nix priority wrapper, "" for normal priority (100)
}

// nixEvalCase is one rendered project of TestDevenvNixEvaluates.
type nixEvalCase struct {
	name  string
	lang  string
	files map[string]string
	// devenvHooks mirrors the hook package definitions of devenv's
	// src/modules/languages/<lang>.nix (devenv 1.x): elixir sets credo,
	// dialyzer, mix-format and mix-test at normal priority; terraform sets
	// terraform-format and terraform-validate via mkOverrideDefault (500).
	devenvHooks []devenvHookDef
}

func nixEvalCases() []nixEvalCase {
	normal := func(hooks ...string) []devenvHookDef {
		defs := make([]devenvHookDef, 0, len(hooks))
		for _, h := range hooks {
			defs = append(defs, devenvHookDef{hook: h})
		}
		return defs
	}
	return []nixEvalCase{
		{
			name:        "elixir",
			lang:        "elixir",
			files:       map[string]string{"mix.exs": "defmodule App.MixProject do\nend\n"},
			devenvHooks: normal("credo", "dialyzer", "mix-format", "mix-test"),
		},
		{
			name:  "dart",
			lang:  "dart",
			files: map[string]string{"pubspec.yaml": "name: app\n"},
		},
		{
			name:  "zig 0.13.0",
			lang:  "zig",
			files: map[string]string{"build.zig.zon": ".{\n    .name = .app,\n    .minimum_zig_version = \"0.13.0\",\n}\n"},
		},
		{
			name: "dotnet 8",
			lang: "dotnet",
			files: map[string]string{
				"App.csproj":  "<Project Sdk=\"Microsoft.NET.Sdk\" />\n",
				"global.json": `{"sdk": {"version": "8.0.100"}}`,
			},
		},
		{
			name:  "terraform",
			lang:  "terraform",
			files: map[string]string{"main.tf": "terraform {}\n"},
			devenvHooks: []devenvHookDef{
				{hook: "terraform-format", priority: "lib.mkOverride 500"},
				{hook: "terraform-validate", priority: "lib.mkOverride 500"},
			},
		},
	}
}

// TestDevenvNixEvaluates evaluates rendered devenv.nix files with Nix.
func TestDevenvNixEvaluates(t *testing.T) {
	t.Run("hook_package_priorities", testHookPackagePriorities)
	t.Run("devenv_print_dev_env", testDevenvPrintDevEnv)
}

// testHookPackagePriorities guards U10-01 against the real module system: it
// evaluates every rendered custom hook package definition with nixpkgs'
// lib.evalModules next to git-hooks.nix's mkDefault built-in default and the
// devenv language module's own definition. Two definitions at one priority
// throw "is defined multiple times" (devenv's elixir module and a
// normal-priority custom mix-format did, at base); the language module's pin
// must win, and git-hooks.nix's default must never be the one evaluated. It
// needs only nix-instantiate and a nixpkgs already in the local store (see
// offlineNixEnv); nothing is fetched.
func testHookPackagePriorities(t *testing.T) {
	nixInstantiate := testutil.RequireTool(t, "nix-instantiate", testutil.RequireNix)
	env := offlineNixEnv(t, nixInstantiate)
	for _, tc := range nixEvalCases() {
		t.Run(tc.name, func(t *testing.T) {
			content := renderDetected(t, t.TempDir(), tc)
			defs := customHookPackageDefs(t, content)
			if len(defs) == 0 {
				t.Fatalf("%s renders no custom hook package definition:\n%s", tc.name, content)
			}
			expr, want := hookPriorityExpr(t, tc, defs)

			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, nixInstantiate, "--eval", "--strict", "--json", "--readonly-mode", "--expr", expr)
			cmd.Env = env
			out, err := cmd.Output()
			if err != nil {
				var exitErr *exec.ExitError
				stderr := ""
				if errors.As(err, &exitErr) {
					stderr = string(exitErr.Stderr)
				}
				if strings.Contains(stderr, "is defined multiple times") {
					t.Fatalf("a custom hook package collides with another definition at the same priority:\n%s\nexpression:\n%s", stderr, expr)
				}
				t.Fatalf("nix-instantiate: %v\n%s\nexpression:\n%s", err, stderr, expr)
			}
			var got map[string]string
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("decoding %s: %v", out, err)
			}
			if !maps.Equal(got, want) {
				t.Errorf("resolved hook packages = %v, want %v", got, want)
			}
		})
	}
}

// testDevenvPrintDevEnv writes each rendered project into a fresh git
// repository and runs devenv print-dev-env, the full devenv evaluation. It
// fetches the devenv inputs, so it runs only when QSDEV_NIXEVAL=1 asks for
// it, and then fails when devenv or git is missing.
func testDevenvPrintDevEnv(t *testing.T) {
	if os.Getenv(nixEvalEnv) != "1" {
		t.Skipf("fetches the devenv inputs; set %s=1 to run it", nixEvalEnv)
	}
	devenvBin := lookPathOrFail(t, "devenv")
	gitBin := lookPathOrFail(t, "git")
	for _, tc := range nixEvalCases() {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			env := isolatedEnv(t)
			answers := detectedAnswers(t, dir, tc)
			reg := ecosystem.DefaultRegistry()
			nix, err := devenv.GenerateDevenvNix(answers, reg)
			if err != nil {
				t.Fatalf("GenerateDevenvNix: %v", err)
			}
			yaml, err := devenv.GenerateDevenvYaml(answers, reg)
			if err != nil {
				t.Fatalf("GenerateDevenvYaml: %v", err)
			}
			for _, f := range []*types.GeneratedFile{nix, yaml} {
				if err := os.WriteFile(filepath.Join(dir, f.Path), f.Content, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
			defer cancel()
			run := func(name string, args ...string) {
				t.Helper()
				cmd := exec.CommandContext(ctx, name, args...)
				cmd.Dir = dir
				cmd.Env = env
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("%s %s: %v\n%s", filepath.Base(name), strings.Join(args, " "), err, out)
				}
			}
			run(gitBin, "init", "-q")
			run(gitBin, "add", "-A")
			run(devenvBin, "print-dev-env")
		})
	}
}

// lookPathOrFail resolves a tool on PATH for the opted-in network test, which
// fails without it: QSDEV_NIXEVAL=1 asked for the run.
func lookPathOrFail(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf("%s=1 but %s is not on PATH: %v", nixEvalEnv, name, err)
	}
	return path
}

// isolatedEnv is the parent environment with HOME, TMPDIR and the XDG base
// directories pointed at fresh temporary directories, so Nix and devenv
// neither read nor write the user's own state. NIX_PATH and PATH are kept.
func isolatedEnv(t *testing.T) []string {
	t.Helper()
	isolated := []string{"HOME", "TMPDIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR"}
	env := slices.DeleteFunc(os.Environ(), func(kv string) bool {
		k, _, _ := strings.Cut(kv, "=")
		return slices.Contains(isolated, k)
	})
	root := t.TempDir()
	for _, k := range isolated {
		dir := filepath.Join(root, strings.ToLower(k))
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		env = append(env, k+"="+dir)
	}
	return env
}

// noFlakeRegistry disables the global flake registry, the one Nix setting that
// makes resolving `nixpkgs=flake:nixpkgs` download
// https://channels.nixos.org/flake-registry.json; the system and user
// registries (local files) still apply.
const noFlakeRegistry = "flake-registry = "

// offlineNixEnv resolves <nixpkgs> once, under the caller's own environment
// with the global flake registry disabled, and returns isolatedEnv pinned to
// that store path (NIX_PATH=nixpkgs=<path>) with the global registry still
// disabled, so the evaluation never reaches the network. Without a locally
// resolvable nixpkgs the test skips, or fails under testutil.RequireNix.
func offlineNixEnv(t *testing.T, nixInstantiate string) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, nixInstantiate, "--find-file", "nixpkgs")
	cmd.Env = withNixConfig(os.Environ(), noFlakeRegistry)
	out, err := cmd.Output()
	nixpkgs := strings.TrimSpace(string(out))
	if err != nil || nixpkgs == "" {
		stderr := ""
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}
		testutil.Unavailable(t, testutil.RequireNix,
			"<nixpkgs> does not resolve offline (point NIX_PATH at a local nixpkgs): %v\n%s", err, stderr)
	}
	env := slices.DeleteFunc(isolatedEnv(t), func(kv string) bool {
		return strings.HasPrefix(kv, "NIX_PATH=")
	})
	return withNixConfig(append(env, "NIX_PATH=nixpkgs="+nixpkgs), noFlakeRegistry)
}

// withNixConfig returns env with line appended to NIX_CONFIG, keeping any
// configuration the caller already passes that way.
func withNixConfig(env []string, line string) []string {
	current := ""
	env = slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
		v, ok := strings.CutPrefix(kv, "NIX_CONFIG=")
		if ok {
			current = v
		}
		return ok
	})
	if current != "" {
		line = current + "\n" + line
	}
	return append(env, "NIX_CONFIG="+line)
}

// detectedAnswers writes the case's manifests into dir and returns the
// --yes answers for them: Detect, FillDefaults, highest hook tier.
func detectedAnswers(t *testing.T, dir string, tc nixEvalCase) types.WizardAnswers {
	t.Helper()
	for name, content := range tc.files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cat := catalog.MustDefault()
	tiers := cat.HookTierOrder()
	answers := types.WizardAnswers{ProjectName: "nixeval", HookTier: tiers[len(tiers)-1]}
	answers.FillDefaults(ecosystem.DefaultRegistry().DetectWithEnvironment(dir).Project, cat)
	if !slices.ContainsFunc(answers.Languages, func(l types.LanguageChoice) bool { return l.Name == tc.lang }) {
		t.Fatalf("language %q not detected; got %+v", tc.lang, answers.Languages)
	}
	return answers
}

// renderDetected renders devenv.nix for the case's detected project.
func renderDetected(t *testing.T, dir string, tc nixEvalCase) string {
	t.Helper()
	got, err := devenv.GenerateDevenvNix(detectedAnswers(t, dir, tc), ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("GenerateDevenvNix: %v", err)
	}
	return string(got.Content)
}

// hookPackageDef is one custom hook's rendered package definition.
type hookPackageDef struct {
	hook string
	expr string // the right-hand side of `package = ...;`, verbatim
}

var (
	customHookOpenRE  = regexp.MustCompile(`^    "?([A-Za-z0-9_.-]+)"? = \{$`)
	customHookPkgRE   = regexp.MustCompile(`^      package = (.+);$`)
	pkgsAttrRE        = regexp.MustCompile(`\bpkgs\.([A-Za-z0-9_.-]+)`)
	languagePackageRE = regexp.MustCompile(`\bconfig\.languages\.([A-Za-z0-9_-]+)\.package\b`)
)

// customHookPackageDefs returns every custom hook's package definition from
// the rendered git-hooks.hooks custom section.
func customHookPackageDefs(t *testing.T, content string) []hookPackageDef {
	t.Helper()
	var defs []hookPackageDef
	hook := ""
	for line := range strings.Lines(customHooksSection(t, content)) {
		line = strings.TrimRight(line, "\r\n")
		if m := customHookOpenRE.FindStringSubmatch(line); m != nil {
			hook = m[1]
		} else if m := customHookPkgRE.FindStringSubmatch(line); m != nil && hook != "" {
			defs = append(defs, hookPackageDef{hook: hook, expr: m[1]})
		}
	}
	return defs
}

// gitHooksDefaultName names the stub package git-hooks.nix's mkDefault
// built-in default resolves to; no custom hook may resolve to it.
const gitHooksDefaultName = "git-hooks-default"

// hookPriorityExpr builds the Nix expression that evaluates the rendered
// custom hook package definitions together with git-hooks.nix's built-in
// defaults and the devenv language module's definitions, and returns the
// hook -> package name it must resolve to. Packages are stub derivations
// named after the expression that defines them, so the winner is visible.
func hookPriorityExpr(t *testing.T, tc nixEvalCase, defs []hookPackageDef) (string, map[string]string) {
	t.Helper()
	want := make(map[string]string, len(defs))
	pkgsAttrs := map[string]bool{}
	languages := map[string]bool{}
	var rendered strings.Builder
	for _, d := range defs {
		pkgsRefs := pkgsAttrRE.FindAllStringSubmatch(d.expr, -1)
		langRefs := languagePackageRE.FindAllStringSubmatch(d.expr, -1)
		switch {
		case len(pkgsRefs) == 1 && len(langRefs) == 0:
			pkgsAttrs[pkgsRefs[0][1]] = true
			want[d.hook] = "pkgs." + pkgsRefs[0][1]
		case len(langRefs) == 1 && len(pkgsRefs) == 0:
			languages[langRefs[0][1]] = true
			want[d.hook] = "languages." + langRefs[0][1] + ".package"
		default:
			t.Fatalf("hook %s: package %q must name exactly one pkgs.<attr> or config.languages.<name>.package", d.hook, d.expr)
		}
		fmt.Fprintf(&rendered, "        hooks.%s.package = %s;\n", strconv.Quote(d.hook), d.expr)
	}

	var devenvDefs strings.Builder
	for _, d := range tc.devenvHooks {
		value := fmt.Sprintf("(stub %q)", "devenv-languages-"+tc.lang)
		if d.priority != "" {
			value = "(" + d.priority + " " + value + ")"
		}
		fmt.Fprintf(&devenvDefs, "        hooks.%s.package = %s;\n", strconv.Quote(d.hook), value)
		if _, ok := want[d.hook]; ok {
			// The language module's own pin outranks the custom definition.
			want[d.hook] = "devenv-languages-" + tc.lang
		}
	}

	var pkgsSet strings.Builder
	for _, attr := range slices.Sorted(maps.Keys(pkgsAttrs)) {
		fmt.Fprintf(&pkgsSet, "      (lib.setAttrByPath (lib.splitString \".\" %q) (stub %q))\n", attr, "pkgs."+attr)
	}
	var langSet strings.Builder
	for _, name := range slices.Sorted(maps.Keys(languages)) {
		fmt.Fprintf(&langSet, "        languages.%s.package = stub %q;\n", strconv.Quote(name), "languages."+name+".package")
	}

	expr := `let
  lib = import <nixpkgs/lib>;
  stub = name: { type = "derivation"; inherit name; outPath = "/nix/store/00000000000000000000000000000000-${name}"; };
  pkgs = lib.foldl' lib.recursiveUpdate { } [
` + pkgsSet.String() + `  ];
  eval = lib.evalModules {
    specialArgs = { inherit pkgs; };
    modules = [
      {
        options.languages = lib.mkOption {
          type = lib.types.attrsOf (lib.types.submodule { options.package = lib.mkOption { type = lib.types.package; }; });
          default = { };
        };
        options.hooks = lib.mkOption {
          type = lib.types.attrsOf (lib.types.submodule {
            options.package = lib.mkOption { type = lib.types.nullOr lib.types.package; default = null; };
          });
          default = { };
        };
      }
      # git-hooks.nix: config.hooks = mapAttrs (_: mapAttrs (_: mkDefault)) { <id>.package = ...; }
      { hooks = lib.genAttrs [ ` + nixStrings(gitHooksBuiltInIDs) + ` ] (_: { package = lib.mkDefault (stub "` + gitHooksDefaultName + `"); }); }
      # devenv src/modules/languages/` + tc.lang + `.nix
      {
` + devenvDefs.String() + `      }
      # the rendered devenv.nix
      ({ config, lib, pkgs, ... }: {
` + langSet.String() + rendered.String() + `      })
    ];
  };
in
lib.genAttrs [ ` + nixStrings(slices.Sorted(maps.Keys(want))) + ` ] (id: eval.config.hooks.${id}.package.name)
`
	return expr, want
}

// nixStrings renders ss as space-separated Nix string literals.
func nixStrings(ss []string) string {
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = strconv.Quote(s)
	}
	return strings.Join(quoted, " ")
}
