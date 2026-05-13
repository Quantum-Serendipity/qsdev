package cigeneration

// CIPlatform identifies the CI system to generate workflows for.
type CIPlatform string

const (
	PlatformGitHubActions CIPlatform = "github"
	PlatformGitLabCI      CIPlatform = "gitlab"
	PlatformNone          CIPlatform = "none"
)

// CIJobID identifies a logical CI job within the generated workflow.
type CIJobID string

const (
	JobLintSAST          CIJobID = "lint-sast"
	JobSecretScan        CIJobID = "secret-scan"
	JobVulnerabilityScan CIJobID = "vulnerability-scan"
	JobContainerSecurity CIJobID = "container-security"
	JobSecurityReview    CIJobID = "security-review"
	JobLicenseCompliance CIJobID = "license-compliance"
)

// CIStep represents a single step within a CI job.
type CIStep struct {
	ToolName        string
	JobID           CIJobID
	Name            string
	Uses            string            // GitHub Action ref (owner/repo@sha)
	UsesComment     string            // human-readable tag comment
	With            map[string]string // action inputs
	Run             string            // shell command (alternative to Uses)
	Env             map[string]string
	Order           int  // lower runs first
	Condition       string
	ContinueOnError bool
}

// CIPermission represents a GitHub Actions workflow-level permission.
type CIPermission struct {
	Scope string // "contents", "security-events", "id-token", "pull-requests"
	Level string // "read", "write"
}

// GenerateConfig carries the user's selections into the generation pipeline.
type GenerateConfig struct {
	Platform      CIPlatform
	EnabledTools  []string
	HasDocker     bool
	HasClaudeCode bool
}
