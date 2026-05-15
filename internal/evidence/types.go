package evidence

import (
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

// EvidenceReport is the top-level compliance evidence report that maps
// a compliance framework's controls to qsdev's defense-in-depth layers.
type EvidenceReport struct {
	SchemaVersion string                 `json:"schemaVersion"`
	GeneratedAt   time.Time              `json:"generatedAt"`
	QsdevVersion   string                 `json:"qsdevVersion"`
	ProjectName   string                 `json:"projectName"`
	Framework     string                 `json:"framework"`
	FrameworkVer  string                 `json:"frameworkVersion"`
	Disclaimer    string                 `json:"disclaimer"`
	Summary       EvidenceSummary        `json:"summary"`
	Controls      []ControlMapping       `json:"controls"`
	Posture       *posture.PostureReport `json:"posture"`
}

// EvidenceSummary summarizes control coverage across the framework.
type EvidenceSummary struct {
	TotalControls    int     `json:"totalControls"`
	AddressedFully   int     `json:"addressedFully"`
	AddressedPartial int     `json:"addressedPartially"`
	NotAddressed     int     `json:"notAddressed"`
	NotApplicable    int     `json:"notApplicable"`
	CoveragePercent  float64 `json:"coveragePercent"`
}

// ControlStatus represents how well a control is addressed by qsdev.
type ControlStatus string

const (
	StatusAddressed     ControlStatus = "addressed"
	StatusPartial       ControlStatus = "partial"
	StatusNotAddressed  ControlStatus = "not-addressed"
	StatusNotApplicable ControlStatus = "not-applicable"
)

// ControlMapping maps a single compliance control to qsdev defense layers
// and supporting artifacts.
type ControlMapping struct {
	ControlID   string             `json:"controlId"`
	ControlName string             `json:"controlName"`
	ControlDesc string             `json:"controlDesc"`
	Category    string             `json:"category"`
	Status      ControlStatus      `json:"status"`
	GdevLayers  []LayerEvidence    `json:"qsdevLayers"`
	Artifacts   []EvidenceArtifact `json:"artifacts"`
	Notes       string             `json:"notes,omitempty"`
}

// LayerEvidence describes how a single qsdev defense layer relates to a control.
type LayerEvidence struct {
	LayerName   string `json:"layerName"`
	Status      string `json:"status"`
	Relevance   string `json:"relevance"` // "primary"|"supporting"
	Description string `json:"description"`
}

// EvidenceArtifact references a file, scan result, or other artifact
// that supports the control mapping.
type EvidenceArtifact struct {
	Type        string `json:"type"` // "config-file"|"scan-result"|"tool-version"
	Path        string `json:"path"`
	Description string `json:"description"`
	Hash        string `json:"hash,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}
