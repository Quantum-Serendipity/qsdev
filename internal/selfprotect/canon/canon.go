package canon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

		protectedPrefixes = []protectedEntry{
			{filepath.Join(home, ".qsdev", "audit") + string(filepath.Separator), "audit"},
			{filepath.Join(home, ".qsdev", "bin") + string(filepath.Separator), "binary"},
			{filepath.Join(home, ".qsdev") + string(filepath.Separator), "config"},
			{filepath.Join(home, ".gdev") + string(filepath.Separator), "config"},
			{filepath.Join(home, ".claude", "settings.json"), "claude-settings"},
			{filepath.Join(home, ".claude", "settings.local.json"), "claude-settings"},
			{filepath.Join(home, ".claude", "managed-settings.json"), "claude-settings"},
			{filepath.Join(home, ".claude", "hooks") + string(filepath.Separator), "claude-settings"},
			{filepath.Join(home, ".claude", "agents") + string(filepath.Separator), "claude-settings"},
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
// Tier 2: parent-walk with lexical normalization for paths that don't exist yet.
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

	// Detect symlink loops or permission errors — don't attempt parent walk.
	if os.IsPermission(err) || isSymlinkLoop(err) {
		return "", fmt.Errorf("canonicalizing path %q: %w", path, err)
	}

	// Tier 2: walk up to the nearest existing ancestor.
	return parentWalk(expanded)
}

func parentWalk(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}

	// Split into components and find the deepest existing ancestor.
	dir := filepath.Dir(abs)
	remaining := []string{filepath.Base(abs)}

	for {
		resolved, evalErr := filepath.EvalSymlinks(dir)
		if evalErr == nil {
			absResolved, absErr := filepath.Abs(resolved)
			if absErr != nil {
				return "", fmt.Errorf("resolving absolute path: %w", absErr)
			}
			// Rejoin the unresolved tail onto the resolved ancestor.
			parts := append([]string{absResolved}, remaining...)
			return filepath.Clean(filepath.Join(parts...)), nil
		}

		if os.IsPermission(evalErr) || isSymlinkLoop(evalErr) {
			return "", fmt.Errorf("canonicalizing path %q: %w", path, evalErr)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding an existing ancestor.
			// Fall back to lexical cleaning.
			return filepath.Clean(abs), nil
		}
		remaining = append([]string{filepath.Base(dir)}, remaining...)
		dir = parent
	}
}

func isSymlinkLoop(err error) bool {
	if pathErr, ok := errors.AsType[*os.PathError](err); ok {
		return errors.Is(pathErr.Err, errors.ErrUnsupported) ||
			strings.Contains(pathErr.Err.Error(), "too many levels of symbolic links")
	}
	return false
}

// IsProtected checks whether a canonical path falls under any protected prefix.
// Returns (true, category) if protected, (false, "") otherwise.
func IsProtected(canonicalPath string) (bool, string) {
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

	// Check prefix-based protected paths.
	// More specific prefixes (audit, binary) are listed before their parents
	// (config) so the first match wins with the most precise category.
	for _, entry := range protectedPrefixes {
		// Exact match handles files like settings.json.
		if canonicalPath == entry.path {
			return true, entry.category
		}
		if strings.HasPrefix(canonicalPath, entry.path) {
			return true, entry.category
		}
	}

	// Hook scripts (.claude/hooks/) and agent definitions (.claude/agents/) are
	// protected wherever the .claude directory lives. A project-relative .claude/
	// canonicalizes OUTSIDE $HOME, so the home-anchored prefixes above cannot catch
	// a project hook/agent path; this segment check does, so SP-001 blocks
	// Write/Edit to a hook or agent file in both the home config and a project
	// checkout.
	normalized := filepath.ToSlash(canonicalPath)
	for _, frag := range protectedClaudeSubdirs {
		if strings.Contains(normalized, frag) {
			return true, "claude-settings"
		}
	}

	// Check suffix-based protected paths.
	for _, entry := range protectedSuffixes {
		if strings.HasSuffix(canonicalPath, entry.path) {
			return true, entry.category
		}
		// Also match when the path is exactly ".mcp.json" (no directory prefix).
		base := entry.path[len(string(filepath.Separator)):]
		if canonicalPath == base || filepath.Base(canonicalPath) == base {
			return true, entry.category
		}
	}

	return false, ""
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
// separator). ContainsProtectedPath matches these when they appear as a complete
// path segment, so a whole-directory operation like `rm -rf .claude` or
// `find .claude -delete` is caught. The boundary check prevents over-matching a
// longer name that merely embeds a token (`my.claude.bak`, `foo.claudex`,
// `my.claude`).
var protectedDirTokens = []string{
	".claude",
	".qsdev",
	".gdev",
	"/etc/gdev",
	"/etc/claude-code",
}

// protectedClaudeSubdirs are .claude subdirectories whose contents are protected
// wherever the .claude directory lives (home config or project checkout): the
// hook scripts that implement enforcement and the agent definitions that steer
// it. IsProtected matches these as substrings so SP-001 guards Write/Edit to them.
var protectedClaudeSubdirs = []string{
	".claude/hooks/",
	".claude/agents/",
}

// ContainsProtectedPath reports whether s contains any protected path
// fragment. Unlike IsProtected (which checks a canonical path against known
// prefixes/suffixes), this performs a substring search on raw text such as
// shell commands where the path may appear anywhere in the string. It matches
// both a protected path with trailing path (`.claude/settings.json`) and a bare
// protected directory name at a path-token boundary (`rm -rf .claude`).
func ContainsProtectedPath(s string) bool {
	normalized := filepath.ToSlash(s)
	for _, p := range protectedSubstringPatterns {
		if strings.Contains(normalized, p) {
			return true
		}
	}
	for _, tok := range protectedDirTokens {
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
