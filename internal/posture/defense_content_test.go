package posture

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// hardenedDevenvNix carries every hardening setting qsdev renders into devenv.nix.
const hardenedDevenvNix = `{ pkgs, ... }:
{
  env = {
    DEVENV_SECURITY_HARDENED = "true";
  };
  unsetEnvVars = [ "AWS_ACCESS_KEY_ID" "GITHUB_TOKEN" ];
  dotenv.enable = false;
  git-hooks.hooks = {
    lock-file-audit = {
      enable = true;
      name = "Lock file change audit";
    };
  };

  scripts."qsdev-security-scan" = {
    description = "Run security scanners";
    exec = ''
      set -euo pipefail
      semgrep --config p/golang $(if [ -d .semgrep ]; then echo --config .semgrep; fi) --metrics=off --error .
    '';
  };
}
`

// preCommitWithLockAudit is a rendered pre-commit config registering the
// lock file audit hook.
const preCommitWithLockAudit = `repos:
  - repo: local
    hooks:
      - id: lock-file-audit
        name: Lock file change audit
`

// settingsWithPackageGuard is a .claude/settings.json registering the package
// guard as a PreToolUse hook, as the generator writes it.
const settingsWithPackageGuard = `{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [
  {"type": "command", "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/package-guard.py"}]}]}}`

// writeProjectFiles writes files (relative path -> content) under a fresh
// project directory and returns it with a generated state listing them.
func writeProjectFiles(t *testing.T, files map[string]string) (string, types.GeneratedState) {
	t.Helper()
	dir := t.TempDir()
	st := types.GeneratedState{Files: map[string]types.FileState{}}
	for rel, content := range files {
		abs := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		st.Files[rel] = types.FileState{}
	}
	return dir, st
}

func layerByName(t *testing.T, cov DefenseCoverage, name string) DefenseLayer {
	t.Helper()
	l := FindLayerByName(cov.Layers, name)
	if l == nil {
		t.Fatalf("layer %q not found", name)
	}
	return *l
}

// TestAssessDefenseLayers_ArtifactContent pins that layers are judged from
// the named artifacts and their content, never from substrings of state paths
// or from a file's mere presence (F331).
func TestAssessDefenseLayers_ArtifactContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		tools map[string]bool
		files map[string]string
		layer string
		want  LayerStatus
	}{
		{
			name:  "block-destructive hook is not lock file enforcement",
			tools: map[string]bool{"attach-guard": true},
			files: map[string]string{".claude/hooks/block-destructive.py": "", "yarn.lock": ""},
			layer: "lock-file-enforcement",
			want:  LayerPartial,
		},
		{
			name:  "pre-commit config without lock audit hook is not lock file enforcement",
			tools: map[string]bool{},
			files: map[string]string{".pre-commit-config.yaml": "repos: []\n"},
			layer: "lock-file-enforcement",
			want:  LayerDisabled,
		},
		{
			name:  "lock audit hook in pre-commit config with attach-guard",
			tools: map[string]bool{"attach-guard": true},
			files: map[string]string{".pre-commit-config.yaml": preCommitWithLockAudit},
			layer: "lock-file-enforcement",
			want:  LayerEnabled,
		},
		{
			name:  "lock audit hook in devenv.nix with attach-guard",
			tools: map[string]bool{"attach-guard": true},
			files: map[string]string{"devenv.nix": hardenedDevenvNix},
			layer: "lock-file-enforcement",
			want:  LayerEnabled,
		},
		{
			name:  "path containing vuln is not a scanner",
			tools: map[string]bool{},
			files: map[string]string{"docs/vuln-policy.md": ""},
			layer: "vulnerability-scanning",
			want:  LayerDisabled,
		},
		{
			name:  "socket MCP alone is partial",
			tools: map[string]bool{"socket-dev-mcp": true},
			files: map[string]string{},
			layer: "vulnerability-scanning",
			want:  LayerPartial,
		},
		{
			name:  "grype config with container-security",
			tools: map[string]bool{"container-security": true},
			files: map[string]string{".grype.yaml": ""},
			layer: "vulnerability-scanning",
			want:  LayerEnabled,
		},
		{
			name:  "package guard OSV checks",
			tools: map[string]bool{"attach-guard": true},
			files: map[string]string{".claude/hooks/package-guard.py": ""},
			layer: "vulnerability-scanning",
			want:  LayerEnabled,
		},
		{
			name:  "unhardened devenv.nix",
			tools: map[string]bool{},
			files: map[string]string{"devenv.nix": "{ pkgs, ... }: { packages = [ pkgs.git ]; }\n"},
			layer: "nix-hardening",
			want:  LayerDisabled,
		},
		{
			name:  "partially hardened devenv.nix",
			tools: map[string]bool{},
			files: map[string]string{"devenv.nix": "{ ... }: { dotenv.enable = false; }\n"},
			layer: "nix-hardening",
			want:  LayerPartial,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, genState := writeProjectFiles(t, tt.files)
			cov := AssessDefenseLayers(dir, tt.tools, types.DetectedProject{}, genState, 3)
			if got := layerByName(t, cov, tt.layer); got.Status != tt.want {
				t.Errorf("%s: status = %q (%s), want %q", tt.layer, got.Status, got.Reason, tt.want)
			}
		})
	}
}

// TestLockFileAuditHookIDMatchesCatalog ties the hook id the posture checks
// for to the catalog's custom hook definition, so a rename cannot silently
// disable the lock-file-enforcement layer.
func TestLockFileAuditHookIDMatchesCatalog(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("loading catalog: %v", err)
	}
	for _, h := range cat.CustomHooks() {
		if h.ID == lockFileAuditHookID {
			return
		}
	}
	t.Errorf("catalog defines no custom hook %q", lockFileAuditHookID)
}

// TestAssessDefenseLayers_CountsAreTierRelative pins that Enabled/Total count
// only the layers in scope at the current tier, like Score (F345).
func TestAssessDefenseLayers_CountsAreTierRelative(t *testing.T) {
	t.Parallel()
	dir, genState := writeProjectFiles(t, map[string]string{
		".claude/hooks/package-guard.py": "",
		".claude/settings.json":          settingsWithPackageGuard,
		".pre-commit-config.yaml":        preCommitWithLockAudit,
	})
	tools := map[string]bool{"attach-guard": true}

	tests := []struct {
		tier               int
		wantEnabled, total int
	}{
		// T1 layers: pretooluse-hooks, install-script-blocking,
		// lock-file-enforcement, vulnerability-scanning, all enabled.
		{tier: 1, wantEnabled: 4, total: 4},
		// T2 adds age-gating (enabled) and secrets-scanning (partial).
		{tier: 2, wantEnabled: 5, total: 6},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("tier %d", tt.tier), func(t *testing.T) {
			t.Parallel()
			cov := AssessDefenseLayers(dir, tools, types.DetectedProject{}, genState, tt.tier)
			if cov.Enabled != tt.wantEnabled || cov.Total != tt.total {
				t.Errorf("tier %d: %d/%d layers, want %d/%d", tt.tier, cov.Enabled, cov.Total, tt.wantEnabled, tt.total)
			}
			if tt.tier == 1 && cov.Score != 100 {
				t.Errorf("tier 1 score = %.1f, want 100 (all in-scope layers enabled)", cov.Score)
			}
		})
	}
}
