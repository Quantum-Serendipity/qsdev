package policy

import (
	"encoding/json"
	"fmt"
	"slices"

	"gopkg.in/yaml.v3"
)

type BypassTier int

const (
	EnforceAlways BypassTier = iota
	Session
	Command
)

var bypassTierNames = map[BypassTier]string{
	EnforceAlways: "enforce_always",
	Session:       "session",
	Command:       "command",
}

var bypassTierValues = map[string]BypassTier{
	"enforce_always": EnforceAlways,
	"session":        Session,
	"command":        Command,
}

func (b BypassTier) String() string {
	if name, ok := bypassTierNames[b]; ok {
		return name
	}
	return fmt.Sprintf("BypassTier(%d)", int(b))
}

func (b *BypassTier) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return fmt.Errorf("decoding bypass tier: %w", err)
	}
	tier, ok := bypassTierValues[s]
	if !ok {
		return fmt.Errorf("parsing bypass tier %q: unknown value", s)
	}
	*b = tier
	return nil
}

type Severity int

const (
	Critical Severity = iota
	High
	Medium
	Low
)

var severityNames = map[Severity]string{
	Critical: "critical",
	High:     "high",
	Medium:   "medium",
	Low:      "low",
}

var severityValues = map[string]Severity{
	"critical": Critical,
	"high":     High,
	"medium":   Medium,
	"low":      Low,
}

func (s Severity) String() string {
	if name, ok := severityNames[s]; ok {
		return name
	}
	return fmt.Sprintf("Severity(%d)", int(s))
}

func (s *Severity) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	if err := value.Decode(&raw); err != nil {
		return fmt.Errorf("decoding severity: %w", err)
	}
	sev, ok := severityValues[raw]
	if !ok {
		return fmt.Errorf("parsing severity %q: unknown value", raw)
	}
	*s = sev
	return nil
}

type ActionType string

const (
	Block  ActionType = "block"
	Warn   ActionType = "warn"
	Audit  ActionType = "audit"
	Prompt ActionType = "prompt"
)

type ConditionType string

const (
	ToolMatch       ConditionType = "tool_match"
	PathGlob        ConditionType = "path_glob"
	RegexMatch      ConditionType = "regex_match"
	CommandMatch    ConditionType = "command_match"
	FileExistence   ConditionType = "file_existence"
	FileType        ConditionType = "file_type"
	DeniedPathCheck ConditionType = "denied_path_check"
	Semantic        ConditionType = "semantic"
	All             ConditionType = "all"
	Any             ConditionType = "any"
	Not             ConditionType = "not"
)

type FailMode string

const (
	FailClosed FailMode = "fail_closed"
	FailOpen   FailMode = "fail_open"
)

type SecurityPolicy struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   PolicyMetadata `yaml:"metadata"`
	Settings   PolicySettings `yaml:"settings,omitempty"`
	Rules      []PolicyRule   `yaml:"rules"`
}

type PolicyMetadata struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Version     string            `yaml:"version"`
	Labels      map[string]string `yaml:"labels,omitempty"`
}

// PolicySettings holds policy-wide settings. Only fields the engine acts on
// belong here: the loader decodes strictly, so a key without a field is
// rejected instead of being accepted and silently ignored.
type PolicySettings struct {
	FailMode FailMode `yaml:"fail_mode,omitempty"`
}

type PolicyRule struct {
	ID          string     `yaml:"id"`
	Category    string     `yaml:"category"`
	Name        string     `yaml:"name"`
	Description string     `yaml:"description,omitempty"`
	Severity    Severity   `yaml:"severity"`
	BypassTier  BypassTier `yaml:"bypass_tier"`
	MonitorMode bool       `yaml:"monitor_mode,omitempty"`
	Enabled     *bool      `yaml:"enabled,omitempty"`
	Conditions  Condition  `yaml:"conditions"`
	Action      Action     `yaml:"action"`
}

func (r PolicyRule) IsEnabled() bool {
	return r.Enabled == nil || *r.Enabled
}

type Condition struct {
	Type ConditionType `yaml:"type"`

	ToolName string `yaml:"tool_name,omitempty"`
	Pattern  string `yaml:"pattern,omitempty"`
	Path     string `yaml:"path,omitempty"`
	FileType string `yaml:"file_type,omitempty"`
	Prompt   string `yaml:"prompt,omitempty"`

	Conditions []Condition `yaml:"conditions,omitempty"`
	Condition  *Condition  `yaml:"condition,omitempty"`
}

// Action is what a matching rule does. As with PolicySettings, only fields the
// evaluator reads are declared, so the strict loader rejects any other key.
type Action struct {
	Type             ActionType `yaml:"type"`
	Message          string     `yaml:"message,omitempty"`
	DefaultOnTimeout string     `yaml:"default_on_timeout,omitempty"`
}

type TierFilter int

const (
	AllTiers TierFilter = iota
	EnforceAlwaysOnly
	SessionCommandOnly
)

type EvalContext struct {
	ToolName  string
	ToolInput json.RawMessage
	FilePath  string
	Command   string
	CWD       string
	// ProjectRoot and SessionID scope bypass grants: only grants issued for
	// this project and this Claude Code session apply.
	ProjectRoot string
	SessionID   string
	// Overrides are the bypasses in force for the call. The policy engine
	// resolves them from its session state for the context's scope.
	Overrides  ActiveOverrides
	TierFilter TierFilter
}

// BypassScope returns the scope bypass grants must match for this call.
func (c *EvalContext) BypassScope() BypassScope {
	return BypassScope{ProjectRoot: c.ProjectRoot, SessionID: c.SessionID}
}

type PolicyDecision struct {
	Action   ActionType
	ExitCode int
	RuleID   string
	Message  string
	Err      error
	Findings []Finding
	// BypassTier is the tier of the rule that produced a Block, so the caller
	// can tell whether a human may lift it.
	BypassTier BypassTier
	// ConsumedTokens lists the command-tier rules whose one-shot bypass token
	// this non-blocking decision relied on. The caller must consume them
	// before letting the call run, and block it when that fails.
	ConsumedTokens []string
}

type Finding struct {
	RuleID   string
	Category string
	Severity Severity
	Message  string
	Monitor  bool
}

// DenyRule is a path pattern projected from a blocking policy rule for the
// MCP confused-deputy check. RuleID and BypassTier identify the originating
// rule so session overrides apply to the projection as they do to the rule.
type DenyRule struct {
	Pattern    string
	Type       string
	RuleID     string
	BypassTier BypassTier
}

// SessionBypassed reports whether an active session-tier grant lifts this
// deny rule, as it lifts the rule in Evaluate.
func (d DenyRule) SessionBypassed(o ActiveOverrides) bool {
	return d.BypassTier == Session && slices.Contains(o.Session, d.RuleID)
}

// CommandTokenHeld reports whether an unconsumed command-tier token lifts this
// deny rule for one call; a call that relies on it must consume the token.
func (d DenyRule) CommandTokenHeld(o ActiveOverrides) bool {
	return d.BypassTier == Command && slices.Contains(o.Command, d.RuleID)
}
