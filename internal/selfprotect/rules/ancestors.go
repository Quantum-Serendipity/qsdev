package rules

import (
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
)

// replaceVerbs delete, move, copy or link a path operand as a whole, so an
// operand that is a directory holding a protected location removes or
// replaces that location without naming it: the delete and link verbs, and
// the copy verbs that take whole trees. Other commands are left to the
// named-path analysis: `du -sh ~` or `tar czf x.tgz ~` only read the tree, and
// find removes its start points only through its expression, which
// findMutatesProtected judges.
var replaceVerbs = func() map[string]bool {
	verbs := maps.Clone(deleteVerbs)
	maps.Copy(verbs, linkVerbs)
	for _, v := range []string{"cp", "mv", "rsync"} {
		verbs[v] = copyVerbs[v]
	}
	delete(verbs, "find")
	return verbs
}()

// replacesProtectedAncestor reports whether a command deletes, moves or
// replaces a directory that holds a home- or system-anchored protected
// location (canon.ProtectedLocations): `rm -rf ~/.config`,
// `mv ~/.config ~/.config.bak` or `mv /tmp/o ~/.config` remove or plant the
// org overlay in one call. A copy or move into an existing directory writes
// an entry below it, so only that entry is checked and `cp notes.txt ~` stays
// allowed. Location-independent entries (.claude, the state directory) have
// no fixed ancestor and are not considered.
func replacesProtectedAncestor(scs []scannedCommand) bool {
	for _, sc := range scs {
		if !replaceVerbs[sc.Name] {
			continue
		}
		for _, p := range ancestorCandidates(sc) {
			if wordReachesProtectedAncestor(sc, p) {
				return true
			}
		}
	}
	return false
}

// ancestorCandidates returns the words of sc whose paths are removed or
// replaced as a whole: every operand of a delete, the sources of a move, and
// the destination of a copy, move or link, or the entries it creates below an
// existing destination directory.
func ancestorCandidates(sc scannedCommand) []string {
	if !copyVerbs[sc.Name] && !linkVerbs[sc.Name] {
		return nonFlagArgs(operandArgs(sc))
	}
	sources, dest := copySourcesAndDest(sc)
	var out []string
	if !isNonDestructiveCopy(sc) && !linkVerbs[sc.Name] {
		out = append(out, sources...) // mv removes its sources
	}
	if dest == "" {
		return out
	}
	resolved, known := resolveWord(sc, dest)
	if !known || hasGlobMeta(dest) || noTargetDirectory(sc) || !isDir(resolved) {
		return append(out, dest)
	}
	for _, src := range sources {
		out = append(out, entryInDir(sc, dest, src))
	}
	return out
}

// entryInDir returns the path a copy, move or link of src into the existing
// directory dir creates. rsync copies a source with a trailing separator, and
// cp a `src/.` one, into dir itself, which is then the entry.
func entryInDir(sc scannedCommand, dir, src string) string {
	base := filepath.Base(src)
	if base == "." || (sc.Name == "rsync" && strings.HasSuffix(src, "/")) || hasGlobMeta(base) {
		return dir
	}
	return filepath.Join(dir, base)
}

// noTargetDirectory reports whether cp/mv/ln was told to treat its
// destination as the target itself (-T, --no-target-directory), even when it
// is an existing directory. rsync's -T takes a value and only makes the check
// stricter.
func noTargetDirectory(sc scannedCommand) bool {
	for _, a := range sc.Args {
		if a == "--" {
			return false
		}
		if a == "--no-target-directory" || (isFlag(a) && !strings.HasPrefix(a, "--") && strings.ContainsRune(a, 'T')) {
			return true
		}
	}
	return false
}

// isDir reports whether p is an existing directory (following symlinks).
func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// wordReachesProtectedAncestor reports whether word p, used by sc, can name a
// directory that is, or holds, a home- or system-anchored protected location,
// after brace and glob expansion and cwd resolution. A relative word from an
// unknown directory is left to the relative-write analysis.
func wordReachesProtectedAncestor(sc scannedCommand, p string) bool {
	if expandedReachesAncestor(sc, p) {
		return true
	}
	variants, ok := expandBraces(p)
	if !ok {
		return true
	}
	for _, v := range variants {
		if v == "" {
			continue
		}
		if hasGlobMeta(v) {
			if globReachesAncestor(filepath.Clean(expandTilde(v))) {
				return true
			}
			continue
		}
		resolved, known := resolveWord(sc, v)
		if known && isRooted(resolved) && isProtectedAncestor(resolved) {
			return true
		}
	}
	return false
}

// isProtectedAncestor reports whether the absolute path p is a protected
// location or a directory above one, under its spelling as written or its
// symlink-resolved one. The comparison is on path keys with a trailing
// separator, so ~/.conf is not taken for an ancestor of ~/.config/<app>/.
func isProtectedAncestor(p string) bool {
	locs := canon.ProtectedLocations()
	if locs == nil {
		return true // the table could not be built: fail closed
	}
	for _, s := range pathSpellings(p) {
		prefix := strings.TrimSuffix(canon.PathKey(s), "/") + "/"
		for _, loc := range locs {
			key := canon.PathKey(loc)
			if strings.HasPrefix(key, prefix) || strings.TrimSuffix(key, "/")+"/" == prefix {
				return true
			}
		}
	}
	return false
}

// globReachesAncestor reports whether the rooted glob pattern can name a
// directory at or above a home- or system-anchored protected location: each
// of its segments matches the location's segment in the same position.
func globReachesAncestor(pattern string) bool {
	if !isRooted(pattern) {
		return false
	}
	segs := strings.Split(canon.PathKey(pattern), "/")
	for _, loc := range canon.ProtectedLocations() {
		locSegs := strings.Split(strings.TrimSuffix(canon.PathKey(loc), "/"), "/")
		if len(segs) > len(locSegs) {
			continue
		}
		matched := true
		for i, seg := range segs {
			if !shellSegMatch(seg, locSegs[i]) {
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

// expandedReachesAncestor reports whether word, an argument of sc built from
// an expansion the scan cannot render (`"$(echo ~)/.config"`, `$H/.config`
// after `H=~`), can still name a directory at or above a protected location
// below the home directory: the expansion may be the home directory, so the
// word fails closed when its literal tail is the home-relative path of such a
// directory (it ends in /.config or /.config/<app>, say). Directories at or
// above the home directory are not tails: `rm -rf "$tmp"` stays allowed.
func expandedReachesAncestor(sc scannedCommand, word string) bool {
	if !slices.Contains(sc.ExpandedArgs, word) {
		return false
	}
	locs := canon.ProtectedLocations()
	if locs == nil {
		return true // the table could not be built: fail closed
	}
	home := expandTilde("~")
	if !isRooted(home) {
		return false
	}
	homeKey := strings.TrimSuffix(canon.PathKey(home), "/") + "/"
	key := canon.PathKey(filepath.Clean(word))
	for _, loc := range locs {
		rel, ok := strings.CutPrefix(canon.PathKey(loc), homeKey)
		if !ok {
			continue
		}
		for dir := strings.TrimSuffix(rel, "/"); dir != "" && dir != "."; dir = path.Dir(dir) {
			if key == dir || strings.HasSuffix(key, "/"+dir) {
				return true
			}
		}
	}
	return false
}
