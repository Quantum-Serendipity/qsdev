package cmdutil

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// ttyReader is stdin a test presents as an interactive terminal, or not.
type ttyReader struct {
	io.Reader
	tty bool
}

func (r ttyReader) IsTerminal() bool { return r.tty }

// sensitiveTree builds a root with the gate installed and one command per
// kind of Sensitivity; ran records the commands whose RunE ran.
func sensitiveTree(ran *[]string) *cobra.Command {
	run := func(cmd *cobra.Command, _ []string) error {
		*ran = append(*ran, cmd.Name())
		return nil
	}
	leaf := func(use string) *cobra.Command {
		c := &cobra.Command{Use: use, RunE: run}
		c.Flags().Bool("force", false, "")
		c.Flags().Bool("dry-run", false, "")
		c.Flags().Bool("strict", true, "")
		return c
	}
	root := &cobra.Command{Use: "qsdev", SilenceUsage: true, SilenceErrors: true}
	group := &cobra.Command{Use: "claude", Aliases: []string{"cc"}}
	group.AddCommand(MarkSensitive(leaf("init"), Sensitivity{Flags: map[string]bool{"force": true}}))
	root.AddCommand(
		MarkSensitive(MarkReadOnly(leaf("teardown"), "dry-run"), Sensitivity{}),
		MarkSensitive(leaf("self-update"), Sensitivity{Flags: map[string]bool{"strict": false}}),
		MarkSensitive(leaf("disable"), Sensitivity{Args: func() []string { return []string{"attach-guard"} }}),
		leaf("status"),
		group,
	)
	InstallHumanGate(root)
	return root
}

// TestHumanGate pins that the root's gate refuses exactly the sensitive
// invocations from an agent session or without a terminal, and lets a human
// at a terminal through.
func TestHumanGate(t *testing.T) {
	tests := []struct {
		args    []string
		agent   bool
		tty     bool
		wantErr string // "" runs the command
	}{
		{args: []string{"teardown", "--force"}, wantErr: "requires a human at an interactive terminal"},
		{args: []string{"teardown", "--force"}, agent: true, tty: true, wantErr: "AI agent session"},
		{args: []string{"teardown", "--force"}, tty: true},
		{args: []string{"teardown", "--dry-run"}},
		{args: []string{"self-update", "--strict=false"}, wantErr: "requires a human"},
		{args: []string{"self-update"}},
		{args: []string{"disable", "attach-guard"}, wantErr: "requires a human"},
		{args: []string{"disable", "context7"}},
		{args: []string{"cc", "init", "--force"}, wantErr: "'qsdev claude init' requires a human"},
		{args: []string{"claude", "init"}},
		{args: []string{"status"}, agent: true},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			marker := ""
			if tt.agent {
				marker = "1"
			}
			t.Setenv("CLAUDECODE", marker)
			var ran []string
			root := sensitiveTree(&ran)
			root.SetIn(ttyReader{Reader: strings.NewReader(""), tty: tt.tty})
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			root.SetArgs(tt.args)
			err := root.Execute()
			if tt.wantErr == "" {
				if err != nil || len(ran) != 1 {
					t.Fatalf("Execute = %v, ran %v; want the command to run", err, ran)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Execute error = %v, want one containing %q", err, tt.wantErr)
			}
			if len(ran) != 0 {
				t.Errorf("refused command ran: %v", ran)
			}
		})
	}
}

// TestSensitiveCommands pins that the specs derive from the marks: one per
// marked command, its path with aliases, and its conditions.
func TestSensitiveCommands(t *testing.T) {
	t.Parallel()
	var ran []string
	specs := SensitiveCommands(sensitiveTree(&ran))
	var paths []string
	for _, s := range specs {
		var words []string
		for _, names := range s.Path {
			words = append(words, strings.Join(names, "|"))
		}
		paths = append(paths, strings.Join(words, " "))
	}
	slices.Sort(paths)
	if want := []string{"claude|cc init", "disable", "self-update", "teardown"}; !slices.Equal(paths, want) {
		t.Fatalf("paths = %q, want %q", paths, want)
	}
	for _, tt := range []struct {
		argv []string
		want bool
	}{
		{[]string{"teardown"}, true},
		{[]string{"teardown", "--dry-run"}, false},
		{[]string{"cc", "init", "--force"}, true},
		{[]string{"claude", "init"}, false},
		{[]string{"self-update", "--strict=false"}, true},
		{[]string{"disable", "attach-guard"}, true},
		{[]string{"disable", "context7"}, false},
	} {
		matched := slices.ContainsFunc(specs, func(s cmdscan.CommandSpec) bool { return s.Matches(tt.argv) })
		if matched != tt.want {
			t.Errorf("specs match %q = %v, want %v", tt.argv, matched, tt.want)
		}
	}
}
