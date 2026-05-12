package update

import (
	"github.com/pmezard/go-difflib/difflib"
)

// ComputeUnifiedDiff returns a unified diff string between old and new content.
func ComputeUnifiedDiff(oldContent, newContent []byte, oldName, newName string) (string, error) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(oldContent)),
		B:        difflib.SplitLines(string(newContent)),
		FromFile: oldName,
		ToFile:   newName,
		Context:  3,
	}
	return difflib.GetUnifiedDiffString(diff)
}
