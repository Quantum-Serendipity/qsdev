package repair

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/posture"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// classifyFindings maps drift findings from a posture DriftReport into
// concrete RepairActions. The classification rules depend on the file's merge
// strategy (from genState) and the drift category.
func classifyFindings(report *posture.DriftReport, genState types.GeneratedState, opts RepairOptions) []RepairAction {
	if report == nil {
		return nil
	}

	var actions []RepairAction

	for _, cat := range report.Categories {
		for _, f := range cat.Findings {
			action := classifyFinding(cat.Name, f, genState, opts)
			actions = append(actions, action)
		}
	}

	return actions
}

// classifyFinding maps a single drift finding to a RepairAction.
func classifyFinding(categoryName string, f posture.DriftFinding, genState types.GeneratedState, opts RepairOptions) RepairAction {
	switch categoryName {
	case "File Modification":
		return classifyFileModification(f, genState, opts)
	case "Pre-Commit Hook Drift":
		return classifyHookDrift(f)
	case "Section Marker Integrity":
		return classifyMarkerDrift(f)
	case "Version Drift":
		return classifyVersionDrift(f)
	case "Tool Availability":
		return classifyToolAvailability(f)
	case "Lock File Drift":
		return classifyLockfileDrift(f)
	default:
		return RepairAction{
			File:        f.Subject,
			Category:    RepairCategory(categoryName),
			Description: f.Description,
			ActionType:  ActionSkip,
			AutoFixable: false,
		}
	}
}

// classifyFileModification handles the "File Modification" drift category.
func classifyFileModification(f posture.DriftFinding, genState types.GeneratedState, opts RepairOptions) RepairAction {
	file := f.Subject

	// devenv.nix is NEVER auto-modified regardless of strategy or flags.
	if file == "devenv.nix" {
		return RepairAction{
			File:        file,
			Category:    CategoryFileDrift,
			Description: "devenv.nix is never auto-modified",
			ActionType:  ActionSkip,
			AutoFixable: false,
		}
	}

	// Check if this is a deleted file.
	isDeleted := strings.Contains(f.Description, "has been deleted")

	if isDeleted {
		return RepairAction{
			File:        file,
			Category:    CategoryFileDrift,
			Description: fmt.Sprintf("Regenerate deleted file %s", file),
			ActionType:  ActionRegenerate,
			AutoFixable: true,
		}
	}

	// Look up the file's merge strategy.
	fileState, found := genState.Files[file]
	if !found {
		return RepairAction{
			File:        file,
			Category:    CategoryFileDrift,
			Description: f.Description,
			ActionType:  ActionSkip,
			AutoFixable: false,
		}
	}

	switch fileState.Strategy {
	case types.Overwrite, types.LibraryManaged:
		// Machine-owned files can be safely regenerated.
		return RepairAction{
			File:        file,
			Category:    CategoryFileDrift,
			Description: fmt.Sprintf("Regenerate machine-owned file %s (strategy: %s)", file, fileState.Strategy),
			ActionType:  ActionRegenerate,
			AutoFixable: true,
		}

	case types.SectionMarker, types.ThreeWayMerge, types.ManualMerge:
		// Human-edited files: only fix with --force or --reset.
		if opts.Force || opts.Reset {
			return RepairAction{
				File:        file,
				Category:    CategoryFileDrift,
				Description: fmt.Sprintf("Regenerate user-edited file %s (forced, strategy: %s)", file, fileState.Strategy),
				ActionType:  ActionRegenerate,
				AutoFixable: true,
			}
		}
		return RepairAction{
			File:        file,
			Category:    CategoryFileDrift,
			Description: fmt.Sprintf("User-edited file %s modified (strategy: %s); use --force to overwrite", file, fileState.Strategy),
			ActionType:  ActionSkip,
			AutoFixable: false,
		}

	default:
		// Unknown or other strategies: skip by default.
		return RepairAction{
			File:        file,
			Category:    CategoryFileDrift,
			Description: f.Description,
			ActionType:  ActionSkip,
			AutoFixable: false,
		}
	}
}

// classifyHookDrift handles the "Pre-Commit Hook Drift" category.
func classifyHookDrift(f posture.DriftFinding) RepairAction {
	return RepairAction{
		File:        f.Subject,
		Category:    CategoryHookDrift,
		Description: fmt.Sprintf("Reinstall hook: %s", f.Description),
		ActionType:  ActionReinstall,
		AutoFixable: true,
	}
}

// classifyMarkerDrift handles the "Section Marker Integrity" category.
func classifyMarkerDrift(f posture.DriftFinding) RepairAction {
	return RepairAction{
		File:        f.Subject,
		Category:    CategoryMarkerDrift,
		Description: fmt.Sprintf("Regenerate to fix marker: %s", f.Description),
		ActionType:  ActionRegenerate,
		AutoFixable: true,
	}
}

// classifyVersionDrift handles the "Version Drift" category.
func classifyVersionDrift(f posture.DriftFinding) RepairAction {
	return RepairAction{
		File:        f.Subject,
		Category:    CategoryEnvDrift,
		Description: "Run 'gdev update' to update configs",
		ActionType:  ActionSkip,
		AutoFixable: false,
	}
}

// classifyToolAvailability handles the "Tool Availability" category.
func classifyToolAvailability(f posture.DriftFinding) RepairAction {
	return RepairAction{
		File:        f.Subject,
		Category:    CategoryToolMissing,
		Description: "Install missing tool binary",
		ActionType:  ActionSkip,
		AutoFixable: false,
	}
}

// classifyLockfileDrift handles the "Lock File Drift" category.
func classifyLockfileDrift(f posture.DriftFinding) RepairAction {
	return RepairAction{
		File:        f.Subject,
		Category:    CategoryEnvDrift,
		Description: "Run package manager install",
		ActionType:  ActionSkip,
		AutoFixable: false,
	}
}
