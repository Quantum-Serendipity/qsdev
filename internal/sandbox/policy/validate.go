package policy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// policyDenyPaths are absolute paths that must never appear as mount sources
// or targets in policy declarations.
var policyDenyPaths = []string{
	"/etc/shadow",
	"/etc/sudoers",
	"/etc/sudoers.d",
	"/root",
}

// policyDenyHomePaths are home-relative paths denied in mount declarations.
// They are expanded to the current user's home directory before checking.
var policyDenyHomePaths = []string{
	".ssh",
	".gnupg",
	".aws",
	".azure",
	".config/gcloud",
	".kube",
	".docker/config.json",
	".netrc",
}

// ValidateMountDecl checks that a MountDecl is safe to use in a sandbox policy.
// It rejects mounts with non-absolute paths, root filesystem sources, and
// paths on the deny list.
func ValidateMountDecl(m MountDecl) error {
	if !filepath.IsAbs(m.Source) {
		return fmt.Errorf("mount source must be absolute: %q", m.Source)
	}
	if !filepath.IsAbs(m.Target) {
		return fmt.Errorf("mount target must be absolute: %q", m.Target)
	}
	if filepath.Clean(m.Source) == "/" {
		return fmt.Errorf("mount source must not be root filesystem: %q", m.Source)
	}

	cleanSource := filepath.Clean(m.Source)
	cleanTarget := filepath.Clean(m.Target)

	for _, deny := range policyDenyPaths {
		if matchesDeny(cleanSource, deny) {
			return fmt.Errorf("mount source %q overlaps sensitive path %q", m.Source, deny)
		}
		if matchesDeny(cleanTarget, deny) {
			return fmt.Errorf("mount target %q overlaps sensitive path %q", m.Target, deny)
		}
	}

	home, _ := os.UserHomeDir()
	if home != "" {
		for _, rel := range policyDenyHomePaths {
			deny := filepath.Join(home, rel)
			if matchesDeny(cleanSource, deny) {
				return fmt.Errorf("mount source %q overlaps sensitive path %q", m.Source, deny)
			}
			if matchesDeny(cleanTarget, deny) {
				return fmt.Errorf("mount target %q overlaps sensitive path %q", m.Target, deny)
			}
		}
	}

	return nil
}

// matchesDeny reports whether path equals deny or is a child of deny.
func matchesDeny(path, deny string) bool {
	return path == deny || strings.HasPrefix(path, deny+"/")
}
