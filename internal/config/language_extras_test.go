package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/golang"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/python"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// moduleSettings returns the settings the real go and python modules
// declare, as the registry would.
func moduleSettings() ecosystem.DeclaredSettings {
	return ecosystem.DeclaredSettings{
		ecosystem.NameGo:     ecosystem.SettingFields(&golang.Module{}),
		ecosystem.NamePython: ecosystem.SettingFields(&python.Module{}),
	}
}

// TestValidateQsdevConfig_LanguageExtras checks committed extras against the
// module declarations: only a declared key with an accepted value passes.
// The undeclared cases reproduce review findings: uv_exclude_newer=project
// dropped the Python release-age gate from the generated CI, directory=docs
// pointed the Go lock file check at a decoy go.mod, and a misspelled key was
// silently ignored.
func TestValidateQsdevConfig_LanguageExtras(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		lang     types.LanguageConfig
		settings ecosystem.DeclaredSettings
		wantErr  string // substring of the message; "" means valid
	}{
		{name: "none", lang: types.LanguageConfig{Name: "go"}, settings: moduleSettings()},
		{name: "go_source nixpkgs", lang: types.LanguageConfig{Name: "go", Extras: []string{"go_source=nixpkgs"}}, settings: moduleSettings()},
		{name: "go_source go-overlay", lang: types.LanguageConfig{Name: "go", Extras: []string{"go_source=go-overlay"}}, settings: moduleSettings()},
		{
			name:     "release-age gate key is not a python setting",
			lang:     types.LanguageConfig{Name: "python", Extras: []string{"uv_exclude_newer=project"}},
			settings: moduleSettings(), wantErr: `unknown setting "uv_exclude_newer": python declares no settings`,
		},
		{
			name:     "detected directory cannot be committed",
			lang:     types.LanguageConfig{Name: "go", Extras: []string{"directory=docs"}},
			settings: moduleSettings(), wantErr: `unknown setting "directory" for go; declared settings: go_source`,
		},
		{
			name:     "misspelled key",
			lang:     types.LanguageConfig{Name: "go", Extras: []string{"go_sorce=nixpkgs"}},
			settings: moduleSettings(), wantErr: `unknown setting "go_sorce" for go`,
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
			settings: moduleSettings(), wantErr: `unknown setting "version"`,
		},
		{
			name:    "no declarations rejects every extra",
			lang:    types.LanguageConfig{Name: "go", Extras: []string{"go_source=nixpkgs"}},
			wantErr: "go declares no settings",
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

// TestProjectPolicyApply_CommittedExtras checks that the committed extras
// reach the answers (create, join and update all apply the policy) only for
// declared settings with accepted values: a committed directory never
// replaces the detected one, and an undeclared key never reaches generation.
func TestProjectPolicyApply_CommittedExtras(t *testing.T) {
	t.Parallel()
	committed := &types.QsdevConfig{Version: types.ConfigVersionCurrent, Languages: []types.LanguageConfig{{
		Name: "go", Version: "1.26.9",
		Extras: []string{"directory=docs", "go_source=nixpkgs", "toolchain=1.99.0", "go_sorce=go-overlay"},
	}}}
	policy, err := ResolveProjectPolicy(committed, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := types.WizardAnswers{Languages: []types.LanguageChoice{{
		Name: "go", Version: "1.26.9", Extras: []string{"go_source=go-overlay", "directory=.", "toolchain=1.26.9"},
	}}}
	policy.Apply(&a, moduleSettings())
	want := []string{"go_source=nixpkgs", "directory=.", "toolchain=1.26.9"}
	if got := a.Languages[0].Extras; !slices.Equal(got, want) {
		t.Errorf("extras = %v, want %v", got, want)
	}

	// Without declarations nothing committed is applied.
	b := types.WizardAnswers{Languages: []types.LanguageChoice{{Name: "go"}}}
	policy.Apply(&b, nil)
	if got := b.Languages[0].Extras; len(got) != 0 {
		t.Errorf("extras without declarations = %v, want none", got)
	}
}

// TestProjectPolicyApply_KeepsPythonReleaseAgeGate reproduces a review
// finding: a committed uv_exclude_newer=project (an extra only detection may
// set, when the project configures its own exclude-newer) removed
// UV_EXCLUDE_NEWER from the generated devenv.nix. It is not a declared
// setting, so the gate stays.
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
