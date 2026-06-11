package render

import (
	"encoding/json"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string         `json:"id"`
	ShortDescription sarifMultiText `json:"shortDescription"`
	DefaultConfig    sarifConfig    `json:"defaultConfiguration"`
}

type sarifMultiText struct {
	Text string `json:"text"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMultiText  `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
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

// ruleID builds a SARIF rule ID from the branding prefix and a suffix.
func ruleID(suffix string) string {
	return branding.Get().AppName + "/" + suffix
}

// buildAllRules constructs the 12 rule definitions using branded IDs.
func buildAllRules() []sarifRule {
	return []sarifRule{
		{ID: ruleID("defense-disabled"), ShortDescription: sarifMultiText{Text: "A defense layer is disabled"}, DefaultConfig: sarifConfig{Level: "warning"}},
		{ID: ruleID("defense-partial"), ShortDescription: sarifMultiText{Text: "A defense layer is only partially enabled"}, DefaultConfig: sarifConfig{Level: "warning"}},
		{ID: ruleID("config-missing"), ShortDescription: sarifMultiText{Text: "A managed configuration file is missing"}, DefaultConfig: sarifConfig{Level: "error"}},
		{ID: ruleID("config-outdated"), ShortDescription: sarifMultiText{Text: "A managed configuration file is outdated"}, DefaultConfig: sarifConfig{Level: "warning"}},
		{ID: ruleID("config-modified"), ShortDescription: sarifMultiText{Text: "A machine-owned configuration file was modified"}, DefaultConfig: sarifConfig{Level: "warning"}},
		{ID: ruleID("vuln-critical"), ShortDescription: sarifMultiText{Text: "Critical vulnerabilities detected"}, DefaultConfig: sarifConfig{Level: "error"}},
		{ID: ruleID("vuln-high"), ShortDescription: sarifMultiText{Text: "High vulnerabilities detected"}, DefaultConfig: sarifConfig{Level: "warning"}},
		{ID: ruleID("lockfile-missing"), ShortDescription: sarifMultiText{Text: "A lockfile is missing"}, DefaultConfig: sarifConfig{Level: "error"}},
		{ID: ruleID("lockfile-stale"), ShortDescription: sarifMultiText{Text: "A lockfile is stale"}, DefaultConfig: sarifConfig{Level: "warning"}},
		{ID: ruleID("hooks-not-installed"), ShortDescription: sarifMultiText{Text: "Git hooks are not installed"}, DefaultConfig: sarifConfig{Level: "warning"}},
		{ID: ruleID("markers-broken"), ShortDescription: sarifMultiText{Text: "Section markers in CLAUDE.md are broken"}, DefaultConfig: sarifConfig{Level: "warning"}},
		{ID: ruleID("tool-unavailable"), ShortDescription: sarifMultiText{Text: "A required tool binary is not on PATH"}, DefaultConfig: sarifConfig{Level: "warning"}},
	}
}

// Category name constants for mapping drift findings to SARIF rules.
const (
	categoryLockfileDrift    = "Lock File Drift"
	categoryHookDrift        = "Pre-Commit Hook Drift"
	categoryMarkerIntegrity  = "Section Marker Integrity"
	categoryToolAvailability = "Tool Availability"
	categoryFileModification = "File Modification"
)

// RenderSARIF produces a SARIF 2.1.0 JSON document from a PostureReport.
func RenderSARIF(report *posture.PostureReport) ([]byte, error) {
	log := sarifLog{
		Schema:  check.SARIFSchemaURL,
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:           branding.Get().AppName,
						Version:        report.QsdevVersion,
						InformationURI: fmt.Sprintf("https://github.com/%s/%s", branding.Get().GitHubOwner, branding.Get().GitHubRepo),
						Rules:          buildAllRules(),
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
		case posture.LayerDisabled:
			*results = append(*results, sarifResult{
				RuleID:  ruleID("defense-disabled"),
				Level:   "warning",
				Message: sarifMultiText{Text: fmt.Sprintf("Defense layer %q is disabled: %s", l.Name, l.Reason)},
			})
		case posture.LayerPartial:
			*results = append(*results, sarifResult{
				RuleID:  ruleID("defense-partial"),
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
				RuleID:  ruleID("config-missing"),
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
				RuleID:  ruleID("config-outdated"),
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
					RuleID:  ruleID("config-modified"),
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
			RuleID:  ruleID("vuln-critical"),
			Level:   "error",
			Message: sarifMultiText{Text: fmt.Sprintf("%d critical vulnerability(ies) detected", report.Dependencies.Totals.Critical)},
		})
	}
	if report.Dependencies.Totals.High > 0 {
		*results = append(*results, sarifResult{
			RuleID:  ruleID("vuln-high"),
			Level:   "warning",
			Message: sarifMultiText{Text: fmt.Sprintf("%d high vulnerability(ies) detected", report.Dependencies.Totals.High)},
		})
	}

	// Map drift findings to appropriate SARIF rules.
	for _, cat := range report.Drift.Categories {
		for _, f := range cat.Findings {
			id, level := driftToSARIF(cat.Name, f)
			if id == "" {
				continue // skip findings that don't map to a rule
			}
			r := sarifResult{
				RuleID:  id,
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
func driftToSARIF(categoryName string, f drift.Finding) (id, level string) {
	switch categoryName {
	case categoryLockfileDrift:
		if f.Severity == drift.Error {
			return ruleID("lockfile-missing"), "error"
		}
		return ruleID("lockfile-stale"), "warning"
	case categoryHookDrift:
		return ruleID("hooks-not-installed"), "warning"
	case categoryMarkerIntegrity:
		return ruleID("markers-broken"), "warning"
	case categoryToolAvailability:
		return ruleID("tool-unavailable"), "warning"
	case categoryFileModification:
		switch f.Severity {
		case drift.Error:
			return ruleID("config-missing"), "error"
		case drift.Warning:
			return ruleID("config-modified"), "warning"
		default:
			return "", "" // Info-level file modification is not a finding
		}
	default:
		return "", "" // Unknown categories are skipped
	}
}

// isMetaSubject returns true if the subject is a meta-reference rather than
// a file path (e.g. "pre-commit", "qsdev version", "marker:xyz").
func isMetaSubject(s string) bool {
	switch {
	case s == "pre-commit", s == "commit-msg", s == ".git":
		return true
	case s == branding.Get().AppName+" version":
		return true
	case len(s) > 7 && s[:7] == "marker:":
		return true
	default:
		return false
	}
}
