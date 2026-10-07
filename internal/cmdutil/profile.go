package cmdutil

import "github.com/spf13/cobra"

// ProfileAnnotation records a command's runtime Profile (see MarkProfile).
const ProfileAnnotation = "qsdev.profile"

// Profile says who runs a command and therefore what the process runtime does
// around it: where its session log goes, whether it checks for updates and
// whether it may print notes for a human.
type Profile string

const (
	// ProfileInteractive is the default: a command a person runs. It logs to
	// the project tier inside a project and to the global tier outside one.
	ProfileInteractive Profile = "interactive"
	// ProfileGlobal is a command about the tool itself rather than a project
	// (self-update, version, report); it always logs to the global tier.
	ProfileGlobal Profile = "global"
	// ProfileUnlogged leaves nothing worth diagnosing (help, shell completion
	// on every TAB press, log browsing) and would only flood the retention
	// cap, so it opens no session log.
	ProfileUnlogged Profile = "unlogged"
	// ProfileAutomatedHook is a command an agent's hooks run on every tool
	// call (enforce, selfprotect, sandbox exec). It logs to the automated
	// sub-tier, kept under its own cap, and never checks for updates.
	ProfileAutomatedHook Profile = "automated-hook"
	// ProfileMCPServer is an MCP server the agent launches. The server opens
	// its own automated session for the root it serves, so the process
	// runtime opens none, and it never checks for updates.
	ProfileMCPServer Profile = "mcp-server"
)

// MarkProfile declares the runtime profile of cmd and of every subcommand
// without a mark of its own. It returns cmd so it can wrap a constructor.
func MarkProfile(cmd *cobra.Command, p Profile) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[ProfileAnnotation] = string(p)
	return cmd
}

// ProfileOf returns cmd's runtime profile: its own mark, else the nearest
// marked ancestor's, else ProfileInteractive. The bare root (which only
// prints usage or the version) and cobra's shell-completion request command
// (created by cobra at execution, so it cannot be marked) are unlogged.
func ProfileOf(cmd *cobra.Command) Profile {
	if !cmd.HasParent() || cmd.Name() == cobra.ShellCompRequestCmd {
		return ProfileUnlogged
	}
	for c := cmd; c.HasParent(); c = c.Parent() {
		if p, ok := c.Annotations[ProfileAnnotation]; ok {
			return Profile(p)
		}
	}
	return ProfileInteractive
}

// Logged reports whether the process runtime opens a session log for p.
func (p Profile) Logged() bool {
	return p == ProfileInteractive || p == ProfileGlobal || p == ProfileAutomatedHook
}

// ProjectLogged reports whether p's session log goes to the project's log
// tier when a project was found (the global tier otherwise).
func (p Profile) ProjectLogged() bool {
	return p == ProfileInteractive || p == ProfileAutomatedHook
}

// Automated reports whether p is run by tooling rather than a person, so its
// logs go to the automated sub-tier.
func (p Profile) Automated() bool {
	return p == ProfileAutomatedHook || p == ProfileMCPServer
}

// ChecksForUpdates reports whether p may start the background update check:
// never for a process an agent launches.
func (p Profile) ChecksForUpdates() bool { return !p.Automated() }
