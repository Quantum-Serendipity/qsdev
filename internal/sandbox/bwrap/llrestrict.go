package bwrap

import (
	"bytes"
	"slices"
)

// Exit codes of the ll-restrict helper (nix/ll-restrict/ll-restrict.c). They
// sit in a range hooks do not normally use, so a helper failure (the hook never
// ran) can be told apart from the hook's own exit status.
const (
	llExitUsage       = 120
	llExitUnsupported = 121
	llExitSetupFailed = 122
	llExitExecFailed  = 123
)

// llDiagnosticPrefix starts every diagnostic ll-restrict writes to stderr.
const llDiagnosticPrefix = "ll-restrict: "

// isLandlockSetupFailure reports whether a sandboxed run ended because
// ll-restrict failed before exec'ing the hook, rather than the hook exiting.
// Both signals are required: one of the helper's reserved exit codes AND its
// diagnostic at the start of stderr.
func isLandlockSetupFailure(exitCode int, stderrHead []byte) bool {
	helperCodes := []int{llExitUsage, llExitUnsupported, llExitSetupFailed, llExitExecFailed}
	if !slices.Contains(helperCodes, exitCode) {
		return false
	}
	return bytes.HasPrefix(stderrHead, []byte(llDiagnosticPrefix)) ||
		bytes.Contains(stderrHead, []byte("\n"+llDiagnosticPrefix))
}
