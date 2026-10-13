package logcmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestResolveLogDir checks that `logs` reads the project log tier of the
// project the command resolved, and the global tier outside a project or
// with --global.
func TestResolveLogDir(t *testing.T) {
	logDir := t.TempDir()
	t.Setenv(branding.Get().EnvLogDirVar, logDir)
	project := filepath.Join(t.TempDir(), "proj")

	tests := []struct {
		name   string
		pc     projectctx.Context
		global bool
		want   string
	}{
		{name: "project found", pc: projectctx.Context{Root: project, Found: true}, want: logging.ProjectLogDir(project)},
		{name: "no project", pc: projectctx.Context{Root: project}, want: logDir},
		{name: "global flag", pc: projectctx.Context{Root: project, Found: true}, global: true, want: logDir},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command()
			cmd.SetContext(projectctx.WithContext(context.Background(), tt.pc))
			var args []string
			if tt.global {
				args = []string{"--global"}
			}
			if err := cmd.ParseFlags(args); err != nil {
				t.Fatal(err)
			}
			if got := resolveLogDir(cmd); got != tt.want {
				t.Errorf("resolveLogDir() = %q, want %q", got, tt.want)
			}
		})
	}
}
