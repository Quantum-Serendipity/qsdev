package config

import (
	"strings"
	"testing"
)

func TestParseVersionConstraint_SimpleOperators(t *testing.T) {
	tests := []struct {
		constraint string
		version    string
		want       bool
	}{
		{">= 1.0.0", "1.0.0", true},
		{">= 1.0.0", "0.9.0", false},
		{"<= 2.0.0", "2.0.0", true},
		{"<= 2.0.0", "2.1.0", false},
		{"> 1.0.0", "1.0.1", true},
		{"> 1.0.0", "1.0.0", false},
		{"< 2.0.0", "1.9.9", true},
		{"< 2.0.0", "2.0.0", false},
		{"= 1.5.0", "1.5.0", true},
		{"= 1.5.0", "1.5.1", false},
		{"!= 1.0.0", "1.0.1", true},
		{"!= 1.0.0", "1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.constraint+"_"+tt.version, func(t *testing.T) {
			vc, err := ParseVersionConstraint(tt.constraint)
			if err != nil {
				t.Fatalf("ParseVersionConstraint(%q): %v", tt.constraint, err)
			}
			got, err := vc.Check(tt.version)
			if err != nil {
				t.Fatalf("Check(%q): %v", tt.version, err)
			}
			if got != tt.want {
				t.Errorf("Check(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestParseVersionConstraint_PessimisticTwoSegment(t *testing.T) {
	vc, err := ParseVersionConstraint("~> 0.15")
	if err != nil {
		t.Fatalf("ParseVersionConstraint: %v", err)
	}

	tests := []struct {
		version string
		want    bool
	}{
		{"0.15.0", true},
		{"0.15.9", true},
		{"0.16.0", false},
		{"0.14.9", false},
		{"1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := vc.Check(tt.version)
			if err != nil {
				t.Fatalf("Check(%q): %v", tt.version, err)
			}
			if got != tt.want {
				t.Errorf("Check(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestParseVersionConstraint_PessimisticThreeSegment(t *testing.T) {
	vc, err := ParseVersionConstraint("~> 0.15.3")
	if err != nil {
		t.Fatalf("ParseVersionConstraint: %v", err)
	}

	tests := []struct {
		version string
		want    bool
	}{
		{"0.15.3", true},
		{"0.15.9", true},
		{"0.15.2", false},
		{"0.16.0", false},
		{"0.14.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := vc.Check(tt.version)
			if err != nil {
				t.Fatalf("Check(%q): %v", tt.version, err)
			}
			if got != tt.want {
				t.Errorf("Check(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

// Regression: the pessimistic operator must keep its upper bound inside OR
// groups, compound constraints and the single-number form.
func TestParseVersionConstraint_PessimisticBounded(t *testing.T) {
	t.Parallel()
	tests := []struct {
		constraint string
		version    string
		want       bool
	}{
		{"~> 1", "1.9.0", true},
		{"~> 1", "3.1.0", false},
		{"~> 1.2 || ~> 2.0", "1.2.5", true},
		{"~> 1.2 || ~> 2.0", "2.0.1", true},
		{"~> 1.2 || ~> 2.0", "1.3.0", false},
		{"~> 1.2 || ~> 2.0", "3.1.0", false},
		{">= 1.0, ~> 1.2 || ^3.0", "2.0.1", false},
		{">= 1.0, ~> 1.2 || ^3.0", "3.4.0", true},
		{"~>0.15", "0.15.4", true},
		{"~>0.15", "0.16.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.constraint+"/"+tt.version, func(t *testing.T) {
			t.Parallel()
			vc, err := ParseVersionConstraint(tt.constraint)
			if err != nil {
				t.Fatalf("ParseVersionConstraint(%q): %v", tt.constraint, err)
			}
			got, err := vc.Check(tt.version)
			if err != nil {
				t.Fatalf("Check(%q): %v", tt.version, err)
			}
			if got != tt.want {
				t.Errorf("%q.Check(%q) = %v, want %v", tt.constraint, tt.version, got, tt.want)
			}
		})
	}
}

func TestParseVersionConstraint_CaretZeroMajor(t *testing.T) {
	vc, err := ParseVersionConstraint("^0.15.0")
	if err != nil {
		t.Fatalf("ParseVersionConstraint: %v", err)
	}

	tests := []struct {
		version string
		want    bool
	}{
		{"0.15.0", true},
		{"0.15.5", true},
		{"0.16.0", false},
		{"1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := vc.Check(tt.version)
			if err != nil {
				t.Fatalf("Check(%q): %v", tt.version, err)
			}
			if got != tt.want {
				t.Errorf("Check(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestParseVersionConstraint_CaretNonZeroMajor(t *testing.T) {
	vc, err := ParseVersionConstraint("^1.2.3")
	if err != nil {
		t.Fatalf("ParseVersionConstraint: %v", err)
	}

	tests := []struct {
		version string
		want    bool
	}{
		{"1.2.3", true},
		{"1.9.0", true},
		{"2.0.0", false},
		{"1.2.2", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := vc.Check(tt.version)
			if err != nil {
				t.Fatalf("Check(%q): %v", tt.version, err)
			}
			if got != tt.want {
				t.Errorf("Check(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestParseVersionConstraint_CommaAND(t *testing.T) {
	vc, err := ParseVersionConstraint(">= 0.15.0, < 1.0.0")
	if err != nil {
		t.Fatalf("ParseVersionConstraint: %v", err)
	}

	tests := []struct {
		version string
		want    bool
	}{
		{"0.15.0", true},
		{"0.99.0", true},
		{"1.0.0", false},
		{"0.14.9", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := vc.Check(tt.version)
			if err != nil {
				t.Fatalf("Check(%q): %v", tt.version, err)
			}
			if got != tt.want {
				t.Errorf("Check(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestParseVersionConstraint_InvalidConstraint(t *testing.T) {
	_, err := ParseVersionConstraint("not a version")
	if err == nil {
		t.Fatal("expected error for invalid constraint")
	}
}

func TestParseVersionConstraint_EmptyConstraint(t *testing.T) {
	_, err := ParseVersionConstraint("")
	if err == nil {
		t.Fatal("expected error for empty constraint")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Errorf("error = %q, want empty constraint message", err.Error())
	}
}

func TestParseVersionConstraint_VPrefix(t *testing.T) {
	vc, err := ParseVersionConstraint(">= 1.0.0")
	if err != nil {
		t.Fatalf("ParseVersionConstraint: %v", err)
	}

	got, err := vc.Check("v1.2.3")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !got {
		t.Error("v-prefixed version should be accepted")
	}
}

func TestCheckBinaryVersion_Satisfied(t *testing.T) {
	err := CheckBinaryVersion(">= 1.0.0", "1.5.0")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckBinaryVersion_NotSatisfied(t *testing.T) {
	err := CheckBinaryVersion(">= 2.0.0", "1.5.0")
	if err == nil {
		t.Fatal("expected error")
	}
	var mismatch *VersionMismatchError
	if ok := isVersionMismatch(err, &mismatch); !ok {
		t.Fatalf("expected *VersionMismatchError, got %T: %v", err, err)
	}
	if mismatch.BinaryVersion != "1.5.0" {
		t.Errorf("BinaryVersion = %q, want 1.5.0", mismatch.BinaryVersion)
	}
	if mismatch.Constraint != ">= 2.0.0" {
		t.Errorf("Constraint = %q, want >= 2.0.0", mismatch.Constraint)
	}
	if mismatch.UpgradeCommand == "" {
		t.Error("UpgradeCommand should not be empty")
	}
	// Check Error() includes actionable message.
	errMsg := mismatch.Error()
	if !strings.Contains(errMsg, "1.5.0") || !strings.Contains(errMsg, ">= 2.0.0") {
		t.Errorf("Error() = %q, want version and constraint info", errMsg)
	}
}

func TestCheckBinaryVersion_NoConstraint(t *testing.T) {
	err := CheckBinaryVersion("", "1.5.0")
	if err != nil {
		t.Errorf("unexpected error for empty constraint: %v", err)
	}
}

func TestCheckBinaryVersion_DevBuild(t *testing.T) {
	devVersions := []string{"dev", "(devel)", ""}
	for _, v := range devVersions {
		t.Run(v, func(t *testing.T) {
			err := CheckBinaryVersion(">= 99.0.0", v)
			if err != nil {
				t.Errorf("dev build %q should pass any constraint, got: %v", v, err)
			}
		})
	}
}

func TestCheckVersionRatchet_Newer(t *testing.T) {
	warn := CheckVersionRatchet("2.0.0", "1.0.0")
	if warn != nil {
		t.Errorf("newer version should not produce warning, got: %v", warn)
	}
}

func TestCheckVersionRatchet_Older(t *testing.T) {
	warn := CheckVersionRatchet("1.0.0", "2.0.0")
	if warn == nil {
		t.Fatal("older version should produce warning")
		return
	}
	if warn.CurrentVersion != "1.0.0" {
		t.Errorf("CurrentVersion = %q, want 1.0.0", warn.CurrentVersion)
	}
	if warn.LastRunVersion != "2.0.0" {
		t.Errorf("LastRunVersion = %q, want 2.0.0", warn.LastRunVersion)
	}
	// Check Error() message.
	errMsg := warn.Error()
	if !strings.Contains(errMsg, "1.0.0") || !strings.Contains(errMsg, "2.0.0") {
		t.Errorf("Error() = %q, want both versions mentioned", errMsg)
	}
}

func TestCheckVersionRatchet_Same(t *testing.T) {
	warn := CheckVersionRatchet("1.0.0", "1.0.0")
	if warn != nil {
		t.Errorf("same version should not produce warning, got: %v", warn)
	}
}

func TestCheckVersionRatchet_DevBuild(t *testing.T) {
	// Dev current version.
	warn := CheckVersionRatchet("dev", "2.0.0")
	if warn != nil {
		t.Error("dev current version should not produce warning")
	}

	// Dev last run version.
	warn = CheckVersionRatchet("1.0.0", "dev")
	if warn != nil {
		t.Error("dev last run version should not produce warning")
	}

	// Empty last run version.
	warn = CheckVersionRatchet("1.0.0", "")
	if warn != nil {
		t.Error("empty last run version should not produce warning")
	}
}

func TestVersionConstraint_String(t *testing.T) {
	vc, err := ParseVersionConstraint("~> 0.15")
	if err != nil {
		t.Fatal(err)
	}
	if vc.String() != "~> 0.15" {
		t.Errorf("String() = %q, want %q", vc.String(), "~> 0.15")
	}
}

// Regression: the constraint written by init must be a lower bound, so a
// newer patch release (with or without build metadata) still satisfies it,
// and dev builds must not write an invalid constraint.
func TestMinimumVersionConstraint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		binaryVersion string
		want          string
	}{
		{"release", "0.8.0", ">= 0.8.0"},
		{"nix build metadata", "0.8.0+abc1234", ">= 0.8.0"},
		{"v prefix", "v1.2.3", ">= 1.2.3"},
		{"prerelease kept", "1.3.0-rc.1+abc", ">= 1.3.0-rc.1"},
		{"dev build", "dev", ""},
		{"devel build", "(devel)", ""},
		{"empty", "", ""},
		{"unparseable", "not-a-version", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := MinimumVersionConstraint(tt.binaryVersion)
			if got != tt.want {
				t.Fatalf("MinimumVersionConstraint(%q) = %q, want %q", tt.binaryVersion, got, tt.want)
			}
			if got == "" {
				return
			}
			// The binary that wrote the constraint must satisfy it.
			if err := CheckBinaryVersion(got, tt.binaryVersion); err != nil {
				t.Fatalf("CheckBinaryVersion(%q, %q) = %v, want nil", got, tt.binaryVersion, err)
			}
		})
	}
}

func TestMinimumVersionConstraint_AcceptsNewerReleases(t *testing.T) {
	t.Parallel()
	constraint := MinimumVersionConstraint("0.8.0+abc1234")
	for _, newer := range []string{"0.8.0+abc1234", "0.8.0", "0.8.1+aaa", "0.9.0", "1.0.0"} {
		if err := CheckBinaryVersion(constraint, newer); err != nil {
			t.Errorf("CheckBinaryVersion(%q, %q) = %v, want nil", constraint, newer, err)
		}
	}
	if err := CheckBinaryVersion(constraint, "0.7.9"); err == nil {
		t.Errorf("CheckBinaryVersion(%q, 0.7.9) = nil, want mismatch", constraint)
	}
}

// isVersionMismatch is a test helper that checks if err is a *VersionMismatchError.
func isVersionMismatch(err error, target **VersionMismatchError) bool {
	if e, ok := err.(*VersionMismatchError); ok {
		*target = e
		return true
	}
	return false
}
