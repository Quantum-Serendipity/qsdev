package rlang_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestDetect_DescriptionMustBeRPackage verifies a generic DESCRIPTION file
// (prose, Perl dist metadata) no longer enables the R ecosystem.
func TestDetect_DescriptionMustBeRPackage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"R package DCF", "Package: mypkg\nVersion: 0.1.0\nImports: dplyr\n", true},
		{"prose description", "This repository contains the deployment scripts.\n", false},
		{"field not at line start", "Summary of the Package: none\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "DESCRIPTION"), []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := newModule().Detect(dir).Detected; got != tt.want {
				t.Errorf("Detected = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestManifestFiles_ReportsRenvLock verifies R manifests reach
// Version-Sentinel coverage (as uncovered) instead of being omitted.
func TestManifestFiles_ReportsRenvLock(t *testing.T) {
	t.Parallel()
	var mod ecosystem.EcosystemModule = newModule()
	mfp, ok := mod.(ecosystem.ManifestFileProvider)
	if !ok {
		t.Fatal("r module does not implement ManifestFileProvider")
	}
	files := mfp.ManifestFiles(ecosystem.ModuleConfig{})
	if len(files) != 1 {
		t.Fatalf("ManifestFiles() = %v, want 1 entry", files)
	}
	if files[0].LockFile != "renv.lock" || files[0].VSSupported {
		t.Errorf("ManifestFiles()[0] = %+v, want renv.lock lockfile, not VS-supported", files[0])
	}

	report := ecosystem.AggregateManifestCoverage([]ecosystem.EcosystemModule{mod},
		func(ecosystem.EcosystemModule) ecosystem.ModuleConfig { return ecosystem.ModuleConfig{} })
	if len(report.Uncovered) != 1 {
		t.Errorf("coverage Uncovered = %v, want the R manifest listed", report.Uncovered)
	}
}
