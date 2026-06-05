package devenv

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules" // register all modules
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// AddonDir is the project-relative directory used by the devenv addon for its
// configuration and state files.
const AddonDir = ".devenv"

// statePath returns the path to the devenv state file, using the branding app name.
func statePath() string {
	return ".devenv/." + branding.Get().AppName + "-state.yaml"
}

// validServices references the canonical service list for shell completion.
var validServices = validation.Services()

// validLanguages references the canonical core language list for shell completion.
var validLanguages = validation.CoreLanguages()

func devenvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devenv",
		Short: "Manage security-hardened devenv.sh development environments",
		Long:  "Create, update, and extend devenv.sh development environments with security hardening.",
	}

	cmd.AddCommand(
		initCmd(),
		updateCmd(),
		makeAddCmd(serviceSpec(true)),
		makeAddCmd(languageSpec(true)),
		makeAddCmd(packageSpec(true)),
		makeAddCmd(overlaySpec(true)),
		makeRemoveCmd(serviceSpec(false)),
		makeRemoveCmd(languageSpec(false)),
		makeRemoveCmd(packageSpec(false)),
		makeRemoveCmd(overlaySpec(false)),
		doctorCmd(),
		setupCmd(),
		changelogCmd(),
	)

	return cmd
}

func initCmd() *cobra.Command {
	var (
		langs             []string
		services          []string
		direnv            bool
		yes               bool
		force             bool
		dryRun            bool
		nixHardeningGuide bool
		profileName       string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a security-hardened devenv environment",
		Long:  "Generate devenv.yaml, devenv.nix, and security configuration files for the current project.",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			// Check for existing devenv.nix unless --force is set.
			if !force {
				nixPath := filepath.Join(projectRoot, "devenv.nix")
				if _, err := os.Stat(nixPath); err == nil {
					return fmt.Errorf("devenv.nix already exists; use --force to overwrite")
				}
			}

			// Detect project characteristics.
			detected := detect.Detect(projectRoot)

			// Build answers from flags.
			answers := buildAnswersFromFlags(projectRoot, langs, services, direnv)
			answers.Detected = detected
			answers.Confirmed = yes
			answers.NixHardeningGuide = nixHardeningGuide
			answers.ProfileName = profileName

			// Generate files.
			registry := ecosystem.DefaultRegistry()
			gen := NewDevenvGenerator(registry, WithProfileRegistry(profile.DefaultProfileRegistry()))
			files, err := gen.Generate(answers)
			if err != nil {
				return fmt.Errorf("generating files: %w", err)
			}

			// Dry-run: show preview and exit.
			if dryRun {
				preview := generate.PreviewFiles(files, nil, projectRoot)
				_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
				return nil
			}

			// Write files to disk.
			result, err := generate.WriteFiles(files, generate.PipelineOptions{
				ProjectRoot: projectRoot,
			})
			if err != nil {
				return fmt.Errorf("writing files: %w", err)
			}

			// Save state and answers.
			successfulFiles := result.SuccessfulFiles(files)
			genState := state.RecordFiles(successfulFiles)
			stateFile := filepath.Join(projectRoot, statePath())
			if err := state.SaveStateToFile(stateFile, genState); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}
			if err := saveAnswers(projectRoot, answers); err != nil {
				return fmt.Errorf("saving answers: %w", err)
			}

			// Print summary.
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), PostGenerationMessage(answers.Direnv, ""))

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&langs, "lang", nil, "Languages to configure (e.g. go,javascript,python)")
	cmd.Flags().StringSliceVar(&services, "services", nil, "Services to configure (e.g. postgres,redis)")
	cmd.Flags().BoolVar(&direnv, "direnv", true, "Enable direnv integration")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompts")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing configuration")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing files")
	cmd.Flags().BoolVar(&nixHardeningGuide, "nix-hardening-guide", false, "Generate docs/nix-conf-hardening.md with system-level Nix security recommendations")
	cmd.Flags().StringVar(&profileName, "profile", "", "Infrastructure profile (consulting-default, startup-github, enterprise)")

	return cmd
}

func updateCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Regenerate devenv files from saved answers",
		Long:  "Re-run generation using previously saved wizard answers, incorporating any detection changes.",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			// Load saved answers.
			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			// Refresh detection.
			answers.Detected = detect.Detect(projectRoot)

			// Check for existing devenv.nix unless --force is set.
			if !force {
				stateFile := filepath.Join(projectRoot, statePath())
				existingState, err := state.LoadStateFromFile(stateFile)
				if err != nil {
					return fmt.Errorf("loading state: %w", err)
				}
				modified := state.CheckModified(existingState, projectRoot)
				var modifiedPaths []string
				for path, status := range modified {
					if status.Status == types.Modified {
						modifiedPaths = append(modifiedPaths, path)
					}
				}
				if len(modifiedPaths) > 0 {
					sort.Strings(modifiedPaths)
					return fmt.Errorf("modified files found (use --force to overwrite):\n  %s",
						strings.Join(modifiedPaths, "\n  "))
				}
			}

			// Generate files.
			registry := ecosystem.DefaultRegistry()
			gen := NewDevenvGenerator(registry, WithProfileRegistry(profile.DefaultProfileRegistry()))
			files, err := gen.Generate(answers)
			if err != nil {
				return fmt.Errorf("generating files: %w", err)
			}

			// Dry-run: show preview and exit.
			if dryRun {
				preview := generate.PreviewFiles(files, nil, projectRoot)
				_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
				return nil
			}

			// Write files to disk.
			result, err := generate.WriteFiles(files, generate.PipelineOptions{
				ProjectRoot: projectRoot,
			})
			if err != nil {
				return fmt.Errorf("writing files: %w", err)
			}

			// Save state and answers.
			successfulFiles := result.SuccessfulFiles(files)
			genState := state.RecordFiles(successfulFiles)
			stateFile := filepath.Join(projectRoot, statePath())
			if err := state.SaveStateToFile(stateFile, genState); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}
			if err := saveAnswers(projectRoot, answers); err != nil {
				return fmt.Errorf("saving answers: %w", err)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Summary())
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite even if files have been modified")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing files")

	return cmd
}

// itemSpec parameterizes the differences between add/remove commands for
// services, languages, packages, and overlays. The factory functions
// makeAddCmd and makeRemoveCmd use it to build cobra.Commands with
// identical control flow but type-specific behavior.
type itemSpec struct {
	singular  string   // "service", "language", "package", "overlay"
	use       string   // cobra Use field
	short     string   // cobra Short description
	long      string   // cobra Long description
	validArgs []string // for shell completion (nil if not applicable)
	multiArg  bool     // true if the command accepts multiple args
	hasForce  bool     // true if the add command supports --force

	// validate checks whether name is an acceptable value. Return nil to skip.
	validate func(name string, projectRoot string) error
	// contains reports whether name is already present in answers.
	contains func(a *types.WizardAnswers, name string) bool
	// add appends name to the appropriate slice in answers.
	add func(a *types.WizardAnswers, name string)
	// remove filters name out of the appropriate slice. Returns true if found.
	remove func(a *types.WizardAnswers, name string) bool
	// postMessage is printed after a successful add or remove (empty to skip).
	postMessage string
}

