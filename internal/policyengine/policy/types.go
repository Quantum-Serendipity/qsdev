package policy

import (
	"encoding/json"
	"fmt"

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

type PolicySettings struct {
	FailMode            FailMode `yaml:"fail_mode,omitempty"`
	EvaluationTimeoutMS int      `yaml:"evaluation_timeout_ms,omitempty"`
	LogFormat           string   `yaml:"log_format,omitempty"`
	InheritFrom         []string `yaml:"inherit_from,omitempty"`
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

type Action struct {
	Type             ActionType `yaml:"type"`
	ExitCode         int        `yaml:"exit_code,omitempty"`
	Message          string     `yaml:"message,omitempty"`
	Stderr           string     `yaml:"stderr,omitempty"`
	TimeoutSeconds   int        `yaml:"timeout_seconds,omitempty"`
	DefaultOnTimeout string     `yaml:"default_on_timeout,omitempty"`
}

type TierFilter int

const (
	AllTiers TierFilter = iota
	EnforceAlwaysOnly
	SessionCommandOnly
)

type EvalContext struct {
	ToolName         string
	ToolInput        json.RawMessage
	FilePath         string
	Command          string
	CWD              string
	SessionOverrides []string
	TierFilter       TierFilter
}

type PolicyDecision struct {
	Action   ActionType
	ExitCode int
	RuleID   string
	Message  string
	Err      error
	Findings []Finding
}

type Finding struct {
	RuleID   string
	Category string
	Severity Severity
	Message  string
	Monitor  bool
}

type DenyRule struct {
	Pattern string
	Type    string
}
