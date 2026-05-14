package outdated

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// lookPathFunc is overridable for testing.
var lookPathFunc = exec.LookPath

// RunOutdated checks for outdated dependencies across detected ecosystems.
// Output is streamed to w with ecosystem headers.
func RunOutdated(ctx context.Context, w io.Writer, projectRoot string, ecosystems []string, opts OutdatedOptions) (*OutdatedResult, error) {
	if len(ecosystems) == 0 {
		fmt.Fprintln(w, "No ecosystems detected in this project.")
		return &OutdatedResult{}, nil
	}

	result := &OutdatedResult{}

	for _, eco := range ecosystems {
		if opts.Ecosystem != "" && opts.Ecosystem != eco {
			continue
		}

		commands := CommandsForEcosystem(eco)
		if len(commands) == 0 {
			continue
		}

		// Find the first command whose binary is available
		var selectedCmd *EcosystemCommand
		for i := range commands {
			if _, err := lookPathFunc(commands[i].Binary); err == nil {
				selectedCmd = &commands[i]
				break
			}
		}

		if selectedCmd == nil {
			binaryNames := make([]string, 0, len(commands))
			for _, c := range commands {
				binaryNames = append(binaryNames, c.Binary)
			}
			check := EcosystemCheck{
				Name:       eco,
				Skipped:    true,
				SkipReason: fmt.Sprintf("%s not found on PATH", strings.Join(binaryNames, "/")),
			}
			result.Ecosystems = append(result.Ecosystems, check)
			fmt.Fprintf(w, "=== %s === (skipped: %s)\n\n", eco, check.SkipReason)
			continue
		}

		fmt.Fprintf(w, "=== %s ===\n", eco)

		// Run the command with a timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		cmd := exec.CommandContext(timeoutCtx, selectedCmd.Binary, selectedCmd.Args...)
		cmd.Dir = projectRoot
		cmd.Stdout = w
		cmd.Stderr = w

		cmdStr := selectedCmd.Binary + " " + strings.Join(selectedCmd.Args, " ")

		check := EcosystemCheck{
			Name:    eco,
			Command: cmdStr,
		}

		err := cmd.Run()
		cancel()

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				check.ExitCode = exitErr.ExitCode()
				if selectedCmd.OutdatedOnExit1 && check.ExitCode == 1 {
					check.HasOutdated = true
				} else if check.ExitCode != 0 {
					check.Error = err
				}
			} else {
				check.Error = err
			}
		}

		// For commands that exit 0 even with outdated packages,
		// we can't easily determine outdated status from the exit code alone.
		// The user sees the output directly, which is sufficient for a thin wrapper.

		result.Ecosystems = append(result.Ecosystems, check)
		fmt.Fprintln(w)
	}

	return result, nil
}
