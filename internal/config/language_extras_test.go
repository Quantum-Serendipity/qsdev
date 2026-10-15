package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/clojure"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/container"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/golang"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/python"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/scala"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// moduleSettings returns the committable settings of the real modules the
// tests use, as the registry would.
func moduleSettings() ecosystem.CommittableSettings {
	c := ecosystem.CommittableSettings{}
	for _, m := range []ecosystem.EcosystemModule{&golang.Module{}, &python.Module{}, &container.Module{}, &scala.Module{}, &clojure.Module{}} {
		if fields := ecosystem.CommittableFields(m); len(fields) > 0 {
			c[m.Name()] = fields
		}
	}
	return c
}

// TestValidateQsdevConfig_LanguageExtras checks committed extras against the
// modules' committable settings: only a committable key with an accepted
// value passes. The rejected keys reproduce review findings, each of which
// weakened a control while qsdev check stayed green: uv_exclude_newer=project
// dropped the Python release-age gate, directory=docs pointed the Go lock
// file check at a decoy go.mod, trusted_registries replaced hadolint's
// allowlist, a scala build_tool of mill dropped the sbt CI lock check, OSV
// scan and proxy config, a clojure build_tool of leiningen dropped the
// clj-watson step, and a misspelled key was ignored.
func TestValidateQsdevConfig_LanguageExtras(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		lang     types.LanguageConfig
		settings ecosystem.CommittableSettings
		wantErr  string // substring of the message; "" means valid
	}{
		{name: "none", lang: types.LanguageConfig{Name: "go"}, settings: moduleSettings()},
		{name: "go_source nixpkgs", lang: types.LanguageConfig{Name: "go", Extras: []string{"go_source=nixpkgs"}}, settings: moduleSettings()},
		{name: "go_source go-overlay", lang: types.LanguageConfig{Name: "go", Extras: []string{"go_source=go-overlay"}}, settings: moduleSettings()},
		{
			name:     "python release-age gate",
			lang:     types.LanguageConfig{Name: "python", Extras: []string{"uv_exclude_newer=project"}},
			settings: moduleSettings(), wantErr: `"uv_exclude_newer" is not a committable setting: python has none`,
		},
		{
			name:     "detected directory",
			lang:     types.LanguageConfig{Name: "go", Extras: []string{"directory=docs"}},
			settings: moduleSettings(), wantErr: `"directory" is not a committable setting for go; committable settings: go_source`,
		},
		{
			name:     "container registry allowlist",
			lang:     types.LanguageConfig{Name: "container", Extras: []string{"trusted_registries=evil.example.com"}},
			settings: moduleSettings(), wantErr: `"trusted_registries" is not a committable setting`,
		},
		{
			name:     "scala build tool",
			lang:     types.LanguageConfig{Name: "scala", Extras: []string{"build_tool=mill"}},
			settings: moduleSettings(), wantErr: `"build_tool" is not a committable setting`,
		},
		{
			name:     "clojure build tool",
			lang:     types.LanguageConfig{Name: "clojure", Extras: []string{"build_tool=leiningen"}},
			settings: moduleSettings(), wantErr: `"build_tool" is not a committable setting`,
		},
		{
			name:     "misspelled key",
			lang:     types.LanguageConfig{Name: "go", Extras: []string{"go_sorce=nixpkgs"}},
			settings: moduleSettings(), wantErr: `"go_sorce" is not a committable setting for go`,
		},
		{
			name:     "misspelled value",
			lang:     types.LanguageConfig{Name: "go", Extras: []string{"go_source=nixpgks"}},
			settings: moduleSettings(), wantErr: "valid values: go-overlay, nixpkgs",
		},
		{
			name:     "bare flag on a select",
			lang:     types.LanguageConfig{Name: "go", Extras: []string{"go_source"}},
			settings: moduleSettings(), wantErr: `invalid value "true"`,
		},
		{
			name:     "dedicated keys are not extras",
			lang:     types.LanguageConfig{Name: "go", Extras: []string{"version=1.26.9"}},
			settings: moduleSettings(), wantErr: `"version" is not a committable setting`,
		},
		{
			name:    "no settings rejects every extra",
			lang:    types.LanguageConfig{Name: "go", Extras: []string{"go_source=nixpkgs"}},
			wantErr: "go has none",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &types.QsdevConfig{Version: types.ConfigVersionCurrent, Languages: []types.LanguageConfig{tt.lang}}
			errs := ValidateQsdevConfig(cfg, ValidateOptions{Settings: tt.settings})
			if tt.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("unexpected errors: %v", errs)
				}
				return
			}
			if len(errs) != 1 || errs[0].Field != "languages[0].extras" || !strings.Contains(errs[0].Message, tt.wantErr) {
				t.Fatalf("errors = %v, want one languages[0].extras error containing %q", errs, tt.wantErr)
			}
		})
	}
}

