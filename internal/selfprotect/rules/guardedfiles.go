package rules

import (
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// BashRewritesFile reports which of names (lower-cased base names) a Bash
// command writes, deletes, moves or edits in place, using the same mutation
// analysis as the protected-path rules. Those files are guarded by a
// before/after content check that only the Write and Edit tools go through, so
// a shell rewrite (`echo ignore-scripts=false >> .npmrc`, `rm .npmrc`) would
// bypass it. A command known to leave a file unchanged does not count (see
// writeTargets: `source .npmrc`, `sed -n 1p .npmrc`, the source of `ln -s`),
// and a target is also matched by the file it resolves to, so a write
// through a symlink to one of the files counts. A line that mentions one of
// the names but cannot be parsed, or whose writing command uses an expansion
// that could carry the name, counts as a rewrite (fail closed).
func BashRewritesFile(ctx *EvalContext, names []string) (string, bool) {
	if ctx.ToolName != "Bash" {
		return "", false
	}
	text := strings.ToLower(looseText(ctx.Command))
	mentioned := ""
	for _, n := range names {
		if strings.Contains(text, n) {
			mentioned = n
			break
		}
	}
	scs, err := ctx.scannedCommands()
	if err != nil {
		return mentioned, mentioned != ""
	}
	for i, sc := range scs {
		targets, writes := writeTargets(sc, scs[:i])
		for _, t := range targets {
			if n := guardedTarget(sc, t, names); n != "" {
				return n, true
			}
			// An inline program (`sh -c 'echo x >> .npmrc'`, `python3 -c
			// "open('.npmrc','w')"`) or an operand=value word (`dd
			// of=.npmrc`) names the file inside the word.
			if interpreterVerbs[sc.Name] || (!isFlag(t) && strings.Contains(t, "=")) {
				if n := nameInWord(t, names); n != "" {
					return n, true
				}
			}
		}
		if mentioned != "" && sc.HasExpansion && writes {
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
	fi, err := os.Stat(p)
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
	resolved, err := filepath.EvalSymlinks(p)
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
