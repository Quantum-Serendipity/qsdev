package devinit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/exitcode"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/risk"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/trust"
)

type hookInput struct {
	ToolName   string          `json:"tool_name"`
	ToolInput  json.RawMessage `json:"tool_input"`
	ToolOutput string          `json:"tool_output,omitempty"`
}

func enforceCmd() *cobra.Command {
	var hookEvent string

	cmd := &cobra.Command{
		Use:    "enforce",
		Short:  "Evaluate security policy for a tool call (invoked by hooks)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnforce(cmd, hookEvent)
		},
	}
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.Flags().StringVar(&hookEvent, "hook", "", "Hook event type (PreToolUse or PostToolUse)")
	_ = cmd.MarkFlagRequired("hook")
	return cmd
}

func runEnforce(cmd *cobra.Command, hookEvent string) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var input hookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("parsing hook input: %w", err)
	}

	evalCtx := buildEvalContext(&input)

	policyFiles := discoverPolicyFiles()
	if len(policyFiles) == 0 {
		return nil
	}

	sessionPath, err := sessionStatePath()
	if err != nil {
		return nil
	}

	stateReader := policy.NewFileSessionStateReader(sessionPath)
	engine, err := policy.NewPolicyEngine(policyFiles, stateReader, policy.EngineOptions{})
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not load policy engine: %v\n", err)
		return nil
	}

	orchestrator := newProductionOrchestrator(engine)

	switch hookEvent {
	case "PreToolUse":
		code := orchestrator.RunPreToolUse(evalCtx)
		if code != 0 {
			return exitcode.New(code, "policy enforcement blocked tool call (exit code %d)", code)
		}
	case "PostToolUse":
		output, _ := orchestrator.RunPostToolUse(evalCtx, input.ToolOutput)
		if output != input.ToolOutput {
			fmt.Fprint(cmd.OutOrStdout(), output)
		}
	}

	return nil
}

func buildEvalContext(input *hookInput) *policy.EvalContext {
	ctx := &policy.EvalContext{
		ToolName:  input.ToolName,
		ToolInput: input.ToolInput,
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input.ToolInput, &fields); err == nil {
		ctx.FilePath = extractStringField(fields, "file_path", "path")
		ctx.Command = extractStringField(fields, "command")
	}

	if cwd, err := os.Getwd(); err == nil {
		ctx.CWD = cwd
	}

	return ctx
}

func extractStringField(fields map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		raw, ok := fields[key]
		if !ok {
			continue
		}
		var val string
		if err := json.Unmarshal(raw, &val); err == nil && val != "" {
			return val
		}
	}
	return ""
}

func discoverPolicyFiles() []string {
	var files []string

	projectFile := filepath.Join(".qsdev", "policy.yaml")
	if _, err := os.Stat(projectFile); err == nil {
		files = append(files, projectFile)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		userFile := filepath.Join(home, ".qsdev", "policy.yaml")
		if _, err := os.Stat(userFile); err == nil {
			files = append(files, userFile)
		}
	}

	return files
}

func sessionStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	return filepath.Join(home, ".qsdev", "session-state.json"), nil
}

// trustConfigPath returns the path to the user's MCP trust configuration.
// NewMcpTrustEngine tolerates a missing file (it falls back to an empty config),
// so an empty path when the home directory cannot be resolved is acceptable.
func trustConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".qsdev", "trust.yaml")
}

// newProductionOrchestrator wires the SecurityOrchestrator with the real risk
// scorer and MCP trust adapter so the confused-deputy check, PostToolUse output
// hardening, and package risk scoring are actually reachable in the hook path.
// Both production call sites (enforce and policy check) must go through this
// seam; constructing the orchestrator with nil risk/trust silently disables
// every MCP-poisoning defense.
func newProductionOrchestrator(engine *policy.PolicyEngine) *policyengine.SecurityOrchestrator {
	trustEngine := trust.NewMcpTrustEngine(trustConfigPath())
	trustAdapter := policyengine.NewTrustAdapter(trustEngine)
	return policyengine.NewSecurityOrchestrator(engine, risk.NewScorer(), trustAdapter)
}
