package posture

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// ErrNotInitialized is returned when Assess is called on a project that has
// not been initialized with qsdev init (no state files or .qsdev.yaml found).
var ErrNotInitialized = errors.New("project not initialized: run 'qsdev init' first")

// Assess performs a security posture assessment of the project at projectPath.
// It loads all state files, checks that the project is initialized, and
// assembles a PostureReport. Returns ErrNotInitialized if no state files or
// .qsdev.yaml exist.
func Assess(projectPath string, opts AssessOptions) (*PostureReport, error) {
	// Validate that the project path exists.
	info, err := os.Stat(projectPath)
	if err != nil {
		return nil, fmt.Errorf("accessing project path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project path is not a directory: %s", projectPath)
	}

	// Load all state files.
	merged := LoadAllStates(projectPath)

	// Check for .qsdev.yaml as an alternative initialization indicator.
	qsdevYAMLExists := false
	if _, err := os.Stat(filepath.Join(projectPath, branding.Get().ConfigFile)); err == nil {
		qsdevYAMLExists = true
	}

	// If no state files were loaded and no .qsdev.yaml exists, the project
	// is not initialized.
	if !merged.HasAnyState() && !qsdevYAMLExists {
		return nil, ErrNotInitialized
	}

	// Determine project name from directory basename.
	projectName := filepath.Base(projectPath)

	// Get version info.
	buildInfo := version.Info()
	qsdevVersion := buildInfo.Version
	if merged.QsdevVersion != "" {
		qsdevVersion = merged.QsdevVersion
	}

	report := &PostureReport{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		QsdevVersion:  qsdevVersion,
		ProjectPath:   projectPath,
		ProjectName:   projectName,
		Score: AggregateScore{
			Grade: "U", // Unscored — will be filled by scoring logic.
		},
		Conformance: ConformanceResult{
			Baseline: ConformanceLevel{
				Checks: []ConformanceCheck{},
			},
			Enhanced: ConformanceLevel{
				Checks: []ConformanceCheck{},
			},
		},
		Defense: DefenseCoverage{
			Layers: []DefenseLayer{},
		},
		Config: ConfigHealth{
			Files: []ConfigFileInfo{},
		},
		Dependencies: DependencyHealth{
			Ecosystems: []EcosystemStatus{},
		},
		Drift: drift.Report{
			Categories: []drift.Category{},
			BySeverity: make(map[drift.Severity]int),
		},
		Tools:      []ToolStatus{},
		Ecosystems: []EcosystemStatus{},
	}

	// Determine progressive tier from config.
	currentTierName := "standard"
	configPath := filepath.Join(projectPath, branding.Get().ConfigFile)
	if cfg, cfgErr := config.ParseQsdevConfig(configPath); cfgErr == nil {
		if cfg.Tier != "" {
			currentTierName = cfg.Tier
		} else {
			currentTierName = tier.Infer(cfg.ClaudeCode.PermissionLevel, cfg.ClaudeCode.MCPServers).String()
		}
	}
	nextTierName, _ := tier.NextTier(currentTierName)
	report.Tier = TierInfo{
		Current:  currentTierName,
		Position: tier.Position(currentTierName),
		Total:    tier.Total(),
		NextTier: nextTierName,
	}

	// Record any state loading errors as drift findings.
	for _, loadErr := range merged.Errors {
		report.Drift.Categories = appendOrCreateCategory(
			report.Drift.Categories,
			"state-files",
			drift.Finding{
				Category:    "state-files",
				Severity:    drift.Warning,
				Subject:     loadErr.Path,
				Description: fmt.Sprintf("Failed to load state file: %s", loadErr.Err),
				Remediation: "Re-run 'qsdev init' to regenerate state files.",
				AutoFixable: true,
			},
		)
		report.Drift.TotalFindings++
		report.Drift.BySeverity[drift.Warning]++
	}

	return report, nil
}

// appendOrCreateCategory adds a finding to the named category, creating the
// category if it does not already exist.
func appendOrCreateCategory(categories []drift.Category, name string, finding drift.Finding) []drift.Category {
	for i, cat := range categories {
		if cat.Name == name {
			categories[i].Findings = append(categories[i].Findings, finding)
			return categories
		}
	}
	return append(categories, drift.Category{
		Name:     name,
		Findings: []drift.Finding{finding},
	})
}
