package claudecode_test

import (
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/claudecode"
)

func TestInstallClaudeStep_NotNil(t *testing.T) {
	step := claudecode.InstallClaudeStep()
	if step == nil {
		t.Fatal("InstallClaudeStep() returned nil")
	}
}
