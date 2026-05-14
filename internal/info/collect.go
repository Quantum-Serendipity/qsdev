package info

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	gdevconfig "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/config"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/state"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/toolreg"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/version"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// ErrNotGdevProject is returned when the project root does not contain a
// .gdev.yaml configuration file.
var ErrNotGdevProject = errors.New("not a gdev-managed project")

// CollectInfo reads cached state files to build a ProjectInfo.
// It reads only .gdev.yaml, .devinit/.gdev-init-state.yaml, and
// .devinit/.gdev-init-answers.yaml — no evaluation, no scanning.
func CollectInfo(projectRoot string) (*ProjectInfo, error) {
	// 1. Check .gdev.yaml exists.
	configPath := filepath.Join(projectRoot, ".gdev.yaml")
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotGdevProject
		}
		return nil, err
	}

	// 2. Parse .gdev.yaml.
	cfg, cfgErr := gdevconfig.ParseGdevConfig(configPath)

	// 3. Load state (graceful if missing).
	statePath := filepath.Join(projectRoot, ".devinit", ".gdev-init-state.yaml")
	genState, _ := state.LoadStateFromFile(statePath)

	// 4. Load answers (graceful if missing).
	answers := loadAnswersQuietly(projectRoot)

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
	info.GdevVersion = genState.GdevVersion
	if info.GdevVersion == "" {
		info.GdevVersion = version.Info().Version
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
	info.ClaudeCodeEnabled = answers.ClaudeCode
	if info.ProjectName == "" {
		info.ProjectName = answers.ProjectName
	}
	if info.SecurityProfile == "" && answers.ComplianceLevel != "" {
		info.SecurityProfile = answers.ComplianceLevel
	}

	// Ecosystems from answers if config didn't have them.
	if len(info.Ecosystems) == 0 {
		for _, lang := range answers.Languages {
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

// loadAnswersQuietly reads .devinit/.gdev-init-answers.yaml without errors.
func loadAnswersQuietly(projectRoot string) types.WizardAnswers {
	path := filepath.Join(projectRoot, ".devinit", ".gdev-init-answers.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return types.WizardAnswers{}
	}
	var answers types.WizardAnswers
	if err := yaml.Unmarshal(data, &answers); err != nil {
		return types.WizardAnswers{}
	}
	return answers
}
