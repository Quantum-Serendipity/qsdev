package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
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

// stateFilePath returns the path to the MCP state file within a project.
func stateFilePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".claude", ".qsdev-claude-state.yaml")
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

			lc := newLifecycle(projectRoot)
			ctx := cmd.Context()

			result, err := lc.Install(ctx, args[0])
			if err != nil {
				return err
			}

			if result.Installed {
				fmt.Fprintf(cmd.OutOrStdout(), "Installed %s via %s (version: %s)\n",
					result.ServerName, result.Method, result.Version)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Could not install %s: %s\n",
					result.ServerName, result.Error)
			}

			return nil
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

			lc := newLifecycle(projectRoot)
			ctx := cmd.Context()

			if all {
				results, err := lc.UpdateAll(ctx)
				if err != nil {
					return err
				}
				for _, r := range results {
					if r.Updated {
						fmt.Fprintf(cmd.OutOrStdout(), "Updated %s: %s -> %s\n",
							r.ServerName, r.PreviousVer, r.NewVersion)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "Could not update %s: %s\n",
							r.ServerName, r.Error)
					}
				}
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("specify a server name or use --all")
			}

			result, err := lc.Update(ctx, args[0])
			if err != nil {
				return err
			}

			if result.Updated {
				fmt.Fprintf(cmd.OutOrStdout(), "Updated %s: %s -> %s\n",
					result.ServerName, result.PreviousVer, result.NewVersion)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Could not update %s: %s\n",
					result.ServerName, result.Error)
			}

			return nil
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

			lc := newLifecycle(projectRoot)
			ctx := cmd.Context()

			result, err := lc.Remove(ctx, args[0])
			if err != nil {
				return err
			}

			if result.Removed {
				fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", result.ServerName)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Could not remove %s: %s\n",
					result.ServerName, result.Error)
			}

			return nil
		},
	}

	return cmd
}

func mcpHealthCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check health of configured MCP servers",
		Long: `Probe all configured MCP servers via their stdio transport and report
health status, tool counts, and response times.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			servers, err := loadMCPServers(projectRoot)
			if err != nil {
				return err
			}

			if len(servers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers configured.")
				return nil
			}

			report := mcphealth.CheckAll(servers, 10*time.Second)

			if jsonOutput {
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling health report: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "MCP Server Health (%d servers)\n", report.TotalCount)
			fmt.Fprintln(cmd.OutOrStdout(), "----------------------------------------")
			for _, s := range report.Servers {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-20s  %-14s  tools: %d  %dms\n",
					s.Name, s.Status, s.ToolCount, s.ResponseMs)
				if s.Error != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "    error: %s\n", s.Error)
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n%d/%d healthy\n", report.HealthyCount, report.TotalCount)

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}
