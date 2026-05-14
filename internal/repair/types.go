package repair

// RepairActionType indicates what kind of repair action should be taken.
type RepairActionType int

const (
	// ActionRegenerate means the file should be regenerated from fresh content.
	ActionRegenerate RepairActionType = iota
	// ActionReinstall means the component (e.g. hook) should be reinstalled.
	ActionReinstall
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
	Error       error
}

// RepairResult holds the outcome of a repair run, partitioned into fixed,
// skipped, and failed actions.
type RepairResult struct {
	Fixed   []RepairAction
	Skipped []RepairAction
	Failed  []RepairAction
}

// ExitCode returns 0 if everything was fixed, 1 if some actions were skipped,
// or 2 if any action failed.
func (r *RepairResult) ExitCode() int {
	if len(r.Failed) > 0 {
		return 2
	}
	if len(r.Skipped) > 0 {
		return 1
	}
	return 0
}
