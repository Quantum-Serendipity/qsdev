package devinit

import (
	"errors"
	"io/fs"
	"os"
	"path"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/gatedodge"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
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

// detectBashGateDodge blocks a Bash command that rewrites a file guarded by a
// gate-dodge result rule: the before/after check runs only for Write and Edit,
// so a shell append, delete or in-place edit, or a package manager's config
// command, would skip it. Reading the file stays allowed.
func detectBashGateDodge(ctx *rules.EvalContext) (bool, string, string) {
	name, ok := rules.BashRewritesFile(ctx, gatedodge.GuardedFileNames())
	if !ok {
		name = configCommandTarget(ctx)
	}
	if name == "" {
		return false, "", ""
	}
	return true, gatedodge.ResultRuleFor(name).ID,
		"shell command changes " + name + ", whose security settings are only verified for the Edit and Write tools; make the change with Edit or Write"
}

// configCommandTarget returns the guarded file a package-manager command on
// the line rewrites (see gatedodge.ConfigCommandTarget), or "". An
// unparseable line is left to BashRewritesFile, which fails closed when it
// names a guarded file.
func configCommandTarget(ctx *rules.EvalContext) string {
	cmds, err := ctx.ParsedCommands()
	if err != nil {
		return ""
	}
	for _, c := range cmds {
		if name := gatedodge.ConfigCommandTarget(path.Base(c.Name), c.Args); name != "" {
			return name
		}
	}
	return ""
}
