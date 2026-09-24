package devinit

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/outdated"
)

// TestRunOutdated_CorruptAnswersReported verifies a broken answers file is
// reported instead of silently falling back to probing every ecosystem.
func TestRunOutdated_CorruptAnswersReported(t *testing.T) {
	dir := t.TempDir()
	writeAnswersFile(t, dir, "languages: [unterminated\n")
	t.Chdir(dir)
	t.Setenv("PATH", t.TempDir())

	err := runOutdated(outdatedCmd(), outdated.OutdatedOptions{})
	if err == nil || !strings.Contains(err.Error(), "loading saved answers") {
		t.Errorf("runOutdated() error = %v, want a saved-answers load error", err)
	}
}
