package check

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// CheckOrgOverlay reports whether the org overlay this run resolves is the
// one a run no human started reads for the project (ctx.OrgConfigDrift). When
// it is not, such a run (an agent's regeneration) ignores it and reads the
// overlay a human recorded at init instead, so what this run would apply and
// what the agent's runs apply differ.
func CheckOrgOverlay(ctx CheckContext) CheckResult {
	b := branding.Get()
	if ctx.OrgConfigDrift == "" {
		return CheckResult{
			Category: CategoryConfigIntegrity,
			Name:     "org_overlay_pinned",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "the org overlay is the one recorded for the project",
		}
	}
	return CheckResult{
		Category: CategoryConfigIntegrity,
		Name:     "org_overlay_pinned",
		Status:   StatusWarn,
		Severity: SeverityHigh,
		Message:  ctx.OrgConfigDrift + "; runs no human starts ignore it",
		Remediation: "If you set " + b.EnvPrefix + "ORG_CONFIG yourself, run '" + b.AppName +
			" init' at your own terminal to record that overlay; otherwise find what sets it (an env file, a devenv import, a shell startup file) and remove it",
	}
}
