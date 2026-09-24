package ecosystem

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// ProjectScanDepth bounds how many directory levels below the project root
// detection looks for ecosystem files (the root is depth 0). It covers the
// common monorepo layouts (infra/main.tf, charts/api/Chart.yaml,
// infra/modules/aws/main.tf) without walking a whole repository.
const ProjectScanDepth = 3

// skippedScanDirs never hold the project's own configuration: dependency and
// provider caches and vendored third-party trees. Hidden directories (.git,
// .terraform, .devenv, ...) are skipped as well.
var skippedScanDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
}

// WalkProjectFiles calls fn with the path of every regular file at most
// maxDepth directory levels below root, skipping hidden directories and
// dependency trees. It is best-effort: unreadable entries are skipped, and fn
// returning false stops the walk.
func WalkProjectFiles(root string, maxDepth int, fn func(path string) bool) {
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // best-effort scan
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			name := d.Name()
			if skippedScanDirs[name] || strings.HasPrefix(name, ".") || scanDepth(root, path) > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && !fn(path) {
			return filepath.SkipAll
		}
		return nil
	})
}

// scanDepth returns how many directory levels dir is below root.
func scanDepth(root, dir string) int {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return ProjectScanDepth + 1
	}
	return strings.Count(filepath.ToSlash(rel), "/") + 1
}

// ProjectDirsWith returns the directories, relative to root in slash form
// ("." for root), holding at least one file for which match returns true,
// sorted, within ProjectScanDepth levels.
func ProjectDirsWith(root string, match func(name string) bool) []string {
	seen := make(map[string]bool)
	var dirs []string
	WalkProjectFiles(root, ProjectScanDepth, func(path string) bool {
		if !match(filepath.Base(path)) {
			return true
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return true
		}
		rel = filepath.ToSlash(rel)
		if !seen[rel] {
			seen[rel] = true
			dirs = append(dirs, rel)
		}
		return true
	})
	slices.Sort(dirs)
	return dirs
}

// shellSafeDirPattern matches relative directory names that are safe to embed
// unquoted in a hook's shell command and in a comma-separated Extras value.
var shellSafeDirPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

// ShellSafeDirs returns the sorted, deduplicated directories from dirs whose
// names match shellSafeDirPattern and do not start with "-" (which a command
// would read as a flag). Other directories are left out rather than quoted.
func ShellSafeDirs(dirs []string) []string {
	var out []string
	for _, d := range dirs {
		if shellSafeDirPattern.MatchString(d) && !strings.HasPrefix(d, "-") {
			out = append(out, d)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
