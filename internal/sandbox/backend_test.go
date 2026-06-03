package sandbox

import (
	"context"
	"testing"
)

// Compile-time interface compliance checks.
var _ SandboxBackend = (*UnsandboxedBackend)(nil)

func TestUnsandboxedBackend_Name(t *testing.T) {
	t.Parallel()
	b := &UnsandboxedBackend{}
	if got := b.Name(); got != "unsandboxed" {
		t.Errorf("Name() = %q, want %q", got, "unsandboxed")
	}
}

func TestUnsandboxedBackend_Available(t *testing.T) {
	t.Parallel()
	b := &UnsandboxedBackend{}
	if err := b.Available(); err != nil {
		t.Errorf("Available() = %v, want nil", err)
	}
}

func TestUnsandboxedBackend_Tier(t *testing.T) {
	t.Parallel()
	b := &UnsandboxedBackend{}
	if got := b.Tier(); got != TierUnsandboxed {
		t.Errorf("Tier() = %v, want %v", got, TierUnsandboxed)
	}
}

func TestUnsandboxedBackend_RunHook_EmptyCommand(t *testing.T) {
	t.Parallel()
	b := &UnsandboxedBackend{}
	result, err := b.RunHook(context.Background(), &SandboxConfig{})
	if err != nil {
		t.Fatalf("RunHook() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Tier != TierUnsandboxed {
		t.Errorf("Tier = %v, want %v", result.Tier, TierUnsandboxed)
	}
}

func TestUnsandboxedBackend_RunHook_Echo(t *testing.T) {
	t.Parallel()
	b := &UnsandboxedBackend{}
	result, err := b.RunHook(context.Background(), &SandboxConfig{
		HookCommand: []string{"echo", "hello sandbox"},
	})
	if err != nil {
		t.Fatalf("RunHook() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if got := string(result.Stdout); got != "hello sandbox\n" {
		t.Errorf("Stdout = %q, want %q", got, "hello sandbox\n")
	}
	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

func TestUnsandboxedBackend_RunHook_NonZeroExit(t *testing.T) {
	t.Parallel()
	b := &UnsandboxedBackend{}
	result, err := b.RunHook(context.Background(), &SandboxConfig{
		HookCommand: []string{"sh", "-c", "exit 42"},
	})
	if err != nil {
		t.Fatalf("RunHook() error = %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}
}

func TestUnsandboxedBackend_RunHook_InvalidCommand(t *testing.T) {
	t.Parallel()
	b := &UnsandboxedBackend{}
	_, err := b.RunHook(context.Background(), &SandboxConfig{
		HookCommand: []string{"/nonexistent/binary"},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}
}
