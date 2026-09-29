package fileutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// GitignoreSectionComment returns the comment line that heads the section of
// entries EnsureGitignoreEntry adds, named after the active branding.
func GitignoreSectionComment() string {
	return "# " + branding.Get().AppName + " local configuration"
}

// EnsureGitignoreEntry ensures that entry appears in the .gitignore file at
// projectRoot. It is idempotent: if the entry, or an equivalent spelling that
// differs only by a leading or trailing slash (such as /.qsdev.local.yaml or
// .devinit for .devinit/), already exists, it is a no-op. If .gitignore does
// not exist, it creates one. The section comment is added only once, even
// across multiple calls. Uses atomic writes confined to projectRoot.
func EnsureGitignoreEntry(projectRoot, entry string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading %s: %w", gitignorePath, err)
	}

	want := normalizeGitignorePattern(entry)
	for line := range strings.SplitSeq(string(content), "\n") {
		if normalizeGitignorePattern(line) == want {
			return nil
		}
	}

	var b strings.Builder
	b.Write(content)

	if len(content) > 0 && content[len(content)-1] != '\n' {
		b.WriteByte('\n')
	}

	comment := GitignoreSectionComment()
	if !strings.Contains(string(content), comment) {
		if len(content) > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(comment)
		b.WriteByte('\n')
	}

	b.WriteString(entry)
	b.WriteByte('\n')

	if err := WriteFileAtomicInRoot(projectRoot, ".gitignore", []byte(b.String()), ModeReadWrite); err != nil {
		return fmt.Errorf("writing %s: %w", gitignorePath, err)
	}
	return nil
}

// normalizeGitignorePattern reduces a .gitignore line to a comparable form by
// trimming whitespace and a leading or trailing slash, so root-anchored and
// directory-only spellings of the same path compare equal.
func normalizeGitignorePattern(line string) string {
	p := strings.TrimSpace(line)
	p = strings.TrimPrefix(p, "/")
	return strings.TrimSuffix(p, "/")
}
