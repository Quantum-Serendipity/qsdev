package generate

import (
	"bytes"
	"fmt"
	"os"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// FileAction represents the outcome of processing a single generated file.
type FileAction int

const (
	ActionCreated FileAction = iota
	ActionUpdated
	ActionSkipped
	ActionFailed
)

var fileActionNames = [...]string{
	ActionCreated: "created",
	ActionUpdated: "updated",
	ActionSkipped: "skipped",
	ActionFailed:  "failed",
}

func (a FileAction) String() string {
	if int(a) >= 0 && int(a) < len(fileActionNames) {
		return fileActionNames[a]
	}
	return "unknown"
}

// FileResult records the outcome of writing a single file.
type FileResult struct {
	Path   string
	Action FileAction
	Error  error
	// PrevHash is the content hash of the file that was on disk before this
	// write (empty when the file was new or could not be read).
	PrevHash  string
	BytesSize int
	// DiskContent is the exact content on disk after a created, updated or
	// skipped (already identical) file was processed. It differs from the
	// generated content when a merge preserved user content. Nil for dry
	// runs and failures.
	DiskContent []byte
	// Mode is the permission mode written (or, for a dry run, that would be
	// written). It can be narrower than the generated mode when the existing
	// file was more restrictive.
	Mode os.FileMode
	// SidecarPath is set (relative to the project root) when a ManualMerge
	// file with local changes was left untouched and the generated content
	// was written beside it for manual merging.
	SidecarPath string
}

// WriteResult aggregates the outcomes of writing a batch of files.
type WriteResult struct {
	Files   []FileResult
	Created int
	Updated int
	Skipped int
	Failed  int
}

// Summary returns a human-readable summary of the write operation.
func (r WriteResult) Summary() string {
	s := fmt.Sprintf("Created %d, updated %d, skipped %d, failed %d",
		r.Created, r.Updated, r.Skipped, r.Failed)
	for _, fr := range r.Files {
		if fr.SidecarPath != "" {
			s += fmt.Sprintf("\n  %s has local changes; merge %s into it manually", fr.Path, fr.SidecarPath)
		}
	}
	return s
}

// HasFailures returns true if any files failed to write.
func (r WriteResult) HasFailures() bool {
	return r.Failed > 0
}

// FailedFiles returns the FileResult entries for files that failed to write.
func (r WriteResult) FailedFiles() []FileResult {
	var failed []FileResult
	for _, fr := range r.Files {
		if fr.Action == ActionFailed {
			failed = append(failed, fr)
		}
	}
	return failed
}

// SuccessfulFiles filters the generated files list down to those now present
// on disk with their intended content, ready for state.RecordFiles: files
// created or updated, and files skipped because they were already identical.
// A file kept as the user's (Skip strategy) or left for a manual merge (a
// sidecar was written) is not included: its disk content is not qsdev output.
//
// When a merge wrote bytes that differ from the generated content, the
// returned entry's Content is the bytes actually on disk (so the recorded
// hash matches the file) and BaseContent keeps the generated content as the
// three-way merge base for the next update. Each returned file carries the
// mode actually written, so recorded state agrees with the disk.
func (r WriteResult) SuccessfulFiles(allFiles []types.GeneratedFile) []types.GeneratedFile {
	done := make(map[string]FileResult, r.Created+r.Updated+r.Skipped)
	for _, fr := range r.Files {
		switch {
		case fr.Action == ActionCreated, fr.Action == ActionUpdated:
			done[fr.Path] = fr
		case fr.Action == ActionSkipped && fr.DiskContent != nil:
			done[fr.Path] = fr
		}
	}
	result := make([]types.GeneratedFile, 0, len(done))
	for _, f := range allFiles {
		fr, ok := done[f.Path]
		if !ok {
			continue
		}
		if fr.DiskContent != nil && !bytes.Equal(fr.DiskContent, f.Content) {
			f.BaseContent = f.Content
			f.Content = fr.DiskContent
		}
		if fr.Mode != 0 {
			f.Mode = fr.Mode
		}
		result = append(result, f)
	}
	return result
}

// ValidationResult records the outcome of validating a single file's content.
type ValidationResult struct {
	Path    string
	Valid   bool
	Error   error
	Skipped bool
	Warning string
}

// PipelineOptions controls the behavior of the generation pipeline.
type PipelineOptions struct {
	DryRun       bool
	SkipValidate bool
	ProjectRoot  string
	// Force records that the user explicitly asked to overwrite existing
	// configuration (e.g. --force). It lets a ManualMerge file with local
	// changes be replaced instead of getting a sidecar. It never overrides
	// Skip, whose files are only created when absent.
	Force bool
	// SectionMergeFunc overrides the merge for existing files with Strategy
	// SectionMarker. It receives the existing and new content and returns the
	// merged result. When nil, merge.SectionMarkers is used. On error the
	// file is reported as failed and left untouched.
	SectionMergeFunc func(existing, newGenerated []byte) ([]byte, error)
	// ThreeWayMergeFunc overrides the merge for existing files with Strategy
	// ThreeWayMerge. It receives the relative path, the on-disk content
	// (theirs), and the newly generated content (ours), with no recorded base
	// (this is the create path). When nil, merge.MergeOnCreate is used, which
	// preserves user-owned keys (e.g. settings.json "env"). On error the file
	// is reported as failed and left untouched.
	ThreeWayMergeFunc func(relPath string, theirs, ours []byte) ([]byte, error)
}
