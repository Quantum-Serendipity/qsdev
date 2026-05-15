package claudecode_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
)

func TestInstallClaudeStep_NotNil(t *testing.T) {
	step := claudecode.InstallClaudeStep()
	if step == nil {
		t.Fatal("InstallClaudeStep() returned nil")
	}
}
