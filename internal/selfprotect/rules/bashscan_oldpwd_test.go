package rules

import (
	"path/filepath"
	"testing"
)

// TestScannedCommands_CdBack pins where `cd -` lands (dirState.noteOldPWD):
// the directory the line's last directory change left, with its protected
// flag, when that change surely ran before it through && lists only, and an
// unknown directory otherwise (the line's starting OLDPWD is not known).
func TestScannedCommands_CdBack(t *testing.T) {
	t.Parallel()
	home := homeDir(t)
	project := filepath.Join(home, "project")
	claude := filepath.Join(home, ".claude")
	tests := []struct {
		name, command string
		wantCwd       string // "" means unknown
		wantProtected bool
	}{
		{"back after a cd", "cd .. && cd - && ls", project, false},
		{"back into a protected directory", "cd ~/.claude && cd /tmp && cd - && ls", claude, true},
		{"back out of a protected directory", "cd ~/.claude && cd - && ls", project, false},
		{"back twice", "cd /tmp && cd - && cd - && ls", "/tmp", false},
		{"back after a command between", "cd /tmp && ls && cd - && ls", project, false},
		{"back after a pushd", "pushd /tmp && cd - && ls", project, false},
		{"no cd before", "cd - && ls", "", false},
		{"after a cd that may fail", "cd /tmp; cd - && ls", "", false},
		{"after a cd in a subshell", "(cd /tmp) && cd - && ls", "", false},
		{"after a cd that may be skipped", "true || cd /tmp && cd - && ls", "", false},
		{"after a command after ;", "cd /tmp && ls; true && cd - && ls", "", false},
		{"after OLDPWD is set", "cd /tmp && OLDPWD=/x && cd - && ls", "", false},
		{"with OLDPWD as a prefix", "cd /tmp && OLDPWD=/x cd - && ls", "", false},
		{"cd - itself after ;", "cd /tmp; cd -; ls", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: tt.command, CWD: project}
			for _, scs := range [][]scannedCommand{mustScan(t, &ctx), mainShellDirs(&ctx, mustScan(t, &ctx))} {
				last := scs[len(scs)-1]
				if tt.wantCwd == "" {
					if !last.cwdUnknown {
						t.Errorf("%q: last command runs in %q, want an unknown directory", tt.command, last.cwd)
					}
					continue
				}
				if last.cwdUnknown || last.cwd != tt.wantCwd || last.inProtectedDir != tt.wantProtected {
					t.Errorf("%q: last command runs in %q (unknown %v, protected %v), want %q (protected %v)",
						tt.command, last.cwd, last.cwdUnknown, last.inProtectedDir, tt.wantCwd, tt.wantProtected)
				}
			}
		})
	}
}

// mustScan returns ctx's scanned commands.
func mustScan(t *testing.T, ctx *EvalContext) []scannedCommand {
	t.Helper()
	scs, err := ctx.scannedCommands()
	if err != nil {
		t.Fatalf("scanning %q: %v", ctx.Command, err)
	}
	return scs
}
