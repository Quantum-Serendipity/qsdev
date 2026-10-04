package rules

import (
	"maps"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
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

// trashVerbs move their operands to the trash, which removes them from where
// they were as surely as rm.
var trashVerbs = map[string]bool{
	"trash-put": true, "trash": true, "gvfs-trash": true, "gvfs-rm": true, "gvfs-move": true,
}

// gioMoveSubcommands are the gio subcommands that remove or move their
// operands.
var gioMoveSubcommands = map[string]bool{"trash": true, "remove": true, "rm": true, "move": true, "mv": true, "rename": true}

// replacesOperands reports whether sc removes or replaces its path operands
// as a whole: a replaceVerbs or trashVerbs command, `gio trash|remove|move`,
// or a tar that deletes what it archives (--remove-files).
func replacesOperands(sc scannedCommand) bool {
	switch {
	case replaceVerbs[sc.Name], trashVerbs[sc.Name]:
		return true
	case sc.Name == "gio":
		return len(sc.Args) > 0 && gioMoveSubcommands[sc.Args[0]]
	case sc.Name == "tar":
		return slices.ContainsFunc(sc.Args, func(a string) bool {
			return cmdscan.WrapperOption{Name: a}.Is("--remove-files")
		})
	}
	return false
}

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
		if !replacesOperands(sc) {
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
	if !known || hasGlobMeta(dest) || noTargetDirectory(sc) || !isDir(sc.fs, resolved) {
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

// isDir reports whether p is an existing directory (following symlinks),
// asking fs.
func isDir(fs *canon.Resolver, p string) bool {
	info, err := fs.Stat(p)
	return err == nil && info.IsDir()
}

// wordReachesProtectedAncestor reports whether word p, used by sc, can name a
// directory that is, or holds, a home- or system-anchored protected location,
// after brace and glob expansion and cwd resolution. A word built from an
// expansion the scan cannot render, a word starting with a tilde form it
// cannot resolve (see unresolvedTilde), and a relative word after a cd whose
// target it could not resolve, may start at a home directory, so their
// literal tail is checked too (see tailReachesAncestor).
func wordReachesProtectedAncestor(sc scannedCommand, p string) bool {
	variants, ok := expandBraces(p)
	if !ok {
		return true
	}
	expanded := slices.Contains(sc.ExpandedArgs, p)
	for _, v := range variants {
		if v == "" {
			continue
		}
		if (expanded || unresolvedTilde(v)) && tailReachesAncestor(v) {
			return true
		}
		if isRelativePath(v) && sc.cwdUnknown {
			joined := path.Join(filepath.ToSlash(sc.cwdHint), filepath.ToSlash(v))
			if tailReachesAncestor(joined) || rootedReachesAncestor(sc.fs, joined) {
				return true
			}
			continue
		}
		if resolved, known := resolveWord(sc, v); known && rootedReachesAncestor(sc.fs, resolved) {
			return true
		}
	}
	return false
}

// rootedReachesAncestor reports whether p, once a leading ~ is expanded, is
// an absolute path or glob that can name a directory at or above a home- or
// system-anchored protected location. Filesystem lookups go through fs.
func rootedReachesAncestor(fs *canon.Resolver, p string) bool {
	p = filepath.Clean(expandTilde(p))
	if !isRooted(p) {
		return false
	}
	if hasGlobMeta(p) {
		return globReachesAncestor(p)
	}
	return isProtectedAncestor(fs, p)
}

// isProtectedAncestor reports whether the absolute path p is a protected
// location or a directory above one, under its spelling as written or its
// symlink-resolved one. The comparison is on path keys with a trailing
// separator, so ~/.conf is not taken for an ancestor of ~/.config/<app>/.
func isProtectedAncestor(fs *canon.Resolver, p string) bool {
	locKeys := canon.ProtectedLocationKeys()
	if locKeys == nil {
		return true // the table could not be built: fail closed
	}
	for _, s := range pathSpellings(fs, p) {
		prefix := strings.TrimSuffix(canon.PathKey(s), "/") + "/"
		for _, key := range locKeys {
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
	for _, key := range canon.ProtectedLocationKeys() {
		locSegs := strings.Split(strings.TrimSuffix(key, "/"), "/")
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
// an expansion the scan cannot render (`"$(cat f)/.config"`, `$H/.config`
// after `H=~`), or starting with a tilde form it cannot resolve, can still
// name a directory at or above a protected location below a home directory
// (see tailReachesAncestor).
func expandedReachesAncestor(sc scannedCommand, word string) bool {
	return (slices.Contains(sc.ExpandedArgs, word) || unresolvedTilde(word)) && tailReachesAncestor(word)
}

// tailReachesAncestor reports whether text, which may follow an unknown
// prefix that can be a home directory, names a directory at or above a
// protected location below it: its trailing segments match, segment by
// segment and as globs, the home-relative path of such a directory (it ends
// in /.config or /.config/<app>, or in /.conf*, say). Every protected home
// counts (canon.ProtectedHomes: HOME and the account's home directory).
// Directories at or above a home directory are not tails: `rm -rf "$tmp"`
// stays allowed.
func tailReachesAncestor(text string) bool {
	locKeys, homes := canon.ProtectedLocationKeys(), canon.ProtectedHomes()
	if locKeys == nil || homes == nil {
		return true // the table could not be built: fail closed
	}
	segs := strings.Split(canon.PathKey(filepath.Clean(text)), "/")
	for _, home := range homes {
		if !isRooted(home) {
			continue
		}
		homeKey := strings.TrimSuffix(canon.PathKey(home), "/") + "/"
		for _, key := range locKeys {
			rel, ok := strings.CutPrefix(key, homeKey)
			if !ok {
				continue
			}
			for dir := strings.TrimSuffix(rel, "/"); dir != "" && dir != "."; dir = path.Dir(dir) {
				if segmentsEndWith(segs, strings.Split(dir, "/")) {
					return true
				}
			}
		}
	}
	return false
}

// unresolvedTilde reports whether p starts with a tilde form the scan cannot
// render: ~name for an account the user database does not know, or a
// directory-stack form (~+, ~-, ~2). It can still be a home directory, so it
// is treated like an expansion (see tailReachesAncestor).
func unresolvedTilde(p string) bool {
	return strings.HasPrefix(p, "~") && !isRooted(expandTilde(p))
}

// segmentsEndWith reports whether the last segments of the glob segments segs
// match the literal segments want.
func segmentsEndWith(segs, want []string) bool {
	if len(segs) < len(want) {
		return false
	}
	tail := segs[len(segs)-len(want):]
	for i, w := range want {
		if !shellSegMatch(tail[i], w) {
			return false
		}
	}
	return true
}
