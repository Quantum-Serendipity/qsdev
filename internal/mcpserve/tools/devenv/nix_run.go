package devenv

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
)

// defaultNixRunTimeout bounds a nix_run invocation when the caller omits timeout.
const defaultNixRunTimeout = 30 * time.Second

// procResult is the outcome of a process-group execution, shared by the
// platform-specific runProcessGroup implementations.
type procResult struct {
	stdout   string
	stderr   string
	exitCode int
	timedOut bool
	duration time.Duration
	startErr error
}

// nixRunner executes `nix run <command> -- <args>` in a dedicated process group
// so that, on timeout or cancellation, the entire group (including orphaned nix
// build children) is killed rather than leaked.
type nixRunner struct{}

func newNixRunner() *nixRunner { return &nixRunner{} }

// handle validates input, ensures nix is available, and runs the command in a
// process group with a timeout. A missing nix binary degrades to not_configured.
func (n *nixRunner) handle(ctx context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	command, ok := toolutil.StringArg(req.Arguments, "command")
	if !ok || command == "" {
		return toolutil.NotConfigured("command is required (e.g. nixpkgs#jq)",
			map[string]any{"example": "nixpkgs#jq"}), nil
	}
	// Enforce the installable policy before doing anything else so a remote
	// reference is rejected deterministically (no nix, no execution).
	if reason, rejected := installableRejection(command); rejected {
		return toolutil.ErrorResult("rejected remote nix installable",
			map[string]any{"command": command, "reason": reason}), nil
	}
	if _, err := exec.LookPath("nix"); err != nil {
		return toolutil.NotConfigured("nix is not installed or not on PATH",
			map[string]any{"error": err.Error(), "remediation": "install Nix or enter the devenv shell"}), nil
	}

	timeout := nixRunTimeout(req.Arguments)
	extraArgs := toolutil.StringSliceArg(req.Arguments, "args")
	stdin := toolutil.StringArgOr(req.Arguments, "stdin", "")

	// argv is an explicit argument array (never a shell string), so user-supplied
	// command/args cannot be interpreted by a shell.
	argv := append([]string{"run", command, "--"}, extraArgs...)

	res := runProcessGroup(ctx, "nix", argv, stdin, timeout)
	if res.startErr != nil {
		return toolutil.ErrorResult("failed to start nix",
			map[string]any{"command": command, "error": res.startErr.Error()}), nil
	}

	structured := map[string]any{
		"command":     command,
		"args":        extraArgs,
		"exit_code":   res.exitCode,
		"stdout":      res.stdout,
		"stderr":      res.stderr,
		"duration_ms": res.duration.Milliseconds(),
		"timed_out":   res.timedOut,
	}
	text := fmt.Sprintf("nix_run: %s exited %d in %dms (timed_out=%t)",
		command, res.exitCode, res.duration.Milliseconds(), res.timedOut)
	result := toolutil.Result(text, structured)
	// A non-zero exit or a timeout is a tool-level error so the caller can branch
	// on IsError without parsing the structured payload.
	if res.timedOut || res.exitCode != 0 {
		result.IsError = true
	}
	return result, nil
}

// installableRejection enforces nix_run's installable policy and returns a
// human-readable reason plus true when ref denotes a REMOTE flake source that
// must not be executed (arbitrary remote code execution). It returns "", false
// for allowed references.
//
// Rejected (default-deny remote): any URL form ("scheme://host/...") and any
// flakeref carrying a URI scheme other than the local "path:" scheme — this
// covers github:, gitlab:, sourcehut:, git+*, http:, https:, tarball+*,
// file+http*, and flake+* references.
//
// Allowed: a bare attribute ("foo"), the project's own flake (".", ".#foo"),
// local filesystem paths ("./x", "../x", "/abs/x", "path:./x"), and scheme-less
// registry aliases ("nixpkgs#hello").
func installableRejection(ref string) (string, bool) {
	trimmed := strings.TrimSpace(ref)
	if strings.Contains(trimmed, "://") {
		return "references a remote URL", true
	}
	if scheme, ok := uriScheme(trimmed); ok && scheme != "path" {
		return fmt.Sprintf("uses remote flakeref scheme %q", scheme), true
	}
	return "", false
}

// uriScheme returns the lowercased URI scheme of ref and true when ref begins
// with a "scheme:" prefix (RFC 3986: a letter followed by letters, digits, and
// the characters "+", "-", "."). A flakeref attribute selector ("#") or path
// ("." or "/") that appears before any colon yields ("", false), so registry
// aliases and local paths are not treated as schemed.
func uriScheme(ref string) (string, bool) {
	for i := 0; i < len(ref); i++ {
		c := ref[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
			continue
		case i > 0 && (c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.'):
			continue
		case c == ':' && i > 0:
			return strings.ToLower(ref[:i]), true
		default:
			return "", false
		}
	}
	return "", false
}

// nixRunTimeout resolves the timeout argument (a Go duration string or a number
// of seconds), defaulting to defaultNixRunTimeout.
func nixRunTimeout(args map[string]any) time.Duration {
	if s, ok := toolutil.StringArg(args, "timeout"); ok && s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			return d
		}
	}
	if secs, ok := toolutil.IntArg(args, "timeout"); ok && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return defaultNixRunTimeout
}
