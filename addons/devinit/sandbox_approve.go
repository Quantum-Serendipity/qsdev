package devinit

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/policy"
)

// approvalStoreFunc opens the sandbox policy approval store. The approve
// command takes it as a parameter so tests can use a private store.
type approvalStoreFunc func() (*policy.ApprovalStore, error)

func newSandboxApproveCmd(openStore approvalStoreFunc) *cobra.Command {
	var policyPath string

	cmd := &cobra.Command{
		Use:   "approve",
		Short: "Approve the project's sandbox policy for use by sandbox exec",
		Long: `Records the current content of the sandbox policy (the policy file and
the *.nix files beside it) as approved in ~/.qsdev/. The policy comes from
the repository whose hooks the sandbox contains, so "sandbox exec" refuses
to evaluate it, and blocks the hook, until this exact content is approved.
Any later change to those files needs a new approval.

The policy is evaluated first, with Nix's restricted evaluation (no access
to other files, the environment or the network); a policy that cannot be
evaluated is not approved. Mounts it declares outside the project directory
and /nix/store are listed, since sandbox exec skips them. Approving requires
a human at an interactive terminal, outside any AI agent session, who
confirms the prompt.`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			return runSandboxApprove(ctx, cmd, policyPath, openStore)
		},
	}
	addPolicyFlag(cmd, &policyPath)
	return cmd
}

func runSandboxApprove(ctx context.Context, cmd *cobra.Command, policyPath string, openStore approvalStoreFunc) error {
	// The approval lets repository content configure the sandbox that
	// contains the repository's hooks, so it must come from a human.
	if err := requireHuman(cmd, "sandbox approve", "a sandbox policy must be reviewed and approved by a human"); err != nil {
		return err
	}
	projectDir, err := sandboxProjectDir()
	if err != nil {
		return err
	}
	policyPath = resolvePolicyPath(cmd, policyPath, projectDir)
	if _, err := os.Stat(policyPath); err != nil {
		return fmt.Errorf("sandbox policy: %w", err)
	}
	store, err := openStore()
	if err != nil {
		return err
	}

	snap, err := policy.ReadSnapshot(policyPath)
	if err != nil {
		return fmt.Errorf("sandbox policy: %w", err)
	}
	spec, err := policy.EvaluateSnapshot(ctx, snap)
	if err != nil {
		return fmt.Errorf("sandbox policy not approved: %w", err)
	}

	w := cmd.OutOrStdout()
	printPolicySummary(w, snap, policy.RejectedMounts(spec, projectDir))
	ok, err := confirmYes(cmd, "Approve the current content of the files listed above (review them first) for sandbox exec? [y/N]: ")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(w, "Aborted; policy not approved.")
		return nil
	}
	if err := store.Approve(snap, time.Now()); err != nil {
		return err
	}
	fmt.Fprintf(w, "Approved sandbox policy %s (recorded in %s)\n", snap.Path, store.Path())
	return nil
}

// printPolicySummary shows the policy being approved: its path, digest, the
// files the approval covers and any mounts sandbox exec will skip.
func printPolicySummary(w io.Writer, snap *policy.Snapshot, rejected []string) {
	fmt.Fprintf(w, "Sandbox policy %s\n", snap.Path)
	fmt.Fprintf(w, "  %-8s %s\n", "Digest:", snap.Digest)
	fmt.Fprintf(w, "  %-8s %s\n", "Files:", strings.Join(snap.FileNames(), ", "))
	if len(rejected) > 0 {
		fmt.Fprintln(w, "\nWarning: these mounts are not allowed and will be skipped:")
		for _, r := range rejected {
			fmt.Fprintf(w, "  - %s\n", r)
		}
	}
	fmt.Fprintln(w)
}
