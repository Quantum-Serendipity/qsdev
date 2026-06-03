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

// isVersionMismatch is a test helper that checks if err is a *VersionMismatchError.
func isVersionMismatch(err error, target **VersionMismatchError) bool {
	if e, ok := err.(*VersionMismatchError); ok {
		*target = e
		return true
	}
	return false
}
