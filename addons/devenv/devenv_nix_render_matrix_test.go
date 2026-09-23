package devenv

import (
	"context"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"

	// Register every ecosystem module with the DefaultRegistry.
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
)

// TestBuildDevenvNixData_HookIDsDisjointForEveryModulePair is the W168
// regression: no hook ID may appear in more than one of SecurityHooks,
// BuiltInHooks and CustomHooks (or twice in one), for any single module or
// pair of modules, or devenv.nix defines the hook twice and fails to parse.
// Rendering each combination also runs the attribute normalizer, which
// rejects any other attribute defined twice.
func TestBuildDevenvNixData_HookIDsDisjointForEveryModulePair(t *testing.T) {
	t.Parallel()
	reg := ecosystem.DefaultRegistry()
	names := reg.Names()
	var combos [][]string
	for i, a := range names {
		combos = append(combos, []string{a})
		for _, b := range names[i+1:] {
			combos = append(combos, []string{a, b})
		}
	}
	for _, combo := range combos {
		var langs []types.LanguageChoice
		for _, n := range combo {
			langs = append(langs, types.LanguageChoice{Name: n})
		}
		answers := types.WizardAnswers{ProjectName: "matrix", Languages: langs}
		data, err := BuildDevenvNixData(answers, reg)
		if err != nil {
			t.Fatalf("%v: BuildDevenvNixData: %v", combo, err)
		}
		seen := make(map[string]string)
		check := func(list, id string) {
			if prev, ok := seen[id]; ok {
				t.Errorf("%v: hook %q is in both %s and %s", combo, id, prev, list)
			}
			seen[id] = list
		}
		for _, id := range data.SecurityHooks {
			check("SecurityHooks", id)
		}
		for _, id := range data.BuiltInHooks {
			check("BuiltInHooks", id)
		}
		for _, h := range data.CustomHooks {
			check("CustomHooks", h.ID)
		}
		if _, err := GenerateDevenvNix(answers, reg); err != nil {
			t.Errorf("%v: GenerateDevenvNix: %v", combo, err)
		}
	}
}

// matrixCases are answers with no language, each module alone, and each
// module with every service and catalog tool enabled.
func matrixCases(t *testing.T) map[string]types.WizardAnswers {
	t.Helper()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("loading catalog: %v", err)
	}
	tools := make(map[string]bool)
	for name := range cat.Tools() {
		tools[name] = true
	}
	var services []types.ServiceChoice
	for _, name := range validation.Services() {
		services = append(services, types.ServiceChoice{Name: name})
	}
	reg := ecosystem.DefaultRegistry()
	cases := map[string]types.WizardAnswers{"none": {ProjectName: "matrix"}}
	for _, lang := range reg.Names() {
		langs := []types.LanguageChoice{{Name: lang}}
		cases["lang-"+lang] = types.WizardAnswers{ProjectName: "matrix", Languages: langs}
		cases["full-"+lang] = types.WizardAnswers{
			ProjectName: "matrix", Languages: langs, Services: services, EnabledTools: tools,
			EnvVars: map[string]string{"EDITOR": "vim"},
		}
	}
	return cases
}

// matrixRenders renders devenv.nix for every matrixCases entry.
func matrixRenders(t *testing.T) map[string][]byte {
	t.Helper()
	cases := matrixCases(t)
	out := make(map[string][]byte, len(cases))
	for name, answers := range cases {
		got, err := GenerateDevenvNix(answers, ecosystem.DefaultRegistry())
		if err != nil {
			t.Fatalf("%s: GenerateDevenvNix: %v", name, err)
		}
		out[name] = got.Content
	}
	return out
}

// TestGenerateDevenvNix_DefinesEachKeyOnce is the W169 regression: every
// attribute set in the generated devenv.nix names each key once, which is
// what statix's W20 (repeated_keys) demands. Normalizing is idempotent only
// when no key repeats; statix and deadnix also run when installed.
func TestGenerateDevenvNix_DefinesEachKeyOnce(t *testing.T) {
	t.Parallel()
	renders := matrixRenders(t)
	for _, name := range slices.Sorted(maps.Keys(renders)) {
		content := string(renders[name])
		again, err := normalizeNixModule(content)
		if err != nil {
			t.Fatalf("%s: renormalizing: %v", name, err)
		}
		if again != content {
			t.Errorf("%s: devenv.nix still repeats a key (normalizing changed it)", name)
		}
	}

	dir := t.TempDir()
	for name, content := range renders {
		if err := os.WriteFile(filepath.Join(dir, name+".nix"), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, linter := range [][]string{{"statix", "check", "--format", "errfmt"}, {"deadnix", "--fail"}} {
		bin, err := exec.LookPath(linter[0])
		if err != nil {
			t.Logf("%s not on PATH; skipping it", linter[0])
			continue
		}
		for _, name := range slices.Sorted(maps.Keys(renders)) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			args := append(slices.Clone(linter[1:]), filepath.Join(dir, name+".nix"))
			out, err := exec.CommandContext(ctx, bin, args...).CombinedOutput()
			cancel()
			if err != nil {
				t.Errorf("%s rejects the %s devenv.nix: %v\n%s", linter[0], name, err, out)
			}
		}
	}
}

// TestGenerateDevenvNix_NormalizePreservesAST checks that grouping repeated
// keys does not change what devenv.nix means: Nix's parser already merges
// `a.b = ...; a.c = ...;` into one attribute set and strips indented strings,
// so `nix-instantiate --parse` of the assembled and the normalized file must
// print the same expression (apart from the module arguments, of which
// normalizing drops the unused ones).
func TestGenerateDevenvNix_NormalizePreservesAST(t *testing.T) {
	t.Parallel()
	nixInstantiate, err := exec.LookPath("nix-instantiate")
	if err != nil {
		t.Skip("nix-instantiate not available")
	}
	cases := matrixCases(t)
	cases["hostile-env"] = types.WizardAnswers{
		ProjectName: "matrix",
		Languages:   []types.LanguageChoice{{Name: "go"}},
		EnvVars: map[string]string{
			"QUOTES":    "a\"b\\c ${x} $${y} '' ''${z} # c ; } {",
			"MULTILINE": "l1\n  l2\n\tl3",
		},
	}
	dir := t.TempDir()
	parse := func(name string, content []byte) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		out, err := exec.CommandContext(ctx, nixInstantiate, "--parse", path).CombinedOutput()
		if err != nil {
			t.Fatalf("%s does not parse: %v\n%s", name, err, out)
		}
		// Drop the `({ formals }: ` prefix.
		body, ok := strings.CutPrefix(string(out), "(")
		if i := strings.Index(body, "}: "); ok && i >= 0 {
			return body[i+3:]
		}
		t.Fatalf("%s: unexpected parse output %.80q", name, out)
		return ""
	}
	for _, name := range slices.Sorted(maps.Keys(cases)) {
		raw, err := renderDevenvNix(cases[name], ecosystem.DefaultRegistry())
		if err != nil {
			t.Fatalf("%s: renderDevenvNix: %v", name, err)
		}
		normalized, err := normalizeNixModule(string(raw))
		if err != nil {
			t.Fatalf("%s: normalizeNixModule: %v", name, err)
		}
		if parse(name+".raw.nix", raw) != parse(name+".nix", []byte(normalized)) {
			t.Errorf("%s: normalizing changed the parsed devenv.nix", name)
		}
	}
}

