package check

import (
	"bufio"
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// CheckSecurityHardening verifies lock files and security configuration
// settings for the configured ecosystems.
func CheckSecurityHardening(ctx CheckContext) []CheckResult {
	if ctx.QsdevConfig == nil {
		return []CheckResult{
			{
				Category: CategorySecurityHarden,
				Name:     "security_hardening",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  configUnavailableMessage(ctx, "security hardening checks"),
			},
		}
	}

	if len(ctx.QsdevConfig.Languages) == 0 {
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
	for _, lang := range ctx.QsdevConfig.Languages {
		results = append(results, checkLanguageLockFiles(ctx.ProjectRoot, lang)...)
	}

	// Check that the package-manager security configs qsdev generates for
	// JavaScript and Python still carry their hardening settings.
	for _, lang := range ctx.QsdevConfig.Languages {
		switch lang.Name {
		case ecosystem.NameJavaScript, ecosystem.NamePython:
			results = append(results, checkSecurityConfigSettings(ctx.ProjectRoot, lang)...)
		}
	}

	// Check that the tools on PATH are new enough to honour those settings.
	for _, lang := range ctx.QsdevConfig.Languages {
		results = append(results, checkToolchainRequirements(ctx, lang)...)
	}

	if len(results) == 0 {
		// Nothing was verified, so report a skip rather than a pass: a green
		// result here would claim hardening that was never checked.
		results = append(results, CheckResult{
			Category: CategorySecurityHarden,
			Name:     "security_hardening",
			Status:   StatusSkip,
			Severity: SeverityInfo,
			Message:  "No ecosystem-specific hardening checks applicable",
		})
	}

	return results
}

// checkLanguageLockFiles returns the lock file results for one configured
// language. An ecosystem whose manifest declares no dependencies is skipped
// (its package manager writes no lock file). Ecosystems in the shared
// LockFilesByEcosystem catalog use checkLockFile; every other ecosystem falls
// back to the lock files its module declares via ManifestFileProvider, so
// ecosystems such as Terraform (.terraform.lock.hcl) are not silently passed.
func checkLanguageLockFiles(projectRoot string, lang types.LanguageConfig) []CheckResult {
	mod, hasMod := ecosystem.DefaultRegistry().ByName(lang.Name)
	if hasMod {
		if dd, ok := mod.(ecosystem.DependencyDeclarer); ok {
			// An error leaves the answer unknown: enforce the lock file.
			if declares, err := dd.DeclaresDependencies(projectRoot); err == nil && !declares {
				return []CheckResult{{
					Category: CategorySecurityHarden,
					Name:     "lockfile_" + lang.Name,
					Status:   StatusSkip,
					Severity: SeverityInfo,
					Message:  fmt.Sprintf("%s manifest declares no dependencies; no lock file expected", lang.Name),
				}}
			}
		}
	}

	if r, ok := checkLockFile(projectRoot, lang.Name); ok {
		return []CheckResult{r}
	}
	if !hasMod {
		return nil
	}
	mfp, ok := mod.(ecosystem.ManifestFileProvider)
	if !ok {
		return nil
	}
	var results []CheckResult
	for _, mf := range mfp.ManifestFiles(ecosystem.ModuleConfig{PackageManager: lang.PackageManager}) {
		if r, ok := checkDeclaredLockFile(projectRoot, lang.Name, mf); ok {
			results = append(results, r)
		}
	}
	return results
}

// checkDeclaredLockFile checks one module-declared manifest/lock file pair.
// It applies only when the pair has a lock file with a required or
// recommended policy and the manifest (which may be a glob such as "*.tf") is
// present in the project root, where ecosystem detection looks. A missing
// required lock file fails; a missing recommended one warns.
func checkDeclaredLockFile(projectRoot, langName string, mf ecosystem.ManifestFileInfo) (CheckResult, bool) {
	if mf.LockFile == "" || mf.LockFilePolicy == ecosystem.LockFilePolicyNone {
		return CheckResult{}, false
	}
	matches, err := filepath.Glob(filepath.Join(projectRoot, mf.Path))
	if err != nil || len(matches) == 0 {
		return CheckResult{}, false
	}

	name := "lockfile_" + langName
	if _, err := os.Stat(filepath.Join(projectRoot, mf.LockFile)); err == nil {
		return CheckResult{
			Category: CategorySecurityHarden,
			Name:     name,
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("Lock file %s found for %s", mf.LockFile, langName),
			FilePath: mf.LockFile,
		}, true
	}

	status, severity := StatusFail, SeverityMedium
	if mf.LockFilePolicy == ecosystem.LockFilePolicyRecommended {
		status, severity = StatusWarn, SeverityLow
	}
	return CheckResult{
		Category:    CategorySecurityHarden,
		Name:        name,
		Status:      status,
		Severity:    severity,
		Message:     fmt.Sprintf("No lock file found for %s (%s expected alongside %s)", langName, mf.LockFile, mf.Path),
		FilePath:    mf.LockFile,
		Remediation: fmt.Sprintf("Generate and commit %s to pin dependency versions", mf.LockFile),
	}, true
}

// checkLockFile verifies that a pinned lock file exists for the ecosystem. A
// file that is also the ecosystem's dependency manifest (pom.xml,
// requirements.txt, vcpkg.json) is never accepted as proof of pinning on its
// own: it yields a warning rather than a pass. ok is false when the ecosystem
// has no known lock files.
func checkLockFile(projectRoot, langName string) (CheckResult, bool) {
	lockFiles, ok := ecosystem.LockFilesByEcosystem[langName]
	if !ok {
		return CheckResult{}, false
	}
	manifests := ecosystem.ManifestsByEcosystem[langName]

	var manifestFound string
	var pureLockFiles []string
	for _, lf := range lockFiles {
		isManifest := slices.Contains(manifests, lf)
		if !isManifest {
			pureLockFiles = append(pureLockFiles, lf)
		}
		if _, err := os.Stat(filepath.Join(projectRoot, lf)); err != nil {
			continue
		}
		if !isManifest {
			return CheckResult{
				Category: CategorySecurityHarden,
				Name:     "lockfile_" + langName,
				Status:   StatusPass,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("Lock file %s found for %s", lf, langName),
				FilePath: lf,
			}, true
		}
		if manifestFound == "" {
			manifestFound = lf
		}
	}

	remediation := "Generate a lock file to pin dependency versions"
	if len(pureLockFiles) > 0 {
		remediation += " (" + strings.Join(pureLockFiles, ", ") + ")"
	}

	if manifestFound != "" {
		return CheckResult{
			Category:    CategorySecurityHarden,
			Name:        "lockfile_" + langName,
			Status:      StatusWarn,
			Severity:    SeverityMedium,
			Message:     fmt.Sprintf("Only %s found for %s; it is a dependency manifest and does not by itself pin versions", manifestFound, langName),
			FilePath:    manifestFound,
			Remediation: remediation + ", or make sure " + manifestFound + " pins exact versions",
		}, true
	}

	return CheckResult{
		Category:    CategorySecurityHarden,
		Name:        "lockfile_" + langName,
		Status:      StatusFail,
		Severity:    SeverityMedium,
		Message:     "No lock file found for " + langName,
		Remediation: remediation,
	}, true
}

// checkSecurityConfigSettings verifies that each security config file the
// ecosystem module generates for the configured package manager exists and
// still contains the hardening settings the generator writes. The expected
// settings are derived from the module itself so this check cannot drift
// from what qsdev generates.
func checkSecurityConfigSettings(projectRoot string, lang types.LanguageConfig) []CheckResult {
	name := "security_config_" + lang.Name

	mod, ok := ecosystem.DefaultRegistry().ByName(lang.Name)
	if !ok {
		return []CheckResult{{
			Category: CategorySecurityHarden,
			Name:     name,
			Status:   StatusWarn,
			Severity: SeverityLow,
			Message:  fmt.Sprintf("No ecosystem module registered for %s; cannot verify its security configuration", lang.Name),
		}}
	}

	var results []CheckResult
	for _, gf := range mod.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: lang.PackageManager}) {
		results = append(results, checkConfigFileSettings(projectRoot, name, lang.Name, gf))
	}
	return results
}

