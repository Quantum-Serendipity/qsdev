package logging

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// AutomatedLogSubdir is the sub-directory of a log tier that holds the session
// logs of machine-invoked commands (see ClassAutomated). It is pruned under its
// own cap, so a burst of hook invocations never evicts user-command logs.
const AutomatedLogSubdir = "automated"

// GlobalLogDir returns the global log directory for non-project operations. It
// returns "" when neither the log-dir override nor the user's home directory is
// available: falling back to a fixed, predictable path under the shared temp
// directory would let another local user pre-create or symlink it.
func GlobalLogDir() string {
	b := branding.Get()
	if dir := os.Getenv(b.EnvLogDirVar); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, "."+b.AppName, "logs")
}

// ProjectLogDir returns the project-scoped log directory.
func ProjectLogDir(projectRoot string) string {
	return filepath.Join(projectRoot, "."+branding.Get().AppName, "logs")
}

// ResolveLogDir determines which log tier to use based on context.
func ResolveLogDir(projectRoot string, projectScoped bool) string {
	if projectScoped && projectRoot != "" {
		return ProjectLogDir(projectRoot)
	}
	return GlobalLogDir()
}

// WalkUp walks from startDir toward the filesystem root, returning the first
// directory for which match reports true, together with true. When no ancestor
// (nor startDir itself) matches it returns ("", false). match is invoked with
// each candidate directory from startDir upward, so callers encode their own
// project-root marker set inside it. This is the single shared traversal
// primitive used by project-root detection across packages; callers keep their
// own marker semantics by supplying the predicate.
func WalkUp(startDir string, match func(dir string) bool) (string, bool) {
	dir := startDir
	for {
		if match(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// DetectProjectRoot walks up from the current directory looking for
// .qsdev.yaml or .qsdev/ to identify a qsdev project root.
// Returns "" if not inside a project.
func DetectProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	b := branding.Get()
	root, ok := WalkUp(dir, func(d string) bool {
		return fileutil.FileExists(d, b.ConfigFile) ||
			fileutil.DirExists(d, "."+b.AppName) ||
			fileutil.DirExists(d, b.StateDir)
	})
	if !ok {
		return ""
	}
	return root
}

// CommandClass says where, if anywhere, a CLI invocation's session log is kept.
type CommandClass int

const (
	// ClassProject logs to the project tier when run inside a project, and to
	// the global tier otherwise. It is the default for user-run commands.
	ClassProject CommandClass = iota
	// ClassGlobal always logs to the global tier.
	ClassGlobal
	// ClassAutomated covers commands invoked by tooling rather than by the
	// user — Claude Code hooks, sandboxed hook execution and MCP servers, which
	// can run on every agent tool call. They log to AutomatedLogSubdir of their
	// tier, retained under a separate cap.
	ClassAutomated
	// ClassUnlogged covers invocations that leave nothing worth diagnosing —
	// shell completion (run on every TAB press), help output and log browsing —
	// and would otherwise flood the retention cap.
	ClassUnlogged
)

// commandClasses maps a command path (the words after the binary name) to its
// class. A command inherits the class of its longest listed ancestor, so
// "logs show" is unlogged via "logs"; anything unlisted is ClassProject.
var commandClasses = map[string]CommandClass{
	cobra.ShellCompRequestCmd:       ClassUnlogged,
	cobra.ShellCompNoDescRequestCmd: ClassUnlogged,
	"help":                          ClassUnlogged,
	"completion":                    ClassUnlogged,
	"logs":                          ClassUnlogged,
	// The universal MCP server initializes its own (automated) session.
	"mcp serve": ClassUnlogged,

	"self-update": ClassGlobal,
	"version":     ClassGlobal,
	"report":      ClassGlobal,

	"selfprotect":  ClassAutomated,
	"enforce":      ClassAutomated,
	"sandbox exec": ClassAutomated,
}

// ClassifyInvocation returns the CommandClass for a CLI invocation, given its
// arguments without the binary name (os.Args[1:]). The command path is the
// leading run of non-flag arguments. A help flag anywhere before a "--"
// terminator, or a bare root invocation (which prints usage or, with
// --version, the version), makes the invocation ClassUnlogged.
func ClassifyInvocation(args []string) CommandClass {
	var path []string
	inPath := true
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if arg == "-h" || arg == "--help" {
			return ClassUnlogged
		}
		if strings.HasPrefix(arg, "-") {
			inPath = false
		}
		if inPath {
			path = append(path, arg)
		}
	}
	if len(path) == 0 {
		return ClassUnlogged
	}
	for n := len(path); n > 0; n-- {
		if class, ok := commandClasses[strings.Join(path[:n], " ")]; ok {
			return class
		}
	}
	return ClassProject
}
