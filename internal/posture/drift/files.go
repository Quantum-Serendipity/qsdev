package drift

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const categoryFileModification = "File Modification"

// detectFileModification checks whether generated files have been modified,
// deleted, or are in an unexpected state compared to the stored generation state.
func detectFileModification(projectDir string, genState types.GeneratedState) Category {
	cat := Category{Name: categoryFileModification}

	if len(genState.Files) == 0 {
		return cat
	}

	statuses := state.CheckModified(genState, projectDir)

	for path, fs := range statuses {
		storedFile := genState.Files[path]

		switch fs.Status {
		case types.Modified:
			switch storedFile.Strategy {
			case types.Overwrite, types.LibraryManaged:
				// Machine-owned files should not be edited manually.
				cat.Findings = append(cat.Findings, Finding{
					Category:    categoryFileModification,
					Severity:    Warning,
					Subject:     path,
					Description: fmt.Sprintf("Machine-owned file %q has been modified (strategy: %s)", path, storedFile.Strategy),
					Expected:    fs.StoredHash,
					Actual:      fs.CurrentHash,
					Remediation: "Run qsdev update to regenerate this file",
					AutoFixable: true,
				})
			case types.SectionMarker, types.ThreeWayMerge:
				// Human-edited files are expected to diverge.
				cat.Findings = append(cat.Findings, Finding{
					Category:    categoryFileModification,
					Severity:    Info,
					Subject:     path,
					Description: fmt.Sprintf("Human-edited file %q has been modified (strategy: %s)", path, storedFile.Strategy),
					Expected:    fs.StoredHash,
					Actual:      fs.CurrentHash,
				})
			default:
				cat.Findings = append(cat.Findings, Finding{
					Category:    categoryFileModification,
					Severity:    Warning,
					Subject:     path,
					Description: fmt.Sprintf("File %q has been modified (strategy: %s)", path, storedFile.Strategy),
					Expected:    fs.StoredHash,
					Actual:      fs.CurrentHash,
					Remediation: "Review the file and decide whether to keep changes or regenerate",
				})
			}

		case types.Deleted:
			cat.Findings = append(cat.Findings, Finding{
				Category:    categoryFileModification,
				Severity:    Error,
				Subject:     path,
				Description: fmt.Sprintf("Generated file %q has been deleted", path),
				Remediation: "Run qsdev update to regenerate this file",
				AutoFixable: true,
			})

		case types.Unknown:
			cat.Findings = append(cat.Findings, Finding{
				Category:    categoryFileModification,
				Severity:    Info,
				Subject:     path,
				Description: fmt.Sprintf("Unable to determine status of %q: %v", path, fs.Error),
			})
		}
	}

	return cat
}
