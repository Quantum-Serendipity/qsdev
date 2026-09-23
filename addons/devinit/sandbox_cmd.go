package devinit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/exitcode"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/backendselect"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/bwrap"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/policy"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// hookBlockExitCode is the exit status Claude Code treats as "block this tool
// call". `sandbox exec` wraps guard hooks, so a failure to set up the sandbox
// must surface as this code: any other non-zero status is a non-blocking hook
// error and would let the guarded call through (fail open).
const hookBlockExitCode = 2

// capabilityProbe reports the host's sandbox capabilities. The command
// constructors take it as a parameter so tests can force a specific backend.
type capabilityProbe func(context.Context) *sandbox.SystemCapabilities

func sandboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sandbox",
		Short: "Manage hook execution sandboxing",
		Long: `Tools for managing the hook execution sandbox.

Use "sandbox exec" to run a command inside the sandbox, and
"sandbox status" to display sandbox capabilities and tier.`,
	}
	cmd.AddCommand(
		newSandboxExecCmd(sandbox.ProbeCapabilitiesDefault),
		newSandboxStatusCmd(sandbox.ProbeCapabilitiesDefault),
	)
	return cmd
}

func newSandboxExecCmd(probe capabilityProbe) *cobra.Command {
	var category string
	var policyPath string
	var hookName string

	cmd := &cobra.Command{
		Use:   "exec [flags] -- COMMAND [ARGS...]",
		Short: "Execute a command inside the hook sandbox",
		Long: `Runs COMMAND inside a sandboxed environment with isolation
appropriate for the specified hook category. The sandbox tier is
automatically selected based on available kernel capabilities.

Standard input is forwarded to COMMAND, and the project directory
($CLAUDE_PROJECT_DIR, or the current directory) is mounted inside the
sandbox. A failure to set up the sandbox exits with status 2 so that a
wrapped Claude Code hook fails closed.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("no command specified; use -- COMMAND [ARGS...]")
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			projectDir, err := sandboxProjectDir()
			if err != nil {
				return sandboxSetupFailure(err)
			}

			// The default policy path is project-relative; resolve it against the
			// project so a hook invoked from a subdirectory still finds it.
			if !cmd.Flags().Changed("policy") && !filepath.IsAbs(policyPath) {
				policyPath = filepath.Join(projectDir, policyPath)
			}

			// CompilePolicy returns the defaults when no policy file exists. A file
			// that exists but cannot be compiled must not silently fall back to the
			// defaults, which would discard the user's stricter rules.
			spec, err := policy.CompilePolicy(ctx, policyPath)
			if err != nil {
				return sandboxSetupFailure(fmt.Errorf("compiling sandbox policy: %w", err))
			}

			if hookName == "" {
				hookName = defaultHookName(args[0])
			}

			cfg := policy.ToSandboxConfig(spec, sandbox.ParseHookCategory(category), hookName)
			cfg.ProjectDir = projectDir
			cfg.HookCommand = args
			cfg.Stdin = cmd.InOrStdin()

			result, err := runSandboxed(ctx, cfg, probe(ctx), cmd.ErrOrStderr())
			if err != nil {
				return sandboxSetupFailure(err)
			}

			if len(result.Stdout) > 0 {
				_, _ = cmd.OutOrStdout().Write(result.Stdout)
			}
			if len(result.Stderr) > 0 {
				_, _ = cmd.ErrOrStderr().Write(result.Stderr)
			}

			if result.ExitCode != 0 {
				return exitcode.New(result.ExitCode, "sandboxed command exited with code %d", result.ExitCode)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&category, "category", "linter",
		"Hook category (linter, formatter, network-linter, generator, test-runner)")
	cmd.Flags().StringVar(&policyPath, "policy", ".qsdev/policy.nix",
		"Path to sandbox policy file (the default is relative to the project directory)")
	cmd.Flags().StringVar(&hookName, "hook-name", "",
		"Name used to look up the policy's hookOverrides (default: the command's base name without extension)")

	return cmd
}

// sandboxSetupFailure converts a failure to prepare or start the sandbox into
// the blocking hook exit code, so a wrapped guard hook fails closed.
func sandboxSetupFailure(err error) error {
	return exitcode.New(hookBlockExitCode, "%s sandbox: %v", branding.Get().AppName, err)
}

// sandboxProjectDir returns the project directory to expose inside the
// sandbox. Claude Code exports CLAUDE_PROJECT_DIR to every hook; outside a hook
// the current directory is used.
func sandboxProjectDir() (string, error) {
	dir := os.Getenv("CLAUDE_PROJECT_DIR")
	if dir == "" {
		return cmdutil.ProjectRoot()
	}
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("CLAUDE_PROJECT_DIR must be an absolute path, got %q", dir)
	}
	dir = filepath.Clean(dir)
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("checking CLAUDE_PROJECT_DIR: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("CLAUDE_PROJECT_DIR %q is not a directory", dir)
	}
	return dir, nil
}

// defaultHookName derives the hookOverrides key for a command from its
// executable: "/p/.claude/hooks/package-guard.py" becomes "package-guard".
func defaultHookName(command string) string {
	base := filepath.Base(command)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func newSandboxStatusCmd(probe capabilityProbe) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Display sandbox capabilities and tier",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			caps := probe(ctx)
			// Resolve exactly as `sandbox exec` does, so the reported backend is
			// the one exec will run hooks under.
			backend, tier := backendselect.ResolveBackend(*caps)

			if jsonOutput {
				return printSandboxStatusJSON(cmd, caps, backend.Name(), tier)
			}
			return printSandboxStatusText(cmd, caps, backend.Name(), tier)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func printSandboxStatusText(cmd *cobra.Command, caps *sandbox.SystemCapabilities, backendName string, tier sandbox.DegradationTier) error {
	w := cmd.OutOrStdout()

	fmt.Fprintf(w, "Sandbox Status\n")
	fmt.Fprintf(w, "==============\n\n")
	fmt.Fprintf(w, "  %-18s %s\n", "Backend:", backendName)
	fmt.Fprintf(w, "  %-18s %s\n", "Tier:", tier.String())
	fmt.Fprintf(w, "  %-18s %s\n", "Security Level:", sandbox.TierSecurityLevel(tier))
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Capabilities\n")
	sym := func(b bool) string {
		if b {
			return "[OK]"
		}
		return "[--]"
	}
	fmt.Fprintf(w, "  %-18s %s\n", "Bubblewrap:", sym(caps.HasBwrap))
	fmt.Fprintf(w, "  %-18s %s\n", "User Namespaces:", sym(caps.HasUserNS))
	fmt.Fprintf(w, "  %-18s %s (ABI v%d)\n", "Landlock:", sym(caps.LandlockABI > 0), caps.LandlockABI)
	fmt.Fprintf(w, "  %-18s %s\n", "Seccomp:", sym(caps.HasSeccomp))
	fmt.Fprintf(w, "  %-18s %s\n", "Cgroups v2:", sym(caps.HasCgroupV2))
	fmt.Fprintf(w, "  %-18s %s\n", "Cgroup Delegation:", sym(caps.HasCgroupDeleg))
	fmt.Fprintf(w, "  %-18s %s\n", "systemd-run:", sym(caps.HasSystemdRun))

	if caps.KernelVersion != "" {
		fmt.Fprintf(w, "  %-18s %s\n", "Kernel:", caps.KernelVersion)
	}

	if msg := sandbox.TierMessage(tier); msg != "" {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Note: %s\n", msg)
	}

	if layers := unenforceableLayers(tier); len(layers) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Warning: the kernel supports %s, but the enforcement tool(s) are not\n",
			strings.Join(layers, " and "))
		fmt.Fprintf(w, "         installed in this build, so %s will NOT be applied at exec time.\n",
			pluralLayers(layers))
	}

	return nil
}

// unenforceableLayers returns the LSM layer names that the given tier advertises
// but that cannot actually be enforced because their userspace tool is missing
// (ll-restrict for Landlock, a compiled BPF filter for seccomp). It lets the
// status command stay honest even when kernel-capability probing reports a tier
// stronger than the tool set can deliver.
func unenforceableLayers(tier sandbox.DegradationTier) []string {
	var layers []string
	if sandbox.TierClaimsLandlock(tier) && sandbox.LLRestrictBin() == "" {
		layers = append(layers, "Landlock")
	}
	if sandbox.TierClaimsSeccomp(tier) && sandbox.SeccompFilterFile() == "" {
		layers = append(layers, "seccomp")
	}
	return layers
}

func pluralLayers(layers []string) string {
	if len(layers) == 1 {
		return "it"
	}
	return "they"
}

// sandboxStatusJSON is the machine-readable shape emitted by
// `sandbox status --json`. Backend names the backend `sandbox exec` will use.
// UnenforceableLayers reports layers the tier advertises but cannot enforce, so
// machine consumers do not treat "full" as a guarantee that every layer is
// applied.
type sandboxStatusJSON struct {
	Backend             string                  `json:"backend"`
	Tier                string                  `json:"tier"`
	SecurityLevel       string                  `json:"security_level"`
	UnenforceableLayers []string                `json:"unenforceable_layers"`
	Capabilities        sandboxCapabilitiesJSON `json:"capabilities"`
}

type sandboxCapabilitiesJSON struct {
	Bwrap       bool   `json:"bwrap"`
	UserNS      bool   `json:"user_ns"`
	LandlockABI int    `json:"landlock_abi"`
	Seccomp     bool   `json:"seccomp"`
	CgroupV2    bool   `json:"cgroup_v2"`
	CgroupDeleg bool   `json:"cgroup_deleg"`
	SystemdRun  bool   `json:"systemd_run"`
	Kernel      string `json:"kernel"`
}

func printSandboxStatusJSON(cmd *cobra.Command, caps *sandbox.SystemCapabilities, backendName string, tier sandbox.DegradationTier) error {
	unenforceable := unenforceableLayers(tier)
	if unenforceable == nil {
		unenforceable = []string{}
	}
	status := sandboxStatusJSON{
		Backend:             backendName,
		Tier:                tier.String(),
		SecurityLevel:       sandbox.TierSecurityLevel(tier),
		UnenforceableLayers: unenforceable,
		Capabilities: sandboxCapabilitiesJSON{
			Bwrap:       caps.HasBwrap,
			UserNS:      caps.HasUserNS,
			LandlockABI: caps.LandlockABI,
			Seccomp:     caps.HasSeccomp,
			CgroupV2:    caps.HasCgroupV2,
			CgroupDeleg: caps.HasCgroupDeleg,
			SystemdRun:  caps.HasSystemdRun,
			Kernel:      caps.KernelVersion,
		},
	}
	out, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshaling sandbox status: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(out))
	return nil
}

// runSandboxed resolves the strongest available sandbox backend for the probed
// capabilities and runs the hook inside it. Any weaker-than-full isolation, and
// any layer the tier advertises but cannot enforce, is reported on warn as well
// as the log: the log file is not visible by default, and an unsandboxed run
// must never look like a sandboxed one.
func runSandboxed(ctx context.Context, cfg *sandbox.SandboxConfig, caps *sandbox.SystemCapabilities, warn io.Writer) (*sandbox.SandboxResult, error) {
	backend, tier := backendselect.ResolveBackend(*caps)
	app := branding.Get().AppName

	if msg := sandbox.TierMessage(tier); msg != "" {
		slog.Warn("sandbox degraded", "backend", backend.Name(), "tier", tier.String(), "message", msg)
		fmt.Fprintf(warn, "%s sandbox: degraded isolation (backend %s, tier %s): %s\n",
			app, backend.Name(), tier, msg)
	}
	if layers := unenforceableLayers(tier); len(layers) > 0 {
		slog.Warn("sandbox layers not enforceable", "tier", tier.String(), "layers", layers)
		fmt.Fprintf(warn, "%s sandbox: %s advertised by tier %s will NOT be applied (enforcement tool missing from this build)\n",
			app, strings.Join(layers, " and "), tier)
	}

	// bubblewrap builds the sandbox from an empty root, so the hook's executable
	// and script interpreter must be reachable through what it mounts.
	if _, confined := backend.(*bwrap.BubblewrapBackend); confined {
		hookCmd, err := namespaceHookCommand(cfg)
		if err != nil {
			return nil, fmt.Errorf("preparing hook command for %s: %w", backend.Name(), err)
		}
		cfg.HookCommand = hookCmd
	}

	result, err := backend.RunHook(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("running hook under %s: %w", backend.Name(), err)
	}
	return result, nil
}
