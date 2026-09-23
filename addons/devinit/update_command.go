package devinit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
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

// Stage names shown in progress output and the update summary.
const (
	stageSelfUpdate   = "Self-update"
	stageConfigRegen  = "Config regeneration"
	stageDevenvInputs = "Devenv inputs"
)

// overwriteModifiedFlag is the `update` flag that discards user edits to
// managed config files.
const overwriteModifiedFlag = "--overwrite-modified"

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
	DryRun bool
	// Force reinstalls the latest binary even when already up to date. It
	// does not affect config regeneration.
	Force bool
	// OverwriteModified regenerates managed config files even when the user
	// modified or deleted them, discarding those edits.
	OverwriteModified bool
	// AllowDowngrade lets an older binary regenerate configs last written by
	// a newer one.
	AllowDowngrade bool
	SelfOnly       bool
	ConfigsOnly    bool
	DepsOnly       bool
	Check          bool
	Changelog      bool
	SkipContainer  bool
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

When stage 1 installs a new binary, stage 2 runs in that new binary so the
project gets the new release's templates. Outside a qsdev project, a plain
'update' only updates the binary.

Config files you have edited are merged or left alone; use
--overwrite-modified to replace them with freshly generated versions.

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
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Reinstall the latest binary even if already up to date (does not overwrite project configs)")
	cmd.Flags().BoolVar(&opts.OverwriteModified, strings.TrimPrefix(overwriteModifiedFlag, "--"), false, "Overwrite managed config files you have modified or deleted, discarding your edits")
	cmd.Flags().BoolVar(&opts.AllowDowngrade, "allow-downgrade", false, "Regenerate configs even if this binary is older than the one that last generated them")
	cmd.Flags().BoolVar(&opts.SelfOnly, "self-only", false, "Only update the binary")
	cmd.Flags().BoolVar(&opts.ConfigsOnly, "configs-only", false, "Only regenerate config files")
	cmd.Flags().BoolVar(&opts.DepsOnly, "deps-only", false, "Only update devenv inputs")
	cmd.Flags().BoolVar(&opts.Check, "check", false, "Check for updates without installing")
	cmd.Flags().BoolVar(&opts.Changelog, "changelog", false, "Show release notes (use with --check)")
	cmd.Flags().BoolVar(&opts.SkipContainer, "skip-container", false, "Skip generating Gateway container config for hookless frameworks")
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

// stageProgress prints "[n/total] message" progress lines.
type stageProgress struct {
	cmd   *cobra.Command
	n     int
	total int
}

func (p *stageProgress) next(msg string) {
	p.n++
	fmt.Fprintf(p.cmd.OutOrStdout(), "[%d/%d] %s\n", p.n, p.total, msg)
}

func runFullUpdate(cmd *cobra.Command, opts FullUpdateOptions) error {
	// Determine which stages to run based on flags.
	runSelf := !opts.ConfigsOnly && !opts.DepsOnly
	runConfigs := !opts.SelfOnly && !opts.DepsOnly
	runDeps := !opts.SelfOnly && !opts.ConfigsOnly

	if opts.Force && !runSelf {
		// --force used to also overwrite edited configs; say so rather than
		// silently changing what an existing script does.
		fmt.Fprintf(cmd.ErrOrStderr(), "Note: --force only reinstalls the binary and has no effect here; use %s to replace edited config files.\n", overwriteModifiedFlag)
	}

	progress := &stageProgress{cmd: cmd}
	for _, active := range []bool{runSelf, runConfigs, runDeps} {
		if active {
			progress.total++
		}
	}

	// Resolve the running binary before a self-update can replace it; once
	// replaced, the OS may report the path of the renamed backup instead.
	exePath := currentExecutable()

	var results []StageResult
	binaryReplaced := false
	if runSelf {
		progress.next("Checking for binary updates...")
		res := runSelfUpdateStage(cmd, opts)
		binaryReplaced = res.Status == StageSuccess && !opts.DryRun
		if binaryReplaced && !runConfigs {
			res.Message += fmt.Sprintf(". Run '%s update --configs-only' in active projects to apply the new templates.", branding.Get().AppName)
		}
		results = append(results, res)
	}

	results = append(results, runProjectStages(cmd, opts, progress, runConfigs, runDeps, binaryReplaced, exePath)...)

	return printStageSummary(cmd, results)
}

