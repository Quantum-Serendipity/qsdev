package ecosystem

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// SetupWarnings returns the warnings each of the answers' languages' module
// reports for the project at projectRoot (see SetupWarner), in language order,
// each prefixed with the module's display name. Each module sees the
// configuration generation uses (ToGenerationConfig), except that the
// release-age window is left unset (DefaultMinReleaseAge): no setup warning
// depends on it, and this package cannot resolve the compliance level's
// window from the catalog. Languages without a registered module contribute
// nothing. When the project sets a registry proxy, a module that installs
// packages (non-empty PackageManagers) but does not route this configuration
// through the proxy (see ProxyKeyProvider) is reported too, so the proxy is
// never assumed to cover it.
func (r *Registry) SetupWarnings(projectRoot string, answers types.WizardAnswers) []string {
	proxySet := answers.Infrastructure.RegistryProxyBase() != ""
	var warnings []string
	for _, lang := range answers.Languages {
		m, ok := r.ByName(lang.Name)
		if !ok {
			continue
		}
		cfg := ToGenerationConfig(lang, answers, 0)
		var msgs []string
		if proxySet {
			msgs = append(msgs, unsupportedProxyWarning(m, cfg)...)
		}
		if w, ok := m.(SetupWarner); ok {
			msgs = append(msgs, w.SetupWarnings(projectRoot, cfg)...)
		}
		for _, msg := range msgs {
			warnings = append(warnings, m.DisplayName()+": "+msg)
		}
	}
	return warnings
}

// unsupportedProxyWarning reports that m's package managers bypass the
// registry proxy with config, or nothing when m has no package managers or
// routes config through the proxy.
func unsupportedProxyWarning(m EcosystemModule, config ModuleConfig) []string {
	pms := m.PackageManagers()
	if len(pms) == 0 || moduleProxyKey(m, config) != "" {
		return nil
	}
	names := make([]string, len(pms))
	for i, pm := range pms {
		names[i] = pm.Name
	}
	return []string{"registry_proxy is set but " + m.DisplayName() + " has no proxy support; " +
		strings.Join(names, ", ") + " will fetch directly from the public registries"}
}
