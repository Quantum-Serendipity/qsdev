package projectctx

import (
	"io/fs"
	"os"
	"path/filepath"
)

// ProbeBoundary returns the directory whose content belongs to the project
// the working directory is in, as a version probe must see it: the nearest
// repository toplevel at or above the working directory, or the working
// directory itself outside a repository. It needs no qsdev marker, since a
// repository not yet initialised is project content too. It returns "" when
// that directory is the home directory or above it (run from ~, from a
// dotfiles repository at ~, or from /), which is no project, or when the
// working directory is unknown.
func ProbeBoundary() string {
	wd, err := WorkingDir()
	if err != nil {
		return ""
	}
	home, err := HomeDir()
	if err != nil {
		home = ""
	}
	return probeBoundary(wd, home)
}

// probeBoundary is ProbeBoundary for the working directory wd and the home
// directory home ("" when unknown). The repository walk has the same device
// ceiling as Resolve; the home walk does not, since the home directory may be
// a mount point below the boundary.
func probeBoundary(wd, home string) string {
	wd = filepath.Clean(wd)
	wdInfo, err := stat(wd)
	if err != nil {
		return wd
	}
	root, rootInfo := wd, wdInfo
	walk(wd, wdInfo, false, func(dir string, info fs.FileInfo) bool {
		if !isGitTop(dir) {
			return false
		}
		root, rootInfo = dir, info
		return true
	})
	if home == "" {
		return root
	}
	home = filepath.Clean(home)
	homeInfo, err := stat(home)
	if err != nil {
		return root
	}
	holdsHome := false
	walk(home, homeInfo, false, func(_ string, info fs.FileInfo) bool {
		holdsHome = os.SameFile(info, rootInfo)
		return holdsHome
	})
	if holdsHome {
		return ""
	}
	return root
}
