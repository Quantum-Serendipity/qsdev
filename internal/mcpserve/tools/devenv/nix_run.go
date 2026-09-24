package devenv

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
)

// defaultNixRunTimeout bounds a nix_run invocation when the caller omits timeout.
const defaultNixRunTimeout = 30 * time.Second

// maxNixRunTimeout is the ceiling applied to a caller-supplied timeout. The
// server is long-lived and shared by every tool, so a single call must not be
// able to pin a process slot (and its output) for an arbitrary duration.
const maxNixRunTimeout = 10 * time.Minute

// maxProcOutputBytes caps how much of each of stdout and stderr is retained in
// memory and returned. Output past the cap is drained and discarded (so the
// child never blocks on a full pipe) and the stream is flagged truncated.
const maxProcOutputBytes = 1 << 20

// procWaitDelay bounds how long Wait lingers after the launched process exits
// (or is killed) for its I/O pipes to close. A descendant that escaped the
// process group (setsid) and still holds a pipe open would otherwise block Wait
// — and the tool call — indefinitely.
const procWaitDelay = 5 * time.Second

// procResult is the outcome of a process-group execution, shared by the
// platform-specific runProcessGroup implementations.
type procResult struct {
	stdout          string
	stderr          string
	stdoutTruncated bool
	stderrTruncated bool
	exitCode        int
	timedOut        bool
	duration        time.Duration
	startErr        error
}

// cappedBuffer is an io.Writer that retains at most limit bytes and silently
// discards the rest, recording that it did so. It always reports the full write
// as consumed so the copying goroutine keeps draining the child's pipe. Each
// instance has a single writer (exec's copy goroutine) and is read only after
// Wait returns, so it needs no locking.
type cappedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func newCappedBuffer(limit int) *cappedBuffer { return &cappedBuffer{limit: limit} }

// Write implements io.Writer.
func (c *cappedBuffer) Write(p []byte) (int, error) {
	if room := c.limit - c.buf.Len(); room > 0 {
		if len(p) <= room {
			c.buf.Write(p)
			return len(p), nil
		}
		c.buf.Write(p[:room])
	}
	if len(p) > 0 {
		c.truncated = true
	}
	return len(p), nil
}

// String returns the retained output.
func (c *cappedBuffer) String() string { return c.buf.String() }

// nixRunner executes `nix run <command> -- <args>` in a dedicated process group
// so that, on timeout or cancellation, the entire group (including orphaned nix
// build children) is killed rather than leaked. nix runs in projectRoot, so a
// local installable such as "." or ".#pkg" names the project's own flake.
type nixRunner struct {
	projectRoot string
}