// makeAddCmd builds a cobra.Command that adds one or more items to the devenv
// configuration using the behavior described by spec.
func makeAddCmd(spec itemSpec) *cobra.Command {
	var (
		force  bool
		dryRun bool
	)

	argsValidator := cobra.ExactArgs(1)
	if spec.multiArg {
		argsValidator = cobra.MinimumNArgs(1)
	}

	cmd := &cobra.Command{
		Use:       spec.use,
		Short:     spec.short,
		Long:      spec.long,
		Args:      argsValidator,
		ValidArgs: spec.validArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			// Validate all arguments before loading state.
			if spec.validate != nil {
				for _, name := range args {
					if err := spec.validate(name, projectRoot); err != nil {
						return err
					}
				}
			}

			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			// Collect the names that are actually new.
			var added []string
			for _, name := range args {
				if spec.contains(&answers, name) {
					if !force {
						if spec.multiArg {
							_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
								"Package %q already configured (use --force to re-add)\n", name)
							continue
						}
						if spec.hasForce {
							return fmt.Errorf("%s %q is already configured; use --force to overwrite", spec.singular, name)
						}
						return fmt.Errorf("%s %q is already configured", spec.singular, name)
					}
					// --force on a duplicate: skip the append but count as success.
					continue
				}
				spec.add(&answers, name)
				added = append(added, name)
			}
			if len(added) == 0 {
				if spec.multiArg {
					return fmt.Errorf("no new packages to add")
				}
				// Single-arg with --force on existing item: still regenerate.
			}

			result, err := regenerateAndPersist(cmd, answers, regenerateOpts{
				projectRoot: projectRoot,
				dryRun:      dryRun,
			})
			if err != nil {
				return err
			}
			if result == nil {
				return nil // dry-run
			}

			if spec.multiArg {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added package(s): %s\n%s\n",
					strings.Join(added, ", "), result.Summary())
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added %s %q.\n%s\n",
					spec.singular, args[0], result.Summary())
			}
			if spec.postMessage != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), spec.postMessage)
			}
			return nil
		},
	}

	if spec.hasForce {
		cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing configuration")
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
}

// makeRemoveCmd builds a cobra.Command that removes one or more items from
// the devenv configuration using the behavior described by spec.
func makeRemoveCmd(spec itemSpec) *cobra.Command {
	var dryRun bool

	argsValidator := cobra.ExactArgs(1)
	if spec.multiArg {
		argsValidator = cobra.MinimumNArgs(1)
	}

	cmd := &cobra.Command{
		Use:       spec.use,
		Short:     spec.short,
		Long:      spec.long,
		Args:      argsValidator,
		ValidArgs: spec.validArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			var removed []string
			for _, name := range args {
				if spec.remove(&answers, name) {
					removed = append(removed, name)
				}
			}
			if len(removed) == 0 {
				if spec.multiArg {
					return fmt.Errorf("none of the specified packages are configured")
				}
				return fmt.Errorf("%s %q is not configured", spec.singular, args[0])
			}

			result, err := regenerateAndPersist(cmd, answers, regenerateOpts{
				projectRoot: projectRoot,
				dryRun:      dryRun,
				cleanup:     true,
			})
			if err != nil {
				return err
			}
			if result == nil {
				return nil // dry-run
			}

			if spec.multiArg {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed package(s): %s\n%s\n",
					strings.Join(removed, ", "), result.Summary())
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed %s %q.\n%s\n",
					spec.singular, args[0], result.Summary())
			}
			if spec.postMessage != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), spec.postMessage)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
}

// serviceSpec returns the itemSpec for service add/remove commands.
func serviceSpec(add bool) itemSpec {
	s := itemSpec{
		singular:  "service",
		validArgs: validServices,
		hasForce:  add,
		validate: func(name string, _ string) error {
			if !validation.IsValidService(name) {
				return fmt.Errorf("unknown service %q; valid services: %v", name, validServices)
			}
			return nil
		},
		contains: func(a *types.WizardAnswers, name string) bool {
			for _, svc := range a.Services {
				if svc.Name == name {
					return true
				}
			}
			return false
		},
		add: func(a *types.WizardAnswers, name string) {
			a.Services = append(a.Services, types.ServiceChoice{Name: name})
		},
		remove: func(a *types.WizardAnswers, name string) bool {
			found := false
			var kept []types.ServiceChoice
			for _, svc := range a.Services {
				if svc.Name == name {
					found = true
				} else {
					kept = append(kept, svc)
				}
			}
			a.Services = kept
			return found
		},
	}
	if add {
		s.use = "add-service <name>"
		s.short = "Add a development service to the environment"
		s.long = "Add a service (database, cache, queue) to the existing devenv configuration."
	} else {
		s.use = "remove-service <name>"
		s.short = "Remove a service from the development environment"
		s.long = "Remove a previously added service (database, cache, queue) from the devenv configuration."
	}
	return s
}

