package rules

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
)

// This file models find expressions: whether a find with a mutating action can
// select a protected file even when no protected path is spelled out.

// findMutatingActions are find primaries that delete, run a command on, or
// write about each matched file.
var findMutatingActions = map[string]bool{
	"-delete": true, "-exec": true, "-execdir": true, "-ok": true, "-okdir": true,
	"-fprint": true, "-fprint0": true, "-fprintf": true, "-fls": true,
}

// findNamePredicates take a basename glob; findPathPredicates a whole-path glob
// (where * also matches '/'); findRegexPredicates a regular expression.
var (
	findNamePredicates  = map[string]bool{"-name": true, "-iname": true, "-lname": true, "-ilname": true}
	findPathPredicates  = map[string]bool{"-path": true, "-ipath": true, "-wholename": true, "-iwholename": true}
	findRegexPredicates = map[string]bool{"-regex": true, "-iregex": true}
)

// findPattern is one name/path predicate of a find expression, compiled once
// so it can be tried against every probe.
type findPattern struct{ match func(probe string) bool }

// findMutatesProtected reports whether a find command can delete or rewrite a
// protected file even though no protected path is spelled out, e.g.
// `find ~ -name settings.json -path '*claude*' -delete`. Only a find with a
// mutating action is considered. Its name/path patterns are evaluated against
// representative protected entries (canon.FindProbes) under each start point:
// the location-independent ones anywhere below it, the home- and
// system-anchored ones (the org overlay, say) where they lie below it.
// Negation, regex predicates, or no pattern at all (every file matches) count
// as selecting a protected file when a start point can contain one.
func findMutatesProtected(sc scannedCommand) bool {
	starts, exprs := splitFindArgs(sc.Args)
	if !findHasMutatingAction(exprs) {
		return false
	}
	patterns, opaque, anyOf := parseFindPatterns(exprs)
	relative, absolute := canon.FindProbes()
	for _, start := range starts {
		if !findStartCanReachProtected(sc, start) {
			continue
		}
		if opaque || len(patterns) == 0 {
			return true
		}
		for _, probe := range findProbesUnder(sc, start, relative, absolute) {
			if findPatternsSelect(patterns, anyOf, probe) {
				return true
			}
		}
	}
	return false
}

// findProbesUnder returns the probes as find would print them below start:
// every relative probe, and every absolute probe within the directory start
// resolves to.
func findProbesUnder(sc scannedCommand, start string, relative, absolute []string) []string {
	prefix := strings.TrimSuffix(filepath.ToSlash(expandTilde(start)), "/")
	probes := make([]string, 0, len(relative))
	for _, rel := range relative {
		probes = append(probes, prefix+"/"+rel)
	}
	dir, known := resolveWord(sc, start)
	if !known {
		return probes
	}
	for _, abs := range absolute {
		rel, err := filepath.Rel(dir, filepath.FromSlash(abs))
		if err != nil || !withinDir(filepath.FromSlash(abs), dir) {
			continue
		}
		if rel = filepath.ToSlash(rel); rel == "." {
			probes = append(probes, prefix)
		} else {
			probes = append(probes, prefix+"/"+rel)
		}
	}
	return probes
}

func findHasMutatingAction(exprs []string) bool {
	for _, e := range exprs {
		if findMutatingActions[e] {
			return true
		}
	}
	return false
}

// parseFindPatterns extracts the name/path predicates of a find expression.
// opaque reports a negation or regex predicate (which cannot be evaluated
// against probes); anyOf reports an -o/-or/, alternative.
func parseFindPatterns(exprs []string) (patterns []findPattern, opaque, anyOf bool) {
	for i := 0; i < len(exprs); i++ {
		e := exprs[i]
		switch {
		case e == "!" || e == "-not":
			opaque = true
		case e == "-o" || e == "-or" || e == ",":
			anyOf = true
		case findRegexPredicates[e]:
			opaque = true
		case (findNamePredicates[e] || findPathPredicates[e]) && i+1 < len(exprs):
			patterns = append(patterns, findPattern{compileFindPattern(e, exprs[i+1])})
			i++
		}
	}
	return patterns, opaque, anyOf
}

