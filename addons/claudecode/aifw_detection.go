package claudecode

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

func (a *Adapter) FrameworkID() aiframework.FrameworkID { return aiframework.ClaudeCode }

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
			// Binary on PATH is supplemental evidence, not sufficient for detection.
			if binPath, err := exec.LookPath(m.Path); err == nil {
				det.Evidence = append(det.Evidence, m.Path+" binary found at "+binPath)
			}
		}
	}

	return det, nil
}

func (a *Adapter) Markers() []aiframework.DetectionMarker {
	return []aiframework.DetectionMarker{
		{Type: aiframework.MarkerDirectory, Path: ".claude", Weight: ecosystem.ConfidenceCertain},
		{Type: aiframework.MarkerFile, Path: "CLAUDE.md", Weight: ecosystem.ConfidenceProbable},
		{Type: aiframework.MarkerFile, Path: ".mcp.json", Weight: ecosystem.ConfidenceProbable},
		{Type: aiframework.MarkerBinary, Path: "claude", Weight: ecosystem.ConfidenceProbable},
	}
}
