package aiframework

import (
	"github.com/Quantum-Serendipity/qsdev/internal/enumtext"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// DetectionAdapter detects whether a specific AI framework is configured in a project.
type DetectionAdapter interface {
	FrameworkID() FrameworkID
	Detect(projectRoot string) (*FrameworkDetection, error)
	Markers() []DetectionMarker
}

// FrameworkDetection holds the result of probing for a framework's presence.
type FrameworkDetection struct {
	Detected    bool
	Confidence  ecosystem.Confidence
	Evidence    []string
	CLIVersion  string
	ConfigPaths []string
}

// MarkerType categorizes what kind of filesystem artifact a detection marker looks for.
type MarkerType int

const (
	MarkerDirectory MarkerType = iota
	MarkerFile
	MarkerBinary
)

var markerTypeNames = [...]string{
	MarkerDirectory: "directory",
	MarkerFile:      "file",
	MarkerBinary:    "binary",
}

var markerTypeText = enumtext.New[MarkerType]("MarkerType", "marker type", "unknown", markerTypeNames[:])

func (m MarkerType) String() string { return markerTypeText.String(m) }

func (m MarkerType) MarshalText() ([]byte, error) { return markerTypeText.MarshalText(m) }

func (m *MarkerType) UnmarshalText(text []byte) error { return markerTypeText.UnmarshalText(text, m) }

// DetectionMarker describes a filesystem artifact that indicates framework presence.
type DetectionMarker struct {
	Type   MarkerType
	Path   string
	Weight ecosystem.Confidence
}