// checkConfigFileSettings compares one generated security config file against
// its on-disk counterpart.
func checkConfigFileSettings(projectRoot, name, langName string, gf types.GeneratedFile) CheckResult {
	remediation := "Run 'qsdev init --update' or 'qsdev repair' to restore the security settings in " + gf.Path

	data, err := os.ReadFile(filepath.Join(projectRoot, filepath.FromSlash(gf.Path)))
	if err != nil {
		return CheckResult{
			Category:    CategorySecurityHarden,
			Name:        name,
			Status:      StatusFail,
			Severity:    SeverityMedium,
			Message:     fmt.Sprintf("%s security config %s could not be read: %v", langName, gf.Path, err),
			FilePath:    gf.Path,
			Remediation: remediation,
		}
	}

	missing := missingSettings(parseSettings(gf.Content), parseSettings(data))
	if len(missing) > 0 {
		return CheckResult{
			Category:    CategorySecurityHarden,
			Name:        name,
			Status:      StatusFail,
			Severity:    SeverityMedium,
			Message:     fmt.Sprintf("%s is missing or weakens hardening settings: %s", gf.Path, strings.Join(missing, ", ")),
			FilePath:    gf.Path,
			Remediation: remediation,
		}
	}

	return CheckResult{
		Category: CategorySecurityHarden,
		Name:     name,
		Status:   StatusPass,
		Severity: SeverityInfo,
		Message:  fmt.Sprintf("%s contains the required hardening settings", gf.Path),
		FilePath: gf.Path,
	}
}

