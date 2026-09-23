package canon

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"
)

var (
	protectedPrefixes []protectedEntry
	protectedSuffixes []protectedEntry
	initOnce          sync.Once
	initErr           error

	// userHomeDir resolves the current user's home directory. It is a package
	// variable (defaulting to os.UserHomeDir) so tests can simulate a
	// home-resolution failure and verify the fail-closed behavior of IsProtected.
	userHomeDir = os.UserHomeDir
)

type protectedEntry struct {
	path     string
	category string
}

func ensureInit() error {
	initOnce.Do(func() {
		home, err := userHomeDir()
		if err != nil {
			initErr = fmt.Errorf("resolving home directory: %w", err)
			return
		}

		// Home- and system-anchored locations. A directory entry ends in a
		// separator and protects everything below it; a file entry matches
		// exactly. Locations protected wherever they live (home config or
		// project checkout) are in protectedSegments instead.
		protectedPrefixes = []protectedEntry{
			{filepath.Join(home, ".qsdev", "bin") + string(filepath.Separator), "binary"},
			{filepath.Join(home, ".claude", "managed-settings.json"), "claude-settings"},
			// ~/.claude.json holds the user-scoped MCP servers, per-project
			// permission grants and trust decisions.
			{filepath.Join(home, ".claude.json"), "claude-settings"},
			{"/etc/gdev/", "system-config"},
			{"/etc/claude-code/", "system-config"},
		}

		protectedSuffixes = []protectedEntry{
			{string(filepath.Separator) + ".mcp.json", "mcp-config"},
		}
	})
	return initErr
}

// ExpandTilde replaces a leading ~ with the user's home directory.
func ExpandTilde(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := userHomeDir()
		if err != nil {
			return "", fmt.Errorf("expanding tilde: %w", err)
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}

// Canonicalize resolves a path to its canonical form.
// Tier 1: filepath.EvalSymlinks + filepath.Abs for paths that exist.
// Tier 2: for a path that does not exist (yet), resolveMissing walks it
// component by component the way the kernel does, following every existing
// symlink — including a dangling final one, which a write follows to create
// its target.
func Canonicalize(path string) (string, error) {
	expanded, err := ExpandTilde(path)
	if err != nil {
		return "", fmt.Errorf("canonicalizing path: %w", err)
	}

	// Tier 1: the full path exists (including through symlinks).
	resolved, err := filepath.EvalSymlinks(expanded)
	if err == nil {
		abs, absErr := filepath.Abs(resolved)
		if absErr != nil {
			return "", fmt.Errorf("resolving absolute path: %w", absErr)
		}
		return abs, nil
	}

	// Detect symlink loops or permission errors — don't attempt the walk.
	if os.IsPermission(err) || isSymlinkLoop(err) {
		return "", fmt.Errorf("canonicalizing path %q: %w", path, err)
	}

	// Tier 2: resolve the existing part and append the missing tail.
	return resolveMissing(expanded)
}

// maxSymlinkHops bounds symlink expansion in resolveMissing (Linux's
// MAXSYMLINKS), so a symlink cycle fails instead of looping forever.
const maxSymlinkHops = 40

// errTooManySymlinks reports a symlink chain longer than maxSymlinkHops.
var errTooManySymlinks = errors.New("too many levels of symbolic links")

// resolveMissing canonicalizes a path whose final target does not exist. It
// resolves components left to right: each existing component is Lstat-ed and,
// if it is a symlink, replaced by its target (relative targets resolve against
// the symlink's directory), and ".." is applied only after the component
// before it has been resolved. Unlike a lexical Clean, this sees that
// <dir>/lnk/../x is <lnk target's parent>/x and that a dangling symlink names
// its target, exactly as a write through the path would. Once a component is
// missing nothing below it exists, so the rest of the path is appended as-is.
func resolveMissing(p string) (string, error) {
	start, err := absWithoutClean(p)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}

	vol := filepath.VolumeName(start)
	resolved := vol + string(filepath.Separator)
	pending := splitPath(start[len(vol):])
	missing := false
	hops := 0

	for len(pending) > 0 {
		comp := pending[0]
		pending = pending[1:]
		switch comp {
		case ".":
			continue
		case "..":
			resolved = filepath.Dir(resolved)
			// A kernel walk fails at the missing component, so the path only
			// reaches a file when the consumer cleans it lexically first
			// (<dir>/missing/../lnk/x -> <dir>/lnk/x); resume resolving so a
			// symlink after the ".." is still followed.
			missing = false
			continue
		}

		next := filepath.Join(resolved, comp)
		if missing {
			resolved = next
			continue
		}
		info, err := os.Lstat(next)
		switch {
		case errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
			missing = true
			resolved = filepath.Join(normalizeExisting(resolved), comp)
			continue
		case err != nil:
			return "", fmt.Errorf("canonicalizing path %q: %w", p, err)
		case info.Mode()&fs.ModeSymlink == 0:
			resolved = next
			continue
		}

		hops++
		if hops > maxSymlinkHops {
			return "", fmt.Errorf("canonicalizing path %q: %w", p, errTooManySymlinks)
		}
		target, err := os.Readlink(next)
		if err != nil {
			return "", fmt.Errorf("canonicalizing path %q: %w", p, err)
		}
		if isRooted(target) {
			// An absolute target restarts resolution at its root (the current
			// volume when a Windows target is rooted without a drive).
			targetVol := filepath.VolumeName(target)
			target = target[len(targetVol):]
			if targetVol == "" {
				targetVol = vol
			}
			resolved = targetVol + string(filepath.Separator)
		}
		pending = append(splitPath(target), pending...)
	}
	return filepath.Clean(resolved), nil
}

