package info

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Unknown is reported for a value that cannot be determined, such as a
// generator version the state file does not record. info never substitutes a
// plausible-looking value (the running binary's version, a guessed profile).
const Unknown = "unknown"

// SecurityProfileUnknown is reported when the project configuration cannot be
// parsed, or the default profile cannot be derived from the catalog, so a
// broken project is never displayed as a healthy default profile.
const SecurityProfileUnknown = Unknown

// ErrNotQsdevProject is returned when the project root does not contain a
// .qsdev.yaml configuration file.
var ErrNotQsdevProject = errors.New("not a qsdev-managed project")

// CollectInfo reads cached state files to build a ProjectInfo.
// It reads only .qsdev.yaml, the init state file (state.InitStateFile) and
// the wizard answers — no evaluation, no scanning. Category counts come from
// the tool registry, and a security profile set by neither the config nor the
// answers defaults to the catalog's compliance level for its default tier. A
// registry or catalog that cannot be loaded is reported in Warnings, like an
// unreadable config or state file.
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

	// 3. Load state (graceful if missing; LoadStateFromFile returns an empty
	// state for a missing file, so any error here means it is unreadable).
	statePath := filepath.Join(projectRoot, filepath.FromSlash(state.InitStateFile()))
	genState, stateErr := state.LoadStateFromFile(statePath)

	// 4. Load answers (graceful if missing).
	wizardAnswers := loadAnswersBestEffort(projectRoot)

	// 5. Build ProjectInfo.
	info := &ProjectInfo{
		ToolsByCategory: make(map[string]int),
	}

	// Surface unreadable project files instead of silently reporting defaults
	// that make a broken project look healthy.
	if cfgErr != nil {
		info.Warnings = append(info.Warnings, fmt.Sprintf("config could not be parsed: %v", cfgErr))
		info.SecurityProfile = SecurityProfileUnknown
	}
	if stateErr != nil {
		info.Warnings = append(info.Warnings, fmt.Sprintf("state could not be loaded: %v", stateErr))
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
		info.QsdevVersion = Unknown
	}
	info.LastUpdated = genState.LastRun
	info.ManagedFileCount = len(genState.Files)

	// Count active tools from state.
	for _, enabled := range genState.EnabledTools {
		if enabled {
			info.ActiveToolCount++
		}
	}

	// Tool breakdown by category from the registry rather than the catalog:
	// the registry also holds tools that providers register outside catalog
	// metadata (the operation skills), which catalog-only counts would miss.
	// A catalog that does not load (a rejected defaults file) is reported,
	// not fatal: the bug report this feeds is what a person reaches for after
	// that failure.
	if registry, err := toolreg.Default(); err != nil {
		info.Warnings = append(info.Warnings, fmt.Sprintf("tool registry could not be built: %v", err))
	} else {
		for _, tool := range registry.All() {
			if genState.EnabledTools[tool.Name] {
				cat := tool.Category.DisplayName()
				info.ToolsByCategory[cat]++
			}
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
		info.SecurityProfile = defaultSecurityProfile()
	}

	return info, nil
}

// defaultSecurityProfile returns the catalog's compliance level for its
// default tier, or SecurityProfileUnknown when the catalog cannot be loaded.
// It adds no warning: the registry warning already reports a catalog failure.
// It is called only when nothing else sets the profile, so a healthy project
// with an explicit level never loads the catalog for it.
func defaultSecurityProfile() string {
	cat, err := catalog.Default()
	if err != nil {
		return SecurityProfileUnknown
	}
	if c := cat.TierCompliance(cat.DefaultTier()); c != "" {
		return c
	}
	return SecurityProfileUnknown
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
