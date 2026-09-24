package ecosystem

import (
	"path"
	"strings"
)

// ExtraDirectory is the Extras key under which a module's Detect records the
// project-relative, slash-separated directory holding the ecosystem's project
// when it is not the repository root, such as the web UI of a Go service in
// frontend/. Unset means the repository root.
const ExtraDirectory = "directory"

// CleanProjectDir validates a project-relative directory for ExtraDirectory.
// It returns the cleaned, slash-separated path, or "" for the project root
// and for any value that could not safely be used: an absolute path, one
// that climbs out of the project (".."), or one with characters that
// ShellSafeDirs rejects (the directory is embedded in shell commands and Nix
// strings, and the value may come from a hand-edited answers file).
func CleanProjectDir(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ""
	}
	clean := path.Clean(dir)
	if clean == "." || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return ""
	}
	if len(ShellSafeDirs([]string{clean})) == 0 {
		return ""
	}
	return clean
}

// Directory returns the validated ExtraDirectory of c ("" for the project
// root; see CleanProjectDir).
func (c ModuleConfig) Directory() string {
	return CleanProjectDir(c.Extra(ExtraDirectory, ""))
}

// InDirectory returns the project-relative, slash-separated path of p, a path
// relative to the ecosystem's project directory (see Directory).
func (c ModuleConfig) InDirectory(p string) string {
	if dir := c.Directory(); dir != "" {
		return path.Join(dir, p)
	}
	return p
}

// InDirectoryCommand returns the shell command cmd run from the ecosystem's
// project directory. The cd runs in a subshell so that commands joined after
// it (a task script, a CI step) still start at the repository root.
func (c ModuleConfig) InDirectoryCommand(cmd string) string {
	if dir := c.Directory(); dir != "" {
		return "(cd " + dir + " && " + cmd + ")"
	}
	return cmd
}
