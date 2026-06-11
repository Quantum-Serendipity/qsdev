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
)

type protectedEntry struct {
	path     string
	category string
}

func ensureInit() error {
	initOnce.Do(func() {
		home, err := os.UserHomeDir()
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
		home, err := os.UserHomeDir()
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
		return false, ""
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
// rules.containsProtectedPathStr.
var protectedSubstringPatterns = []string{
	".claude/",
	".qsdev/",
	".gdev/",
	"/etc/gdev/",
	"/etc/claude-code/",
}

// ContainsProtectedPath reports whether s contains any protected path
// fragment. Unlike IsProtected (which checks a canonical path against known
// prefixes/suffixes), this performs a substring search on raw text such as
// shell commands where the path may appear anywhere in the string.
func ContainsProtectedPath(s string) bool {
	normalized := filepath.ToSlash(s)
	for _, p := range protectedSubstringPatterns {
		if strings.Contains(normalized, p) {
			return true
		}
	}
	return false
}
