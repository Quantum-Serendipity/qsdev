package modules

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestVersionedFragments_RejectInvalidVersions verifies a configured version
// (answers files are editable) is validated before it is written into
// devenv.nix.
func TestVersionedFragments_RejectInvalidVersions(t *testing.T) {
	t.Parallel()
	tests := []struct{ lang, version string }{
		{"zig", `0.14.0" ; evil = "`},
		{"zig", "latest"},
		{"swift", `6.0" pkgs.evil "`},
		{"swift", "6"},
	}
	for _, tt := range tests {
		t.Run(tt.lang+"/"+tt.version, func(t *testing.T) {
			t.Parallel()
			mod, _ := ecosystem.DefaultRegistry().ByName(tt.lang)
			if frag, err := mod.DevenvNixFragment(ecosystem.ModuleConfig{Version: tt.version}); err == nil {
				t.Errorf("DevenvNixFragment(%q) = %q, want error", tt.version, frag)
			}
		})
	}
}

// TestDetectionReachesModuleOutput runs the --yes path end to end (detection,
// FillDefaults, ToModuleConfig) and checks the generated packages, Nix
// fragment and hooks reflect what Detect learned. Tier 2+ ecosystems used to
// keep only a detected bool, so every module took its default branch: Flutter
// got no SDK, PHP 8.4 got php83, Meson got no meson/ninja.
func TestDetectionReachesModuleOutput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		files        map[string]string
		lang         string
		wantPackages []string // DevenvPackages / DevenvPackageExprs substrings
		wantFragment []string
		wantHook     map[string]string // hook ID -> exact Entry
	}{
		{
			name:         "flutter",
			files:        map[string]string{"pubspec.yaml": "name: app\nflutter:\n  uses-material-design: true\n"},
			lang:         "dart",
			wantPackages: []string{"flutter"},
		},
		{
			name:         "php 8.4",
			files:        map[string]string{"composer.json": `{"require": {"php": "~8.4.0"}}`},
			lang:         "php",
			wantFragment: []string{"pkgs.php84"},
			wantHook:     map[string]string{"phpcs": "phpcs --standard=PSR12"},
		},
		{
			name:         "php 8.2 with phpcs ruleset",
			files:        map[string]string{"composer.json": `{"require": {"php": "~8.2.0"}}`, "phpcs.xml.dist": "<ruleset/>\n"},
			lang:         "php",
			wantFragment: []string{"pkgs.php82"},
			wantHook:     map[string]string{"phpcs": "phpcs"},
		},
		{
			name:         "meson",
			files:        map[string]string{"meson.build": "project('x', 'c')\n", "subprojects/zlib.wrap": "[wrap-file]\n"},
			lang:         "cpp",
			wantPackages: []string{"meson", "ninja"},
		},
		{
			name:         "haskell stack",
			files:        map[string]string{"stack.yaml": "resolver: lts-22.0\n", "app.cabal": "name: app\n"},
			lang:         "haskell",
			wantFragment: []string{"languages.haskell.stack.enable = true"},
		},
		{
			name:         "zig minimum version",
			files:        map[string]string{"build.zig": "", "build.zig.zon": ".{\n    .name = .app,\n    .minimum_zig_version = \"0.14.0\",\n}\n"},
			lang:         "zig",
			wantFragment: []string{"languages.zig.package = (let r = builtins.tryEval (pkgs.zig_0_14 or null);"},
		},
		{
			name:         "swift 6 tools version",
			files:        map[string]string{"Package.swift": "// swift-tools-version: 6.0\nimport PackageDescription\n"},
			lang:         "swift",
			wantFragment: []string{`lib.versionOlder pkgs.swift.version "6.0"`},
		},
		{
			name:         "bazel 8",
			files:        map[string]string{"MODULE.bazel": "module(name = \"x\")\n", ".bazelversion": "8.2.1\n"},
			lang:         "bazel",
			wantPackages: []string{"pkgs.bazel_8 or null"},
		},
		{
			name: "rails bundling rubocop",
			files: map[string]string{
				"Gemfile":      "source 'https://rubygems.org'\ngem 'rubocop-rails-omakase'\n",
				"Gemfile.lock": "GEM\n  remote: https://rubygems.org/\n  specs:\n    rubocop (1.66.1)\n      json (~> 2.3)\n",
			},
			lang:     "ruby",
			wantHook: map[string]string{"rubocop": "bundle exec rubocop --autocorrect --force-exclusion"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			langs := fillFromDetection(t, writeTree(t, tt.files))
			lc, ok := languageByName(langs, tt.lang)
			if !ok {
				t.Fatalf("language %q not selected; got %+v", tt.lang, langs)
			}
			mod, ok := ecosystem.DefaultRegistry().ByName(tt.lang)
			if !ok {
				t.Fatalf("module %q not registered", tt.lang)
			}
			cfg := ecosystem.ToModuleConfig(lc)

			var pkgs []string
			if pp, ok := mod.(ecosystem.PackageProvider); ok {
				pkgs = append(pkgs, pp.DevenvPackages(cfg)...)
			}
			if ep, ok := mod.(ecosystem.PackageExprProvider); ok {
				pkgs = append(pkgs, ep.DevenvPackageExprs(cfg)...)
			}
			for _, want := range tt.wantPackages {
				if !slices.ContainsFunc(pkgs, func(p string) bool { return strings.Contains(p, want) }) {
					t.Errorf("packages %v missing %q (config %+v)", pkgs, want, cfg)
				}
			}

			frag, err := mod.DevenvNixFragment(cfg)
			if err != nil {
				t.Fatalf("DevenvNixFragment: %v", err)
			}
			for _, want := range tt.wantFragment {
				if !strings.Contains(frag, want) {
					t.Errorf("fragment missing %q (config %+v):\n%s", want, cfg, frag)
				}
			}

			for id, want := range tt.wantHook {
				i := slices.IndexFunc(mod.PreCommitHooks(cfg), func(h ecosystem.HookConfig) bool { return h.ID == id })
				if i < 0 {
					t.Errorf("hook %q not generated", id)
					continue
				}
				if entry := mod.PreCommitHooks(cfg)[i].Entry; entry != want {
					t.Errorf("hook %q entry = %q, want %q", id, entry, want)
				}
			}
		})
	}
}
