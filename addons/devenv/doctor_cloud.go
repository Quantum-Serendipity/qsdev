package devenv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/cloudisolation"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// cloudIsolationSection statically assesses the credential isolation of each
// cloud provider the project at projectRoot configures (see
// cloudisolation.Assess): the declared devenv environment, the Claude Code
// deny rules and the sandbox read-deny list. No cloud CLI runs. It returns nil
// outside a project and for a project with no cloud provider; an input that
// cannot be read becomes a warning in the section.
func cloudIsolationSection(projectRoot string) *doctor.CloudSection {
	if projectRoot == "" {
		return nil
	}
	cfgFile := branding.Get().ConfigFile
	cfg, err := qsdevconfig.ParseQsdevConfig(filepath.Join(projectRoot, cfgFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return doctor.NewCloudSection(nil, []string{
			fmt.Sprintf("cloud credential isolation not assessed: %v", err),
		})
	}
	if len(cloudisolation.ConfiguredProviders(cfg.Languages)) == 0 {
		return nil
	}

	var warnings []string
	env, err := ProjectDeclaredEnv(projectRoot)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("environment separation judged without some devenv modules: %v", err))
	}
	settings, err := cloudisolation.ReadSettings(projectRoot)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("credential masking and deny rules judged as absent: %v", err))
	}
	claudeCode := check.ClaudeCodeConfigured(cfg, filepath.Join(projectRoot, state.InitStateFile()))
	return doctor.NewCloudSection(cloudisolation.Assess(cfg.Languages, env, settings, claudeCode), warnings)
}
