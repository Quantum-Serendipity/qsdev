package check

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// CheckConfigIntegrity verifies that .qsdev.yaml exists, parses correctly,
// and has valid profile, language, and service names.
func CheckConfigIntegrity(ctx CheckContext) []CheckResult {
	if ctx.QsdevConfig == nil {
		if configParseFailed(ctx) {
			return []CheckResult{
				{
					Category:    CategoryConfigIntegrity,
					Name:        "config_parse",
					Status:      StatusFail,
					Severity:    SeverityCritical,
					Message:     fmt.Sprintf("%s could not be parsed: %v", branding.Get().ConfigFile, ctx.ConfigErr),
					FilePath:    branding.Get().ConfigFile,
					Remediation: "Fix the configuration in " + branding.Get().ConfigFile,
				},
			}
		}
		return []CheckResult{
			{
				Category:    CategoryConfigIntegrity,
				Name:        "config_exists",
				Status:      StatusFail,
				Severity:    SeverityCritical,
				Message:     branding.Get().ConfigFile + " not found",
				Remediation: "Run '" + branding.Get().AppName + " init' to create a project configuration",
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
		Message:  branding.Get().ConfigFile + " found and parsed successfully",
	})

	// Validate using config.ValidateQsdevConfig.
	opts := config.ValidateOptions{
		ProfileNames: ctx.ProfileNames,
		ToolNames:    ctx.ToolNames,
	}
	if errs := config.ValidateQsdevConfig(ctx.QsdevConfig, opts); len(errs) > 0 {
		for _, ve := range errs {
			results = append(results, CheckResult{
				Category:    CategoryConfigIntegrity,
				Name:        "config_validation",
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Message:     ve.Error(),
				Remediation: "Fix the configuration in " + branding.Get().ConfigFile,
			})
		}
	}

	// Additional explicit checks for language names.
	allLangsValid := true
	for _, lang := range ctx.QsdevConfig.Languages {
		if !validation.IsValidLanguage(lang.Name) {
			allLangsValid = false
			// Already reported by ValidateQsdevConfig above.
		}
	}
	if allLangsValid && len(ctx.QsdevConfig.Languages) > 0 {
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
	for _, svc := range ctx.QsdevConfig.Services {
		if !validation.IsValidService(svc.Name) {
			allSvcsValid = false
		}
	}
	if allSvcsValid && len(ctx.QsdevConfig.Services) > 0 {
		results = append(results, CheckResult{
			Category: CategoryConfigIntegrity,
			Name:     "services_valid",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "All service names are valid",
		})
	}

	// Check profile name.
	if ctx.QsdevConfig.Profile != "" && ctx.ProfileNames != nil {
		found := false
		for _, p := range ctx.ProfileNames {
			if p == ctx.QsdevConfig.Profile {
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
				Message:  "Profile " + ctx.QsdevConfig.Profile + " is valid",
			})
		}
		// If not found, ValidateQsdevConfig already reported it.
	}

	return results
}

// configParseFailed reports whether the config is unavailable because it
// exists but could not be read or parsed (as opposed to simply not existing).
func configParseFailed(ctx CheckContext) bool {
	return ctx.ConfigErr != nil && !errors.Is(ctx.ConfigErr, fs.ErrNotExist)
}

// configUnavailableMessage explains why config-dependent checks are skipped,
// distinguishing a missing config from one that failed to parse.
func configUnavailableMessage(ctx CheckContext, skipping string) string {
	if configParseFailed(ctx) {
		return branding.Get().ConfigFile + " could not be parsed; skipping " + skipping
	}
	return "No " + branding.Get().ConfigFile + " found; skipping " + skipping
}
