package bwrap

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

func TestPrepareLandlockFlags_Linter(t *testing.T) {
	t.Parallel()
	cfg := &sandbox.SandboxConfig{
		ProjectDir:   "/home/user/project",
		HookCategory: sandbox.CategoryLinter,
	}

	flags := PrepareLandlockFlags(cfg)
	if sandbox.LLRestrictBin() == "" {
		if flags != nil {
			t.Error("expected nil flags when ll-restrict unavailable")
		}
		t.Skip("ll-restrict not available, skipping flag verification")
	}

	assertContains(t, flags, "--ro")
	assertContains(t, flags, "--deny-net")
}

func TestPrepareLandlockFlags_Formatter(t *testing.T) {
	t.Parallel()
	cfg := &sandbox.SandboxConfig{
		ProjectDir:   "/home/user/project",
		HookCategory: sandbox.CategoryFormatter,
	}

	flags := PrepareLandlockFlags(cfg)
	if sandbox.LLRestrictBin() == "" {
		t.Skip("ll-restrict not available")
	}

	assertContains(t, flags, "--rw")
	assertContains(t, flags, "--deny-net")
}

func TestPrepareLandlockFlags_NetworkLinter(t *testing.T) {
	t.Parallel()
	cfg := &sandbox.SandboxConfig{
		ProjectDir:   "/home/user/project",
		HookCategory: sandbox.CategoryNetworkLinter,
	}

	flags := PrepareLandlockFlags(cfg)
	if sandbox.LLRestrictBin() == "" {
		t.Skip("ll-restrict not available")
	}

	assertNotContains(t, flags, "--deny-net")
}

func TestInjectLandlock_Unavailable(t *testing.T) {
	t.Parallel()
	if sandbox.LLRestrictBin() != "" {
		t.Skip("ll-restrict is available, cannot test unavailable path")
	}

	original := []string{"/usr/bin/hook", "--arg1"}
	result := InjectLandlock(original, &sandbox.SandboxConfig{
		HookCategory: sandbox.CategoryLinter,
	})

	if len(result) != len(original) {
		t.Errorf("expected command unchanged, got %v", result)
	}
}

func TestExpandDenyPaths(t *testing.T) {
	t.Parallel()
	paths := expandDenyPaths()
	if len(paths) == 0 {
		t.Error("expected non-empty deny paths")
	}
	for _, p := range paths {
		if p == "" {
			t.Error("found empty deny path")
		}
		if p[0] == '~' {
			t.Errorf("tilde not expanded in path: %s", p)
		}
	}
}

func assertContains(t *testing.T, s []string, want string) {
	t.Helper()
	if !slices.Contains(s, want) {
		t.Errorf("slice does not contain %q: %v", want, s)
	}
}

func assertNotContains(t *testing.T, s []string, unwant string) {
	t.Helper()
	if slices.Contains(s, unwant) {
		t.Errorf("slice should not contain %q: %v", unwant, s)
	}
}
