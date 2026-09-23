package devenv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/validation"
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
	return state.DevenvStateFile()
}

// validServices returns the canonical service list for shell completion. It
// is resolved on use, not at package initialization, so a broken catalog
// overlay cannot crash the binary before any command runs.
func validServices() []string { return validation.Services() }

// validLanguages returns the canonical core language list for shell completion.
func validLanguages() []string { return validation.CoreLanguages() }

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

			// Write files and persist state and answers. A stale or corrupt
			// state file from an earlier run must not block a fresh init, so a
			// load failure starts from empty state.
			oldState, err := state.LoadStateFromFile(filepath.Join(projectRoot, statePath()))
			if err != nil {
				oldState = types.GeneratedState{}
			}
			result, err := writeAndPersist(cmd, projectRoot, answers, files, oldState, false, force)
			if err != nil {
				return err
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

			result, err := regenerateAndPersist(cmd, answers, regenerateOpts{
				projectRoot: projectRoot,
				dryRun:      dryRun,
				force:       force,
			})
			if err != nil {
				return err
			}
			if result == nil {
				return nil // dry-run
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
	singular  string          // "service", "language", "package", "overlay"
	use       string          // cobra Use field
	short     string          // cobra Short description
	long      string          // cobra Long description
	validArgs func() []string // shell-completion candidates, resolved lazily (nil if not applicable)
	multiArg  bool            // true if the command accepts multiple args
	hasForce  bool            // true if --force re-adds an entry that is already configured

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
		Use:               spec.use,
		Short:             spec.short,
		Long:              spec.long,
		Args:              argsValidator,
		ValidArgsFunction: cmdutil.CompleteFrom(spec.validArgs),
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

			// Collect the names that are actually new. With --force, entries
			// that are already configured still regenerate the environment.
			reAdd := force && spec.hasForce
			var added []string
			for _, name := range args {
				if !spec.contains(&answers, name) {
					spec.add(&answers, name)
					added = append(added, name)
					continue
				}
				switch {
				case reAdd:
					// Already present: nothing to append, but still regenerate.
				case spec.multiArg:
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
						"Package %q already configured (use --force to regenerate anyway)\n", name)
				case spec.hasForce:
					return fmt.Errorf("%s %q is already configured; use --force to overwrite", spec.singular, name)
				default:
					return fmt.Errorf("%s %q is already configured", spec.singular, name)
				}
			}
			if len(added) == 0 && spec.multiArg && !reAdd {
				return fmt.Errorf("no new packages to add")
			}

			result, err := regenerateAndPersist(cmd, answers, regenerateOpts{
				projectRoot: projectRoot,
				dryRun:      dryRun,
				force:       force,
			})
			if err != nil {
				return err
			}
			if result == nil {
				return nil // dry-run
			}

			switch {
			case spec.multiArg && len(added) == 0:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Package(s) already configured; regenerated environment.\n%s\n",
					result.Summary())
			case spec.multiArg:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added package(s): %s\n%s\n",
					strings.Join(added, ", "), result.Summary())
			default:
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added %s %q.\n%s\n",
					spec.singular, args[0], result.Summary())
			}
			if spec.postMessage != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), spec.postMessage)
			}
			return nil
		},
	}

	forceUsage := "Overwrite generated files even if they have local modifications"
	if spec.hasForce {
		forceUsage = "Re-add existing entries and overwrite generated files even if they have local modifications"
	}
	cmd.Flags().BoolVar(&force, "force", false, forceUsage)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")

	return cmd
}

