package exitcode

import (
	"errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     int
		format   string
		args     []any
		wantCode int
		wantMsg  string
	}{
		{
			name:     "simple message",
			code:     1,
			format:   "something failed",
			wantCode: 1,
			wantMsg:  "something failed",
		},
		{
			name:     "formatted message",
			code:     42,
			format:   "exit code %d from %s",
			args:     []any{42, "sandbox"},
			wantCode: 42,
			wantMsg:  "exit code 42 from sandbox",
		},
		{
			name:     "zero exit code",
			code:     0,
			format:   "success",
			wantCode: 0,
			wantMsg:  "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := New(tt.code, tt.format, tt.args...)
			if e.ExitCode() != tt.wantCode {
				t.Errorf("ExitCode() = %d, want %d", e.ExitCode(), tt.wantCode)
			}
			if e.Error() != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", e.Error(), tt.wantMsg)
			}
		})
	}
}

func TestErrorSatisfiesErrorInterface(t *testing.T) {
	t.Parallel()

	var err error = New(1, "test")
	if err.Error() != "test" {
		t.Errorf("error interface: got %q, want %q", err.Error(), "test")
	}
}

func TestErrorUnwrapWithErrorsAs(t *testing.T) {
	t.Parallel()

	original := New(5, "inner error")
	wrapped := fmt.Errorf("outer: %w", original)

	var target *Error
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As did not find *exitcode.Error in wrapped chain")
	}
	if target.ExitCode() != 5 {
		t.Errorf("ExitCode() = %d, want 5", target.ExitCode())
	}
}

func TestExitCodeErrInterfaceCompat(t *testing.T) {
	t.Parallel()

	// Verify the type satisfies the gdev ExitCodeErr contract:
	// an interface with ExitCode() int method.
	type exitCodeErr interface {
		ExitCode() int
	}

	var e exitCodeErr = New(3, "test")
	if e.ExitCode() != 3 {
		t.Errorf("ExitCode() via interface = %d, want 3", e.ExitCode())
	}
}
