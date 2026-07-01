package claudecode

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// Detect probes projectRoot for Claude Code artifacts (the .claude/ directory,
// CLAUDE.md, .mcp.json, and the claude binary on PATH) by delegating to the
// real addon detection logic, which walks the markers returned by Markers().
func (a *Adapter) Detect(projectRoot string) (*aiframework.FrameworkDetection, error) {
	det, err := a.addon.Detect(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("detecting claude code project: %w", err)
	}
	return det, nil
}

// Markers reports the filesystem artifacts that indicate Claude Code is
// configured, delegating to the addon's marker definitions so detection stays
// in lock-step with the addon.
func (a *Adapter) Markers() []aiframework.DetectionMarker {
	return a.addon.Markers()
}
