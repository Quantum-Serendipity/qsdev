package posture

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/toolreg"
)

const categoryMarkerIntegrity = "Section Marker Integrity"

var (
	openMarkerRe  = regexp.MustCompile(`<!-- gdev:(\S+) -->`)
	closeMarkerRe = regexp.MustCompile(`<!-- /gdev:(\S+) -->`)
)

// detectMarkerIntegrity checks that section markers in CLAUDE.md are properly
// paired and that all expected markers from enabled tools are present.
func detectMarkerIntegrity(projectDir string, enabledTools map[string]bool) DriftCategory {
	cat := DriftCategory{Name: categoryMarkerIntegrity}

	claudeMDPath := filepath.Join(projectDir, "CLAUDE.md")
	content, err := os.ReadFile(claudeMDPath)
	if err != nil {
		if os.IsNotExist(err) {
			cat.Findings = append(cat.Findings, DriftFinding{
				Category:    categoryMarkerIntegrity,
				Severity:    DriftError,
				Subject:     "CLAUDE.md",
				Description: "CLAUDE.md does not exist",
				Remediation: "Run gdev init or gdev update to generate CLAUDE.md",
			})
		} else {
			cat.Findings = append(cat.Findings, DriftFinding{
				Category:    categoryMarkerIntegrity,
				Severity:    DriftInfo,
				Subject:     "CLAUDE.md",
				Description: fmt.Sprintf("Unable to read CLAUDE.md: %v", err),
			})
		}
		return cat
	}

	text := string(content)

	// Find all opening and closing markers.
	openMatches := openMarkerRe.FindAllStringSubmatch(text, -1)
	closeMatches := closeMarkerRe.FindAllStringSubmatch(text, -1)

	openSet := make(map[string]bool, len(openMatches))
	for _, m := range openMatches {
		openSet[m[1]] = true
	}

	closeSet := make(map[string]bool, len(closeMatches))
	for _, m := range closeMatches {
		closeSet[m[1]] = true
	}

	// Check for unpaired markers.
	for id := range openSet {
		if !closeSet[id] {
			cat.Findings = append(cat.Findings, DriftFinding{
				Category:    categoryMarkerIntegrity,
				Severity:    DriftWarning,
				Subject:     fmt.Sprintf("marker:%s", id),
				Description: fmt.Sprintf("Opening marker <!-- gdev:%s --> has no matching closing marker", id),
				Remediation: fmt.Sprintf("Add <!-- /gdev:%s --> or run gdev update", id),
			})
		}
	}

	for id := range closeSet {
		if !openSet[id] {
			cat.Findings = append(cat.Findings, DriftFinding{
				Category:    categoryMarkerIntegrity,
				Severity:    DriftWarning,
				Subject:     fmt.Sprintf("marker:%s", id),
				Description: fmt.Sprintf("Closing marker <!-- /gdev:%s --> has no matching opening marker", id),
				Remediation: fmt.Sprintf("Add <!-- gdev:%s --> or run gdev update", id),
			})
		}
	}

	// Determine expected markers from enabled tools.
	expectedMarkers := expectedClaudeMDMarkers(enabledTools)
	for _, marker := range expectedMarkers {
		if !openSet[marker] && !closeSet[marker] {
			cat.Findings = append(cat.Findings, DriftFinding{
				Category:    categoryMarkerIntegrity,
				Severity:    DriftWarning,
				Subject:     fmt.Sprintf("marker:%s", marker),
				Description: fmt.Sprintf("Expected marker pair for %q is entirely missing from CLAUDE.md", marker),
				Remediation: fmt.Sprintf("Run gdev update to add the %s section to CLAUDE.md", marker),
			})
		}
	}

	return cat
}

// expectedClaudeMDMarkers returns the set of SectionIDs that should appear
// in CLAUDE.md based on the enabled tools.
func expectedClaudeMDMarkers(enabledTools map[string]bool) []string {
	registry := toolreg.DefaultRegistry()
	var markers []string
	seen := make(map[string]bool)

	for toolName, enabled := range enabledTools {
		if !enabled {
			continue
		}
		tool, ok := registry.ByName(toolName)
		if !ok {
			continue
		}
		for _, owned := range tool.OwnedFiles {
			if owned.Path == "CLAUDE.md" && owned.SectionID != "" && !seen[owned.SectionID] {
				seen[owned.SectionID] = true
				markers = append(markers, owned.SectionID)
			}
		}
	}

	return markers
}
