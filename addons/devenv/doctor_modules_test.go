package devenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

func TestModuleCheckSection(t *testing.T) {
	t.Parallel()

	noEnv := func(string) (string, bool) { return "", false }

	tests := []struct {
		name        string
		files       map[string]string
		wantNil     bool
		wantStatus  map[string]string
		wantWarning string
	}{
		{name: "no project config", wantNil: true},
		{
			name:    "no module with checks",
			files:   map[string]string{".qsdev.yaml": "version: 2\nlanguages:\n  - name: go\n"},
			wantNil: true,
		},
		{
			name: "azure placeholder counts as unset",
			files: map[string]string{
				".qsdev.yaml": "version: 2\nlanguages:\n  - name: azure\n",
				"devenv.nix":  "{ pkgs, ... }:\n{\n  env.ARM_SUBSCRIPTION_ID = \"PLACEHOLDER -- set me\";\n}\n",
			},
			wantStatus: map[string]string{
				"azure-sub":  doctor.ModuleCheckWarn,
				"azure-auth": doctor.ModuleCheckNotRun,
			},
		},
		{
			name: "gcp declared in devenv.local.nix",
			files: map[string]string{
				".qsdev.yaml":      "version: 2\nlanguages:\n  - name: gcp\n",
				"devenv.local.nix": "{ pkgs, ... }:\n{\n  env.CLOUDSDK_ACTIVE_CONFIG_NAME = \"proj-dev\";\n}\n",
			},
			wantStatus: map[string]string{
				"gcp-config": doctor.ModuleCheckOK,
				"gcp-auth":   doctor.ModuleCheckNotRun,
			},
		},
		{
			name: "unparsable devenv module",
			files: map[string]string{
				".qsdev.yaml":      "version: 2\nlanguages:\n  - name: aws\n",
				"devenv.local.nix": "not nix",
			},
			wantStatus:  map[string]string{"aws-profile": doctor.ModuleCheckWarn},
			wantWarning: "devenv.local.nix",
		},
		{
			name:        "malformed config",
			files:       map[string]string{".qsdev.yaml": "languages: ["},
			wantWarning: "not run",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for rel, content := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			// The resolved path points at a script that would leave a marker
			// if anything executed it.
			marker := filepath.Join(dir, "executed")
			script := filepath.Join(dir, "cli")
			if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch "+marker+"\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			env := doctor.ModuleCheckEnv{
				LookupEnv: noEnv,
				LookPath:  func(string) (string, error) { return script, nil },
			}

			ms := moduleCheckSection(dir, ecosystem.DefaultRegistry(), env)
			if _, err := os.Stat(marker); err == nil {
				t.Fatal("a doctor check command was executed")
			}
			if tt.wantNil {
				if ms != nil {
					t.Errorf("moduleCheckSection() = %+v, want nil", ms)
				}
				return
			}
			if ms == nil {
				t.Fatal("moduleCheckSection() = nil")
			}
			got := make(map[string]string, len(ms.Checks))
			for _, c := range ms.Checks {
				got[c.Name] = c.Status
			}
			for name, want := range tt.wantStatus {
				if got[name] != want {
					t.Errorf("check %s status = %q, want %q (all: %+v)", name, got[name], want, ms.Checks)
				}
			}
			if tt.wantWarning != "" && !strings.Contains(strings.Join(ms.Warnings, "\n"), tt.wantWarning) {
				t.Errorf("warnings = %v, want one mentioning %q", ms.Warnings, tt.wantWarning)
			}
		})
	}

	if ms := moduleCheckSection("", ecosystem.DefaultRegistry(), doctor.ModuleCheckEnv{}); ms != nil {
		t.Errorf("moduleCheckSection(\"\") = %+v, want nil", ms)
	}
}
