package devinit

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/outdated"
	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
)

// TestRunOutdated_CorruptAnswersReported verifies a broken answers file is
// reported instead of silently falling back to probing every ecosystem.
func TestRunOutdated_CorruptAnswersReported(t *testing.T) {
	dir := t.TempDir()
	writeAnswersFile(t, dir, "languages: [unterminated\n")
	t.Chdir(dir)
	t.Setenv("PATH", t.TempDir())

	err := runOutdated(outdatedCmd(), outdated.OutdatedOptions{})
	if err == nil || !strings.Contains(err.Error(), "loading saved answers") {
		t.Errorf("runOutdated() error = %v, want a saved-answers load error", err)
	}
}

// TestOutdated_RequiresOnline proves `outdated` contacts no registry unless
// asked: without --online it starts nothing, names each command it would run
// and fails with an error naming --online, so a CI step cannot pass while
// checking nothing. With --online the selected command runs as before.
func TestOutdated_RequiresOnline(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantRun bool
		wantErr string
	}{
		{name: "offline by default", args: []string{"--ecosystem", "go"}, wantErr: "--online"},
		{name: "online runs the command", args: []string{"--ecosystem", "go", "--online"}, wantRun: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			binDir := t.TempDir()
			marker := filepath.Join(t.TempDir(), "ran")
			writeFakeExecutable(t, binDir, "go", fmt.Sprintf("echo \"$@\" > %q\n", marker))
			t.Setenv("PATH", binDir)
			t.Chdir(dir)
			if !tt.wantRun {
				t.Setenv(procexec.ForbidExecEnv, "1")
			}

			cmd := outdatedCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.wantErr == "" && err != nil {
				t.Fatalf("outdated %v: %v\n%s", tt.args, err, out.String())
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("outdated %v error = %v, want one naming %q", tt.args, err, tt.wantErr)
			}
			if tt.wantErr != "" && !errors.Is(err, outdated.ErrOnlineRequired) {
				t.Errorf("error %v does not wrap ErrOnlineRequired", err)
			}
			ran, statErr := os.ReadFile(marker)
			if (statErr == nil) != tt.wantRun {
				t.Fatalf("go stub ran = %v, want %v", statErr == nil, tt.wantRun)
			}
			if tt.wantRun && strings.TrimSpace(string(ran)) != "list -m -u all" {
				t.Errorf("go stub argv = %q, want %q", ran, "list -m -u all")
			}
			if !tt.wantRun && !strings.Contains(out.String(), "would run: go list -m -u all") {
				t.Errorf("output does not name the command: %q", out.String())
			}
		})
	}
}
