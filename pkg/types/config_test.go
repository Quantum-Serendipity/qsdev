package types_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestHooksConfigClone_PackageGuard checks that Clone deep-copies the
// package_guard lists and the min_age_days pointer, so mutating the copy
// never reaches the committed policy it was taken from.
func TestHooksConfigClone_PackageGuard(t *testing.T) {
	t.Parallel()
	days := 7
	orig := types.HooksConfig{PackageGuard: types.PackageGuardConfig{
		Allowlist:  []string{"lodash"},
		Denylist:   []string{"npm:evil"},
		MinAgeDays: &days,
		NewDepGate: types.NewDepGateDeny,
	}}
	cp := orig.Clone()
	cp.PackageGuard.Allowlist[0] = "changed"
	cp.PackageGuard.Denylist[0] = "changed"
	*cp.PackageGuard.MinAgeDays = 1

	if got := orig.PackageGuard.Allowlist[0]; got != "lodash" {
		t.Errorf("original allowlist mutated through clone: %q", got)
	}
	if got := orig.PackageGuard.Denylist[0]; got != "npm:evil" {
		t.Errorf("original denylist mutated through clone: %q", got)
	}
	if got := *orig.PackageGuard.MinAgeDays; got != 7 {
		t.Errorf("original min_age_days mutated through clone: %d", got)
	}
	if cp.PackageGuard.NewDepGate != types.NewDepGateDeny {
		t.Errorf("clone new_dep_gate = %q, want %q", cp.PackageGuard.NewDepGate, types.NewDepGateDeny)
	}
	if empty := (types.HooksConfig{}).Clone(); empty.PackageGuard.MinAgeDays != nil {
		t.Errorf("clone of unset min_age_days = %v, want nil", empty.PackageGuard.MinAgeDays)
	}
}

// TestHooksConfigClone_CredentialScan checks that Clone deep-copies
// credential_scan.extra_patterns.
func TestHooksConfigClone_CredentialScan(t *testing.T) {
	t.Parallel()
	orig := types.HooksConfig{CredentialScan: types.CredentialScanConfig{ExtraPatterns: []string{"corp_[a-z]{20}"}}}
	cp := orig.Clone()
	cp.CredentialScan.ExtraPatterns[0] = "changed"
	if got := orig.CredentialScan.ExtraPatterns[0]; got != "corp_[a-z]{20}" {
		t.Errorf("original extra_patterns mutated through clone: %q", got)
	}
}

func TestNewDepGates(t *testing.T) {
	t.Parallel()
	got := types.NewDepGates()
	want := []string{types.NewDepGateAllow, types.NewDepGateAsk, types.NewDepGateDeny}
	if len(got) != len(want) {
		t.Fatalf("NewDepGates() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NewDepGates() = %v, want %v", got, want)
		}
	}
	got[0] = "mutated"
	if types.NewDepGates()[0] != types.NewDepGateAllow {
		t.Error("NewDepGates returned a shared slice")
	}
}
