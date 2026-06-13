package generate

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// WriteFiles writes the given generated files to disk according to the
// pipeline options. It validates content, creates directories, and writes
// files atomically. Processing continues on failure; all results are returned.
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

	registry := NewValidatorRegistry()
	var result WriteResult

	for _, file := range files {
		fr := FileResult{
			Path:      file.Path,
			BytesSize: len(file.Content),
		}

		// Reject absolute paths.
		if filepath.IsAbs(file.Path) {
			fr.Action = ActionFailed
			fr.Error = fmt.Errorf("file path must be relative: %q", file.Path)
			result.Files = append(result.Files, fr)
			result.Failed++
			continue
		}

		// Reject path traversal.
		if containsPathTraversal(file.Path) {
			fr.Action = ActionFailed
			fr.Error = fmt.Errorf("file path contains path traversal: %q", file.Path)
			result.Files = append(result.Files, fr)
			result.Failed++
			continue
		}

		// Validate content unless skipped.
		if !opts.SkipValidate && !file.SkipValidation {
			vr := registry.Validate(file.Path, file.Content)
			if !vr.Valid && !vr.Skipped {
				fr.Action = ActionFailed
				fr.Error = fmt.Errorf("validation failed for %s: %w", file.Path, vr.Error)
				result.Files = append(result.Files, fr)
				result.Failed++
				continue
			}
		}

		fullPath := filepath.Join(opts.ProjectRoot, file.Path)

		// Verify the resolved path doesn't escape the project root via symlinks.
		if dir := filepath.Dir(fullPath); dir != resolvedRoot {
			resolved, resolveErr := filepath.EvalSymlinks(dir)
			if resolveErr == nil && !pathHasPrefix(resolved, resolvedRoot) {
				fr.Action = ActionFailed
				fr.Error = fmt.Errorf("resolved path escapes project root: %q", file.Path)
				result.Files = append(result.Files, fr)
				result.Failed++
				continue
			}
		}

		// Determine action: created vs updated.
		_, statErr := os.Stat(fullPath)
		if statErr == nil {
			fr.Action = ActionUpdated
		} else {
			fr.Action = ActionCreated
		}

		// Apply default mode.
		mode := file.Mode
		if mode == 0 {
			mode = fileutil.ModeReadWrite
		}

		if opts.DryRun {
			result.Files = append(result.Files, fr)
			switch fr.Action {
			case ActionCreated:
				result.Created++
			case ActionUpdated:
				result.Updated++
			}
			continue
		}

		// Apply section-marker merge when the file already exists and the
		// caller provided a merge function.
		contentToWrite := file.Content
		if file.Strategy == types.SectionMarker && statErr == nil && opts.SectionMergeFunc != nil {
			if existingContent, readErr := os.ReadFile(fullPath); readErr == nil {
				if merged, mergeErr := opts.SectionMergeFunc(existingContent, file.Content); mergeErr == nil {
					contentToWrite = merged
					fr.BytesSize = len(contentToWrite)
				}
			}
		}

		// Write atomically.
		if err := fileutil.WriteFileAtomic(fullPath, contentToWrite, mode); err != nil {
			fr.Action = ActionFailed
			fr.Error = fmt.Errorf("write %s: %w", file.Path, err)
			slog.Warn("file write failed", "path", file.Path, "error", err)
			result.Files = append(result.Files, fr)
			result.Failed++
			continue
		}
		slog.Debug("file written", "path", file.Path, "action", fr.Action, "bytes", fr.BytesSize)

		result.Files = append(result.Files, fr)
		switch fr.Action {
		case ActionCreated:
			result.Created++
		case ActionUpdated:
			result.Updated++
		}
	}

	return result, nil
}

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
