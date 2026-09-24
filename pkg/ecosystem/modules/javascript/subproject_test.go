package javascript_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/javascript"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestDetect_Subproject covers projects whose package.json is not in the
// repository root (a Go or Python service with its UI in frontend/): the
// JavaScript project directory is recorded in the ExtraDirectory extra and
// everything else is read from there.
func TestDetect_Subproject(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		files        map[string]string
		wantDetected bool
		wantDir      string
		wantPM       string
		wantEvidence []string
	}{
		{
			name:         "root package.json wins over subprojects",
			files:        map[string]string{"package.json": `{}`, "web/package.json": `{}`, "web/pnpm-lock.yaml": ""},
			wantDetected: true,
			wantDir:      "",
			wantPM:       "npm",
		},
		{
			name:         "frontend subproject with lockfile",
			files:        map[string]string{"go.mod": "module x", "frontend/package.json": `{}`, "frontend/package-lock.json": "{}"},
			wantDetected: true,
			wantDir:      "frontend",
			wantPM:       "npm",
			wantEvidence: []string{"package.json found in frontend/"},
		},
		{
			name:         "subproject package manager from its own lockfile",
			files:        map[string]string{"web/package.json": `{}`, "web/pnpm-lock.yaml": ""},
			wantDetected: true,
			wantDir:      "web",
			wantPM:       "pnpm",
		},
		{
			name:         "subproject packageManager pin without lockfile",
			files:        map[string]string{"web/package.json": `{"packageManager":"yarn@1.22.22"}`},
			wantDetected: true,
			wantDir:      "web",
			wantPM:       "yarn",
		},
		{
			name: "workspace members are not separate projects",
			files: map[string]string{
				"frontend/package.json":             `{}`,
				"frontend/packages/ui/package.json": `{}`,
			},
			wantDetected: true,
			wantDir:      "frontend",
			wantPM:       "npm",
		},
		{
			name: "locked subproject preferred and others reported",
			files: map[string]string{
				"docs/package.json":      `{}`,
				"web/package.json":       `{}`,
				"web/yarn.lock":          "",
				"tools/cli/package.json": `{}`,
			},
			wantDetected: true,
			wantDir:      "web",
			wantPM:       "yarn",
			wantEvidence: []string{"WARNING: other JavaScript projects are not configured", "docs/", "tools/cli/"},
		},
		{
			name:         "shallowest subproject preferred without lockfiles",
			files:        map[string]string{"a/b/package.json": `{}`, "z/package.json": `{}`},
			wantDetected: true,
			wantDir:      "z",
			wantPM:       "npm",
		},
		{
			name:         "dependency trees are not projects",
			files:        map[string]string{"node_modules/left-pad/package.json": `{}`, "vendor/x/package.json": `{}`},
			wantDetected: false,
		},
		{
			name:         "shell-unsafe directory is not used",
			files:        map[string]string{"my app/package.json": `{}`},
			wantDetected: false,
		},
		{
			name:         "beyond the scan depth",
			files:        map[string]string{"a/b/c/d/package.json": `{}`},
			wantDetected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, dir, name, content)
			}
			got := (&javascript.Module{}).Detect(dir)
			if got.Detected != tt.wantDetected {
				t.Fatalf("Detected = %v, want %v (evidence %v)", got.Detected, tt.wantDetected, got.Evidence)
			}
			if !tt.wantDetected {
				return
			}
			if d := got.SuggestedConfig.Extras[ecosystem.ExtraDirectory]; d != tt.wantDir {
				t.Errorf("directory extra = %q, want %q", d, tt.wantDir)
			}
			if got.SuggestedConfig.PackageManager != tt.wantPM {
				t.Errorf("PackageManager = %q, want %q", got.SuggestedConfig.PackageManager, tt.wantPM)
			}
			for _, e := range tt.wantEvidence {
				assertEvidenceContains(t, got.Evidence, e)
			}
		})
	}
}

