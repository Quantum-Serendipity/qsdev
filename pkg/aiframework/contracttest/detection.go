package contracttest

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

func TestDetectionAdapter(t *testing.T, adapter aiframework.DetectionAdapter, fixtures ContractFixtures) {
	t.Helper()

	t.Run("FrameworkIDNonEmpty", func(t *testing.T) {
		if id := adapter.FrameworkID(); id == "" {
			t.Error("FrameworkID() returned empty string")
		}
	})

	t.Run("MarkersNonEmpty", func(t *testing.T) {
		markers := adapter.Markers()
		if len(markers) == 0 {
			t.Error("Markers() returned empty slice")
		}
	})

	t.Run("DetectPresent", func(t *testing.T) {
		if fixtures.PresentRoot == "" {
			t.Skip("PresentRoot not provided")
		}
		det, err := adapter.Detect(fixtures.PresentRoot)
		if err != nil {
			t.Fatalf("Detect() error: %v", err)
		}
		if !det.Detected {
			t.Error("Detect() returned Detected=false for PresentRoot")
		}
		if len(det.Evidence) == 0 {
			t.Error("Detect() returned no evidence for detected framework")
		}
	})

	t.Run("DetectAbsent", func(t *testing.T) {
		if fixtures.AbsentRoot == "" {
			t.Skip("AbsentRoot not provided")
		}
		det, err := adapter.Detect(fixtures.AbsentRoot)
		if err != nil {
			t.Fatalf("Detect() error: %v", err)
		}
		if det.Detected {
			t.Error("Detect() returned Detected=true for AbsentRoot")
		}
	})
}