// parseSettings extracts top-level "key = value" (INI/TOML/.npmrc) and
// "key: value" (YAML) settings from a config file. Comments, section headers,
// and indented (nested) lines are ignored; values are unquoted and have
// trailing inline comments removed.
func parseSettings(content []byte) map[string]string {
	settings := make(map[string]string)
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := sc.Text()
		if line == "" || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "[") {
			continue
		}

		sep := strings.IndexAny(line, "=:")
		if sep <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:sep])
		value := strings.TrimSpace(line[sep+1:])
		if i := strings.Index(value, " #"); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}
		settings[key] = strings.Trim(value, `"'`)
	}
	return settings
}

// missingSettings lists each expected setting that is absent from, or weaker
// in, the actual settings, formatted as "key=value".
func missingSettings(expected, actual map[string]string) []string {
	var missing []string
	for _, key := range slices.Sorted(maps.Keys(expected)) {
		want := expected[key]
		got, ok := actual[key]
		if !ok || !settingSatisfied(want, got) {
			missing = append(missing, key+"="+want)
		}
	}
	return missing
}

// settingSatisfied reports whether got satisfies the generated value want:
// equal (booleans compared case-insensitively), or, for numeric thresholds
// with the same unit suffix (e.g. release-age gates "3", "4320", "7d"), at
// least as strict.
func settingSatisfied(want, got string) bool {
	if strings.EqualFold(want, got) {
		return true
	}
	wantN, wantUnit, okW := splitNumber(want)
	gotN, gotUnit, okG := splitNumber(got)
	return okW && okG && wantUnit == gotUnit && gotN >= wantN
}

// splitNumber splits a value such as "7d" or "4320" into its numeric part and
// unit suffix.
func splitNumber(s string) (float64, string, bool) {
	end := 0
	for end < len(s) && (s[end] >= '0' && s[end] <= '9' || s[end] == '.') {
		end++
	}
	if end == 0 {
		return 0, "", false
	}
	n, err := strconv.ParseFloat(s[:end], 64)
	if err != nil {
		return 0, "", false
	}
	return n, s[end:], true
}
