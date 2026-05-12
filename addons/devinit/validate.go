package devinit

import (
	"fmt"
	"strings"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// Validation constants — these mirror the lists in devenv/commands.go and
// claudecode/commands.go but are defined here to avoid importing unexported
// variables from those packages.
var (
	validLanguages = []string{
		"go", "javascript", "python", "rust",
		"java", "dotnet", "docker", "terraform",
		"php", "ruby", "scala",
		"cpp", "helm", "ansible", "shell",
		"elixir", "dart", "swift", "haskell", "clojure", "bazel", "nix",
		"perl", "r", "lua", "zig", "powershell",
	}

	validServices = []string{
		"postgres", "redis", "mysql",
		"mongodb", "elasticsearch", "rabbitmq",
	}

	validPermissionPresets = []string{
		"minimal", "standard", "permissive", "custom",
	}

	validNodePkgMgrs   = []string{"npm", "pnpm", "yarn", "bun"}
	validPythonPkgMgrs = []string{"pip", "uv", "poetry"}
)

// ValidateAnswers checks that all user-provided values are valid.
// It returns a combined error describing every validation failure, or nil
// when all values pass.
func ValidateAnswers(answers types.WizardAnswers) error {
	var errs []string

	// Validate language names.
	for _, lang := range answers.Languages {
		if !containsStr(validLanguages, lang.Name) {
			errs = append(errs, fmt.Sprintf("unknown language %q; valid languages: %v", lang.Name, validLanguages))
		}

		// Validate node package manager if set.
		if lang.Name == "javascript" && lang.PackageManager != "" {
			if !containsStr(validNodePkgMgrs, lang.PackageManager) {
				errs = append(errs, fmt.Sprintf("unknown node package manager %q; valid values: %v", lang.PackageManager, validNodePkgMgrs))
			}
		}

		// Validate python package manager if set.
		if lang.Name == "python" && lang.PackageManager != "" {
			if !containsStr(validPythonPkgMgrs, lang.PackageManager) {
				errs = append(errs, fmt.Sprintf("unknown python package manager %q; valid values: %v", lang.PackageManager, validPythonPkgMgrs))
			}
		}
	}

	// Validate service names.
	for _, svc := range answers.Services {
		if !containsStr(validServices, svc.Name) {
			errs = append(errs, fmt.Sprintf("unknown service %q; valid services: %v", svc.Name, validServices))
		}
	}

	// Validate permission level.
	if answers.PermissionLevel != "" {
		if !containsStr(validPermissionPresets, answers.PermissionLevel) {
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

// containsStr checks whether a string slice includes the given value.
func containsStr(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
