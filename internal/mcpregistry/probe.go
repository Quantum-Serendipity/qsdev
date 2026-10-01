package mcpregistry

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Skip is a server the probe gate declined to start or dial.
type Skip struct {
	Name   string
	Reason string
	// Overridable is set when probing untrusted entries would lift the skip:
	// the only objection is that the entry matches no trusted definition.
	Overridable bool
}

// ProbeOptions parameterizes ProbeAll.
type ProbeOptions struct {
	// AllowUntrusted also probes entries matching no trusted definition, with
	// their remote ${VAR} references left unexpanded. Only the CLI's explicit
	// --probe-untrusted sets it; MCP tools never do.
	AllowUntrusted bool
	// Timeout bounds each probe, counted from when it starts; it must be positive.
	Timeout time.Duration
	// OverrideHint is appended to the reason of each Overridable skip, e.g. the
	// flag that sets AllowUntrusted.
	OverrideHint string
}

// PlanProbes is the single policy for live MCP health probes. It splits
// servers into those safe to probe and those skipped with a reason:
//
//   - a package launcher, or qsdev's own MCP server, is never started;
//   - a remote URL must be https, or plain http to a loopback host;
//   - an entry must match a trusted definition of the same name, unless
//     allowUntrusted is set.
//
// The first two rules always apply, to the entry as the probe would start or
// dial it (after ${VAR} expansion), so a skip they cause is not Overridable.
// Probed entries carry ExpandEnv only when they match a trusted definition, so
// an untrusted remote endpoint never receives host environment values. The
// input slice is not modified.
func PlanProbes(servers []mcphealth.ServerConfig, trusted map[string][]LaunchSpec, allowUntrusted bool) (probe []mcphealth.ServerConfig, skipped []Skip) {
	for _, cfg := range servers {
		cfg.ExpandEnv = MatchesTrusted(cfg, trusted[cfg.Name])
		if reason := unsafeProbeReason(cfg); reason != "" {
			skipped = append(skipped, Skip{Name: cfg.Name, Reason: reason})
			continue
		}
		if !cfg.ExpandEnv && !allowUntrusted {
			skipped = append(skipped, Skip{Name: cfg.Name, Reason: untrustedReason(cfg), Overridable: true})
			continue
		}
		probe = append(probe, cfg)
	}
	return probe, skipped
}

// ProbeAll probes the servers PlanProbes allows, at most a few at a time with
// opts.Timeout each, and reports every server, sorted by name: the skipped
// ones as not-probed with their reason.
func ProbeAll(ctx context.Context, servers []mcphealth.ServerConfig, trusted map[string][]LaunchSpec, opts ProbeOptions) *mcphealth.HealthReport {
	probe, skipped := PlanProbes(servers, trusted, opts.AllowUntrusted)
	report := mcphealth.CheckAll(ctx, probe, opts.Timeout)
	for _, s := range skipped {
		reason := s.Reason
		if s.Overridable {
			reason += opts.OverrideHint
		}
		report.Servers = append(report.Servers, mcphealth.ServerHealth{
			Name: s.Name, Status: mcphealth.StatusNotProbed, Error: reason,
		})
	}
	report.TotalCount = len(report.Servers)
	slices.SortFunc(report.Servers, func(a, b mcphealth.ServerHealth) int {
		return strings.Compare(a.Name, b.Name)
	})
	return report
}

// unsafeProbeReason applies the rules no override lifts to cfg as the probe
// would start or dial it, returning why it must not be probed, or "".
func unsafeProbeReason(cfg mcphealth.ServerConfig) string {
	target := mcphealth.ProbeTarget(cfg)
	if reason := probeSkipReason(target); reason != "" {
		return reason
	}
	if target.URL == "" {
		return ""
	}
	if err := mcphealth.ProbeableURL(target.URL); err != nil {
		return err.Error()
	}
	return ""
}

