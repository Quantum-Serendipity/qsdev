package devinit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/fileutil"
)

const gitignoreSectionComment = "# qsdev local configuration"

// EnsureGitignoreEntry ensures that entry appears in the .gitignore file at
// projectRoot. It is idempotent: if the entry already exists, it is a no-op.
// If .gitignore does not exist, it creates one. Uses atomic writes for
// crash safety.
func EnsureGitignoreEntry(projectRoot, entry string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	lines := strings.Split(string(content), "\n")

	// Check if entry already exists.
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return nil // Already present.
		}
	}

	// Build new content.
	var b strings.Builder
	b.Write(content)

	// Ensure there is a trailing newline before we append.
	if len(content) > 0 && content[len(content)-1] != '\n' {
		b.WriteByte('\n')
	}

	// Add section comment if not already present.
	existing := string(content)
	if !strings.Contains(existing, gitignoreSectionComment) {
		// Add a blank line separator if file is non-empty.
		if len(content) > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(gitignoreSectionComment)
		b.WriteByte('\n')
	}

	b.WriteString(entry)
	b.WriteByte('\n')

	return fileutil.WriteFileAtomic(gitignorePath, []byte(b.String()), 0o644)
}
