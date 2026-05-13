package devinit

import (
	"fmt"
	"strings"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/validation"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// Validation lists are sourced from the shared validation package.
var (
	validLanguages        = validation.Languages()
	validServices         = validation.Services()
	validPermissionPresets = validation.PermissionPresets()
	validNodePkgMgrs      = validation.NodePackageManagers()
	validPythonPkgMgrs    = validation.PythonPackageManagers()
)

// ValidateAnswers checks that all user-provided values are valid.
// It returns a combined error describing every validation failure, or nil
// when all values pass.
func ValidateAnswers(answers types.WizardAnswers) error {
	var errs []string

	// Validate language names.
	for _, lang := range answers.Languages {
		if !validation.IsValidLanguage(lang.Name) {
			errs = append(errs, fmt.Sprintf("unknown language %q; valid languages: %v", lang.Name, validLanguages))
		}

		// Validate node package manager if set.
		if lang.Name == "javascript" && lang.PackageManager != "" {
			if !validation.IsValidNodePackageManager(lang.PackageManager) {
				errs = append(errs, fmt.Sprintf("unknown node package manager %q; valid values: %v", lang.PackageManager, validNodePkgMgrs))
			}
		}

		// Validate python package manager if set.
		if lang.Name == "python" && lang.PackageManager != "" {
			if !validation.IsValidPythonPackageManager(lang.PackageManager) {
				errs = append(errs, fmt.Sprintf("unknown python package manager %q; valid values: %v", lang.PackageManager, validPythonPkgMgrs))
			}
		}
	}

	// Validate service names.
	for _, svc := range answers.Services {
		if !validation.IsValidService(svc.Name) {
			errs = append(errs, fmt.Sprintf("unknown service %q; valid services: %v", svc.Name, validServices))
		}
	}

	// Validate permission level.
	if answers.PermissionLevel != "" {
		if !validation.IsValidPermissionPreset(answers.PermissionLevel) {
			errs = append(errs, fmt.Sprintf("unknown permission preset %q; valid presets: %v", answers.PermissionLevel, validPermissionPresets))
		}
	}

	// Validate environment variable format.
	for k := range answers.EnvVars {
		if k == "" {
			errs = append(errs, "environment variable has empty key")
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
}