// findPatternsSelect reports whether the patterns select probe: all of them
// (an implicit -and), or any of them when the expression has an alternative.
func findPatternsSelect(patterns []findPattern, anyOf bool, probe string) bool {
	matchedAll, matchedAny := true, false
	for _, p := range patterns {
		if p.match(probe) {
			matchedAny = true
		} else {
			matchedAll = false
		}
	}
	return matchedAll || (anyOf && matchedAny)
}

// splitFindArgs separates find's start points (the leading non-expression
// words) from its expression. With no start point find searches ".".
func splitFindArgs(args []string) (starts, exprs []string) {
	i := 0
	for i < len(args) && !strings.HasPrefix(args[i], "-") && args[i] != "!" && args[i] != "(" {
		i++
	}
	starts, exprs = args[:i], args[i:]
	if len(starts) == 0 {
		starts = []string{"."}
	}
	return starts, exprs
}

// findStartCanReachProtected reports whether a find start point can contain a
// protected entry: it names one, it is the working directory or an ancestor of
// it, it is an ancestor of the home directory or /etc, or it is at or above a
// home- or system-anchored protected location (~/.config, say), also when an
// expansion spells the home directory (see expandedReachesAncestor).
func findStartCanReachProtected(sc scannedCommand, start string) bool {
	if refersProtected(sc, start) || expandedReachesAncestor(sc, start) {
		return true
	}
	p, known := resolveWord(sc, start)
	if !known || !isRooted(p) {
		// Unknown base directory: "." and parent traversals may cover the
		// project root; a plain subdirectory name cannot hold its .claude.
		clean := filepath.ToSlash(filepath.Clean(start))
		return !known || clean == "." || clean == ".." || strings.HasPrefix(clean, "../")
	}
	for _, base := range []string{sc.cwd, expandTilde("~"), "/etc"} {
		if base != "" && base != "~" && isRooted(base) && withinDir(filepath.Clean(base), p) {
			return true
		}
	}
	return isProtectedAncestor(sc.fs, p)
}

// compileFindPattern returns the matcher of one find name/path predicate for
// probe, a full path as find would print it. A malformed pattern matches
// (fail closed).
func compileFindPattern(pred, glob string) func(probe string) bool {
	fold := strings.HasPrefix(pred, "-i")
	if fold {
		glob = strings.ToLower(glob)
	}
	norm := func(probe string) string {
		if fold {
			return strings.ToLower(probe)
		}
		return probe
	}
	if findNamePredicates[pred] {
		return func(probe string) bool {
			ok, err := path.Match(glob, path.Base(norm(probe)))
			return ok || err != nil
		}
	}
	re, err := regexp.Compile("^" + globToRegexp(glob) + "$")
	if err != nil {
		return func(string) bool { return true }
	}
	return func(probe string) bool { return re.MatchString(norm(probe)) }
}

// globToRegexp converts a find -path glob to a regular expression in which *
// and ? also match '/', as fnmatch without FNM_PATHNAME does.
func globToRegexp(glob string) string {
	var b strings.Builder
	for i := 0; i < len(glob); i++ {
		switch c := glob[i]; c {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '[':
			end := strings.IndexByte(glob[i+1:], ']')
			if end < 0 {
				b.WriteString(`\[`)
				continue
			}
			class := glob[i+1 : i+1+end]
			if strings.HasPrefix(class, "!") {
				class = "^" + class[1:]
			}
			b.WriteString("[" + strings.ReplaceAll(class, `\`, `\\`) + "]")
			i += end + 1
		case '\\':
			if i+1 < len(glob) {
				i++
				b.WriteString(regexp.QuoteMeta(string(glob[i])))
			}
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	return b.String()
}
