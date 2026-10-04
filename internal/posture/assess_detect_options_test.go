package posture

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/sysinfo"
)

// TestAssess_UsesDetectOptions checks project detection runs with the
// caller's host probes, so a caller can keep Assess off the real machine.
func TestAssess_UsesDetectOptions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".qsdev.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var osProbes atomic.Int32
	opts := AssessOptions{DetectOptions: []detect.Option{
		detect.WithOSDetector(func() *sysinfo.OSInfo {
			osProbes.Add(1)
			return &sysinfo.OSInfo{Family: "debian"}
		}),
	}}
	if _, err := Assess(root, opts); err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if got := osProbes.Load(); got != 1 {
		t.Errorf("OS detector ran %d times, want 1", got)
	}
}
