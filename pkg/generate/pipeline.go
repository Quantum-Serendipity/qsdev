package generate

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// SidecarSuffix is appended to a ManualMerge file's path when the generated
// content cannot be written in place and is left beside it for manual merge.
// It matches the devenv.nix update sidecar (internal/update).
const SidecarSuffix = ".new"

// WriteFiles writes the given generated files to disk according to the
// pipeline options. It validates content, creates directories, and writes
// files atomically. Processing continues on failure; all results are returned.
//
// Each file's MergeStrategy is enforced when the file already exists (see
// existingContent), so no caller can accidentally overwrite user content by
// omitting a merge option.
func WriteFiles(files []types.GeneratedFile, opts PipelineOptions) (WriteResult, error) {
	if !filepath.IsAbs(opts.ProjectRoot) {
		return WriteResult{}, fmt.Errorf("project root must be absolute: %q", opts.ProjectRoot)
	}

	info, err := os.Stat(opts.ProjectRoot)
	if err != nil {
		return WriteResult{}, fmt.Errorf("project root does not exist: %w", err)
	}
	if !info.IsDir() {
		return WriteResult{}, fmt.Errorf("project root is not a directory: %q", opts.ProjectRoot)
	}

	// Resolve the project root once for consistent symlink escape checks.
	// On Windows, EvalSymlinks may normalize casing or resolve junctions.
	resolvedRoot, err := filepath.EvalSymlinks(opts.ProjectRoot)
	if err != nil {
		return WriteResult{}, fmt.Errorf("resolving project root symlinks: %w", err)
	}

	w := &fileWriter{
		opts:         opts,
		resolvedRoot: resolvedRoot,
	}
	for _, file := range files {
		w.write(file)
	}
	return w.result, nil
}

// fileWriter carries the per-call state of WriteFiles.
type fileWriter struct {
	opts         PipelineOptions
	resolvedRoot string
	result       WriteResult

	// priorStates holds the project's recorded generation states, loaded on
	// first use by a Skip or ManualMerge decision.
	priorStates       []types.GeneratedState
	priorStatesLoaded bool
}

func (w *fileWriter) fail(fr FileResult, err error) {
	fr.Action = ActionFailed
	fr.Error = err
	w.result.Files = append(w.result.Files, fr)
	w.result.Failed++
}

func (w *fileWriter) record(fr FileResult) {
	w.result.Files = append(w.result.Files, fr)
	switch fr.Action {
	case ActionCreated:
		w.result.Created++
	case ActionUpdated:
		w.result.Updated++
	case ActionSkipped:
		w.result.Skipped++
	}
}

func (w *fileWriter) write(file types.GeneratedFile) {
	fr := FileResult{
		Path:      file.Path,
		BytesSize: len(file.Content),
	}

	if err := w.validatePath(file); err != nil {
		w.fail(fr, err)
		return
	}

	fullPath := filepath.Join(w.opts.ProjectRoot, file.Path)

	// Verify the resolved path doesn't escape the project root via symlinks.
	if err := checkWithinRoot(fullPath, w.resolvedRoot); err != nil {
		w.fail(fr, fmt.Errorf("%s: %w", file.Path, err))
		return
	}

	// Apply default mode.
	mode := file.Mode
	if mode == 0 {
		mode = fileutil.ModeReadWrite
	}

	existingInfo, statErr := os.Stat(fullPath)
	contentToWrite := file.Content
	fr.Action = ActionCreated
	if statErr == nil {
		fr.Action = ActionUpdated
		// Never widen access to an existing file: a user who restricted it
		// (e.g. settings.json holding tokens at 0600) keeps that restriction.
		mode = restrictMode(mode, existingInfo.Mode().Perm())

		decision, err := w.existingContent(file, fullPath)
		if err != nil {
			slog.Warn("existing file not written", "path", file.Path, "error", err)
			w.fail(fr, err)
			return
		}
		switch {
		case decision.sidecar:
			if err := w.writeSidecar(file, fullPath, mode, &fr); err != nil {
				w.fail(fr, err)
				return
			}
			fr.Action = ActionSkipped
			w.record(fr)
			return
		case decision.skip:
			slog.Info("existing file kept (skip strategy)", "path", file.Path)
			fr.Action = ActionSkipped
			w.record(fr)
			return
		}
		contentToWrite = decision.content
		fr.BytesSize = len(contentToWrite)
	}
	fr.Mode = mode

	if w.opts.DryRun {
		w.record(fr)
		return
	}

	// Skip byte- and mode-identical files: rewriting them only bumps the
	// mtime, which makes direnv/devenv re-evaluate on every run. The file is
	// still reported with its disk content so state records it.
	if statErr == nil {
		var identical bool
		fr.PrevHash, identical = compareOnDisk(fullPath, contentToWrite, mode)
		if identical {
			fr.Action = ActionSkipped
			fr.DiskContent = contentToWrite
			w.record(fr)
			return
		}
	}

	// Write atomically.
	if err := fileutil.WriteFileAtomic(fullPath, contentToWrite, mode); err != nil {
		slog.Warn("file write failed", "path", file.Path, "error", err)
		w.fail(fr, fmt.Errorf("write %s: %w", file.Path, err))
		return
	}
	slog.Debug("file written", "path", file.Path, "action", fr.Action, "bytes", fr.BytesSize)
	fr.DiskContent = contentToWrite
	w.record(fr)
}

