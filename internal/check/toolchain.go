package check

import (
	"fmt"
	"regexp"

	"golang.org/x/mod/semver"

	"github.com/Quantum-Serendipity/qsdev/internal/toolcheck"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ToolProber runs binary with versionArg, as toolcheck.Detect does, and
// reports what it found.
type ToolProber func(binary, versionArg string) toolcheck.Info

// toolVersionRe finds the first dotted version number in a tool's version
// output ("11.19.0", "npm 11.19.0", "v11.19.0").
var toolVersionRe = regexp.MustCompile(`\d+(?:\.\d+){1,2}`)

// checkToolchainRequirements probes the tools a configured language's
// generated security settings depend on (see
// ecosystem.ToolchainRequirementProvider) and fails when the tool on PATH is
// too old to honour them, since such a tool ignores the setting silently.
func checkToolchainRequirements(ctx CheckContext, lang types.LanguageConfig) []CheckResult {
	mod, ok := ecosystem.DefaultRegistry().ByName(lang.Name)
	if !ok {
		return nil
	}
	provider, ok := mod.(ecosystem.ToolchainRequirementProvider)
	if !ok {
		return nil
	}
	reqs := provider.ToolchainRequirements(ecosystem.ModuleConfig{
		Version:        lang.Version,
		PackageManager: lang.PackageManager,
	})

	results := make([]CheckResult, 0, len(reqs))
	for _, req := range reqs {
		results = append(results, checkToolchainRequirement(ctx.ProbeTool, lang.Name, req))
	}
	return results
}

// checkToolchainRequirement probes one required tool version.
func checkToolchainRequirement(probe ToolProber, langName string, req ecosystem.ToolchainRequirement) CheckResult {
	result := CheckResult{
		Category: CategorySecurityHarden,
		Name:     "toolchain_" + langName + "_" + req.Binary,
		Severity: SeverityInfo,
	}
	if probe == nil {
		result.Status = StatusSkip
		result.Message = fmt.Sprintf("No tool prober configured; cannot verify that %s supports %s", req.Binary, req.Setting)
		return result
	}

	info := probe(req.Binary, req.VersionArg)
	if !info.Found {
		result.Status = StatusSkip
		result.Message = fmt.Sprintf("%s is not on PATH; run the check inside the devenv shell to verify it supports %s", req.Binary, req.Setting)
		return result
	}

	found := toolVersionRe.FindString(info.Version)
	if found == "" {
		found = toolVersionRe.FindString(info.Output)
	}
	if found == "" || !semver.IsValid("v"+found) {
		result.Status = StatusWarn
		result.Severity = SeverityLow
		result.Message = fmt.Sprintf("Could not read the version of %s (%s); %s needs %s >= %s", req.Binary, info.Path, req.Setting, req.Binary, req.MinVersion)
		return result
	}

	if semver.Compare("v"+found, "v"+req.MinVersion) < 0 {
		result.Status = StatusFail
		result.Severity = SeverityHigh
		result.Message = fmt.Sprintf("%s %s (%s) ignores %s; it needs %s >= %s, so the setting is not enforced", req.Binary, found, info.Path, req.Setting, req.Binary, req.MinVersion)
		result.Remediation = fmt.Sprintf("Run '%s init --update' (it regenerates devenv.nix and refreshes devenv.lock) so the devenv shell provides %s >= %s, then re-enter the shell ('direnv reload') and run the check there", branding.Get().AppName, req.Binary, req.MinVersion)
		return result
	}

	result.Status = StatusPass
	result.Message = fmt.Sprintf("%s %s supports %s", req.Binary, found, req.Setting)
	return result
}
