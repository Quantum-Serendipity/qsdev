package teardown

import (
	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ClassifiedFile describes a tracked file's ownership and modification state.
type ClassifiedFile struct {
	Path       string
	Owner      string
	Ownership  toolreg.OwnershipType
	SectionIDs []string // For shared files: all section IDs from all tools.
	Modified   bool
	Deleted    bool
	// BaseContent is the generated content recorded in state (set for
	// three-way-merged files such as settings.json).
	BaseContent []byte
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

	// Share the modification check with update/check so a chmod-only change is
	// "modified" everywhere (and the file is preserved rather than removed).
	statuses := state.CheckModified(genState, projectRoot)

	var classified []ClassifiedFile

	for relPath, fs := range genState.Files {
		cf := ClassifiedFile{
			Path:        relPath,
			Owner:       fs.Owner,
			BaseContent: fs.BaseContent,
		}

		switch statuses[relPath].Status {
		case types.Deleted:
			cf.Deleted = true
		case types.Unmodified:
			// Safe to remove or clean.
		default:
			// Modified, or Unknown (unreadable): preserve.
			cf.Modified = true
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
				cf.SectionIDs = sliceutil.Dedup(sectionIDs)
			} else {
				cf.Ownership = toolreg.Exclusive
			}
		}

		classified = append(classified, cf)
	}

	return classified
}
