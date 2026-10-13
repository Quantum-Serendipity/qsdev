package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	gdevaddons "fastcat.org/go/gdev/addons"
	gdevinstance "fastcat.org/go/gdev/instance"

	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestMain applies qsdev's real configuration and default runtime and locks
// customizations, so tests can build the command tree exactly as the binary
// does. The startup helper process must observe package initialization alone,
// so it skips this.
func TestMain(m *testing.M) {
	// The guardrail tests' PATH runs the test binary as a stubbed program.
	if name, ok := stubbedProgram(); ok {
		os.Exit(runStubbedProgram(name)) //nolint:forbidigo // test entrypoint
	}
	// The guardrail invariant tests run the test binary as qsdev itself.
	if os.Getenv(cliHelperEnv) == "1" {
		if os.Getenv(humanHelperEnv) == "1" {
			mainAsHuman()
		}
		main()
		return
	}
	if os.Getenv(startupHelperEnv) == "1" {
		os.Exit(m.Run()) //nolint:forbidigo // test entrypoint
	}
	configure()
	// The standard commands (logs among them) come with the default runtime.
	// Skip its background update check; session logging stays off because the
	// test binary's own arguments name no command.
	_ = os.Setenv(branding.Get().EnvNoUpdate, "1")
	instance.DefaultRuntime()
	gdevaddons.Initialize()
	gdevinstance.TestMain(templateCleanupRunner{m})
}

const bogusSubcommand = "qsdev-no-such-subcommand"

// TestCommandTreeRejectsUnknownSubcommands is the W158 regression test. It
// builds qsdev's complete command tree and walks it: the root and every command
// group, including those gdev builds itself (`config`), must reject an unknown
// subcommand with an error instead of printing help and exiting 0.
func TestCommandTreeRejectsUnknownSubcommands(t *testing.T) {
	root := instance.NewRootCommand()
	groups := commandGroups(root)

	for _, want := range []string{"qsdev", "qsdev config", "qsdev devenv", "qsdev claude", "qsdev logs"} {
		if !containsPath(groups, want) {
			t.Fatalf("command group %q not found in the tree; walked %d groups", want, len(groups))
		}
	}

	for _, g := range groups {
		t.Run(g.CommandPath(), func(t *testing.T) {
			// Check the validator first: a group that accepted the word would
			// run a real command below.
			if g.Args == nil || g.Args(g, []string{bogusSubcommand}) == nil {
				t.Fatalf("%q accepts unknown subcommand %q", g.CommandPath(), bogusSubcommand)
			}
			args := append(strings.Fields(g.CommandPath())[1:], bogusSubcommand)
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			root.SetArgs(args)
			err := root.Execute()
			want := `unknown command "` + bogusSubcommand + `" for "` + g.CommandPath() + `"`
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("qsdev %s: error = %v, want containing %q", strings.Join(args, " "), err, want)
			}
		})
	}
}

// TestCommandTreeRejectsTypos checks the reported cases end to end against
// the real tree: each typo fails, and a subcommand typo suggests the intended
// command. (A typo followed by flags the root does not define fails on the
// unknown flag first, which is equally non-zero.)
func TestCommandTreeRejectsTypos(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr []string
	}{
		{"root typo", []string{"chek"}, []string{`unknown command "chek" for "qsdev"`, "check"}},
		{"root typo with flags", []string{"chek", "--audit-level", "high"}, []string{"unknown flag: --audit-level"}},
		{"group typo", []string{"devenv", "add-pakage", "jq"}, []string{`unknown command "add-pakage" for "qsdev devenv"`, "add-package"}},
		{"gdev-built group typo", []string{"config", "nope"}, []string{`unknown command "nope" for "qsdev config"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := instance.NewRootCommand()
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			root.SetArgs(tt.args)
			err := root.Execute()
			if err == nil {
				t.Fatalf("qsdev %s succeeded, want an error", strings.Join(tt.args, " "))
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %v, want containing %q", err, want)
				}
			}
		})
	}
}

// commandGroups returns every command in the tree under root that has
// subcommands, root first.
func commandGroups(root *cobra.Command) []*cobra.Command {
	var groups []*cobra.Command
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.HasSubCommands() {
			groups = append(groups, c)
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	return groups
}

func containsPath(cmds []*cobra.Command, path string) bool {
	for _, c := range cmds {
		if c.CommandPath() == path {
			return true
		}
	}
	return false
}

// TestCommandTree_CatalogGateExemptions pins which commands of the real tree
// the root catalog gate covers, derived from their runtime profiles: no
// command a hook or the MCP server runs is gated (a gate error would make
// Claude Code treat self-protection as a non-blocking failure and fail open),
// and check and the whole defaults group, which report and repair a broken
// defaults file, are catalog-optional. A command a person runs on a project
// is gated.
func TestCommandTree_CatalogGateExemptions(t *testing.T) {
	root := instance.NewRootCommand()
	var automated int
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if cmdutil.ProfileOf(c).Automated() {
			automated++
			if cmdutil.CatalogRequired(c) {
				t.Errorf("%q (profile %q) requires the catalog", c.CommandPath(), cmdutil.ProfileOf(c))
			}
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	if automated == 0 {
		t.Fatal("found no automated command in the tree")
	}

	find := func(t *testing.T, args ...string) *cobra.Command {
		t.Helper()
		c, rest, err := root.Find(args)
		if err != nil || len(rest) != 0 || c == root {
			t.Fatalf("command %q not found: %v", args, err)
		}
		return c
	}
	if check := find(t, "check"); cmdutil.CatalogRequired(check) {
		t.Errorf("%q requires the catalog; it must report the failure itself", check.CommandPath())
	}
	defaults := find(t, "defaults")
	var walkDefaults func(c *cobra.Command)
	walkDefaults = func(c *cobra.Command) {
		if cmdutil.CatalogRequired(c) {
			t.Errorf("%q requires the catalog it exists to repair", c.CommandPath())
		}
		for _, sub := range c.Commands() {
			walkDefaults(sub)
		}
	}
	walkDefaults(defaults)
	if status := find(t, "status"); !cmdutil.CatalogRequired(status) {
		t.Errorf("%q does not require the catalog", status.CommandPath())
	}
}
