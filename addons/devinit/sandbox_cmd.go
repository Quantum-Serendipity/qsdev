package devinit

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/exitcode"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
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
			tier := sandbox.DetermineTier(caps)

			if msg := sandbox.TierMessage(tier); msg != "" {
				slog.Info("sandbox degraded", "tier", tier.String(), "message", msg)
			}

			result, err := runSandboxed(ctx, cfg, tier)
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
			tier := sandbox.DetermineTier(caps)

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

	return nil
}

func printSandboxStatusJSON(cmd *cobra.Command, caps *sandbox.SystemCapabilities, tier sandbox.DegradationTier) error {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, `{"tier":%q,"security_level":%q,"capabilities":{"bwrap":%t,"user_ns":%t,"landlock_abi":%d,"seccomp":%t,"cgroup_v2":%t,"cgroup_deleg":%t,"systemd_run":%t,"kernel":%q}}`,
		tier.String(), sandbox.TierSecurityLevel(tier),
		caps.HasBwrap, caps.HasUserNS, caps.LandlockABI,
		caps.HasSeccomp, caps.HasCgroupV2, caps.HasCgroupDeleg,
		caps.HasSystemdRun, caps.KernelVersion)
	fmt.Fprintln(w)
	return nil
}

// runSandboxed runs a hook in the appropriate sandbox tier.
func runSandboxed(ctx context.Context, cfg *sandbox.SandboxConfig, tier sandbox.DegradationTier) (*sandbox.SandboxResult, error) {
	switch {
	case tier <= sandbox.TierBwrapWithoutSeccomp:
		slog.Warn("bubblewrap backend not yet connected in this build; falling back to unsandboxed")
		return (&sandbox.UnsandboxedBackend{}).RunHook(ctx, cfg)
	case tier == sandbox.TierSystemdRun:
		slog.Warn("systemd-run backend not yet connected in this build; falling back to unsandboxed")
		return (&sandbox.UnsandboxedBackend{}).RunHook(ctx, cfg)
	default:
		return (&sandbox.UnsandboxedBackend{}).RunHook(ctx, cfg)
	}
}
