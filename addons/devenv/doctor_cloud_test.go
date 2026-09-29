package devenv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
)

func TestCloudIsolationSection(t *testing.T) {
	t.Parallel()

	write := func(t *testing.T, dir, rel, content string) {
		t.Helper()
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	settingsWith := func(t *testing.T, provider cloudcommon.CloudProvider) string {
		t.Helper()
		doc := map[string]any{
			"permissions": map[string]any{"deny": cloudcommon.BashDenyRules(provider)},
			"sandbox":     map[string]any{"filesystem": map[string]any{"denyRead": cloudcommon.ReadDenyPaths(provider)}},
		}
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	const awsConfig = "version: 2\nlanguages:\n  - name: go\n  - name: aws\n"

	tests := []struct {
		name         string
		files        map[string]string
		wantNil      bool
		wantStatus   string
		wantWarning  string
		wantProvider string
	}{
		{name: "no project config", wantNil: true},
		{
			name:    "no cloud provider",
			files:   map[string]string{".qsdev.yaml": "version: 2\nlanguages:\n  - name: go\n"},
			wantNil: true,
		},
		{
			name: "isolated",
			files: map[string]string{
				".qsdev.yaml":           awsConfig,
				".claude/settings.json": settingsWith(t, cloudcommon.AWS),
				"devenv.nix":            "{ pkgs, ... }:\n{\n  env = {\n    AWS_PROFILE = \"proj\";\n  };\n}\n",
			},
			wantStatus:   "isolated",
			wantProvider: "aws",
		},
		{
			name: "placeholder and unparsable local module",
			files: map[string]string{
				".qsdev.yaml":           awsConfig,
				".claude/settings.json": settingsWith(t, cloudcommon.AWS),
				"devenv.nix":            "{ pkgs, ... }:\n{\n  env.AWS_PROFILE = \"PLACEHOLDER\";\n}\n",
				"devenv.local.nix":      "not nix",
			},
			wantStatus:   "degraded",
			wantWarning:  "devenv.local.nix",
			wantProvider: "aws",
		},
		{
			name: "no settings with Claude Code enabled",
			files: map[string]string{
				".qsdev.yaml": awsConfig + "claude_code:\n  enabled: true\n",
			},
			wantStatus:   "misconfigured",
			wantProvider: "aws",
		},
		{
			name: "no settings without Claude Code",
			files: map[string]string{
				".qsdev.yaml": awsConfig,
				"devenv.nix":  "{ pkgs, ... }:\n{\n  env.AWS_PROFILE = \"dev\";\n}\n",
			},
			wantStatus:   "isolated",
			wantProvider: "aws",
		},
		{
			name:        "malformed config",
			files:       map[string]string{".qsdev.yaml": "languages: ["},
			wantWarning: "not assessed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for rel, content := range tt.files {
				write(t, dir, rel, content)
			}
			cs := cloudIsolationSection(dir)
			if tt.wantNil {
				if cs != nil {
					t.Errorf("cloudIsolationSection() = %+v, want nil", cs)
				}
				return
			}
			if cs == nil {
				t.Fatal("cloudIsolationSection() = nil")
			}
			if tt.wantProvider != "" {
				if len(cs.Providers) != 1 || cs.Providers[0].Name != tt.wantProvider || cs.Providers[0].Status != tt.wantStatus {
					t.Errorf("providers = %+v, want %s %s", cs.Providers, tt.wantProvider, tt.wantStatus)
				}
			}
			if tt.wantWarning != "" && !strings.Contains(strings.Join(cs.Warnings, "\n"), tt.wantWarning) {
				t.Errorf("warnings = %v, want one mentioning %q", cs.Warnings, tt.wantWarning)
			}
		})
	}

	if cs := cloudIsolationSection(""); cs != nil {
		t.Errorf("cloudIsolationSection(\"\") = %+v, want nil", cs)
	}
}
