package rules

import (
	"path"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// ShellRewritesFile reports which of names (lower-cased base names) a shell
// tool's command (cmdscan.IsShellTool) writes, deletes, moves or edits in
// place, using the same mutation analysis as the protected-path rules. Those
// files are guarded by a before/after content check that only the Write and
// Edit tools go through, so a shell rewrite (`echo ignore-scripts=false >>
// .npmrc`, `rm .npmrc`) would bypass it. A command known to leave a file
// unchanged does not count (see writeTargets: `source .npmrc`, `sed -n 1p
// .npmrc`, the source of `ln -s`), and a target is also matched by the file
// it resolves to, so a write through a symlink to one of the files counts. A
// line that mentions one of the names but cannot be parsed, or whose writing
// command uses an expansion that could carry the name, counts as a rewrite
// (fail closed). A command whose name the line redefines as a function or
// alias, or that a script the line sources may define
// (cmdscan.Redefinitions: `alias gh='rm -f'; gh pr create .npmrc`, `.
// ./defs.sh; gh pr create .npmrc`), may write any of its words. PowerShell is judged in its own dialect (see
// psRewritesFile).
func ShellRewritesFile(ctx *EvalContext, names []string) (string, bool) {
	if !cmdscan.IsShellTool(ctx.ToolName) {
		return "", false
	}
	if isPowerShell(ctx) {
		return psRewritesFile(ctx, names)
	}
	mentioned := firstContained(names, strings.ToLower(looseText(ctx.Command)))
	scs, err := ctx.scannedCommands()
	if err != nil {
		return mentioned, mentioned != ""
	}
	cmds := make([]cmdscan.Command, len(scs))
	for i, sc := range scs {
		cmds[i] = sc.Command
	}
	redefined := cmdscan.Redefinitions(cmds)
	return rewrittenName(scs, mentioned, names, lineJudge{
		skip:      func(int) bool { return false },
		expands:   func(i int) bool { return scs[i].HasExpansion },
		redefined: func(i int) bool { return redefined(scs[i].Name) },
	})
}

// psRewritesFile is ShellRewritesFile for a PowerShell line. Its commands are
// read as the POSIX path analysis sees them (psScanned: `\` as `/`, $HOME as
// ~, Remove-Item and Move-Item as rm and mv), a read command
// (cmdscan.PSCommand.IsRead: Get-Content, type, git diff) is skipped, and a
// variable in a word or redirect, a `$(...)` subexpression, pipeline input
// (`gci .npmrc | ri`) or a parenthesised argument (`Set-Content ('.npmrc')
// x`) is an expansion that may carry a mentioned name, as the tokenizer does
// not attach a pipeline's or a group's value to the command that receives
// it (cmdscan.PSCommand.Piped, GroupArg); so is any word of a line that
// uses the ${...} variable notation, which can name a file
// (`${C:\p\.npmrc} = 'x'`), or calls a .NET member (`[IO.File]::...`), whose
// arguments are not modelled. A line that may define an alias or function
// (cmdscan.PSCommand.MayRedefine) may give any command name another meaning.
// The tokenizer cannot fail, so there is no unparseable fallback.
func psRewritesFile(ctx *EvalContext, names []string) (string, bool) {
	line := ctx.powerShell()
	mentioned := firstContained(names, strings.ToLower(line.norm), strings.ToLower(line.loose))
	scs := ctx.psScanned()
	opaque := strings.Contains(line.norm, "${") || strings.Contains(line.norm, "]::")
	redefined := slices.ContainsFunc(line.cmds, cmdscan.PSCommand.MayRedefine)
	return rewrittenName(scs, mentioned, names, lineJudge{
		skip: func(i int) bool { return line.cmds[i].IsRead() },
		expands: func(i int) bool {
			c := line.cmds[i]
			return opaque || scs[i].HasExpansion || c.Subexpression || c.Piped || c.GroupArg ||
				slices.ContainsFunc(c.WriteRedirects, func(r string) bool {
					_, expanded := psPathWord(r)
					return expanded
				})
		},
		redefined: func(int) bool { return redefined },
	})
}

// lineJudge is how rewrittenName judges the commands of one line, by index.
type lineJudge struct {
	// skip accepts a command that only reads.
	skip func(i int) bool
	// expands reports that a command uses an expansion that may carry a name
	// the line mentions.
	expands func(i int) bool
	// redefined reports that the line may define the command's name as an
	// alias or function, so it may run anything with any of its words.
	redefined func(i int) bool
}

// firstContained returns the first of names that occurs in any of texts, or
// "".
func firstContained(names []string, texts ...string) string {
	for _, n := range names {
		for _, text := range texts {
			if strings.Contains(text, n) {
				return n
			}
		}
	}
	return ""
}

// rewrittenName returns the entry of names that a command of scs writes (see
// writeTargets and guardedTarget), and true; or, when the line mentions one
// of them (mentioned) and a writing command uses an expansion (judge.expands)
// that could carry it, mentioned (fail closed). Commands judge.skip accepts
// are not judged, unless the line may redefine them (judge.redefined): such
// a command may write any of its words.
func rewrittenName(scs []scannedCommand, mentioned string, names []string, judge lineJudge) (string, bool) {
	for i, sc := range scs {
		redefined := judge.redefined(i)
		if !redefined && judge.skip(i) {
			continue
		}
		targets, writes := writeTargets(sc, scs[:i])
		if redefined {
			targets, writes = slices.Concat(sc.Args, sc.WriteRedirects), true
		}
		for _, t := range targets {
			if n := guardedTarget(sc, t, names); n != "" {
				return n, true
			}
			// An inline program (`sh -c 'echo x >> .npmrc'`, `python3 -c
			// "open('.npmrc','w')"`) or an operand=value word (`dd
			// of=.npmrc`) names the file inside the word.
			if runsScript(sc.Name) || (!isFlag(t) && strings.Contains(t, "=")) {
				if n := nameInWord(t, names); n != "" {
					return n, true
				}
			}
		}
		if mentioned != "" && (judge.expands(i) || redefined) && writes {
			return mentioned, true
		}
	}
	return "", false
}

// writeTargets returns the words naming the files sc may write, and whether
// it may write any file: its write redirects, plus the operands
// cmdscan.WrittenOperands models for it, or else every operand
// mutationTargets lists for a command that is not read-only. A command that
// only records files in git (recordsOnly) writes none of its operands.
// earlier are the commands of the line that run before sc, which may create
// a directory sc writes into (see mayBeDir).
func writeTargets(sc scannedCommand, earlier []scannedCommand) ([]string, bool) {
	if recordsOnly(sc) {
		return sc.WriteRedirects, len(sc.WriteRedirects) > 0
	}
	isDir := func(w string) bool { return mayBeDir(sc, w, earlier) }
	if ops, ok := cmdscan.WrittenOperands(sc.Command, isDir); ok {
		targets := slices.Concat(ops, sc.WriteRedirects)
		return targets, len(targets) > 0
	}
	return slices.Concat(mutationTargets(sc), sc.WriteRedirects), len(sc.WriteRedirects) > 0 || isMutating(sc)
}

// mayBeDir reports whether word, used by sc, may name a directory when sc
// runs: it names one now, where it resolves to is unknown, or a command that
// runs before sc on the line (earlier) may create or replace it (see
// mayCreateFiles). The last covers `mkdir -p pkg && ln -s ../x/.npmrc pkg`,
// where pkg does not exist yet when the hook runs.
func mayBeDir(sc scannedCommand, word string, earlier []scannedCommand) bool {
	if hasGlobMeta(word) || slices.ContainsFunc(earlier, mayCreateFiles) {
		return true
	}
	p, ok := resolveWord(sc, word)
	if !ok || !isRooted(p) {
		return true
	}
	fi, err := sc.fs.Stat(p)
	return err == nil && fi.IsDir()
}

// mayCreateFiles reports whether sc may create a file or directory anywhere:
// it writes through a redirect, or it is not a read-only command. Which paths
// such a command creates is not modelled (`tar -x`, `git clone`, a sourced
// script), so any of them may be one a later command of the line uses.
func mayCreateFiles(sc scannedCommand) bool {
	return len(sc.WriteRedirects) > 0 || isMutating(sc)
}

// guardedTarget returns the entry of names that the write target word
// names (see guardedName), or that the existing file it resolves to from
// sc's working directory is named, following symlinks, or "".
func guardedTarget(sc scannedCommand, word string, names []string) string {
	if n := guardedName(word, names); n != "" {
		return n
	}
	if isFlag(word) || hasGlobMeta(word) {
		return ""
	}
	p, ok := resolveWord(sc, word)
	if !ok || !isRooted(p) {
		return ""
	}
	resolved, err := sc.fs.EvalSymlinks(p)
	if err != nil || resolved == p {
		return ""
	}
	return guardedName(resolved, names)
}

// gitIndexSubcommands record working-tree files in git without changing them.
var gitIndexSubcommands = map[string]bool{"add": true, "stage": true, "commit": true}

// recordsOnly reports whether sc only records files in git (`git add .npmrc`,
// `git commit -m msg .npmrc`), which leaves their content as it is. A global
// option before the subcommand could redefine what runs, so it does not count.
func recordsOnly(sc scannedCommand) bool {
	return sc.Name == "git" && len(sc.Args) > 0 && gitIndexSubcommands[sc.Args[0]]
}

// nameInWord returns the entry of names that occurs in word as a whole path
// segment (so `.npmrc.bak` does not count), or "".
func nameInWord(word string, names []string) string {
	lower := strings.ToLower(word)
	for _, n := range names {
		if segmentIndex(lower, n, false) {
			return n
		}
	}
	return ""
}

// guardedName returns the entry of names that word can name by its base name,
// after brace and glob expansion, or "".
func guardedName(word string, names []string) string {
	variants, ok := expandBraces(word)
	if !ok {
		variants = []string{word}
	}
	for _, v := range variants {
		base := strings.ToLower(path.Base(strings.ReplaceAll(v, `\`, "/")))
		for _, n := range names {
			if base == n || (hasGlobMeta(base) && shellSegMatch(base, n)) {
				return n
			}
		}
	}
	return ""
}
