package aiframework

import (
	"context"
	"fmt"
	"os"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ToolAdapter translates framework-agnostic security policies into
// framework-specific enforcement artifacts.
type ToolAdapter interface {
	FrameworkID() FrameworkID
	EnforcementTier() EnforcementTier
	TranslatePermissions(ctx context.Context, policy *PermissionPolicy) (*PermissionArtifacts, error)
	TranslateIgnorePatterns(ctx context.Context, patterns []IgnorePattern) ([]types.GeneratedFile, error)
	InjectCredentials(ctx context.Context, scope *CredentialScope) (*CredentialArtifacts, error)
	ReportGaps(ctx context.Context, policy *PermissionPolicy) []EnforcementGap
}

// EnforcementTier ranks how strongly a framework can enforce security policies.
type EnforcementTier int

const (
	TierKernel   EnforcementTier = iota // Sandbox physically prevents the action.
	TierHook                            // Pre-execution check can deny.
	TierPolicy                          // Agent told what is allowed via config.
	TierAdvisory                        // Instructions say do not.
	TierExternal                        // qsdev wraps with external isolation.
)

var enforcementTierNames = [...]string{
	TierKernel:   "kernel",
	TierHook:     "hook",
	TierPolicy:   "policy",
	TierAdvisory: "advisory",
	TierExternal: "external",
}

func (t EnforcementTier) String() string {
	if int(t) >= 0 && int(t) < len(enforcementTierNames) {
		return enforcementTierNames[t]
	}
	return "unknown"
}

func (t EnforcementTier) MarshalText() ([]byte, error) {
	s := t.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown EnforcementTier value %d", int(t))
	}
	return []byte(s), nil
}

func (t *EnforcementTier) UnmarshalText(text []byte) error {
	for i, name := range enforcementTierNames {
		if name == string(text) {
			*t = EnforcementTier(i)
			return nil
		}
	}
	return fmt.Errorf("unknown enforcement tier: %q", string(text))
}

// Strength returns a numeric value for tier comparison.
// Higher values indicate stronger enforcement.
func (t EnforcementTier) Strength() int {
	strengths := [...]int{5, 4, 3, 2, 1}
	if int(t) >= 0 && int(t) < len(strengths) {
		return strengths[t]
	}
	return 0
}

type PermissionArtifacts struct {
	GeneratedFiles []types.GeneratedFile
	WrapperScripts []WrapperScript
	Instructions   []InstructionBlock
	ActiveTier     EnforcementTier
}

type WrapperScript struct {
	Path    string
	Content []byte
	Mode    os.FileMode
}

type InstructionBlock struct {
	Title    string
	Content  string
	Priority int
}

// IgnoreCategory classifies the reason for an exclusion pattern.
type IgnoreCategory int

const (
	CategoryCredential IgnoreCategory = iota
	CategoryBinary
	CategoryVendor
	CategoryInfrastructure
	CategoryQsdevInternal
)

var ignoreCategoryNames = [...]string{
	CategoryCredential:     "credential",
	CategoryBinary:         "binary",
	CategoryVendor:         "vendor",
	CategoryInfrastructure: "infrastructure",
	CategoryQsdevInternal:  "qsdev_internal",
}

func (c IgnoreCategory) String() string {
	if int(c) >= 0 && int(c) < len(ignoreCategoryNames) {
		return ignoreCategoryNames[c]
	}
	return "unknown"
}

func (c IgnoreCategory) MarshalText() ([]byte, error) {
	s := c.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown IgnoreCategory value %d", int(c))
	}
	return []byte(s), nil
}

func (c *IgnoreCategory) UnmarshalText(text []byte) error {
	for i, name := range ignoreCategoryNames {
		if name == string(text) {
			*c = IgnoreCategory(i)
			return nil
		}
	}
	return fmt.Errorf("unknown ignore category: %q", string(text))
}

type IgnorePattern struct {
	Pattern  string
	Reason   string
	Category IgnoreCategory
}

type APIKeyRequirement struct {
	Provider string
	EnvVar   string
	Required bool
}

type CredentialScope struct {
	AWSProfile        string
	GCPProject        string
	AzureSubscription string
	APIKeys           []APIKeyRequirement
	SandboxFilters    []string
}

// DefaultSandboxFilters returns the default credential environment variable
// patterns to strip from sandbox environments.
func DefaultSandboxFilters() []string {
	return []string{
		"*_SECRET*", "*_TOKEN*", "*_KEY*", "*_PASSWORD*",
		"AWS_*", "GITHUB_TOKEN", "NPM_TOKEN",
	}
}

type CredentialArtifacts struct {
	EnvVars        map[string]string
	ExcludePaths   []string
	MCPServer      *MCPServerSpec
	GeneratedFiles []types.GeneratedFile
}

type EnforcementGap struct {
	Rule         PermissionRule
	RequiredTier EnforcementTier
	ActualTier   EnforcementTier
	Description  string
	Mitigation   string
}
