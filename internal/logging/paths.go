package logging

import (
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// GlobalLogDir returns the global log directory for non-project operations.
func GlobalLogDir() string {
	b := branding.Get()
	if dir := os.Getenv(b.EnvLogDirVar); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
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
		return fileExists(filepath.Join(d, b.ConfigFile)) ||
			dirExists(filepath.Join(d, "."+b.AppName)) ||
			dirExists(filepath.Join(d, b.StateDir))
	})
	if !ok {
		return ""
	}
	return root
}

// IsProjectScopedCommand returns true for commands that should write to
// the project log tier rather than the global tier.
func IsProjectScopedCommand(commandPath string) bool {
	app := branding.Get().AppName
	globalCommands := map[string]bool{
		app + " self-update": true,
		app + " version":     true,
		app + " report":      true,
		app + " report bug":  true,
		app + " logs":        true,
		app + " completion":  true,
	}

	for cmd := range globalCommands {
		if commandPath == cmd {
			return false
		}
	}
	return true
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
