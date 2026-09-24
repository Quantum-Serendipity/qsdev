package cloudisolation

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestParseSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		data         string
		wantDeny     []string
		wantDenyRead []string
		wantErr      bool
	}{
		{
			name:         "deny and sandbox denyRead",
			data:         `{"permissions":{"deny":["Bash(aws configure set)", 7]},"sandbox":{"filesystem":{"denyRead":["~/.aws/credentials"]}}}`,
			wantDeny:     []string{"Bash(aws configure set)"},
			wantDenyRead: []string{"~/.aws/credentials"},
		},
		{
			name: "decoy keys are not read",
			data: `{"Permissions":{"deny":["Bash(aws configure set)"]},"Sandbox":{"filesystem":{"denyRead":["~/.aws/credentials"]}}}`,
		},
		{
			name: "no isolation keys",
			data: `{}`,
		},
		{
			name:    "malformed",
			data:    `{"permissions":`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseSettings([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseSettings error = %v, wantErr %v", err, tt.wantErr)
			}
			if !slices.Equal(got.Deny, tt.wantDeny) || !slices.Equal(got.DenyRead, tt.wantDenyRead) {
				t.Errorf("ParseSettings() = %+v, want deny %v denyRead %v", got, tt.wantDeny, tt.wantDenyRead)
			}
		})
	}
}

func TestReadSettings(t *testing.T) {
	t.Parallel()

	t.Run("missing file is empty", func(t *testing.T) {
		t.Parallel()
		got, err := ReadSettings(t.TempDir())
		if err != nil || got.Present || len(got.Deny) != 0 || len(got.DenyRead) != 0 {
			t.Errorf("ReadSettings() = %+v, %v; want empty and not present, nil", got, err)
		}
	})

	t.Run("reads the project file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755); err != nil {
			t.Fatal(err)
		}
		data := []byte(`{"permissions":{"deny":["Read(~/.azure/accessTokens.json)"]}}`)
		if err := os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := ReadSettings(dir)
		if err != nil {
			t.Fatalf("ReadSettings: %v", err)
		}
		if !got.Present || !slices.Equal(got.Deny, []string{"Read(~/.azure/accessTokens.json)"}) {
			t.Errorf("ReadSettings().Deny = %v", got.Deny)
		}
	})
}

func TestConfiguredProviders(t *testing.T) {
	t.Parallel()

	langs := []types.LanguageConfig{{Name: "golang"}, {Name: "gcp"}, {Name: "aws"}, {Name: "gcp"}, {Name: "terraform"}}
	got := ConfiguredProviders(langs)
	want := []cloudcommon.CloudProvider{cloudcommon.GCP, cloudcommon.AWS}
	if !slices.Equal(got, want) {
		t.Errorf("ConfiguredProviders() = %v, want %v", got, want)
	}
	if got := Assess([]types.LanguageConfig{{Name: "golang"}}, nil, Settings{}, true); got != nil {
		t.Errorf("Assess() with no cloud provider = %v, want nil", got)
	}
}

func TestStatus(t *testing.T) {
	t.Parallel()

	full := Settings{Deny: cloudcommon.BashDenyRules(cloudcommon.AWS), DenyRead: cloudcommon.ReadDenyPaths(cloudcommon.AWS), Present: true}
	dev := map[string]string{"AWS_PROFILE": "dev"}
	langs := []types.LanguageConfig{{Name: "aws"}}
	tests := []struct {
		name       string
		env        map[string]string
		settings   Settings
		claudeCode bool
		want       string
		wantLayers int
	}{
		{"all layers", dev, full, true, StatusIsolated, 3},
		{"placeholder profile", map[string]string{"AWS_PROFILE": "PLACEHOLDER"}, full, true, StatusDegraded, 3},
		{"no deny rules", dev, Settings{DenyRead: full.DenyRead, Present: true}, true, StatusMisconfigured, 3},
		{"empty settings file", dev, Settings{Present: true}, false, StatusMisconfigured, 3},
		{"settings missing with Claude Code", dev, Settings{}, true, StatusMisconfigured, 3},
		{"no Claude Code, profile set", dev, Settings{}, false, StatusIsolated, 1},
		{"no Claude Code, no profile", nil, Settings{}, false, StatusDegraded, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reports := Assess(langs, tt.env, tt.settings, tt.claudeCode)
			if len(reports) != 1 {
				t.Fatalf("Assess() returned %d reports, want 1", len(reports))
			}
			if got := len(reports[0].Statuses); got != tt.wantLayers {
				t.Errorf("Assess() reported %d layers, want %d", got, tt.wantLayers)
			}
			if got := Status(reports[0]); got != tt.want {
				t.Errorf("Status() = %q, want %q", got, tt.want)
			}
			if got, want := reports[0].AllLayersActive, tt.want == StatusIsolated; got != want {
				t.Errorf("AllLayersActive = %v, want %v", got, want)
			}
		})
	}
}
