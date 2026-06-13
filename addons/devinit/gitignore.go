package devinit

import "github.com/Quantum-Serendipity/qsdev/pkg/fileutil"

const gitignoreSectionComment = "# qsdev local configuration"

// EnsureGitignoreEntry ensures that entry appears in the .gitignore file at
// projectRoot. It delegates to the canonical implementation in pkg/fileutil.
func EnsureGitignoreEntry(projectRoot, entry string) error {
	return fileutil.EnsureGitignoreEntry(projectRoot, entry)
}
