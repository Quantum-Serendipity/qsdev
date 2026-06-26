package mcpserve

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Project-root resolution environment variables, consulted in this order.
const (
	envProjectRoot     = "QSDEV_PROJECT_ROOT"
	envGdevProjectRoot = "GDEV_PROJECT_ROOT"
)

// fallbackMarkers are weaker project-root markers used only when no .qsdev.yaml
// is found while walking up the tree. Order is irrelevant: any one of them
// present in a directory makes that directory a candidate root.
var fallbackMarkers = []string{".git", "go.mod", "package.json"}

// ResolveOptions carries every input the project-root resolver may consult. The
// function-valued fields are injectable so the resolver is fully unit-testable
// without touching the real environment or working directory.
type ResolveOptions struct {
	// FlagRoot is the value of an explicit --project-root flag (empty if unset).
	FlagRoot string
	// RootsDirs are directories reported by the MCP roots protocol. This list is
	// often empty: many clients do not implement roots, or return an
	// "unsupported method" (-32601) error, which the caller should treat as an
	// empty list rather than a failure.
	RootsDirs []string
	// Getenv reads environment variables; defaults to os.Getenv when nil.
	Getenv func(string) string
	// Getwd reports the current working directory; defaults to os.Getwd when nil.
	Getwd func() (string, error)
}

// ResolveProjectRoot determines the project root for an MCP session.
//
// Precedence for choosing the START directory:
//
//  1. --project-root flag, when set. An explicitly-passed flag is a deliberate
//     operator override and therefore wins over every auto-detection source.
//  2. The first MCP roots-protocol directory, when any were reported.
//  3. The QSDEV_PROJECT_ROOT environment variable, then GDEV_PROJECT_ROOT.
//  4. The current working directory (os.Getwd).
//
// From the chosen start directory the resolver walks UP the tree to the nearest
// directory containing .qsdev.yaml (the definitive qsdev project marker). If no
// .qsdev.yaml is found, it walks up again looking for a weaker marker (.git,
// go.mod, package.json). If nothing matches, the absolute start directory is
// returned so callers always receive a usable root.
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
		getwd = os.Getwd
	}

	start, err := pickStartDir(o.FlagRoot, o.RootsDirs, getenv, getwd)
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path for %q: %w", start, err)
	}

	if root, ok := walkUpForFile(abs, configFileName()); ok {
		return root, nil
	}
	if root, ok := walkUpForAny(abs, fallbackMarkers); ok {
		return root, nil
	}
	return abs, nil
}

// pickStartDir applies the start-directory precedence chain.
func pickStartDir(flagRoot string, rootsDirs []string, getenv func(string) string, getwd func() (string, error)) (string, error) {
	if flagRoot != "" {
		return flagRoot, nil
	}
	for _, d := range rootsDirs {
		if d != "" {
			return d, nil
		}
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

// configFileName returns the qsdev project marker filename (.qsdev.yaml),
// sourced from branding so it tracks any rebrand rather than being hardcoded.
func configFileName() string {
	if cf := branding.Get().ConfigFile; cf != "" {
		return cf
	}
	return ".qsdev.yaml"
}

// walkUpForFile walks from dir toward the filesystem root, returning the first
// directory that directly contains a regular file named name. It shares the
// traversal logic with the rest of qsdev via logging.WalkUp, supplying its own
// "regular file named name" marker predicate.
func walkUpForFile(dir, name string) (string, bool) {
	return logging.WalkUp(dir, func(d string) bool {
		return regularFileExists(filepath.Join(d, name))
	})
}

// walkUpForAny walks from dir toward the filesystem root, returning the first
// directory that contains any of the given markers (file or directory). Like
// walkUpForFile it delegates traversal to logging.WalkUp and keeps its own
// "any marker present" predicate.
func walkUpForAny(dir string, markers []string) (string, bool) {
	return logging.WalkUp(dir, func(d string) bool {
		for _, m := range markers {
			if pathExists(filepath.Join(d, m)) {
				return true
			}
		}
		return false
	})
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
