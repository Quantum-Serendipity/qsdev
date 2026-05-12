package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/fileutil"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
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
			mode = 0644
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

		// Write atomically.
		if err := fileutil.WriteFileAtomic(fullPath, file.Content, mode); err != nil {
			fr.Action = ActionFailed
			fr.Error = fmt.Errorf("write %s: %w", file.Path, err)
			result.Files = append(result.Files, fr)
			result.Failed++
			continue
		}

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
	fmt.Fprintf(&b, "%-40s  %-10s  %s\n", "File", "Action", "Size")
	b.WriteString(strings.Repeat("-", 60) + "\n")

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
		fmt.Fprintf(&b, "%-40s  %-10s  %s\n", file.Path, action, size)
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

// containsPathTraversal checks whether a file path contains ".." components.
func containsPathTraversal(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
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
