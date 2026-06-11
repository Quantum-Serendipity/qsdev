package check

import (
	"encoding/json"
	"io"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// SARIFSchemaURL is the canonical SARIF 2.1.0 JSON schema URL.
const SARIFSchemaURL = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json"

// SARIF 2.1.0 types — minimum viable subset.

type sarifReport struct {
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
	Name    string `json:"name"`
	Version string `json:"version"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
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

func formatSARIF(report *CheckReport, w io.Writer) error {
	sr := sarifReport{
		Schema:  SARIFSchemaURL,
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{
					Driver: sarifDriver{
						Name:    branding.Get().AppName,
						Version: report.Version,
					},
				},
			},
		},
	}

	var results []sarifResult
	for _, r := range report.Checks {
		if r.Status != StatusFail {
			continue
		}

		result := sarifResult{
			RuleID:  r.Name,
			Level:   sarifSeverity(r.Severity),
			Message: sarifMessage{Text: r.Message},
		}

		if r.FilePath != "" {
			result.Locations = []sarifLocation{
				{
					PhysicalLocation: sarifPhysicalLocation{
						ArtifactLocation: sarifArtifactLocation{
							URI: r.FilePath,
						},
					},
				},
			}
		}

		results = append(results, result)
	}

	sr.Runs[0].Results = results

	data, err := json.MarshalIndent(sr, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

func sarifSeverity(s CheckSeverity) string {
	switch s {
	case SeverityCritical, SeverityHigh:
		return "error"
	case SeverityMedium:
		return "warning"
	case SeverityLow, SeverityInfo:
		return "note"
	default:
		return "note"
	}
}
