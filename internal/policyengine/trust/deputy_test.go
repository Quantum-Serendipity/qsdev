package trust

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckAccess(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "secret.env")
	if err := os.WriteFile(secretFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	denyRules := []DenyRule{
		{Pattern: tmpDir + "/*", Type: "path"},
	}

	tests := []struct {
		name      string
		tool      string
		args      map[string]string
		rules     []DenyRule
		wantBlock bool
	}{
		{
			name:      "known tool accessing denied path",
			tool:      "mcp__filesystem__read_file",
			args:      map[string]string{"path": secretFile},
			rules:     denyRules,
			wantBlock: true,
		},
		{
			name:      "unknown mcp tool not blocked",
			tool:      "mcp__custom__do_stuff",
			args:      map[string]string{"path": secretFile},
			rules:     denyRules,
			wantBlock: false,
		},
		{
			name:      "non-mcp tool not checked",
			tool:      "Read",
			args:      map[string]string{"path": secretFile},
			rules:     denyRules,
			wantBlock: false,
		},
		{
			name:      "known tool accessing allowed path",
			tool:      "mcp__filesystem__read_file",
			args:      map[string]string{"path": "/tmp/allowed.txt"},
			rules:     denyRules,
			wantBlock: false,
		},
		{
			name:      "empty deny rules",
			tool:      "mcp__filesystem__read_file",
			args:      map[string]string{"path": secretFile},
			rules:     nil,
			wantBlock: false,
		},
		{
			name:      "github create_or_update_file denied",
			tool:      "mcp__github__create_or_update_file",
			args:      map[string]string{"path": secretFile},
			rules:     denyRules,
			wantBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			argsJSON, err := json.Marshal(tt.args)
			if err != nil {
				t.Fatalf("marshaling args: %v", err)
			}

			blocked, reason := CheckAccess(tt.tool, argsJSON, tt.rules)

			if blocked != tt.wantBlock {
				t.Errorf("blocked = %v, want %v (reason: %s)", blocked, tt.wantBlock, reason)
			}

			if blocked && reason == "" {
				t.Error("blocked but no reason provided")
			}
		})
	}
}

func TestCheckAccessPathCanonicalization(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "a", "b")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("creating nested dir: %v", err)
	}
	testFile := filepath.Join(nestedDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	denyRules := []DenyRule{
		{Pattern: tmpDir + "/*", Type: "path"},
	}

	// Use a relative-style path with ..
	relativePath := filepath.Join(nestedDir, "..", "..", "a", "b", "file.txt")

	args, _ := json.Marshal(map[string]string{"path": relativePath})

	blocked, _ := CheckAccess("mcp__filesystem__read_file", args, denyRules)
	if !blocked {
		t.Error("relative path should resolve and be blocked")
	}
}

func TestCheckAccessEmptyArgs(t *testing.T) {
	t.Parallel()

	denyRules := []DenyRule{{Pattern: "/secret/*", Type: "path"}}

	blocked, _ := CheckAccess("mcp__filesystem__read_file", nil, denyRules)
	if blocked {
		t.Error("nil args should not block")
	}

	blocked, _ = CheckAccess("mcp__filesystem__read_file", json.RawMessage("{}"), denyRules)
	if blocked {
		t.Error("empty object args should not block")
	}
}

func TestCheckAccessExactPathMatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	exactFile := filepath.Join(tmpDir, "exact.txt")
	if err := os.WriteFile(exactFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}

	denyRules := []DenyRule{
		{Pattern: exactFile, Type: "path"},
	}

	args, _ := json.Marshal(map[string]string{"path": exactFile})

	blocked, _ := CheckAccess("mcp__filesystem__read_file", args, denyRules)
	if !blocked {
		t.Error("exact path match should block")
	}
}
