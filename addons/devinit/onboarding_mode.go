package devinit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// OnboardingMode represents the detected mode for the init command.
type OnboardingMode int

const (
	// ModeCreate indicates a fresh project with no existing configuration.
	ModeCreate OnboardingMode = iota
	// ModeJoin indicates a project with .qsdev.yaml but no local state.
	ModeJoin
	// ModeUpdate indicates the binary version has changed since last init.
	ModeUpdate
	// ModeRepair indicates drifted or corrupt state that needs fixing.
	ModeRepair
)

// String returns the mode as a human-readable string.
func (m OnboardingMode) String() string {
	switch m {
	case ModeCreate:
		return "create"
	case ModeJoin:
		return "join"
	case ModeUpdate:
		return "update"
	case ModeRepair:
		return "repair"
	default:
		return "unknown"
	}
}

// ModeDetectionResult holds the outcome of onboarding mode detection.
type ModeDetectionResult struct {
	Mode         OnboardingMode
	Explanation  string
	AlreadySetUp bool
	DriftReport  *DriftReport
}

// DriftReport describes files that have drifted from their expected state.
type DriftReport struct {
	Modified []string
	Deleted  []string
	Summary  string
}

// DetectOnboardingMode examines projectRoot to determine the correct onboarding
// mode. The decision tree is:
//  1. No .qsdev.yaml -> ModeCreate
//  2. No .devinit/.qsdev-init-state.yaml -> ModeJoin
//  3. State file unreadable -> ModeRepair
//  4. Version mismatch (non-dev) -> ModeUpdate
//  5. Files drifted (modified or deleted) -> ModeRepair
//  6. All matches -> ModeJoin with AlreadySetUp=true
func DetectOnboardingMode(projectRoot string) (*ModeDetectionResult, error) {
	// 1. Check config file exists.
	cfgFile := branding.Get().ConfigFile
	cfgPath := filepath.Join(projectRoot, cfgFile)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return &ModeDetectionResult{
			Mode:        ModeCreate,
			Explanation: fmt.Sprintf("No %s found. Starting fresh project setup.", cfgFile),
		}, nil
	}

	// 2. Check state file exists.
	stateFile := filepath.Join(projectRoot, stateFilePath())
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return &ModeDetectionResult{
			Mode:        ModeJoin,
			Explanation: fmt.Sprintf("Found %s but no local state. Setting up as new team member.", cfgFile),
		}, nil
	}

	// 3. Load state.
	existingState, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		return &ModeDetectionResult{
			Mode:        ModeRepair,
			Explanation: "State file is unreadable. Running repair.",
		}, nil
	}

	// 4. Compare versions.
	currentVersion := version.Info().Version
	storedVersion := existingState.QsdevVersion
	if storedVersion != "" && currentVersion != "" &&
		storedVersion != "dev" && currentVersion != "dev" &&
		storedVersion != currentVersion {
		return &ModeDetectionResult{
			Mode:        ModeUpdate,
			Explanation: fmt.Sprintf("%s updated from %s to %s.", branding.Get().AppName, storedVersion, currentVersion),
		}, nil
	}

	// 5. Check for drifted files.
	modStatus := state.CheckModified(existingState, projectRoot)
	var modified, deleted []string
	for path, fs := range modStatus {
		switch fs.Status {
		case types.Modified:
			modified = append(modified, path)
		case types.Deleted:
			deleted = append(deleted, path)
		}
	}

	if len(modified)+len(deleted) > 0 {
		var parts []string
		if len(modified) > 0 {
			parts = append(parts, fmt.Sprintf("%d modified", len(modified)))
		}
		if len(deleted) > 0 {
			parts = append(parts, fmt.Sprintf("%d deleted", len(deleted)))
		}
		summary := fmt.Sprintf("Drift detected: %s.", strings.Join(parts, ", "))

		return &ModeDetectionResult{
			Mode:        ModeRepair,
			Explanation: summary,
			DriftReport: &DriftReport{
				Modified: modified,
				Deleted:  deleted,
				Summary:  summary,
			},
		}, nil
	}

	// 6. All files match.
	return &ModeDetectionResult{
		Mode:         ModeJoin,
		AlreadySetUp: true,
		Explanation:  "Project is already set up. Nothing to do.",
	}, nil
}

// overrideMode parses a mode string and returns a forced ModeDetectionResult.
// Valid values are "create", "join", "update", "repair".
func overrideMode(modeStr string, projectRoot string) (*ModeDetectionResult, error) {
	var mode OnboardingMode
	switch strings.ToLower(modeStr) {
	case "create":
		mode = ModeCreate
	case "join":
		mode = ModeJoin
	case "update":
		mode = ModeUpdate
	case "repair":
		mode = ModeRepair
	default:
		return nil, fmt.Errorf("invalid mode %q; valid values: create, join, update, repair", modeStr)
	}

	return &ModeDetectionResult{
		Mode:        mode,
		Explanation: fmt.Sprintf("Mode overridden to %q.", modeStr),
	}, nil
}
