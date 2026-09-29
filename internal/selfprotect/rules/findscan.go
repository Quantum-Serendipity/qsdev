package rules

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"
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

// findProbePaths are representative protected entries, relative to a
// directory that holds them, used to decide whether a find's name/path
// patterns can select a protected file.
var findProbePaths = []string{
	".claude", ".claude/settings.json", ".claude/settings.local.json",
	".claude/hooks", ".claude/hooks/package-guard.py", ".claude/hooks/audit-log.sh",
	".claude/agents", ".claude/agents/agent.md",
	".qsdev", ".qsdev/config.yaml", ".qsdev/bin", ".qsdev/bin/qsdev",
	".qsdev/audit", ".qsdev/audit/audit.jsonl", ".qsdev/audit/events.log",
	".gdev", ".gdev/config.yaml",
}

// findPattern is one name/path predicate of a find expression.
type findPattern struct{ pred, glob string }

// findMutatesProtected reports whether a find command can delete or rewrite a
// protected file even though no protected path is spelled out, e.g.
// `find ~ -name settings.json -path '*claude*' -delete`. Only a find with a
// mutating action is considered. Its name/path patterns are evaluated against
// representative protected entries under each start point; negation, regex
// predicates, or no pattern at all (every file matches) count as selecting a
// protected file when a start point can contain one.
func findMutatesProtected(sc scannedCommand) bool {
	starts, exprs := splitFindArgs(sc.Args)
	if !findHasMutatingAction(exprs) {
		return false
	}
	patterns, opaque, anyOf := parseFindPatterns(exprs)
	for _, start := range starts {
		if !findStartCanReachProtected(sc, start) {
			continue
		}
		if opaque || len(patterns) == 0 {
			return true
		}
		prefix := strings.TrimSuffix(filepath.ToSlash(expandTilde(start)), "/")
		for _, rel := range findProbePaths {
			if findPatternsSelect(patterns, anyOf, prefix+"/"+rel) {
				return true
			}
		}
	}
	return false
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
			patterns = append(patterns, findPattern{e, exprs[i+1]})
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
		if findPatternMatches(p.pred, p.glob, probe) {
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
// it, or it is an ancestor of the home directory or /etc.
func findStartCanReachProtected(sc scannedCommand, start string) bool {
	if refersProtected(sc, start) {
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
	return false
}

// findPatternMatches applies one find name/path predicate to probe, a full path
// as find would print it. A malformed pattern matches (fail closed).
func findPatternMatches(pred, glob, probe string) bool {
	if strings.HasPrefix(pred, "-i") {
		glob, probe = strings.ToLower(glob), strings.ToLower(probe)
	}
	if findNamePredicates[pred] {
		ok, err := path.Match(glob, path.Base(probe))
		return ok || err != nil
	}
	re, err := regexp.Compile("^" + globToRegexp(glob) + "$")
	return err != nil || re.MatchString(probe)
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
