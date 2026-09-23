package rules

import (
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// This file holds the Bash analysis shared by every self-protection rule. The
// rules used to gate on substring tests of the raw command text, which a shell
// spelling defeats: `.cl""aude`, `.cl\aude` and `.c*e` never contain `.claude`,
// and `cd .claude && echo x > settings.json` never names the file it clobbers.
// The helpers below reason about the words the shell actually passes (quotes
// and escapes removed by cmdscan), expand braces, evaluate globs, and resolve
// relative paths against the working directory each command runs in.

// scannedCommand is a parsed simple command annotated with the working
// directory it runs in, tracked through any cd/pushd/popd earlier on the line.
type scannedCommand struct {
	cmdscan.Command
	// cwd is the effective working directory. It is relative (or "") when the
	// session directory is not known, in which case relative paths are
	// resolved lexically against it.
	cwd string
	// inProtectedDir reports that the effective working directory is a
	// protected directory (or inside one), so a relative path may name a
	// protected file.
	inProtectedDir bool
	// cwdUnknown reports that an earlier cd target could not be resolved (it
	// used an expansion, a glob, `-`, the directory stack, or CDPATH), so
	// relative paths may land anywhere.
	cwdUnknown bool
}

// scannedCommands returns the parsed commands annotated with their effective
// working directory, starting from ctx.CWD. The result is memoized on ctx. A
// non-nil error means the command was unparseable; callers must fail closed.
func (ctx *EvalContext) scannedCommands() ([]scannedCommand, error) {
	cmds, err := ctx.ParsedCommands()
	if ctx.scannedDone || err != nil {
		return ctx.scanned, err
	}
	ctx.scannedDone = true

	st := dirState{
		cwd:         ctx.CWD,
		inProtected: ctx.CWD != "" && isProtectedDir(ctx.CWD),
		cdpath:      strings.Contains(looseText(ctx.Command), "CDPATH"),
	}
	out := make([]scannedCommand, 0, len(cmds))
	for _, c := range cmds {
		sc := scannedCommand{Command: c, cwd: st.cwd, inProtectedDir: st.inProtected, cwdUnknown: st.unknown}
		out = append(out, sc)
		st.apply(sc)
	}
	ctx.scanned = out
	return out, nil
}

// dirState tracks the working directory across the commands of one line.
type dirState struct {
	cwd         string
	inProtected bool
	unknown     bool
	// cdpath reports that the line sets CDPATH, which makes cd search other
	// directories for a bare relative target (`CDPATH=.claude cd hooks`).
	cdpath bool
}

// apply updates the directory state for a cd/pushd/popd command. A subshell's
// cd is treated as persisting, which errs toward treating later relative paths
// as protected (fail closed).
func (st *dirState) apply(sc scannedCommand) {
	switch sc.Name {
	case "cd", "pushd":
	case "popd":
		// The destination is the directory stack, which is not modelled.
		// Leaving a protected directory cannot be proven, so keep that flag.
		st.cwd, st.unknown = "", true
		return
	default:
		return
	}
	operands := nonFlagArgs(sc.Args)
	target := "~"
	if len(operands) > 0 {
		target = operands[0]
	}
	if sc.HasExpansion || target == "-" || hasGlobMeta(target) ||
		(sc.Name == "pushd" && (len(operands) == 0 || strings.HasPrefix(target, "+"))) ||
		(st.cdpath && usesCDPATH(target)) {
		// The destination is not statically known. Keep any protected
		// directory the literal part names (`cd "$HOME/.claude"`, `cd .c*e`).
		st.cwd, st.unknown = "", true
		st.inProtected = st.inProtected || lexicalProtected(target) || globProtected(target)
		return
	}
	expanded := expandTilde(target)
	switch {
	case isRooted(expanded):
		st.cwd = filepath.Clean(expanded)
	case st.unknown:
		// A relative move from an unknown directory stays unknown; leaving a
		// protected directory cannot be proven, so only ever add the flag.
		st.inProtected = st.inProtected || isProtectedDir(expanded)
		return
	default:
		st.cwd = filepath.Join(st.cwd, expanded)
	}
	st.unknown = false
	st.inProtected = isProtectedDir(st.cwd)
}

// usesCDPATH reports whether cd would look a relative target up through
// CDPATH: bash does so unless it starts with `/`, `.` or `..`.
func usesCDPATH(target string) bool {
	return isRelativePath(target) && target != "." && target != ".." &&
		!strings.HasPrefix(target, "./") && !strings.HasPrefix(target, "../")
}

// expandTilde expands a leading ~ (or ~/) to the home directory, leaving the
// path unchanged when it has none or home cannot be resolved.
func expandTilde(p string) string {
	if expanded, err := canon.ExpandTilde(p); err == nil {
		return expanded
	}
	return p
}

// protectedDirNames are the protected directory names that can appear in any
// location (home config or project checkout). They mirror the dot-directory
// entries of canon's protected substring table and are used to decide whether
// a glob segment such as `.c*e` can expand to one of them.
var protectedDirNames = []string{".claude", ".qsdev", ".gdev"}

// protectedSystemDirs are the absolute protected directories that are not
// reached through a protected dot-directory, split into path segments.
var protectedSystemDirs = [][]string{{"etc", "gdev"}, {"etc", "claude-code"}}

// isProtectedDir reports whether dir is a protected directory or lies inside
// one, i.e. whether a file directly inside it is protected. It is precise
// (not a substring test), so a directory that merely has a protected
// directory as an ancestor, such as a Claude Code worktree under
// .claude/worktrees/, is not itself treated as protected.
func isProtectedDir(dir string) bool {
	if canon.ContainsProtectedPath(filepath.Base(dir)) {
		return true // a .claude/.qsdev/.gdev directory itself
	}
	protected, category := canon.IsProtected(filepath.Join(dir, "x"))
	return protected && category != "mcp-config"
}

// resolvedProtected reports whether the resolved path p is protected: a
// protected file, a protected directory, or an entry directly inside a
// protected directory (so `settings.json` under a .claude directory counts).
// An absolute path is also checked after canonicalization, so a symlink that
// leads into a protected directory (`/tmp/c/settings.json` with /tmp/c ->
// ~/.claude) is recognised.
func resolvedProtected(p string) bool {
	p = filepath.Clean(p)
	if pathProtected(p) {
		return true
	}
	if !isRooted(p) {
		return false
	}
	canonical, err := canon.Canonicalize(p)
	return err == nil && canonical != p && pathProtected(canonical)
}

// pathProtected is resolvedProtected for a path taken as written.
func pathProtected(p string) bool {
	if protected, category := canon.IsProtected(p); protected && category != "mcp-config" {
		return true
	}
	return isProtectedDir(p) || isProtectedDir(filepath.Dir(p))
}

// lexicalProtected reports whether the literal text of p names a protected
// path, either as written or after tilde expansion and cleaning (which
// collapses `//` and `/./` spellings such as `/etc//claude-code`).
func lexicalProtected(p string) bool {
	if p == "" {
		return false
	}
	if canon.ContainsProtectedPath(p) {
		return true
	}
	return canon.ContainsProtectedPath(filepath.ToSlash(filepath.Clean(expandTilde(p))))
}

// hasGlobMeta reports whether s contains a shell glob metacharacter.
func hasGlobMeta(s string) bool { return strings.ContainsAny(s, "*?[") }

// shellSegMatch matches one path segment against a glob segment with the
// shell's default rules: a leading '.' in name must be matched by a literal
// leading '.' in the pattern. A malformed pattern counts as a match (fail
// closed).
func shellSegMatch(pattern, name string) bool {
	if strings.HasPrefix(name, ".") && !strings.HasPrefix(pattern, ".") {
		return false
	}
	ok, err := path.Match(pattern, name)
	return ok || err != nil
}

// globProtected reports whether a glob word can expand to a protected path: a
// glob segment that matches a protected directory name (`.c*e`, `.clau?e`), or
// an absolute pattern whose leading segments match a protected system
// directory (`/etc/g?ev/...`).
func globProtected(p string) bool {
	if !hasGlobMeta(p) {
		return false
	}
	cleaned := filepath.ToSlash(filepath.Clean(expandTilde(p)))
	segs := strings.Split(cleaned, "/")
	for _, seg := range segs {
		if !hasGlobMeta(seg) {
			continue
		}
		for _, name := range protectedDirNames {
			if shellSegMatch(seg, name) {
				return true
			}
		}
	}
	if !strings.HasPrefix(cleaned, "/") {
		return false
	}
	segs = segs[1:]
	for _, dir := range protectedSystemDirs {
		if len(segs) < len(dir) {
			continue
		}
		matched := true
		for i, want := range dir {
			if !shellSegMatch(segs[i], want) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// maxBraceVariants caps brace expansion so a hostile word cannot blow up the
// analysis; exceeding it is treated as naming a protected path (fail closed).
const maxBraceVariants = 64

// expandBraces performs bash brace expansion on s. Comma lists expand to each
// alternative; a sequence (`{a..z}`) is replaced with `*` so the glob check
// covers every value it can produce. ok is false when the expansion exceeds
// maxBraceVariants.
func expandBraces(s string) ([]string, bool) {
	open, closing, alts, seq := findBraceGroup(s)
	if open < 0 {
		return []string{s}, true
	}
	prefix, suffix := s[:open], s[closing+1:]
	if seq {
		return expandBraces(prefix + "*" + suffix)
	}
	var out []string
	for _, alt := range alts {
		vs, ok := expandBraces(prefix + alt + suffix)
		if !ok {
			return nil, false
		}
		out = append(out, vs...)
		if len(out) > maxBraceVariants {
			return nil, false
		}
	}
	return out, true
}

// findBraceGroup finds the first expandable brace group in s: a `{...}` with a
// top-level comma (returned as alts) or a `..` sequence (seq). It returns
// open == -1 when s has none.
func findBraceGroup(s string) (open, closing int, alts []string, seq bool) {
	for i := 0; i < len(s); i++ {
		if s[i] != '{' {
			continue
		}
		depth, start := 0, i+1
		var parts []string
		for j := i; j < len(s); j++ {
			switch s[j] {
			case '{':
				depth++
			case ',':
				if depth == 1 {
					parts = append(parts, s[start:j])
					start = j + 1
				}
			case '}':
				depth--
				if depth != 0 {
					continue
				}
				if len(parts) > 0 {
					return i, j, append(parts, s[start:j]), false
				}
				if strings.Contains(s[i+1:j], "..") {
					return i, j, nil, true
				}
				j = len(s) // not expandable; look for a later group
			}
		}
	}
	return -1, -1, nil, false
}

// isRelativePath reports whether p is a relative path that the shell resolves
// against the working directory (not absolute, not ~-anchored).
func isRelativePath(p string) bool {
	return p != "" && !isRooted(p) && !strings.HasPrefix(p, "~")
}

// resolveWord returns word p as the path it names from sc's working directory,
// and false when p is relative but that directory is unknown.
func resolveWord(sc scannedCommand, p string) (string, bool) {
	p = expandTilde(p)
	if isRooted(p) {
		return filepath.Clean(p), true
	}
	if sc.cwdUnknown {
		return "", false
	}
	return filepath.Join(sc.cwd, p), true
}

// refersProtected reports whether the word p, used by command sc, can name a
// protected path: literally, through brace or glob expansion, or by resolving
// a relative path against the directory sc runs in.
func refersProtected(sc scannedCommand, p string) bool {
	variants, ok := expandBraces(p)
	if !ok {
		return true
	}
	for _, v := range variants {
		if lexicalProtected(v) || globProtected(v) {
			return true
		}
		if v == "" || hasGlobMeta(v) {
			continue
		}
		resolved, known := resolveWord(sc, v)
		if !known {
			if sc.inProtectedDir {
				return true // relative to a protected directory we cannot leave
			}
			continue
		}
		if resolvedProtected(resolved) {
			return true
		}
	}
	return false
}

// argRefersProtected is refersProtected for one argument word, which may be an
// option: the value of `--opt=value` is checked as a path, and a short option
// cluster (`-t.claude/hooks`, `-C.claude`) is checked for an attached value.
func argRefersProtected(sc scannedCommand, arg string) bool {
	if !isFlag(arg) || arg == "-" {
		return refersProtected(sc, arg)
	}
	if strings.HasPrefix(arg, "--") {
		if i := strings.IndexByte(arg, '='); i >= 0 {
			return refersProtected(sc, arg[i+1:])
		}
		return false
	}
	for i := 2; i < len(arg); i++ {
		if lexicalProtected(arg[i:]) || globProtected(arg[i:]) {
			return true
		}
	}
	return false
}

func anyArgRefersProtected(sc scannedCommand, args []string) bool {
	for _, a := range args {
		if argRefersProtected(sc, a) {
			return true
		}
	}
	return false
}

func anyRefersProtected(sc scannedCommand, paths []string) bool {
	for _, p := range paths {
		if refersProtected(sc, p) {
			return true
		}
	}
	return false
}

// looseText strips quote and escape characters from a raw command, so a
// quote-split spelling (`.cl""aude`) can still be recognised when the command
// cannot be parsed at all.
func looseText(s string) string {
	return strings.NewReplacer(`"`, "", `'`, "", `\`, "").Replace(s)
}

// lineMentionsProtected reports whether a Bash command references a protected
// path anywhere: in the raw text, in a quote-split spelling, in any parsed
// argument or redirect (after quote removal, brace and glob expansion, and
// cwd resolution), through a cd into a protected directory, or through a find
// whose patterns select protected files. It is the trigger every Bash rule
// shares; the per-rule analysis then decides whether the reference is a
// mutation. The result is memoized on ctx.
func lineMentionsProtected(ctx *EvalContext) bool {
	if !ctx.mentionsDone {
		ctx.mentions, ctx.mentionsDone = scanMentionsProtected(ctx), true
	}
	return ctx.mentions
}

func scanMentionsProtected(ctx *EvalContext) bool {
	if containsProtectedPathStr(ctx.Command) || containsProtectedPathStr(looseText(ctx.Command)) {
		return true
	}
	scs, err := ctx.scannedCommands()
	if err != nil {
		return false
	}
	for _, sc := range scs {
		if sc.inProtectedDir ||
			anyArgRefersProtected(sc, sc.Args) ||
			anyRefersProtected(sc, sc.WriteRedirects) ||
			anyRefersProtected(sc, sc.ReadRedirects) {
			return true
		}
		if sc.Name == "find" && findMutatesProtected(sc) {
			return true
		}
	}
	return false
}

// bashMutatesProtected is the shared protected-mutation predicate behind the
// Bash self-protection rules: the line references a protected path and either
// cannot be parsed (fail closed) or one of its commands mutates one (see
// protectedMutation). The result is memoized on ctx.
func bashMutatesProtected(ctx *EvalContext) bool {
	if ctx.mutatesDone {
		return ctx.mutates
	}
	ctx.mutatesDone = true
	if !lineMentionsProtected(ctx) {
		return false
	}
	scs, err := ctx.scannedCommands()
	ctx.mutates = err != nil || protectedMutation(scs)
	return ctx.mutates
}

// hasVerb reports whether the command invokes one of verbs, judged from the
// raw text (re), its quote-stripped form, or the parsed command words (which
// catch a verb split by empty quotes, such as r""m).
func hasVerb(ctx *EvalContext, re *regexp.Regexp, verbs map[string]bool) bool {
	if re.MatchString(ctx.Command) || re.MatchString(looseText(ctx.Command)) {
		return true
	}
	scs, _ := ctx.scannedCommands()
	for _, sc := range scs {
		if verbs[sc.Name] {
			return true
		}
	}
	return false
}

// dirChangeVerbs change the working directory; scannedCommands models them, so
// they are not mutations themselves.
var dirChangeVerbs = map[string]bool{"cd": true, "pushd": true, "popd": true}

// isMutating reports whether command sc may change files through its
// arguments: anything but a proven read-only command (cmdscan.IsSafeReadCommand,
// which models git/sort/rg by their arguments), a directory change, or a
// nameless (redirect-only) command.
func isMutating(sc scannedCommand) bool {
	return sc.Name != "" && !cmdscan.IsSafeReadCommand(sc.Command) && !dirChangeVerbs[sc.Name]
}

// patternOptions are tar's and rsync's options whose value is a file-name
// pattern, or a file of patterns, that selects operands rather than naming a
// target (the --exclude/--include family). rsync also spells --filter as -f;
// for tar, -f names the archive, so it is not listed here. Only these two
// verbs are modelled: for any other command an option value is opaque and
// stays a potential target.
var patternOptions = map[string]bool{
	"--exclude": true, "--include": true, "--filter": true,
	"--exclude-from": true, "--include-from": true,
}

func isPatternOption(verb, opt string) bool {
	switch verb {
	case "tar":
		return patternOptions[opt]
	case "rsync":
		return patternOptions[opt] || opt == "-f"
	default:
		return false
	}
}

// operandArgs returns sc's arguments without pattern options and their values,
// so `--exclude=.claude` is not mistaken for an operand.
func operandArgs(sc scannedCommand) []string {
	out := make([]string, 0, len(sc.Args))
	for i := 0; i < len(sc.Args); i++ {
		a := sc.Args[i]
		if eq := strings.IndexByte(a, '='); eq > 0 && strings.HasPrefix(a, "--") && isPatternOption(sc.Name, a[:eq]) {
			continue
		}
		if isPatternOption(sc.Name, a) {
			i++ // skip the separate pattern value
			continue
		}
		out = append(out, a)
	}
	return out
}

// copySourcesAndDest returns the source operands and the destination of a
// cp/rsync command. The destination is cp's -t/--target-directory directory
// when given, else the last positional operand (a lone operand is treated as
// the destination).
func copySourcesAndDest(sc scannedCommand) (sources []string, dest string) {
	var positionals []string
	targetDir := ""
	endOfOpts := false
	args := operandArgs(sc)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case endOfOpts || !isFlag(a) || a == "-":
			positionals = append(positionals, a)
		case a == "--":
			endOfOpts = true
		case sc.Name != "cp":
			// rsync has no target-directory option (its -t preserves times).
		case a == "-t" || a == "--target-directory":
			if i+1 < len(args) {
				targetDir = args[i+1]
				i++
			}
		case strings.HasPrefix(a, "--target-directory="):
			targetDir = strings.TrimPrefix(a, "--target-directory=")
		case strings.HasPrefix(a, "-t") && !strings.HasPrefix(a, "--"):
			targetDir = a[2:]
		}
	}
	switch {
	case targetDir != "":
		return positionals, targetDir
	case len(positionals) == 0:
		return nil, ""
	default:
		return positionals[:len(positionals)-1], positionals[len(positionals)-1]
	}
}

// isNonDestructiveCopy reports whether sc is a copy that leaves its sources in
// place and creates no alias of them (cp without -l/-s, or rsync without
// --remove-source-files), so only its destination is written. A linking cp
// makes the destination a writable alias of the source, so every operand is
// a mutation target.
func isNonDestructiveCopy(sc scannedCommand) bool {
	switch sc.Name {
	case "cp":
		return !cpCreatesLinks(sc.Args)
	case "rsync":
		return !slices.Contains(sc.Args, "--remove-source-files")
	default:
		return false
	}
}

// cpCreatesLinks reports whether cp's options make it link instead of copy:
// --link, --symbolic-link, or -l/-s in a short option cluster (letters after
// an option that takes a value, -t or -S, are that value).
func cpCreatesLinks(args []string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if a == "--link" || a == "--symbolic-link" {
			return true
		}
		if !isFlag(a) || strings.HasPrefix(a, "--") {
			continue
		}
		for _, c := range a[1:] {
			if c == 't' || c == 'S' {
				break
			}
			if c == 'l' || c == 's' {
				return true
			}
		}
	}
	return false
}

// readsProgramFromStdin reports whether an interpreter takes its program from
// standard input (no script file and no inline program, but a here-document,
// here-string, or input redirect), which hides the program from argv.
func readsProgramFromStdin(sc scannedCommand) bool {
	if !interpreterVerbs[sc.Name] || len(sc.ReadRedirects) == 0 {
		return false
	}
	for _, a := range sc.Args {
		if !isFlag(a) {
			return false // a script file or an inline program (sh -c '...')
		}
	}
	return true
}

// interpreterVerbs run a script file given as their first operand. The script
// is read and executed, not written, so running a protected hook script
// (`bash .claude/hooks/test.sh`) is not a mutation of it. Any further
// arguments, and every argument of an inline program (`sh -c`, `perl -pi -e`),
// stay opaque.
var interpreterVerbs = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "dash": true, "ksh": true,
	"python": true, "python3": true, "node": true, "perl": true, "ruby": true,
}

// mutationTargets returns the argument words that command sc may create,
// modify, or remove. A read-only verb writes nothing through its arguments (its
// write redirects are checked separately). A non-destructive copy (cp, rsync)
// writes only its destination, so reading a protected source for an in-repo
// backup is not a mutation of it. Every other command — including interpreters
// and unknown binaries, whose argument semantics are opaque — is assumed to
// mutate any path it is given, except an interpreter's script file and the
// patterns of --exclude-style options.
func mutationTargets(sc scannedCommand) []string {
	if !isMutating(sc) {
		return nil
	}
	args := operandArgs(sc)
	if isNonDestructiveCopy(sc) {
		var targets []string
		for _, a := range args {
			if isFlag(a) && a != "-" {
				targets = append(targets, a) // e.g. --backup-dir=, -t<dir>
			}
		}
		if _, dest := copySourcesAndDest(sc); dest != "" {
			targets = append(targets, dest)
		}
		return targets
	}
	if interpreterVerbs[sc.Name] && len(args) > 0 && !isFlag(args[0]) {
		return args[1:]
	}
	return args
}

// relativeWriteTarget reports whether sc writes (by argument or redirect) to a
// relative path, which cannot be placed when the working directory is unknown.
func relativeWriteTarget(sc scannedCommand) bool {
	for _, a := range mutationTargets(sc) {
		if !isFlag(a) && isRelativePath(a) {
			return true
		}
	}
	for _, r := range sc.WriteRedirects {
		if isRelativePath(r) {
			return true
		}
	}
	return false
}

// protectedMutation reports whether a Bash command mutates a protected path
// through any command, not just a fixed list of verbs: an in-place editor
// (sed -i, perl -pi), an interpreter (python -c), install, patch, chmod, a
// wrapper (sh -c, sudo) or an unknown binary. It is evaluated only once the
// line references a protected path (lineMentionsProtected). A command then
// counts as a mutation when:
//
//   - any of its mutation targets or write redirects names a protected path;
//   - it is not read-only and uses an expansion (a variable can carry the
//     protected path named elsewhere on the line: `V=.claude/x; foo "$V"`);
//   - it is not read-only and consumes a protected path on its input: from an
//     upstream pipeline stage that read one (`echo .claude/x | xargs rm`) or
//     an input redirect (`xargs rm < .claude/list`);
//   - it is an interpreter reading its program from a here-document,
//     here-string, or input redirect, whose text argv does not show;
//   - it writes a relative path after a cd whose target was not resolvable;
//   - it is a find whose actions can delete or rewrite protected files;
//   - it sets variables (a prefix or bare assignment): PATH, LD_PRELOAD or
//     GIT_EXTERNAL_DIFF can make a read-only command word run arbitrary code,
//     and the assigned value is not analysed.
//
// Non-destructive copies (cp, rsync) and the other copy-family verbs are
// exempt from the input rule: their exfiltration semantics are modelled by
// protectedExfil. The parse error case is handled by the caller (fail closed).
func protectedMutation(scs []scannedCommand) bool {
	upstreamProtected := make(map[int]bool)
	for _, sc := range scs {
		if len(sc.Assigns) > 0 {
			return true
		}
		if anyArgRefersProtected(sc, mutationTargets(sc)) || anyRefersProtected(sc, sc.WriteRedirects) {
			return true
		}
		if len(sc.WriteRedirects) > 0 && sc.HasExpansion {
			return true // the redirect target may be the expanded word
		}
		if isMutating(sc) {
			if sc.HasExpansion || readsProgramFromStdin(sc) {
				return true
			}
			protectedInput := (sc.Pipeline != 0 && upstreamProtected[sc.Pipeline]) || anyRefersProtected(sc, sc.ReadRedirects)
			if protectedInput && !copyVerbs[sc.Name] {
				return true
			}
			if sc.Name == "find" && findMutatesProtected(sc) {
				return true
			}
		}
		if sc.cwdUnknown && relativeWriteTarget(sc) {
			return true
		}
		if sc.Pipeline != 0 && (anyArgRefersProtected(sc, sc.Args) || anyRefersProtected(sc, sc.ReadRedirects)) {
			upstreamProtected[sc.Pipeline] = true
		}
	}
	return false
}

// readsProtected reports whether sc reads a protected path: through an input
// redirect, a copy source, or (for any other command) an argument.
func readsProtected(sc scannedCommand) bool {
	if anyRefersProtected(sc, sc.ReadRedirects) {
		return true
	}
	if isNonDestructiveCopy(sc) {
		sources, _ := copySourcesAndDest(sc)
		return anyRefersProtected(sc, sources)
	}
	return anyArgRefersProtected(sc, sc.Args)
}

// writesOutsideRepo reports whether the write target p of command sc lies
// outside the repository rooted at root. An unknown root, a filesystem root
// (a session in `/` would otherwise make every sink, /dev/tcp included, look
// in-repo), a remote (host:path) spec, or a relative target after an
// unresolvable cd all count as outside, so exfiltration checks stay
// conservative (fail closed). A leading ~ is expanded first, so `~/x` is seen
// as outside a project repo.
func writesOutsideRepo(sc scannedCommand, p, root string) bool {
	if root == "" || isFilesystemRoot(root) || looksRemote(p) {
		return true
	}
	resolved, known := resolveWord(sc, p)
	if !known {
		return true
	}
	if !isRooted(resolved) && sc.cwd == "" {
		resolved = filepath.Join(root, resolved)
	}
	return !withinDir(resolved, root)
}

// isFilesystemRoot reports whether dir is a filesystem root (`/`, `C:\`).
func isFilesystemRoot(dir string) bool {
	clean := filepath.Clean(dir)
	return filepath.Dir(clean) == clean
}

// withinDir reports whether p is dir itself or lies below it (lexically). A
// filesystem root (`/`, `C:\`) already ends in a separator after cleaning, so
// one is only appended when missing; otherwise nothing would lie below `/`.
func withinDir(p, dir string) bool {
	p, dir = filepath.Clean(p), filepath.Clean(dir)
	if p == dir {
		return true
	}
	if !strings.HasSuffix(dir, string(filepath.Separator)) {
		dir += string(filepath.Separator)
	}
	return strings.HasPrefix(p, dir)
}

// protectedExfil reports whether a command sends a protected read out of the
// repository rooted at root: a write redirect or copy destination outside the
// repo fed by a protected read (`cat .claude/settings.json > /tmp/x`,
// `cp .claude/settings.json ~/x`), or a pipeline that streams a protected read
// into an exfiltration sink. An in-repo backup stays allowed.
func protectedExfil(scs []scannedCommand, root string) bool {
	for _, sc := range scs {
		if !readsProtected(sc) {
			continue
		}
		for _, target := range sc.WriteRedirects {
			if writesOutsideRepo(sc, target, root) {
				return true
			}
		}
		if isNonDestructiveCopy(sc) {
			if _, dest := copySourcesAndDest(sc); dest != "" && writesOutsideRepo(sc, dest, root) {
				return true
			}
		}
	}
	return pipelineExfil(scs, root)
}

// pipelineExfil reports whether any pipeline reads a protected source in an
// upstream stage and pipes it to an exfiltration sink downstream. File-writing
// sinks (tee/dd/cp/…) count only when their target is outside the repo, so an
// in-repo backup like `cat <protected> | tee settings.bak` stays allowed and a
// pure filter like `cat <protected> | grep foo` (a read) is not exfil.
func pipelineExfil(scs []scannedCommand, root string) bool {
	groups := make(map[int][]scannedCommand)
	for _, sc := range scs {
		if sc.Pipeline != 0 {
			groups[sc.Pipeline] = append(groups[sc.Pipeline], sc)
		}
	}
	for _, stages := range groups {
		upstream := false
		for _, sc := range stages {
			if readsProtected(sc) {
				upstream = true
			}
			if !upstream || !exfilSinks[sc.Name] {
				continue
			}
			if !fileSinkVerbs[sc.Name] {
				return true // network or opaque interpreter sink
			}
			for _, p := range append(nonFlagArgs(sc.Args), sc.WriteRedirects...) {
				if writesOutsideRepo(sc, p, root) {
					return true
				}
			}
		}
	}
	return false
}

// protectedArea is a protected subtree with its own rule (hook scripts, the
// audit trail, the security binary). frag is the subtree as path segments
// (".claude/hooks"); ancestors are the directories above it whose recursive
// mutation reaches it (".claude").
type protectedArea struct {
	frag      string
	ancestors []string
}

var (
	hooksArea  = protectedArea{frag: ".claude/hooks", ancestors: []string{".claude"}}
	auditArea  = protectedArea{frag: ".qsdev/audit", ancestors: []string{".qsdev"}}
	binaryArea = protectedArea{frag: ".qsdev/bin", ancestors: []string{".qsdev"}}
)

// mentionedIn reports whether text names the area or an ancestor, used as the
// fail-closed trigger when a command cannot be parsed.
func (a protectedArea) mentionedIn(text string) bool {
	return a.touches(text) || a.touches(looseText(text))
}

// touches reports whether s contains the area as a whole path-segment sequence
// (at or below it), or names one of its ancestor directories itself (not a
// sibling below the ancestor). Segment boundaries include whitespace and
// quotes, so a path inside a wrapper's script (`sh -c 'chmod 0 .claude/hooks'`)
// is found too.
func (a protectedArea) touches(s string) bool {
	s = filepath.ToSlash(s)
	if segmentIndex(s, a.frag, true) {
		return true
	}
	for _, anc := range a.ancestors {
		if segmentIndex(s, anc, false) {
			return true
		}
	}
	return false
}

// segmentIndex reports whether seq occurs in s starting at a segment boundary.
// With below, the occurrence may continue into a deeper path (`seq/...`);
// without it, only a trailing '/' may follow before the next boundary, so the
// occurrence names the directory itself.
func segmentIndex(s, seq string, below bool) bool {
	for from := 0; ; {
		i := strings.Index(s[from:], seq)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(seq)
		if start == 0 || isSegmentBoundary(s[start-1]) {
			rest := s[end:]
			rest = strings.TrimPrefix(rest, "/")
			if below && end < len(s) && s[end] == '/' {
				return true
			}
			if rest == "" || isSegmentBoundary(rest[0]) && rest[0] != '/' {
				return true
			}
		}
		from = start + 1
	}
}

// isSegmentBoundary reports whether b ends a path segment: a separator or a
// byte that cannot appear in a path token (whitespace, quotes, shell
// metacharacters).
func isSegmentBoundary(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return false
	case b == '.', b == '-', b == '_':
		return false
	default:
		return true
	}
}

// wordTouchesArea reports whether word p (used by sc) names the area, as
// written, after cleaning, or resolved against sc's working directory.
func wordTouchesArea(sc scannedCommand, p string, a protectedArea) bool {
	variants, ok := expandBraces(p)
	if !ok {
		return true
	}
	for _, v := range variants {
		forms := []string{v, filepath.Clean(expandTilde(v))}
		if isRelativePath(v) {
			if resolved, known := resolveWord(sc, v); known {
				forms = append(forms, resolved)
			}
		}
		for _, f := range forms {
			if a.touches(f) {
				return true
			}
		}
	}
	return false
}

// mutatesArea reports whether any command writes to (by argument or redirect)
// the protected area or one of its ancestors. It backs the area-specific rules
// (hook scripts, audit trail, security binary) so each reports on the same
// mutation analysis as the generic config rule.
func mutatesArea(scs []scannedCommand, a protectedArea) bool {
	for _, sc := range scs {
		for _, w := range append(mutationTargets(sc), sc.WriteRedirects...) {
			if isFlag(w) && w != "-" {
				w = flagValue(w)
			}
			if w != "" && wordTouchesArea(sc, w, a) {
				return true
			}
		}
	}
	return false
}

// copiesFromArea reports whether a copy-family command (cp, mv, rsync, tar,
// dd, tee) names the area in any operand, i.e. relocates or duplicates its
// files.
func copiesFromArea(scs []scannedCommand, a protectedArea) bool {
	for _, sc := range scs {
		if !copyVerbs[sc.Name] {
			continue
		}
		for _, w := range sc.Args {
			if isFlag(w) && w != "-" {
				w = flagValue(w)
			}
			if w != "" && wordTouchesArea(sc, w, a) {
				return true
			}
		}
	}
	return false
}

// bashMutatesArea is the Bash check behind an area-specific rule: fail closed
// when an unparseable command mentions the area, else report whether a
// command mutates it.
func bashMutatesArea(ctx *EvalContext, a protectedArea) bool {
	scs, err := ctx.scannedCommands()
	if err != nil {
		return a.mentionedIn(ctx.Command)
	}
	return mutatesArea(scs, a)
}

// flagValue returns the path-like value embedded in an option word: the part
// after '=' of a long option, or the attached value of a short option cluster
// (everything after the first option letter).
func flagValue(a string) string {
	if strings.HasPrefix(a, "--") {
		if i := strings.IndexByte(a, '='); i >= 0 {
			return a[i+1:]
		}
		return ""
	}
	if len(a) > 2 {
		return a[2:]
	}
	return ""
}
