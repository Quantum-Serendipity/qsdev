package devinit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/exitcode"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/backendselect"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/policy"
)

func sandboxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sandbox",
		Short: "Manage hook execution sandboxing",
		Long: `Tools for managing the hook execution sandbox.

Use "sandbox exec" to run a command inside the sandbox, and
"sandbox status" to display sandbox capabilities and tier.`,
	}
	cmd.AddCommand(sandboxExecCmd(), sandboxStatusCmd())
	return cmd
}

func sandboxExecCmd() *cobra.Command {
	var category string
	var policyPath string

	cmd := &cobra.Command{
		Use:   "exec [flags] -- COMMAND [ARGS...]",
		Short: "Execute a command inside the hook sandbox",
		Long: `Runs COMMAND inside a sandboxed environment with isolation
appropriate for the specified hook category. The sandbox tier is
automatically selected based on available kernel capabilities.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("no command specified; use -- COMMAND [ARGS...]")
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			cat := sandbox.ParseHookCategory(category)

			spec, err := policy.CompilePolicy(ctx, policyPath)
			if err != nil {
				slog.Warn("policy compilation failed, using defaults", "error", err)
				spec = policy.DefaultPolicy()
			}

			cfg := policy.ToSandboxConfig(spec, cat, "")
			cfg.HookCommand = args

			caps := sandbox.ProbeCapabilitiesDefault(ctx)

			result, err := runSandboxed(ctx, cfg, caps)
			if err != nil {
				return fmt.Errorf("sandbox execution failed: %w", err)
			}

			if len(result.Stdout) > 0 {
				_, _ = os.Stdout.Write(result.Stdout)
			}
			if len(result.Stderr) > 0 {
				_, _ = os.Stderr.Write(result.Stderr)
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
		"Path to sandbox policy file")

	return cmd
}

func sandboxStatusCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Display sandbox capabilities and tier",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			caps := sandbox.ProbeCapabilitiesDefault(ctx)
			_, tier := backendselect.ResolveBackend(*caps)

			if jsonOutput {
				return printSandboxStatusJSON(cmd, caps, tier)
			}
			return printSandboxStatusText(cmd, caps, tier)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func printSandboxStatusText(cmd *cobra.Command, caps *sandbox.SystemCapabilities, tier sandbox.DegradationTier) error {
	w := cmd.OutOrStdout()

	fmt.Fprintf(w, "Sandbox Status\n")
	fmt.Fprintf(w, "==============\n\n")
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
// `sandbox status --json`. UnenforceableLayers reports layers the tier
// advertises but cannot enforce, so machine consumers do not treat "full" as a
// guarantee that every layer is applied.
type sandboxStatusJSON struct {
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

func printSandboxStatusJSON(cmd *cobra.Command, caps *sandbox.SystemCapabilities, tier sandbox.DegradationTier) error {
	unenforceable := unenforceableLayers(tier)
	if unenforceable == nil {
		unenforceable = []string{}
	}
	status := sandboxStatusJSON{
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
// capabilities and runs the hook inside it. It warns only on genuine degradation
// (any tier weaker than full), so a caller can tell when the requested isolation
// could not be fully applied.
func runSandboxed(ctx context.Context, cfg *sandbox.SandboxConfig, caps *sandbox.SystemCapabilities) (*sandbox.SandboxResult, error) {
	backend, tier := backendselect.ResolveBackend(*caps)

	if msg := sandbox.TierMessage(tier); msg != "" {
		slog.Warn("sandbox degraded", "tier", tier.String(), "message", msg)
	}

	return backend.RunHook(ctx, cfg)
}
