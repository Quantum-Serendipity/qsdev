package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Regression: `profile` held the infra profile (written by init), the
// project-type profile (validated by check and read by join) and a tier
// profile at once, so `qsdev check` rejected every infra profile init wrote.
// Schema v2 splits it into `infra_profile` and project-type `profile`.

func TestMigrateV1SplitProfile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		raw         map[string]any
		wantProfile any
		wantInfra   any
		wantErr     string
	}{
		{"infra profile moves", map[string]any{"profile": "enterprise"}, nil, "enterprise", ""},
		{"default infra profile moves", map[string]any{"profile": "consulting-default"}, nil, "consulting-default", ""},
		{"project-type profile stays", map[string]any{"profile": "go-web"}, "go-web", nil, ""},
		{"unknown name stays as project-type profile", map[string]any{"profile": "acme"}, "acme", nil, ""},
		{"no profile", map[string]any{}, nil, nil, ""},
		{"infra_profile is not a v1 key", map[string]any{"infra_profile": "enterprise"}, nil, nil, "not a version 1 key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := migrateV1SplitProfile(tt.raw)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got["profile"] != tt.wantProfile || got["infra_profile"] != tt.wantInfra {
				t.Errorf("profile=%v infra_profile=%v, want %v %v", got["profile"], got["infra_profile"], tt.wantProfile, tt.wantInfra)
			}
		})
	}
}

func TestParseQsdevConfigBytes_SplitsProfile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		yaml        string
		wantProfile string
		wantInfra   string
		wantErr     string
	}{
		{"v1 infra profile migrates", "version: 1\nprofile: enterprise\n", "", "enterprise", ""},
		{"v1 project-type profile kept", "version: 1\nprofile: go-web\n", "go-web", "", ""},
		{"v2 keys read as written", "version: 2\nprofile: go-web\ninfra_profile: startup-github\n", "go-web", "startup-github", ""},
		{"v2 infra name under profile is not reinterpreted", "version: 2\nprofile: enterprise\n", "enterprise", "", ""},
		{"v1 rejects infra_profile", "version: 1\ninfra_profile: enterprise\n", "", "", "not a version 1 key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := ParseQsdevConfigBytes([]byte(tt.yaml))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Version != types.ConfigVersionCurrent {
				t.Errorf("Version = %d, want %d", cfg.Version, types.ConfigVersionCurrent)
			}
			if cfg.Profile != tt.wantProfile || cfg.InfraProfile != tt.wantInfra {
				t.Errorf("Profile=%q InfraProfile=%q, want %q %q", cfg.Profile, cfg.InfraProfile, tt.wantProfile, tt.wantInfra)
			}
		})
	}
}

// Migrating a v1 file must not re-type values the migration does not touch:
// an unquoted `version: 3.10` round-tripped through Go values became "3.1".
func TestParseQsdevConfigBytes_MigrationKeepsValuesAsWritten(t *testing.T) {
	t.Parallel()
	const legacy = "version: 1\nprofile: enterprise # infra\nlanguages:\n  - name: python\n    version: 3.10\n" +
		"services:\n  - name: postgres\n    version: 016\n"
	cfg, err := ParseQsdevConfigBytes([]byte(legacy))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cfg.Languages[0].Version; got != "3.10" {
		t.Errorf("language version = %q, want %q", got, "3.10")
	}
	if got := cfg.Services[0].Version; got != "016" {
		t.Errorf("service version = %q, want %q", got, "016")
	}
	if cfg.InfraProfile != "enterprise" || cfg.Profile != "" {
		t.Errorf("Profile=%q InfraProfile=%q, want \"\" enterprise", cfg.Profile, cfg.InfraProfile)
	}
}

