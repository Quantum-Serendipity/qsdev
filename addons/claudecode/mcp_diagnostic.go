package claudecode

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// probeTimeout bounds each live probe of a configured server.
const probeTimeout = 10 * time.Second

// probeUntrustedHint is appended to a not-probed reason that --probe-untrusted
// would lift.
const probeUntrustedHint = "; rerun with --probe-untrusted to probe it anyway"

// The flags that make status and health start or dial servers, and so take
// them out of the read-only contract.
const (
	probeFlag          = "probe"
	probeUntrustedFlag = "probe-untrusted"
)

// probeEligibleNote marks a server --probe would start or dial.
const probeEligibleNote = "would probe with --probe"

// errMCPUnhealthy is returned by `mcp health` when a configured server is not
// healthy (with --probe) or is misconfigured (without), so CI health gates fail.
var errMCPUnhealthy = errors.New("one or more MCP servers are not healthy")

// mcpProbeOptions parameterizes the shared status/health diagnostic.
type mcpProbeOptions struct {
	title          string // report heading, e.g. "MCP Server Status"
	showPrereqs    bool   // print unmet prerequisites per server
	failUnhealthy  bool   // return errMCPUnhealthy when a server fails the check
	jsonOutput     bool
	probe          bool // start or dial the trusted servers
	probeUntrusted bool // also run commands that match no trusted definition; implies probe
}

// mcpProbeHelp is the shared help paragraph of `mcp status` and `mcp health`.
const mcpProbeHelp = `By default nothing is started or dialed: each entry in .mcp.json is checked
statically (its command is on PATH, its URL is https, the environment it needs
is set) and reported with whether --probe would probe it, or why not.

With --probe, servers matching a trusted definition (the built-in or
organization catalog, or the binary's configuration) are started or dialed.
Any other entry comes from the repository and is probed only under
--probe-untrusted, which implies --probe; a remote endpoint then receives its
${VAR} references unexpanded. A package launcher (npx, uvx and the like) is
never started, since running it downloads the package.`

func mcpStatusCmd() *cobra.Command {
	opts := mcpProbeOptions{title: "MCP Server Status", showPrereqs: true}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the status of configured MCP servers",
		Long: `Show the MCP servers configured in .mcp.json; with --probe, also their
health, tool counts and unmet prerequisites.

` + mcpProbeHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPDiagnostic(cmd, opts)
		},
	}
	return withMCPProbeFlags(cmd, &opts)
}

// withMCPProbeFlags registers the flags status and health share and marks cmd
// read-only unless one of the probe flags is set. It returns cmd.
func withMCPProbeFlags(cmd *cobra.Command, opts *mcpProbeOptions) *cobra.Command {
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&opts.probe, probeFlag, false, "Start or dial the servers that match a trusted definition")
	cmd.Flags().BoolVar(&opts.probeUntrusted, probeUntrustedFlag, false, "Also start servers whose command matches no trusted definition (implies --probe)")
	return cmdutil.MarkReadOnly(cmd, "", probeFlag, probeUntrustedFlag)
}

// runMCPDiagnostic loads .mcp.json and reports on it: statically by default,
// or by probing the servers PlanProbes allows under --probe.
func runMCPDiagnostic(cmd *cobra.Command, opts mcpProbeOptions) error {
	pc, err := cmdutil.Project(cmd)
	if err != nil {
		return err
	}
	projectRoot := pc.Root

	servers, err := mcpregistry.ConfiguredServers(projectRoot, mcpregistry.DefaultRegistry())
	if err != nil {
		return err
	}

	if len(servers) == 0 && !opts.jsonOutput {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers configured.")
		return nil
	}

	trusted := mcpregistry.TrustedDefinitions(configuredServerSpecs())
	if opts.probe || opts.probeUntrusted {
		return runMCPProbe(cmd, servers, trusted, opts)
	}
	return runMCPStatic(cmd, servers, trusted, opts)
}

// probedReport is the --probe JSON report: the probe results, marked probed.
type probedReport struct {
	Probed bool `json:"probed"`
	*mcphealth.HealthReport
}

// runMCPProbe probes the servers PlanProbes allows and reports the result.
// Untrusted servers are reported as not probed unless opts.probeUntrusted is
// set.
func runMCPProbe(cmd *cobra.Command, servers []mcphealth.ServerConfig, trusted map[string][]mcpregistry.LaunchSpec, opts mcpProbeOptions) error {
	// Untrusted entries probed under --probe-untrusted keep ExpandEnv unset, so
	// a remote endpoint receives its ${VAR} references literally.
	report := mcpregistry.ProbeAll(cmd.Context(), servers, trusted, mcpregistry.ProbeOptions{
		AllowUntrusted: opts.probeUntrusted,
		Timeout:        probeTimeout,
		OverrideHint:   probeUntrustedHint,
	})

	if opts.jsonOutput {
		if err := writeJSON(cmd.OutOrStdout(), probedReport{Probed: true, HealthReport: report}); err != nil {
			return err
		}
	} else {
		writeProbeReport(cmd.OutOrStdout(), report, opts)
	}

	if opts.failUnhealthy && report.HealthyCount < report.TotalCount {
		return fmt.Errorf("%w: %d/%d healthy", errMCPUnhealthy, report.HealthyCount, report.TotalCount)
	}
	return nil
}

