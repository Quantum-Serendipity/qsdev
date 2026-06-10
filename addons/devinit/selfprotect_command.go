package devinit

import (
	"context"
	"fmt"
	"os"
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
		Run: func(cmd *cobra.Command, args []string) {
			runSelfprotect(cmd)
		},
	}
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}

func runSelfprotect(cmd *cobra.Command) {
	defer func() {
		if r := recover(); r != nil {
			hookio.WriteError(cmd.ErrOrStderr(), fmt.Sprintf("%v", r))
			os.Exit(2)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), selfprotectTimeout)
	defer cancel()

	call, err := hookio.ParseToolCall(ctx, os.Stdin)
	if err != nil {
		hookio.WriteError(cmd.ErrOrStderr(), err.Error())
		os.Exit(2)
	}

	input := hookio.ParseInput(call.ToolInput)
	evalCtx := buildSelfprotectContext(call.ToolName, &input)

	if blocked, category, reason := evasion.Check(call.ToolName, input.Command, input.FilePath); blocked {
		hookio.WriteEvasionDeny(cmd.ErrOrStderr(), category, reason)
		os.Exit(2)
	}

	verdict, matches := rules.Tier1Rules.EvaluateAll(evalCtx)
	if verdict == rules.Deny {
		hookio.WriteDeny(cmd.ErrOrStderr(), matches[0].Rule.ID, matches[0].Reason)
		os.Exit(2)
	}

	if isWriteOrEditTool(call.ToolName) && input.Content != "" {
		if blocked, ruleID, reason := gatedodge.Detect(input.FilePath, input.Content); blocked {
			hookio.WriteDeny(cmd.ErrOrStderr(), ruleID, reason)
			os.Exit(2)
		}
	}
}

func buildSelfprotectContext(toolName string, input *hookio.ToolInput) *rules.EvalContext {
	ctx := &rules.EvalContext{
		ToolName: toolName,
		FilePath: input.FilePath,
		Command:  input.Command,
		Content:  input.Content,
	}

	if cwd, err := os.Getwd(); err == nil {
		ctx.CWD = cwd
	}

	if input.FilePath != "" {
		if canonical, err := canon.Canonicalize(input.FilePath); err == nil {
			ctx.CanonicalPath = canonical
		}
	}

	return ctx
}

func isWriteOrEditTool(toolName string) bool {
	return toolName == "Write" || toolName == "Edit" || toolName == "MultiEdit"
}
