package posture

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/version"
)

// ErrNotInitialized is returned when Assess is called on a project that has
// not been initialized with gdev init (no state files or .gdev.yaml found).
var ErrNotInitialized = errors.New("project not initialized: run 'gdev init' first")

// Assess performs a security posture assessment of the project at projectPath.
// It loads all state files, checks that the project is initialized, and
// assembles a PostureReport. Returns ErrNotInitialized if no state files or
// .gdev.yaml exist.
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

	// Check for .gdev.yaml as an alternative initialization indicator.
	gdevYAMLExists := false
	if _, err := os.Stat(filepath.Join(projectPath, ".gdev.yaml")); err == nil {
		gdevYAMLExists = true
	}

	// If no state files were loaded and no .gdev.yaml exists, the project
	// is not initialized.
	if !merged.HasAnyState() && !gdevYAMLExists {
		return nil, ErrNotInitialized
	}

	// Determine project name from directory basename.
	projectName := filepath.Base(projectPath)

	// Get version info.
	buildInfo := version.Info()
	gdevVersion := buildInfo.Version
	if merged.GdevVersion != "" {
		gdevVersion = merged.GdevVersion
	}

	report := &PostureReport{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		GdevVersion:   gdevVersion,
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
		Drift: DriftReport{
			Categories: []DriftCategory{},
			BySeverity: make(map[DriftSeverity]int),
		},
		Tools:      []ToolStatus{},
		Ecosystems: []EcosystemStatus{},
	}

	// Record any state loading errors as drift findings.
	for _, loadErr := range merged.Errors {
		report.Drift.Categories = appendOrCreateCategory(
			report.Drift.Categories,
			"state-files",
			DriftFinding{
				Category:    "state-files",
				Severity:    DriftWarning,
				Subject:     loadErr.Path,
				Description: fmt.Sprintf("Failed to load state file: %s", loadErr.Err),
				Remediation: "Re-run 'gdev init' to regenerate state files.",
				AutoFixable: true,
			},
		)
		report.Drift.TotalFindings++
		report.Drift.BySeverity[DriftWarning]++
	}

	return report, nil
}

// appendOrCreateCategory adds a finding to the named category, creating the
// category if it does not already exist.
func appendOrCreateCategory(categories []DriftCategory, name string, finding DriftFinding) []DriftCategory {
	for i, cat := range categories {
		if cat.Name == name {
			categories[i].Findings = append(categories[i].Findings, finding)
			return categories
		}
	}
	return append(categories, DriftCategory{
		Name:     name,
		Findings: []DriftFinding{finding},
	})
}
