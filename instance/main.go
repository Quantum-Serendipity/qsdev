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
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/logcmd"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/shim"
	"github.com/Quantum-Serendipity/qsdev/internal/selfupdate"
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
//
// An argv of `sandbox shim …` is a sandbox backend starting a hook: it runs
// the shim before any runtime, addon or config work, none of which the shim
// may do.
func Main() {
	if shim.Invoked(os.Args) {
		os.Exit(shim.Main(os.Args, os.Stderr))
	}
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
// It also installs the human gate (cmdutil.InstallHumanGate), which refuses
// every command marked sensitive unless a human runs it, and the catalog gate
// (installCatalogGate), which fails a command that needs the defaults catalog
// cleanly when the catalog does not load, and gives the commands the
// frameworks build (see markFrameworkProfiles) their runtime profiles.
//
// It must be called after customizations are locked down, as [Main] does.
func NewRootCommand() *cobra.Command {
	root := gdevcmd.Root()
	markFrameworkProfiles(root)
	cmdutil.RejectUnknownSubcommands(root)
	installGates(root, loadCatalog)
	return root
}

// installGates installs the human gate and the catalog gate on root. The
// catalog gate is installed last so it runs first: the human gate's
// sensitivity checks may read the catalog (disable's security-tool list
// does), so the catalog must have loaded, or the command failed cleanly,
// before they run. A catalog load is read-only, so loading before a refusal
// costs nothing.
func installGates(root *cobra.Command, load func() error) {
	cmdutil.InstallHumanGate(root)
	installCatalogGate(root, load)
}

// loadCatalog loads the defaults catalog (built-in, org and project layers).
// catalog.Default caches its result, so later loads by the command reuse it.
func loadCatalog() error {
	_, err := catalog.Default()
	return err
}

// installCatalogGate makes root fail every command that needs the defaults
// catalog (cmdutil.CatalogRequired) with a wrapped error naming the repair
// command when load fails, before the command runs. Once the gate passes the
// catalog is cached as loaded, so the command's own catalog lookups cannot
// fail. Commands that do not need it (hooks, the MCP server, global and
// unlogged commands, catalog-optional ones) run regardless. It chains any
// PersistentPreRunE root already has, as cmdutil.InstallHumanGate does.
func installCatalogGate(root *cobra.Command, load func() error) {
	next := root.PersistentPreRunE
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cmdutil.CatalogRequired(cmd) {
			if err := load(); err != nil {
				return fmt.Errorf("%w (run '%s')", catalog.LoadError(err), catalog.ValidateCommand())
			}
		}
		if next != nil {
			return next(cmd, args)
		}
		return nil
	}
}

// markFrameworkProfiles marks the runtime profile (cmdutil.MarkProfile) of the
// commands cobra and gdev build rather than an addon: cobra's help and
// default completion commands, created here instead of at execution so they
// can carry a mark, are unlogged, and gdev's version is global. A command an
// addon registered under one of those names keeps its own mark. gdev's addons
// command is also marked catalog-optional (cmdutil.MarkCatalogOptional).
func markFrameworkProfiles(root *cobra.Command) {
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	frameworkProfiles := map[string]cmdutil.Profile{
		"help":       cmdutil.ProfileUnlogged,
		"completion": cmdutil.ProfileUnlogged,
		"version":    cmdutil.ProfileGlobal,
	}
	for _, c := range root.Commands() {
		p, ok := frameworkProfiles[c.Name()]
		if _, marked := c.Annotations[cmdutil.ProfileAnnotation]; ok && !marked {
			cmdutil.MarkProfile(c, p)
		}
		// gdev's addons command lists the compiled-in addons and reads no
		// defaults catalog.
		if c.Name() == "addons" {
			cmdutil.MarkCatalogOptional(c)
		}
	}
}

// DefaultRuntime installs the runtime every tool built on this framework
// needs and returns it:
//
//   - the universal MCP server's framework adapters and the external-log
//     providers (RegisterFrameworkAdapters)
//   - the release version stamped into VersionPackage (ApplyBuildVersion)
//   - the standard commands: self-update, logs and bug-report
//   - the --debug flag, and a cobra initializer (Runtime.initCommand) that
//     resolves the executing command's project once, applies that root's
//     .<app>/defaults.yaml catalog layer and org overlay pin, and, per its
//     runtime profile (cmdutil.ProfileOf), opens the redacting session log
//     (per-project, global or automated) and starts the background
//     self-update check, whose notice Finish prints
//   - error logging for every command
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

	rt := &Runtime{logsCmd: logcmd.Command()}
	AddCommands(standardCommands(rt.logsCmd)...)

	// --debug is consumed before cobra parses (it is not a registered flag)
	// and turns on debug logging for the session log.
	os.Args = extractDebugFlag(os.Args)

	cobra.OnInitialize(
		func() {
			rt.initCommand(rt.logsCmd.Root(), os.Args[1:], os.Stderr, selfupdate.NoticeWanted(os.Stderr))
		},
		func() { instrumentCommandErrors(rt.logsCmd.Root()) },
	)
	// Cobra runs finalizers for the executed command whether or not it
	// failed, so the session is closed (and the update notice shown) even
	// when the tree is run by something that exits on failure, such as gdev's
	// cmd.Main. Finish runs at most once, so Main's own call is harmless.
	cobra.OnFinalize(rt.Finish)
	return rt
}

// startUpdateCheck starts the background update check for an interactive
// process and returns its notice channel. A non-interactive process (hook,
// MCP server, CI) gets nil, so it makes no request, writes no cache and Finish
// prints nothing.
func startUpdateCheck(interactive bool, currentVersion string) <-chan string {
	if !interactive {
		return nil
	}
	return selfupdate.BackgroundCheck(currentVersion)
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
