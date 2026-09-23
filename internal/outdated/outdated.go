package outdated

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// lookPathFunc is overridable for testing.
var lookPathFunc = exec.LookPath

// commandTimeout bounds each ecosystem's outdated command.
const commandTimeout = 60 * time.Second

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

		selectedCmd, skipReason := selectCommand(projectRoot, commands, opts.PackageManagers[eco])
		if selectedCmd != nil && selectedCmd.Unsupported != nil {
			skipReason = selectedCmd.Unsupported(projectRoot)
		}
		if skipReason != "" {
			check := EcosystemCheck{Name: eco, Skipped: true, SkipReason: skipReason}
			result.Ecosystems = append(result.Ecosystems, check)
			fmt.Fprintf(w, "=== %s === (skipped: %s)\n\n", eco, check.SkipReason)
			continue
		}

		fmt.Fprintf(w, "=== %s ===\n", eco)
		result.Ecosystems = append(result.Ecosystems, runCommand(ctx, w, projectRoot, eco, selectedCmd))
		fmt.Fprintln(w)
	}

	return result, nil
}

// selectCommand picks the ecosystem command to run. The project's configured
// package manager wins; otherwise a command whose marker file (lockfile or
// build file) is present in projectRoot; otherwise the catalog's fallback order.
// Among the chosen candidates the first whose binary is on PATH runs, so a pnpm
// project is never checked with `npm outdated` merely because npm comes first
// on PATH. When no candidate binary is available it returns a skip reason.
func selectCommand(projectRoot string, commands []EcosystemCommand, configured string) (*EcosystemCommand, string) {
	candidates := filterCommands(commands, func(c EcosystemCommand) bool {
		return configured != "" && c.Binary == configured
	})
	if len(candidates) == 0 {
		candidates = filterCommands(commands, func(c EcosystemCommand) bool {
			return hasMarker(projectRoot, c.Markers)
		})
	}
	if len(candidates) == 0 {
		candidates = commands
	}

	for i := range candidates {
		if _, err := lookPathFunc(candidates[i].Binary); err == nil {
			return &candidates[i], ""
		}
	}
	binaryNames := make([]string, 0, len(candidates))
	for _, c := range candidates {
		binaryNames = append(binaryNames, c.Binary)
	}
	return nil, fmt.Sprintf("%s not found on PATH", strings.Join(binaryNames, "/"))
}

func filterCommands(commands []EcosystemCommand, keep func(EcosystemCommand) bool) []EcosystemCommand {
	var out []EcosystemCommand
	for _, c := range commands {
		if keep(c) {
			out = append(out, c)
		}
	}
	return out
}

func hasMarker(projectRoot string, markers []string) bool {
	return slices.ContainsFunc(markers, func(m string) bool {
		_, err := os.Stat(filepath.Join(projectRoot, m))
		return err == nil
	})
}

// runCommand runs one ecosystem's outdated command with a timeout, streaming
// its output to w, and classifies the exit: the tool's "outdated found" exit
// sets HasOutdated; any other failure (other non-zero exit, timeout kill, or a
// command that could not start) is recorded in Error.
func runCommand(ctx context.Context, w io.Writer, projectRoot, eco string, selected *EcosystemCommand) EcosystemCheck {
	timeoutCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(timeoutCtx, selected.Binary, selected.Args...)
	cmd.Dir = projectRoot
	cmd.Stdout = w
	cmd.Stderr = w

	check := EcosystemCheck{
		Name:    eco,
		Command: selected.Binary + " " + strings.Join(selected.Args, " "),
	}

	err := cmd.Run()
	if err == nil {
		// For commands that exit 0 even with outdated packages, the exit code
		// alone cannot tell; the user sees the tool's output directly.
		return check
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		check.ExitCode = exitErr.ExitCode()
		if selected.OutdatedOnExit1 && check.ExitCode == 1 {
			check.HasOutdated = true
			return check
		}
	}
	check.Error = fmt.Errorf("running %s: %w", check.Command, err)
	return check
}
