package claudecode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// execRunner implements mcpregistry.CommandRunner using os/exec.
type execRunner struct{}

func (e *execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// stateFilePath returns the path to the MCP state file within a project. MCP
// lifecycle records share the claude addon's (branded) state file, whose
// writers preserve them across regenerations.
func stateFilePath(projectRoot string) string {
	return filepath.Join(projectRoot, statePath())
}

// newLifecycle creates an McpLifecycle wired to the real command runner and
// file-backed state persistence.
func newLifecycle(projectRoot string) *mcpregistry.McpLifecycle {
	statePath := stateFilePath(projectRoot)
	return &mcpregistry.McpLifecycle{
		CmdRunner: &execRunner{},
		StateLoader: func() (*types.GeneratedState, error) {
			s, err := state.LoadStateFromFile(statePath)
			if err != nil {
				return nil, err
			}
			return &s, nil
		},
		StateSaver: func(s *types.GeneratedState) error {
			return state.SaveStateToFile(statePath, *s)
		},
	}
}

func mcpInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <server>",
		Short: "Install an MCP server binary",
		Long: `Install an MCP server using its declared install method (uv tool, npm global,
or nix package). The server must be known to the registry.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			return runMCPInstall(cmd.Context(), cmd.OutOrStdout(), newLifecycle(projectRoot), args[0])
		},
	}

	return cmd
}

func mcpUpdateCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "update [server]",
		Short: "Update an MCP server to latest version",
		Long: `Update an installed MCP server to the latest available version. Use --all
to update all MCP servers recorded in the project state.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			if !all && len(args) == 0 {
				return fmt.Errorf("specify a server name or use --all")
			}
			lc := newLifecycle(projectRoot)
			if all {
				return runMCPUpdateAll(cmd.Context(), cmd.OutOrStdout(), lc)
			}
			return runMCPUpdate(cmd.Context(), cmd.OutOrStdout(), lc, args[0])
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Update all installed MCP servers")

	return cmd
}

func mcpRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <server>",
		Short: "Remove an installed MCP server",
		Long:  `Remove an MCP server binary and clean up its state entry.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			return runMCPRemove(cmd.Context(), cmd.OutOrStdout(), newLifecycle(projectRoot), args[0])
		},
	}

	return cmd
}

// errMCPOperationFailed marks an install, update or remove whose package-manager
// operation did not succeed, so the command exits non-zero.
var errMCPOperationFailed = errors.New("MCP server operation failed")

// runMCPInstall installs one server, returning errMCPOperationFailed when the
// install did not succeed.
func runMCPInstall(ctx context.Context, w io.Writer, lc *mcpregistry.McpLifecycle, name string) error {
	result, err := lc.Install(ctx, name)
	if err != nil {
		return err
	}
	if !result.Installed {
		_, _ = fmt.Fprintf(w, "Could not install %s: %s\n", result.ServerName, result.Error)
		return fmt.Errorf("%w: install %s: %s", errMCPOperationFailed, result.ServerName, result.Error)
	}
	_, _ = fmt.Fprintf(w, "Installed %s via %s (version: %s)\n", result.ServerName, result.Method, result.Version)
	return nil
}

// runMCPUpdate updates one server, returning errMCPOperationFailed when the
// update did not succeed.
func runMCPUpdate(ctx context.Context, w io.Writer, lc *mcpregistry.McpLifecycle, name string) error {
	result, err := lc.Update(ctx, name)
	if err != nil {
		return err
	}
	if !reportUpdate(w, result) {
		return fmt.Errorf("%w: update %s: %s", errMCPOperationFailed, result.ServerName, result.Error)
	}
	return nil
}

// runMCPUpdateAll updates every recorded server, printing each outcome, and
// returns errMCPOperationFailed counting the failures.
func runMCPUpdateAll(ctx context.Context, w io.Writer, lc *mcpregistry.McpLifecycle) error {
	results, err := lc.UpdateAll(ctx)
	if err != nil {
		return err
	}
	var failed []string
	for _, r := range results {
		if !reportUpdate(w, r) {
			failed = append(failed, r.ServerName)
		}
	}
	if len(failed) > 0 {
		slices.Sort(failed)
		return fmt.Errorf("%w: %d of %d updates failed (%s)", errMCPOperationFailed, len(failed), len(results), strings.Join(failed, ", "))
	}
	return nil
}

// reportUpdate prints an update outcome and reports whether it succeeded.
func reportUpdate(w io.Writer, r *mcpregistry.UpdateResult) bool {
	if r.Updated {
		_, _ = fmt.Fprintf(w, "Updated %s: %s -> %s\n", r.ServerName, r.PreviousVer, r.NewVersion)
		return true
	}
	_, _ = fmt.Fprintf(w, "Could not update %s: %s\n", r.ServerName, r.Error)
	return false
}

// runMCPRemove removes one server, returning errMCPOperationFailed when the
// removal did not succeed.
func runMCPRemove(ctx context.Context, w io.Writer, lc *mcpregistry.McpLifecycle, name string) error {
	result, err := lc.Remove(ctx, name)
	if err != nil {
		return err
	}
	if !result.Removed {
		_, _ = fmt.Fprintf(w, "Could not remove %s: %s\n", result.ServerName, result.Error)
		return fmt.Errorf("%w: remove %s: %s", errMCPOperationFailed, result.ServerName, result.Error)
	}
	_, _ = fmt.Fprintf(w, "Removed %s\n", result.ServerName)
	return nil
}

func mcpHealthCmd() *cobra.Command {
	opts := mcpProbeOptions{title: "MCP Server Health", failUnhealthy: true}

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check health of configured MCP servers",
		Long: `Probe all configured MCP servers via their transport and report health
status, tool counts, and response times. Exits non-zero when any server is not
healthy, so it can gate CI.

Only servers whose command matches a trusted definition are started; pass
--probe-untrusted to also run repository-supplied commands.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPProbe(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&opts.probeUntrusted, "probe-untrusted", false, "Also start servers whose command matches no trusted definition")

	return cmd
}
