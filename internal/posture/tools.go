package posture

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
)

// buildToolStatuses lists every tool in the catalog, plus any enabled tool the
// catalog does not know, with whether it is enabled and whether the binary it
// needs is on PATH. The result is sorted by name. loadRegistry supplies the
// tool registry (toolreg.Default outside tests); its error is returned.
func buildToolStatuses(loadRegistry func() (*toolreg.Registry, error), enabledTools map[string]bool) ([]ToolStatus, error) {
	reg, err := loadRegistry()
	if err != nil {
		return nil, fmt.Errorf("loading the tool registry for the tool listing: %w", err)
	}
	statuses := []ToolStatus{}
	known := make(map[string]bool)
	for _, t := range reg.All() {
		known[t.Name] = true
		statuses = append(statuses, ToolStatus{
			Name:        t.Name,
			DisplayName: t.DisplayName,
			Category:    string(t.Category),
			Enabled:     enabledTools[t.Name],
			Available:   drift.ToolAvailable(t.Name),
			ConfigFile:  exclusiveFile(t),
			Description: t.Description,
		})
	}
	for name, enabled := range enabledTools {
		if known[name] {
			continue
		}
		statuses = append(statuses, ToolStatus{
			Name:        name,
			DisplayName: name,
			Enabled:     enabled,
			Available:   drift.ToolAvailable(name),
		})
	}
	slices.SortFunc(statuses, func(a, b ToolStatus) int { return strings.Compare(a.Name, b.Name) })
	return statuses, nil
}

// exclusiveFile returns the first file the tool owns outright, which is its
// configuration file, or "" when it only contributes to shared files.
func exclusiveFile(t *toolreg.Tool) string {
	if files := t.ExclusiveFiles(); len(files) > 0 {
		return files[0].Path
	}
	return ""
}
