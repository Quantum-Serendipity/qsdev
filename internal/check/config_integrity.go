package check

import (
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/config"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/validation"
)

// CheckConfigIntegrity verifies that .gdev.yaml exists, parses correctly,
// and has valid profile, language, and service names.
func CheckConfigIntegrity(ctx CheckContext) []CheckResult {
	if ctx.GdevConfig == nil {
		return []CheckResult{
			{
				Category:    CategoryConfigIntegrity,
				Name:        "config_exists",
				Status:      StatusFail,
				Severity:    SeverityCritical,
				Message:     ".gdev.yaml not found",
				Remediation: "Run 'gdev init' to create a project configuration",
			},
		}
	}

	var results []CheckResult

	// Config exists and parsed.
	results = append(results, CheckResult{
		Category: CategoryConfigIntegrity,
		Name:     "config_exists",
		Status:   StatusPass,
		Severity: SeverityInfo,
		Message:  ".gdev.yaml found and parsed successfully",
	})

	// Validate using config.ValidateGdevConfig.
	opts := config.ValidateOptions{
		ProfileNames: ctx.ProfileNames,
		ToolNames:    ctx.ToolNames,
	}
	if errs := config.ValidateGdevConfig(ctx.GdevConfig, opts); len(errs) > 0 {
		for _, ve := range errs {
			results = append(results, CheckResult{
				Category:    CategoryConfigIntegrity,
				Name:        "config_validation",
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Message:     ve.Error(),
				Remediation: "Fix the configuration in .gdev.yaml",
			})
		}
	}

	// Additional explicit checks for language names.
	allLangsValid := true
	for _, lang := range ctx.GdevConfig.Languages {
		if !validation.IsValidLanguage(lang.Name) {
			allLangsValid = false
			// Already reported by ValidateGdevConfig above.
		}
	}
	if allLangsValid && len(ctx.GdevConfig.Languages) > 0 {
		results = append(results, CheckResult{
			Category: CategoryConfigIntegrity,
			Name:     "languages_valid",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "All language names are valid",
		})
	}

	// Check service names.
	allSvcsValid := true
	for _, svc := range ctx.GdevConfig.Services {
		if !validation.IsValidService(svc.Name) {
			allSvcsValid = false
		}
	}
	if allSvcsValid && len(ctx.GdevConfig.Services) > 0 {
		results = append(results, CheckResult{
			Category: CategoryConfigIntegrity,
			Name:     "services_valid",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "All service names are valid",
		})
	}

	// Check profile name.
	if ctx.GdevConfig.Profile != "" && ctx.ProfileNames != nil {
		found := false
		for _, p := range ctx.ProfileNames {
			if p == ctx.GdevConfig.Profile {
				found = true
				break
			}
		}
		if found {
			results = append(results, CheckResult{
				Category: CategoryConfigIntegrity,
				Name:     "profile_valid",
				Status:   StatusPass,
				Severity: SeverityInfo,
				Message:  "Profile " + ctx.GdevConfig.Profile + " is valid",
			})
		}
		// If not found, ValidateGdevConfig already reported it.
	}

	return results
}
