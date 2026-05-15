package devinit

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/internal/repair"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func repairCmd() *cobra.Command {
	var opts repair.RepairOptions
	cmd := &cobra.Command{
		Use:   "repair",
		Short: "Fix corrupted or drifted qsdev-managed files",
		Long: `Detects and fixes issues with qsdev-managed configuration files.

Conservative by default: only fixes unambiguously safe issues (machine-owned
files with strategy overwrite or library-managed). User-edited files with
section-marker, three-way-merge, or manual-merge strategies are skipped
unless --force is given.

Use --force for aggressive repair, --reset to regenerate everything.
The devenv.nix file is never auto-modified regardless of flags.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRepairCommand(cmd, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview what would be fixed without making changes")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Fix files even when user modifications detected (backup first)")
	cmd.Flags().StringVar(&opts.TargetFile, "file", "", "Repair a specific file only")
	cmd.Flags().BoolVar(&opts.Reset, "reset", false, "Regenerate all files from saved answers (nuclear option)")
	return cmd
}

func runRepairCommand(cmd *cobra.Command, opts repair.RepairOptions) error {
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining project root: %w", err)
	}

	// Load answers.
	answers, err := loadAnswersOrEmpty(projectRoot)
	if err != nil {
		return err
	}
	answers.Detected = detect.Detect(projectRoot)
	answers.ProjectRoot = projectRoot

	// Load state.
	stateFile := filepath.Join(projectRoot, statePath)
	existingState, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	// Build enabled tools map.
	enabledTools := make(map[string]bool)
	for k, v := range existingState.EnabledTools {
		enabledTools[k] = v
	}

	// Detect drift.
	driftReport := posture.DetectDrift(projectRoot, existingState, enabledTools)

	// If reset or there are findings, regenerate fresh files.
	freshFiles := make(map[string]types.GeneratedFile)
	if opts.Reset || driftReport.TotalFindings > 0 {
		freshFiles = regenerateFreshFiles(answers)
	}

	// Run repair.
	result, updatedState, err := repair.Repair(projectRoot, opts, existingState, freshFiles, driftReport)
	if err != nil {
		return err
	}

	// Save updated state.
	if !opts.DryRun && len(result.Fixed) > 0 {
		updatedState.QsdevVersion = version.Info().Version
		updatedState.LastRun = time.Now().UTC()
		if err := state.SaveStateToFile(stateFile, *updatedState); err != nil {
			return fmt.Errorf("saving state: %w", err)
		}
	}

	// Print summary.
	w := cmd.OutOrStdout()
	if len(result.Fixed) > 0 {
		if opts.DryRun {
			fmt.Fprintf(w, "Would fix (%d):\n", len(result.Fixed))
		} else {
			fmt.Fprintf(w, "Fixed (%d):\n", len(result.Fixed))
		}
		for _, a := range result.Fixed {
			fmt.Fprintf(w, "  [fix] %s — %s\n", a.File, a.Description)
		}
	}
	if len(result.Skipped) > 0 {
		fmt.Fprintf(w, "Skipped (%d):\n", len(result.Skipped))
		for _, a := range result.Skipped {
			fmt.Fprintf(w, "  [skip] %s — %s\n", a.File, a.Description)
		}
	}
	if len(result.Failed) > 0 {
		fmt.Fprintf(w, "Failed (%d):\n", len(result.Failed))
		for _, a := range result.Failed {
			errMsg := ""
			if a.Error != nil {
				errMsg = ": " + a.Error.Error()
			}
			fmt.Fprintf(w, "  [fail] %s — %s%s\n", a.File, a.Description, errMsg)
		}
	}
	if driftReport.TotalFindings == 0 {
		fmt.Fprintln(w, "No issues found. Project is healthy.")
	}

	exitCode := result.ExitCode()
	if exitCode != 0 {
		return fmt.Errorf("repair completed with exit code %d", exitCode)
	}
	return nil
}

// regenerateFreshFiles runs both generators to produce a map of path to fresh
// GeneratedFile. Errors from individual generators are silently ignored so that
// the repair can still proceed with whatever files were successfully generated.
func regenerateFreshFiles(answers types.WizardAnswers) map[string]types.GeneratedFile {
	freshFiles := make(map[string]types.GeneratedFile)
	registry := ecosystem.DefaultRegistry()

	// Generate devenv files (unless claude-only mode).
	if answers.MergeMode != "claude-only" {
		gen := devenv.NewDevenvGenerator(registry, devenv.WithProfileRegistry(profile.DefaultProfileRegistry()))
		files, err := gen.Generate(answers)
		if err == nil {
			for _, f := range files {
				freshFiles[f.Path] = f
			}
		}
	}

	// Generate Claude Code files.
	if answers.ClaudeCode {
		gen := claudecode.NewClaudeCodeGenerator(registry, claudecode.Config{})
		files, err := gen.Generate(answers)
		if err == nil {
			for _, f := range files {
				freshFiles[f.Path] = f
			}
		}
	}

	return freshFiles
}
