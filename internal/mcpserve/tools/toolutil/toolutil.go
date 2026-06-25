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

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
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
