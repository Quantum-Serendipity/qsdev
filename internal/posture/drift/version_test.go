package drift

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestDetectVersionDrift_SameVersion(t *testing.T) {
	currentVersion := version.Info().Version

	genState := types.GeneratedState{
		QsdevVersion: currentVersion,
	}

	cat := detectVersionDrift(genState)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings when versions match, got %d: %+v", len(cat.Findings), cat.Findings)
	}
}

func TestDetectVersionDrift_DifferentVersion(t *testing.T) {
	genState := types.GeneratedState{
		QsdevVersion: "v0.0.1-old",
	}

	cat := detectVersionDrift(genState)

	if len(cat.Findings) != 1 {
		t.Fatalf("expected 1 finding for version mismatch, got %d", len(cat.Findings))
	}

	f := cat.Findings[0]
	if f.Severity != Info {
		t.Errorf("expected severity %q, got %q", Info, f.Severity)
	}
	if f.Expected != "v0.0.1-old" {
		t.Errorf("expected Expected=%q, got %q", "v0.0.1-old", f.Expected)
	}
	currentVersion := version.Info().Version
	if f.Actual != currentVersion {
		t.Errorf("expected Actual=%q, got %q", currentVersion, f.Actual)
	}
}

func TestDetectVersionDrift_EmptyStateVersion(t *testing.T) {
	genState := types.GeneratedState{
		QsdevVersion: "",
	}

	cat := detectVersionDrift(genState)

	if len(cat.Findings) != 1 {
		t.Fatalf("expected 1 finding for empty version, got %d", len(cat.Findings))
	}

	f := cat.Findings[0]
	if f.Severity != Info {
		t.Errorf("expected severity %q, got %q", Info, f.Severity)
	}
	if f.Subject != "qsdev version" {
		t.Errorf("expected subject %q, got %q", "qsdev version", f.Subject)
	}
	if f.Expected != "" {
		t.Errorf("expected empty Expected for pre-tracking state, got %q", f.Expected)
	}
}
