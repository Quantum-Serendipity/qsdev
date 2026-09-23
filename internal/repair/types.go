package repair

import "github.com/Quantum-Serendipity/qsdev/internal/posture/drift"

// RepairActionType indicates what kind of repair action should be taken.
type RepairActionType int

const (
	// ActionRegenerate means the file should be regenerated from fresh content.
	ActionRegenerate RepairActionType = iota
	// ActionSkip means the finding requires manual intervention or a different command.
	ActionSkip
)

// RepairCategory classifies the type of issue found during repair analysis.
type RepairCategory string

const (
	CategoryFileDrift     RepairCategory = "file-drift"
	CategoryConfigCorrupt RepairCategory = "config-corrupt"
	CategoryToolMissing   RepairCategory = "tool-missing"
	CategoryEnvDrift      RepairCategory = "env-drift"
	CategoryHookDrift     RepairCategory = "hook-drift"
	CategoryMarkerDrift   RepairCategory = "marker-drift"
)

// RepairOptions configures the behavior of a repair run.
type RepairOptions struct {
	DryRun     bool
	Force      bool
	Reset      bool
	TargetFile string
}

// RepairAction describes a single repair action to take (or skip) for a file.
type RepairAction struct {
	File        string
	Category    RepairCategory
	Description string
	BackupPath  string
	ActionType  RepairActionType
	AutoFixable bool
	// Severity is the severity of the drift finding this action came from
	// (empty for actions synthesized by --reset).
	Severity drift.Severity
	// ResolvedBy names a file whose successful regeneration in the same run
	// also resolves this finding (e.g. CLAUDE.md for section-marker drift).
	ResolvedBy string
	Error      error
}

// RepairResult holds the outcome of a repair run, partitioned into fixed,
// skipped, and failed actions.
type RepairResult struct {
	Fixed   []RepairAction
	Skipped []RepairAction
	Failed  []RepairAction
}

// ExitCode returns 2 if any action failed, 1 if an actionable finding was
// skipped (it still needs manual attention), or 0 otherwise. Skipped
// informational findings — expected edits to user-owned files, version or
// environment notes — do not affect the exit code.
func (r *RepairResult) ExitCode() int {
	if len(r.Failed) > 0 {
		return 2
	}
	for _, a := range r.Skipped {
		if a.Severity != drift.Info {
			return 1
		}
	}
	return 0
}
