package teardown

import (
	"fmt"
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
		absPath := filepath.Join(opts.ProjectRoot, fa.Path)
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
		if err := cleanSharedFile(opts.ProjectRoot, fa.Path, registry); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("cleaning %s: %w", fa.Path, err))
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

// cleanSharedFile reads a shared file, collects all qsdev section IDs from
// the registry, and removes each section using the appropriate surgery function.
func cleanSharedFile(projectRoot, relPath string, registry *toolreg.Registry) error {
	absPath := filepath.Join(projectRoot, relPath)

	content, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Collect all section IDs for this file from the registry.
	var sectionIDs []string
	for _, tool := range registry.All() {
		for _, fo := range tool.OwnedFiles {
			if fo.Path == relPath && fo.Ownership == toolreg.Shared && fo.SectionID != "" {
				sectionIDs = append(sectionIDs, fo.SectionID)
			}
		}
	}

	if len(sectionIDs) == 0 {
		return nil
	}

	// Apply removals for each section ID.
	updated := content
	for _, sid := range sectionIDs {
		var surgErr error
		updated, surgErr = applySurgeryRemove(relPath, updated, sid)
		if surgErr != nil {
			return fmt.Errorf("removing section %q: %w", sid, surgErr)
		}
	}

	return os.WriteFile(absPath, updated, fileutil.ModeReadWrite)
}

// applySurgeryRemove dispatches to the correct surgery remove function based
// on file extension/name.
func applySurgeryRemove(relPath string, content []byte, sectionID string) ([]byte, error) {
	base := filepath.Base(relPath)
	ext := filepath.Ext(relPath)

	switch {
	case base == ".mcp.json":
		return surgery.JSONRemoveMCPServer(content, sectionID)
	case ext == ".md":
		return surgery.MarkdownRemoveSection(content, sectionID)
	case ext == ".nix":
		return surgery.NixRemoveSection(content, sectionID)
	case strings.HasSuffix(base, "settings.json"):
		return surgery.MarkdownRemoveSection(content, sectionID)
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
