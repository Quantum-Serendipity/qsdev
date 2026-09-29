package posture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan"
	"github.com/Quantum-Serendipity/qsdev/internal/vulnscan/vulnscantest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestBuildEcosystemStatuses_PythonLockSelection covers how a Python project
// is scanned: a dedicated poetry.lock is preferred over a loose requirements.txt
// (catalog order lists requirements.txt first), and a requirements.txt with no
// exact pins is reported as not scanned rather than scanned clean.
func TestBuildEcosystemStatuses_PythonLockSelection(t *testing.T) {
	t.Parallel()
	const poetryLock = "[[package]]\nname = \"requests\"\nversion = \"2.19.0\"\n"
	const looseRequirements = "requests>=2\nflask\n"

	tests := []struct {
		name        string
		files       map[string]string
		wantLock    string
		wantScanned bool
		wantHigh    int
	}{
		{
			name:        "poetry.lock wins over loose requirements.txt",
			files:       map[string]string{"requirements.txt": looseRequirements, "poetry.lock": poetryLock},
			wantLock:    "poetry.lock",
			wantScanned: true,
			wantHigh:    1,
		},
		{
			name:        "loose requirements.txt alone is not scanned",
			files:       map[string]string{"requirements.txt": looseRequirements},
			wantLock:    "requirements.txt",
			wantScanned: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, body := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
					t.Fatalf("write %s: %v", name, err)
				}
			}
			srv := vulnscantest.NewServer(t,
				map[int][]string{0: {"GHSA-HIGH-001"}},
				map[string]string{"GHSA-HIGH-001": "HIGH"},
			)
			scanner := &vulnscan.Scanner{BaseURL: srv.URL, HTTPClient: srv.Client()}
			detected := types.DetectedProject{Ecosystems: map[string]bool{ecosystem.NamePython: true}}

			ecos := buildEcosystemStatuses(detected, dir, scanner)
			if len(ecos) != 1 {
				t.Fatalf("got %d ecosystems, want 1", len(ecos))
			}
			got := ecos[0]
			if got.LockFile != tt.wantLock {
				t.Errorf("LockFile = %q, want %q", got.LockFile, tt.wantLock)
			}
			if got.Scanned != tt.wantScanned {
				t.Errorf("Scanned = %t, want %t", got.Scanned, tt.wantScanned)
			}
			if got.ScanError {
				t.Error("ScanError = true, want false")
			}
			if got.VulnCounts.High != tt.wantHigh {
				t.Errorf("VulnCounts.High = %d, want %d", got.VulnCounts.High, tt.wantHigh)
			}
		})
	}
}
