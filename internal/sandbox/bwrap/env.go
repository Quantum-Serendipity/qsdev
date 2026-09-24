package bwrap

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// allowedExact is the set of environment variable names always permitted
// through the sandbox filter.
var allowedExact = map[string]bool{
	"PATH":               true,
	"HOME":               true,
	"TMPDIR":             true,
	"LANG":               true,
	"TERM":               true,
	"NODE_PATH":          true,
	"PYTHONPATH":         true,
	"GOPATH":             true,
	"GOMODCACHE":         true,
	"GOROOT":             true,
	"RUSTUP_HOME":        true,
	"CARGO_HOME":         true,
	"XDG_CACHE_HOME":     true,
	"XDG_CONFIG_HOME":    true,
	"CLAUDE_PROJECT_DIR": true,
	"USER":               true,
	"SHELL":              true,
}

// allowedPrefixes lists name prefixes that are always permitted.
// FILE_BOUNDARY_ and TOOL_GATES_ carry the file-boundary and tool-gates
// hooks' policy (settings.json "env"), which a sandboxed hook must still
// receive: stripped, the tool gates would allow every tool. With GOMODCACHE
// and GOROOT above, FILE_BOUNDARY_ tells the hook which dependency sources it
// may let the agent read.
var allowedPrefixes = []string{
	"FILE_BOUNDARY_",
	"TOOL_GATES_",
	"LC_",
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_INDEX_FILE",
	"GIT_AUTHOR_",
	"GIT_COMMITTER_",
}

// deniedExact is the set of variable names that are always stripped,
// even if they would otherwise match the allowlist.
var deniedExact = map[string]bool{
	"AWS_ACCESS_KEY_ID":     true,
	"AWS_SECRET_ACCESS_KEY": true,
	"AWS_SESSION_TOKEN":     true,
	"GITHUB_TOKEN":          true,
	"NPM_TOKEN":             true,
	"DOCKER_PASSWORD":       true,
	"REGISTRY_PASSWORD":     true,
}

// deniedSuffixes lists name suffixes that cause a variable to be stripped.
var deniedSuffixes = []string{
	"_SECRET",
	"_TOKEN",
	"_KEY",
	"_PASSWORD",
	"_CREDENTIALS",
}

// FilterEnvironment returns a copy of env containing only allowlisted
// variables. Credential patterns are always stripped, even when a variable
// matches the allowlist. Every hook category currently gets the same
// allowlist; the category parameter is reserved for per-category additions.
// The result is never nil, but callers must still pass it to exec.Cmd.Env
// through sandbox.EnvList: an empty map rendered as a nil Env would make the
// hook inherit the full parent environment.
func FilterEnvironment(env map[string]string, _ sandbox.HookCategory) map[string]string {
	out := make(map[string]string, len(env))
	for k, v := range env {
		if isDenied(k) {
			continue
		}
		if isAllowed(k) {
			out[k] = v
		}
	}
	return out
}

func isDenied(name string) bool {
	if deniedExact[name] {
		return true
	}
	for _, suffix := range deniedSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func isAllowed(name string) bool {
	if allowedExact[name] {
		return true
	}
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