// runProjectStages runs the config-regeneration and devenv-input stages.
func runProjectStages(
	cmd *cobra.Command,
	opts FullUpdateOptions,
	progress *stageProgress,
	runConfigs, runDeps, binaryReplaced bool,
	exePath string,
) []StageResult {
	var results []StageResult

	// A plain `update` is also how the binary is upgraded from any directory.
	// Outside a qsdev project the project stages do not apply, so skip them
	// instead of failing. An explicit --configs-only/--deps-only still runs.
	notProject := !opts.ConfigsOnly && !opts.DepsOnly && !inQsdevProject()
	skipNotProject := func(name string) StageResult {
		return StageResult{Name: name, Status: StageSkipped, Message: fmt.Sprintf("not a %s project", branding.Get().AppName)}
	}

	if runConfigs {
		progress.next("Regenerating project configs...")
		switch {
		case notProject:
			results = append(results, skipNotProject(stageConfigRegen))
		case binaryReplaced:
			// This process still carries the old binary's templates; let the
			// freshly installed binary regenerate the configs.
			results = append(results, runConfigStageInBinary(cmd, exePath, opts))
		default:
			results = append(results, runConfigUpdateStage(cmd, opts))
		}
	}

	if runDeps {
		progress.next("Updating devenv inputs...")
		if notProject {
			results = append(results, skipNotProject(stageDevenvInputs))
		} else {
			results = append(results, runDevenvInputStage(cmd, opts))
		}
	}

	return results
}

// printStageSummary prints the per-stage summary and returns an error if any
// stage failed.
func printStageSummary(cmd *cobra.Command, results []StageResult) error {
	w := cmd.OutOrStdout()
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

// inQsdevProject reports whether the working directory is a qsdev project:
// it holds the shared project config or saved answers. A project config
// without local answers still counts, so the config stage reports how to set
// the project up instead of claiming it is not a project. When the project
// root cannot be determined it returns true so the project stages run and
// report the underlying error.
func inQsdevProject() bool {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return true
	}
	for _, p := range []string{filepath.Join(projectRoot, branding.Get().ConfigFile), answersPath(projectRoot)} {
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			return true
		}
	}
	return false
}

// currentExecutable returns the resolved path of the running binary, or ""
// if it cannot be determined.
func currentExecutable() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}

// configStageArgs builds the arguments that re-run only the config stage in
// another qsdev binary, forwarding the flags that affect it.
func configStageArgs(opts FullUpdateOptions) []string {
	args := []string{"update", "--configs-only"}
	if opts.OverwriteModified {
		args = append(args, overwriteModifiedFlag)
	}
	if opts.AllowDowngrade {
		args = append(args, "--allow-downgrade")
	}
	if opts.SkipContainer {
		args = append(args, "--skip-container")
	}
	return args
}

// runConfigStageInBinary regenerates configs by running the binary at
// exePath (the freshly installed one) with `update --configs-only`.
func runConfigStageInBinary(cmd *cobra.Command, exePath string, opts FullUpdateOptions) StageResult {
	rerun := fmt.Sprintf("re-run '%s update --configs-only' to apply the new templates", branding.Get().AppName)
	if exePath == "" {
		return StageResult{Name: stageConfigRegen, Status: StageSkipped, Message: "could not locate the updated binary; " + rerun}
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	child := exec.CommandContext(ctx, exePath, configStageArgs(opts)...)
	child.Stdin = cmd.InOrStdin()
	child.Stdout = cmd.OutOrStdout()
	child.Stderr = cmd.ErrOrStderr()

	err := child.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return StageResult{Name: stageConfigRegen, Status: StageSuccess, Message: "configs regenerated by the updated binary"}
	case errors.As(err, &exitErr):
		wrapped := fmt.Errorf("updated binary failed to regenerate configs: %w", err)
		return StageResult{Name: stageConfigRegen, Status: StageFailed, Message: wrapped.Error(), Err: wrapped}
	default:
		return StageResult{
			Name:    stageConfigRegen,
			Status:  StageSkipped,
			Message: fmt.Sprintf("could not run the updated binary (%v); %s", err, rerun),
		}
	}
}

