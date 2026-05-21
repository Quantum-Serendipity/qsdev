package devinit

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
	"github.com/spf13/cobra"
)

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
