package aiframework

import (
	"fmt"

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

func (m MarkerType) String() string {
	if int(m) >= 0 && int(m) < len(markerTypeNames) {
		return markerTypeNames[m]
	}
	return "unknown"
}

func (m MarkerType) MarshalText() ([]byte, error) {
	s := m.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown MarkerType value %d", int(m))
	}
	return []byte(s), nil
}

func (m *MarkerType) UnmarshalText(text []byte) error {
	for i, name := range markerTypeNames {
		if name == string(text) {
			*m = MarkerType(i)
			return nil
		}
	}
	return fmt.Errorf("unknown marker type: %q", string(text))
}

// DetectionMarker describes a filesystem artifact that indicates framework presence.
type DetectionMarker struct {
	Type   MarkerType
	Path   string
	Weight ecosystem.Confidence
}