// untrustedReason explains why an entry matching no trusted definition is not
// probed, and how to vouch for it lastingly.
func untrustedReason(cfg mcphealth.ServerConfig) string {
	reason := "untrusted remote endpoint matches no trusted definition (its env references would be sent to it)"
	if cfg.URL == "" {
		cmdline := strings.Join(append([]string{cfg.Command}, cfg.Args...), " ")
		reason = fmt.Sprintf("command %q matches no trusted definition", cmdline)
	}
	return reason + "; " + orgOverlayHint(cfg.Name)
}

// orgOverlayHint tells the user how to trust a server: define it under
// mcp_servers in the organization catalog overlay, which (unlike .mcp.json and
// the project overlay) is not repository content.
func orgOverlayHint(name string) string {
	where := "the organization catalog overlay"
	if p := catalog.OrgConfigPath(); p != "" {
		where += " (" + p + ")"
	}
	return fmt.Sprintf("define %s under mcp_servers in %s to trust it", name, where)
}

// probeSkipReason reports why a live health probe must not start cfg, or ""
// when starting it is safe. Health probes run on behalf of diagnostics (often
// triggered by an agent tool call), so they must never download and execute a
// package: a package launcher fetches whatever version is currently published,
// outside the package guard. The launcher is detected whether it is the command
// itself or is started through a wrapper (`cmd /c npx ...`, `sh -c "uvx ..."`,
// `env npx ...`). A probe must also not spawn another copy of the qsdev MCP
// server that may be answering the very call doing the probing.
func probeSkipReason(cfg mcphealth.ServerConfig) string {
	switch {
	case cfg.URL != "":
		return ""
	case cfg.Command == "":
		return "no command configured"
	}
	if launcher := networkLauncherIn(cfg.Command, cfg.Args); launcher != "" {
		return fmt.Sprintf("package launcher %s would download and run the package; %s", launcher, pinnedBinaryHint(cfg.Name))
	}
	if isSelfServer(cfg) {
		return "this qsdev MCP server"
	}
	return ""
}

// networkLauncherIn returns the name of the package launcher the invocation
// runs (including subcommand and env-wrapped launchers), or of the first
// always-fetching launcher found in any whitespace-separated word of the
// command or its arguments (a shell or cmd /c wrapper), or "".
func networkLauncherIn(command string, args []string) string {
	if launcher := fetchingLauncher(command, args); launcher != "" {
		return launcher
	}
	words := append([]string{command}, args...)
	for _, w := range words {
		for _, field := range strings.Fields(w) {
			if LaunchesFromNetwork(field) {
				return commandName(field)
			}
		}
	}
	return ""
}

// pinnedBinaryHint tells the user how to replace a package launcher with a
// pinned, locally installed binary: `qsdev mcp install <name>` when the catalog
// can install name's pinned release, otherwise to configure a binary they
// installed themselves (an install the catalog cannot do would just fail).
func pinnedBinaryHint(name string) string {
	if def, ok := DefaultRegistry().ByName(name); ok && installable(def) {
		return fmt.Sprintf("run `%s mcp install %s` to install a pinned binary", branding.Get().AppName, name)
	}
	return "install a pinned release locally and configure its binary as the command"
}

// isSelfServer reports whether cfg launches qsdev's own universal MCP server
// (`qsdev mcp serve ...`). A server restricted with --module to a tool module
// (as the catalog's agent-postmortem and version-sentinel servers are) is a
// separate, narrow process rather than another copy of the server that may be
// answering the probing call, so it is probed like any other local server.
func isSelfServer(cfg mcphealth.ServerConfig) bool {
	if commandName(cfg.Command) != branding.Get().AppName ||
		len(cfg.Args) < 2 || cfg.Args[0] != "mcp" || cfg.Args[1] != "serve" {
		return false
	}
	for _, arg := range cfg.Args[2:] {
		if arg == "--module" || strings.HasPrefix(arg, "--module=") {
			return false
		}
	}
	return true
}
