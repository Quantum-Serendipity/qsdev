package aiframework

// PermissionPolicy declares framework-agnostic permission rules.
type PermissionPolicy struct {
	Preset       string
	AllowRules   []PermissionRule
	DenyRules    []PermissionRule
	AskRules     []PermissionRule
	ApprovalMode string
}

// PermissionRule is a single allow/deny/ask pattern with an optional reason.
type PermissionRule struct {
	Pattern string
	Reason  string
}

// SandboxPolicy declares filesystem and network sandboxing constraints.
type SandboxPolicy struct {
	WritablePaths  []string
	ReadOnlyPaths  []string
	DeniedPaths    []string
	NetworkAllowed []string
	NetworkDenied  []string
}

// ModelPreferences declares model selection preferences.
type ModelPreferences struct {
	PreferredModel string
	FallbackModel  string
	MaxTokens      int
}

// HookConfiguration declares lifecycle hooks to be deployed.
type HookConfiguration struct {
	Hooks []HookSpec
}

// HookSpec describes a single hook to deploy.
type HookSpec struct {
	Event    string
	Command  string
	Matchers []string
	Timeout  int
	FailOpen bool
}
