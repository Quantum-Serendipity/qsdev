package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestValidateQsdevConfig_GoSource checks the go_source extra: the two known
// values pass, and an unknown value, or the extra on another language, fails
// with a message listing the valid values.
func TestValidateQsdevConfig_GoSource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		lang    types.LanguageConfig
		wantErr string // substring of the message; "" means valid
	}{
		{name: "absent", lang: types.LanguageConfig{Name: "go"}},
		{name: "go-overlay", lang: types.LanguageConfig{Name: "go", Extras: []string{"go_source=go-overlay"}}},
		{name: "nixpkgs", lang: types.LanguageConfig{Name: "go", Extras: []string{"go_source=nixpkgs"}}},
		{name: "unknown value", lang: types.LanguageConfig{Name: "go", Extras: []string{"go_source=nixpgks"}}, wantErr: "valid values: go-overlay, nixpkgs"},
		{name: "bare flag", lang: types.LanguageConfig{Name: "go", Extras: []string{"go_source"}}, wantErr: "valid values: go-overlay, nixpkgs"},
		{name: "last entry wins", lang: types.LanguageConfig{Name: "go", Extras: []string{"go_source=nixpkgs", "go_source=bogus"}}, wantErr: "unknown Go source"},
		{name: "other language", lang: types.LanguageConfig{Name: "python", Extras: []string{"go_source=nixpkgs"}}, wantErr: "only to the go language"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &types.QsdevConfig{Version: types.ConfigVersionCurrent, Languages: []types.LanguageConfig{tt.lang}}
			errs := ValidateQsdevConfig(cfg, ValidateOptions{})
			if tt.wantErr == "" {
				if len(errs) != 0 {
					t.Fatalf("unexpected errors: %v", errs)
				}
				return
			}
			if len(errs) != 1 || errs[0].Field != "languages[0].extras.go_source" || !strings.Contains(errs[0].Message, tt.wantErr) {
				t.Fatalf("errors = %v, want one languages[0].extras.go_source error containing %q", errs, tt.wantErr)
			}
		})
	}
}

// TestLanguageExtras_RoundTrip checks that a committed extra reaches the
// answers (join through ConfigToAnswers, update through AdoptCommitted, over
// a stale answers value) and survives a day-2 sync and a re-create, while
// extras that only the answers carry (detected ones) are never committed.
func TestLanguageExtras_RoundTrip(t *testing.T) {
	t.Parallel()
	const committed = "version: 2\nlanguages:\n  - name: go\n    version: 1.26.9\n    extras:\n      - go_source=nixpkgs\n"
	root := t.TempDir()
	writeCommittedConfig(t, root, committed)
	cfg, err := ParseQsdevConfig(filepath.Join(root, branding.Get().ConfigFile))
	if err != nil {
		t.Fatal(err)
	}

	joined := ConfigToAnswers(cfg, types.DetectedProject{}, root)
	if v, _ := joined.Languages[0].Setting(types.SettingGoSource); v != types.GoSourceNixpkgs {
		t.Errorf("join: go_source = %q, want nixpkgs", v)
	}

	stale := types.WizardAnswers{Languages: []types.LanguageChoice{{
		Name: "go", Version: "1.26.9", Extras: []string{"go_source=go-overlay", "toolchain=1.26.9"},
	}}}
	AdoptCommitted(root, &stale)
	if v, _ := stale.Languages[0].Setting(types.SettingGoSource); v != types.GoSourceNixpkgs {
		t.Errorf("update: go_source = %q, want the committed nixpkgs", v)
	}

	if err := SyncProjectConfig(root, stale); err != nil {
		t.Fatalf("SyncProjectConfig: %v", err)
	}
	synced, err := ParseQsdevConfig(filepath.Join(root, branding.Get().ConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	if got := synced.Languages[0].Extras; !slices.Equal(got, []string{"go_source=nixpkgs"}) {
		t.Errorf("synced extras = %v, want only the committed go_source=nixpkgs", got)
	}

	fresh := AnswersToConfig(stale, "")
	if got := fresh.Languages[0].Extras; got != nil {
		t.Errorf("AnswersToConfig extras = %v, want none recorded from answers", got)
	}
	PreserveCommittedPolicy(&fresh, cfg)
	if got := fresh.Languages[0].Extras; !slices.Equal(got, []string{"go_source=nixpkgs"}) {
		t.Errorf("re-created extras = %v, want the committed go_source=nixpkgs", got)
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
