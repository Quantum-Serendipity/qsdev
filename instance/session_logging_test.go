package instance

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func TestExtractDebugFlag(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		want      []string
		wantDebug bool
	}{
		{
			name: "no debug flag",
			args: []string{"qsdev", "status"},
			want: []string{"qsdev", "status"},
		},
		{
			name:      "debug flag removed",
			args:      []string{"qsdev", "--debug", "status"},
			want:      []string{"qsdev", "status"},
			wantDebug: true,
		},
		{
			name: "debug after terminator passes through to child",
			args: []string{"qsdev", "sandbox", "exec", "--category", "lint", "--", "eslint", "--debug", "."},
			want: []string{"qsdev", "sandbox", "exec", "--category", "lint", "--", "eslint", "--debug", "."},
		},
		{
			name:      "debug before terminator is consumed, after is kept",
			args:      []string{"qsdev", "--debug", "sandbox", "exec", "--", "tool", "--debug"},
			want:      []string{"qsdev", "sandbox", "exec", "--", "tool", "--debug"},
			wantDebug: true,
		},
	}

	envVar := branding.Get().EnvLogVar
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(envVar, "")
			got := extractDebugFlag(tt.args)
			if !slices.Equal(got, tt.want) {
				t.Errorf("extractDebugFlag(%q) = %q, want %q", tt.args, got, tt.want)
			}
			if gotDebug := os.Getenv(envVar) == "debug"; gotDebug != tt.wantDebug {
				t.Errorf("debug logging enabled = %v, want %v", gotDebug, tt.wantDebug)
			}
		})
	}
}

// exitErr is an error carrying a process exit code, as gdev's ExitCodeErr.
type exitErr struct{ code int }

func (e exitErr) Error() string { return "denied" }
func (e exitErr) ExitCode() int { return e.code }

func TestInstrumentCommandErrors(t *testing.T) {
	errBoom := errors.New("boom")
	tests := []struct {
		name      string
		args      []string
		wantErr   error
		wantLog   []string
		wantNoLog bool
	}{
		{name: "run error is logged", args: []string{"fail"}, wantErr: errBoom, wantLog: []string{`"msg":"command failed"`, `"error":"boom"`, `"command":"app fail"`}},
		{name: "nested subcommand error is logged", args: []string{"group", "leaf"}, wantErr: errBoom, wantLog: []string{`"command":"app group leaf"`}},
		{name: "pre-run error is logged", args: []string{"prefail"}, wantErr: errBoom, wantLog: []string{`"error":"boom"`}},
		{name: "args validation error is logged", args: []string{"exact", "a", "b"}, wantLog: []string{`"msg":"command failed"`, "accepts 1 arg"}},
		{name: "exit code is recorded", args: []string{"deny"}, wantLog: []string{`"exit_code":2`}},
		{name: "success logs nothing", args: []string{"ok"}, wantNoLog: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
			t.Cleanup(func() { slog.SetDefault(prev) })

			root := newTestTree(errBoom)
			instrumentCommandErrors(root)
			root.SetArgs(tt.args)
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			err := root.Execute()

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Execute error = %v, want %v (errors must pass through unchanged)", err, tt.wantErr)
			}
			logged := buf.String()
			if tt.wantNoLog {
				if err != nil || logged != "" {
					t.Errorf("successful command: err=%v, logged %q", err, logged)
				}
				return
			}
			if err == nil {
				t.Fatal("expected the command to fail")
			}
			if n := strings.Count(logged, `"command failed"`); n != 1 {
				t.Errorf("failure logged %d time(s), want 1:\n%s", n, logged)
			}
			for _, want := range tt.wantLog {
				if !strings.Contains(logged, want) {
					t.Errorf("log missing %s:\n%s", want, logged)
				}
			}
		})
	}
}

func newTestTree(errBoom error) *cobra.Command {
	root := &cobra.Command{Use: "app", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(
		&cobra.Command{Use: "fail", RunE: func(*cobra.Command, []string) error { return errBoom }},
		&cobra.Command{Use: "ok", RunE: func(*cobra.Command, []string) error { return nil }},
		&cobra.Command{
			Use:     "prefail",
			PreRunE: func(*cobra.Command, []string) error { return errBoom },
			RunE:    func(*cobra.Command, []string) error { return nil },
		},
		&cobra.Command{Use: "exact", Args: cobra.ExactArgs(1), RunE: func(*cobra.Command, []string) error { return nil }},
		&cobra.Command{Use: "deny", RunE: func(*cobra.Command, []string) error { return exitErr{code: 2} }},
	)
	group := &cobra.Command{Use: "group"}
	group.AddCommand(&cobra.Command{Use: "leaf", RunE: func(*cobra.Command, []string) error { return errBoom }})
	root.AddCommand(group)
	return root
}
