package devinit

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/gatedodge"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/judge"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
)

func selfprotectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "selfprotect",
		Short:  "Evaluate self-protection rules for a tool call (invoked by hooks)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelfprotect(cmd)
		},
	}
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmdutil.MarkProfile(cmd, cmdutil.ProfileAutomatedHook)
}

// errSelfprotectDeny is returned after the deny reason has been written to
// stderr. It carries exit code 2, the only code Claude Code treats as a block,
// and an empty message so nothing is appended to the hook's stderr payload.
var errSelfprotectDeny = &ExitError{Code: 2}

// runSelfprotect evaluates the self-protection rules for the hook payload on
// stdin within hookio.EvalDeadline.
func runSelfprotect(cmd *cobra.Command) error {
	return runSelfprotectWith(cmd, hookio.EvalDeadline, selfprotectEvaluatorFor(cmdutil.SensitiveCommands(cmd.Root())))
}

// selfprotectEvaluatorFor returns the production selfprotectEvaluator, with
// SP-014 blocking the sensitive commands described by sensitive.
func selfprotectEvaluatorFor(sensitive []cmdscan.CommandSpec) selfprotectEvaluator {
	return func(ctx context.Context, stdin io.Reader, stderr io.Writer) error {
		return evaluateSelfprotect(ctx, stdin, stderr, sensitive)
	}
}

// selfprotectEvaluator reads a hook payload from stdin and evaluates it,
// writing any deny reason to w and returning errSelfprotectDeny, or nil to
// allow the call.
type selfprotectEvaluator func(ctx context.Context, stdin io.Reader, w io.Writer) error

// runSelfprotectWith runs evaluate under one deadline covering both the stdin
// read and the evaluation. Every deny path, including malformed input, a
// panic and an overrun deadline, writes its reason to stderr and returns
// errSelfprotectDeny; an allowed call returns nil.
func runSelfprotectWith(cmd *cobra.Command, deadline time.Duration, evaluate selfprotectEvaluator) error {
	stderr := cmd.ErrOrStderr()
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, deadline)
	defer cancel()

	stdin := cmd.InOrStdin()
	timedOut, err := hookio.RunWithDeadline(ctx, func(ctx context.Context, w io.Writer) error {
		return evaluate(ctx, stdin, w)
	}, stderr)
	switch {
	case timedOut, errors.Is(err, errSelfprotectDeny):
		return errSelfprotectDeny
	case err != nil:
		hookio.WriteError(stderr, err.Error())
		return errSelfprotectDeny
	}
	return nil
}

// evaluateSelfprotect evaluates the hook payload on stdin, with SP-014
// blocking the sensitive commands described by sensitive.
func evaluateSelfprotect(ctx context.Context, stdin io.Reader, stderr io.Writer, sensitive []cmdscan.CommandSpec) error {
	call, err := hookio.ParseToolCall(ctx, stdin)
	if err != nil {
		hookio.WriteError(stderr, err.Error())
		return errSelfprotectDeny
	}

	input, err := hookio.ParseInput(call.ToolName, call.ToolInput)
	if err != nil {
		hookio.WriteError(stderr, err.Error())
		return errSelfprotectDeny
	}
	evalCtx := buildSelfprotectContext(call, &input)
	evalCtx.ToolInput = call.ToolInput
	evalCtx.SensitiveCommands = sensitive

	if d, denied := judge.Evaluate(evalCtx); denied {
		return writeDenial(stderr, d)
	}
	if cmdscan.IsNixRunTool(call.ToolName) {
		if err := judgeNixRun(call, evalCtx.CWD, sensitive, stderr); err != nil {
			return err
		}
	}

	if edited := input.EditedContent(); isWriteOrEditTool(call.ToolName) && edited != "" {
		if blocked, ruleID, reason := gatedodge.Detect(input.FilePath, edited); blocked {
			hookio.WriteDeny(stderr, ruleID, reason)
			return errSelfprotectDeny
		}
	}

	if isWriteOrEditTool(call.ToolName) {
		if blocked, ruleID, reason := gatedodge.DetectChange(input.FilePath, evalCtx.FileChange); blocked {
			hookio.WriteDeny(stderr, ruleID, reason)
			return errSelfprotectDeny
		}
	}
	if blocked, ruleID, reason := detectResultGateDodge(call.ToolName, input, evalCtx.CanonicalPath); blocked {
		hookio.WriteDeny(stderr, ruleID, reason)
		return errSelfprotectDeny
	}
	return nil
}

