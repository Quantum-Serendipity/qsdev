package devinit

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/policy"
)

// isolateApprovals points the sandbox policy approval store and the policy
// cache beside it at a private home directory. It uses t.Setenv, so callers
// are not parallel.
func isolateApprovals(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

// recordApproval approves the policy's current content in the default store.
func recordApproval(t *testing.T, policyPath string) {
	t.Helper()
	store, err := policy.DefaultApprovalStore()
	if err != nil {
		t.Fatalf("opening approval store: %v", err)
	}
	snap, err := policy.ReadSnapshot(policyPath)
	if err != nil {
		t.Fatalf("reading policy snapshot: %v", err)
	}
	if err := store.Approve(snap, time.Now()); err != nil {
		t.Fatalf("approving policy: %v", err)
	}
}

// writeProjectPolicy writes .qsdev/policy.nix under project and returns its path.
func writeProjectPolicy(t *testing.T, project, content string) string {
	t.Helper()
	dir := filepath.Join(project, ".qsdev")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "policy.nix")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// runApproveCmd runs `sandbox approve` against the default store with stdin
// as the confirmation answer, simulating a human at a terminal.
func runApproveCmd(t *testing.T, stdin string) (string, error) {
	t.Helper()
	orig := humanAtTerminal
	t.Cleanup(func() { humanAtTerminal = orig })
	humanAtTerminal = func(io.Reader) bool { return true }
	t.Setenv("CLAUDECODE", "")

	cmd := newSandboxApproveCmd(policy.DefaultApprovalStore)
	var out bytes.Buffer
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SilenceErrors = true
	err := cmd.Execute()
	return out.String(), err
}

// policyApproved reports whether the policy's current content is approved in
// the default store.
func policyApproved(t *testing.T, policyPath string) bool {
	t.Helper()
	store, err := policy.DefaultApprovalStore()
	if err != nil {
		t.Fatalf("opening approval store: %v", err)
	}
	snap, err := policy.ReadSnapshot(policyPath)
	if err != nil {
		t.Fatalf("reading policy snapshot: %v", err)
	}
	return store.Check(snap) == nil
}

// TestSandboxExec_UnapprovedPolicyFailsClosed is the regression test for the
// repository-controlled policy being evaluated on every hook: a policy nobody
// approved blocks the hook (exit 2) without being evaluated or run.
func TestSandboxExec_UnapprovedPolicyFailsClosed(t *testing.T) {
	isolateApprovals(t)
	project := t.TempDir()
	t.Setenv("CLAUDE_PROJECT_DIR", project)
	writeProjectPolicy(t, project, `{ backend = "auto"; }`)
	marker := filepath.Join(project, "hook-ran")

	_, _, err := runSandboxExec(t, noSandboxProbe, "", "--", "touch", marker)
	if got := requireExitCode(t, err); got != hookBlockExitCode {
		t.Fatalf("exit code = %d, want %d (err: %v)", got, hookBlockExitCode, err)
	}
	if !strings.Contains(err.Error(), "not approved") || !strings.Contains(err.Error(), "sandbox approve") {
		t.Errorf("error = %v, want it to say the policy is not approved and how to approve it", err)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("hook ran under an unapproved policy")
	}
}

// TestSandboxApprove_Refusals pins that approve records nothing unless a
// human at a terminal, outside an agent session, confirms an evaluable policy.
// It needs no nix: every case stops before or at evaluation, or is declined.
func TestSandboxApprove_Refusals(t *testing.T) {
	type approveCase struct {
		name        string
		policy      string // "" writes no policy file
		interactive bool
		agent       bool
		stdin       string
		wantErr     string
	}
	tests := []approveCase{
		{name: "agent session", policy: "{ }", interactive: true, agent: true, wantErr: "AI agent session"},
		{name: "no terminal", policy: "{ }", wantErr: "interactive terminal"},
		{name: "missing policy", interactive: true, wantErr: "sandbox policy"},
		{name: "unevaluable policy", policy: "{ this is not nix", interactive: true, stdin: "y\n", wantErr: "not approved"},
	}
	if _, err := exec.LookPath("nix"); err == nil {
		// Declining needs an evaluable policy, so a working nix.
		tests = append(tests, approveCase{name: "declined", policy: "{ }", interactive: true, stdin: "n\n"})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateApprovals(t)
			project := t.TempDir()
			t.Setenv("CLAUDE_PROJECT_DIR", project)
			policyPath := filepath.Join(project, ".qsdev", "policy.nix")
			if tt.policy != "" {
				policyPath = writeProjectPolicy(t, project, tt.policy)
			}

			orig := humanAtTerminal
			t.Cleanup(func() { humanAtTerminal = orig })
			humanAtTerminal = func(io.Reader) bool { return tt.interactive }
			agent := ""
			if tt.agent {
				agent = "1"
			}
			t.Setenv("CLAUDECODE", agent)

			cmd := newSandboxApproveCmd(policy.DefaultApprovalStore)
			var out bytes.Buffer
			cmd.SetIn(strings.NewReader(tt.stdin))
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SilenceErrors = true
			err := cmd.Execute()

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("approve: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("approve error = %v, want one containing %q", err, tt.wantErr)
			}
			if tt.policy != "" && policyApproved(t, policyPath) {
				t.Error("the policy was recorded as approved")
			}
		})
	}
}

// TestSandboxApprove_EndToEnd approves a policy with the real `nix eval`, runs
// a hook under it, and pins that an edit after approval blocks the hook again.
func TestSandboxApprove_EndToEnd(t *testing.T) {
	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix is not installed")
	}
	isolateApprovals(t)
	project := t.TempDir()
	t.Setenv("CLAUDE_PROJECT_DIR", project)
	policyPath := writeProjectPolicy(t, project, `{
  hookOverrides.touch.extraMounts = [
    { source = "/run/user/1000"; target = "/run/user/1000"; }
  ];
}`)

	out, err := runApproveCmd(t, "yes\n")
	if err != nil {
		t.Fatalf("approve: %v\n%s", err, out)
	}
	for _, want := range []string{"Digest:", "policy.nix", "hookOverrides.touch", "/run/user/1000", "Approved sandbox policy"} {
		if !strings.Contains(out, want) {
			t.Errorf("approve output missing %q:\n%s", want, out)
		}
	}
	if !policyApproved(t, policyPath) {
		t.Fatal("the confirmed policy was not recorded as approved")
	}

	marker := filepath.Join(project, "hook-ran")
	if _, _, err := runSandboxExec(t, noSandboxProbe, "", "--", "touch", marker); err != nil {
		t.Fatalf("exec under the approved policy: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("hook did not run under the approved policy: %v", err)
	}

	if err := os.WriteFile(policyPath, []byte(`{ backend = "auto"; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err = runSandboxExec(t, noSandboxProbe, "", "--", "true")
	if got := requireExitCode(t, err); got != hookBlockExitCode {
		t.Errorf("exec after editing the approved policy: exit code = %d, want %d (err: %v)", got, hookBlockExitCode, err)
	}
}
