package teardown

import (
	"os"
	"path/filepath"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/state"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/toolreg"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// ClassifiedFile describes a tracked file's ownership and modification state.
type ClassifiedFile struct {
	Path       string
	Owner      string
	Ownership  toolreg.OwnershipType
	SectionIDs []string // For shared files: all section IDs from all tools.
	Modified   bool
	Deleted    bool
}

// ClassifyFiles examines each file in genState against its on-disk state
// and cross-references the tool registry to determine ownership.
func ClassifyFiles(genState types.GeneratedState, projectRoot string, registry *toolreg.Registry) []ClassifiedFile {
	if len(genState.Files) == 0 {
		return nil
	}

	// Build a lookup: path -> []FileOwnership (from all tools).
	type ownerInfo struct {
		toolName   string
		ownership  toolreg.OwnershipType
		sectionIDs []string
	}
	pathOwners := make(map[string][]ownerInfo)
	for _, tool := range registry.All() {
		for _, fo := range tool.OwnedFiles {
			info := ownerInfo{
				toolName:  tool.Name,
				ownership: fo.Ownership,
			}
			if fo.Ownership == toolreg.Shared && fo.SectionID != "" {
				info.sectionIDs = []string{fo.SectionID}
			}
			pathOwners[fo.Path] = append(pathOwners[fo.Path], info)
		}
	}

	var classified []ClassifiedFile

	for relPath, fs := range genState.Files {
		cf := ClassifiedFile{
			Path:  relPath,
			Owner: fs.Owner,
		}

		absPath := filepath.Join(projectRoot, relPath)

		// Check if file exists.
		_, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				cf.Deleted = true
			}
			// For other errors, treat as deleted for teardown purposes.
			if !os.IsNotExist(err) {
				cf.Deleted = true
			}
		}

		// Check if modified.
		if !cf.Deleted {
			currentHash, hashErr := state.ComputeFileHash(absPath)
			if hashErr != nil {
				cf.Modified = true // Can't read -> treat as modified (preserve).
			} else if currentHash != fs.Hash {
				cf.Modified = true
			}
		}

		// Determine ownership from registry.
		owners, found := pathOwners[relPath]
		if !found {
			// Not in any tool's OwnedFiles -> default to Exclusive.
			cf.Ownership = toolreg.Exclusive
		} else {
			// Determine if shared or exclusive.
			isShared := false
			var sectionIDs []string
			for _, oi := range owners {
				if oi.ownership == toolreg.Shared {
					isShared = true
					sectionIDs = append(sectionIDs, oi.sectionIDs...)
				}
			}
			if isShared {
				cf.Ownership = toolreg.Shared
				cf.SectionIDs = dedup(sectionIDs)
			} else {
				cf.Ownership = toolreg.Exclusive
			}
		}

		classified = append(classified, cf)
	}

	return classified
}

// dedup removes duplicate strings while preserving order.
func dedup(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	var result []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
