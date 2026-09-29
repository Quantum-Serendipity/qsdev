package posture

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestAssess_UntrackedPreCommitConfig guards against an untracked pre-commit
// config leaking into qsdev's tracked state: it must never show up in config
// health or file-modification drift, and it credits only the hooks it runs.
func TestAssess_UntrackedPreCommitConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        string
		wantPreCommit bool
		wantSecrets   LayerStatus
	}{
		{
			name:          "no hooks",
			config:        "repos: []\n",
			wantPreCommit: false,
			wantSecrets:   LayerDisabled,
		},
		{
			name: "ripsecrets hook",
			config: "repos:\n" +
				"  - repo: local\n" +
				"    hooks:\n" +
				"      - id: ripsecrets\n",
			wantPreCommit: true,
			wantSecrets:   LayerPartial,
		},
		{
			name: "gitleaks and ripsecrets hooks",
			config: "repos:\n" +
				"  - repo: local\n" +
				"    hooks:\n" +
				"      - id: ripsecrets\n" +
				"      - id: gitleaks\n",
			wantPreCommit: true,
			wantSecrets:   LayerEnabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeState(t, root, ".claude/.qsdev-claude-state.yaml", types.GeneratedState{
				QsdevVersion: "1.0.0",
				LastRun:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				Files:        map[string]types.FileState{},
				EnabledTools: map[string]bool{"attach-guard": false},
			})
			if err := os.WriteFile(filepath.Join(root, preCommitConfigFile), []byte(tt.config), 0o644); err != nil {
				t.Fatal(err)
			}

			report, err := Assess(root, AssessOptions{})
			if err != nil {
				t.Fatalf("Assess: %v", err)
			}

			for _, f := range report.Config.Files {
				if f.Path == preCommitConfigFile {
					t.Errorf("untracked %s reported in config health: %+v", preCommitConfigFile, f)
				}
			}
			for _, cat := range report.Drift.Categories {
				for _, f := range cat.Findings {
					if f.Subject == preCommitConfigFile && cat.Name == "File Modification" {
						t.Errorf("untracked %s reported as drift: %+v", preCommitConfigFile, f)
					}
				}
			}

			secrets := FindLayerByName(report.Defense.Layers, "secrets-scanning")
			if secrets == nil || secrets.Status != tt.wantSecrets {
				t.Errorf("secrets-scanning = %+v, want status %q", secrets, tt.wantSecrets)
			}
			for _, c := range report.Conformance.Baseline.Checks {
				if c.Name == CheckPreCommitHooks && c.Pass != tt.wantPreCommit {
					t.Errorf("pre-commit-hooks check pass = %v, want %v", c.Pass, tt.wantPreCommit)
				}
			}
		})
	}
}

// TestObservedProtections_LeavesInputsUntouched guards the aliasing bug: the
// state map Assess passes to config health and drift detection must not gain
// entries for files qsdev does not track.
func TestObservedProtections_LeavesInputsUntouched(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	config := "repos:\n  - repo: local\n    hooks:\n      - id: gitleaks\n"
	if err := os.WriteFile(filepath.Join(root, preCommitConfigFile), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := map[string]bool{"semgrep": true}
	files := map[string]types.FileState{"CLAUDE.md": {Hash: "abc"}}

	active, present := observedProtections(root, tools, types.GeneratedState{Files: files, EnabledTools: tools})

	if _, ok := files[preCommitConfigFile]; ok {
		t.Error("observedProtections added the pre-commit config to the tracked state map")
	}
	if tools["gitleaks"] {
		t.Error("observedProtections added a hook to the enabled-tools map")
	}
	if _, ok := present.Files[preCommitConfigFile]; !ok {
		t.Error("present view is missing the pre-commit config that declares hooks")
	}
	if !active["gitleaks"] || !active["semgrep"] {
		t.Errorf("active tools = %v, want gitleaks and semgrep", active)
	}
}