// writeJSON writes v as indented JSON.
func writeJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}
	_, _ = fmt.Fprintln(w, string(data))
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

// staticServer is one server of the static report: its doctor finding and
// whether --probe would start or dial it.
type staticServer struct {
	doctor.MCPServerInfo
	ProbeEligible   bool   `json:"probe_eligible"`
	ProbeSkipReason string `json:"probe_skip_reason,omitempty"`
}

// staticReport is the default report of status and health: configuration
// only, nothing started or dialed.
type staticReport struct {
	Probed             bool           `json:"probed"`
	Servers            []staticServer `json:"servers"`
	MisconfiguredCount int            `json:"misconfigured_count"`
	TotalCount         int            `json:"total_count"`
	Warnings           []string       `json:"warnings,omitempty"`
}

// runMCPStatic reports the configuration of servers without starting or
// dialing any of them.
func runMCPStatic(cmd *cobra.Command, servers []mcphealth.ServerConfig, trusted map[string][]mcpregistry.LaunchSpec, opts mcpProbeOptions) error {
	report := buildStaticReport(servers, mcpregistry.DefaultRegistry(), trusted)

	if opts.jsonOutput {
		if err := writeJSON(cmd.OutOrStdout(), report); err != nil {
			return err
		}
	} else {
		writeStaticReport(cmd.OutOrStdout(), report, opts.title)
	}

	if opts.failUnhealthy && report.MisconfiguredCount > 0 {
		return fmt.Errorf("%w: %d/%d misconfigured", errMCPUnhealthy, report.MisconfiguredCount, report.TotalCount)
	}
	return nil
}

// buildStaticReport composes the doctor's static findings
// (doctor.MCPServerFindings) with the probe policy (mcpregistry.PlanProbes)
// for servers, the configured servers in .mcp.json order. Both evaluate the
// same slice, read once, so row i of the findings is servers[i].
func buildStaticReport(servers []mcphealth.ServerConfig, reg *mcpregistry.McpServerRegistry, trusted map[string][]mcpregistry.LaunchSpec) *staticReport {
	report := &staticReport{Servers: []staticServer{}}
	section := doctor.MCPServerFindings(servers, reg)
	if section == nil {
		return report
	}

	_, skipped := mcpregistry.PlanProbes(servers, trusted, false)
	skips := make(map[string]mcpregistry.Skip, len(skipped))
	for _, s := range skipped {
		skips[s.Name] = s
	}

	report.Warnings = section.Warnings
	for i, srv := range servers {
		info := section.Servers[i]
		row := staticServer{MCPServerInfo: info, ProbeEligible: true}
		if s, ok := skips[srv.Name]; ok {
			row.ProbeEligible = false
			row.ProbeSkipReason = s.Reason
			if s.Overridable {
				row.ProbeSkipReason += probeUntrustedHint
			}
		}
		if info.Status == doctor.MCPStatusMisconfigured {
			report.MisconfiguredCount++
		}
		report.Servers = append(report.Servers, row)
	}
	report.TotalCount = len(report.Servers)
	return report
}

// writeStaticReport prints the human-readable static report.
func writeStaticReport(w io.Writer, report *staticReport, title string) {
	_, _ = fmt.Fprintf(w, "%s (%d servers, configuration only)\n", title, report.TotalCount)
	_, _ = fmt.Fprintln(w, "----------------------------------------")
	for _, s := range report.Servers {
		probe := probeEligibleNote
		if !s.ProbeEligible {
			probe = mcphealth.StatusNotProbed
		}
		_, _ = fmt.Fprintf(w, "  %-20s  %-14s  %-5s  %s\n", s.DisplayName(), s.Status, s.Transport, probe)
		for _, issue := range s.Issues {
			_, _ = fmt.Fprintf(w, "    %s: %s\n", issue.Severity, issue.Message)
			if issue.Remediation != "" {
				_, _ = fmt.Fprintf(w, "      fix: %s\n", issue.Remediation)
			}
		}
		if s.ProbeSkipReason != "" {
			_, _ = fmt.Fprintf(w, "    %s: %s\n", mcphealth.StatusNotProbed, s.ProbeSkipReason)
		}
	}
	for _, warning := range report.Warnings {
		_, _ = fmt.Fprintf(w, "  warning: %s\n", warning)
	}
	_, _ = fmt.Fprintf(w, "\n%d/%d misconfigured; nothing was started or dialed (rerun with --probe to check liveness)\n",
		report.MisconfiguredCount, report.TotalCount)
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
			pc, err := cmdutil.Project(cmd)
			if err != nil {
				return err
			}
			projectRoot := pc.Root

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

	return cmdutil.MarkReadOnly(cmd, "")
}
