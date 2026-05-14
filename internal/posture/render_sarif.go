package posture

import (
	"encoding/json"
	"fmt"
)

// SARIF 2.1.0 types — local, unexported.
type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name            string          `json:"name"`
	Version         string          `json:"version"`
	InformationURI  string          `json:"informationUri"`
	Rules           []sarifRule     `json:"rules"`
}

type sarifRule struct {
	ID               string          `json:"id"`
	ShortDescription sarifMultiText  `json:"shortDescription"`
	DefaultConfig    sarifConfig     `json:"defaultConfiguration"`
}

type sarifMultiText struct {
	Text string `json:"text"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID    string           `json:"ruleId"`
	Level     string           `json:"level"`
	Message   sarifMultiText   `json:"message"`
	Locations []sarifLocation  `json:"locations,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

// allRules defines the 12 rule IDs and their default SARIF levels.
var allRules = []sarifRule{
	{ID: "gdev/defense-disabled", ShortDescription: sarifMultiText{Text: "A defense layer is disabled"}, DefaultConfig: sarifConfig{Level: "warning"}},
	{ID: "gdev/defense-partial", ShortDescription: sarifMultiText{Text: "A defense layer is only partially enabled"}, DefaultConfig: sarifConfig{Level: "warning"}},
	{ID: "gdev/config-missing", ShortDescription: sarifMultiText{Text: "A managed configuration file is missing"}, DefaultConfig: sarifConfig{Level: "error"}},
	{ID: "gdev/config-outdated", ShortDescription: sarifMultiText{Text: "A managed configuration file is outdated"}, DefaultConfig: sarifConfig{Level: "warning"}},
	{ID: "gdev/config-modified", ShortDescription: sarifMultiText{Text: "A machine-owned configuration file was modified"}, DefaultConfig: sarifConfig{Level: "warning"}},
	{ID: "gdev/vuln-critical", ShortDescription: sarifMultiText{Text: "Critical vulnerabilities detected"}, DefaultConfig: sarifConfig{Level: "error"}},
	{ID: "gdev/vuln-high", ShortDescription: sarifMultiText{Text: "High vulnerabilities detected"}, DefaultConfig: sarifConfig{Level: "warning"}},
	{ID: "gdev/lockfile-missing", ShortDescription: sarifMultiText{Text: "A lockfile is missing"}, DefaultConfig: sarifConfig{Level: "error"}},
	{ID: "gdev/lockfile-stale", ShortDescription: sarifMultiText{Text: "A lockfile is stale"}, DefaultConfig: sarifConfig{Level: "warning"}},
	{ID: "gdev/hooks-not-installed", ShortDescription: sarifMultiText{Text: "Git hooks are not installed"}, DefaultConfig: sarifConfig{Level: "warning"}},
	{ID: "gdev/markers-broken", ShortDescription: sarifMultiText{Text: "Section markers in CLAUDE.md are broken"}, DefaultConfig: sarifConfig{Level: "warning"}},
	{ID: "gdev/tool-unavailable", ShortDescription: sarifMultiText{Text: "A required tool binary is not on PATH"}, DefaultConfig: sarifConfig{Level: "warning"}},
}

// RenderSARIF produces a SARIF 2.1.0 JSON document from a PostureReport.
func RenderSARIF(report *PostureReport) ([]byte, error) {
	log := sarifLog{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:           "gdev",
						Version:        report.GdevVersion,
						InformationURI: "https://github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap",
						Rules:          allRules,
					},
				},
				Results: []sarifResult{},
			},
		},
	}

	results := &log.Runs[0].Results

	// Map disabled/partial defense layers.
	for _, l := range report.Defense.Layers {
		switch l.Status {
		case LayerDisabled:
			*results = append(*results, sarifResult{
				RuleID:  "gdev/defense-disabled",
				Level:   "warning",
				Message: sarifMultiText{Text: fmt.Sprintf("Defense layer %q is disabled: %s", l.Name, l.Reason)},
			})
		case LayerPartial:
			*results = append(*results, sarifResult{
				RuleID:  "gdev/defense-partial",
				Level:   "warning",
				Message: sarifMultiText{Text: fmt.Sprintf("Defense layer %q is partially enabled (%d/10): %s", l.Name, l.Score, l.Reason)},
			})
		}
		// LayerEnabled and LayerNotApplicable do not emit results.
	}

	// Map config health findings.
	for _, f := range report.Config.Files {
		switch f.State {
		case "missing":
			*results = append(*results, sarifResult{
				RuleID:  "gdev/config-missing",
				Level:   "error",
				Message: sarifMultiText{Text: fmt.Sprintf("Configuration file %q is missing", f.Path)},
				Locations: []sarifLocation{{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{URI: f.Path},
					},
				}},
			})
		case "outdated":
			*results = append(*results, sarifResult{
				RuleID:  "gdev/config-outdated",
				Level:   "warning",
				Message: sarifMultiText{Text: fmt.Sprintf("Configuration file %q is outdated", f.Path)},
				Locations: []sarifLocation{{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{URI: f.Path},
					},
				}},
			})
		case "modified":
			if f.Category == "machine-owned" {
				*results = append(*results, sarifResult{
					RuleID:  "gdev/config-modified",
					Level:   "warning",
					Message: sarifMultiText{Text: fmt.Sprintf("Machine-owned configuration file %q has been modified", f.Path)},
					Locations: []sarifLocation{{
						PhysicalLocation: sarifPhysicalLocation{
							ArtifactLocation: sarifArtifactLocation{URI: f.Path},
						},
					}},
				})
			}
		}
	}

	// Map vulnerability counts.
	if report.Dependencies.Totals.Critical > 0 {
		*results = append(*results, sarifResult{
			RuleID:  "gdev/vuln-critical",
			Level:   "error",
			Message: sarifMultiText{Text: fmt.Sprintf("%d critical vulnerability(ies) detected", report.Dependencies.Totals.Critical)},
		})
	}
	if report.Dependencies.Totals.High > 0 {
		*results = append(*results, sarifResult{
			RuleID:  "gdev/vuln-high",
			Level:   "warning",
			Message: sarifMultiText{Text: fmt.Sprintf("%d high vulnerability(ies) detected", report.Dependencies.Totals.High)},
		})
	}

	// Map drift findings to appropriate SARIF rules.
	for _, cat := range report.Drift.Categories {
		for _, f := range cat.Findings {
			ruleID, level := driftToSARIF(cat.Name, f)
			if ruleID == "" {
				continue // skip findings that don't map to a rule
			}
			r := sarifResult{
				RuleID:  ruleID,
				Level:   level,
				Message: sarifMultiText{Text: f.Description},
			}
			if f.Subject != "" && !isMetaSubject(f.Subject) {
				r.Locations = []sarifLocation{{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{URI: f.Subject},
					},
				}}
			}
			*results = append(*results, r)
		}
	}

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	return data, nil
}

