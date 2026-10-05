package logging

import (
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// AutomatedLogSubdir is the sub-directory of a log tier that holds the session
// logs of machine-invoked commands (hooks and MCP servers; see Config.Automated). It is pruned under its
// own cap, so a burst of hook invocations never evicts user-command logs.
const AutomatedLogSubdir = "automated"

// GlobalLogDir returns the global log directory for non-project operations:
// the log-dir override when set, else the logs directory in the per-user state
// directory (projectctx.UserDirs). It returns "" when neither is available:
// falling back to a fixed, predictable path under the shared temp directory
// would let another local user pre-create or symlink it.
func GlobalLogDir() string {
	if dir := os.Getenv(branding.Get().EnvLogDirVar); dir != "" {
		return dir
	}
	dirs, err := projectctx.UserDirs()
	if err != nil {
		return ""
	}
	return dirs.Logs()
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
