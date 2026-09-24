//go:build !windows

package toolcheck_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
)

// writeTool writes an executable shell script and returns its path.
func writeTool(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestDetect_TimeoutNotDefeatedByGrandchild verifies the probe timeout holds
// when the tool is a wrapper whose child process keeps the output pipe open
// after the wrapper itself is killed.
func TestDetect_TimeoutNotDefeatedByGrandchild(t *testing.T) {
	t.Parallel()

	tool := writeTool(t, "sleep 10\necho v1\n")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	info := toolcheck.Detect(ctx, tool, "--version")
	elapsed := time.Since(start)

	if !info.Found {
		t.Error("expected the tool to be found")
	}
	if elapsed > 5*time.Second {
		t.Errorf("Detect took %v; the timeout was not enforced", elapsed)
	}
}

func TestDetect_Output(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		script      string
		wantVersion string
		wantOutput  string // substring of Output
		notInOutput string
	}{
		{
			name:        "stderr warning does not displace version",
			script:      "echo 'warning: something' >&2\necho '1.2.3'\n",
			wantVersion: "1.2.3",
			wantOutput:  "1.2.3",
			notInOutput: "warning",
		},
		{
			name:        "multi-line output is kept whole",
			script:      "echo 'ShellCheck - shell script analysis tool'\necho 'version: 0.11.0'\n",
			wantVersion: "ShellCheck - shell script analysis tool",
			wantOutput:  "version: 0.11.0",
		},
		{
			name:        "version printed on stderr only",
			script:      "echo 'tool 4.5.6' >&2\n",
			wantVersion: "tool 4.5.6",
			wantOutput:  "tool 4.5.6",
		},
		{
			name:        "leading blank lines skipped",
			script:      "echo\necho '  7.8.9  '\n",
			wantVersion: "7.8.9",
			wantOutput:  "7.8.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			info := toolcheck.Detect(context.Background(), writeTool(t, tt.script), "--version")
			if info.Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", info.Version, tt.wantVersion)
			}
			if !strings.Contains(info.Output, tt.wantOutput) {
				t.Errorf("Output = %q, want it to contain %q", info.Output, tt.wantOutput)
			}
			if tt.notInOutput != "" && strings.Contains(info.Output, tt.notInOutput) {
				t.Errorf("Output = %q, must not contain %q", info.Output, tt.notInOutput)
			}
		})
	}
}
