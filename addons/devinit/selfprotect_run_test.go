package devinit

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
)

func executeSelfprotect(t *testing.T, stdin string) (string, error) {
	t.Helper()
	cmd := selfprotectCmd()
	var stderr bytes.Buffer
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&stderr)
	cmd.SetArgs(nil)
	err := cmd.Execute()
	return stderr.String(), err
}

// TestRunSelfprotect_VerdictsInProcess drives the real hook entry point in
// process: every deny path must return exit code 2 (not os.Exit) with the
// reason on stderr, and an allowed call must return nil.
func TestRunSelfprotect_VerdictsInProcess(t *testing.T) {
	protected := filepath.Join(t.TempDir(), ".claude", "hooks", "guard.sh")
	payload := func(tool string, input map[string]string) string {
		data, err := json.Marshal(map[string]any{"tool_name": tool, "tool_input": input})
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	tests := []struct {
		name       string
		stdin      string
		wantDeny   bool
		wantStderr string
	}{
		{
			name:       "malformed input is denied",
			stdin:      "not json",
			wantDeny:   true,
			wantStderr: "internal error",
		},
		{
			name:       "write to a protected hook script is denied",
			stdin:      payload("Write", map[string]string{"file_path": protected, "content": "exit 0"}),
			wantDeny:   true,
			wantStderr: "qsdev-selfprotect:",
		},
		{
			name:  "benign write is allowed",
			stdin: payload("Write", map[string]string{"file_path": filepath.Join(t.TempDir(), "notes.txt"), "content": "hello"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stderr, err := executeSelfprotect(t, tt.stdin)
			if !tt.wantDeny {
				if err != nil {
					t.Fatalf("expected allow, got %v (stderr %q)", err, stderr)
				}
				return
			}
			var exitErr *ExitError
			if !errors.As(err, &exitErr) || exitErr.Code != 2 {
				t.Fatalf("expected exit code 2, got %v (stderr %q)", err, stderr)
			}
			if exitErr.Error() != "" {
				t.Errorf("deny error message = %q, want empty so nothing is appended to stderr", exitErr.Error())
			}
			if !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("stderr %q does not contain %q", stderr, tt.wantStderr)
			}
		})
	}
}

// TestBuildSelfprotectContext_CanonicalizationFailureStaysProtected pins the
// fail-closed fallback: when a protected target cannot be canonicalized (here
// an ancestor directory is unreadable), CanonicalPath must not be left empty,
// which would make every path rule allow the write.
func TestBuildSelfprotectContext_CanonicalizationFailureStaysProtected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission bits do not block traversal on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}

	locked := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	target := filepath.Join(locked, ".claude", "hooks", "guard.sh")
	ctx := buildSelfprotectContext("Write", &hookio.ToolInput{FilePath: target, Content: "exit 0"})

	if ctx.CanonicalPath == "" {
		t.Fatal("CanonicalPath is empty after a canonicalization failure")
	}
	if v, _ := rules.Tier1Rules.EvaluateAll(ctx); v != rules.Deny {
		t.Errorf("write to protected path behind an unreadable directory = %v, want Deny", v)
	}
}
