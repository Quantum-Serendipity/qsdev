package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	gdevaddons "fastcat.org/go/gdev/addons"
	gdevcmd "fastcat.org/go/gdev/cmd"
	gdevinstance "fastcat.org/go/gdev/instance"
	gdevconfig "fastcat.org/go/gdev/lib/config"

	"github.com/Quantum-Serendipity/qsdev/instance"
)

// humanHelperEnv makes the helper process (cliHelperEnv) run qsdev with its
// stdin presented as an interactive terminal: the test stands in for the
// human that the gate on sensitive commands (cmdutil.InstallHumanGate)
// requires. Only this test binary reads it; the shipped binary has no such
// switch.
const humanHelperEnv = "QSDEV_GUARDRAIL_CLI_HUMAN"

// humanStdin is stdin presented as an interactive terminal.
type humanStdin struct{ io.Reader }

func (humanStdin) IsTerminal() bool { return true }

// helperRunner adapts a function to gdev's instance.TestMain Run() int.
type helperRunner func() int

func (r helperRunner) Run() int { return r() }

// mainAsHuman is main, as instance.Main runs the tree, with stdin presented
// as a terminal. It exits the process.
func mainAsHuman() {
	configure()
	rt := instance.DefaultRuntime()
	gdevaddons.Initialize()
	gdevinstance.TestMain(helperRunner(func() int {
		defer rt.Finish()
		if err := gdevconfig.Initialize(); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		}
		root := instance.NewRootCommand()
		root.SetIn(humanStdin{os.Stdin})
		err := root.ExecuteContext(context.Background())
		if err == nil {
			return 0
		}
		fmt.Fprintln(os.Stderr, err.Error())
		var ece gdevcmd.ExitCodeErr
		if errors.As(err, &ece) {
			return ece.ExitCode()
		}
		return 1
	}))
}
