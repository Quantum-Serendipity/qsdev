package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// statusNotProbed marks a server that was not started or dialed because it does
// not match a trusted definition.
const statusNotProbed = "not-probed"

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

	servers, err := loadMCPServers(projectRoot)
	if err != nil {
		return err
	}

	if len(servers) == 0 && !opts.jsonOutput {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No MCP servers configured.")
		return nil
	}

	probe, skipped := mcpregistry.PartitionTrusted(servers, trustedMCPDefinitions())
	if opts.probeUntrusted {
		// Untrusted entries run as configured but keep ExpandEnv unset, so a
		// remote endpoint receives its ${VAR} references literally.
		for name := range skipped {
			probe[name] = servers[name]
		}
		skipped = nil
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	report := mcphealth.CheckAll(ctx, probe)
	addNotProbed(report, skipped)

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

// addNotProbed appends a not-probed entry for each skipped server, giving the
// reason it was skipped, and keeps the report sorted by name.
func addNotProbed(report *mcphealth.HealthReport, skipped map[string]string) {
	for name, reason := range skipped {
		report.Servers = append(report.Servers, mcphealth.ServerHealth{
			Name:   name,
			Status: statusNotProbed,
			Error:  reason + "; rerun with --probe-untrusted to probe it anyway",
		})
	}
	report.TotalCount = len(report.Servers)
	slices.SortFunc(report.Servers, func(a, b mcphealth.ServerHealth) int {
		return strings.Compare(a.Name, b.Name)
	})
}

// trustedMCPDefinitions returns the server definitions qsdev itself vouches
// for, keyed by server name: the embedded catalog plus the user's organization
// overlay (including the variants generation derives from it), and servers
// configured into the binary. The project catalog overlay is deliberately
// excluded — like .mcp.json, it is repository content.
func trustedMCPDefinitions() map[string][]mcpregistry.LaunchSpec {
	trusted := make(map[string][]mcpregistry.LaunchSpec)
	var opts []catalog.LoadOption
	if org := catalog.OrgConfigFile(); org != "" {
		opts = append(opts, catalog.WithOrgConfigFile(org))
	}
	if cat, err := catalog.Load(opts...); err == nil {
		for name, def := range cat.MCPServers() {
			for _, e := range catalogServerVariants(def) {
				trusted[name] = append(trusted[name], entryLaunchSpec(e))
				if name == sembleServerName {
					// Generation writes this variant when text-file indexing is on.
					trusted[name] = append(trusted[name], configLaunchSpec(sembleTextFilesServer(e)))
				}
			}
		}
	}
	for _, srv := range addon.Config.MCPServers {
		trusted[srv.Name] = append(trusted[srv.Name], configLaunchSpec(srv))
	}
	return trusted
}

// entryLaunchSpec is the trusted definition a generated .mcp.json entry makes.
func entryLaunchSpec(e MCPServerEntry) mcpregistry.LaunchSpec {
	return mcpregistry.LaunchSpec{Command: e.Command, Args: e.Args, Env: e.Env, URL: e.URL, Headers: e.Headers}
}

// configLaunchSpec is the trusted definition of a configured stdio server.
func configLaunchSpec(c MCPServerConfig) mcpregistry.LaunchSpec {
	return mcpregistry.LaunchSpec{Command: c.Command, Args: c.Args, Env: c.Env}
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

			servers, err := loadMCPServers(projectRoot)
			if err != nil {
				return err
			}

			if jsonOutput {
				if servers == nil {
					servers = map[string]mcphealth.ServerConfig{}
				}
				data, err := json.MarshalIndent(servers, "", "  ")
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
			for _, name := range slices.Sorted(maps.Keys(servers)) {
				cfg := servers[name]
				if cfg.URL != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-20s  http %s\n", name, cfg.URL)
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-20s  %s %v\n", name, cfg.Command, cfg.Args)
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

func loadMCPServers(projectRoot string) (map[string]mcphealth.ServerConfig, error) {
	mcpPath := filepath.Join(projectRoot, ".mcp.json")
	data, err := os.ReadFile(mcpPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading .mcp.json: %w", err)
	}

	var mcp McpJSON
	if err := json.Unmarshal(data, &mcp); err != nil {
		return nil, fmt.Errorf("parsing .mcp.json: %w", err)
	}

	cat, catErr := catalog.Default()

	servers := make(map[string]mcphealth.ServerConfig, len(mcp.MCPServers))
	for name, entry := range mcp.MCPServers {
		cfg := mcphealth.ServerConfig{
			Name:    name,
			Command: entry.Command,
			Args:    entry.Args,
			URL:     entry.URL,
			Env:     entry.Env,
			Headers: entry.Headers,
		}
		if catErr == nil {
			if def, ok := cat.MCPServer(name); ok {
				cfg.RequiredEnv = def.RequiredEnv
			}
		}
		servers[name] = cfg
	}

	return servers, nil
}
