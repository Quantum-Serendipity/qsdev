package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/gobwas/glob"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
)

// PathMatcher matches file paths against a path_glob / denied_path_check
// pattern. It is the single implementation of path-pattern semantics shared by
// the policy conditions and the MCP confused-deputy check, so a pattern means
// the same thing wherever it is enforced: gobwas glob syntax with no separator
// characters, so `*` and `**` both cross `/` and `**/x` matches x at any depth.
//
// A path is never matched only as the model spelled it. Match tries every form
// returned by PathForms (raw, lexically cleaned absolute, symlink-resolved, and
// CWD-relative), so dot-segments, doubled slashes, `..`, `~` and symlinked
// aliases cannot slip a protected path past the pattern.
type PathMatcher struct {
	globs    []glob.Glob
	foldCase bool
}

// CompilePathMatcher compiles pattern into a PathMatcher. Besides the pattern as
// written, it compiles the variant with a leading `~` expanded and the variant
// whose literal directory prefix is symlink-resolved, so an absolute pattern
// still matches the canonical (symlink-resolved) spelling of a path under it.
func CompilePathMatcher(pattern string) (*PathMatcher, error) {
	m := &PathMatcher{foldCase: caseInsensitiveFS()}
	for _, p := range patternVariants(pattern) {
		if m.foldCase {
			p = strings.ToLower(p)
		}
		g, err := glob.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("compiling path pattern %q: %w", pattern, err)
		}
		m.globs = append(m.globs, g)
	}
	return m, nil
}

// Match reports whether any form of path (see PathForms) matches the pattern.
// cwd anchors relative paths; when empty, the process working directory is used.
func (m *PathMatcher) Match(path, cwd string) bool {
	return m.MatchForms(PathForms(path, cwd))
}

// MatchForms reports whether any of the precomputed path forms matches. Callers
// matching one path against many patterns compute PathForms once and use this.
func (m *PathMatcher) MatchForms(forms []string) bool {
	for _, f := range forms {
		if m.foldCase {
			f = strings.ToLower(f)
		}
		for _, g := range m.globs {
			if g.Match(f) {
				return true
			}
		}
	}
	return false
}

// PathForms returns the distinct spellings of path a pattern is matched against:
// the raw input, the lexically cleaned absolute path (with `~` expanded and
// relative paths anchored at cwd), its symlink-resolved canonical form, and —
// for paths inside cwd — the cwd-relative form, so project-relative patterns
// such as `src/**` apply to the absolute paths first-party tools send. Forms use
// forward slashes, matching the pattern syntax on every platform.
func PathForms(path, cwd string) []string {
	if path == "" {
		return nil
	}
	forms := []string{path}
	add := func(p string) {
		if p == "" {
			return
		}
		p = filepath.ToSlash(p)
		if !slices.Contains(forms, p) {
			forms = append(forms, p)
		}
	}

	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}

	lexical := absolutePath(path, cwd)
	add(lexical)

	canonical, err := canon.Canonicalize(lexical)
	if err == nil {
		add(canonical)
	}

	if cwd != "" {
		for _, p := range []string{lexical, canonical} {
			add(relativeTo(cwd, p))
		}
	}
	return forms
}

// absolutePath expands a leading `~`, anchors a relative path at cwd, and
// cleans the result lexically (collapsing `.`, `..` and doubled separators).
func absolutePath(path, cwd string) string {
	expanded, err := canon.ExpandTilde(path)
	if err != nil {
		expanded = path
	}
	if !filepath.IsAbs(expanded) && cwd != "" {
		expanded = filepath.Join(cwd, expanded)
	}
	return filepath.Clean(expanded)
}

// relativeTo returns p relative to base when p lies inside base, else "".
func relativeTo(base, p string) string {
	if p == "" {
		return ""
	}
	rel, err := filepath.Rel(base, p)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return rel
}

