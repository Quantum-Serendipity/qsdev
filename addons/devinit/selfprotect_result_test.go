package devinit

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
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

		{"edit replacing the registry", "Edit", hookio.ToolInput{FilePath: npmrc, OldString: "https://r", NewString: "https://s"}, npmrc, true, "GD-004"},

		{"edit keeps ignore-scripts", "Edit", hookio.ToolInput{FilePath: npmrc, OldString: "registry=https://r\n", NewString: "registry=https://r\nsave-exact=true\n"}, npmrc, false, ""},
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

// sparseFile creates path as a sparse file of size bytes.
func sparseFile(t *testing.T, path string, size int64) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(size); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestDetectResultGateDodge_OversizeDenies pins U05-14's size cap: a guarded
// file larger than rules.MaxGuardedFileBytes is not read whole into memory
// and the call is denied, while a file of exactly the cap is still read (and
// here allowed, since a Write of the same content removes nothing).
func TestDetectResultGateDodge_OversizeDenies(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	over := filepath.Join(dir, "over", ".npmrc")
	if err := os.Mkdir(filepath.Dir(over), 0o755); err != nil {
		t.Fatal(err)
	}
	sparseFile(t, over, rules.MaxGuardedFileBytes+1)
	blocked, ruleID, reason := detectResultGateDodge("Write", hookio.ToolInput{FilePath: over, Content: "registry=https://x\n"}, over)
	if !blocked || ruleID != "GD-004" || !strings.Contains(reason, "too large") {
		t.Errorf("oversize .npmrc = (%v, %q, %q), want GD-004 deny naming 'too large'", blocked, ruleID, reason)
	}

	at := filepath.Join(dir, "at", ".npmrc")
	if err := os.Mkdir(filepath.Dir(at), 0o755); err != nil {
		t.Fatal(err)
	}
	sparseFile(t, at, rules.MaxGuardedFileBytes)
	content := strings.Repeat("\x00", int(rules.MaxGuardedFileBytes))
	if blocked, ruleID, reason := detectResultGateDodge("Write", hookio.ToolInput{FilePath: at, Content: content}, at); blocked {
		t.Errorf(".npmrc of exactly the cap = (%v, %q, %q), want it read and allowed", blocked, ruleID, reason)
	}
}

// TestSelfprotect_QsdevYAMLOversizeDeniesGD001 pins that the GD-001 check
// (rules.FileChange) refuses a .qsdev.yaml over the size cap.
func TestSelfprotect_QsdevYAMLOversizeDeniesGD001(t *testing.T) {
	target := filepath.Join(t.TempDir(), ".qsdev.yaml")
	sparseFile(t, target, rules.MaxGuardedFileBytes+1)
	assertSelfprotectDenies(t, writePayload(t, target, "security:\n  level: strict\n"), "GD-001")
}

// writePayload is a Write hook payload for target.
func writePayload(t *testing.T, target, content string) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"tool_name":  "Write",
		"tool_input": map[string]any{"file_path": target, "content": content},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// assertSelfprotectDenies runs the hook in process on stdin and requires a
// deny by ruleID, not by the SP-TIMEOUT backstop.
func assertSelfprotectDenies(t *testing.T, stdin, ruleID string) {
	t.Helper()
	stderr, err := executeSelfprotect(t, stdin)
	if !errors.Is(err, errSelfprotectDeny) {
		t.Fatalf("err = %v, want errSelfprotectDeny (stderr: %q)", err, stderr)
	}
	if !strings.Contains(stderr, ruleID) || strings.Contains(stderr, "SP-TIMEOUT") {
		t.Errorf("stderr %q: want %s and no SP-TIMEOUT", stderr, ruleID)
	}
}