// normalizeExisting returns the platform's canonical spelling of dir, an
// existing directory whose symlinks resolveMissing has already followed. On
// Windows filepath.EvalSymlinks also expands 8.3 short names (CLAUDE~1 ->
// .claude) and fixes the case of each component, which a component-wise
// Lstat walk does not; elsewhere it returns dir unchanged. On error dir is
// kept as-is.
func normalizeExisting(dir string) string {
	if norm, err := filepath.EvalSymlinks(dir); err == nil {
		return norm
	}
	return dir
}

// absWithoutClean makes p absolute WITHOUT lexically cleaning it, so ".."
// components survive for resolveMissing to apply after symlink resolution
// (filepath.Abs would collapse lnk/.. before lnk is resolved).
func absWithoutClean(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}
	if filepath.VolumeName(p) != "" {
		// Windows drive-relative path ("C:foo"): only Abs knows that drive's
		// working directory.
		return filepath.Abs(p)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	if isRooted(p) {
		return filepath.VolumeName(cwd) + p, nil
	}
	return cwd + string(filepath.Separator) + p, nil
}

// isRooted reports whether p is absolute or starts at a root separator.
func isRooted(p string) bool {
	return filepath.IsAbs(p) || (p != "" && os.IsPathSeparator(p[0]))
}

// splitPath splits p into its non-empty components, accepting every separator
// the platform does.
func splitPath(p string) []string {
	return strings.FieldsFunc(p, func(r rune) bool {
		return r < 0x80 && os.IsPathSeparator(uint8(r))
	})
}

func isSymlinkLoop(err error) bool {
	if errors.Is(err, errTooManySymlinks) {
		return true
	}
	if pathErr, ok := errors.AsType[*os.PathError](err); ok {
		return errors.Is(pathErr.Err, errors.ErrUnsupported) ||
			strings.Contains(pathErr.Err.Error(), "too many levels of symbolic links")
	}
	return false
}

// IsProtected checks whether a canonical path falls under any protected prefix.
// Returns (true, category) if protected, (false, "") otherwise.
//
// On case-insensitive filesystems (the macOS and Windows defaults) the match
// ignores case, so ~/.CLAUDE/settings.json cannot slip past a rule that names
// ~/.claude/settings.json while opening the same file. On Windows it also
// ignores the name aliases the filesystem strips (trailing dots and spaces, and
// an alternate-data-stream suffix such as "::$DATA").
func IsProtected(canonicalPath string) (bool, string) {
	return isProtected(canonicalPath, platformMatch)
}

