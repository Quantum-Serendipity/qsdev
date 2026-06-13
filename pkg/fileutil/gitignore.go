package fileutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const gitignoreSectionComment = "# qsdev local configuration"

// EnsureGitignoreEntry ensures that entry appears in the .gitignore file at
// projectRoot. It is idempotent: if the entry already exists, it is a no-op.
// If .gitignore does not exist, it creates one. The section comment is added
// only once, even across multiple calls. Uses atomic writes for crash safety.
func EnsureGitignoreEntry(projectRoot, entry string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}

	var b strings.Builder
	b.Write(content)

	if len(content) > 0 && content[len(content)-1] != '\n' {
		b.WriteByte('\n')
	}

	existing := string(content)
	if !strings.Contains(existing, gitignoreSectionComment) {
		if len(content) > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(gitignoreSectionComment)
		b.WriteByte('\n')
	}

	b.WriteString(entry)
	b.WriteByte('\n')

	return WriteFileAtomic(gitignorePath, []byte(b.String()), ModeReadWrite)
}
