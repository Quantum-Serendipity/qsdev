package generate

import (
	"bytes"
	"log/slog"
	"path"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// lfAttributeLines returns the .gitattributes lines that check out every
// script in files (generated content starting with "#!") and every file
// with PinLF set with LF line endings, whatever core.autocrlf says. Git for
// Windows converts text files to CRLF by default, and a CR at the end of the
// interpreter line stops the script from starting on Linux and macOS (or, in
// a file a script parses, ends up in the values it reads), so a file
// committed from one checkout must not be converted in another. Each pattern
// is anchored at the project root with glob characters escaped; a path
// holding whitespace, which a pattern cannot spell unquoted, is left out.
func lfAttributeLines(files []types.GeneratedFile) []string {
	var lines []string
	for _, f := range files {
		if !f.PinLF && !bytes.HasPrefix(f.Content, []byte("#!")) {
			continue
		}
		p := path.Clean(strings.ReplaceAll(f.Path, `\`, "/"))
		if !filepath.IsLocal(filepath.FromSlash(p)) {
			continue // WriteFiles refuses it
		}
		if strings.ContainsAny(p, " \t\r\n") {
			slog.Warn("generated file path cannot be pinned to LF in .gitattributes", "path", f.Path)
			continue
		}
		lines = append(lines, "/"+escapeAttributePattern(p)+" text eol=lf")
	}
	return lines
}

// escapeAttributePattern escapes the characters a .gitattributes pattern
// reads as glob syntax, so the pattern matches p literally.
func escapeAttributePattern(p string) string {
	var b strings.Builder
	for _, r := range p {
		if strings.ContainsRune(`\*?[`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// pinLFLineEndings adds lfAttributeLines(files) to the project's
// .gitattributes. A failure is logged, not fatal: the files are still
// written, and only a CRLF checkout elsewhere is left unprotected.
func pinLFLineEndings(projectRoot string, files []types.GeneratedFile) {
	lines := lfAttributeLines(files)
	if len(lines) == 0 {
		return
	}
	if err := fileutil.EnsureGitattributesLines(projectRoot, lines); err != nil {
		slog.Warn("could not pin generated files to LF line endings in .gitattributes", "error", err)
	}
}
