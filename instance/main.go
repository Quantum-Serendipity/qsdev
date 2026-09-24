package instance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	gdevaddons "fastcat.org/go/gdev/addons"
	gdevcmd "fastcat.org/go/gdev/cmd"
	gdevinstance "fastcat.org/go/gdev/instance"
	gdevconfig "fastcat.org/go/gdev/lib/config"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
)

// Main starts the application. Call it last in main(), in place of gdev's
// cmd.Main: it runs the same startup (addon initialization, customization
// lockdown, config initialization, a context cancelled by SIGINT/SIGTERM) but
// builds the command tree with [NewRootCommand], so a mistyped subcommand at
// any level fails instead of printing help and exiting 0.
//
// Main never returns: it exits with status 0 on success, the error's
// ExitCode() when it implements gdev's cmd.ExitCodeErr, and 1 otherwise. The
// atExit functions run in order after the command finishes and before the
// process exits, whether or not it failed.
func Main(atExit ...func()) {
	gdevaddons.Initialize()
	// gdev exposes its customization lockdown only through instance.TestMain,
	// which locks customizations, runs the given Run() and exits with its
	// result: exactly the tail of gdev's cmd.Main, with the tree walk added.
	gdevinstance.TestMain(mainRunner(func() int {
		code := run()
		for _, fn := range atExit {
			fn()
		}
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