func TestValidateQsdevConfig_ProfilesAndEnums(t *testing.T) {
	t.Parallel()
	projectProfiles := []string{"go-web", "ts-fullstack"}
	tests := []struct {
		name      string
		cfg       types.QsdevConfig
		wantField string
		wantMsg   []string
	}{
		{"valid infra and project profiles", types.QsdevConfig{Profile: "go-web", InfraProfile: "enterprise"}, "", nil},
		{"infra profile is not checked against project profiles", types.QsdevConfig{InfraProfile: "startup-github"}, "", nil},
		{"unknown infra profile lists the infra registry", types.QsdevConfig{InfraProfile: "enterprse"}, "infra_profile",
			[]string{"consulting-default", "enterprise", "startup-github"}},
		{"infra name is not a project-type profile", types.QsdevConfig{Profile: "enterprise"}, "profile", projectProfiles},
		{"unknown tier", types.QsdevConfig{Tier: "ful"}, "tier", validation.Tiers()},
		{"unknown permission level lists every preset", types.QsdevConfig{ClaudeCode: types.ClaudeCodeConfig{PermissionLevel: "open"}},
			"claude_code.permission_level", validation.PermissionPresets()},
		{"unknown security level", types.QsdevConfig{Security: types.SecurityConfig{Level: "max"}}, "security.level", validation.SecurityLevels()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := ValidateQsdevConfig(&tt.cfg, ValidateOptions{ProfileNames: projectProfiles})
			if tt.wantField == "" {
				if len(errs) != 0 {
					t.Fatalf("unexpected errors: %v", errs)
				}
				return
			}
			if len(errs) != 1 || errs[0].Field != tt.wantField {
				t.Fatalf("errs = %v, want one error on %s", errs, tt.wantField)
			}
			for _, want := range tt.wantMsg {
				if !strings.Contains(errs[0].Message, want) {
					t.Errorf("message %q does not list %q", errs[0].Message, want)
				}
			}
		})
	}
}

func TestAnswersToConfig_ProfilesRoundTrip(t *testing.T) {
	t.Parallel()
	answers := types.WizardAnswers{ProjectTypeProfile: "go-web", ProfileName: "enterprise", Tier: "standard"}
	cfg := AnswersToConfig(answers, "")
	if cfg.Profile != "go-web" || cfg.InfraProfile != "enterprise" {
		t.Fatalf("Profile=%q InfraProfile=%q, want go-web enterprise", cfg.Profile, cfg.InfraProfile)
	}
	data, err := MarshalProjectConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseQsdevConfigBytes(data)
	if err != nil {
		t.Fatalf("written config does not parse: %v", err)
	}
	if errs := ValidateQsdevConfig(parsed, ValidateOptions{ProfileNames: []string{"go-web"}}); len(errs) != 0 {
		t.Errorf("written config does not validate: %v", errs)
	}
	back := ConfigToAnswers(parsed, types.DetectedProject{}, t.TempDir())
	if back.ProjectTypeProfile != "go-web" || back.ProfileName != "enterprise" {
		t.Errorf("join reads ProjectTypeProfile=%q ProfileName=%q", back.ProjectTypeProfile, back.ProfileName)
	}
}

// A version 1 file is rewritten at the current schema by the next sync
// (`qsdev init --update`), even when no answer-derived key changed.
func TestSyncProjectConfig_MigratesLegacySchema(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, branding.Get().ConfigFile)
	legacy := "version: 1\ntier: standard\nprofile: enterprise\nclaude_code:\n  enabled: false\n"
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SyncProjectConfig(dir, types.WizardAnswers{Tier: "standard"}); err != nil {
		t.Fatalf("SyncProjectConfig: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{fmt.Sprintf("version: %d\n", types.ConfigVersionCurrent), "infra_profile: enterprise\n"} {
		if !strings.Contains(content, want) {
			t.Errorf("migrated config lacks %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "\nprofile:") {
		t.Errorf("infra profile left under profile:\n%s", content)
	}

	// A second sync with nothing changed leaves the migrated file alone.
	if err := SyncProjectConfig(dir, types.WizardAnswers{Tier: "standard"}); err != nil {
		t.Fatalf("second SyncProjectConfig: %v", err)
	}
	again, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != content {
		t.Errorf("current-schema config rewritten:\n%s", again)
	}
}
