package devinit_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestBuildPlanPreview_BasicDevenv(t *testing.T) {
	fs := devinit.NewExportFormState(
		devinit.WithSelectedLanguages([]string{"go"}),
		devinit.WithDirenv(true),
	)

	preview := devinit.ExportBuildPlanPreview(fs)

	if !strings.Contains(preview, "devenv.yaml") {
		t.Error("preview should mention devenv.yaml")
	}
	if !strings.Contains(preview, "devenv.nix") {
		t.Error("preview should mention devenv.nix")
	}
	if !strings.Contains(preview, ".envrc") {
		t.Error("preview should mention .envrc when direnv is enabled")
	}
	if !strings.Contains(preview, "Go") {
		t.Error("preview devenv.nix description should mention Go")
	}
}

func TestBuildPlanPreview_WithClaude(t *testing.T) {
	fs := devinit.NewExportFormState(
		devinit.WithSelectedLanguages([]string{"go"}),
		devinit.WithClaudeCode(true),
		devinit.WithPermissionLevel("standard"),
		devinit.WithSafetyBlock(true),
	)

	preview := devinit.ExportBuildPlanPreview(fs)

	if !strings.Contains(preview, "CLAUDE.md") {
		t.Error("preview should mention CLAUDE.md when Claude is enabled")
	}
	if !strings.Contains(preview, ".claude/settings.json") {
		t.Error("preview should mention .claude/settings.json when Claude is enabled")
	}
	if !strings.Contains(preview, "standard permissions") {
		t.Error("preview should show permission level")
	}
	if !strings.Contains(preview, ".claude/rules/") {
		t.Error("preview should mention .claude/rules/ when Claude is enabled")
	}
	if !strings.Contains(preview, "safety-block") {
		t.Error("preview should mention safety-block hook when enabled")
	}
}

func TestBuildPlanPreview_WithServices(t *testing.T) {
	fs := devinit.NewExportFormState(
		devinit.WithSelectedLanguages([]string{"go"}),
		devinit.WithSelectedServices([]string{"postgres", "redis"}),
	)

	preview := devinit.ExportBuildPlanPreview(fs)

	if !strings.Contains(preview, "PostgreSQL") {
		t.Error("preview should mention PostgreSQL service")
	}
	if !strings.Contains(preview, "Redis") {
		t.Error("preview should mention Redis service")
	}
}

func TestBuildPlanPreview_NoClaude(t *testing.T) {
	fs := devinit.NewExportFormState(
		devinit.WithSelectedLanguages([]string{"go"}),
		devinit.WithClaudeCode(false),
	)

	preview := devinit.ExportBuildPlanPreview(fs)

	if strings.Contains(preview, "CLAUDE.md") {
		t.Error("preview should not contain CLAUDE.md when Claude is disabled")
	}
	if strings.Contains(preview, ".claude/") {
		t.Error("preview should not contain .claude/ paths when Claude is disabled")
	}
}

func TestBuildPlanPreview_WithSkills(t *testing.T) {
	fs := devinit.NewExportFormState(
		devinit.WithClaudeCode(true),
		devinit.WithPermissionLevel("standard"),
		devinit.WithSkills([]string{"deploy", "review-pr"}),
	)

	preview := devinit.ExportBuildPlanPreview(fs)

	if !strings.Contains(preview, ".claude/skills/") {
		t.Error("preview should mention .claude/skills/ when skills are selected")
	}
	if !strings.Contains(preview, "deploy") {
		t.Error("preview should list selected skill names")
	}
}

func TestBuildPlanPreview_WithMCP(t *testing.T) {
	fs := devinit.NewExportFormState(
		devinit.WithClaudeCode(true),
		devinit.WithPermissionLevel("standard"),
		devinit.WithMCPServers([]string{"github", "filesystem"}),
	)

	preview := devinit.ExportBuildPlanPreview(fs)

	if !strings.Contains(preview, ".mcp.json") {
		t.Error("preview should mention .mcp.json when MCP servers are selected")
	}
}

func TestBuildPlanPreview_NoDirenv(t *testing.T) {
	fs := devinit.NewExportFormState(
		devinit.WithSelectedLanguages([]string{"go"}),
		devinit.WithDirenv(false),
	)

	preview := devinit.ExportBuildPlanPreview(fs)

	if strings.Contains(preview, ".envrc") {
		t.Error("preview should not mention .envrc when direnv is disabled")
	}
}

func TestBuildPlanPreview_WithNixHardeningGuide(t *testing.T) {
	t.Parallel()

	fs := devinit.NewExportFormState(
		devinit.WithSelectedLanguages([]string{"go"}),
		devinit.WithNixHardeningGuide(true),
	)

	preview := devinit.ExportBuildPlanPreview(fs)

	if !strings.Contains(preview, "nix-hardening.md") {
		t.Error("preview should mention nix-hardening.md when nix hardening guide is enabled")
	}
}

func TestBuildPlanPreview_WithoutNixHardeningGuide(t *testing.T) {
	t.Parallel()

	fs := devinit.NewExportFormState(
		devinit.WithSelectedLanguages([]string{"go"}),
		devinit.WithNixHardeningGuide(false),
	)

	preview := devinit.ExportBuildPlanPreview(fs)

	if strings.Contains(preview, "nix-hardening.md") {
		t.Error("preview should not mention nix-hardening.md when nix hardening guide is disabled")
	}
}

func TestBuildDetailedDefaults_GoProject(t *testing.T) {
	t.Parallel()

	detected := types.DetectedProject{
		HasGoMod:  true,
		GoVersion: "1.24",
	}
	defaults := devinit.ExportMapDetectionToDefaults(detected, "/tmp/project")

	output := devinit.ExportBuildDetailedDefaults(detected, defaults)

	if !strings.Contains(output, "Go") {
		t.Error("output should contain Go language")
	}
	if !strings.Contains(output, "1.24") {
		t.Error("output should contain Go version 1.24")
	}
	if !strings.Contains(output, "direnv") {
		t.Error("output should mention direnv")
	}
}

func TestBuildDetailedDefaults_EmptyDetection(t *testing.T) {
	t.Parallel()

	detected := types.DetectedProject{}
	defaults := devinit.ExportMapDetectionToDefaults(detected, "/tmp/project")

	output := devinit.ExportBuildDetailedDefaults(detected, defaults)

	if !strings.Contains(output, "(none") {
		t.Errorf("output should contain '(none' for empty languages, got:\n%s", output)
	}
}
