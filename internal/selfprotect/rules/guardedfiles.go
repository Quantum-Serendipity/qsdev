package rules

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
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
		skip:       func(int) bool { return false },
		expands:    func(i int) bool { return scs[i].HasExpansion },
		redefined:  func(i int) bool { return redefined(scs[i].Name) },
		shellGlobs: true,
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
	// shellGlobs reports that the shell expands a glob in a target word on
	// the entries that exist (see globScope). Otherwise a glob is judged by
	// the names it matches alone.
	shellGlobs bool
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
// a command may write any of its words, and its globs are judged by name.
func rewrittenName(scs []scannedCommand, mentioned string, names []string, judge lineJudge) (string, bool) {
	created := false // an earlier command of the line may create files
	homeAt := firstHomeAssign(scs)
	for i, sc := range scs {
		earlierCreated := created
		created = created || mayCreateFiles(sc)
		redefined := judge.redefined(i)
		if !redefined && judge.skip(i) {
			continue
		}
		if n := assignedName(sc, names); n != "" {
			return n, true
		}
		targets, writes := writeTargets(scs, i)
		if redefined {
			targets, writes = slices.Concat(sc.Args, sc.WriteRedirects), true
		}
		scope := byName
		if judge.shellGlobs && !redefined {
			scope = globScope(sc, earlierCreated || !cwdCertain(scs, i, homeAt), homeAt < i)
		}
		for _, t := range targets {
			if n := guardedTarget(sc, t, names, scope(t)); n != "" {
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

// assignedName returns the entry of names that a variable sc sets hands to
// the programs it runs, or "": a write-target pin (isWriteTargetPinVar)
// names a file git writes even for a read (`GIT_INDEX_FILE=devenv.yaml git
// read-tree HEAD` replaces devenv.yaml with an index), so its value is a
// write target; and a variable whose value a program runs as a command line
// (runsProgramVar: `GIT_EDITOR='cp /dev/null devenv.yaml #' git commit
// --amend`) names the file inside that value, as an inline program does. It
// reads the prefix, bare and declared assignments of sc (`export
// GIT_INDEX_FILE=...`) and the NAME=value operands of env, so the git that
// reads the variable may run later on the line; a value set without export
// is judged too, as the name may already be exported.
func assignedName(sc scannedCommand, names []string) string {
	for _, a := range slices.Concat(sc.AssignTexts, envOperandAssigns(sc)) {
		name, value, _ := strings.Cut(a, "=")
		switch {
		case isWriteTargetPinVar(name):
			if n := guardedTarget(sc, value, names, nil); n != "" {
				return n
			}
		case runsProgramVar(name):
			if n := nameInWord(value, names); n != "" {
				return n
			}
		}
	}
	return ""
}

// envOperandAssigns returns the NAME=value operands of an env command (`env
// GIT_INDEX_FILE=x git status`), which set NAME for the program env runs, or
// nil for any other command.
func envOperandAssigns(sc scannedCommand) []string {
	if path.Base(sc.Name) != "env" {
		return nil
	}
	var assigns []string
	for _, a := range sc.Args {
		if name, _, ok := strings.Cut(a, "="); ok && name != "" && !isFlag(a) {
			assigns = append(assigns, a)
		}
	}
	return assigns
}

// writeTargets returns the words naming the files scs[i] may write, and
// whether it may write any file: its write redirects, plus the operands
// cmdscan.WrittenOperands models for it, or else every operand
// mutationTargets lists for a command that is not read-only. A command that
// only records files in git (recordsOnly), and a find that only lists on a
// line where nothing acts on what it prints (findListsOnly: `find . -name
// '*.yaml'`), write none of their operands; any other find keeps them all, so
// its name and path patterns count wherever they match. The commands of the
// line that run before scs[i] may create a directory it writes into (see
// mayBeDir).
func writeTargets(scs []scannedCommand, i int) ([]string, bool) {
	sc, earlier := scs[i], scs[:i]
	if recordsOnly(sc) || findListsOnly(scs, i) {
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

// findLists reports whether sc runs find, directly or through a wrapper,
// without an action that deletes, writes or runs a command on what it finds.
func findLists(sc scannedCommand) bool {
	args, ok := programInvocation(append([]string{sc.Name}, sc.Args...), programNamed("find"))
	if !ok {
		return false
	}
	_, exprs := splitFindArgs(args)
	return !findHasMutatingAction(exprs)
}

// findListsOnly reports whether scs[i] is a find that only lists (findLists)
// on a line where nothing may act on what it prints: every other command is
// read-only and writes no file (see mayCreateFiles), or is itself such a
// find. Otherwise its output may feed a command that deletes or rewrites the
// files it names, through a pipe (`find . -name devenv.yaml | xargs rm`), a
// loop or a substitution (`rm $(find ...)`), or a file a later command reads
// (`find ... > list; xargs rm < list`), and its patterns count as targets.
func findListsOnly(scs []scannedCommand, i int) bool {
	if !findLists(scs[i]) {
		return false
	}
	for j, o := range scs {
		if j != i && mayCreateFiles(o) && (!findLists(o) || len(o.WriteRedirects) > 0) {
			return false
		}
	}
	return true
}

// cwdCertain reports whether scs[i] surely runs in the working directory the
// line's model gives it (scannedCommand.cwd). The model takes every cd as
// succeeding and persisting, which over-approximates where a relative path
// may land but cannot say where a glob is expanded. The directory is certain
// when no earlier command changes it, or when every command from the first
// that does through scs[i] runs in the line's own shell outside a pipeline,
// each after that first one runs only when the commands before it in one &&
// list succeed, and every directory change among them goes where the model
// says (certainCd: `cd ./src && rm -rf [!x]*`): a failed cd then stops the
// rest. A cd in a subshell, pipeline or background job (`(cd src) && rm ...`,
// `cd src & rm ...`) leaves the shell where it was, and a command after `;` or
// `||` runs where a failed cd left it (`cd nosuch; rm [d]evenv.yaml`).
// homeAt is the index of the first command that may set HOME (see
// firstHomeAssign).
func cwdCertain(scs []scannedCommand, i, homeAt int) bool {
	first := slices.IndexFunc(scs[:i], func(sc scannedCommand) bool { return dirChangeVerbs[sc.Name] })
	if first < 0 {
		return true
	}
	for j, sc := range scs[first : i+1] {
		if sc.Subshell || sc.Pipeline != 0 || (j > 0 && sc.Guard != cmdscan.GuardedByAnd) ||
			(dirChangeVerbs[sc.Name] && !certainCd(sc, homeAt < first+j)) {
			return false
		}
	}
	return true
}

// certainCd reports whether the cd or pushd sc, when it succeeds, lands in the
// directory the line's model gives it: it takes exactly one operand and no
// option, that operand is neither a directory-stack form (`+N`, `-N`, `-`) nor
// an expansion, it is not looked up through CDPATH or zsh's cdpath (it is
// absolute, `.`, `..`, starts with `./` or `../`, or is a resolved `~`; see
// usesCDPATH), and it names an existing directory. Otherwise the shell may go
// elsewhere: zsh's `cd old new` substitutes inside $PWD, `cd +1` picks from
// the directory stack, `cd src` may follow CDPATH, and a missing directory
// leaves the model guessing. popd always fails this, and so does a `~` or
// `~/...` operand after the line may have changed HOME (homeAssigned, see
// firstHomeAssign), which the model resolves with the hook's own HOME.
func certainCd(sc scannedCommand, homeAssigned bool) bool {
	if sc.Name == "popd" || sc.HasExpansion || len(sc.Args) != 1 {
		return false
	}
	target := sc.Args[0]
	if isFlag(target) || strings.HasPrefix(target, "+") || hasGlobMeta(target) ||
		unresolvedTilde(target) || (homeAssigned && homeTilde(target)) || usesCDPATH(target) {
		return false
	}
	p, ok := resolveWord(sc, target)
	if !ok || !isRooted(p) {
		return false
	}
	fi, err := sc.fs.Stat(p)
	return err == nil && fi.IsDir()
}

// byName is the glob scope (see globScope) that judges every glob of a
// command by the names it matches alone.
func byName(string) func(variant, name string) bool { return nil }

// globScope returns, for each target word of sc, how guardedName judges a
// glob in it that matches a guarded name: where the shell expands it
// (globHolds), or, when that is not known, by the name alone (nil). The
// shell expands a glob on the entries that exist in its directory, and
// leaves one that matches nothing literal (bash) or fails (zsh), so `rm -rf
// src/[!x]*` cannot remove a devenv.yaml that src does not hold. Judged by
// name are: a word built from an expansion, whose directory is not known; a
// glob find or git matches itself below any directory (a find pattern, a
// pathspec), quoted or not; a glob an interpreter or shell runs inside its
// script; and every glob of a command whose working directory is not certain
// (uncertain: an earlier command of the line may create files, see
// mayCreateFiles, as in `tar -xf a.tar; rm src/[d]evenv.yaml`, or may have
// left the directory elsewhere than the line's model says, see cwdCertain).
// So is a glob in a `~/...` word after the line may have changed HOME
// (homeAssigned, see firstHomeAssign): `HOME=$PWD; rm ~/[d]evenv.yaml`.
func globScope(sc scannedCommand, uncertain, homeAssigned bool) func(word string) func(variant, name string) bool {
	_, matchesItself := programInvocation(append([]string{sc.Name}, sc.Args...), programNamed("find", "git"))
	if uncertain || matchesItself || runsScript(sc.Name) {
		return byName
	}
	expanded := make(map[string]bool, len(sc.ExpandedArgs))
	for _, w := range sc.ExpandedArgs {
		expanded[w] = true
	}
	if sc.HasExpansion {
		for _, w := range sc.WriteRedirects {
			expanded[w] = true
		}
	}
	holds := func(variant, name string) bool { return globHolds(sc, variant, name) }
	return func(word string) func(variant, name string) bool {
		if expanded[word] || (homeAssigned && homeTilde(word)) {
			return nil
		}
		return holds
	}
}

// firstHomeAssign returns the index of the first command of scs that may set
// or clear HOME, in any form cmdscan.Command.Assigns lists (`HOME=..;`,
// `export HOME=..`, `read HOME`, unset) or through a variable whose name it
// computes (see assignsEnv), or len(scs). The shell expands a `~` in a later
// command with that value, which the line's model does not know (expandTilde
// reads the hook's own HOME). A prefix assignment of a command does not
// count for that command's own words: the shell expands them before it
// applies it (`HOME=.. cd ~`).
func firstHomeAssign(scs []scannedCommand) int {
	setsHome := assignsEnv([]string{"HOME"})
	if i := slices.IndexFunc(scs, func(sc scannedCommand) bool {
		return sc.AssignsDynamic || setsHome(sc.Command)
	}); i >= 0 {
		return i
	}
	return len(scs)
}

// homeTilde reports whether word starts with a tilde the shell expands from
// HOME: `~` alone or `~/...`. A `~name` names an account's home directory
// from the user database instead.
func homeTilde(word string) bool {
	return word == "~" || strings.HasPrefix(word, "~/")
}

// globHolds reports whether the glob variant (a target word after brace
// expansion, with `/` separators), whose last segment matches name, can
// expand to an entry of that name: the directory it names, resolved from
// sc's working directory, holds one. It answers true (fail closed) when the
// directory part is itself a glob, climbs with `..` (which the shell takes
// through symlinks), names no directory segment after a trailing `/`, starts
// with a tilde that names no home directory (`~+` and `~-`, the working
// directories, and `~N`, the directory stack), or cannot be resolved (an
// unknown working directory), and when the lookup fails other than by the
// entry not existing.
func globHolds(sc scannedCommand, variant, name string) bool {
	i := strings.LastIndexByte(variant, '/')
	dir := variant[:i+1]
	if i == len(variant)-1 || hasGlobMeta(dir) || slices.Contains(strings.Split(dir, "/"), "..") ||
		unresolvedTilde(variant) {
		return true
	}
	p, ok := resolveWord(sc, dir+".")
	if !ok || !isRooted(p) {
		return true
	}
	_, err := sc.fs.Lstat(filepath.Join(p, name))
	return !errors.Is(err, fs.ErrNotExist)
}

// guardedTarget returns the entry of names that the write target word
// names (see guardedName; holds judges a glob), or that the existing file it
// resolves to from sc's working directory is named, following symlinks, or
// "".
func guardedTarget(sc scannedCommand, word string, names []string, holds func(variant, name string) bool) string {
	if n := guardedName(word, names, holds); n != "" {
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
	return guardedName(resolved, names, nil)
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
// after brace and glob expansion, or "". A glob base name that matches an
// entry counts where holds accepts the variant for it (see globScope), or
// always when holds is nil.
func guardedName(word string, names []string, holds func(variant, name string) bool) string {
	variants, ok := expandBraces(word)
	if !ok {
		variants = []string{word}
	}
	for _, v := range variants {
		slashed := strings.ReplaceAll(v, `\`, "/")
		base := strings.ToLower(path.Base(slashed))
		for _, n := range names {
			if base == n || (hasGlobMeta(base) && shellSegMatch(base, n) && (holds == nil || holds(slashed, n))) {
				return n
			}
		}
	}
	return ""
}