func isProtected(canonicalPath string, opts matchOptions) (bool, string) {
	if err := ensureInit(); err != nil {
		// Fail closed. If the home directory cannot be resolved we cannot build
		// the home-anchored protected-prefix table, so we cannot prove that a
		// path is UNprotected. A self-protection control must never silently drop
		// protection because of an environment error (the phase-28 fail-closed
		// mandate), so treat every path as protected and let the rules deny the
		// operation. This is deliberately conservative and only triggers when
		// os.UserHomeDir fails (e.g. HOME/USERPROFILE unset), which is rare. The
		// "config" category makes SP-001 — and, via deny-overrides, the whole
		// Tier-1 rule set — block the operation.
		return true, "config"
	}

	key := opts.key(canonicalPath)

	// Check home- and system-anchored paths.
	for _, entry := range protectedPrefixes {
		entryKey := opts.key(entry.path)
		if strings.HasSuffix(entryKey, "/") {
			if strings.HasPrefix(key, entryKey) {
				return true, entry.category
			}
		} else if key == entryKey {
			return true, entry.category
		}
	}

	// Check locations protected wherever they live. A project-relative .claude/
	// or .qsdev/ canonicalizes OUTSIDE $HOME, so an anchored prefix cannot catch
	// it; the segment match does, so SP-001/SP-013 guard Write/Edit to them in
	// both the home config and a project checkout.
	for _, seg := range protectedSegments {
		if hasPathSegment(key, seg.segment) {
			return true, seg.category
		}
	}

	// Check suffix-based protected paths.
	for _, entry := range protectedSuffixes {
		entryKey := opts.key(entry.path)
		if strings.HasSuffix(key, entryKey) {
			return true, entry.category
		}
		// Also match when the path is exactly ".mcp.json" (no directory prefix).
		base := strings.TrimPrefix(entryKey, "/")
		if key == base || path.Base(key) == base {
			return true, entry.category
		}
	}

	return false, ""
}

// segmentEntry is a protected location matched wherever it appears in a path.
// A segment ending in "/" is a directory (it and everything below it are
// protected); otherwise it names a single file.
type segmentEntry struct {
	segment  string
	category string
}

// protectedSegments are the qsdev and Claude Code control files protected in
// any location — home config or project checkout. More specific entries come
// first so the first match yields the most precise category.
//
// The .claude entries are the files that register or steer enforcement: the
// settings that register the PreToolUse hooks and deny rules, the hook scripts,
// and the agent, command and skill definitions that can drive tools. Prose
// instruction files (CLAUDE.md, .claude/rules/) are legitimate edit targets and
// stay with gatedodge's content checks rather than being blocked outright.
// Other .claude content (e.g. Claude Code's own .claude/worktrees/ checkouts)
// stays writable.
//
// Every segment starts with one of protectedSubstringPatterns, so the
// raw-command check (ContainsProtectedPath) covers every location this table
// protects; TestProtectedSegmentsCoveredByCommandScan enforces that.
var protectedSegments = []segmentEntry{
	{".qsdev/audit/", "audit"},
	{".qsdev/", "config"},
	{".gdev/", "config"},
	{".claude/settings.json", "claude-settings"},
	{".claude/settings.local.json", "claude-settings"},
	{".claude/hooks/", "claude-settings"},
	{".claude/agents/", "claude-settings"},
	{".claude/commands/", "claude-settings"},
	{".claude/skills/", "claude-settings"},
}

// hasPathSegment reports whether the slash-separated path key contains seg as
// whole path components: a directory segment ("x/y/") matches the directory
// itself or anything below it, a file segment ("x/y") matches only at the end.
// Component boundaries are required, so "foo.claude/hooks/x" does not match
// ".claude/hooks/".
func hasPathSegment(key, seg string) bool {
	bounded := "/" + key
	if strings.HasSuffix(seg, "/") {
		return strings.Contains(bounded+"/", "/"+seg)
	}
	return strings.HasSuffix(bounded, "/"+seg)
}

// matchOptions describe how the host filesystem compares names.
type matchOptions struct {
	// foldCase compares names case-insensitively (macOS and Windows defaults).
	foldCase bool
	// windowsAliases strips the name aliases Windows ignores when it opens a
	// file: trailing dots and spaces, and an alternate-data-stream suffix.
	windowsAliases bool
}

var platformMatch = matchOptions{
	foldCase:       runtime.GOOS == "darwin" || runtime.GOOS == "windows",
	windowsAliases: runtime.GOOS == "windows",
}

// key returns p in the form protected-path comparisons use: slash-separated,
// with the platform's filesystem name aliases normalized away.
func (o matchOptions) key(p string) string {
	s := filepath.ToSlash(p)
	if o.windowsAliases {
		s = stripWindowsAliases(s)
	}
	if o.foldCase {
		s = strings.ToLower(s)
	}
	return s
}

