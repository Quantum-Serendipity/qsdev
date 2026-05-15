package evidence

import (
	"fmt"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
)

// Generate produces an EvidenceReport by evaluating a compliance framework's
// controls against the current posture report. It iterates over all controls
// defined in the framework, derives their status from the posture's defense
// layers, computes a summary, and includes the standard disclaimer.
func Generate(fw *Framework, report *posture.PostureReport, projectName string) (*EvidenceReport, error) {
	if fw == nil {
		return nil, fmt.Errorf("framework must not be nil")
	}
	if report == nil {
		return nil, fmt.Errorf("posture report must not be nil")
	}

	controls := fw.Controls()
	mappings := make([]ControlMapping, 0, len(controls))

	for _, def := range controls {
		cm := DeriveControlMapping(def, report.Defense.Layers)
		mappings = append(mappings, cm)
	}

	summary := computeSummary(mappings)

	return &EvidenceReport{
		SchemaVersion: posture.SchemaVersion,
		GeneratedAt:   time.Now().UTC(),
		QsdevVersion:   version.Info().Version,
		ProjectName:   projectName,
		Framework:     fw.Name,
		FrameworkVer:  fw.Version,
		Disclaimer:    Disclaimer,
		Summary:       summary,
		Controls:      mappings,
		Posture:       report,
	}, nil
}

// computeSummary tallies control statuses and computes coverage percentage.
func computeSummary(mappings []ControlMapping) EvidenceSummary {
	var s EvidenceSummary
	s.TotalControls = len(mappings)

	for _, cm := range mappings {
		switch cm.Status {
		case StatusAddressed:
			s.AddressedFully++
		case StatusPartial:
			s.AddressedPartial++
		case StatusNotAddressed:
			s.NotAddressed++
		case StatusNotApplicable:
			s.NotApplicable++
		}
	}

	// Coverage percent = (fully + 0.5*partial) / (total - n/a) * 100
	applicable := s.TotalControls - s.NotApplicable
	if applicable > 0 {
		covered := float64(s.AddressedFully) + 0.5*float64(s.AddressedPartial)
		s.CoveragePercent = (covered / float64(applicable)) * 100.0
	} else {
		s.CoveragePercent = 100.0
	}

	return s
}
