package devinit

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/selfupdate"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// StageStatus represents the outcome of an update stage.
type StageStatus int

const (
	// StageSuccess indicates the stage completed successfully with changes.
	StageSuccess StageStatus = iota
	// StageSkipped indicates the stage was not executed.
	StageSkipped
	// StageFailed indicates the stage encountered an error.
	StageFailed
	// StageUpToDate indicates no changes were needed.
	StageUpToDate
)

func (s StageStatus) String() string {
	switch s {
	case StageSuccess:
		return "updated"
	case StageSkipped:
		return "skipped"
	case StageFailed:
		return "failed"
	case StageUpToDate:
		return "up-to-date"
	default:
		return "unknown"
	}
}

// StageResult captures the outcome of a single update stage.
type StageResult struct {
	Name    string
	Status  StageStatus
	Message string
	Err     error
}

// FullUpdateOptions holds configuration for the coordinated update command.
type FullUpdateOptions struct {
	DryRun      bool
	Force       bool
	SelfOnly    bool
	ConfigsOnly bool
	DepsOnly    bool
	Check       bool
	Changelog   bool
}

func updateCmd() *cobra.Command {
	var opts FullUpdateOptions
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update qsdev binary, project configs, and devenv inputs",
		Long: `Perform a coordinated update in up to three stages:

  Stage 1: Update qsdev binary to the latest version
  Stage 2: Regenerate project configuration files from saved answers
  Stage 3: Update devenv flake inputs (nix packages)

Use --check to see available updates without installing.
Use stage-specific flags to run only one stage.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.Check {
				return runCheckOnly(cmd, opts)
			}
			return runFullUpdate(cmd, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Preview changes without writing")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Force update even if already up to date")
	cmd.Flags().BoolVar(&opts.SelfOnly, "self-only", false, "Only update the binary")
	cmd.Flags().BoolVar(&opts.ConfigsOnly, "configs-only", false, "Only regenerate config files")
	cmd.Flags().BoolVar(&opts.DepsOnly, "deps-only", false, "Only update devenv inputs")
	cmd.Flags().BoolVar(&opts.Check, "check", false, "Check for updates without installing")
	cmd.Flags().BoolVar(&opts.Changelog, "changelog", false, "Show release notes (use with --check)")
	return cmd
}

func runCheckOnly(cmd *cobra.Command, opts FullUpdateOptions) error {
	w := cmd.OutOrStdout()

	currentVersion := strings.TrimPrefix(version.Info().Version, "v")
	if currentVersion == "" || currentVersion == "dev" || currentVersion == "(devel)" {
		fmt.Fprintln(w, "Dev build — version check skipped.")
		return nil
	}

	cfg := selfupdate.DefaultConfig()
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	var release *selfupdate.Release
	var err error
	if opts.Force {
		release, err = selfupdate.FetchLatestRelease(ctx, cfg)
	} else {
		release, err = selfupdate.CheckForUpdate(ctx, cfg, currentVersion)
	}
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	if release == nil {
		fmt.Fprintf(w, "Already up to date (v%s).\n", currentVersion)
		return nil
	}

	fmt.Fprintf(w, "Update available: v%s → v%s\n", currentVersion, release.Version)
	fmt.Fprintf(w, "Release: %s\n", release.URL)

	if opts.Changelog && release.Body != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, release.Body)
	}

	fmt.Fprintf(w, "\nRun '%s update' to install.\n", branding.Get().AppName)
	return nil
}

func runFullUpdate(cmd *cobra.Command, opts FullUpdateOptions) error {
	w := cmd.OutOrStdout()

	// Determine which stages to run based on flags.
	runSelf := !opts.ConfigsOnly && !opts.DepsOnly
	runConfigs := !opts.SelfOnly && !opts.DepsOnly
	runDeps := !opts.SelfOnly && !opts.ConfigsOnly

	// Count active stages for progress display.
	total := 0
	if runSelf {
		total++
	}
	if runConfigs {
		total++
	}
	if runDeps {
		total++
	}

	var results []StageResult
	stage := 0

	if runSelf {
		stage++
		fmt.Fprintf(w, "[%d/%d] Checking for binary updates...\n", stage, total)
		results = append(results, runSelfUpdateStage(cmd, opts))
	}

	if runConfigs {
		stage++
		fmt.Fprintf(w, "[%d/%d] Regenerating project configs...\n", stage, total)
		results = append(results, runConfigUpdateStage(cmd, opts))
	}

	if runDeps {
		stage++
		fmt.Fprintf(w, "[%d/%d] Updating devenv inputs...\n", stage, total)
		results = append(results, runDevenvInputStage(cmd, opts))
	}

	// Print summary.
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Update Summary:")
	var hadFailure bool
	for _, r := range results {
		indicator := "  ✓"
		switch r.Status {
		case StageFailed:
			indicator = "  ✗"
			hadFailure = true
		case StageSkipped:
			indicator = "  -"
		}
		msg := r.Message
		if msg == "" {
			msg = r.Status.String()
		}
		fmt.Fprintf(w, "%s %s: %s\n", indicator, r.Name, msg)
	}

	if hadFailure {
		return fmt.Errorf("one or more update stages failed")
	}
	return nil
}

func runSelfUpdateStage(cmd *cobra.Command, opts FullUpdateOptions) StageResult {
	currentVersion := strings.TrimPrefix(version.Info().Version, "v")

	if currentVersion == "" || currentVersion == "dev" || currentVersion == "(devel)" {
		return StageResult{
			Name:    "Self-update",
			Status:  StageSkipped,
			Message: "dev build, skipping version check",
		}
	}

	cfg := selfupdate.DefaultConfig()
	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	var release *selfupdate.Release
	var err error
	if opts.Force {
		release, err = selfupdate.FetchLatestRelease(ctx, cfg)
	} else {
		release, err = selfupdate.CheckForUpdate(ctx, cfg, currentVersion)
	}
	if err != nil {
		return StageResult{
			Name:    "Self-update",
			Status:  StageFailed,
			Message: err.Error(),
			Err:     err,
		}
	}

	if release == nil {
		return StageResult{
			Name:    "Self-update",
			Status:  StageUpToDate,
			Message: fmt.Sprintf("v%s is the latest", currentVersion),
		}
	}

	if opts.DryRun {
		return StageResult{
			Name:    "Self-update",
			Status:  StageSuccess,
			Message: fmt.Sprintf("would update v%s → v%s", currentVersion, release.Version),
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "  Updating v%s → v%s...\n", currentVersion, release.Version)
	if err := selfupdate.DoUpdate(ctx, cfg, release); err != nil {
		return StageResult{
			Name:    "Self-update",
			Status:  StageFailed,
			Message: err.Error(),
			Err:     err,
		}
	}

	msg := fmt.Sprintf("v%s → v%s", currentVersion, release.Version)
	if isMinorBump(currentVersion, release.Version) {
		msg += fmt.Sprintf(". Run '%s update --configs-only' in active projects to regenerate configs.", branding.Get().AppName)
	}

	return StageResult{
		Name:    "Self-update",
		Status:  StageSuccess,
		Message: msg,
	}
}

func runConfigUpdateStage(cmd *cobra.Command, opts FullUpdateOptions) StageResult {
	err := runUpdate(cmd, UpdateOptions{
		Force:  opts.Force,
		DryRun: opts.DryRun,
	})
	if err != nil {
		return StageResult{
			Name:    "Config regeneration",
			Status:  StageFailed,
			Message: err.Error(),
			Err:     err,
		}
	}
	return StageResult{
		Name:    "Config regeneration",
		Status:  StageSuccess,
		Message: "configs regenerated",
	}
}

func runDevenvInputStage(cmd *cobra.Command, opts FullUpdateOptions) StageResult {
	if _, err := exec.LookPath("devenv"); err != nil {
		return StageResult{
			Name:    "Devenv inputs",
			Status:  StageSkipped,
			Message: "devenv not installed",
		}
	}

	if opts.DryRun {
		return StageResult{
			Name:    "Devenv inputs",
			Status:  StageSkipped,
			Message: "dry-run, would run: devenv update",
		}
	}

	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return StageResult{
			Name:    "Devenv inputs",
			Status:  StageFailed,
			Message: err.Error(),
			Err:     err,
		}
	}

	devenvCmd := exec.Command("devenv", "update")
	devenvCmd.Dir = projectRoot
	devenvCmd.Stdout = cmd.OutOrStdout()
	devenvCmd.Stderr = cmd.ErrOrStderr()

	if err := devenvCmd.Run(); err != nil {
		return StageResult{
			Name:    "Devenv inputs",
			Status:  StageFailed,
			Message: err.Error(),
			Err:     err,
		}
	}

	return StageResult{
		Name:    "Devenv inputs",
		Status:  StageSuccess,
		Message: "devenv inputs updated",
	}
}

// isMinorBump returns true if the major or minor version component changed.
func isMinorBump(oldVer, newVer string) bool {
	oldMajor, oldMinor := parseMajorMinor(oldVer)
	newMajor, newMinor := parseMajorMinor(newVer)
	return oldMajor != newMajor || oldMinor != newMinor
}

func parseMajorMinor(v string) (int, int) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	major, minor := 0, 0
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return major, minor
}
