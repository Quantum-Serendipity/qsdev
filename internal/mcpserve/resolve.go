package mcpserve

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
)

// Project-root resolution environment variables, consulted in this order.
const (
	envProjectRoot     = container.EnvProjectRoot
	envGdevProjectRoot = "GDEV_PROJECT_ROOT"
)

// ResolveOptions carries every input the project-root resolver may consult. The
// function-valued fields are injectable so the resolver is fully unit-testable
// without touching the real environment or working directory.
type ResolveOptions struct {
	// FlagRoot is the value of an explicit --project-root flag (empty if unset).
	FlagRoot string
	// Getenv reads environment variables; defaults to os.Getenv when nil.
	Getenv func(string) string
	// Getwd reports the current working directory; defaults to
	// projectctx.WorkingDir when nil.
	Getwd func() (string, error)
}

// ResolveProjectRoot determines the project root the server operates within. It
// runs once at process start, before any client connects, so the root is fixed
// for the server's lifetime and shared by every session (the Guardrail policy
// and project context are derived from it). The MCP roots protocol is therefore
// not consulted; a client that needs a different workspace launches the server
// with --project-root or QSDEV_PROJECT_ROOT.
//
// Precedence for choosing the START directory:
//
//  1. --project-root flag, when set. An explicitly-passed flag is a deliberate
//     operator override and therefore wins over every auto-detection source.
//  2. The QSDEV_PROJECT_ROOT environment variable, then GDEV_PROJECT_ROOT.
//  3. The current working directory (os.Getwd).
//
// From the chosen start directory the project is resolved by
// projectctx.Resolve, the resolver every qsdev command shares: the nearest
// trusted project marker, bounded by the git toplevel and the device. Without
// a trusted marker the root is the absolute start directory, exactly as for
// the CLI, so callers always receive a usable root. The start directory must
// exist.
//
// NOTE on precedence vs. the spec's numbered list: the spec lists the flag last
// but explicitly labels it "an explicit override". Treating an explicit flag as
// highest-priority is the sensible reading and is documented here and on the
// flag itself.
func ResolveProjectRoot(o ResolveOptions) (string, error) {
	getenv := o.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	getwd := o.Getwd
	if getwd == nil {
		getwd = projectctx.WorkingDir
	}

	start, err := pickStartDir(o.FlagRoot, getenv, getwd)
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path for %q: %w", start, err)
	}

	pc, err := projectctx.Resolve(abs, projectctx.Enclosing)
	if err != nil {
		return "", err
	}
	// Root is the trusted marker's directory, or the start directory when no
	// trusted marker exists. The git toplevel is deliberately not a fallback:
	// nothing vouches for who created that .git, so another local user could
	// plant one (and a .qsdev.yaml policy) in a shared ancestor. The CLI
	// resolves the same Root, so both agree on every start directory.
	return pc.Root, nil
}

// pickStartDir applies the start-directory precedence chain.
func pickStartDir(flagRoot string, getenv func(string) string, getwd func() (string, error)) (string, error) {
	if flagRoot != "" {
		return flagRoot, nil
	}
	if v := getenv(envProjectRoot); v != "" {
		return v, nil
	}
	if v := getenv(envGdevProjectRoot); v != "" {
		return v, nil
	}
	wd, err := getwd()
	if err != nil {
		return "", fmt.Errorf("determining working directory: %w", err)
	}
	return wd, nil
}
