package devinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/conformance"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestEvaluateCustomConformance is the regression test for .qsdev-policy.yaml
// being silently ignored by `qsdev check`: its requirements must be evaluated
// against the project's posture and reported, failing closed when they cannot
// be verified.
func TestEvaluateCustomConformance(t *testing.T) {
	t.Parallel()
	const policy = "conformance:\n  custom:\n    requirements:\n" +
		"      - name: score\n        check: score.total >= 0\n" +
		"      - name: no critical\n        check: dependencies.totals.critical == 0\n"
	tests := []struct {
		name        string
		initialized bool
		policy      string
		want        []check.PolicyRequirement
	}{
		{name: "no policy file", initialized: true},
		{
			name:        "requirements evaluated",
			initialized: true,
			policy:      policy,
			want: []check.PolicyRequirement{
				{Name: "score", Pass: true, Reason: "score.total >= 0"},
				// No --scan: zero totals are inconclusive, not clean.
				{Name: "no critical", Reason: "not scanned"},
			},
		},
		{
			// No requirements to judge: the project is not even assessed, so
			// an uninitialized project does not fail here.
			name:   "policy without custom section",
			policy: "conformance: {}\n",
		},
		{
			name:        "malformed policy fails closed",
			initialized: true,
			policy:      "conformance:\n  custom:\n    requirement: []\n",
			want: []check.PolicyRequirement{
				{Name: string(conformance.PolicyFileCheck), Reason: "field requirement not found"},
			},
		},
		{
			name:   "uninitialized project fails closed",
			policy: policy,
			want: []check.PolicyRequirement{
				{Name: string(conformance.PolicyFileCheck), Reason: "project not initialized"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.initialized {
				writeTestFile(t, filepath.Join(dir, branding.Get().ConfigFile), "tier: standard\n")
			}
			if tt.policy != "" {
				writeTestFile(t, filepath.Join(dir, conformance.PolicyFileName()), tt.policy)
			}

			got := evaluateCustomConformance(dir, false)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("got %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("got nil, want evaluated requirements")
			}
			if got.PolicyFile != conformance.PolicyFileName() {
				t.Errorf("PolicyFile = %q, want %q", got.PolicyFile, conformance.PolicyFileName())
			}
			if len(got.Requirements) != len(tt.want) {
				t.Fatalf("requirements = %+v, want %+v", got.Requirements, tt.want)
			}
			for i, want := range tt.want {
				req := got.Requirements[i]
				if req.Name != want.Name || req.Pass != want.Pass || !strings.Contains(req.Reason, want.Reason) {
					t.Errorf("requirement %d = %+v, want %+v (reason containing)", i, req, want)
				}
			}
		})
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