func runSelfUpdateStage(cmd *cobra.Command, opts FullUpdateOptions) StageResult {
	currentVersion := strings.TrimPrefix(version.Info().Version, "v")

	if currentVersion == "" || currentVersion == "dev" || currentVersion == "(devel)" {
		return StageResult{
			Name:    stageSelfUpdate,
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
		release, err = selfupdate.ResolveForcedUpdate(ctx, cfg, currentVersion)
	} else {
		release, err = selfupdate.CheckForUpdate(ctx, cfg, currentVersion)
	}
	if errors.Is(err, selfupdate.ErrDowngrade) {
		// --force also forces config regeneration; refusing to downgrade the
		// binary is not a failure of the update as a whole.
		return StageResult{
			Name:    "Self-update",
			Status:  StageSkipped,
			Message: err.Error(),
		}
	}
	if err != nil {
		return StageResult{
			Name:    stageSelfUpdate,
			Status:  StageFailed,
			Message: err.Error(),
			Err:     err,
		}
	}

	if release == nil {
		return StageResult{
			Name:    stageSelfUpdate,
			Status:  StageUpToDate,
			Message: fmt.Sprintf("v%s is the latest", currentVersion),
		}
	}

	if opts.DryRun {
		return StageResult{
			Name:    stageSelfUpdate,
			Status:  StageSuccess,
			Message: fmt.Sprintf("would update v%s → v%s", currentVersion, release.Version),
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "  Updating v%s → v%s...\n", currentVersion, release.Version)
	if err := selfupdate.DoUpdate(ctx, cfg, release); err != nil {
		return StageResult{
			Name:    stageSelfUpdate,
			Status:  StageFailed,
			Message: err.Error(),
			Err:     err,
		}
	}

	return StageResult{
		Name:    stageSelfUpdate,
		Status:  StageSuccess,
		Message: fmt.Sprintf("v%s → v%s", currentVersion, release.Version),
	}
}

func runConfigUpdateStage(cmd *cobra.Command, opts FullUpdateOptions) StageResult {
	err := runUpdate(cmd, UpdateOptions{
		Force:          opts.OverwriteModified,
		AllowDowngrade: opts.AllowDowngrade,
		OverwriteFlag:  overwriteModifiedFlag,
		DryRun:         opts.DryRun,
		SkipContainer:  opts.SkipContainer,
	})
	if err != nil {
		msg := err.Error()
		var ratchet *qsdevconfig.RatchetWarning
		if errors.As(err, &ratchet) {
			app := branding.Get().AppName
			msg = fmt.Sprintf("%s %s is older than %s, which last generated this project's files; update %s or pass --allow-downgrade",
				app, ratchet.CurrentVersion, ratchet.LastRunVersion, app)
		}
		return StageResult{
			Name:    stageConfigRegen,
			Status:  StageFailed,
			Message: msg,
			Err:     err,
		}
	}
	msg := "configs regenerated"
	if opts.DryRun {
		msg = "dry-run, configs previewed"
	}
	return StageResult{
		Name:    stageConfigRegen,
		Status:  StageSuccess,
		Message: msg,
	}
}

func runDevenvInputStage(cmd *cobra.Command, opts FullUpdateOptions) StageResult {
	if _, err := exec.LookPath("devenv"); err != nil {
		return StageResult{
			Name:    stageDevenvInputs,
			Status:  StageSkipped,
			Message: "devenv not installed",
		}
	}

	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return StageResult{
			Name:    stageDevenvInputs,
			Status:  StageFailed,
			Message: err.Error(),
			Err:     err,
		}
	}

	// A claude-only project owns its devenv environment; qsdev must not bump
	// its lock file.
	answers, err := loadAnswersOrEmpty(projectRoot)
	if err != nil {
		wrapped := fmt.Errorf("loading saved answers: %w", err)
		return StageResult{Name: stageDevenvInputs, Status: StageFailed, Message: wrapped.Error(), Err: wrapped}
	}
	if scopeFromAnswers(answers).ClaudeOnly {
		return StageResult{
			Name:    stageDevenvInputs,
			Status:  StageSkipped,
			Message: "claude-only project, devenv is not managed by " + branding.Get().AppName,
		}
	}

	if opts.DryRun {
		return StageResult{
			Name:    stageDevenvInputs,
			Status:  StageSkipped,
			Message: "dry-run, would run: devenv update",
		}
	}

	devenvCmd := exec.Command("devenv", "update")
	devenvCmd.Dir = projectRoot
	devenvCmd.Stdout = cmd.OutOrStdout()
	devenvCmd.Stderr = cmd.ErrOrStderr()

	if err := devenvCmd.Run(); err != nil {
		return StageResult{
			Name:    stageDevenvInputs,
			Status:  StageFailed,
			Message: err.Error(),
			Err:     err,
		}
	}

	return StageResult{
		Name:    stageDevenvInputs,
		Status:  StageSuccess,
		Message: "devenv inputs updated",
	}
}