// stripWindowsAliases removes, from each component of a slash-separated path,
// an alternate-data-stream suffix ("settings.json::$DATA" -> "settings.json")
// and trailing dots and spaces (".claude." -> ".claude"), which Windows
// discards when it opens the file. A drive component ("C:") and "."/".." are
// left alone.
func stripWindowsAliases(s string) string {
	parts := strings.Split(s, "/")
	for i, part := range parts {
		if part == "." || part == ".." || (len(part) == 2 && part[1] == ':') {
			continue
		}
		if j := strings.IndexByte(part, ':'); j >= 0 {
			part = part[:j]
		}
		parts[i] = strings.TrimRight(part, ". ")
	}
	return strings.Join(parts, "/")
}

// protectedSubstringPatterns are path fragments used by ContainsProtectedPath
// to detect protected path references in raw command strings. This is the
// union of all patterns previously in evasion.containsProtectedPath and
// rules.containsProtectedPathStr. Each carries a trailing separator, so a
// protected path FOLLOWED by more path (`.claude/settings.json`) is matched.
var protectedSubstringPatterns = []string{
	".claude/",
	".qsdev/",
	".gdev/",
	"/etc/gdev/",
	"/etc/claude-code/",
}

// protectedDirTokens are the bare protected directory names (no trailing
// separator), derived from protectedSubstringPatterns so the two lists cannot
// drift apart. ContainsProtectedPath matches these when they appear as a
// complete path segment, so a whole-directory operation like `rm -rf .claude`
// or `find .claude -delete` is caught. The boundary check prevents
// over-matching a longer name that merely embeds a token (`my.claude.bak`,
// `foo.claudex`, `my.claude`).
var protectedDirTokens = func() []string {
	tokens := make([]string, len(protectedSubstringPatterns))
	for i, p := range protectedSubstringPatterns {
		tokens[i] = strings.TrimSuffix(p, "/")
	}
	return tokens
}()

// protectedFileTokens are protected file names that sit directly in a
// directory other code may legitimately touch (the home directory), so no
// directory fragment above covers them. ContainsProtectedPath matches them as
// complete path segments, like protectedDirTokens, so `~/.claude.json.bak` and
// `my.claude.json` do not match.
var protectedFileTokens = []string{
	".claude.json",
}

// ContainsProtectedPath reports whether s contains any protected path
// fragment. Unlike IsProtected (which checks a canonical path against known
// prefixes/suffixes), this performs a substring search on raw text such as
// shell commands where the path may appear anywhere in the string. It matches
// both a protected path with trailing path (`.claude/settings.json`) and a bare
// protected directory name at a path-token boundary (`rm -rf .claude`). Like
// IsProtected, it ignores case on case-insensitive filesystems.
func ContainsProtectedPath(s string) bool {
	return containsProtectedPath(s, platformMatch.foldCase)
}

func containsProtectedPath(s string, foldCase bool) bool {
	normalized := filepath.ToSlash(s)
	if foldCase {
		normalized = strings.ToLower(normalized)
	}
	for _, p := range protectedSubstringPatterns {
		if strings.Contains(normalized, p) {
			return true
		}
	}
	for _, tok := range slices.Concat(protectedDirTokens, protectedFileTokens) {
		if containsSegment(normalized, tok) {
			return true
		}
	}
	return false
}

// containsSegment reports whether tok appears in s as a complete path segment:
// bounded on both sides by the start/end of the string or a character that is not
// part of a path token (a separator, whitespace, a shell metacharacter, a quote).
// This matches a bare protected directory name (`.claude`) while rejecting a
// longer name that merely embeds it (`my.claude.bak`, `foo.claudex`, `my.claude`).
func containsSegment(s, tok string) bool {
	for from := 0; ; {
		i := strings.Index(s[from:], tok)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(tok)
		if isTokenBoundary(s, start-1) && isTokenBoundary(s, end) {
			return true
		}
		from = start + 1
	}
}

// isTokenBoundary reports whether index i marks a path-token boundary in s: an
// out-of-range index (the string start or end) or a byte that cannot appear
// inside a single path token.
func isTokenBoundary(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return true
	}
	return !isPathTokenByte(s[i])
}

// isPathTokenByte reports whether b can appear inside one path token: an ASCII
// letter or digit, or the filename punctuation '.', '-', '_'. Every other byte
// (separators, whitespace, shell metacharacters, quotes) is a token boundary.
func isPathTokenByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	case b == '.', b == '-', b == '_':
		return true
	default:
		return false
	}
}
