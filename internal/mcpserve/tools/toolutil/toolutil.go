// Package toolutil holds the shared helpers used by the security and devenv MCP
// tool handlers (Phase 32, Unit 32.9): argument extraction from the decoded
// JSON arguments map, structured-result builders, and the canonical
// not_configured graceful-degradation result.
//
// These helpers exist so the seven tool handlers stay free of repetitive
// type-assertion and result-shaping boilerplate and so every tool degrades
// identically when a prerequisite is missing.
package toolutil

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// StringArg returns the string value of args[name]. The boolean reports whether
// a string value was present (a missing key or a non-string value yields "",
// false). JSON decoding yields a Go string for JSON strings.
func StringArg(args map[string]any, name string) (string, bool) {
	v, ok := args[name].(string)
	return v, ok
}

// StringArgOr returns args[name] when it is a non-empty string, otherwise def.
func StringArgOr(args map[string]any, name, def string) string {
	if v, ok := args[name].(string); ok && v != "" {
		return v
	}
	return def
}

// StringSliceArg returns args[name] as a []string. It accepts both a JSON array
// (decoded as []any whose string elements are collected) and an already-typed
// []string. A missing key or wrong type yields nil. Non-string elements within
// an array are skipped rather than failing, so a malformed argument degrades to
// the strings it could recover.
func StringSliceArg(args map[string]any, name string) []string {
	switch v := args[name].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// IntArg returns the integer value of args[name]. JSON decoding yields float64
// for all JSON numbers, so a float64 is truncated to an int; an int or int64 is
// also accepted. The boolean reports whether a numeric value was present.
func IntArg(args map[string]any, name string) (int, bool) {
	switch v := args[name].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}

// Result builds a success ToolResult carrying a human-readable text summary and
// a structured JSON-serializable payload. Both are surfaced to the client: the
// text for display, the structured content for programmatic consumption.
func Result(text string, structured any) *spi.ToolResult {
	return &spi.ToolResult{Text: text, Structured: structured}
}

// NotConfigured builds the canonical graceful-degradation result: a structured
// not_configured payload (status + reason + any extra fields) with IsError set.
// The mcpserve bridge preserves IsError on the structured path, so MCP clients
// see both the machine-readable reason and the error flag. A missing
// prerequisite or unconfigured provider must reach a handler through this helper
// rather than returning a Go error, so the server never surfaces it as a
// protocol error or crashes.
func NotConfigured(reason string, extra map[string]any) *spi.ToolResult {
	payload := map[string]any{"status": "not_configured", "reason": reason}
	for k, v := range extra {
		payload[k] = v
	}
	return &spi.ToolResult{
		Text:       "not_configured: " + reason,
		Structured: payload,
		IsError:    true,
	}
}

// ErrorResult builds a tool-level error result carrying a structured payload and
// IsError. It is used for runtime failures (an external API rejected the call, a
// subprocess failed to start) that are not missing-prerequisite conditions.
func ErrorResult(reason string, extra map[string]any) *spi.ToolResult {
	payload := map[string]any{"status": "error", "reason": reason}
	for k, v := range extra {
		payload[k] = v
	}
	return &spi.ToolResult{
		Text:       "error: " + reason,
		Structured: payload,
		IsError:    true,
	}
}

// MarshalText renders v as indented JSON for the human-readable Text field,
// falling back to fallback when marshaling fails (it never returns an error so
// callers can use it inline).
func MarshalText(v any, fallback string) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fallback
	}
	return string(b)
}

// EmptyObjectSchema is the canonical JSON Schema for a tool that accepts no
// arguments. It is the one shared empty-object schema every no-argument tool
// across the project context surface and the framework adapters registers, so a
// single definition keeps the wire schema identical everywhere. A fresh map is
// returned per call so callers may not mutate a shared instance.
func EmptyObjectSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

// DetectedLanguages renders a detection result's programming languages as the
// canonical label set: the COMPLETE language surface qsdev detects (go, node,
// rust, python, the two JVM build systems, and dotnet), with a version suffix
// appended when one was detected. It is the single detection->language-labels
// helper shared by the generic project context surface (qsdev_project_info /
// qsdev_detect) and the framework stub adapters, so both report exactly the same
// languages for a project rather than diverging.
func DetectedLanguages(d types.DetectedProject) []string {
	var out []string
	add := func(present bool, label string) {
		if present {
			out = append(out, label)
		}
	}
	add(d.HasGoMod, langLabel("go", d.GoVersion))
	add(d.HasPackageJSON, langLabel("node", d.NodeVersion))
	add(d.HasCargoToml, "rust")
	add(d.HasPyProject, langLabel("python", d.PythonVersion))
	add(d.HasPomXML, "java (maven)")
	add(d.HasBuildGradle, "java/kotlin (gradle)")
	add(d.HasCsproj, "dotnet")
	return out
}

// langLabel appends a detected version to a language name (e.g. "go 1.22"),
// returning the bare name when no version was detected.
func langLabel(name, version string) string {
	if version != "" {
		return name + " " + version
	}
	return name
}

// SortedTrueKeys returns the keys of m whose value is true, sorted. It is the
// single map[string]bool->present-keys helper shared by the generic project
// context surface (qsdev_project_info / qsdev_detect) and the framework stub
// adapters, so both report the same ecosystem/flag set for a project.
func SortedTrueKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		if v {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// JoinOrNone joins items with ", ", or returns "(none)" when the slice is empty.
// It is shared so every detection summary renders an empty list identically.
func JoinOrNone(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}

// ConfineToRoot confines candidate to root and returns the cleaned, absolute
// path when it stays inside root, or ok=false when it escapes (path traversal).
// A relative candidate is resolved against root; an absolute candidate must
// still fall within it. It is the single path-containment primitive shared by the
// file-reading tools (policy_check, security_scan), so every caller-supplied path
// is confined identically and an escaping path can be degraded to not_configured
// rather than reading an arbitrary host file.
//
// Containment is checked twice: first lexically (cheap, needs no filesystem
// access and works for not-yet-created targets), then symlink-aware — symlinks in
// the longest existing prefix of both root and candidate are resolved and the
// check is repeated, so an in-root symlink whose target escapes the root is
// rejected. Both sides are resolved so a root that itself lives under a symlinked
// path (e.g. /tmp -> /private/tmp) does not cause a false rejection; a not-yet-
// existing leaf stays lexical.
func ConfineToRoot(root, candidate string) (string, bool) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(absRoot, candidate)
	}
	candidate = filepath.Clean(candidate)

	if !withinRoot(absRoot, candidate) {
		return "", false
	}
	if !withinRoot(resolveExistingSymlinks(absRoot), resolveExistingSymlinks(candidate)) {
		return "", false
	}
	return candidate, true
}

// withinRoot reports whether candidate is lexically inside root (root itself
// counts as inside). Both must be absolute, cleaned paths.
func withinRoot(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel != ".." && !strings.HasPrefix(rel, "../")
}

// resolveExistingSymlinks resolves symlinks in the longest existing prefix of p,
// leaving any non-existent trailing components lexical. This lets containment
// follow a symlink that escapes the root even when the final target does not yet
// exist. p must be absolute.
func resolveExistingSymlinks(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	parent := filepath.Dir(p)
	if parent == p { // reached the filesystem root with nothing resolvable
		return p
	}
	return filepath.Join(resolveExistingSymlinks(parent), filepath.Base(p))
}
