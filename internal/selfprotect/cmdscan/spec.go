package cmdscan

import (
	"path"
	"slices"
	"strconv"
	"strings"
)

// CommandSpec describes the invocations of one CLI subcommand that a rule
// matches, as words after the program name: the subcommand path, then flags
// and positional arguments in any order.
type CommandSpec struct {
	// Path holds, per subcommand level below the program, the words that
	// name it (its name and aliases).
	Path [][]string
	// ReadOnly holds the spellings of the flag (e.g. "--dry-run") that make
	// an invocation read-only; such an invocation never matches.
	ReadOnly []string
	// Flags match an invocation that sets any of them to its Value.
	Flags []FlagCond
	// Args returns the positional arguments that match an invocation; it is
	// called only once the path matched. Nil matches none.
	Args func() []string
}

// FlagCond is a boolean flag set to Value, written as any of Spellings
// ("--force", "-f"). A bare spelling sets it to true; "--force=false" sets
// it to false.
type FlagCond struct {
	Spellings []string
	Value     bool
}

// Always reports whether every invocation of the path matches: no flag or
// argument condition narrows it.
func (s CommandSpec) Always() bool {
	return len(s.Flags) == 0 && s.Args == nil
}

// Matches reports whether argv, the words after the program name, invoke the
// spec's subcommand in a matching way. Flags before and between path words
// are skipped. A flag value that does not parse as a boolean counts as
// matching, so a spelling the CLI would reject never hides a match.
func (s CommandSpec) Matches(argv []string) bool {
	rest, ok := s.afterPath(argv)
	if !ok || slices.ContainsFunc(rest, func(w string) bool { return flagValue(w, s.ReadOnly, true) }) {
		return false
	}
	if s.Always() {
		return true
	}
	for _, w := range rest {
		for _, f := range s.Flags {
			if flagValue(w, f.Spellings, f.Value) {
				return true
			}
		}
	}
	if s.Args == nil {
		return false
	}
	args := s.Args()
	return slices.ContainsFunc(rest, func(w string) bool {
		return !strings.HasPrefix(w, "-") && slices.Contains(args, w)
	})
}

// afterPath returns the words of argv after the spec's path, and false when
// argv does not start with it.
func (s CommandSpec) afterPath(argv []string) ([]string, bool) {
	level := 0
	for i, w := range argv {
		if level == len(s.Path) {
			return argv[i:], true
		}
		if strings.HasPrefix(w, "-") {
			continue
		}
		if !slices.Contains(s.Path[level], w) {
			return nil, false
		}
		level++
	}
	return nil, level == len(s.Path)
}

// flagValue reports whether word sets the flag spelled as one of spellings to
// want.
func flagValue(word string, spellings []string, want bool) bool {
	name, value, hasValue := strings.Cut(word, "=")
	if !slices.Contains(spellings, name) {
		return false
	}
	if !hasValue {
		return want
	}
	got, err := strconv.ParseBool(value)
	return err != nil || got == want
}

// InvokedSpecs returns the specs that command invokes the program app with,
// judged on the raw text: quotes are dropped and the text is split at
// whitespace and shell operators, so quoting, wrappers (env, sh -c, eval) and
// compound commands do not hide an invocation. A backslash is read both as a
// shell escape (q\sdev) and as a Windows path separator. An invocation
// mentioned in an argument (`echo "qsdev teardown"`) matches too; the
// caller's rule errs towards the human gate. A program word built from an
// expansion is not resolved.
func InvokedSpecs(command, app string, specs []CommandSpec) []CommandSpec {
	var hit []CommandSpec
	for _, args := range appInvocations(command, app) {
		for _, s := range specs {
			if s.Matches(args) {
				hit = append(hit, s)
			}
		}
	}
	return hit
}

// InvokesProgram reports whether command may run the program app, judged on
// the raw text the way InvokedSpecs judges it.
func InvokesProgram(command, app string) bool {
	return len(appInvocations(command, app)) > 0
}

// appInvocations returns, for each word of command that names the program
// app, the words that follow it (see InvokedSpecs for how the text is split).
func appInvocations(command, app string) [][]string {
	raw := strings.FieldsFunc(strings.NewReplacer(`'`, "", `"`, "").Replace(command), isWordBreak)
	words := make([]string, len(raw))
	for i, w := range raw {
		words[i] = strings.ReplaceAll(w, `\`, "")
	}
	var out [][]string
	for i, w := range raw {
		// Case-folded: Windows and macOS resolve QSDEV to qsdev.
		if strings.EqualFold(ProgramName(w), app) || strings.EqualFold(ProgramName(words[i]), app) {
			out = append(out, words[i+1:])
		}
	}
	return out
}

// isWordBreak reports whether r separates words for InvokedSpecs.
func isWordBreak(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', ';', '|', '&', '(', ')', '<', '>', '`', '{', '}':
		return true
	}
	return false
}

// ProgramName returns the program a command word names: its base name (after
// a / or \ separator) without a Windows executable suffix, so
// `C:\bin\qsdev.exe` and `/usr/bin/qsdev` both name qsdev. Compare the result
// case-insensitively: Windows and macOS file systems resolve QSDEV to qsdev.
func ProgramName(word string) string {
	base := path.Base(strings.ReplaceAll(word, `\`, "/"))
	if ext := path.Ext(base); strings.EqualFold(ext, ".exe") {
		base = strings.TrimSuffix(base, ext)
	}
	return base
}
