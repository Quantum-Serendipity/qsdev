package main

import (
	"errors"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"fastcat.org/go/gdev/addons/bootstrap"
	"fastcat.org/go/gdev/cmd"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/internal/bugreport"
	"github.com/Quantum-Serendipity/qsdev/internal/logcmd"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
	"github.com/Quantum-Serendipity/qsdev/internal/selfupdate"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

var (
	logSession *logging.Session
	updateCh   <-chan string
	finishOnce sync.Once
)

func main() {
	logsCmd := configure()

	// Pre-parse --debug from args before cobra processes them.
	// Sets QSDEV_LOG=debug so the OnInitialize callback picks it up.
	// The flag is consumed here and removed from os.Args so cobra
	// doesn't reject it as unknown.
	os.Args = extractDebugFlag(os.Args)

	cobra.OnInitialize(initLogging, func() { instrumentCommandErrors(logsCmd.Root()) })

	updateCh = selfupdate.BackgroundCheck(version.Info().Version)
	// instance.Main exits the process; finishSession runs just before, on
	// success and failure alike.
	instance.Main(finishSession)
}

// configure applies qsdev's branding, runtime wiring and addon configuration
// and registers its own commands: every customization that must happen before
// the command tree is built. It returns the `logs` command, which doubles as a
// handle on the command tree: a statically registered command is attached to
// the root directly, so logsCmd.Root() reaches the root once execution starts.
func configure() *cobra.Command {
	instance.SetBranding(branding.Default())
	// Framework adapters, external-log providers and the release version are
	// wired through the instance package so downstream tools (including
	// scaffold-instance output) get exactly the same runtime as qsdev.
	instance.RegisterFrameworkAdapters()
	instance.ApplyBuildVersion()

	bootstrap.Configure(
		bootstrap.WithSteps(
			devenv.InstallDevenvStep(),
			devenv.InstallDirenvStep(),
			claudecode.InstallClaudeStep(),
		),
	)

	devenv.Configure(
		devenv.WithDefaultLanguages("go"),
		devenv.WithDirenv(true),
	)
	claudecode.Configure(
		claudecode.WithDefaultPermissions(claudecode.PermissionPresetStandard),
	)
	devinit.Configure(
		devinit.WithDetectProjectType(true),
		devinit.WithPlanPreview(true),
	)

	logsCmd := logcmd.Command()
	instance.AddCommands(selfupdate.Command(), logsCmd, bugreport.Command())
	return logsCmd
}

// finishSession writes the session's closing record and prints any pending
// update notice, exactly once per process.
func finishSession() {
	finishOnce.Do(func() {
		if logSession != nil {
			logSession.Close()
		}
		selfupdate.PrintNotice(updateCh)
	})
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

// extractDebugFlag scans args for --debug, sets QSDEV_LOG=debug if found,
// and returns args with --debug removed. Scanning stops at the "--"
// terminator: everything after it belongs to a child command (e.g.
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

// initLogging is called by cobra.OnInitialize during Execute(), before any
// command's PreRun. It sets up the two-tier structured logging system.
func initLogging() {
	class := classifyInvocation(os.Args[1:])
	if class == logging.ClassUnlogged {
		return
	}

	level := logging.LevelFromEnv()
	stderrToo := strings.EqualFold(os.Getenv(branding.Get().EnvLogVar), "debug")

	projectRoot := logging.DetectProjectRoot()
	commandPath := detectCommandFromArgs()

	var err error
	logSession, err = logging.Init(logging.Config{
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
	if logSession != nil {
		logSession.Command = commandPath
		slog.Info("command starting", "command", commandPath)
	}
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

// detectCommandFromArgs builds a rough command path from os.Args for the
// opening log record. This runs before cobra parses, so it uses simple
// heuristics: take args until one starts with "-".
func detectCommandFromArgs() string {
	parts := []string{branding.Get().AppName}
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			break
		}
		parts = append(parts, arg)
	}
	return strings.Join(parts, " ")
}
