package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
)

// TestCommandProfiles walks qsdev's real command tree and pins every
// command's runtime profile: the hooks are automated-hook, the MCP servers
// (`mcp serve` and every legacy `mcp <module>` alias) are mcp-server, the
// commands about the tool itself are global, help, completion, log browsing
// and the bare root are unlogged, and everything else is interactive. A
// listed path covers its subcommands.
func TestCommandProfiles(t *testing.T) {
	root := instance.NewRootCommand()
	want := map[string]cmdutil.Profile{
		"enforce":      cmdutil.ProfileAutomatedHook,
		"selfprotect":  cmdutil.ProfileAutomatedHook,
		"sandbox exec": cmdutil.ProfileAutomatedHook,
		"mcp serve":    cmdutil.ProfileMCPServer,
		"self-update":  cmdutil.ProfileGlobal,
		"version":      cmdutil.ProfileGlobal,
		"report":       cmdutil.ProfileGlobal,
		"logs":         cmdutil.ProfileUnlogged,
		"help":         cmdutil.ProfileUnlogged,
		"completion":   cmdutil.ProfileUnlogged,
	}
	for _, m := range tools.LegacyServerModules() {
		want["mcp "+m] = cmdutil.ProfileMCPServer
	}
	expected := func(c *cobra.Command) cmdutil.Profile {
		if !c.HasParent() {
			return cmdutil.ProfileUnlogged
		}
		path := strings.Fields(c.CommandPath())[1:]
		for n := len(path); n > 0; n-- {
			if p, ok := want[strings.Join(path[:n], " ")]; ok {
				return p
			}
		}
		return cmdutil.ProfileInteractive
	}

	seen := map[string]bool{}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		seen[strings.Join(strings.Fields(c.CommandPath())[1:], " ")] = true
		if got, exp := cmdutil.ProfileOf(c), expected(c); got != exp {
			t.Errorf("ProfileOf(%q) = %q, want %q", c.CommandPath(), got, exp)
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	for path := range want {
		if !seen[path] {
			t.Errorf("command %q is not in the tree", path)
		}
	}
}