func newNixRunner(projectRoot string) *nixRunner { return &nixRunner{projectRoot: projectRoot} }

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
	if reason, rejected := installableRejection(n.projectRoot, command); rejected {
		return toolutil.ErrorResult("rejected nix installable",
			map[string]any{"command": command, "reason": reason}), nil
	}
	if _, err := exec.LookPath("nix"); err != nil {
		return toolutil.NotConfigured("nix is not installed or not on PATH",
			map[string]any{"error": err.Error(), "remediation": "install Nix or enter the devenv shell"}), nil
	}

	timeout, clamped := nixRunTimeout(req.Arguments)
	extraArgs := toolutil.StringSliceArg(req.Arguments, "args")
	stdin := toolutil.StringArgOr(req.Arguments, "stdin", "")

	// argv is an explicit argument array (never a shell string), so user-supplied
	// command/args cannot be interpreted by a shell. command is guaranteed to be
	// a positional installable (never a flag), so nix treats everything after
	// "--" as program arguments and args can never supply the installable.
	argv := append([]string{"run", command, "--"}, extraArgs...)

	res := runProcessGroup(ctx, n.projectRoot, "nix", argv, stdin, timeout)
	if res.startErr != nil {
		return toolutil.ErrorResult("failed to start nix",
			map[string]any{"command": command, "error": res.startErr.Error()}), nil
	}

	structured := map[string]any{
		"command":          command,
		"args":             extraArgs,
		"exit_code":        res.exitCode,
		"stdout":           res.stdout,
		"stderr":           res.stderr,
		"stdout_truncated": res.stdoutTruncated,
		"stderr_truncated": res.stderrTruncated,
		"duration_ms":      res.duration.Milliseconds(),
		"timed_out":        res.timedOut,
		"timeout_ms":       timeout.Milliseconds(),
		"timeout_clamped":  clamped,
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
// human-readable reason plus true when ref must not be executed. It returns
// "", false for allowed references.
//
// Rejected: an empty ref or one starting with "-" (nix would parse it as an
// option, and the first program argument after "--" would then become the
// installable, bypassing this policy); any URL form ("scheme://host/...") and
// any flakeref carrying a URI scheme other than the local "path:" scheme —
// this covers github:, gitlab:, sourcehut:, git+*, http:, https:, tarball+*,
// file+http*, and flake+* references (arbitrary remote code execution); and
// local paths that resolve outside projectRoot.
//
// Allowed: a bare attribute ("foo"), the project's own flake (".", ".#foo"),
// local filesystem paths inside the project ("./x", "/abs/x", "path:./x"), and
// scheme-less registry aliases ("nixpkgs#hello"). Registry aliases resolve
// through the nix flake registry, so which flakes they reach is governed by the
// host's registry configuration rather than by this check.
func installableRejection(projectRoot, ref string) (string, bool) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" || strings.HasPrefix(trimmed, "-") {
		return "is empty or an option, not an installable", true
	}
	if strings.Contains(trimmed, "://") {
		return "references a remote URL", true
	}
	// A Windows drive letter (`C:\x`) is a volume, not a one-letter scheme; the
	// ref is then a local path and is confined below like any other.
	if scheme, ok := uriScheme(trimmed); ok && scheme != "path" && filepath.VolumeName(trimmed) == "" {
		return fmt.Sprintf("uses remote flakeref scheme %q", scheme), true
	}
	if p, ok := localInstallablePath(trimmed); ok {
		if _, inside := toolutil.ConfineToRoot(projectRoot, p); !inside {
			return "local flake path is outside the project root", true
		}
	}
	return "", false
}

// flakeIDRef matches nix's indirect (registry) flakeref syntax: a flake id,
// optionally followed by "/<ref-or-rev>" (e.g. "nixpkgs", "nixpkgs/nixos-24.05").
// A scheme-less ref that does not match it is parsed by nix as a filesystem
// path, whatever its first character.
var flakeIDRef = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*(/[a-zA-Z0-9@][a-zA-Z0-9_./@+-]*)?$`)

// localInstallablePath returns the filesystem path a local flakeref points at
// (without its "#attr" or "?query" suffix) and true, or "", false when ref is
// not a path reference (i.e. a registry alias). Following nix's flakeref
// parser, a ref is a path when it uses the "path:" scheme or is not of the
// registry-alias form, so "./x", "/x" and "_x/../../y" are all paths.
func localInstallablePath(ref string) (string, bool) {
	p := ref
	isPathScheme := len(p) >= len("path:") && strings.EqualFold(p[:len("path:")], "path:")
	if isPathScheme {
		p = p[len("path:"):]
	}
	if i := strings.IndexAny(p, "#?"); i >= 0 {
		p = p[:i]
	}
	if !isPathScheme && flakeIDRef.MatchString(p) {
		return "", false
	}
	if p == "" {
		p = "."
	}
	return p, true
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
// of seconds), defaulting to defaultNixRunTimeout. A requested timeout above
// maxNixRunTimeout is clamped to it; the boolean reports whether that happened.
func nixRunTimeout(args map[string]any) (time.Duration, bool) {
	d := defaultNixRunTimeout
	if s, ok := toolutil.StringArg(args, "timeout"); ok && s != "" {
		if parsed, err := time.ParseDuration(s); err == nil && parsed > 0 {
			d = parsed
		}
	} else if secs, ok := toolutil.IntArg(args, "timeout"); ok && secs > 0 {
		if int64(secs) > int64(maxNixRunTimeout/time.Second) {
			return maxNixRunTimeout, true
		}
		d = time.Duration(secs) * time.Second
	}
	if d > maxNixRunTimeout {
		return maxNixRunTimeout, true
	}
	return d, false
}
