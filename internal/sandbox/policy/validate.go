package policy

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
)

// ValidateMountDecl checks that a MountDecl is safe to use in a sandbox policy.
// It rejects mounts with non-absolute paths, root filesystem sources or targets,
// and paths on the deny list. Both the cleaned and symlink-resolved forms are
// checked against the deny list, consistent with bwrap's ValidateMountPath.
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
	if filepath.Clean(m.Target) == "/" {
		return fmt.Errorf("mount target must not be root filesystem: %q", m.Target)
	}

	if err := checkDenyList(m.Source, "source"); err != nil {
		return err
	}
	if err := checkDenyList(m.Target, "target"); err != nil {
		return err
	}

	return nil
}

// checkDenyList checks a single path against the shared deny list, using both
// the cleaned path and its symlink-resolved form.
func checkDenyList(path, role string) error {
	cleaned := filepath.Clean(path)

	candidates := []string{cleaned}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		if r := filepath.Clean(resolved); r != cleaned {
			candidates = append(candidates, r)
		}
	}

	for _, candidate := range candidates {
		for _, deny := range denylist.AllDenyPaths() {
			if matchesDeny(candidate, deny) {
				return fmt.Errorf("mount %s %q overlaps sensitive path %q", role, path, deny)
			}
		}
	}

	return nil
}

// matchesDeny reports whether path equals deny or is a child of deny.
func matchesDeny(path, deny string) bool {
	return path == deny || strings.HasPrefix(path, deny+"/")
}
