package aiframework

import (
	"context"

	"github.com/Quantum-Serendipity/qsdev/internal/enumtext"
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

var hookEventText = enumtext.New[HookEvent]("HookEvent", "hook event", "unknown", hookEventNames[:])

func (e HookEvent) String() string { return hookEventText.String(e) }

func (e HookEvent) MarshalText() ([]byte, error) { return hookEventText.MarshalText(e) }

func (e *HookEvent) UnmarshalText(text []byte) error { return hookEventText.UnmarshalText(text, e) }

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

var hookInputFormatText = enumtext.New[HookInputFormat]("HookInputFormat", "hook input format", "unknown", hookInputFormatNames[:])

func (f HookInputFormat) String() string { return hookInputFormatText.String(f) }

func (f HookInputFormat) MarshalText() ([]byte, error) { return hookInputFormatText.MarshalText(f) }

func (f *HookInputFormat) UnmarshalText(text []byte) error {
	return hookInputFormatText.UnmarshalText(text, f)
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

var hookResponseFormatText = enumtext.New[HookResponseFormat]("HookResponseFormat", "hook response format", "unknown", hookResponseFormatNames[:])

func (f HookResponseFormat) String() string { return hookResponseFormatText.String(f) }

func (f HookResponseFormat) MarshalText() ([]byte, error) {
	return hookResponseFormatText.MarshalText(f)
}

func (f *HookResponseFormat) UnmarshalText(text []byte) error {
	return hookResponseFormatText.UnmarshalText(text, f)
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

var hookEnforcementModeText = enumtext.New[HookEnforcementMode]("HookEnforcementMode", "hook enforcement mode", "unknown", hookEnforcementModeNames[:])

func (m HookEnforcementMode) String() string { return hookEnforcementModeText.String(m) }

func (m HookEnforcementMode) MarshalText() ([]byte, error) {
	return hookEnforcementModeText.MarshalText(m)
}

func (m *HookEnforcementMode) UnmarshalText(text []byte) error {
	return hookEnforcementModeText.UnmarshalText(text, m)
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
