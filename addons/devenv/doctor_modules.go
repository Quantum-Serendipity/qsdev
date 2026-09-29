package devenv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// moduleCheckSection statically runs the doctor checks that the ecosystem
// modules configured in the project at projectRoot contribute through
// ecosystem.DoctorCheckProvider (see doctor.EvaluateModuleCheck). No check
// command is executed. registry resolves module names; env.Declared is filled
// from the project's devenv modules. It returns nil outside a project and when
// no configured module contributes a check.
func moduleCheckSection(projectRoot string, registry *ecosystem.Registry, env doctor.ModuleCheckEnv) *doctor.ModuleCheckSection {
	if projectRoot == "" {
		return nil
	}
	cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(projectRoot, branding.Get().ConfigFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return doctor.NewModuleCheckSection(nil, []string{
			fmt.Sprintf("ecosystem checks not run: %v", err),
		})
	}

	type moduleChecks struct {
		name   string
		checks []ecosystem.DoctorCheck
	}
	var pending []moduleChecks
	for _, lang := range cfg.Languages {
		mod, ok := registry.ByName(lang.Name)
		if !ok {
			continue
		}
		dcp, ok := mod.(ecosystem.DoctorCheckProvider)
		if !ok {
			continue
		}
		mc := ecosystem.ModuleConfig{Version: lang.Version, PackageManager: lang.PackageManager}
		if checks := dcp.DoctorChecks(mc); len(checks) > 0 {
			pending = append(pending, moduleChecks{name: lang.Name, checks: checks})
		}
	}
	if len(pending) == 0 {
		return nil
	}

	var warnings []string
	declared, err := ProjectDeclaredEnv(projectRoot)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("environment checks judged without some devenv modules: %v", err))
	}
	env.Declared = declared

	var results []doctor.ModuleCheckInfo
	for _, p := range pending {
		for _, c := range p.checks {
			results = append(results, doctor.EvaluateModuleCheck(p.name, c, env))
		}
	}
	return doctor.NewModuleCheckSection(results, warnings)
}
