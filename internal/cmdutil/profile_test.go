package cmdutil

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestProfileOf pins how a command's runtime profile is derived: its own mark,
// else the nearest marked ancestor's, else interactive; the bare root and
// cobra's completion request command are unlogged without a mark.
func TestProfileOf(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "qsdev"}
	hook := MarkProfile(&cobra.Command{Use: "enforce"}, ProfileAutomatedHook)
	hookSub := &cobra.Command{Use: "check", Run: func(*cobra.Command, []string) {}}
	hook.AddCommand(hookSub)
	group := &cobra.Command{Use: "sandbox"}
	exec := MarkProfile(&cobra.Command{Use: "exec", Run: func(*cobra.Command, []string) {}}, ProfileAutomatedHook)
	status := &cobra.Command{Use: "status", Run: func(*cobra.Command, []string) {}}
	group.AddCommand(exec, status)
	logs := MarkProfile(&cobra.Command{Use: "logs"}, ProfileUnlogged)
	logsShow := &cobra.Command{Use: "show", Run: func(*cobra.Command, []string) {}}
	logs.AddCommand(logsShow)
	// A child's own mark overrides its ancestor's.
	logsGlobal := MarkProfile(&cobra.Command{Use: "export", Run: func(*cobra.Command, []string) {}}, ProfileGlobal)
	logs.AddCommand(logsGlobal)
	complete := &cobra.Command{Use: cobra.ShellCompRequestCmd + " [command-line]", Aliases: []string{cobra.ShellCompNoDescRequestCmd}}
	root.AddCommand(hook, group, logs, complete)

	tests := []struct {
		cmd  *cobra.Command
		want Profile
	}{
		{root, ProfileUnlogged},
		{hook, ProfileAutomatedHook},
		{hookSub, ProfileAutomatedHook},
		{group, ProfileInteractive},
		{exec, ProfileAutomatedHook},
		{status, ProfileInteractive},
		{logs, ProfileUnlogged},
		{logsShow, ProfileUnlogged},
		{logsGlobal, ProfileGlobal},
		{complete, ProfileUnlogged},
		{&cobra.Command{Use: "orphan"}, ProfileUnlogged},
	}
	for _, tt := range tests {
		t.Run(tt.cmd.CommandPath(), func(t *testing.T) {
			t.Parallel()
			if got := ProfileOf(tt.cmd); got != tt.want {
				t.Errorf("ProfileOf(%q) = %q, want %q", tt.cmd.CommandPath(), got, tt.want)
			}
		})
	}
}

// TestProfile_Predicates pins which profiles open a session log, use the
// automated tier and start the background update check.
func TestProfile_Predicates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		p                                  Profile
		logged, automated, update, project bool
	}{
		{ProfileInteractive, true, false, true, true},
		{ProfileGlobal, true, false, true, false},
		{ProfileUnlogged, false, false, true, false},
		{ProfileAutomatedHook, true, true, false, true},
		// The MCP server opens its own automated session.
		{ProfileMCPServer, false, true, false, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.p), func(t *testing.T) {
			t.Parallel()
			if got := tt.p.Logged(); got != tt.logged {
				t.Errorf("Logged() = %v, want %v", got, tt.logged)
			}
			if got := tt.p.Automated(); got != tt.automated {
				t.Errorf("Automated() = %v, want %v", got, tt.automated)
			}
			if got := tt.p.ChecksForUpdates(); got != tt.update {
				t.Errorf("ChecksForUpdates() = %v, want %v", got, tt.update)
			}
			if got := tt.p.ProjectLogged(); got != tt.project {
				t.Errorf("ProjectLogged() = %v, want %v", got, tt.project)
			}
		})
	}
}
