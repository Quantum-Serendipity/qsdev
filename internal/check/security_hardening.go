package check

import (
	"os"
	"path/filepath"
)

// lockFileMapping maps language names to their expected lock file(s).
var lockFileMapping = map[string][]string{
	"go":         {"go.sum"},
	"javascript": {"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb"},
	"python":     {"requirements.txt", "poetry.lock", "uv.lock", "Pipfile.lock"},
	"rust":       {"Cargo.lock"},
	"java":       {"gradle.lockfile", "pom.xml"},
	"dotnet":     {"packages.lock.json"},
	"ruby":       {"Gemfile.lock"},
	"php":        {"composer.lock"},
}

// CheckSecurityHardening verifies security-related file presence for
// detected ecosystems.
func CheckSecurityHardening(ctx CheckContext) []CheckResult {
	if ctx.GdevConfig == nil {
		return []CheckResult{
			{
				Category: CategorySecurityHarden,
				Name:     "security_hardening",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  "No .gdev.yaml found; skipping security hardening checks",
			},
		}
	}

	if len(ctx.GdevConfig.Languages) == 0 {
		return []CheckResult{
			{
				Category: CategorySecurityHarden,
				Name:     "security_hardening",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  "No languages configured; skipping security hardening checks",
			},
		}
	}

	var results []CheckResult

	// Check lock files for each language.
	for _, lang := range ctx.GdevConfig.Languages {
		lockFiles, ok := lockFileMapping[lang.Name]
		if !ok {
			continue
		}

		found := false
		for _, lf := range lockFiles {
			p := filepath.Join(ctx.ProjectRoot, lf)
			if _, err := os.Stat(p); err == nil {
				found = true
				break
			}
		}

		if !found {
			results = append(results, CheckResult{
				Category:    CategorySecurityHarden,
				Name:        "lockfile_" + lang.Name,
				Status:      StatusFail,
				Severity:    SeverityMedium,
				Message:     "No lock file found for " + lang.Name,
				Remediation: "Generate a lock file to pin dependency versions",
			})
		} else {
			results = append(results, CheckResult{
				Category: CategorySecurityHarden,
				Name:     "lockfile_" + lang.Name,
				Status:   StatusPass,
				Severity: SeverityInfo,
				Message:  "Lock file found for " + lang.Name,
			})
		}
	}

	// Check .npmrc for JavaScript projects.
	results = append(results, checkJSHardening(ctx)...)

	// Check Python hardening.
	results = append(results, checkPythonHardening(ctx)...)

	if len(results) == 0 {
		results = append(results, CheckResult{
			Category: CategorySecurityHarden,
			Name:     "security_hardening",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "No ecosystem-specific hardening checks applicable",
		})
	}

	return results
}

func checkJSHardening(ctx CheckContext) []CheckResult {
	hasJS := false
	for _, lang := range ctx.GdevConfig.Languages {
		if lang.Name == "javascript" {
			hasJS = true
			break
		}
	}
	if !hasJS {
		return nil
	}

	npmrcPath := filepath.Join(ctx.ProjectRoot, ".npmrc")
	if _, err := os.Stat(npmrcPath); err != nil {
		return []CheckResult{
			{
				Category:    CategorySecurityHarden,
				Name:        "npmrc_exists",
				Status:      StatusFail,
				Severity:    SeverityMedium,
				Message:     ".npmrc not found for JavaScript project",
				Remediation: "Create .npmrc with security settings (e.g., package-lock=true, ignore-scripts=true)",
			},
		}
	}

	return []CheckResult{
		{
			Category: CategorySecurityHarden,
			Name:     "npmrc_exists",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  ".npmrc found for JavaScript project",
		},
	}
}

func checkPythonHardening(ctx CheckContext) []CheckResult {
	hasPython := false
	for _, lang := range ctx.GdevConfig.Languages {
		if lang.Name == "python" {
			hasPython = true
			break
		}
	}
	if !hasPython {
		return nil
	}

	// Check for pip.conf or pyproject.toml.
	pipConfPath := filepath.Join(ctx.ProjectRoot, "pip.conf")
	pyprojectPath := filepath.Join(ctx.ProjectRoot, "pyproject.toml")

	_, pipErr := os.Stat(pipConfPath)
	_, pyprojectErr := os.Stat(pyprojectPath)

	if pipErr != nil && pyprojectErr != nil {
		return []CheckResult{
			{
				Category:    CategorySecurityHarden,
				Name:        "python_config_exists",
				Status:      StatusFail,
				Severity:    SeverityMedium,
				Message:     "Neither pip.conf nor pyproject.toml found for Python project",
				Remediation: "Create pyproject.toml with dependency and security configuration",
			},
		}
	}

	return []CheckResult{
		{
			Category: CategorySecurityHarden,
			Name:     "python_config_exists",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "Python configuration file found",
		},
	}
}
