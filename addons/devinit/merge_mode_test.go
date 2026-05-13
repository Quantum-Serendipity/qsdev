package devinit_test

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devinit"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestDetectExistingConfig_NoExisting(t *testing.T) {
	detected := types.DetectedProject{}
	ec := devinit.ExportDetectExistingConfig(detected)

	if ec.HasDevenv {
		t.Error("expected HasDevenv to be false")
	}
	if ec.HasClaude {
		t.Error("expected HasClaude to be false")
	}
	if ec.HasEnvrc {
		t.Error("expected HasEnvrc to be false")
	}
	if ec.HasMcp {
		t.Error("expected HasMcp to be false")
	}
	if len(ec.Files) != 0 {
		t.Errorf("expected no files, got %v", ec.Files)
	}
	if ec.NeedsMergeMode() {
		t.Error("expected NeedsMergeMode() to be false with no existing config")
	}
}

func TestDetectExistingConfig_DevenvOnly(t *testing.T) {
	detected := types.DetectedProject{
		HasDevenvNix: true,
	}
	ec := devinit.ExportDetectExistingConfig(detected)

	if !ec.HasDevenv {
		t.Error("expected HasDevenv to be true")
	}
	if ec.HasClaude {
		t.Error("expected HasClaude to be false")
	}
	if !ec.NeedsMergeMode() {
		t.Error("expected NeedsMergeMode() to be true with devenv detected")
	}
	found := false
	for _, f := range ec.Files {
		if f == "devenv.nix" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'devenv.nix' in files, got %v", ec.Files)
	}
}

func TestDetectExistingConfig_DevenvYaml(t *testing.T) {
	detected := types.DetectedProject{
		HasDevenvYaml: true,
	}
	ec := devinit.ExportDetectExistingConfig(detected)

	if !ec.HasDevenv {
		t.Error("expected HasDevenv to be true")
	}
	found := false
	for _, f := range ec.Files {
		if f == "devenv.yaml" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'devenv.yaml' in files, got %v", ec.Files)
	}
}

func TestDetectExistingConfig_ClaudeOnly(t *testing.T) {
	detected := types.DetectedProject{
		HasClaudeDir: true,
		HasClaudeMd:  true,
	}
	ec := devinit.ExportDetectExistingConfig(detected)

	if !ec.HasClaude {
		t.Error("expected HasClaude to be true")
	}
	if ec.HasDevenv {
		t.Error("expected HasDevenv to be false")
	}
	if !ec.NeedsMergeMode() {
		t.Error("expected NeedsMergeMode() to be true with claude detected")
	}
	if len(ec.Files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(ec.Files), ec.Files)
	}
}

func TestDetectExistingConfig_FullExisting(t *testing.T) {
	detected := types.DetectedProject{
		HasDevenvNix:      true,
		HasDevenvYaml:     true,
		HasClaudeDir:      true,
		HasClaudeMd:       true,
		HasClaudeSettings: true,
		HasEnvrc:          true,
		HasMcpJson:        true,
	}
	ec := devinit.ExportDetectExistingConfig(detected)

	if !ec.HasDevenv {
		t.Error("expected HasDevenv to be true")
	}
	if !ec.HasClaude {
		t.Error("expected HasClaude to be true")
	}
	if !ec.HasEnvrc {
		t.Error("expected HasEnvrc to be true")
	}
	if !ec.HasMcp {
		t.Error("expected HasMcp to be true")
	}
	if !ec.NeedsMergeMode() {
		t.Error("expected NeedsMergeMode() to be true with full config")
	}
	// devenv.nix, devenv.yaml, .claude/, CLAUDE.md, .claude/settings.json, .envrc, .mcp.json
	if len(ec.Files) != 7 {
		t.Errorf("expected 7 files, got %d: %v", len(ec.Files), ec.Files)
	}
}

func TestDetectExistingConfig_McpOnly(t *testing.T) {
	detected := types.DetectedProject{
		HasMcpJson: true,
	}
	ec := devinit.ExportDetectExistingConfig(detected)

	if !ec.HasMcp {
		t.Error("expected HasMcp to be true")
	}
	if !ec.NeedsMergeMode() {
		t.Error("expected NeedsMergeMode() to be true with MCP detected")
	}
}

func TestDetectExistingConfig_EnvrcOnly(t *testing.T) {
	detected := types.DetectedProject{
		HasEnvrc: true,
	}
	ec := devinit.ExportDetectExistingConfig(detected)

	if !ec.HasEnvrc {
		t.Error("expected HasEnvrc to be true")
	}
	if !ec.NeedsMergeMode() {
		t.Error("expected NeedsMergeMode() to be true with .envrc detected")
	}
}
