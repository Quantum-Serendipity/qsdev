package aiframework

import (
	"context"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// HookDeployer installs lifecycle hooks in a framework's native format.
type HookDeployer interface {
	FrameworkID() FrameworkID
	SupportedEvents() []HookEvent
	Protocol() HookProtocol
	Deploy(ctx context.Context, hooks []HookPolicy) ([]types.GeneratedFile, error)
	Undeploy(ctx context.Context, projectRoot string) error
}

// HookEvent identifies a lifecycle event that can trigger hooks.
type HookEvent int

const (
	EventPreToolUse HookEvent = iota
	EventPostToolUse
	EventSessionStart
	EventSessionEnd
)

var hookEventNames = [...]string{
	EventPreToolUse:   "pre_tool_use",
	EventPostToolUse:  "post_tool_use",
	EventSessionStart: "session_start",
	EventSessionEnd:   "session_end",
}

func (e HookEvent) String() string {
	if int(e) >= 0 && int(e) < len(hookEventNames) {
		return hookEventNames[e]
	}
	return "unknown"
}

func (e HookEvent) MarshalText() ([]byte, error) {
	s := e.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown HookEvent value %d", int(e))
	}
	return []byte(s), nil
}

func (e *HookEvent) UnmarshalText(text []byte) error {
	for i, name := range hookEventNames {
		if name == string(text) {
			*e = HookEvent(i)
			return nil
		}
	}
	return fmt.Errorf("unknown hook event: %q", string(text))
}

// HookInputFormat describes how hook input is provided.
type HookInputFormat int

const (
	InputJSONStdin HookInputFormat = iota
	InputStarlarkEval
	InputShellExec
	InputActionsYAML
)

var hookInputFormatNames = [...]string{
	InputJSONStdin:    "json_stdin",
	InputStarlarkEval: "starlark_eval",
	InputShellExec:    "shell_exec",
	InputActionsYAML:  "actions_yaml",
}

func (f HookInputFormat) String() string {
	if int(f) >= 0 && int(f) < len(hookInputFormatNames) {
		return hookInputFormatNames[f]
	}
	return "unknown"
}

func (f HookInputFormat) MarshalText() ([]byte, error) {
	s := f.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown HookInputFormat value %d", int(f))
	}
	return []byte(s), nil
}

func (f *HookInputFormat) UnmarshalText(text []byte) error {
	for i, name := range hookInputFormatNames {
		if name == string(text) {
			*f = HookInputFormat(i)
			return nil
		}
	}
	return fmt.Errorf("unknown hook input format: %q", string(text))
}

// HookResponseFormat describes how a hook communicates its result.
type HookResponseFormat int

const (
	ResponseExitCode HookResponseFormat = iota
	ResponseStdoutJSON
	ResponseStarlarkReturn
)

var hookResponseFormatNames = [...]string{
	ResponseExitCode:       "exit_code",
	ResponseStdoutJSON:     "stdout_json",
	ResponseStarlarkReturn: "starlark_return",
}

func (f HookResponseFormat) String() string {
	if int(f) >= 0 && int(f) < len(hookResponseFormatNames) {
		return hookResponseFormatNames[f]
	}
	return "unknown"
}

func (f HookResponseFormat) MarshalText() ([]byte, error) {
	s := f.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown HookResponseFormat value %d", int(f))
	}
	return []byte(s), nil
}

func (f *HookResponseFormat) UnmarshalText(text []byte) error {
	for i, name := range hookResponseFormatNames {
		if name == string(text) {
			*f = HookResponseFormat(i)
			return nil
		}
	}
	return fmt.Errorf("unknown hook response format: %q", string(text))
}

// HookEnforcementMode describes how strictly a hook's result is enforced.
type HookEnforcementMode int

const (
	EnforcementHardDeny HookEnforcementMode = iota
	EnforcementAdvisory
	EnforcementAuditOnly
)

var hookEnforcementModeNames = [...]string{
	EnforcementHardDeny:  "hard_deny",
	EnforcementAdvisory:  "advisory",
	EnforcementAuditOnly: "audit_only",
}

func (m HookEnforcementMode) String() string {
	if int(m) >= 0 && int(m) < len(hookEnforcementModeNames) {
		return hookEnforcementModeNames[m]
	}
	return "unknown"
}

func (m HookEnforcementMode) MarshalText() ([]byte, error) {
	s := m.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown HookEnforcementMode value %d", int(m))
	}
	return []byte(s), nil
}

func (m *HookEnforcementMode) UnmarshalText(text []byte) error {
	for i, name := range hookEnforcementModeNames {
		if name == string(text) {
			*m = HookEnforcementMode(i)
			return nil
		}
	}
	return fmt.Errorf("unknown hook enforcement mode: %q", string(text))
}

// HookProtocol describes the invocation protocol for a framework's hooks.
type HookProtocol struct {
	InputFormat     HookInputFormat
	ResponseFormat  HookResponseFormat
	EnforcementMode HookEnforcementMode
}

// HookLogicID identifies a reusable hook logic implementation.
type HookLogicID string

const (
	LogicPackageGuard        HookLogicID = "package-guard"
	LogicDestructiveBlock    HookLogicID = "destructive-command-block"
	LogicCredentialScan      HookLogicID = "credential-scan"
	LogicAgentSelfProtection HookLogicID = "agent-self-protection"
	LogicFileBoundary        HookLogicID = "file-boundary"
	LogicToolGates           HookLogicID = "tool-gates"
)

// HookPolicy declares an abstract hook to be deployed.
type HookPolicy struct {
	Event        HookEvent
	ToolMatchers []string
	Logic        HookLogicID
	Timeout      int
	FailOpen     bool
}
