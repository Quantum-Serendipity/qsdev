package devinit

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/render"
)

// TestExitForAudit_Monotonic is the regression test for stricter audit levels
// (low, moderate) passing a baseline-conformance failure that the default
// "high" level catches.
func TestExitForAudit_Monotonic(t *testing.T) {
	t.Parallel()

	conformanceFail := &posture.PostureReport{}
	conformanceFail.Conformance.Baseline.Pass = false

	moderateVuln := &posture.PostureReport{}
	moderateVuln.Conformance.Baseline.Pass = true
	moderateVuln.Dependencies.Totals.Moderate = 1

	tests := []struct {
		name     string
		report   *posture.PostureReport
		level    string
		wantExit bool
	}{
		{"high catches conformance failure", conformanceFail, "high", true},
		{"moderate includes the high conformance check", conformanceFail, "moderate", true},
		{"low includes the high conformance check", conformanceFail, "low", true},
		{"info includes the high conformance check", conformanceFail, "info", true},
		{"critical does not check conformance", conformanceFail, "critical", false},
		{"none never fails", conformanceFail, "none", false},
		{"moderate vuln fails at moderate", moderateVuln, "moderate", true},
		{"moderate vuln fails at low", moderateVuln, "low", true},
		{"moderate vuln passes at high", moderateVuln, "high", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := exitForAudit(tt.report, tt.level)
			var exitErr *ExitError
			if got := errors.As(err, &exitErr); got != tt.wantExit {
				t.Errorf("exitForAudit(%q) exit = %v, want %v (err %v)", tt.level, got, tt.wantExit, err)
			}
		})
	}
}

func TestNormalizeStatusAuditLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
		wantErr  bool
	}{
		{in: "none", want: "none"},
		{in: "info", want: "info"},
		{in: "any", want: "info"},
		{in: "medium", want: "moderate"},
		{in: "critical", want: "critical"},
		{in: "bogus", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeStatusAuditLevel(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("normalizeStatusAuditLevel(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestResolveFormat_ExplicitFormat is the regression test for --format json or
// sarif silently printing text (and disabling the CI JSON default).
func TestResolveFormat_ExplicitFormat(t *testing.T) {
	t.Parallel()
	for _, f := range statusFormats {
		t.Run(string(f), func(t *testing.T) {
			t.Parallel()
			cmd := &cobra.Command{}
			if got := resolveFormat(cmd, postureStatusOptions{format: string(f)}); got != f {
				t.Errorf("resolveFormat(--format %s) = %q", f, got)
			}
		})
	}
	if got := resolveFormat(&cobra.Command{}, postureStatusOptions{format: "", jsonFlag: true}); got != render.JSON {
		t.Errorf("--json resolved to %q", got)
	}
}

func TestStatusCmd_RejectsInvalidFlags(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"--audit-level", "bogus"}, {"--format", "yaml"}} {
		cmd := statusCmd()
		cmd.SetArgs(args)
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		if err := cmd.Execute(); err == nil {
			t.Errorf("status %v: expected a validation error", args)
		}
	}
}
