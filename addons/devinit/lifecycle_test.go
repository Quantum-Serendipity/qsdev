package devinit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
	"github.com/spf13/cobra"
)

// TestWriteToolFiles_AlwaysOnAgentPostmortem_StandardTier is the BL-P1-9
// regression: enabling the always-on agent-postmortem tool at the standard tier
// must write its exclusive SKILL.md (RED before the fix — silently omitted).
func TestWriteToolFiles_AlwaysOnAgentPostmortem_StandardTier(t *testing.T) {
	root := t.TempDir()
	tool, ok := toolreg.DefaultRegistry().ByName("agent-postmortem")
	if !ok {
		t.Fatal("agent-postmortem not registered")
	}
	answers := types.WizardAnswers{
		Tier:         "standard",
		ProjectRoot:  root,
		AgentTools:   types.AgentToolsAnswers{PostmortemEnabled: true},
		EnabledTools: map[string]bool{},
	}

	written, err := writeToolFiles(tool, "agent-postmortem", root, answers)
	if err != nil {
		t.Fatalf("writeToolFiles: %v", err)
	}

	skillPath := filepath.Join(root, ".claude", "skills", "agent-postmortem", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("expected SKILL.md written at standard tier, got: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("---\n")) {
		t.Errorf("generated SKILL.md must carry YAML front-matter, got:\n%.80s", data)
	}

	var sawSkill bool
	for _, f := range written {
		if f.Path == ".claude/skills/agent-postmortem/SKILL.md" {
			sawSkill = true
		}
	}
	if !sawSkill {
		t.Error("SKILL.md not reported in written files")
	}
}

// TestWriteToolFiles_OptInBelowFull_RefusesInsteadOfFalseSuccess is the BL-P1-9
// honesty guard: enabling a Full-gated opt-in tool (lookup-docs) below Full must
// refuse with an actionable tier error and write nothing — never report success
// while silently omitting the SKILL.md and leaving CLAUDE.md advertising it.
func TestWriteToolFiles_OptInBelowFull_RefusesInsteadOfFalseSuccess(t *testing.T) {
	root := t.TempDir()
	tool, ok := toolreg.DefaultRegistry().ByName("lookup-docs")
	if !ok {
		t.Fatal("lookup-docs not registered")
	}
	answers := types.WizardAnswers{
		Tier:         "standard",
		ProjectRoot:  root,
		EnabledTools: map[string]bool{},
	}

	_, err := writeToolFiles(tool, "lookup-docs", root, answers)
	if err == nil {
		t.Fatal("expected an actionable error enabling a Full-gated tool below Full, got nil")
	}
	if !strings.Contains(err.Error(), "tier") {
		t.Errorf("error should mention the tier requirement, got: %v", err)
	}

	// Nothing may be written on refusal — no advertisement, no missing SKILL.md.
	if _, statErr := os.Stat(filepath.Join(root, "CLAUDE.md")); statErr == nil {
		t.Error("CLAUDE.md must not be written when enable refuses")
	}
	if _, statErr := os.Stat(filepath.Join(root, ".claude", "skills", "lookup-docs", "SKILL.md")); statErr == nil {
		t.Error("lookup-docs SKILL.md must be absent when enable refuses")
	}
}

func TestPrintWrittenFiles_Empty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	printWrittenFiles(cmd, nil, "wrote")

	if buf.Len() != 0 {
		t.Errorf("expected no output for empty files, got %q", buf.String())
	}
}

func TestPrintWrittenFiles_MultipleFiles(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	files := []types.GeneratedFile{
		{Path: "secretspec.toml"},
		{Path: ".claude/settings.json"},
	}
	printWrittenFiles(cmd, files, "wrote")

	out := buf.String()
	if !strings.Contains(out, "Files wrote:") {
		t.Error("output should contain 'Files wrote:' header")
	}
	if !strings.Contains(out, "secretspec.toml") {
		t.Error("output should list secretspec.toml")
	}
	if !strings.Contains(out, ".claude/settings.json") {
		t.Error("output should list .claude/settings.json")
	}
}

func TestPrintWrittenFiles_VerbParam(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	files := []types.GeneratedFile{{Path: "test.txt"}}
	printWrittenFiles(cmd, files, "would write")

	if !strings.Contains(buf.String(), "Files would write:") {
		t.Errorf("expected 'Files would write:' header, got %q", buf.String())
	}
}

func TestIsMachineReadableFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		format check.OutputFormat
		want   bool
	}{
		{check.FormatJSON, true},
		{check.FormatSARIF, true},
		{check.FormatJUnit, true},
		{check.FormatHuman, false},
		{check.OutputFormat("unknown"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			t.Parallel()
			if got := isMachineReadableFormat(tt.format); got != tt.want {
				t.Errorf("isMachineReadableFormat(%q) = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}
