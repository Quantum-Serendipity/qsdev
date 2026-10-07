package check

import (
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// CheckOrgOverlay reports whether the org overlay this run resolves is the
// one the CLI reads for the project (ctx.OrgConfigDrift): the overlay a human
// pinned with 'defaults pin', or the account's home overlay when none is
// pinned. With <EnvPrefix>ORG_CONFIG unset the run names the pinned overlay,
// so it passes and reports that overlay (ctx.OrgConfigSource). When the
// variable names another overlay, the CLI ignores it and reads the pinned one
// instead, so what the variable names and what regenerations apply differ.
func CheckOrgOverlay(ctx CheckContext) CheckResult {
	b := branding.Get()
	if ctx.OrgConfigDrift == "" {
		msg := "the org overlay is the one the CLI reads for the project"
		if ctx.OrgConfigSource != "" {
			msg = "the org overlay is " + ctx.OrgConfigSource
		}
		return CheckResult{
			Category: CategoryConfigIntegrity,
			Name:     "org_overlay_pinned",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  msg,
		}
	}
	return CheckResult{
		Category: CategoryConfigIntegrity,
		Name:     "org_overlay_pinned",
		Status:   StatusWarn,
		Severity: SeverityHigh,
		Message:  ctx.OrgConfigDrift + "; " + b.AppName + " ignores it",
		Remediation: "If you set " + b.EnvPrefix + "ORG_CONFIG yourself, run '" + b.AppName +
			" defaults pin' at your own terminal to approve that overlay for the project (--global for every project); " +
			"otherwise find what sets it (an env file, a devenv import, a shell startup file) and remove it",
	}
}

// CheckOrgOverlayLoaded reports an org overlay the catalog skipped
// (ctx.OrgOverlayErr): one that fails to parse or tries to loosen the
// built-in security floor. Every command that generates or changes the
// project refuses to run past it, so it fails at high severity; it reports
// nothing when the overlay applied or there is none.
func CheckOrgOverlayLoaded(ctx CheckContext) []CheckResult {
	if ctx.OrgOverlayErr == nil {
		return nil
	}
	return []CheckResult{{
		Category: CategoryConfigIntegrity,
		Name:     "config_catalog",
		Status:   StatusFail,
		Severity: SeverityHigh,
		Message:  catalog.LoadError(ctx.OrgOverlayErr).Error() + "; the org overlay is skipped",
		Remediation: "Run '" + catalog.ValidateCommand() + "' and fix the defaults file it names; " +
			branding.Get().AppName + " refuses to generate or change the project until it loads",
	}}
}
