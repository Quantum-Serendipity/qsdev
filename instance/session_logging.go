package instance

import (
	"errors"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/cmd"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// initLogging is called by cobra.OnInitialize during Execute(), before any
// command's PreRun. It sets up the two-tier structured logging system.
func (r *Runtime) initLogging() {
	class := classifyInvocation(os.Args[1:])
	if class == logging.ClassUnlogged {
		return
	}

	level := logging.LevelFromEnv()
	stderrToo := strings.EqualFold(os.Getenv(branding.Get().EnvLogVar), "debug")

	projectRoot := logging.DetectProjectRoot()
	commandPath := detectCommandFromArgs(os.Args[1:])

	session, err := logging.Init(logging.Config{
		Level:         level,
		StderrToo:     stderrToo,
		ProjectRoot:   projectRoot,
		ProjectScoped: class != logging.ClassGlobal && projectRoot != "",
		Automated:     class == logging.ClassAutomated,
	})
	if err != nil {
		slog.Warn("logging init failed", "error", err)
		return
	}
	r.logSession = session
	if session != nil {
		session.Command = commandPath
		slog.Info("command starting", "command", commandPath)
	}
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

// classifyInvocation extends logging.ClassifyInvocation with the legacy
// `mcp <module>` aliases of `mcp serve --module <module>`, which, like
// `mcp serve`, open their own automated logging session.
func classifyInvocation(args []string) logging.CommandClass {
	class := logging.ClassifyInvocation(args)
	if class == logging.ClassProject && len(args) >= 2 && args[0] == "mcp" &&
		slices.Contains(tools.LegacyServerModules(), args[1]) {
		return logging.ClassUnlogged
	}
	return class
}

// detectCommandFromArgs builds a rough command path from the command-line
// arguments (without the program name) for the opening log record. It runs
// before cobra parses, so it uses a simple heuristic: take args until one
// starts with "-".
func detectCommandFromArgs(args []string) string {
	parts := []string{branding.Get().AppName}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}
