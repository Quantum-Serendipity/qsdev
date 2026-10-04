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

// scriptAttributeLines returns the .gitattributes lines that check out every
// script in files (generated content starting with "#!") with LF line
// endings, whatever core.autocrlf says. Git for Windows converts text files
// to CRLF by default, and a CR at the end of the interpreter line stops the
// script from starting on Linux and macOS, so a hook committed from one
// checkout must not be converted in another. Each pattern is anchored at the
// project root with glob characters escaped; a path holding whitespace, which
// a pattern cannot spell unquoted, is left out.
func scriptAttributeLines(files []types.GeneratedFile) []string {
	var lines []string
	for _, f := range files {
		if !bytes.HasPrefix(f.Content, []byte("#!")) {
			continue
		}
		p := path.Clean(strings.ReplaceAll(f.Path, `\`, "/"))
		if !filepath.IsLocal(filepath.FromSlash(p)) {
			continue // WriteFiles refuses it
		}
		if strings.ContainsAny(p, " \t\r\n") {
			slog.Warn("generated script path cannot be pinned to LF in .gitattributes", "path", f.Path)
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

// pinScriptLineEndings adds scriptAttributeLines(files) to the project's
// .gitattributes. A failure is logged, not fatal: the scripts are still
// written, and only a CRLF checkout elsewhere is left unprotected.
func pinScriptLineEndings(projectRoot string, files []types.GeneratedFile) {
	lines := scriptAttributeLines(files)
	if len(lines) == 0 {
		return
	}
	if err := fileutil.EnsureGitattributesLines(projectRoot, lines); err != nil {
		slog.Warn("could not pin generated scripts to LF line endings in .gitattributes", "error", err)
	}
}