// validatePath rejects absolute and traversing paths and, unless skipped,
// content that fails its validator.
func (w *fileWriter) validatePath(file types.GeneratedFile) error {
	if filepath.IsAbs(file.Path) {
		return fmt.Errorf("file path must be relative: %q", file.Path)
	}
	if containsPathTraversal(file.Path) {
		return fmt.Errorf("file path contains path traversal: %q", file.Path)
	}
	if !w.opts.SkipValidate && !file.SkipValidation {
		return ValidateContent(file.Path, file.Content)
	}
	return nil
}

// existingDecision is what to do with a generated file whose target exists.
type existingDecision struct {
	content []byte // content to write in place
	skip    bool   // keep the existing file untouched
	sidecar bool   // keep the existing file; write generated content beside it
}

// existingContent applies file.Strategy to an existing target:
//
//   - Overwrite, LibraryManaged: replace with the generated content.
//   - Skip: keep the existing file (it belongs to the user) unless it is
//     unmodified qsdev output — the generated content itself, or content
//     matching a recorded state hash — which is regenerated in place
//     (skip-if-exists; Force does not override this).
//   - ManualMerge: replace only when the existing file is known qsdev output
//     (identical content, or content matching a recorded state hash) or the
//     caller set Force; otherwise leave it and write a sidecar.
//   - SectionMarker, ThreeWayMerge: merge generated content into the existing
//     file (an empty existing file is simply replaced).
//   - Any other strategy has no merge implementation and fails rather than
//     overwrite content it cannot preserve.
func (w *fileWriter) existingContent(file types.GeneratedFile, fullPath string) (existingDecision, error) {
	switch file.Strategy {
	case types.Overwrite, types.LibraryManaged:
		return existingDecision{content: file.Content}, nil
	}

	existing, err := os.ReadFile(fullPath)
	if err != nil {
		// Cannot read the existing file — fail rather than blindly overwrite
		// content we could not inspect.
		return existingDecision{}, fmt.Errorf("reading existing %s for merge: %w", file.Path, err)
	}

	switch file.Strategy {
	case types.Skip:
		if bytes.Equal(existing, file.Content) || w.isRecordedOutput(file.Path, existing) {
			return existingDecision{content: file.Content}, nil
		}
		return existingDecision{skip: true}, nil

	case types.ManualMerge:
		if w.opts.Force || bytes.Equal(existing, file.Content) || w.isRecordedOutput(file.Path, existing) {
			return existingDecision{content: file.Content}, nil
		}
		return existingDecision{sidecar: true}, nil

	case types.SectionMarker, types.ThreeWayMerge:
		if len(bytes.TrimSpace(existing)) == 0 {
			// Empty on-disk file: nothing to preserve, write generated content.
			return existingDecision{content: file.Content}, nil
		}
		merged, err := w.mergeExisting(file, existing)
		if err != nil {
			// A merge error must NOT silently overwrite user content; surface
			// it so the file is left intact for repair.
			return existingDecision{}, fmt.Errorf("merging %s: %w", file.Path, err)
		}
		return existingDecision{content: merged}, nil

	default:
		return existingDecision{}, fmt.Errorf("merge strategy %s is not supported for existing file %s", file.Strategy, file.Path)
	}
}

// mergeExisting runs the section-marker or three-way merge for an existing
// file, preferring a caller-supplied function and defaulting to internal/merge.
func (w *fileWriter) mergeExisting(file types.GeneratedFile, existing []byte) ([]byte, error) {
	if file.Strategy == types.SectionMarker {
		if w.opts.SectionMergeFunc != nil {
			return w.opts.SectionMergeFunc(existing, file.Content)
		}
		return merge.SectionMarkersOrAppend(existing, file.Content)
	}
	if w.opts.ThreeWayMergeFunc != nil {
		return w.opts.ThreeWayMergeFunc(file.Path, existing, file.Content)
	}
	return merge.MergeOnCreate(file.Path, existing, file.Content)
}

// isRecordedOutput reports whether existing matches content qsdev recorded
// for relPath in any of the project's state files, i.e. the file on disk is
// unmodified generated output that may be regenerated in place.
func (w *fileWriter) isRecordedOutput(relPath string, existing []byte) bool {
	if !w.priorStatesLoaded {
		w.priorStatesLoaded = true
		states, err := state.LoadProjectStates(w.opts.ProjectRoot)
		if err != nil {
			slog.Warn("loading recorded state for manual-merge check", "error", err)
		}
		w.priorStates = states
	}
	return state.IsRecordedOutput(w.priorStates, relPath, existing)
}