// patternVariants returns pattern plus its `~`-expanded and prefix-canonicalized
// variants. Only the literal directory prefix before the first glob
// metacharacter is resolved; the glob tail is kept verbatim.
func patternVariants(pattern string) []string {
	variants := []string{pattern}
	add := func(p string) {
		if p != "" && !slices.Contains(variants, p) {
			variants = append(variants, p)
		}
	}

	expanded := pattern
	if pattern == "~" || strings.HasPrefix(pattern, "~/") {
		if e, err := canon.ExpandTilde(pattern); err == nil {
			expanded = filepath.ToSlash(e)
			add(expanded)
		}
	}

	if !filepath.IsAbs(filepath.FromSlash(expanded)) {
		return variants
	}
	metaIdx := strings.IndexAny(expanded, `*?[{\`)
	if metaIdx < 0 {
		if c, err := canon.Canonicalize(expanded); err == nil {
			add(filepath.ToSlash(c))
		}
		return variants
	}
	slash := strings.LastIndex(expanded[:metaIdx], "/")
	if slash <= 0 {
		return variants
	}
	prefix, tail := expanded[:slash], expanded[slash:]
	if c, err := canon.Canonicalize(filepath.FromSlash(prefix)); err == nil {
		add(filepath.ToSlash(c) + tail)
	}
	return variants
}

// caseInsensitiveFS reports whether the host's default filesystem folds case,
// in which case `.SSH/id_rsa` names the same file as `.ssh/id_rsa`.
func caseInsensitiveFS() bool {
	return runtime.GOOS == "darwin" || runtime.GOOS == "windows"
}

// pathArgKeys are the tool-input fields that carry a local file path: the
// first-party file tools (file_path, notebook_path), common MCP spellings
// (path, file), multi-path tools (paths), and move/copy tools (source,
// destination).
var pathArgKeys = []string{"file_path", "path", "file", "notebook_path", "paths", "source", "destination"}

// extractPathsFromInput returns every path-like argument in a tool input,
// flattening string arrays. Malformed JSON yields no paths; a path field of the
// wrong shape is skipped without hiding the readable paths in other fields.
func extractPathsFromInput(input json.RawMessage) []string {
	if len(input) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(input, &m); err != nil {
		return nil
	}
	paths, _ := PathArgs(m, pathArgKeys)
	return paths
}

// PathArgs collects the path values of the named fields from decoded tool
// arguments. A field may hold a string or an array of strings; an absent or
// null field is skipped. Any other shape — including a non-string array element
// — is reported as an error so callers guarding a known path-bearing tool can
// fail closed instead of skipping a path they could not read. A malformed field
// never hides the others: every readable path from every field is returned
// alongside the joined error.
func PathArgs(fields map[string]json.RawMessage, keys []string) ([]string, error) {
	var paths []string
	var errs []error
	for _, key := range keys {
		raw, ok := fields[key]
		if !ok || string(raw) == "null" {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			if s != "" {
				paths = append(paths, s)
			}
			continue
		}
		var list []json.RawMessage
		if err := json.Unmarshal(raw, &list); err != nil {
			errs = append(errs, fmt.Errorf("path argument %q is neither a string nor a string array: %w", key, err))
			continue
		}
		for i, elem := range list {
			var p string
			if err := json.Unmarshal(elem, &p); err != nil {
				errs = append(errs, fmt.Errorf("path argument %q element %d is not a string: %w", key, i, err))
				continue
			}
			if p != "" {
				paths = append(paths, p)
			}
		}
	}
	return paths, errors.Join(errs...)
}

// evalPaths returns the paths a path condition inspects: the context's file
// path plus every path-like argument in the tool input.
func evalPaths(ctx *EvalContext) []string {
	paths := extractPathsFromInput(ctx.ToolInput)
	if ctx.FilePath != "" && !slices.Contains(paths, ctx.FilePath) {
		paths = append([]string{ctx.FilePath}, paths...)
	}
	return paths
}

// matchContextPaths reports whether any path in ctx matches m.
func matchContextPaths(m *PathMatcher, ctx *EvalContext) bool {
	for _, p := range evalPaths(ctx) {
		if m.Match(p, ctx.CWD) {
			return true
		}
	}
	return false
}
