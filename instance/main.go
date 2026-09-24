package instance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	gdevaddons "fastcat.org/go/gdev/addons"
	gdevcmd "fastcat.org/go/gdev/cmd"
	gdevinstance "fastcat.org/go/gdev/instance"
	gdevconfig "fastcat.org/go/gdev/lib/config"

	"github.com/Quantum-Serendipity/qsdev/internal/bugreport"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/logcmd"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/selfupdate"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
)

// Runtime is the process-wide runtime installed by DefaultRuntime: the
// command session's log and the pending self-update notice.
type Runtime struct {
	// logsCmd doubles as a handle on the command tree: a statically
	// registered command is attached to the root directly, so logsCmd.Root()
	// reaches the root once execution starts.
	logsCmd    *cobra.Command
	logSession *logging.Session
	updateCh   <-chan string
	finishOnce sync.Once
}

var (
	defaultRuntime     *Runtime
	defaultRuntimeOnce sync.Once
)

// Main installs the default runtime (see DefaultRuntime) and starts the
// application. Call it last in main, after SetBranding, the addon Configure
// calls and any AddCommands, in place of gdev's cmd.Main:
//
//	instance.SetBranding(cfg)
//	devinit.Configure(...)
//	instance.Main()
//
// It runs the same startup as cmd.Main (addon initialization, customization
// lockdown, config initialization, a context cancelled by SIGINT/SIGTERM) but
// builds the command tree with [NewRootCommand], so a mistyped subcommand at
// any level fails instead of printing help and exiting 0.
//
// Main never returns: it exits with status 0 on success, the error's
// ExitCode() when it implements gdev's cmd.ExitCodeErr, and 1 otherwise. The
// runtime's Finish runs after the command finishes and before the process
// exits, whether or not it failed.
func Main() {
	rt := DefaultRuntime()
	gdevaddons.Initialize()
	// gdev exposes its customization lockdown only through instance.TestMain,
	// which locks customizations, runs the given Run() and exits with its
	// result: exactly the tail of gdev's cmd.Main, with the tree walk added.
	gdevinstance.TestMain(mainRunner(func() int {
		code := run()
		rt.Finish()
		return code
	}))
}

// mainRunner adapts a function to the Run() int interface gdev's
// instance.TestMain executes.
type mainRunner func() int

func (r mainRunner) Run() int { return r() }

// run executes the command tree and returns the process exit code, reporting a
// failure on stderr the way gdev's cmd.Main does.
func run() int {
	if err := gdevconfig.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := NewRootCommand().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return exitCode(err)
	}
	return 0
}

// exitCode maps a command error to a process exit code: the code an error in
// its chain carries via gdev's cmd.ExitCodeErr, or 1.
func exitCode(err error) int {
	var ece gdevcmd.ExitCodeErr
	if errors.As(err, &ece) {
		return ece.ExitCode()
	}
	return 1
}

// NewRootCommand builds the application's root command from every registered
// command (gdev's cmd.Root) and walks the finished tree so that the root and
// every command group reject unknown subcommands. Cobra otherwise treats a
// command without Run/RunE as non-runnable and answers `app chek` or
// `app config nope` with help and a nil error, so a typo in a CI step or agent
// skill exits 0 as if it had succeeded. The walk runs after construction so it
// also covers commands the framework builds itself (the root, `config`).
//
// It must be called after customizations are locked down, as [Main] does.
func NewRootCommand() *cobra.Command {
	root := gdevcmd.Root()
	cmdutil.RejectUnknownSubcommands(root)
	return root
}

// DefaultRuntime installs the runtime every tool built on this framework
// needs and returns it:
//
//   - the universal MCP server's framework adapters and the external-log
//     providers (RegisterFrameworkAdapters)
//   - the release version stamped into VersionPackage (ApplyBuildVersion)
//   - the project's .<app>/defaults.yaml catalog layer (UseProjectDefaults)
//   - the standard commands: self-update, logs and bug-report
//   - the --debug flag, the redacting session log (per-project, global or
//     automated, per the invocation) and error logging for every command
//   - the background self-update check, whose notice Finish prints
//
// Main calls it. Call it directly only to install the runtime without Main
// (as tests building the command tree do); a caller that then runs the tree
// itself must call the returned Runtime's Finish afterwards. It must be called
// after SetBranding and before customizations are locked down; later calls
// return the same Runtime.
func DefaultRuntime() *Runtime {
	defaultRuntimeOnce.Do(func() {
		gdevinstance.CheckCanCustomize()
		defaultRuntime = installDefaultRuntime()
	})
	return defaultRuntime
}

func installDefaultRuntime() *Runtime {
	RegisterFrameworkAdapters()
	ApplyBuildVersion()
	UseProjectDefaults()

	rt := &Runtime{logsCmd: logcmd.Command()}
	AddCommands(standardCommands(rt.logsCmd)...)

	// --debug is consumed before cobra parses (it is not a registered flag)
	// and turns on debug logging for initLogging.
	os.Args = extractDebugFlag(os.Args)

	cobra.OnInitialize(rt.initLogging, func() { instrumentCommandErrors(rt.logsCmd.Root()) })
	// Cobra runs finalizers for the executed command whether or not it
	// failed, so the session is closed (and the update notice shown) even
	// when the tree is run by something that exits on failure, such as gdev's
	// cmd.Main. Finish runs at most once, so Main's own call is harmless.
	cobra.OnFinalize(rt.Finish)

	rt.updateCh = selfupdate.BackgroundCheck(version.Info().Version)
	return rt
}

// standardCommands returns the commands every tool on this framework ships:
// self-update, logs (logsCmd) and bug-report.
func standardCommands(logsCmd *cobra.Command) []*cobra.Command {
	return []*cobra.Command{selfupdate.Command(), logsCmd, bugreport.Command()}
}

// Finish writes the session's closing record and prints any pending update
// notice, exactly once per process.
func (r *Runtime) Finish() {
	r.finishOnce.Do(func() {
		if r.logSession != nil {
			r.logSession.Close()
		}
		selfupdate.PrintNotice(r.updateCh)
	})
}