// writeDenial writes d to stderr in the hook's deny format and returns
// errSelfprotectDeny.
func writeDenial(stderr io.Writer, d judge.Denial) error {
	if d.Evasion {
		hookio.WriteEvasionDeny(stderr, d.RuleID, d.Reason)
	} else {
		hookio.WriteDeny(stderr, d.RuleID, d.Reason)
	}
	return errSelfprotectDeny
}

// judgeNixRun judges a call of the MCP server's nix_run tool as the Bash
// command lines it is equivalent to (cmdscan.NixRunCommandLines), each run in
// cwd, so the tool cannot do what a Bash call may not. The server holds each
// call to the same checks before running it; judging it here as well keeps
// the decision in the hook Claude Code consults for every tool. A tool_input
// that cannot be read is denied.
func judgeNixRun(call *hookio.ToolCall, cwd string, sensitive []cmdscan.CommandSpec, stderr io.Writer) error {
	in, err := hookio.ParseNixRunInput(call.ToolInput)
	if err != nil {
		hookio.WriteError(stderr, err.Error())
		return errSelfprotectDeny
	}
	for _, line := range cmdscan.NixRunCommandLines(in.Installable, in.Args, in.Stdin) {
		ctx := &rules.EvalContext{ToolName: "Bash", Command: line, CWD: cwd, SensitiveCommands: sensitive}
		if d, denied := judge.Evaluate(ctx); denied {
			return writeDenial(stderr, d)
		}
	}
	return nil
}

// buildSelfprotectContext builds the rule context for call. CWD is the
// envelope's cwd: the session directory, which the Bash tool keeps across
// calls (a `cd` in one call moves the next), so relative paths are resolved
// against it, not against the hook process's own directory. Only when the
// envelope has none does it fall back to the process working directory.
// FilePath keeps the spelling the tool used, which rules match on;
// CanonicalPath is the resolved target.
func buildSelfprotectContext(call *hookio.ToolCall, input *hookio.ToolInput) *rules.EvalContext {
	ctx := &rules.EvalContext{
		ToolName: call.ToolName,
		FilePath: input.FilePath,
		Command:  input.Command,
		Content:  input.EditedContent(),
		Edits:    textEdits(call.ToolName, input),
		CWD:      call.CWD,
	}

	if ctx.CWD == "" {
		if cwd, err := projectctx.WorkingDir(); err == nil {
			ctx.CWD = cwd
		}
	}

	if input.FilePath != "" {
		ctx.CanonicalPath = targetPath(input.FilePath, ctx.CWD)
	}

	return ctx
}

// targetPath returns the canonical form of filePath for the path-based rules,
// resolving a relative path against cwd (or the process working directory
// when cwd is empty). A Windows drive-relative (C:x) or rooted (\x) path is
// not joined to cwd: it is left to canonicalization, as before.
//
// When canonicalization fails (a permission error on an ancestor, a symlink
// loop, an unresolvable home directory) it falls back to the lexically
// cleaned absolute path instead of leaving CanonicalPath empty: an empty path
// is never protected, so dropping the error would let every path rule allow
// the call. With the fallback a protected target is still denied, and an
// unresolvable home makes canon.IsProtected fail closed.
func targetPath(filePath, cwd string) string {
	p, err := canon.ExpandTilde(filePath)
	if err != nil {
		p = filePath
	}
	if cwd != "" && !canon.IsRooted(p) && filepath.VolumeName(p) == "" {
		// Not filepath.Join: cleaning would collapse lnk/.. before the
		// symlink lnk is resolved, naming a different file than the tool's.
		if !os.IsPathSeparator(cwd[len(cwd)-1]) {
			cwd += string(filepath.Separator)
		}
		p = cwd + p
	}
	if canonical, err := canon.Canonicalize(p); err == nil {
		return canonical
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return filepath.Clean(p)
}

// textEdits returns the replacements of an Edit/MultiEdit call, which rules
// apply to the current file to see what the call leaves behind.
func textEdits(toolName string, input *hookio.ToolInput) []rules.TextEdit {
	switch toolName {
	case "Edit":
		return []rules.TextEdit{{OldString: input.OldString, NewString: input.NewString, ReplaceAll: input.ReplaceAll}}
	case "MultiEdit":
		edits := make([]rules.TextEdit, 0, len(input.Edits))
		for _, e := range input.Edits {
			edits = append(edits, rules.TextEdit{OldString: e.OldString, NewString: e.NewString, ReplaceAll: e.ReplaceAll})
		}
		return edits
	default:
		return nil
	}
}

func isWriteOrEditTool(toolName string) bool {
	return toolName == "Write" || toolName == "Edit" || toolName == "MultiEdit"
}
