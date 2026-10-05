package instance

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/cmd"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// initCommand is the runtime's cobra initializer. Cobra runs initializers
// for the executing command after parsing its flags and before validating its
// arguments and running its hooks, so this is not a root PersistentPreRunE:
// instrumentCommandErrors must see Args failures logged. It finds the executing
// command with root.Find on args (the arguments the root is executed with),
// then:
//
//   - resolves its project once, with the command's root mode
//     (cmdutil.RootMode), and stores it for cmdutil.Project;
//   - for a command a person runs, moves the legacy global logs to the
//     per-user state directory once (migrateLegacyLogs), before the session
//     log can create the new directory;
//   - opens the session log in the tier its runtime profile selects
//     (cmdutil.ProfileOf);
//   - starts the background update check, except for hooks and MCP servers,
//     and only when stderrIsTerminal (see selfupdate.NoticeWanted);
//   - for an interactive command outside any project, notes on stderr the
//     untrusted project markers the resolution skipped, so a planted marker
//     or a CI uid mismatch is not silently ignored.
func (r *Runtime) initCommand(root *cobra.Command, args []string, stderr io.Writer, stderrIsTerminal bool) {
	cmd, _, err := root.Find(args)
	if err != nil {
		return // cobra reports the unknown command itself
	}
	profile := cmdutil.ProfileOf(cmd)
	pc, err := projectctx.ResolveWorkingDir(cmdutil.RootMode(cmd))
	switch {
	case err != nil:
		slog.Warn("resolving the project", "error", err)
	case cmd.HasParent():
		// The bare root acts on no project. Its context is the one every
		// subcommand inherits, so a project stored there would outlive this
		// execution when the tree is executed again (as tests do).
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		cmd.SetContext(projectctx.WithContext(ctx, pc))
	}

	moved, migrateErr := migrateLegacyLogs(profile)
	r.logSession = openSessionLog(cmd, profile, pc)
	switch {
	case migrateErr != nil:
		slog.Debug("leaving the legacy logs in place", "error", migrateErr)
	case moved:
		slog.Info("moved the legacy logs to the state directory", "dir", logging.GlobalLogDir())
	}
	if profile.ChecksForUpdates() {
		r.updateCh = startUpdateCheck(stderrIsTerminal, version.Info().Version)
	}
	if profile == cmdutil.ProfileInteractive && !pc.Found && len(pc.Ignored) > 0 {
		_, _ = fmt.Fprintf(stderr, "%s: ignoring untrusted project marker(s) %s (owned by another user or writable by others); running outside a project\n",
			branding.Get().AppName, strings.Join(pc.Ignored, ", "))
	}
}

// migrateLegacyLogs moves the legacy global logs (~/.<app>/logs) to the
// global log tier in the per-user state directory (projectctx.MigrateLegacyLogs)
// and reports whether it moved them. Only commands a person runs do so: a hook
// or MCP server process never renames user files. With a log-dir override the
// user chose their own log location, so the legacy logs are left alone.
//
// Global and unlogged commands migrate too, not just interactive ones, so the
// move happens on the first command a person runs. A hook or MCP session that
// opened the new directory first does not block it: the legacy entries are
// merged into the existing directory.
func migrateLegacyLogs(profile cmdutil.Profile) (bool, error) {
	if profile.Automated() || os.Getenv(branding.Get().EnvLogDirVar) != "" {
		return false, nil
	}
	dirs, err := projectctx.UserDirs()
	if err != nil {
		return false, fmt.Errorf("locating the user directories: %w", err)
	}
	return projectctx.MigrateLegacyLogs(dirs)
}

// openSessionLog opens the session log of cmd, run with profile in project
// pc: in the project's log tier when profile logs there and a project was
// found, in the global tier otherwise, and in the tier's automated sub-tier
// for an automated profile. It returns nil for an unlogged profile, or when
// logging is off or fails.
func openSessionLog(cmd *cobra.Command, profile cmdutil.Profile, pc projectctx.Context) *logging.Session {
	if !profile.Logged() {
		return nil
	}
	projectScoped := profile.ProjectLogged() && pc.Found
	projectRoot := ""
	if projectScoped {
		projectRoot = pc.Root
	}
	session, err := logging.Init(logging.Config{
		Level:         logging.LevelFromEnv(),
		StderrToo:     strings.EqualFold(os.Getenv(branding.Get().EnvLogVar), "debug"),
		ProjectRoot:   projectRoot,
		ProjectScoped: projectScoped,
		Automated:     profile.Automated(),
	})
	if err != nil {
		slog.Warn("logging init failed", "error", err)
		return nil
	}
	if session != nil {
		session.Command = cmd.CommandPath()
		slog.Info("command starting", "command", session.Command)
	}
	return session
}

// extractDebugFlag scans args for --debug, sets the branded log variable to
// "debug" if found, and returns args with --debug removed. Scanning stops at
// the "--" terminator: everything after it belongs to a child command (e.g.
// `sandbox exec -- tool --debug`) and is passed through verbatim.
func extractDebugFlag(args []string) []string {
	filtered := make([]string, 0, len(args))
	found := false
	for i, arg := range args {
		if arg == "--" {
			filtered = append(filtered, args[i:]...)
			break
		}
		if arg == "--debug" {
			found = true
			continue
		}
		filtered = append(filtered, arg)
	}
	if found {
		_ = os.Setenv(branding.Get().EnvLogVar, "debug")
	}
	return filtered
}

// instrumentCommandErrors wraps every error-returning hook in the command tree
// under root so a failing command logs its error to the session log before
// gdev prints it and exits. It must run before the executing command's hooks
// are invoked (cobra initializers do), and at most once per tree.
func instrumentCommandErrors(root *cobra.Command) {
	if root == nil {
		return
	}
	wrapRun := func(fn func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
		if fn == nil {
			return nil
		}
		return func(c *cobra.Command, args []string) error {
			err := fn(c, args)
			logCommandFailure(c, err)
			return err
		}
	}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.Args != nil {
			validate := c.Args
			c.Args = func(c *cobra.Command, args []string) error {
				err := validate(c, args)
				logCommandFailure(c, err)
				return err
			}
		}
		c.PersistentPreRunE = wrapRun(c.PersistentPreRunE)
		c.PreRunE = wrapRun(c.PreRunE)
		c.RunE = wrapRun(c.RunE)
		c.PostRunE = wrapRun(c.PostRunE)
		c.PersistentPostRunE = wrapRun(c.PersistentPostRunE)
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}

// logCommandFailure records a command's error (and its exit code, when it
// carries one) in the session log.
func logCommandFailure(c *cobra.Command, err error) {
	if err == nil {
		return
	}
	attrs := []any{"command", c.CommandPath(), "error", err.Error()}
	var ece cmd.ExitCodeErr
	if errors.As(err, &ece) {
		attrs = append(attrs, "exit_code", ece.ExitCode())
	}
	slog.Error("command failed", attrs...)
}
