package doctor

import (
	"github.com/Quantum-Serendipity/qsdev/internal/cloudisolation"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cloudcommon"
)

// NewCloudSection builds the doctor's cloud credential isolation section from
// static fail-safe reports (see cloudisolation.Assess). warnings carry inputs
// that could not be read, such as an unparsable devenv.local.nix. It returns
// nil when there is nothing to report.
func NewCloudSection(reports []cloudcommon.FailSafeReport, warnings []string) *CloudSection {
	if len(reports) == 0 && len(warnings) == 0 {
		return nil
	}
	cs := &CloudSection{Detected: true, Warnings: warnings}
	for _, r := range reports {
		info := CloudProviderInfo{
			Name:        string(r.Provider),
			DisplayName: cloudcommon.DisplayName(r.Provider),
			Status:      cloudisolation.Status(r),
		}
		for _, s := range r.Statuses {
			info.Layers = append(info.Layers, CloudLayerInfo{
				Name:     s.Layer.String(),
				Active:   s.Active,
				Enforced: cloudisolation.Enforced(s.Layer),
				Detail:   s.Details,
			})
		}
		cs.Providers = append(cs.Providers, info)
	}
	return cs
}
