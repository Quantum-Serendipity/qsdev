package devinit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
)

// TestDetectResultGateDodge verifies that a Write/Edit which only removes a
// protective setting is blocked. Such a call introduces no text (an Edit to an
// empty new_string), so gatedodge.Detect on the edited content never saw it.
func TestDetectResultGateDodge(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	npmrc := filepath.Join(dir, ".npmrc")
	if err := os.WriteFile(npmrc, []byte("registry=https://r\nignore-scripts=true\n"), 0o644); err != nil {
		t.Fatalf("writing .npmrc: %v", err)
	}
	shared := filepath.Join(dir, "npmrc-shared")
	if err := os.WriteFile(shared, []byte("ignore-scripts=true\n"), 0o644); err != nil {
		t.Fatalf("writing shared npmrc: %v", err)
	}
	missing := filepath.Join(dir, "sub", ".npmrc")
	other := filepath.Join(dir, "main.go")

	tests := []struct {
		name    string
		tool    string
		input   hookio.ToolInput
		path    string
		blocked bool
		ruleID  string
	}{
		{"edit deletes ignore-scripts", "Edit", hookio.ToolInput{FilePath: npmrc, OldString: "ignore-scripts=true\n"}, npmrc, true, "GD-004"},
		{"empty write", "Write", hookio.ToolInput{FilePath: npmrc}, npmrc, true, "GD-004"},
		{"multiedit flips value", "MultiEdit", hookio.ToolInput{FilePath: npmrc, Edits: []hookio.EditOp{{OldString: "=true", NewString: "=0"}}}, npmrc, true, "GD-004"},
		{"edit that cannot be applied exactly", "Edit", hookio.ToolInput{FilePath: npmrc, OldString: "ignore-scripts = true", NewString: ""}, npmrc, true, "GD-004"},
		{"guarded name resolving to another name", "Edit", hookio.ToolInput{FilePath: filepath.Join(dir, "link", ".npmrc"), OldString: "ignore-scripts=true\n"}, shared, true, "GD-004"},
		{"raw path when canonicalization failed", "Edit", hookio.ToolInput{FilePath: npmrc, OldString: "ignore-scripts=true\n"}, "", true, "GD-004"},

		{"edit keeps ignore-scripts", "Edit", hookio.ToolInput{FilePath: npmrc, OldString: "https://r", NewString: "https://s"}, npmrc, false, ""},
		{"new file without the setting", "Write", hookio.ToolInput{FilePath: missing, Content: "registry=https://r\n"}, missing, false, ""},
		{"unguarded file", "Write", hookio.ToolInput{FilePath: other}, other, false, ""},
		{"read tool", "Read", hookio.ToolInput{FilePath: npmrc}, npmrc, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, ruleID, reason := detectResultGateDodge(tt.tool, tt.input, tt.path)
			if blocked != tt.blocked || ruleID != tt.ruleID {
				t.Errorf("detectResultGateDodge = (%v, %q, %q), want (%v, %q)", blocked, ruleID, reason, tt.blocked, tt.ruleID)
			}
		})
	}
}
