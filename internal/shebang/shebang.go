// Package shebang parses the "#!" interpreter line of a script the way the
// Linux kernel does, so every caller that has to know what a script runs
// (self-protection, the sandbox, qsdev check) reads it the same way.
package shebang

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// MaxLen is the number of leading bytes the kernel reads to parse an
// interpreter line (BINPRM_BUF_SIZE); a longer line is truncated.
const MaxLen = 256

// Line is a parsed interpreter line. Following Linux, the interpreter ends at
// the first blank (space or tab) and everything after it, trimmed of blanks,
// is one optional argument. Like the kernel, Parse does not treat '\r' as a
// blank: a CRLF line keeps it at the end of Interpreter or Arg. The zero Line
// means the file has no interpreter line.
type Line struct {
	Interpreter string
	Arg         string
}

// Read returns the interpreter line of the file at p, or the zero Line when
// the file does not start with "#!".
func Read(p string) (Line, error) {
	f, err := os.Open(p) //nolint:gosec // callers pass the script they need to inspect
	if err != nil {
		return Line{}, fmt.Errorf("opening %s: %w", p, err)
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, MaxLen)
	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return Line{}, fmt.Errorf("reading %s: %w", p, err)
	}
	return Parse(buf[:n]), nil
}

// Parse returns the interpreter line at the start of head, or the zero Line
// when head does not start with "#!".
func Parse(head []byte) Line {
	rest, ok := bytes.CutPrefix(head, []byte("#!"))
	if !ok {
		return Line{}
	}
	text, _, _ := bytes.Cut(rest, []byte("\n"))
	line := strings.Trim(string(text), blanks)
	interp, arg := line, ""
	if i := strings.IndexAny(line, blanks); i >= 0 {
		interp, arg = line[:i], line[i+1:]
	}
	return Line{Interpreter: interp, Arg: strings.Trim(arg, blanks)}
}

// blanks are the only characters the kernel treats as separators in an
// interpreter line.
const blanks = " \t"

// ViaEnv reports whether the interpreter is env, which looks the program up
// on PATH.
func (l Line) ViaEnv() bool {
	return l.Interpreter != "" && path.Base(filepath.ToSlash(l.Interpreter)) == "env"
}

// Program returns every platform's reading of the program the line names,
// leniently: for env, the first word of the argument that is neither an option
// nor a VAR=value assignment, split as BSD and macOS split it ("" when there
// is none); otherwise the interpreter. Line endings are dropped. It suits
// callers that must cover whatever the line may run (self-protection); use
// EnvProgram for what env actually runs on one platform.
func (l Line) Program() string {
	if !l.ViaEnv() {
		return strings.TrimSpace(l.Interpreter)
	}
	return firstProgramWord(strings.Fields(l.Arg))
}

// EnvProgram returns the program env looks up on PATH when the kernel of goos
// starts the script, and false when env runs no program (a bare option, an
// assignment) or fails to parse its argument. Linux passes the argument as
// one word, so a blank in it is part of the program name unless env splits
// it itself (-S, --split-string); macOS and the BSDs split it into words.
// Windows (Git Bash) tolerates a CRLF line ending and otherwise follows Linux.
// It returns false when the line does not run env.
func (l Line) EnvProgram(goos string) (string, bool) {
	if !l.ViaEnv() {
		return "", false
	}
	arg := l.Arg
	if goos == "windows" {
		arg = strings.TrimRight(arg, "\r")
	}
	var words []string
	value, splitByEnv := splitStringValue(arg)
	switch {
	case splitsInterpreterArg(goos):
		words = strings.FieldsFunc(arg, isBlank)
	case splitByEnv:
		words = strings.FieldsFunc(value, isBlank)
	case strings.HasPrefix(arg, "-"):
		// One word holding an option and more: env rejects it (exit 125), or
		// it is a lone option and env runs no program.
		return "", false
	default:
		words = []string{arg}
	}
	prog := firstProgramWord(words)
	return prog, prog != ""
}

// splitStringValue returns the value of a GNU env -S (--split-string)
// argument, which env splits into words itself.
func splitStringValue(arg string) (string, bool) {
	for _, prefix := range []string{"--split-string=", "-vS", "-S"} {
		if v, ok := strings.CutPrefix(arg, prefix); ok {
			return v, true
		}
	}
	return "", false
}

// splitsInterpreterArg reports whether the kernel of goos splits an
// interpreter line's argument into words.
func splitsInterpreterArg(goos string) bool {
	switch goos {
	case "darwin", "ios", "freebsd", "openbsd", "netbsd", "dragonfly":
		return true
	}
	return false
}

// firstProgramWord returns the first word that is neither an option nor a
// VAR=value assignment, or "".
func firstProgramWord(words []string) string {
	for _, w := range words {
		if !strings.HasPrefix(w, "-") && !strings.Contains(w, "=") {
			return w
		}
	}
	return ""
}

func isBlank(r rune) bool { return r == ' ' || r == '\t' }
