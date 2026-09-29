package cmdutil

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTree builds root -> group(devenv) -> {leaf(add-package), group(nested) -> leaf(run)}
// mirroring qsdev's group commands, which have subcommands but no Run, plus a
// runnable group root -> logs -> leaf(list) like `qsdev logs`, which lists
// logs when called bare. The root mirrors gdev's: Args: cobra.NoArgs and no
// Run, so cobra never consults its validator. The devenv group has a
// persistent pre-run hook that records itself, to prove a typo is rejected
// before any hook runs. As in qsdev, the groups are wrapped at registration
// (as the addons do) and the whole tree is then walked again (as
// instance.Main does), which also proves applying it twice is harmless.
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
	root := &cobra.Command{Use: "qsdev", Args: cobra.NoArgs, SilenceErrors: true, SilenceUsage: true}
	group := &cobra.Command{
		Use:   "devenv",
		Short: "Manage devenv",
		PersistentPreRunE: func(*cobra.Command, []string) error {
			*ran = append(*ran, "pre-run")
			return nil
		},
	}
	nested := &cobra.Command{Use: "nested"}
	nested.AddCommand(leaf("run"))
	group.AddCommand(leaf("add-package"), nested)
	logs := leaf("logs")
	logs.Args = nil // cobra's legacy validation accepts any args on a subcommand
	logs.AddCommand(leaf("list"))
	check := leaf("check")
	check.Args = cobra.NoArgs
	root.AddCommand(RejectUnknownSubcommands(group, logs)...)
	root.AddCommand(check)
	RejectUnknownSubcommands(root)
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
		wantErr  string   // "" means success
		wantRan  []string // invocations expected, nil for none
		wantHelp bool
	}{
		{"typo in subcommand fails", []string{"devenv", "add-pakage", "jq"}, `unknown command "add-pakage" for "qsdev devenv"`, nil, false},
		{"typo suggests the real command", []string{"devenv", "add-pakage"}, "add-package", nil, false},
		{"unknown nested subcommand fails", []string{"devenv", "nested", "nope"}, `unknown command "nope" for "qsdev devenv nested"`, nil, false},
		{"bare group shows help", []string{"devenv"}, "", []string{"pre-run"}, true},
		{"group help word shows help", []string{"devenv", "help"}, "", []string{"pre-run"}, true},
		{"group help for a subcommand shows its help", []string{"devenv", "help", "nested"}, "", []string{"pre-run"}, true},
		{"group help for an unknown topic fails", []string{"devenv", "help", "nope"}, `unknown command "nope" for "qsdev devenv"`, nil, false},
		{"known subcommand still runs", []string{"devenv", "add-package", "jq"}, "", []string{"pre-run", "qsdev devenv add-package jq"}, false},
		{"nested leaf still runs", []string{"devenv", "nested", "run"}, "", []string{"pre-run", "qsdev devenv nested run "}, false},
		{"runnable group rejects unknown subcommand", []string{"logs", "nope"}, `unknown command "nope" for "qsdev logs"`, nil, false},
		{"runnable group still runs bare", []string{"logs"}, "", []string{"qsdev logs "}, false},
		{"runnable group subcommand still runs", []string{"logs", "list"}, "", []string{"qsdev logs list "}, false},
		{"typo at the root fails", []string{"chek"}, `unknown command "chek" for "qsdev"`, nil, false},
		{"typo at the root suggests the real command", []string{"chek"}, "check", nil, false},
		{"typo at the root with flags fails", []string{"chek", "--", "x"}, `unknown command "chek" for "qsdev"`, nil, false},
		{"bare root shows help", []string{}, "", nil, true},
		{"root help command still works", []string{"help", "devenv"}, "", nil, true},
		{"root leaf still runs", []string{"check"}, "", []string{"qsdev check "}, false},
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
			if !slices.Equal(ran, tt.wantRan) {
				t.Errorf("ran = %q, want %q", ran, tt.wantRan)
			}
			if got := strings.Contains(out.String(), "Usage:"); got != tt.wantHelp {
				t.Errorf("help printed = %v, want %v\n%s", got, tt.wantHelp, out.String())
			}
		})
	}
}