// makeRemoveCmd builds a cobra.Command that removes one or more items from
// the devenv configuration using the behavior described by spec.
func makeRemoveCmd(spec itemSpec) *cobra.Command {
	var (
		force  bool
		dryRun bool
	)

	argsValidator := cobra.ExactArgs(1)
	if spec.multiArg {
		argsValidator = cobra.MinimumNArgs(1)
	}

	cmd := &cobra.Command{
		Use:               spec.use,
		Short:             spec.short,
		Long:              spec.long,
		Args:              argsValidator,
		ValidArgsFunction: cmdutil.CompleteFrom(spec.validArgs),
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
				force:       force,
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

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite generated files even if they have local modifications")
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
				return fmt.Errorf("unknown service %q; valid services: %v", name, validServices())
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
				return fmt.Errorf("unknown language %q; valid languages: %v", name, validLanguages())
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

// nixAttrPathPattern matches a dotted nixpkgs attribute path such as "jq" or
// "python3Packages.requests". Every segment must be a plain Nix identifier;
// quoted attribute names, whitespace and all other Nix syntax are rejected.
var nixAttrPathPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_'-]*(\.[A-Za-z_][A-Za-z0-9_'-]*)*$`)

// nixKeywords are the reserved words of the Nix language. They match the
// identifier grammar but change how an expression parses, so they may not be
// used as attribute path segments.
var nixKeywords = map[string]bool{
	"assert": true, "else": true, "if": true, "in": true, "inherit": true,
	"let": true, "or": true, "rec": true, "then": true, "with": true,
}

// validateNixPackageName reports whether name is safe to splice into
// devenv.nix as `pkgs.<name>`. Package names are rendered verbatim as Nix
// code, so anything other than a plain attribute path would let a caller
// inject arbitrary Nix expressions into the generated environment.
func validateNixPackageName(name string) error {
	if !nixAttrPathPattern.MatchString(name) {
		return fmt.Errorf("invalid package name %q: must be a nixpkgs attribute path such as \"jq\" or \"python3Packages.requests\"", name)
	}
	for segment := range strings.SplitSeq(name, ".") {
		if nixKeywords[segment] {
			return fmt.Errorf("invalid package name %q: %q is a reserved Nix keyword", name, segment)
		}
	}
	return nil
}

// packageSpec returns the itemSpec for package add/remove commands.
func packageSpec(add bool) itemSpec {
	s := itemSpec{
		singular:    "package",
		multiArg:    true,
		hasForce:    add,
		postMessage: devenvActivateMessage,
		validate: func(name string, _ string) error {
			return validateNixPackageName(name)
		},
		contains: func(a *types.WizardAnswers, name string) bool {
			return slices.Contains(a.ExtraPackages, name)
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
	force       bool // overwrite generated files even if they were modified locally
	cleanup     bool // remove orphaned files that are no longer produced
}

// regenerateAndPersist generates files from answers, writes them to disk, and
// persists both state and answers. Unless force is set it refuses to overwrite
// generated files that have been modified locally. For remove commands, set
// cleanup=true to detect and delete orphaned files that are no longer produced.
func regenerateAndPersist(cmd *cobra.Command, answers types.WizardAnswers, opts regenerateOpts) (*generate.WriteResult, error) {
	registry := ecosystem.DefaultRegistry()
	gen := NewDevenvGenerator(registry, WithProfileRegistry(profile.DefaultProfileRegistry()))
	files, err := gen.Generate(answers)
	if err != nil {
		return nil, fmt.Errorf("generating files: %w", err)
	}

	// Refuse to clobber local edits to previously generated files.
	if !opts.force {
		if err := checkModifiedGeneratedFiles(opts.projectRoot, files); err != nil {
			return nil, err
		}
	}

	// Dry-run: show preview and exit.
	if opts.dryRun {
		preview := generate.PreviewFiles(files, nil, opts.projectRoot)
		_, _ = fmt.Fprint(cmd.OutOrStdout(), preview)
		return nil, nil
	}

	// Load old state before writing so entries for files that are not
	// rewritten survive, and so orphans can be detected.
	oldState, err := state.LoadStateFromFile(filepath.Join(opts.projectRoot, statePath()))
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	return writeAndPersist(cmd, opts.projectRoot, answers, files, oldState, opts.cleanup, opts.force)
}

// writeAndPersist writes files to disk, records their state and saves the
// answers. Existing files whose strategy is Skip are left untouched. State is
// saved for every file that was written, while prior entries for files that
// were skipped or failed are carried forward so their modification tracking
// survives. Answers are saved only when every file was written; otherwise an
// error listing the failures is returned so the command exits non-zero and
// the configuration change is not recorded. force is passed to the pipeline.
func writeAndPersist(cmd *cobra.Command, projectRoot string, answers types.WizardAnswers, files []types.GeneratedFile, oldState types.GeneratedState, cleanup, force bool) (*generate.WriteResult, error) {
	toWrite, preserved := splitPreservedFiles(projectRoot, files)

	// force lets a ManualMerge file with local edits (devenv.nix) be replaced
	// instead of getting a sidecar; it never overrides Skip.
	result, err := generate.WriteFiles(toWrite, generate.PipelineOptions{
		ProjectRoot: projectRoot,
		Force:       force,
	})
	if err != nil {
		return nil, fmt.Errorf("writing files: %w", err)
	}
	for _, path := range preserved {
		result.Files = append(result.Files, generate.FileResult{Path: path, Action: generate.ActionSkipped})
		result.Skipped++
	}

	// Only remove orphans when the new configuration was fully written;
	// otherwise the previous configuration may still need them.
	var released []string
	if cleanup && !result.HasFailures() {
		released = cleanupOrphanedFiles(cmd, oldState, files, projectRoot)
	}

	genState := state.RecordFiles(result.SuccessfulFiles(toWrite))
	carryForwardState(&genState, oldState, released)
	if err := state.SaveStateToFile(filepath.Join(projectRoot, statePath()), genState); err != nil {
		return nil, fmt.Errorf("saving state: %w", err)
	}

	if result.HasFailures() {
		return &result, partialWriteError(result)
	}

	if err := saveAnswers(projectRoot, answers); err != nil {
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

// checkModifiedGeneratedFiles returns an error naming every file in files
// whose on-disk content differs from all versions qsdev recorded for it. The
// devenv, init and Claude Code state files are all consulted because the
// devinit lifecycle (qsdev enable/disable/init --update) also rewrites devenv
// files. Untracked, missing and Skip-strategy files are never reported: the
// first carry no generation record and Skip files are not overwritten.
func checkModifiedGeneratedFiles(projectRoot string, files []types.GeneratedFile) error {
	recorded, err := recordedFileHashes(projectRoot)
	if err != nil {
		return err
	}

	var modified []string
	for _, f := range files {
		hashes := recorded[f.Path]
		if f.Strategy == types.Skip || len(hashes) == 0 {
			continue
		}
		current, err := state.ComputeFileHash(filepath.Join(projectRoot, f.Path))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("checking %s for local modifications: %w", f.Path, err)
		}
		if !hashes[current] {
			modified = append(modified, f.Path)
		}
	}
	if len(modified) == 0 {
		return nil
	}
	sort.Strings(modified)
	return fmt.Errorf("modified files found (use --force to overwrite):\n  %s",
		strings.Join(modified, "\n  "))
}

// recordedFileHashes maps each generated path to the set of content hashes
// recorded for it across all qsdev state files.
func recordedFileHashes(projectRoot string) (map[string]map[string]bool, error) {
	recorded := make(map[string]map[string]bool)
	for _, rel := range state.StateFilePaths() {
		st, err := state.LoadStateFromFile(filepath.Join(projectRoot, rel))
		if err != nil {
			return nil, fmt.Errorf("loading state: %w", err)
		}
		for path, fs := range st.Files {
			if recorded[path] == nil {
				recorded[path] = make(map[string]bool)
			}
			recorded[path][fs.Hash] = true
		}
	}
	return recorded, nil
}

// splitPreservedFiles separates files whose Skip strategy means an existing
// on-disk copy must be left alone (e.g. a user-owned .envrc) from the files
// that should be written. It returns the files to write and the paths kept.
func splitPreservedFiles(projectRoot string, files []types.GeneratedFile) ([]types.GeneratedFile, []string) {
	toWrite := make([]types.GeneratedFile, 0, len(files))
	var preserved []string
	for _, f := range files {
		if f.Strategy == types.Skip {
			if _, err := os.Lstat(filepath.Join(projectRoot, f.Path)); err == nil {
				preserved = append(preserved, f.Path)
				continue
			}
		}
		toWrite = append(toWrite, f)
	}
	return toWrite, preserved
}

// carryForwardState copies entries from oldState into newState for paths that
// were not rewritten in this run (skipped, failed, or orphaned but kept),
// except for released paths that are no longer tracked.
func carryForwardState(newState *types.GeneratedState, oldState types.GeneratedState, released []string) {
	for path, fs := range oldState.Files {
		if _, written := newState.Files[path]; written || slices.Contains(released, path) {
			continue
		}
		newState.Files[path] = fs
	}
}

// partialWriteError describes the files that WriteFiles failed to write.
func partialWriteError(result generate.WriteResult) error {
	var details strings.Builder
	for _, ff := range result.FailedFiles() {
		fmt.Fprintf(&details, "\n  - %s: %v", ff.Path, ff.Error)
	}
	return fmt.Errorf("%d file(s) failed to write; configuration change not saved:%s",
		result.Failed, details.String())
}

// cleanupOrphanedFiles removes files that were previously tracked in state but
// are no longer produced after a configuration change. Modified orphans are
// preserved with a warning and handed over to the user. It returns the orphan
// paths that should no longer be tracked in state: those removed (or already
// gone) and those left in place because of local modifications.
func cleanupOrphanedFiles(cmd *cobra.Command, oldState types.GeneratedState, newFiles []types.GeneratedFile, projectRoot string) []string {
	var released []string
	orphans := state.OrphanedFiles(oldState, newFiles)
	for _, orphanPath := range orphans {
		absPath := filepath.Join(projectRoot, orphanPath)
		fs, ok := oldState.Files[orphanPath]
		if ok {
			currentHash, err := state.ComputeFileHash(absPath)
			if err == nil && currentHash != fs.Hash {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Orphaned file %s has local modifications; not removing\n", orphanPath)
				released = append(released, orphanPath)
				continue
			}
		}
		err := os.Remove(absPath)
		switch {
		case err == nil:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Removed orphaned file: %s\n", orphanPath)
			released = append(released, orphanPath)
		case os.IsNotExist(err):
			released = append(released, orphanPath)
		default:
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  Warning: could not remove orphaned file %s: %v\n", orphanPath, err)
		}
	}
	return released
}
