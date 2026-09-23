package teardown

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/surgery"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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
		removeEmptyParents(opts.ProjectRoot, fa.Path)
	}

	// Clean shared files by removing qsdev sections.
	for _, fa := range plan.Clean {
		changed, err := cleanSharedFile(opts.ProjectRoot, fa, registry)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("cleaning %s: %w", fa.Path, err))
			continue
		}
		if changed == cleanRemoved {
			result.Removed = append(result.Removed, FileAction{Path: fa.Path, Reason: "nothing left after removing " + branding.Get().AppName + " content"})
			continue
		}
		if changed == cleanUnchanged {
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

// cleanOutcome is what cleaning a shared file did to it.
type cleanOutcome int

const (
	cleanUnchanged cleanOutcome = iota // no qsdev content found
	cleanRewritten                     // qsdev content removed, user content kept
	cleanRemoved                       // only qsdev content was there: file deleted
)

// cleanSharedFile removes qsdev-owned content from a shared file and reports
// what changed. A symlinked shared file (e.g. CLAUDE.md pointing at AGENTS.md)
// is rewritten at its target, which must stay inside the project; the write is
// atomic and keeps the target's permissions. A Markdown file left with only
// qsdev's scaffold held no user content and is deleted.
func cleanSharedFile(projectRoot string, fa FileAction, registry *toolreg.Registry) (cleanOutcome, error) {
	absPath, err := resolveTrackedPath(projectRoot, fa.Path)
	if err != nil {
		return cleanUnchanged, err
	}
	target, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cleanUnchanged, nil
		}
		return cleanUnchanged, fmt.Errorf("resolving %s: %w", fa.Path, err)
	}
	if err := checkWithinRoot(projectRoot, target); err != nil {
		return cleanUnchanged, err
	}

	info, err := os.Stat(target)
	if err != nil {
		return cleanUnchanged, fmt.Errorf("stat %s: %w", fa.Path, err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return cleanUnchanged, fmt.Errorf("reading %s: %w", fa.Path, err)
	}

	updated, err := removeQsdevContent(fa, content, registry)
	if err != nil {
		return cleanUnchanged, err
	}
	if bytes.Equal(updated, content) {
		return cleanUnchanged, nil
	}

	if isMarkdown(fa.Path) && onlyScaffold(fa.Path, updated) {
		if err := os.Remove(target); err != nil {
			return cleanUnchanged, fmt.Errorf("removing %s: %w", fa.Path, err)
		}
		return cleanRemoved, nil
	}
	if err := fileutil.WriteFileAtomic(target, updated, info.Mode().Perm()); err != nil {
		return cleanUnchanged, fmt.Errorf("writing %s: %w", fa.Path, err)
	}
	return cleanRewritten, nil
}

// onlyScaffold reports whether a cleaned Markdown file holds nothing but
// whitespace and the title qsdev writes above its generated block when it
// creates the file ("# CLAUDE.md"), i.e. no user content.
func onlyScaffold(relPath string, content []byte) bool {
	rest := bytes.TrimSpace(content)
	return len(rest) == 0 || string(rest) == "# "+filepath.Base(relPath)
}

// isMarkdown reports whether relPath is cleaned with the Markdown surgery.
func isMarkdown(relPath string) bool {
	base := filepath.Base(relPath)
	return base != ".mcp.json" && filepath.Ext(relPath) != ".nix" && !strings.HasSuffix(base, "settings.json")
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
	// The generated block itself (CLAUDE.md's BEGIN/END GENERATED SECTION,
	// which also lists the skills teardown deletes) is qsdev's too.
	if isMarkdown(fa.Path) {
		var err error
		if updated, err = merge.RemoveSection(updated); err != nil {
			return nil, fmt.Errorf("removing generated section: %w", err)
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

// removeEmptyParents removes the directories above the removed file relPath
// that it leaves empty (e.g. .claude/skills/<skill>/), stopping at the
// project root.
func removeEmptyParents(projectRoot, relPath string) {
	for dir := path.Dir(relPath); dir != "." && dir != "/"; dir = path.Dir(dir) {
		entries, err := os.ReadDir(filepath.Join(projectRoot, filepath.FromSlash(dir)))
		if err != nil || len(entries) > 0 {
			return
		}
		removeIfEmpty(filepath.Join(projectRoot, filepath.FromSlash(dir)))
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
