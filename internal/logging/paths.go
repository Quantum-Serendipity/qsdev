package logging

import (
	"os"
	"path/filepath"
)

// GlobalLogDir returns the global log directory for non-project operations.
func GlobalLogDir() string {
	if dir := os.Getenv("QSDEV_LOG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".qsdev", "logs")
}

// ProjectLogDir returns the project-scoped log directory.
func ProjectLogDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".qsdev", "logs")
}

// ResolveLogDir determines which log tier to use based on context.
func ResolveLogDir(projectRoot string, projectScoped bool) string {
	if projectScoped && projectRoot != "" {
		return ProjectLogDir(projectRoot)
	}
	return GlobalLogDir()
}

// DetectProjectRoot walks up from the current directory looking for
// .qsdev.yaml or .qsdev/ to identify a qsdev project root.
// Returns "" if not inside a project.
func DetectProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if fileExists(filepath.Join(dir, ".qsdev.yaml")) {
			return dir
		}
		if dirExists(filepath.Join(dir, ".qsdev")) {
			return dir
		}
		if dirExists(filepath.Join(dir, ".devinit")) {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// IsProjectScopedCommand returns true for commands that should write to
// the project log tier rather than the global tier.
func IsProjectScopedCommand(commandPath string) bool {
	globalCommands := map[string]bool{
		"qsdev self-update": true,
		"qsdev version":     true,
		"qsdev report":      true,
		"qsdev report bug":  true,
		"qsdev logs":        true,
		"qsdev completion":  true,
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
