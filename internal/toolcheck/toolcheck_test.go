package toolcheck_test

import (
	"context"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
)

func TestDetect_ExistingTool(t *testing.T) {
	// "bash" is guaranteed to be present on any dev system.
	info := toolcheck.Detect(context.Background(), "bash", "--version")

	if !info.Found {
		t.Fatal("expected bash to be found")
	}
	if info.Path == "" {
		t.Error("expected non-empty path for bash")
	}
	if info.Version == "" {
		t.Error("expected non-empty version string for bash")
	}
}

func TestDetect_NonexistentTool(t *testing.T) {
	info := toolcheck.Detect(context.Background(), "this-tool-does-not-exist-xyz-999", "--version")

	if info.Found {
		t.Error("expected Found to be false for nonexistent tool")
	}
	if info.Path != "" {
		t.Errorf("expected empty path, got %q", info.Path)
	}
	if info.Version != "" {
		t.Errorf("expected empty version, got %q", info.Version)
	}
}
