package bwrap

import (
	"testing"
)

// TestIsLandlockSetupFailure pins how a helper failure is told apart from the
// hook's own exit status: both a reserved ll-restrict exit code AND the
// helper's diagnostic prefix are required, so a hook that merely exits 121 (or
// merely prints "ll-restrict: ") keeps its own verdict.
func TestIsLandlockSetupFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   int
		stderr string
		want   bool
	}{
		{name: "landlock unsupported", code: llExitUnsupported, stderr: "ll-restrict: Landlock unavailable (kernel too old or disabled)\n", want: true},
		{name: "setup failed", code: llExitSetupFailed, stderr: "ll-restrict: landlock_restrict_self: Operation not permitted\n", want: true},
		{name: "exec failed", code: llExitExecFailed, stderr: "ll-restrict: execvp: No such file or directory\n", want: true},
		{name: "usage error", code: llExitUsage, stderr: "ll-restrict: unknown option: --bogus\n", want: true},
		{name: "diagnostic after bwrap warning", code: llExitSetupFailed, stderr: "bwrap: warning\nll-restrict: prctl failed\n", want: true},
		{name: "hook blocks with exit 2", code: 2, stderr: "blocked by policy\n", want: false},
		{name: "hook exits with a reserved code", code: llExitUnsupported, stderr: "some hook output\n", want: false},
		{name: "hook prints the prefix but exits 1", code: 1, stderr: "ll-restrict: spoofed\n", want: false},
		{name: "success", code: 0, stderr: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isLandlockSetupFailure(tt.code, []byte(tt.stderr)); got != tt.want {
				t.Errorf("isLandlockSetupFailure(%d, %q) = %v, want %v", tt.code, tt.stderr, got, tt.want)
			}
		})
	}
}
