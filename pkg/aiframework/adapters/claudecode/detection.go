package claudecode

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// Detect probes projectRoot for the Claude Code artifacts returned by
// Markers(). A .claude/ directory, CLAUDE.md or .mcp.json marks the project as
// detected; the claude binary on PATH is recorded as supplemental evidence
// only, since an installed CLI says nothing about this project.
func (a *Adapter) Detect(projectRoot string) (*aiframework.FrameworkDetection, error) {
	det := &aiframework.FrameworkDetection{}

	for _, m := range a.Markers() {
		switch m.Type {
		case aiframework.MarkerDirectory, aiframework.MarkerFile:
			fullPath := filepath.Join(projectRoot, m.Path)
			if _, err := os.Stat(fullPath); err == nil {
				det.Detected = true
				det.Evidence = append(det.Evidence, m.Path+" found")
				det.ConfigPaths = append(det.ConfigPaths, fullPath)
				if m.Weight > det.Confidence {
					det.Confidence = m.Weight
				}
			}
		case aiframework.MarkerBinary:
			if binPath, err := exec.LookPath(m.Path); err == nil {
				det.Evidence = append(det.Evidence, m.Path+" binary found at "+binPath)
			}
		}
	}

	return det, nil
}

// Markers reports the filesystem artifacts that indicate Claude Code is
// configured: the .claude/ directory, CLAUDE.md, .mcp.json, and the claude
// binary on PATH.
func (a *Adapter) Markers() []aiframework.DetectionMarker {
	return []aiframework.DetectionMarker{
		{Type: aiframework.MarkerDirectory, Path: ".claude", Weight: ecosystem.ConfidenceCertain},
		{Type: aiframework.MarkerFile, Path: "CLAUDE.md", Weight: ecosystem.ConfidenceProbable},
		{Type: aiframework.MarkerFile, Path: ".mcp.json", Weight: ecosystem.ConfidenceProbable},
		{Type: aiframework.MarkerBinary, Path: "claude", Weight: ecosystem.ConfidenceProbable},
	}
}