// TestDetect_SubprojectFiles checks that the Node.js version, TypeScript,
// Yarn flavour and tool detection read the subproject, not the root.
func TestDetect_SubprojectFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "frontend/package.json", `{"packageManager":"yarn@1.22.22","devDependencies":{"eslint":"9.0.0"}}`)
	writeFile(t, dir, "frontend/.nvmrc", "v22\n")
	writeFile(t, dir, "frontend/tsconfig.json", `{}`)

	got := (&javascript.Module{}).Detect(dir)
	cfg := got.SuggestedConfig
	if cfg.Version != "22" {
		t.Errorf("Version = %q, want 22 (from frontend/.nvmrc)", cfg.Version)
	}
	for key, want := range map[string]string{
		"typescript":                "true",
		javascript.ExtraYarnClassic: "true",
		javascript.ExtraESLint:      "node_modules",
	} {
		if cfg.Extras[key] != want {
			t.Errorf("Extras[%q] = %q, want %q", key, cfg.Extras[key], want)
		}
	}
}

// TestSubprojectGeneration checks that every generated artifact targets the
// JavaScript project directory when one is recorded.
func TestSubprojectGeneration(t *testing.T) {
	t.Parallel()
	m := &javascript.Module{}
	extras := func(kv ...string) map[string]string {
		e := map[string]string{ecosystem.ExtraDirectory: "frontend"}
		for i := 0; i+1 < len(kv); i += 2 {
			e[kv[i]] = kv[i+1]
		}
		return e
	}

	t.Run("devenv directory", func(t *testing.T) {
		t.Parallel()
		frag, err := m.DevenvNixFragment(ecosystem.ModuleConfig{Version: "22", PackageManager: "pnpm", Extras: extras()})
		if err != nil {
			t.Fatal(err)
		}
		want := `directory = "${config.devenv.root}/frontend";`
		if !strings.Contains(frag, want) {
			t.Errorf("fragment missing %q:\n%s", want, frag)
		}
		root, err := m.DevenvNixFragment(ecosystem.ModuleConfig{Version: "22", PackageManager: "pnpm"})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(root, "directory") {
			t.Errorf("root project fragment sets a directory:\n%s", root)
		}
	})

	t.Run("unsafe directory ignored", func(t *testing.T) {
		t.Parallel()
		cfg := ecosystem.ModuleConfig{Version: "22", Extras: map[string]string{ecosystem.ExtraDirectory: `x";${abort}`}}
		frag, err := m.DevenvNixFragment(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(frag, "directory") || strings.Contains(frag, "abort") {
			t.Errorf("fragment embeds an unsafe directory:\n%s", frag)
		}
		if got := m.SecurityConfigs(cfg)[0].Path; got != ".npmrc" {
			t.Errorf("security config path = %q, want .npmrc", got)
		}
	})

	t.Run("security configs", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			pm, classic, want string
		}{
			{"npm", "", "frontend/.npmrc"},
			{"pnpm", "", "frontend/pnpm-workspace.yaml"},
			{"yarn", "", "frontend/.yarnrc.yml"},
			{"yarn", "true", "frontend/.yarnrc"},
			{"bun", "", "frontend/bunfig.toml"},
		}
		for _, tt := range tests {
			files := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: tt.pm, Extras: extras(javascript.ExtraYarnClassic, tt.classic)})
			if len(files) != 1 || files[0].Path != tt.want {
				t.Errorf("%s (classic=%q): paths = %v, want [%s]", tt.pm, tt.classic, generatedPaths(files), tt.want)
			}
		}
	})

	t.Run("CI, verification and manifests", func(t *testing.T) {
		t.Parallel()
		cfg := ecosystem.ModuleConfig{PackageManager: "npm", Extras: extras(javascript.ExtraESLint, "node_modules")}

		ci := m.CICommands(cfg)
		wantCI := []string{
			"(cd frontend && npm ci --ignore-scripts)",
			"(cd frontend && npm audit --audit-level=moderate)",
		}
		if len(ci) != len(wantCI) {
			t.Fatalf("CI commands = %+v, want %q", ci, wantCI)
		}
		for i, want := range wantCI {
			if ci[i].Command != want {
				t.Errorf("CI command %d = %q, want %q", i, ci[i].Command, want)
			}
		}

		vc := m.VerificationCommands(cfg)
		for _, cmd := range vc.All() {
			if !strings.HasPrefix(cmd, "(cd frontend && ") {
				t.Errorf("verification command %q does not run in frontend/", cmd)
			}
		}

		mf := m.ManifestFiles(cfg)
		if len(mf) != 1 || mf[0].Path != "frontend/package.json" || mf[0].LockFile != "frontend/package-lock.json" {
			t.Errorf("manifest files = %+v, want frontend/package.json + frontend/package-lock.json", mf)
		}
	})
}

