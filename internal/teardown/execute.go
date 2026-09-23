package teardown

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/surgery"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// Execute carries out the teardown plan, removing files, cleaning shared
// files, and removing directories. If opts.DryRun is true, no changes are
// made; the result describes what would have been done.
func Execute(plan *TeardownPlan, opts TeardownOptions, registry *toolreg.Registry) (*TeardownResult, error) {
	result := &TeardownResult{
		Preserved: plan.Preserve,
	}

	if opts.DryRun {
		result.Removed = plan.Remove
		result.Cleaned = plan.Clean
		result.DirsRemoved = plan.Dirs
		return result, nil
	}

	// Remove exclusive files.
	for _, fa := range plan.Remove {
		absPath, err := resolveTrackedPath(opts.ProjectRoot, fa.Path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("refusing to remove %s: %w", fa.Path, err))
			continue
		}
		if err := os.Remove(absPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			result.Errors = append(result.Errors, fmt.Errorf("removing %s: %w", fa.Path, err))
			continue
		}
		result.Removed = append(result.Removed, fa)
	}

	// Clean shared files by removing qsdev sections.
	for _, fa := range plan.Clean {
		changed, err := cleanSharedFile(opts.ProjectRoot, fa, registry)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("cleaning %s: %w", fa.Path, err))
			continue
		}
		if !changed {
			result.Preserved = append(result.Preserved, FileAction{
				Path:   fa.Path,
				Reason: "no qsdev-managed content found to remove",
			})
			continue
		}
		result.Cleaned = append(result.Cleaned, fa)
	}

	// Remove directories.
	for _, dir := range plan.Dirs {
		absPath := filepath.Join(opts.ProjectRoot, dir)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			continue
		}
		if err := os.RemoveAll(absPath); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("removing directory %s: %w", dir, err))
			continue
		}
		result.DirsRemoved = append(result.DirsRemoved, dir)
	}

	// Post-cleanup: remove directories that are now empty.
	emptyCheckDirs := []string{
		".claude/skills",
		".claude/hooks",
		".claude/agents",
		".claude",
		".github",
		".version-sentinel",
		".cosign",
	}
	for _, dir := range emptyCheckDirs {
		absPath := filepath.Join(opts.ProjectRoot, dir)
		removeIfEmpty(absPath)
	}

	return result, nil
}

// cleanSharedFile removes qsdev-owned content from a shared file and reports
// whether the file changed. A symlinked shared file (e.g. CLAUDE.md pointing
// at AGENTS.md) is rewritten at its target, which must stay inside the
// project; the write is atomic and keeps the target's permissions.
func cleanSharedFile(projectRoot string, fa FileAction, registry *toolreg.Registry) (bool, error) {
	absPath, err := resolveTrackedPath(projectRoot, fa.Path)
	if err != nil {
		return false, err
	}
	target, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("resolving %s: %w", fa.Path, err)
	}
	if err := checkWithinRoot(projectRoot, target); err != nil {
		return false, err
	}

	info, err := os.Stat(target)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", fa.Path, err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", fa.Path, err)
	}

	updated, err := removeQsdevContent(fa, content, registry)
	if err != nil {
		return false, err
	}
	if bytes.Equal(updated, content) {
		return false, nil
	}

	if err := fileutil.WriteFileAtomic(target, updated, info.Mode().Perm()); err != nil {
		return false, fmt.Errorf("writing %s: %w", fa.Path, err)
	}
	return true, nil
}

// removeQsdevContent returns content with qsdev's contributions removed.
// settings.json is structured JSON with no section markers, so it is cleaned
// against the recorded generated content; other shared files have each
// registry section removed with the surgery matching their format.
func removeQsdevContent(fa FileAction, content []byte, registry *toolreg.Registry) ([]byte, error) {
	if strings.HasSuffix(filepath.Base(fa.Path), "settings.json") {
		return stripGeneratedSettings(content, fa.BaseContent)
	}

	// Collect all section IDs for this file from the registry.
	var sectionIDs []string
	for _, tool := range registry.All() {
		for _, fo := range tool.OwnedFiles {
			if fo.Path == fa.Path && fo.Ownership == toolreg.Shared && fo.SectionID != "" {
				sectionIDs = append(sectionIDs, fo.SectionID)
			}
		}
	}

	updated := content
	for _, sid := range sectionIDs {
		var err error
		updated, err = applySurgeryRemove(fa.Path, updated, sid)
		if err != nil {
			return nil, fmt.Errorf("removing section %q: %w", sid, err)
		}
	}
	return updated, nil
}

// applySurgeryRemove dispatches to the correct marker-based surgery remove
// function based on file extension/name.
func applySurgeryRemove(relPath string, content []byte, sectionID string) ([]byte, error) {
	base := filepath.Base(relPath)
	ext := filepath.Ext(relPath)

	switch {
	case base == ".mcp.json":
		return surgery.JSONRemoveMCPServer(content, sectionID)
	case ext == ".nix":
		return surgery.NixRemoveSection(content, sectionID)
	default:
		return surgery.MarkdownRemoveSection(content, sectionID)
	}
}

// removeIfEmpty removes a directory if it exists and is empty.
func removeIfEmpty(absPath string) {
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		_ = os.Remove(absPath)
	}
}
