package devinit

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/exitcode"
)

// TestExitError_IsSharedExitcodeError locks ExitError to the shared
// internal/exitcode.Error, so exit-code handling (errors.As, the gdev
// ExitCodeErr contract) has one implementation.
func TestExitError_IsSharedExitcodeError(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("status: %w", &ExitError{Code: exitNotInitialized})

	var target *exitcode.Error
	if !errors.As(err, &target) {
		t.Fatalf("errors.As(%v, *exitcode.Error) = false", err)
	}
	if target.ExitCode() != exitNotInitialized {
		t.Errorf("ExitCode() = %d, want %d", target.ExitCode(), exitNotInitialized)
	}
}
