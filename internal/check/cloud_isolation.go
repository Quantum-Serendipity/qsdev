package check

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/cloudisolation"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
)

// CheckCloudIsolation verifies the three credential isolation layers of each
// cloud provider the project configures, statically (no cloud CLI runs): the
// per-project environment variable declared in the devenv modules
// (ctx.DeclaredEnv), and the credential file masking and credential-command
// deny rules in .claude/settings.json.
//
// A missing masking path or deny rule fails at high severity: qsdev generates
// both, so their absence means the settings were weakened or predate the
// current rule set. A missing or placeholder environment variable only warns,
// since its value is account-specific and often lives in the untracked
// devenv.local.nix that a CI checkout does not have. A project set up
// without Claude Code and without .claude/settings.json is judged on
// environment separation only (see cloudisolation.Assess).
func CheckCloudIsolation(ctx CheckContext) []CheckResult {
	if ctx.QsdevConfig == nil {
		return nil // the config integrity check reports why
	}
	if len(cloudisolation.ConfiguredProviders(ctx.QsdevConfig.Languages)) == 0 {
		return nil
	}
	settings, err := cloudisolation.ReadSettings(ctx.ProjectRoot)
	if err != nil {
		return []CheckResult{{
			Category:    CategorySecurityHarden,
			Name:        "cloud_isolation_settings",
			Status:      StatusFail,
			Severity:    SeverityHigh,
			Message:     fmt.Sprintf("Cannot verify cloud credential isolation: %v", err),
			FilePath:    ClaudeSettingsRelPath,
			Remediation: "Fix " + ClaudeSettingsRelPath + " or run 'qsdev init --update' to regenerate it",
		}}
	}

	var results []CheckResult
	for _, report := range cloudisolation.Assess(ctx.QsdevConfig.Languages, ctx.DeclaredEnv, settings, claudeCodeConfigured(ctx)) {
		for _, s := range report.Statuses {
			results = append(results, cloudLayerResult(ctx, report.Provider, s))
		}
	}
	return results
}

// cloudLayerResult converts one layer status into a check result.
func cloudLayerResult(ctx CheckContext, provider cloudcommon.CloudProvider, s cloudcommon.FailSafeStatus) CheckResult {
	r := CheckResult{
		Category: CategorySecurityHarden,
		Name:     fmt.Sprintf("cloud_isolation_%s_%s", provider, strings.ReplaceAll(s.Layer.String(), " ", "_")),
		Message:  fmt.Sprintf("%s %s: %s", cloudcommon.DisplayName(provider), s.Layer, s.Details),
		Metadata: map[string]string{"provider": string(provider), "layer": s.Layer.String()},
	}
	if s.Active {
		r.Status, r.Severity = StatusPass, SeverityInfo
		return r
	}
	if cloudisolation.Enforced(s.Layer) {
		r.Status, r.Severity = StatusFail, SeverityHigh
		r.FilePath = ClaudeSettingsRelPath
		r.Remediation = "Run 'qsdev init --update' to restore the generated cloud credential rules in " + ClaudeSettingsRelPath
		return r
	}
	envVar := cloudcommon.EnvVarForProvider(provider)
	r.Status, r.Severity = StatusWarn, SeverityLow
	r.Remediation = fmt.Sprintf("Set env.%s in devenv.local.nix, or run 'qsdev init --env %s=<value>'", envVar, envVar)
	if ctx.DeclaredEnvErr != nil {
		r.Message += fmt.Sprintf(" (some devenv modules could not be read: %v)", ctx.DeclaredEnvErr)
	}
	return r
}
