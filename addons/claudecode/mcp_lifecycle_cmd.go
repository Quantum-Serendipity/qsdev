package claudecode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/config"
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
	return filepath.Join(projectRoot, filepath.FromSlash(statePath()))
}

// newLifecycle creates an McpLifecycle for installs and updates: it runs
// commands through runner, persists state to the project's state file, and
// gates releases at the project's compliance tier (projectReleaseAge). It
// fails before any command runs when that tier cannot be determined.
func newLifecycle(projectRoot string, runner mcpregistry.CommandRunner) (*mcpregistry.McpLifecycle, error) {
	minAge, err := projectReleaseAge(projectRoot)
	if err != nil {
		return nil, err
	}
	lc := stateLifecycle(projectRoot, runner)
	lc.MinReleaseAge = minAge
	return lc, nil
}

// projectReleaseAge returns the release-age gate of projectRoot's effective
// security level, the same tier source generation uses. A project without a
// committed config keeps the baseline (0: the package managers' floors); a
// config that exists but cannot be loaded is an error, so an install fails
// closed instead of silently falling back to the baseline.
func projectReleaseAge(projectRoot string) (time.Duration, error) {
	policy, err := config.LoadProjectPolicy(projectRoot)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("loading the project security level: %w", err)
	}
	return catalog.EffectiveAgeGate(config.EffectiveSecurityLevel(policy.Effective.Config)), nil
}

// stateLifecycle creates an McpLifecycle wired to runner and file-backed
// state persistence, with no release-age gate of its own (removal installs
// nothing).
func stateLifecycle(projectRoot string, runner mcpregistry.CommandRunner) *mcpregistry.McpLifecycle {
	stateFile := stateFilePath(projectRoot)
	return &mcpregistry.McpLifecycle{
		CmdRunner: runner,
		StateLoader: func() (*types.GeneratedState, error) {
			s, err := state.LoadStateFromFile(stateFile)
			if err != nil {
				return nil, err
			}
			return &s, nil
		},
		StateSaver: func(s *types.GeneratedState) error {
			return state.SaveProjectState(projectRoot, statePath(), *s)
		},
	}
}

func mcpInstallCmd() *cobra.Command { return newMCPInstallCmd(&execRunner{}) }

// newMCPInstallCmd builds `mcp install`, running package-manager commands
// through runner.
func newMCPInstallCmd(runner mcpregistry.CommandRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <server>",
		Short: "Install an MCP server binary",
		Long: `Install an MCP server using its declared install method (uv tool, npm global,
or nix package). The server must be known to the registry, and uv tool and npm
global installs take the exact release the catalog pins, refusing releases
younger than the project's security-level age gate. Once installed, the
next ` + "`qsdev init --update`" + ` points .mcp.json at the installed binary instead of a
fetch-on-run launcher.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pc, err := cmdutil.Project(cmd)
			if err != nil {
				return err
			}
			lc, err := newLifecycle(pc.Root, runner)
			if err != nil {
				return err
			}
			return runMCPInstall(cmd.Context(), cmd.OutOrStdout(), lc, args[0])
		},
	}

	return cmd
}

func mcpUpdateCmd() *cobra.Command { return newMCPUpdateCmd(&execRunner{}) }

// newMCPUpdateCmd builds `mcp update`, running package-manager commands
// through runner.
func newMCPUpdateCmd(runner mcpregistry.CommandRunner) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "update [server]",
		Short: "Update an MCP server to its pinned version",
		Long: `Move an installed MCP server to the exact release the catalog pins. Use --all
to update all MCP servers recorded in the project state.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pc, err := cmdutil.Project(cmd)
			if err != nil {
				return err
			}
			if !all && len(args) == 0 {
				return fmt.Errorf("specify a server name or use --all")
			}
			lc, err := newLifecycle(pc.Root, runner)
			if err != nil {
				return err
			}
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
			pc, err := cmdutil.Project(cmd)
			if err != nil {
				return err
			}
			return runMCPRemove(cmd.Context(), cmd.OutOrStdout(), stateLifecycle(pc.Root, &execRunner{}), args[0])
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
	_, _ = fmt.Fprintln(w, mcpRegenerateHint)
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
		_, _ = fmt.Fprintln(w, mcpRegenerateHint)
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
	_, _ = fmt.Fprintln(w, mcpRegenerateHint)
	return nil
}

// mcpRegenerateHint follows a change to the installed MCP servers: .mcp.json
// runs a server's installed binary only while the project state records the
// pinned release as installed, and is rewritten on regeneration.
const mcpRegenerateHint = "Run `qsdev init --update` to regenerate .mcp.json for the installed MCP servers."

func mcpHealthCmd() *cobra.Command {
	opts := mcpProbeOptions{title: "MCP Server Health", failUnhealthy: true}

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check health of configured MCP servers",
		Long: `Check the MCP servers configured in .mcp.json and exit non-zero on a failure,
so it can gate CI. By default the check is static and fails when any server
is misconfigured; with --probe it reports health status, tool counts and
response times and fails when any server is not healthy.

` + mcpProbeHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPDiagnostic(cmd, opts)
		},
	}
	return withMCPProbeFlags(cmd, &opts)
}