// languageSpec returns the itemSpec for language add/remove commands.
func languageSpec(add bool) itemSpec {
	s := itemSpec{
		singular:  "language",
		validArgs: validLanguages,
		hasForce:  add,
		validate: func(name string, _ string) error {
			if !validation.IsValidLanguage(name) {
				return fmt.Errorf("unknown language %q; valid languages: %v", name, validLanguages)
			}
			return nil
		},
		contains: func(a *types.WizardAnswers, name string) bool {
			for _, lang := range a.Languages {
				if lang.Name == name {
					return true
				}
			}
			return false
		},
		add: func(a *types.WizardAnswers, name string) {
			a.Languages = append(a.Languages, types.LanguageChoice{Name: name})
		},
		remove: func(a *types.WizardAnswers, name string) bool {
			found := false
			var kept []types.LanguageChoice
			for _, lang := range a.Languages {
				if lang.Name == name {
					found = true
				} else {
					kept = append(kept, lang)
				}
			}
			a.Languages = kept
			return found
		},
	}
	if add {
		s.use = "add-language <name>"
		s.short = "Add a language ecosystem to the environment"
		s.long = "Add a language/platform ecosystem module to the existing devenv configuration."
	} else {
		s.use = "remove-language <name>"
		s.short = "Remove a language ecosystem from the environment"
		s.long = "Remove a previously added language/platform ecosystem from the devenv configuration."
	}
	return s
}

const devenvActivateMessage = "Run 'direnv allow' or re-enter 'devenv shell' to activate."

// packageSpec returns the itemSpec for package add/remove commands.
func packageSpec(add bool) itemSpec {
	s := itemSpec{
		singular:    "package",
		multiArg:    true,
		hasForce:    add,
		postMessage: devenvActivateMessage,
		contains: func(a *types.WizardAnswers, name string) bool {
			for _, p := range a.ExtraPackages {
				if p == name {
					return true
				}
			}
			return false
		},
		add: func(a *types.WizardAnswers, name string) {
			a.ExtraPackages = append(a.ExtraPackages, name)
		},
		remove: func(a *types.WizardAnswers, name string) bool {
			found := false
			var kept []string
			for _, p := range a.ExtraPackages {
				if p == name {
					found = true
				} else {
					kept = append(kept, p)
				}
			}
			a.ExtraPackages = kept
			return found
		},
	}
	if add {
		s.use = "add-package <name> [name...]"
		s.short = "Add system packages to the development environment"
		s.long = "Add Nix packages (e.g., imagemagick, ffmpeg, jq) to the devenv shell without editing Nix files."
	} else {
		s.use = "remove-package <name> [name...]"
		s.short = "Remove system packages from the development environment"
		s.long = "Remove previously added Nix packages from the devenv shell."
	}
	return s
}

// overlaySpec returns the itemSpec for overlay add/remove commands.
func overlaySpec(add bool) itemSpec {
	s := itemSpec{
		singular:    "overlay",
		postMessage: devenvActivateMessage,
		contains: func(a *types.WizardAnswers, name string) bool {
			return slices.Contains(a.Overlays, name)
		},
		add: func(a *types.WizardAnswers, name string) {
			a.Overlays = append(a.Overlays, name)
		},
		remove: func(a *types.WizardAnswers, name string) bool {
			found := false
			var kept []string
			for _, o := range a.Overlays {
				if o == name {
					found = true
				} else {
					kept = append(kept, o)
				}
			}
			a.Overlays = kept
			return found
		},
	}
	if add {
		s.use = "add-overlay <path>"
		s.short = "Add a Nix overlay to the development environment"
		s.long = "Register a Nix overlay file (e.g. ./nix/go-overlay.nix) so it persists across qsdev updates."
		s.validate = func(name string, projectRoot string) error {
			absOverlay := name
			if !filepath.IsAbs(name) {
				absOverlay = filepath.Join(projectRoot, name)
			}
			if _, err := os.Stat(absOverlay); err != nil {
				return fmt.Errorf("overlay file not found: %s", name)
			}
			return nil
		}
	} else {
		s.use = "remove-overlay <path>"
		s.short = "Remove a Nix overlay from the development environment"
		s.long = "Unregister a Nix overlay file so it is no longer included in devenv.nix."
	}
	return s
}

