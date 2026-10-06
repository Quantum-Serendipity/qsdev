// Command skipreport fails when a test skipped while naming a tool that a
// required testutil switch governs. CI's nix-sandbox job provisions those
// tools, sets the switches to 1 and runs it over the suite's `go test -json`
// report, so a raw t.Skip("nix-instantiate not available") that bypasses
// testutil.RequireTool cannot silently drop coverage there.
//
// Usage: go run ./internal/testutil/cmd/skipreport test.json
package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("skipreport: ")
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: skipreport <go test -json report>")
	}
	required := testutil.RequiredSwitches()
	if len(required) == 0 {
		return errors.New("no switch is set to 1, so no skip is governed; set the switches the job provisions")
	}
	f, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("opening report: %w", err)
	}
	defer f.Close()

	skips, err := testutil.FindGovernedSkips(f, required)
	if err != nil {
		return fmt.Errorf("reading %s: %w", args[0], err)
	}
	for _, s := range skips {
		fmt.Fprintln(out, s)
	}
	if len(skips) > 0 {
		return fmt.Errorf("%d skipped test(s) name a required tool; use testutil.RequireTool or testutil.Unavailable", len(skips))
	}
	fmt.Fprintf(out, "no skipped test names a tool of %q\n", required)
	return nil
}
