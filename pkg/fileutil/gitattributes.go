package fileutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// GitattributesSectionComment returns the comment line that heads the section
// of lines EnsureGitattributesLines adds, named after the active branding.
func GitattributesSectionComment() string {
	return "# " + branding.Get().AppName + " generated files"
}

// EnsureGitattributesLines ensures that every line in lines appears in the
// .gitattributes file at projectRoot, appending the missing ones (in order,
// under the section comment, added once) with a single atomic write confined
// to projectRoot. A line is present when an existing line has the same
// whitespace-separated fields, whatever its line ending. When nothing is
// missing the file is not written, and no file is created.
func EnsureGitattributesLines(projectRoot string, lines []string) error {
	path := filepath.Join(projectRoot, ".gitattributes")
	content, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	present := make(map[string]bool)
	for line := range strings.SplitSeq(string(content), "\n") {
		present[strings.Join(strings.Fields(line), " ")] = true
	}
	var missing []string
	for _, line := range lines {
		key := strings.Join(strings.Fields(line), " ")
		if key == "" || present[key] || slices.Contains(missing, line) {
			continue
		}
		missing = append(missing, line)
	}
	if len(missing) == 0 {
		return nil
	}

	var b strings.Builder
	b.Write(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		b.WriteByte('\n')
	}
	comment := GitattributesSectionComment()
	if !present[comment] {
		if len(content) > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(comment + "\n")
	}
	for _, line := range missing {
		b.WriteString(line + "\n")
	}
	if err := WriteFileAtomicInRoot(projectRoot, ".gitattributes", []byte(b.String()), ModeReadWrite); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