// TestSubprojectHooks checks that the eslint and prettier hooks of a
// subproject run from its directory, on its files only, with the tool the
// project provides: git-hooks.nix's built-in hooks run from the repository
// root, where ESLint 9 finds no eslint.config.* and every other JavaScript
// file in the repository would be linted too.
func TestSubprojectHooks(t *testing.T) {
	t.Parallel()
	m := &javascript.Module{}
	tests := []struct {
		name, extra, src string
		wantFiles        string
		wantNix          string
		wantScript       []string
	}{
		{
			name: "eslint from node_modules", extra: javascript.ExtraESLint, src: "node_modules",
			wantFiles:  `^frontend/.*\.(c|m)?[jt]sx?$`,
			wantScript: []string{"cd frontend || exit 1", `files+=("${f#frontend/}")`, `exec ./node_modules/.bin/eslint --fix "${files[@]}"`},
		},
		{
			name: "eslint from nixpkgs", extra: javascript.ExtraESLint, src: "nix",
			wantFiles: `^frontend/.*\.(c|m)?[jt]sx?$`, wantNix: "eslint",
			wantScript: []string{"cd frontend || exit 1", `exec eslint --fix "${files[@]}"`},
		},
		{
			name: "prettier from node_modules", extra: javascript.ExtraPrettier, src: "node_modules",
			wantFiles:  "^frontend/",
			wantScript: []string{"cd frontend || exit 1", `exec ./node_modules/.bin/prettier --ignore-unknown --list-different --write "${files[@]}"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hooks := m.PreCommitHooks(ecosystem.ModuleConfig{Extras: map[string]string{ecosystem.ExtraDirectory: "frontend", tt.extra: tt.src}})
			if len(hooks) != 1 {
				t.Fatalf("hooks = %+v, want one", hooks)
			}
			h := hooks[0]
			if h.BuiltIn || h.Files != tt.wantFiles || h.NixPackage != tt.wantNix || len(h.Settings) != 0 || !h.PassFilenames {
				t.Errorf("hook = %+v, want a script hook on %q with package %q", h, tt.wantFiles, tt.wantNix)
			}
			for _, want := range tt.wantScript {
				if !strings.Contains(h.Script, want) {
					t.Errorf("script missing %q:\n%s", want, h.Script)
				}
			}
		})
	}

	t.Run("root project keeps the built-in hook", func(t *testing.T) {
		t.Parallel()
		hooks := m.PreCommitHooks(ecosystem.ModuleConfig{Extras: map[string]string{javascript.ExtraESLint: "node_modules"}})
		if len(hooks) != 1 || !hooks[0].BuiltIn || hooks[0].Script != "" || hooks[0].Settings["binPath"] != "./node_modules/.bin/eslint" {
			t.Errorf("hooks = %+v, want the built-in eslint hook with binPath ./node_modules/.bin/eslint", hooks)
		}
	})
}

func generatedPaths(files []types.GeneratedFile) []string {
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	return paths
}
