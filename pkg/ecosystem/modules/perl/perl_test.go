package perl_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/perl"
)

// newModule returns a fresh Module for testing.
func newModule() *perl.Module {
	return &perl.Module{}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ ecosystem.EcosystemModule = (*perl.Module)(nil)
}

// --- Basic metadata ---

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, newModule(), "perl", "Perl", 4)
}

// --- Detection tests ---

func TestDetect_Cpanfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cpanfile"), []byte("requires 'Mojolicious';\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", r.Confidence)
	}
	if !slices.Contains(r.Evidence, "cpanfile found") {
		t.Errorf("Evidence = %v, want to contain %q", r.Evidence, "cpanfile found")
	}
}

func TestDetect_MakefilePL(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Makefile.PL"), []byte("use ExtUtils::MakeMaker;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want Probable", r.Confidence)
	}
}

func TestDetect_CpanfileSnapshot(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cpanfile.snapshot"), []byte("# carton snapshot\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	r := m.Detect(dir)

	if !r.Detected {
		t.Fatal("expected Detected = true")
	}
	if r.SuggestedConfig.PackageManager != "carton" {
		t.Errorf("PackageManager = %q, want %q", r.SuggestedConfig.PackageManager, "carton")
	}
}

func TestDetect_NotPresent(t *testing.T) {
	dir := t.TempDir()

	m := newModule()
	r := m.Detect(dir)

	if r.Detected {
		t.Fatal("expected Detected = false for empty directory")
	}
	if r.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want Absent", r.Confidence)
	}
	if len(r.Evidence) != 0 {
		t.Errorf("Evidence = %v, want empty", r.Evidence)
	}
}

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment(t *testing.T) {
	m := newModule()
	frag, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}
	if frag == "" {
		t.Error("DevenvNixFragment() returned empty string")
	}
	if !strings.Contains(frag, "perl") {
		t.Errorf("fragment missing perl reference:\n%s", frag)
	}
}

// TestDevenvPackages verifies the CI tools devenv's Perl language lacks are
// provisioned for every configuration: carton also creates the
// cpanfile.snapshot a project without one needs.
func TestDevenvPackages(t *testing.T) {
	t.Parallel()
	want := []string{"perlPackages.Carton", "perlPackages.CPANAudit"}
	for _, cfg := range []ecosystem.ModuleConfig{{}, {PackageManager: "carton"}} {
		if got := (&perl.Module{}).DevenvPackages(cfg); !slices.Equal(got, want) {
			t.Errorf("DevenvPackages(%+v) = %q, want %q", cfg, got, want)
		}
	}
}

// --- CI commands ---

// TestCICommands_GatedOnCarton checks only Carton projects (a committed
// cpanfile.snapshot) get the Carton install and the cpan-audit scan, and that
// the scan audits the project's declared dependencies rather than @INC
// (U10-07).
func TestCICommands_GatedOnCarton(t *testing.T) {
	t.Parallel()

	type step struct {
		name, command string
		phase         ecosystem.CIPhase
	}
	tests := []struct {
		pm   string
		want []step
	}{
		{pm: ""},
		{pm: "carton", want: []step{
			{name: "carton-install", command: "carton install --deployment", phase: ecosystem.CIPhaseInstall},
			{name: "cpan-audit", command: "cpan-audit deps .", phase: ecosystem.CIPhaseScan},
		}},
	}
	for _, tt := range tests {
		t.Run("pm="+tt.pm, func(t *testing.T) {
			t.Parallel()
			var got []step
			for _, c := range newModule().CICommands(ecosystem.ModuleConfig{PackageManager: tt.pm}) {
				got = append(got, step{name: c.Name, command: c.Command, phase: c.Phase})
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("CICommands(%q) = %+v, want %+v", tt.pm, got, tt.want)
			}
		})
	}
}

// TestSetupWarnings_NoSnapshot checks a cpanfile project without a committed
// cpanfile.snapshot is told its security scan does not run.
func TestSetupWarnings_NoSnapshot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files []string
		want  bool
	}{
		{name: "cpanfile only", files: []string{"cpanfile"}, want: true},
		{name: "cpanfile and snapshot", files: []string{"cpanfile", "cpanfile.snapshot"}},
		{name: "Makefile.PL only", files: []string{"Makefile.PL"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, f := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, f), []byte("# x\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := newModule().SetupWarnings(dir, ecosystem.ModuleConfig{})
			if !tt.want {
				if len(got) != 0 {
					t.Fatalf("SetupWarnings() = %q, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], "security scan not run") ||
				!strings.Contains(got[0], "carton install") || !strings.Contains(got[0], "commit cpanfile.snapshot") {
				t.Fatalf("SetupWarnings() = %q, want one warning to run `carton install` and commit cpanfile.snapshot", got)
			}
		})
	}
}
