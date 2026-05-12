package generate

import "fmt"

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
	if int(a) < len(fileActionNames) {
		return fileActionNames[a]
	}
	return "unknown"
}

// FileResult records the outcome of writing a single file.
type FileResult struct {
	Path      string
	Action    FileAction
	Error     error
	PrevHash  string
	BytesSize int
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
}
