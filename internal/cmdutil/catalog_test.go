package cmdutil

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestCatalogRequired pins which commands need the defaults catalog to load
// before they run: an interactive command, unless it or an ancestor is marked
// catalog-optional. Every other profile acts on no project's catalog (global,
// unlogged) or keeps its own fail-closed contract (hooks, MCP server).
func TestCatalogRequired(t *testing.T) {
	t.Parallel()
	run := func(*cobra.Command, []string) {}
	root := &cobra.Command{Use: "qsdev"}
	status := &cobra.Command{Use: "status", Run: run}
	group := &cobra.Command{Use: "devenv"}
	doctor := &cobra.Command{Use: "doctor", Run: run}
	group.AddCommand(doctor)
	defaults := MarkCatalogOptional(&cobra.Command{Use: "defaults", Run: run})
	validate := &cobra.Command{Use: "validate", Run: run}
	defaults.AddCommand(validate)
	check := MarkCatalogOptional(&cobra.Command{Use: "check", Run: run})
	hook := MarkProfile(&cobra.Command{Use: "selfprotect", Run: run}, ProfileAutomatedHook)
	sandbox := &cobra.Command{Use: "sandbox"}
	exec := MarkProfile(&cobra.Command{Use: "exec", Run: run}, ProfileAutomatedHook)
	sandbox.AddCommand(exec)
	mcp := MarkProfile(&cobra.Command{Use: "mcp", Run: run}, ProfileMCPServer)
	version := MarkProfile(&cobra.Command{Use: "version", Run: run}, ProfileGlobal)
	logs := MarkProfile(&cobra.Command{Use: "logs"}, ProfileUnlogged)
	logsShow := &cobra.Command{Use: "show", Run: run}
	logs.AddCommand(logsShow)
	complete := &cobra.Command{Use: cobra.ShellCompRequestCmd, Run: run}
	groupWithHelp := &cobra.Command{Use: "claude"}
	hooksList := &cobra.Command{Use: "list", Run: run}
	groupWithHelp.AddCommand(hooksList)
	RejectUnknownSubcommands(groupWithHelp)
	runnableGroup := &cobra.Command{Use: "session", Run: run}
	runnableGroup.AddCommand(&cobra.Command{Use: "list", Run: run})
	RejectUnknownSubcommands(runnableGroup)
	root.AddCommand(status, group, defaults, check, hook, sandbox, mcp, version, logs, complete, groupWithHelp, runnableGroup)

	tests := []struct {
		cmd  *cobra.Command
		want bool
	}{
		{status, true},
		{group, false},         // not runnable: cobra only shows its help
		{groupWithHelp, false}, // made runnable only to show help
		{hooksList, true},      // a subcommand of a help-only group still needs it
		{runnableGroup, true},  // a group that does work of its own
		{doctor, true},
		{sandbox, false}, // a bare group
		{exec, false},
		{defaults, false},
		{validate, false}, // inherited from the defaults group
		{check, false},
		{hook, false},
		{mcp, false},
		{version, false},
		{logs, false},
		{logsShow, false},
		{complete, false},
		{root, false},
	}
	for _, tt := range tests {
		t.Run(tt.cmd.CommandPath(), func(t *testing.T) {
			t.Parallel()
			if got := CatalogRequired(tt.cmd); got != tt.want {
				t.Errorf("CatalogRequired(%q) = %v, want %v (profile %q)",
					tt.cmd.CommandPath(), got, tt.want, ProfileOf(tt.cmd))
			}
		})
	}
}

// TestCatalogRequired_FalseForEveryNonInteractiveProfile: no profile but
// interactive requires the catalog, marked or not.
func TestCatalogRequired_FalseForEveryNonInteractiveProfile(t *testing.T) {
	t.Parallel()
	for _, p := range []Profile{ProfileGlobal, ProfileUnlogged, ProfileAutomatedHook, ProfileMCPServer} {
		root := &cobra.Command{Use: "qsdev"}
		cmd := MarkProfile(&cobra.Command{Use: "c", Run: func(*cobra.Command, []string) {}}, p)
		root.AddCommand(cmd)
		if CatalogRequired(cmd) {
			t.Errorf("CatalogRequired with profile %q = true, want false", p)
		}
	}
}
