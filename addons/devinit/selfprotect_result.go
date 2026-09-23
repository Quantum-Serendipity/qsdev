package devinit

import (
	"errors"
	"io/fs"
	"os"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/gatedodge"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
)

// detectResultGateDodge checks the content a Write/Edit/MultiEdit leaves in a
// guarded config file against the file's current content, so a call that
// deletes a protective setting (an Edit to an empty new_string, a Write that
// omits the line) is caught even though it introduces no text for
// gatedodge.Detect to scan. canonicalPath is the resolved target, or "" when
// it could not be resolved. The rule is chosen by the name the tool uses or
// the resolved name, so a guarded file reached through a symlink stays guarded;
// the content is read from the resolved target. It fails closed when the
// result cannot be determined.
func detectResultGateDodge(toolName string, input hookio.ToolInput, canonicalPath string) (bool, string, string) {
	if !isWriteOrEditTool(toolName) {
		return false, "", ""
	}
	target := canonicalPath
	if target == "" {
		target = input.FilePath
	}
	rule := gatedodge.ResultRuleFor(input.FilePath)
	if rule == nil {
		rule = gatedodge.ResultRuleFor(target)
	}
	if rule == nil {
		return false, "", ""
	}

	current, err := os.ReadFile(target)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return true, rule.ID, "cannot read the current file to check what this change removes: " + err.Error()
	}
	after, ok := input.ResultingContent(toolName, string(current))
	if !ok {
		return true, rule.ID, "cannot determine the file's content after this change: old_string must match the current file exactly once"
	}
	if blocked, reason := rule.Check(string(current), after); blocked {
		return true, rule.ID, reason
	}
	return false, "", ""
}
