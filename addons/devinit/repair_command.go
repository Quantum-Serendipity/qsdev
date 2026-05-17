package devinit

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
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
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	// Load answers.
	answers, err := loadAnswersOrEmpty(projectRoot)
	if err != nil {
		return err
	}
	answers.Detected = detect.Detect(projectRoot)
	answers.ProjectRoot = projectRoot

	// Load state.
	stateFile := filepath.Join(projectRoot, stateFilePath())
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
	driftReport := drift.Detect(projectRoot, existingState, enabledTools)

	// If reset or there are findings, regenerate fresh files.
	freshFiles := make(map[string]types.GeneratedFile)
	if opts.Reset || driftReport.TotalFindings > 0 {
		var genErr error
		freshFiles, genErr = regenerateFreshFiles(answers)
		if genErr != nil {
			return genErr
		}
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
		return &repairExitErr{code: exitCode, failed: len(result.Failed), skipped: len(result.Skipped)}
	}
	return nil
}

type repairExitErr struct {
	code    int
	failed  int
	skipped int
}

func (e *repairExitErr) Error() string {
	return fmt.Sprintf("repair: %d failed, %d skipped", e.failed, e.skipped)
}

func (e *repairExitErr) ExitCode() int { return e.code }

// regenerateFreshFiles runs both generators to produce a map of path to fresh
// GeneratedFile. Individual generator failures are logged as warnings so repair
// can proceed with partial results. Returns an error only if ALL generators fail.
func regenerateFreshFiles(answers types.WizardAnswers) (map[string]types.GeneratedFile, error) {
	freshFiles := make(map[string]types.GeneratedFile)
	registry := ecosystem.DefaultRegistry()
	var genErrors int

	// Generate devenv files (unless claude-only mode).
	if answers.MergeMode != "claude-only" {
		gen := devenv.NewDevenvGenerator(registry, devenv.WithProfileRegistry(profile.DefaultProfileRegistry()))
		files, err := gen.Generate(answers)
		if err != nil {
			slog.Warn("devenv generator failed during repair", "error", err)
			genErrors++
		} else {
			for _, f := range files {
				freshFiles[f.Path] = f
			}
		}
	}

	// Generate Claude Code files.
	if answers.ClaudeCode {
		gen := claudecode.NewClaudeCodeGenerator(registry, claudecode.Config{})
		files, err := gen.Generate(answers)
		if err != nil {
			slog.Warn("claudecode generator failed during repair", "error", err)
			genErrors++
		} else {
			for _, f := range files {
				freshFiles[f.Path] = f
			}
		}
	}

	if len(freshFiles) == 0 && genErrors > 0 {
		return nil, fmt.Errorf("all generators failed; cannot determine expected file state for repair")
	}

	return freshFiles, nil
}
