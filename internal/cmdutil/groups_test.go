package cmdutil

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTree builds root -> group(devenv) -> {leaf(add-package), group(nested) -> leaf(run)}
// mirroring qsdev's group commands, which have subcommands but no Run, plus a
// runnable group root -> logs -> leaf(list) like `qsdev logs`, which lists
// logs when called bare.
func newTree(ran *[]string) *cobra.Command {
	leaf := func(name string) *cobra.Command {
		return &cobra.Command{
			Use:  name,
			Args: cobra.ArbitraryArgs,
			RunE: func(c *cobra.Command, args []string) error {
				*ran = append(*ran, c.CommandPath()+" "+strings.Join(args, " "))
				return nil
			},
		}
	}
	root := &cobra.Command{Use: "qsdev", SilenceErrors: true, SilenceUsage: true}
	group := &cobra.Command{Use: "devenv", Short: "Manage devenv"}
	nested := &cobra.Command{Use: "nested"}
	nested.AddCommand(leaf("run"))
	group.AddCommand(leaf("add-package"), nested)
	logs := leaf("logs")
	logs.Args = nil // cobra's legacy validation accepts any args on a subcommand
	logs.AddCommand(leaf("list"))
	RejectUnknownSubcommands(group, logs)
	root.AddCommand(group, logs)
	return root
}

// TestRejectUnknownSubcommands is the W158 regression test: an unknown
// subcommand of a command group must fail instead of printing help and
// exiting 0.
func TestRejectUnknownSubcommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		wantErr  string // "" means success
		wantRan  string // leaf invocation expected, "" for none
		wantHelp bool
	}{
		{"typo in subcommand fails", []string{"devenv", "add-pakage", "jq"}, `unknown command "add-pakage" for "qsdev devenv"`, "", false},
		{"typo suggests the real command", []string{"devenv", "add-pakage"}, "add-package", "", false},
		{"unknown nested subcommand fails", []string{"devenv", "nested", "nope"}, `unknown command "nope" for "qsdev devenv nested"`, "", false},
		{"bare group shows help", []string{"devenv"}, "", "", true},
		{"group help word shows help", []string{"devenv", "help"}, "", "", true},
		{"group help for a subcommand shows its help", []string{"devenv", "help", "nested"}, "", "", true},
		{"group help for an unknown topic fails", []string{"devenv", "help", "nope"}, `unknown command "nope" for "qsdev devenv"`, "", false},
		{"known subcommand still runs", []string{"devenv", "add-package", "jq"}, "", "qsdev devenv add-package jq", false},
		{"nested leaf still runs", []string{"devenv", "nested", "run"}, "", "qsdev devenv nested run ", false},
		{"runnable group rejects unknown subcommand", []string{"logs", "nope"}, `unknown command "nope" for "qsdev logs"`, "", false},
		{"runnable group still runs bare", []string{"logs"}, "", "qsdev logs ", false},
		{"runnable group subcommand still runs", []string{"logs", "list"}, "", "qsdev logs list ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var ran []string
			root := newTree(&ran)
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			root.SetArgs(tt.args)

			err := root.Execute()
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
			if tt.wantRan != "" && (len(ran) != 1 || ran[0] != tt.wantRan) {
				t.Errorf("ran = %q, want [%q]", ran, tt.wantRan)
			}
			if tt.wantRan == "" && len(ran) != 0 {
				t.Errorf("ran = %q, want nothing", ran)
			}
			if got := strings.Contains(out.String(), "Usage:"); got != tt.wantHelp {
				t.Errorf("help printed = %v, want %v\n%s", got, tt.wantHelp, out.String())
			}
		})
	}
}
