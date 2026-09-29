package repair

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// neverAutoRepairFiles are exempt from all automatic repair actions regardless
// of merge strategy or CLI flags. Both are legitimately hand-edited by users
// (custom inputs/packages, follows overrides, keep tweaks), so silently
// regenerating them would discard intentional edits.
var neverAutoRepairFiles = map[string]bool{
	"devenv.nix":  true,
	"devenv.yaml": true,
}

// claudeMDPath is the project-relative path of the file that holds the
// section markers checked by drift's marker-integrity category.
const claudeMDPath = "CLAUDE.md"

// classifyFindings maps drift findings from a posture DriftReport into
// concrete RepairActions. The classification rules depend on the file's merge
// strategy (from genState) and the drift category.
func classifyFindings(report *drift.Report, genState types.GeneratedState, opts RepairOptions) []RepairAction {
	if report == nil {
		return nil
	}

	var actions []RepairAction

	for _, cat := range report.Categories {
		for _, f := range cat.Findings {
			action := classifyFinding(cat.Name, f, genState, opts)
			action.Severity = f.Severity
			actions = append(actions, action)
		}
	}

	return actions
}

// classifyFinding maps a single drift finding to a RepairAction.
func classifyFinding(categoryName string, f drift.Finding, genState types.GeneratedState, opts RepairOptions) RepairAction {
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
func classifyFileModification(f drift.Finding, genState types.GeneratedState, opts RepairOptions) RepairAction {
	file := f.Subject

	// A deleted file has no hand-edits to protect, so regenerate it — even one on
	// the never-auto-modify list (the exemption below guards *modification* of an
	// existing file, not recreation of a missing one).
	if f.FileStatus == types.Deleted {
		return RepairAction{
			File:        file,
			Category:    CategoryFileDrift,
			Description: fmt.Sprintf("Regenerate deleted file %s", file),
			ActionType:  ActionRegenerate,
			AutoFixable: true,
		}
	}

	// devenv.nix/devenv.yaml are NEVER auto-modified regardless of strategy or flags.
	if neverAutoRepairFiles[file] {
		return RepairAction{
			File:        file,
			Category:    CategoryFileDrift,
			Description: fmt.Sprintf("%s is never auto-modified", file),
			ActionType:  ActionSkip,
			AutoFixable: false,
		}
	}

	// --reset regenerates every other managed file regardless of strategy.
	if opts.Reset {
		return RepairAction{
			File:        file,
			Category:    CategoryFileDrift,
			Description: fmt.Sprintf("Regenerate %s (--reset)", file),
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
		// Human-edited files: only fix with --force (or --reset, handled above).
		if opts.Force {
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

// classifyHookDrift handles the "Pre-Commit Hook Drift" category. Git hooks
// are installed by the devenv shell (or qsdev update), not by writing a
// generated file, so repair reports the finding with its remediation.
func classifyHookDrift(f drift.Finding) RepairAction {
	return RepairAction{
		File:        f.Subject,
		Category:    CategoryHookDrift,
		Description: withRemediation(f),
		ActionType:  ActionSkip,
		AutoFixable: false,
	}
}

// classifyMarkerDrift handles the "Section Marker Integrity" category. Marker
// findings concern sections inside CLAUDE.md, a user-edited file, so they are
// reported with their remediation; they count as resolved when CLAUDE.md
// itself is regenerated in the same run.
func classifyMarkerDrift(f drift.Finding) RepairAction {
	return RepairAction{
		File:        f.Subject,
		Category:    CategoryMarkerDrift,
		Description: withRemediation(f),
		ActionType:  ActionSkip,
		AutoFixable: false,
		ResolvedBy:  claudeMDPath,
	}
}

// withRemediation formats a finding's description followed by its
// remediation, when it has one.
func withRemediation(f drift.Finding) string {
	if f.Remediation == "" {
		return f.Description
	}
	return f.Description + "; " + f.Remediation
}

// classifyVersionDrift handles the "Version Drift" category.
func classifyVersionDrift(f drift.Finding) RepairAction {
	return RepairAction{
		File:        f.Subject,
		Category:    CategoryEnvDrift,
		Description: "Run 'qsdev update' to update configs",
		ActionType:  ActionSkip,
		AutoFixable: false,
	}
}

// classifyToolAvailability handles the "Tool Availability" category.
func classifyToolAvailability(f drift.Finding) RepairAction {
	return RepairAction{
		File:        f.Subject,
		Category:    CategoryToolMissing,
		Description: "Install missing tool binary",
		ActionType:  ActionSkip,
		AutoFixable: false,
	}
}

// classifyLockfileDrift handles the "Lock File Drift" category.
func classifyLockfileDrift(f drift.Finding) RepairAction {
	return RepairAction{
		File:        f.Subject,
		Category:    CategoryEnvDrift,
		Description: "Run package manager install",
		ActionType:  ActionSkip,
		AutoFixable: false,
	}
}