// TestProjectPolicyApply_CommittedExtras checks .qsdev.yaml is the single
// source of truth for committable settings in the answers create, join and
// update generate from: a committed value is set, an absent one is cleared
// so the module's default applies (removing go_source from .qsdev.yaml
// reverts to go-overlay on the next update), and no other committed key,
// such as directory, ever replaces a detected value.
func TestProjectPolicyApply_CommittedExtras(t *testing.T) {
	t.Parallel()
	stale := []string{"go_source=nixpkgs", "directory=.", "toolchain=1.26.9"}
	tests := []struct {
		name      string
		committed []string
		settings  ecosystem.CommittableSettings
		want      []string
		warnings  int
	}{
		{
			name:      "committed value set, others ignored",
			committed: []string{"directory=docs", "go_source=nixpkgs", "toolchain=1.99.0", "go_sorce=go-overlay"},
			settings:  moduleSettings(),
			want:      []string{"go_source=nixpkgs", "directory=.", "toolchain=1.26.9"},
			warnings:  3,
		},
		{
			name:     "absent value cleared to the default",
			settings: moduleSettings(),
			want:     []string{"directory=.", "toolchain=1.26.9"},
		},
		{
			name:      "invalid value cleared and reported",
			committed: []string{"go_source=nixpgks"},
			settings:  moduleSettings(),
			want:      []string{"directory=.", "toolchain=1.26.9"},
			warnings:  1,
		},
		{
			name:      "without settings nothing applies",
			committed: []string{"go_source=go-overlay"},
			want:      stale,
			warnings:  1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			committed := &types.QsdevConfig{Version: types.ConfigVersionCurrent, Languages: []types.LanguageConfig{{
				Name: "go", Version: "1.26.9", Extras: tt.committed,
			}}}
			policy, err := ResolveProjectPolicy(committed, nil)
			if err != nil {
				t.Fatal(err)
			}
			a := types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "go", Version: "1.26.9", Extras: slices.Clone(stale)}}}
			policy.Apply(&a, tt.settings)
			if got := a.Languages[0].Extras; !slices.Equal(got, tt.want) {
				t.Errorf("extras = %v, want %v", got, tt.want)
			}
			warnings := policy.ExtrasWarnings(tt.settings)
			if len(warnings) != tt.warnings {
				t.Errorf("ExtrasWarnings() = %q, want %d warnings", warnings, tt.warnings)
			}
			for _, w := range warnings {
				if !strings.HasPrefix(w, "languages[0].extras: ") || !strings.HasSuffix(w, "; ignored") {
					t.Errorf("warning %q does not name the field and say it is ignored", w)
				}
			}
		})
	}
}

// TestProjectPolicyApply_KeepsPythonReleaseAgeGate reproduces a review
// finding: a committed uv_exclude_newer=project (an extra only detection may
// set, when the project configures its own exclude-newer) removed
// UV_EXCLUDE_NEWER from the generated devenv.nix. It is not committable, so
// the gate stays.
func TestProjectPolicyApply_KeepsPythonReleaseAgeGate(t *testing.T) {
	t.Parallel()
	committed := &types.QsdevConfig{Version: types.ConfigVersionCurrent, Languages: []types.LanguageConfig{{
		Name: "python", PackageManager: "uv", Extras: []string{"uv_exclude_newer=project"},
	}}}
	policy, err := ResolveProjectPolicy(committed, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "python", PackageManager: "uv"}}}
	policy.Apply(&a, moduleSettings())
	fragment, err := (&python.Module{}).DevenvNixFragment(ecosystem.ToModuleConfig(a.Languages[0]))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fragment, "UV_EXCLUDE_NEWER") {
		t.Errorf("devenv.nix fragment lost the uv release-age gate:\n%s", fragment)
	}
}

