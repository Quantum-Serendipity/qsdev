package devinit

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ValidateAnswers checks that all user-provided values are valid.
// It returns a combined error describing every validation failure, or nil
// when all values pass.
//
// Answers reach generation from team-shared inputs (the committed .qsdev.yaml
// in join mode, --answers-file), so besides the enumerated names this also
// enforces the syntax of every free-form value that is spliced into devenv.nix
// (versions, package managers, extra packages, env keys, service settings).
func ValidateAnswers(answers types.WizardAnswers) error {
	var errs []string

	errs = append(errs, validateLanguageChoices(answers.Languages)...)
	errs = append(errs, validateServiceChoices(answers.Services)...)

	// Validate permission level.
	if answers.PermissionLevel != "" {
		if !validation.IsValidPermissionPreset(answers.PermissionLevel) {
			errs = append(errs, fmt.Sprintf("unknown permission preset %q; valid presets: %v", answers.PermissionLevel, validation.PermissionPresets()))
		}
	}

	// Validate tier.
	if answers.Tier != "" {
		if !validation.IsValidTier(answers.Tier) {
			errs = append(errs, fmt.Sprintf("unknown tier %q; valid tiers: %v", answers.Tier, validation.Tiers()))
		}
	}

	// Extra packages are emitted unquoted as pkgs.<name>.
	for _, pkg := range answers.ExtraPackages {
		if !validation.IsValidNixAttrPath(pkg) {
			errs = append(errs, fmt.Sprintf("invalid package name %q: must be a Nix attribute path such as jq or python3Packages.black", pkg))
		}
	}

	// The branch pattern is spliced into the branch-naming pre-push hook.
	if err := validation.CheckBranchPattern(answers.BranchPattern); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate environment variable names; keys are emitted unquoted.
	for _, k := range slices.Sorted(maps.Keys(answers.EnvVars)) {
		switch {
		case k == "":
			errs = append(errs, "environment variable has empty key")
		case !validation.IsValidEnvKey(k):
			errs = append(errs, fmt.Sprintf("invalid environment variable name %q: must match [A-Za-z_][A-Za-z0-9_]*", k))
		}
	}

	// The hook policy is handed to the hooks through settings.json: extra
	// read paths widen what the agent may read, tool-gates entries decide
	// which tools it may call.
	for _, e := range validation.CheckHookPolicy(answers.HookPolicy) {
		errs = append(errs, "invalid "+e.Error())
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
}

// validateLanguageChoices checks language names, versions and package managers.
func validateLanguageChoices(langs []types.LanguageChoice) []string {
	var errs []string
	for _, lang := range langs {
		if !validation.IsValidLanguage(lang.Name) {
			errs = append(errs, fmt.Sprintf("unknown language %q; valid languages: %v", lang.Name, validation.Languages()))
		}

		if lang.Version != "" && !validation.IsValidVersionConstraint(lang.Version) {
			errs = append(errs, fmt.Sprintf("invalid %s version %q: only letters, digits, spaces and . _ - + * ^ ~ < > = ! | , / are allowed", lang.Name, lang.Version))
		}

		if lang.PackageManager == "" {
			continue
		}
		switch {
		case lang.Name == "javascript" && !validation.IsValidNodePackageManager(lang.PackageManager):
			errs = append(errs, fmt.Sprintf("unknown node package manager %q; valid values: %v", lang.PackageManager, validation.NodePackageManagers()))
		case lang.Name == "python" && !validation.IsValidPythonPackageManager(lang.PackageManager):
			errs = append(errs, fmt.Sprintf("unknown python package manager %q; valid values: %v", lang.PackageManager, validation.PythonPackageManagers()))
		case !validation.IsValidToken(lang.PackageManager):
			errs = append(errs, fmt.Sprintf("invalid %s package manager %q: must be a single word of letters, digits, '.', '_' or '-'", lang.Name, lang.PackageManager))
		}
	}
	return errs
}

// validateServiceChoices checks service names, versions and settings. Service
// versions are spliced into Nix attribute names (pkgs.postgresql_<version>),
// so they must be bare tokens.
func validateServiceChoices(services []types.ServiceChoice) []string {
	var errs []string
	for _, svc := range services {
		if !validation.IsValidService(svc.Name) {
			errs = append(errs, fmt.Sprintf("unknown service %q; valid services: %v", svc.Name, validation.Services()))
		}
		if svc.Version != "" && !validation.IsValidToken(svc.Version) {
			errs = append(errs, fmt.Sprintf("invalid %s version %q: must be a single word of letters, digits, '.', '_' or '-'", svc.Name, svc.Version))
		}
		for _, k := range slices.Sorted(maps.Keys(svc.Settings)) {
			if !validation.IsValidEnvKey(k) || !validation.IsValidToken(svc.Settings[k]) {
				errs = append(errs, fmt.Sprintf("invalid %s setting %s=%q: keys must be identifiers and values a single word of letters, digits, '.', '_' or '-'", svc.Name, k, svc.Settings[k]))
			}
		}
	}
	return errs
}
