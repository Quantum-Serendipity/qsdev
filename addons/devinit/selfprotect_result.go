package devinit

import (
	"errors"
	"io/fs"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/gatedodge"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
)

// detectGuardedFileChange checks a Write/Edit/MultiEdit of a content-guarded
// file (gatedodge.Lookup). canonicalPath is the resolved target, or "" when it
// could not be resolved. Every entry matching the resolved target or the name
// the tool uses runs, the target's first, so a guarded name linked to another
// guarded file gets the target's checks too (claude.md -> .qsdev.yaml is
// GD-001) and a guarded name linked elsewhere stays guarded.
//
// Each entry's Introduced check scans the text the call introduces. Then the
// target is read once and the content the call leaves in it is reconstructed
// strictly (hookio.ResultingContent), and each Change check compares the two,
// so a call that deletes a protective setting (an Edit to an empty
// new_string, a Write that omits the line) is caught although it introduces
// nothing. It fails closed when the current content cannot be read, the
// result cannot be reconstructed, or a Change check cannot verify it.
func detectGuardedFileChange(toolName string, input hookio.ToolInput, canonicalPath string) (blocked bool, ruleID, reason string) {
	if !isWriteOrEditTool(toolName) {
		return false, "", ""
	}
	entries := gatedodge.Lookup(canonicalPath, input.FilePath)
	edited := input.EditedContent()
	var changed []gatedodge.GuardedFile
	for _, g := range entries {
		if g.Introduced != nil {
			if reason := g.Introduced(edited); reason != "" {
				return true, g.RuleID, reason
			}
		}
		if g.Change != nil {
			changed = append(changed, g)
		}
	}
	if len(changed) == 0 {
		return false, "", ""
	}

	target := canonicalPath
	if target == "" {
		target = input.FilePath
	}
	current, err := rules.ReadGuardedFile(target)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return true, changed[0].RuleID, "cannot read the current file to check what this change removes: " + err.Error()
	}
	after, ok := input.ResultingContent(toolName, string(current))
	if !ok {
		return true, changed[0].RuleID, "cannot determine the file's content after this change: old_string must match the current file exactly once"
	}
	for _, g := range changed {
		reason, err := g.Change(string(current), after)
		switch {
		case err != nil:
			return true, g.RuleID, "cannot verify change to security configuration: " + err.Error()
		case reason != "":
			return true, g.RuleID, reason
		}
	}
	return false, "", ""
}