// TestRecordCommittableSettings checks what a new project records: the
// committable settings its answers hold with a non-default value, and never
// a detected or non-committable extra.
func TestRecordCommittableSettings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		extras []string
		want   []string
	}{
		{"non-default recorded", []string{"directory=.", "go_source=nixpkgs", "toolchain=1.26.9"}, []string{"go_source=nixpkgs"}},
		{"default left out", []string{"go_source=go-overlay"}, nil},
		{"invalid left out", []string{"go_source=nixpgks"}, nil},
		{"detected only", []string{"toolchain=1.26.9"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "go", Extras: tt.extras}}}
			cfg := AnswersToConfig(a, "")
			RecordCommittableSettings(&cfg, a, moduleSettings())
			if got := cfg.Languages[0].Extras; !slices.Equal(got, tt.want) {
				t.Errorf("recorded extras = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLanguageExtras_KeptAsCommitted checks that a day-2 sync and a re-create
// keep the committed extras, while extras only the answers carry (detected
// ones) are never committed.
func TestLanguageExtras_KeptAsCommitted(t *testing.T) {
	t.Parallel()
	const committed = "version: 2\nlanguages:\n  - name: go\n    version: 1.26.9\n    extras:\n      - go_source=nixpkgs\n"
	root := t.TempDir()
	writeCommittedConfig(t, root, committed)
	path := filepath.Join(root, branding.Get().ConfigFile)
	cfg, err := ParseQsdevConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	a := types.WizardAnswers{Languages: []types.LanguageChoice{{
		Name: "go", Version: "1.26.9", Extras: []string{"go_source=go-overlay", "toolchain=1.26.9"},
	}}}

	if err := SyncProjectConfig(root, a); err != nil {
		t.Fatalf("SyncProjectConfig: %v", err)
	}
	synced, err := ParseQsdevConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := synced.Languages[0].Extras; !slices.Equal(got, []string{"go_source=nixpkgs"}) {
		t.Errorf("synced extras = %v, want only the committed go_source=nixpkgs", got)
	}

	fresh := AnswersToConfig(a, "")
	if got := fresh.Languages[0].Extras; got != nil {
		t.Errorf("AnswersToConfig extras = %v, want none recorded from answers", got)
	}
	PreserveCommittedPolicy(&fresh, cfg)
	if got := fresh.Languages[0].Extras; !slices.Equal(got, []string{"go_source=nixpkgs"}) {
		t.Errorf("re-created extras = %v, want the committed go_source=nixpkgs", got)
	}
}

// TestLocalConfig_RejectsExtras checks .qsdev.local.yaml cannot set language
// extras: they are reported as ignored and never reach the effective config,
// for a committed language or a language the local layer adds.
func TestLocalConfig_RejectsExtras(t *testing.T) {
	t.Parallel()
	project := &types.QsdevConfig{Version: types.ConfigVersionCurrent, Languages: []types.LanguageConfig{{Name: "go"}}}
	local := &LocalConfig{Languages: []types.LanguageConfig{
		{Name: "go", Extras: []string{"go_source=nixpkgs"}},
		{Name: "python", Extras: []string{"uv_exclude_newer=project"}},
	}}
	policy, err := ResolveProjectPolicy(project, local)
	if err != nil {
		t.Fatal(err)
	}
	for _, lang := range policy.Effective.Config.Languages {
		if len(lang.Extras) != 0 {
			t.Errorf("effective %s extras = %v, want none from the local layer", lang.Name, lang.Extras)
		}
	}
	warnings := strings.Join(policy.Warnings(), "\n")
	for _, want := range []string{"languages.go.extras: go_source=nixpkgs ignored", "languages.python.extras: uv_exclude_newer=project ignored"} {
		if !strings.Contains(warnings, want) {
			t.Errorf("warnings = %q, want %q", warnings, want)
		}
	}
}

// TestLanguageExtras_RejectsUnknownYAMLShape checks extras is a list of
// strings in .qsdev.yaml; a map is rejected rather than silently dropped.
func TestLanguageExtras_RejectsUnknownYAMLShape(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), branding.Get().ConfigFile)
	content := "version: 2\nlanguages:\n  - name: go\n    extras:\n      go_source: nixpkgs\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseQsdevConfig(path); err == nil {
		t.Error("ParseQsdevConfig accepted extras as a map; want an error")
	}
}