// TestGenerateDevenvNix_UserEnvOverridesModuleEnv is the W172 regression: a
// user env var that an ecosystem module also sets (GOFLAGS) wins instead of
// defining env.GOFLAGS twice.
func TestGenerateDevenvNix_UserEnvOverridesModuleEnv(t *testing.T) {
	t.Parallel()
	answers := types.WizardAnswers{
		ProjectName: "envs",
		Languages:   []types.LanguageChoice{{Name: "go"}},
		EnvVars:     map[string]string{"GOFLAGS": "-tags=integration"},
	}
	got, err := GenerateDevenvNix(answers, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("GenerateDevenvNix: %v", err)
	}
	attrs, err := NixModuleAttrs(string(got.Content))
	if err != nil {
		t.Fatalf("NixModuleAttrs: %v", err)
	}
	for path, want := range map[string]string{
		"env.GOFLAGS": `"-tags=integration"`, // user value wins
		"env.GOSUMDB": `"sum.golang.org"`,    // module default kept
	} {
		if attrs[path] != want {
			t.Errorf("%s = %q, want %q", path, attrs[path], want)
		}
	}
	if nixInstantiate, err := exec.LookPath("nix-instantiate"); err == nil {
		path := filepath.Join(t.TempDir(), "devenv.nix")
		if err := os.WriteFile(path, got.Content, 0o644); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(nixInstantiate, "--parse", path).CombinedOutput(); err != nil {
			t.Fatalf("devenv.nix does not parse: %v\n%s", err, out)
		}
	}
}

// TestLockFileAudit_WatchesEcosystemLockFiles is the W068 regression: the
// lock-file-audit hook also watches the selected ecosystems' lock files, in
// any directory, not only devenv.lock and flake.lock.
func TestLockFileAudit_WatchesEcosystemLockFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		langs    []string
		match    []string
		notMatch []string
	}{
		{
			name:     "no language keeps the nix lock files",
			match:    []string{"devenv.lock", "flake.lock"},
			notMatch: []string{"package-lock.json", "go.sum"},
		},
		{
			name:     "javascript adds every js package manager lock file",
			langs:    []string{"javascript"},
			match:    []string{"flake.lock", "package-lock.json", "npm-shrinkwrap.json", "web/yarn.lock", "pnpm-lock.yaml", "bun.lock"},
			notMatch: []string{"package.json", "my-package-lock.json", "Cargo.lock"},
		},
		{
			name:     "manifests that double as lock files are left out",
			langs:    []string{"python", "rust"},
			match:    []string{"uv.lock", "poetry.lock", "crates/Cargo.lock"},
			notMatch: []string{"requirements.txt", "pyproject.toml"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var langs []types.LanguageChoice
			for _, l := range tt.langs {
				langs = append(langs, types.LanguageChoice{Name: l})
			}
			data, err := BuildDevenvNixData(types.WizardAnswers{ProjectName: "locks", Languages: langs}, ecosystem.DefaultRegistry())
			if err != nil {
				t.Fatalf("BuildDevenvNixData: %v", err)
			}
			idx := slices.IndexFunc(data.CustomHooks, func(h CustomHookData) bool { return h.ID == "lock-file-audit" })
			if idx < 0 {
				t.Fatal("lock-file-audit hook missing")
			}
			re, err := regexp.Compile(data.CustomHooks[idx].Files)
			if err != nil {
				t.Fatalf("files pattern %q: %v", data.CustomHooks[idx].Files, err)
			}
			for _, f := range tt.match {
				if !re.MatchString(f) {
					t.Errorf("files pattern %q does not match %s", re, f)
				}
			}
			for _, f := range tt.notMatch {
				if re.MatchString(f) {
					t.Errorf("files pattern %q matches %s", re, f)
				}
			}
		})
	}
}
