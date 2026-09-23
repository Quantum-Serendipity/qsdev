package generate

import (
	"bytes"
	"fmt"

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
	return fmt.Sprintf("Created %d, updated %d, skipped %d, failed %d",
		r.Created, r.Updated, r.Skipped, r.Failed)
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
// on disk with their intended content (ActionCreated, ActionUpdated, or
// ActionSkipped because the file was already identical), ready for
// state.RecordFiles.
//
// When a merge wrote bytes that differ from the generated content, the
// returned entry's Content is the bytes actually on disk (so the recorded
// hash matches the file) and BaseContent keeps the generated content as the
// three-way merge base for the next update.
func (r WriteResult) SuccessfulFiles(allFiles []types.GeneratedFile) []types.GeneratedFile {
	done := make(map[string]FileResult, r.Created+r.Updated+r.Skipped)
	for _, fr := range r.Files {
		switch fr.Action {
		case ActionCreated, ActionUpdated, ActionSkipped:
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
	// SectionMergeFunc, when non-nil, is called for files with Strategy
	// SectionMarker that already exist on disk. It receives the existing
	// and new content and returns the merged result. On error the file is
	// reported ActionFailed and left untouched on disk.
	SectionMergeFunc func(existing, newGenerated []byte) ([]byte, error)
	// ThreeWayMergeFunc, when non-nil, is called for files with Strategy
	// ThreeWayMerge that already exist on disk. It receives the relative path,
	// the on-disk content (theirs), and the newly generated content (ours),
	// with no recorded base (this is the create path). It returns the merged
	// result; on error the file is reported ActionFailed and left untouched on
	// disk. This is what preserves user-owned top-level keys (e.g.
	// settings.json "env") when init overwrites an existing, unrecorded file.
	ThreeWayMergeFunc func(relPath string, theirs, ours []byte) ([]byte, error)
}
