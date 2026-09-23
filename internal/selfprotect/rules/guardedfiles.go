package rules

import (
	"path"
	"slices"
	"strings"
)

// BashRewritesFile reports which of names (lower-cased base names) a Bash
// command writes, deletes, moves or edits in place, using the same mutation
// analysis as the protected-path rules. Those files are guarded by a
// before/after content check that only the Write and Edit tools go through, so
// a shell rewrite (`echo ignore-scripts=false >> .npmrc`, `rm .npmrc`) would
// bypass it. A line that mentions one of the names but cannot be parsed, or
// whose mutating command uses an expansion that could carry the name, counts
// as a rewrite (fail closed).
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
	for _, sc := range scs {
		targets := sc.WriteRedirects
		mutates := len(sc.WriteRedirects) > 0
		if !recordsOnly(sc) {
			targets = slices.Concat(mutationTargets(sc), sc.WriteRedirects)
			mutates = mutates || isMutating(sc)
		}
		for _, t := range targets {
			if n := guardedName(t, names); n != "" {
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
		if mentioned != "" && sc.HasExpansion && mutates {
			return mentioned, true
		}
	}
	return "", false
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