// regenerateOpts controls regenerateAndPersist behavior.
type regenerateOpts struct {
	projectRoot string
	dryRun      bool
	cleanup     bool // load old state and remove orphaned files after write
}

// regenerateAndPersist generates files from answers, writes them to disk, and
// persists both state and answers. For remove commands, set cleanup=true to
// detect and delete orphaned files that are no longer produced.
func regenerateAndPersist(cmd *cobra.Command, answers types.WizardAnswers, opts regenerateOpts) (*generate.WriteResult, error) {
	registry := ecosystem.DefaultRegistry()
	gen := NewDevenvGenerator(registry, WithProfileRegistry(profile.DefaultProfileRegistry()))
	files, err := gen.Generate(answers)
	if err != nil {
		return nil, fmt.Errorf("generating files: %w", err)
	}

	// Dry-run: show preview and exit.
	if opts.dryRun {
		preview := generate.PreviewFiles(files, nil, opts.projectRoot)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
		return nil, nil
	}

	// Load old state before writing so we can detect orphans.
	stateFile := filepath.Join(opts.projectRoot, statePath())
	var oldState types.GeneratedState
	if opts.cleanup {
		oldState, _ = state.LoadStateFromFile(stateFile)
	}

	// Write files to disk.
	result, err := generate.WriteFiles(files, generate.PipelineOptions{
		ProjectRoot: opts.projectRoot,
	})
	if err != nil {
		return nil, fmt.Errorf("writing files: %w", err)
	}

	// Cleanup orphaned files from prior generation.
	if opts.cleanup {
		cleanupOrphanedFiles(cmd, oldState, files, opts.projectRoot)
	}

	// Save state and answers.
	successfulFiles := result.SuccessfulFiles(files)
	genState := state.RecordFiles(successfulFiles)
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		return nil, fmt.Errorf("saving state: %w", err)
	}
	if err := saveAnswers(opts.projectRoot, answers); err != nil {
		return nil, fmt.Errorf("saving answers: %w", err)
	}

	return &result, nil
}

// buildAnswersFromFlags constructs a WizardAnswers from CLI flag values.
func buildAnswersFromFlags(projectRoot string, langs, services []string, direnv bool) types.WizardAnswers {
	answers := types.WizardAnswers{
		ProjectRoot: projectRoot,
		ProjectName: filepath.Base(projectRoot),
		Direnv:      direnv,
	}

	for _, name := range langs {
		answers.Languages = append(answers.Languages, types.LanguageChoice{
			Name: name,
		})
	}

	for _, name := range services {
		answers.Services = append(answers.Services, types.ServiceChoice{
			Name: name,
		})
	}

	return answers
}

// cleanupOrphanedFiles removes files that were previously tracked in state but
// are no longer produced after a configuration change. Modified orphans are
// preserved with a warning.
func cleanupOrphanedFiles(cmd *cobra.Command, oldState types.GeneratedState, newFiles []types.GeneratedFile, projectRoot string) {
	orphans := state.OrphanedFiles(oldState, newFiles)
	for _, orphanPath := range orphans {
		absPath := filepath.Join(projectRoot, orphanPath)
		fs, ok := oldState.Files[orphanPath]
		if ok {
			currentHash, err := state.ComputeFileHash(absPath)
			if err == nil && currentHash != fs.Hash {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Orphaned file %s has local modifications; not removing\n", orphanPath)
				continue
			}
		}
		if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Warning: could not remove orphaned file %s: %v\n", orphanPath, err)
		} else if err == nil {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Removed orphaned file: %s\n", orphanPath)
		}
	}
}
