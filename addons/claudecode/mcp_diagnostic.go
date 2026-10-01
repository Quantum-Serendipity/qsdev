package claudecode

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// probeTimeout bounds each live probe of a configured server.
const probeTimeout = 10 * time.Second

// probeUntrustedHint is appended to a not-probed reason that --probe-untrusted
// would lift.
const probeUntrustedHint = "; rerun with --probe-untrusted to probe it anyway"

// errMCPUnhealthy is returned by `mcp health` when any configured server is not
// healthy, so CI health gates fail.
var errMCPUnhealthy = errors.New("one or more MCP servers are not healthy")

// mcpProbeOptions parameterizes the shared status/health probe.
type mcpProbeOptions struct {
	title          string // report heading, e.g. "MCP Server Status"
	showPrereqs    bool   // print unmet prerequisites per server
	failUnhealthy  bool   // return errMCPUnhealthy when not every server is healthy
	jsonOutput     bool
	probeUntrusted bool // run commands that match no trusted definition
}

func mcpStatusCmd() *cobra.Command {
	opts := mcpProbeOptions{title: "MCP Server Status", showPrereqs: true}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show health status of configured MCP servers",
		Long: `Probe the MCP servers configured in .mcp.json and show their health,
tool counts and unmet prerequisites.

Only servers matching a trusted definition (the built-in or organization
catalog, or the binary's configuration) are started or dialed. Any other entry
comes from the repository and is not probed unless --probe-untrusted is given,
and then a remote endpoint receives its ${VAR} references unexpanded; use
'mcp list' to inspect the configuration without running anything.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPProbe(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&opts.probeUntrusted, "probe-untrusted", false, "Also start servers whose command matches no trusted definition")

	return cmd
}

// runMCPProbe loads .mcp.json, probes the trusted servers and reports the
// result. Untrusted servers are reported as not probed unless
// opts.probeUntrusted is set.
func runMCPProbe(cmd *cobra.Command, opts mcpProbeOptions) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	servers, err := mcpregistry.ConfiguredServers(projectRoot, mcpregistry.DefaultRegistry())
	if err != nil {
		return err
	}

	if len(servers) == 0 && !opts.jsonOutput {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers configured.")
		return nil
	}

	// Untrusted entries probed under --probe-untrusted keep ExpandEnv unset, so
	// a remote endpoint receives its ${VAR} references literally.
	report := mcpregistry.ProbeAll(cmd.Context(), servers, mcpregistry.TrustedDefinitions(configuredServerSpecs()), mcpregistry.ProbeOptions{
		AllowUntrusted: opts.probeUntrusted,
		Timeout:        probeTimeout,
		OverrideHint:   probeUntrustedHint,
	})

	if opts.jsonOutput {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling report: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	} else {
		writeProbeReport(cmd.OutOrStdout(), report, opts)
	}

	if opts.failUnhealthy && report.HealthyCount < report.TotalCount {
		return fmt.Errorf("%w: %d/%d healthy", errMCPUnhealthy, report.HealthyCount, report.TotalCount)
	}
	return nil
}

// writeProbeReport prints the human-readable probe report.
func writeProbeReport(w io.Writer, report *mcphealth.HealthReport, opts mcpProbeOptions) {
	_, _ = fmt.Fprintf(w, "%s (%d servers)\n", opts.title, report.TotalCount)
	_, _ = fmt.Fprintln(w, "----------------------------------------")
	for _, s := range report.Servers {
		_, _ = fmt.Fprintf(w, "  %-20s  %-14s  tools: %d  %dms\n",
			s.Name, s.Status, s.ToolCount, s.ResponseMs)
		if s.Error != "" {
			_, _ = fmt.Fprintf(w, "    error: %s\n", s.Error)
		}
		if !opts.showPrereqs {
			continue
		}
		for _, p := range s.Prerequisites {
			if !p.Met {
				_, _ = fmt.Fprintf(w, "    prerequisite: %s (%s) — %s\n", p.Name, p.Type, p.Detail)
			}
		}
	}
	_, _ = fmt.Fprintf(w, "\n%d/%d healthy\n", report.HealthyCount, report.TotalCount)
}

// configuredServerSpecs returns the launch definitions of the servers
// configured into this binary, the extra trusted set passed to
// mcpregistry.TrustedDefinitions.
func configuredServerSpecs() map[string][]mcpregistry.LaunchSpec {
	specs := make(map[string][]mcpregistry.LaunchSpec, len(addon.Config.MCPServers))
	for _, srv := range addon.Config.MCPServers {
		specs[srv.Name] = append(specs[srv.Name], mcpregistry.LaunchSpec{Command: srv.Command, Args: srv.Args, Env: srv.Env})
	}
	return specs
}

func mcpListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured MCP servers without health-checking",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			servers, err := mcpregistry.ConfiguredServers(projectRoot, mcpregistry.DefaultRegistry())
			if err != nil {
				return err
			}

			if jsonOutput {
				// Keyed by name, as this output has always been.
				byName := make(map[string]mcphealth.ServerConfig, len(servers))
				for _, cfg := range servers {
					byName[cfg.Name] = cfg
				}
				data, err := json.MarshalIndent(byName, "", "  ")
				if err != nil {
					return fmt.Errorf("marshaling servers: %w", err)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			if len(servers) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers configured.")
				return nil
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Configured MCP Servers (%d)\n", len(servers))
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "----------------------------------------")
			for _, cfg := range servers {
				if cfg.URL != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-20s  http %s\n", cfg.Name, cfg.URL)
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-20s  %s %v\n", cfg.Name, cfg.Command, cfg.Args)
				}
				if len(cfg.RequiredEnv) > 0 {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    required env: %v\n", cfg.RequiredEnv)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}
