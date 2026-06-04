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
		addServiceCmd(),
		addLanguageCmd(),
		addPackageCmd(),
		addOverlayCmd(),
		removeServiceCmd(),
		removeLanguageCmd(),
		removePackageCmd(),
		removeOverlayCmd(),
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

func addServiceCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:       "add-service <name>",
		Short:     "Add a development service to the environment",
		Long:      "Add a service (database, cache, queue) to the existing devenv configuration.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: validServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			serviceName := args[0]

			// Validate service name.
			if !validation.IsValidService(serviceName) {
				return fmt.Errorf("unknown service %q; valid services: %v", serviceName, validServices)
			}

			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			// Load saved answers.
			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			// Check for duplicate (skip when --force is set).
			alreadyPresent := false
			for _, svc := range answers.Services {
				if svc.Name == serviceName {
					alreadyPresent = true
					break
				}
			}
			if alreadyPresent && !force {
				return fmt.Errorf("service %q is already configured; use --force to overwrite", serviceName)
			}
			if !alreadyPresent {
				answers.Services = append(answers.Services, types.ServiceChoice{
					Name: serviceName,
				})
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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added service %q.\n%s\n", serviceName, result.Summary())
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
}

func addLanguageCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:       "add-language <name>",
		Short:     "Add a language ecosystem to the environment",
		Long:      "Add a language/platform ecosystem module to the existing devenv configuration.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: validLanguages,
		RunE: func(cmd *cobra.Command, args []string) error {
			langName := args[0]

			// Validate language name.
			if !validation.IsValidLanguage(langName) {
				return fmt.Errorf("unknown language %q; valid languages: %v", langName, validLanguages)
			}

			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			// Load saved answers.
			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			// Check for duplicate (skip when --force is set).
			alreadyPresent := false
			for _, lang := range answers.Languages {
				if lang.Name == langName {
					alreadyPresent = true
					break
				}
			}
			if alreadyPresent && !force {
				return fmt.Errorf("language %q is already configured; use --force to overwrite", langName)
			}
			if !alreadyPresent {
				answers.Languages = append(answers.Languages, types.LanguageChoice{
					Name: langName,
				})
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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added language %q.\n%s\n", langName, result.Summary())
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
}

func addPackageCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "add-package <name> [name...]",
		Short: "Add system packages to the development environment",
		Long:  "Add Nix packages (e.g., imagemagick, ffmpeg, jq) to the devenv shell without editing Nix files.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			existing := make(map[string]bool)
			for _, p := range answers.ExtraPackages {
				existing[p] = true
			}
			var added []string
			for _, pkg := range args {
				if existing[pkg] && !force {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Package %q already configured (use --force to re-add)\n", pkg)
					continue
				}
				if !existing[pkg] {
					answers.ExtraPackages = append(answers.ExtraPackages, pkg)
					added = append(added, pkg)
				}
			}
			if len(added) == 0 {
				return fmt.Errorf("no new packages to add")
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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added package(s): %s\n%s\n", strings.Join(added, ", "), result.Summary())
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Run 'direnv allow' or re-enter 'devenv shell' to activate.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing configuration")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
}

func removePackageCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "remove-package <name> [name...]",
		Short: "Remove system packages from the development environment",
		Long:  "Remove previously added Nix packages from the devenv shell.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			toRemove := make(map[string]bool)
			for _, pkg := range args {
				toRemove[pkg] = true
			}

			var kept []string
			var removed []string
			for _, p := range answers.ExtraPackages {
				if toRemove[p] {
					removed = append(removed, p)
				} else {
					kept = append(kept, p)
				}
			}
			if len(removed) == 0 {
				return fmt.Errorf("none of the specified packages are configured")
			}
			answers.ExtraPackages = kept

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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed package(s): %s\n%s\n", strings.Join(removed, ", "), result.Summary())
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Run 'direnv allow' or re-enter 'devenv shell' to activate.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
}

func addOverlayCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "add-overlay <path>",
		Short: "Add a Nix overlay to the development environment",
		Long:  "Register a Nix overlay file (e.g. ./nix/go-overlay.nix) so it persists across qsdev updates.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			overlayPath := args[0]

			absOverlay := overlayPath
			if !filepath.IsAbs(overlayPath) {
				absOverlay = filepath.Join(projectRoot, overlayPath)
			}
			if _, err := os.Stat(absOverlay); err != nil {
				return fmt.Errorf("overlay file not found: %s", overlayPath)
			}

			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			if slices.Contains(answers.Overlays, overlayPath) {
				return fmt.Errorf("overlay %q is already configured", overlayPath)
			}
			answers.Overlays = append(answers.Overlays, overlayPath)

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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added overlay: %s\n%s\n", overlayPath, result.Summary())
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Run 'direnv allow' or re-enter 'devenv shell' to activate.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
}

func removeOverlayCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "remove-overlay <path>",
		Short: "Remove a Nix overlay from the development environment",
		Long:  "Unregister a Nix overlay file so it is no longer included in devenv.nix.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			overlayPath := args[0]
			found := false
			var kept []string
			for _, o := range answers.Overlays {
				if o == overlayPath {
					found = true
				} else {
					kept = append(kept, o)
				}
			}
			if !found {
				return fmt.Errorf("overlay %q is not configured", overlayPath)
			}
			answers.Overlays = kept

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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed overlay: %s\n%s\n", overlayPath, result.Summary())
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Run 'direnv allow' or re-enter 'devenv shell' to activate.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
}

func removeServiceCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:       "remove-service <name>",
		Short:     "Remove a service from the development environment",
		Long:      "Remove a previously added service (database, cache, queue) from the devenv configuration.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: validServices,
		RunE: func(cmd *cobra.Command, args []string) error {
			serviceName := args[0]
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			found := false
			var kept []types.ServiceChoice
			for _, svc := range answers.Services {
				if svc.Name == serviceName {
					found = true
				} else {
					kept = append(kept, svc)
				}
			}
			if !found {
				return fmt.Errorf("service %q is not configured", serviceName)
			}
			answers.Services = kept

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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed service %q.\n%s\n", serviceName, result.Summary())
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
}

func removeLanguageCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:       "remove-language <name>",
		Short:     "Remove a language ecosystem from the environment",
		Long:      "Remove a previously added language/platform ecosystem from the devenv configuration.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: validLanguages,
		RunE: func(cmd *cobra.Command, args []string) error {
			langName := args[0]
			projectRoot, err := cmdutil.ProjectRoot()
			if err != nil {
				return err
			}

			answers, err := loadAnswers(projectRoot)
			if err != nil {
				return err
			}

			found := false
			var kept []types.LanguageChoice
			for _, lang := range answers.Languages {
				if lang.Name == langName {
					found = true
				} else {
					kept = append(kept, lang)
				}
			}
			if !found {
				return fmt.Errorf("language %q is not configured", langName)
			}
			answers.Languages = kept

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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed language %q.\n%s\n", langName, result.Summary())
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
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
