package devinit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/evasion"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/gatedodge"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/hookio"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/rules"
)

const selfprotectTimeout = 5 * time.Second

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
	return cmd
}

// errSelfprotectDeny is returned after the deny reason has been written to
// stderr. It carries exit code 2, the only code Claude Code treats as a block,
// and an empty message so nothing is appended to the hook's stderr payload.
var errSelfprotectDeny = &ExitError{Code: 2}

// runSelfprotect evaluates the self-protection rules for the hook payload on
// stdin. Every deny path, including malformed input and a panic, writes its
// reason to stderr and returns errSelfprotectDeny; an allowed call returns nil.
func runSelfprotect(cmd *cobra.Command) (err error) {
	stderr := cmd.ErrOrStderr()
	defer func() {
		if r := recover(); r != nil {
			hookio.WriteError(stderr, fmt.Sprintf("%v", r))
			err = errSelfprotectDeny
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), selfprotectTimeout)
	defer cancel()

	call, err := hookio.ParseToolCall(ctx, cmd.InOrStdin())
	if err != nil {
		hookio.WriteError(stderr, err.Error())
		return errSelfprotectDeny
	}

	input := hookio.ParseInput(call.ToolInput)
	evalCtx := buildSelfprotectContext(call.ToolName, &input)

	// Parse the Bash command once here (memoized on evalCtx); the rules below
	// reuse the same parse via ctx.ParsedCommands().
	cmds, parseErr := evalCtx.ParsedCommands()
	if blocked, category, reason := evasion.CheckParsed(call.ToolName, input.Command, input.FilePath, cmds, parseErr); blocked {
		hookio.WriteEvasionDeny(stderr, category, reason)
		return errSelfprotectDeny
	}

	verdict, matches := rules.Tier1Rules.EvaluateAll(evalCtx)
	if verdict == rules.Deny {
		hookio.WriteDeny(stderr, matches[0].Rule.ID, matches[0].Reason)
		return errSelfprotectDeny
	}

	if edited := input.EditedContent(); isWriteOrEditTool(call.ToolName) && edited != "" {
		if blocked, ruleID, reason := gatedodge.Detect(input.FilePath, edited); blocked {
			hookio.WriteDeny(stderr, ruleID, reason)
			return errSelfprotectDeny
		}
	}
	return nil
}

func buildSelfprotectContext(toolName string, input *hookio.ToolInput) *rules.EvalContext {
	ctx := &rules.EvalContext{
		ToolName: toolName,
		FilePath: input.FilePath,
		Command:  input.Command,
		Content:  input.EditedContent(),
	}

	if cwd, err := os.Getwd(); err == nil {
		ctx.CWD = cwd
	}

	if input.FilePath != "" {
		ctx.CanonicalPath = targetPath(input.FilePath)
	}

	return ctx
}

// targetPath returns the canonical form of filePath for the path-based rules.
// When canonicalization fails (a permission error on an ancestor, a symlink
// loop, an unresolvable home directory) it falls back to the lexically
// cleaned absolute path instead of leaving CanonicalPath empty: an empty path
// is never protected, so dropping the error would let every path rule allow
// the call. With the fallback a protected target is still denied, and an
// unresolvable home makes canon.IsProtected fail closed.
func targetPath(filePath string) string {
	if canonical, err := canon.Canonicalize(filePath); err == nil {
		return canonical
	}
	expanded, err := canon.ExpandTilde(filePath)
	if err != nil {
		expanded = filePath
	}
	if abs, err := filepath.Abs(expanded); err == nil {
		return abs
	}
	return filepath.Clean(expanded)
}

func isWriteOrEditTool(toolName string) bool {
	return toolName == "Write" || toolName == "Edit" || toolName == "MultiEdit"
}
