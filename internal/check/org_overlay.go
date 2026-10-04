package check

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// CheckOrgOverlay reports whether the org overlay this run resolves is the
// one the CLI reads for the project (ctx.OrgConfigDrift): the overlay a human
// pinned with 'defaults pin', or the account's home overlay when none is
// pinned. When it is not, the CLI ignores it and reads that one instead, so
// what <EnvPrefix>ORG_CONFIG names and what regenerations apply differ.
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