// driftToSARIF maps a drift category + finding to a SARIF rule ID and level.
func driftToSARIF(categoryName string, f DriftFinding) (ruleID, level string) {
	switch categoryName {
	case categoryLockfileDrift:
		if f.Severity == DriftError {
			return "gdev/lockfile-missing", "error"
		}
		return "gdev/lockfile-stale", "warning"
	case categoryHookDrift:
		return "gdev/hooks-not-installed", "warning"
	case categoryMarkerIntegrity:
		return "gdev/markers-broken", "warning"
	case categoryToolAvailability:
		return "gdev/tool-unavailable", "warning"
	case categoryFileModification:
		switch f.Severity {
		case DriftError:
			return "gdev/config-missing", "error"
		case DriftWarning:
			return "gdev/config-modified", "warning"
		default:
			return "", "" // Info-level file modification is not a finding
		}
	default:
		return "", "" // Unknown categories are skipped
	}
}

// isMetaSubject returns true if the subject is a meta-reference rather than
// a file path (e.g. "pre-commit", "gdev version", "marker:xyz").
func isMetaSubject(s string) bool {
	switch {
	case s == "pre-commit", s == "commit-msg", s == ".git":
		return true
	case s == "gdev version":
		return true
	case len(s) > 7 && s[:7] == "marker:":
		return true
	default:
		return false
	}
}
