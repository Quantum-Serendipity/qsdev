package bwrap

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"
)

// mountDenyPaths are absolute paths that must never be bind-mounted into a
// sandbox. These are system-level sensitive files.
var mountDenyPaths = []string{
	"/etc/shadow",
	"/etc/sudoers",
	"/etc/sudoers.d",
	"/root",
}

// mountDenyHomePaths are home-relative paths that must never be bind-mounted.
// They are expanded to the current user's home directory before checking.
var mountDenyHomePaths = []string{
	".ssh",
	".gnupg",
	".aws",
	".azure",
	".config/gcloud",
	".kube",
	".docker/config.json",
	".netrc",
}

// ValidateMountPath checks that a path is safe to use as a bind-mount source
// or target inside a sandbox. It rejects relative paths, symlinks that resolve
// to a different location, and paths on the deny list.
func ValidateMountPath(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("mount path must be absolute: %q", path)
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err == nil && resolved != path {
		return fmt.Errorf("mount path %q resolves to %q via symlink; use the resolved path directly", path, resolved)
	}

	clean := filepath.Clean(path)

	for _, deny := range mountDenyPaths {
		if clean == deny || strings.HasPrefix(clean, deny+"/") {
			return fmt.Errorf("mount path %q is denied: overlaps sensitive path %q", path, deny)
		}
	}

	home := ""
	if u, err := user.Current(); err == nil {
		home = u.HomeDir
	}
	if home != "" {
		for _, rel := range mountDenyHomePaths {
			deny := filepath.Join(home, rel)
			if clean == deny || strings.HasPrefix(clean, deny+"/") {
				return fmt.Errorf("mount path %q is denied: overlaps sensitive path %q", path, deny)
			}
		}
	}

	return nil
}
