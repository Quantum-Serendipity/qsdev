package info

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ErrNotQsdevProject is returned when the project root does not contain a
// .qsdev.yaml configuration file.
var ErrNotQsdevProject = errors.New("not a qsdev-managed project")

// CollectInfo reads cached state files to build a ProjectInfo.
// It reads only .qsdev.yaml, .devinit/.qsdev-init-state.yaml, and
// .devinit/.qsdev-init-answers.yaml — no evaluation, no scanning.
func CollectInfo(projectRoot string) (*ProjectInfo, error) {
	// 1. Check .qsdev.yaml exists.
	configPath := filepath.Join(projectRoot, branding.Get().ConfigFile)
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotQsdevProject
		}
		return nil, err
	}

	// 2. Parse .qsdev.yaml.
	cfg, cfgErr := qsdevconfig.ParseQsdevConfig(configPath)

	// 3. Load state (graceful if missing).
	statePath := filepath.Join(projectRoot, branding.Get().StateDir, "."+branding.Get().AppName+"-init-state.yaml")
	genState, _ := state.LoadStateFromFile(statePath)

	// 4. Load answers (graceful if missing).
	wizardAnswers := loadAnswersBestEffort(projectRoot)

	// 5. Build ProjectInfo.
	info := &ProjectInfo{
		ToolsByCategory: make(map[string]int),
	}

	// From config.
	if cfgErr == nil && cfg != nil {
		info.ConfigVersion = cfg.Version
		if cfg.Security.Level != "" {
			info.SecurityProfile = cfg.Security.Level
		}
		for _, lang := range cfg.Languages {
			info.Ecosystems = append(info.Ecosystems, lang.Name)
		}
	}

	// From state.
	info.QsdevVersion = genState.QsdevVersion
	if info.QsdevVersion == "" {
		info.QsdevVersion = version.Info().Version
	}
	info.LastUpdated = genState.LastRun
	info.ManagedFileCount = len(genState.Files)

	// Count active tools from state.
	for _, enabled := range genState.EnabledTools {
		if enabled {
			info.ActiveToolCount++
		}
	}

	// Tool breakdown by category from registry.
	registry := toolreg.DefaultRegistry()
	for _, tool := range registry.All() {
		if genState.EnabledTools[tool.Name] {
			cat := tool.Category.DisplayName()
			info.ToolsByCategory[cat]++
		}
	}

	// From answers.
	info.ClaudeCodeEnabled = wizardAnswers.ClaudeCode
	info.ProjectName = wizardAnswers.ProjectName
	if info.SecurityProfile == "" && wizardAnswers.ComplianceLevel != "" {
		info.SecurityProfile = wizardAnswers.ComplianceLevel
	}

	// Ecosystems from answers if config didn't have them.
	if len(info.Ecosystems) == 0 {
		for _, lang := range wizardAnswers.Languages {
			info.Ecosystems = append(info.Ecosystems, lang.Name)
		}
	}

	// Defaults.
	if info.ProjectName == "" {
		info.ProjectName = filepath.Base(projectRoot)
	}
	if info.SecurityProfile == "" {
		info.SecurityProfile = "standard"
	}

	return info, nil
}

// loadAnswersBestEffort reads the primary answers file. A missing file yields
// zero-value answers; an unreadable or corrupt file is logged and also yields
// zero-value answers, so `info` still reports what it can without hiding the
// corruption.
func loadAnswersBestEffort(projectRoot string) types.WizardAnswers {
	a, err := answers.LoadPrimary(projectRoot)
	if err != nil {
		slog.Warn("ignoring unreadable answers file", "path", answers.PrimaryPath(projectRoot), "error", err)
		return types.WizardAnswers{}
	}
	return a
}