// writeSidecar writes the generated content next to a ManualMerge file that
// has user modifications, leaving the original untouched.
func (w *fileWriter) writeSidecar(file types.GeneratedFile, fullPath string, mode os.FileMode, fr *FileResult) error {
	fr.SidecarPath = file.Path + SidecarSuffix
	slog.Warn("existing file has local changes; wrote generated version beside it for manual merge",
		"path", file.Path, "sidecar", fr.SidecarPath)
	if w.opts.DryRun {
		return nil
	}
	if err := fileutil.WriteFileAtomic(fullPath+SidecarSuffix, file.Content, mode); err != nil {
		return fmt.Errorf("write sidecar %s: %w", fr.SidecarPath, err)
	}
	return nil
}

// restrictMode returns generated with any group/other permission bits the
// existing file does not grant removed, so rewriting a file never widens who
// can access it. Owner bits come from generated (e.g. a hook stays executable).
func restrictMode(generated, existing os.FileMode) os.FileMode {
	const groupOther os.FileMode = 0o077
	return generated &^ (groupOther &^ existing)
}

// checkWithinRoot verifies that writing fullPath cannot land outside root
// through a symlink. The longest existing ancestor of fullPath is resolved
// (MkdirAll would follow a symlinked ancestor even when deeper directories do
// not exist yet), and a symlinked leaf is resolved as well. Any resolution
// error other than "does not exist" fails closed.
func checkWithinRoot(fullPath, resolvedRoot string) error {
	if li, err := os.Lstat(fullPath); err == nil && li.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(fullPath)
		if err != nil {
			return fmt.Errorf("resolving symlink: %w", err)
		}
		if !pathHasPrefix(resolved, resolvedRoot) {
			return errEscapesRoot
		}
	}

	dir := filepath.Dir(fullPath)
	for {
		resolved, err := filepath.EvalSymlinks(dir)
		if err == nil {
			if !pathHasPrefix(resolved, resolvedRoot) {
				return errEscapesRoot
			}
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("resolving %s: %w", dir, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("no existing ancestor for %s: %w", fullPath, err)
		}
		dir = parent
	}
}

// errEscapesRoot reports a write target that resolves outside the project.
var errEscapesRoot = errors.New("resolved path escapes project root")

// PreviewFiles returns a table-formatted string showing what would happen
// if the given files were written. The state parameter is used to determine
// whether existing files have been modified since last generation.
func PreviewFiles(files []types.GeneratedFile, state *types.GeneratedState, projectRoot string) string {
	if len(files) == 0 {
		return "No files to generate."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%-40s  %-10s  %-16s  %-12s  %s\n", "File", "Action", "Strategy", "Owner", "Size")
	b.WriteString(strings.Repeat("-", 90) + "\n")

	for _, file := range files {
		action := "create"
		if state != nil {
			if _, exists := state.Files[file.Path]; exists {
				action = "update"
			}
		} else {
			fullPath := filepath.Join(projectRoot, file.Path)
			if _, err := os.Stat(fullPath); err == nil {
				action = "update"
			}
		}

		size := formatSize(len(file.Content))
		owner := file.Owner
		if owner == "" {
			owner = "-"
		}
		fmt.Fprintf(&b, "%-40s  %-10s  %-16s  %-12s  %s\n", file.Path, action, file.Strategy, owner, size)
	}

	return b.String()
}

// ValidateFiles validates all the given files and returns results.
func ValidateFiles(files []types.GeneratedFile) []ValidationResult {
	registry := NewValidatorRegistry()
	results := make([]ValidationResult, 0, len(files))
	for _, file := range files {
		vr := registry.Validate(file.Path, file.Content)
		results = append(results, vr)
	}
	return results
}

// pathHasPrefix checks whether resolved is under root, accounting for
// filesystem separator boundaries and case-insensitive paths on Windows.
func pathHasPrefix(resolved, root string) bool {
	sep := string(filepath.Separator)
	a := resolved + sep
	b := root + sep
	if runtime.GOOS == "windows" {
		a = strings.ToLower(a)
		b = strings.ToLower(b)
	}
	return strings.HasPrefix(a, b)
}

// containsPathTraversal checks whether a file path contains ".." components.
func containsPathTraversal(path string) bool {
	for part := range strings.SplitSeq(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

// VerifyWritten checks that files reported as created or updated by WriteFiles
// actually exist on disk. It returns the relative paths of any missing files.
func VerifyWritten(result WriteResult, projectRoot string) []string {
	var missing []string
	for _, fr := range result.Files {
		if fr.Action != ActionCreated && fr.Action != ActionUpdated {
			continue
		}
		path := filepath.Join(projectRoot, fr.Path)
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, fr.Path)
		}
	}
	return missing
}

// formatSize returns a human-readable file size string.
func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	kb := float64(bytes) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1f KB", kb)
	}
	mb := kb / 1024
	return fmt.Sprintf("%.1f MB", mb)
}
