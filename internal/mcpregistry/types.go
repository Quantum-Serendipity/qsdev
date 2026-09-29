package mcpregistry

import (
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

// McpCategory classifies MCP servers by their functional role.
type McpCategory string

const (
	CategoryDocumentation  McpCategory = "documentation"
	CategorySecurity       McpCategory = "security"
	CategoryIntegration    McpCategory = "integration"
	CategoryAgent          McpCategory = "agent"
	CategoryInfrastructure McpCategory = "infrastructure"
)

// DisplayName returns a human-friendly label for the category.
func (c McpCategory) DisplayName() string {
	switch c {
	case CategoryDocumentation:
		return "Documentation"
	case CategorySecurity:
		return "Security"
	case CategoryIntegration:
		return "Integration"
	case CategoryAgent:
		return "Agent"
	case CategoryInfrastructure:
		return "Infrastructure"
	default:
		return string(c)
	}
}

// McpTransport identifies the communication transport for an MCP server.
type McpTransport string

const (
	TransportStdio McpTransport = "stdio"
	TransportSSE   McpTransport = "sse"
	TransportHTTP  McpTransport = "http"
)

// DefinitionSource indicates how an MCP server definition was discovered.
type DefinitionSource string

const (
	SourceBuiltin  DefinitionSource = "builtin"
	SourceCatalog  DefinitionSource = "catalog"
	SourceConfig   DefinitionSource = "config"
	SourceDetected DefinitionSource = "detected"
)

// ComplianceLevel indicates the security assurance tier of an MCP server.
type ComplianceLevel int

const (
	ComplianceBasic ComplianceLevel = iota
	ComplianceStandard
	ComplianceSecure
	ComplianceVerified
	ComplianceAttested
)

// String returns a human-readable label for the compliance level.
func (cl ComplianceLevel) String() string {
	switch cl {
	case ComplianceBasic:
		return "basic"
	case ComplianceStandard:
		return "standard"
	case ComplianceSecure:
		return "secure"
	case ComplianceVerified:
		return "verified"
	case ComplianceAttested:
		return "attested"
	default:
		return "unknown"
	}
}

// McpInstallMethod describes how an MCP server binary is provisioned.
type McpInstallMethod int

const (
	InstallManual McpInstallMethod = iota
	InstallUvTool
	InstallNpmGlobal
	InstallNixPackage
)

// String returns a human-readable label for the install method.
func (m McpInstallMethod) String() string {
	switch m {
	case InstallManual:
		return "manual"
	case InstallUvTool:
		return "uv-tool"
	case InstallNpmGlobal:
		return "npm-global"
	case InstallNixPackage:
		return "nix-package"
	default:
		return "unknown"
	}
}

// McpCapabilities describes the protocol capabilities an MCP server advertises.
type McpCapabilities struct {
	Tools         bool
	ToolCount     int
	Resources     bool
	ResourceCount int
	Prompts       bool
	PromptCount   int
}

// McpServerDefinition is the complete metadata for a single MCP server
// known to the registry. Its compliance grade is not stored: GradeServer
// computes it from the definition, so every consumer sees the same grade.
type McpServerDefinition struct {
	Name            string
	DisplayName     string
	Category        McpCategory
	Description     string
	Command         string
	Args            []string
	URL             string
	Headers         map[string]string
	Env             map[string]string
	RequiredEnv     []string
	Transport       McpTransport
	ProtocolVersion string
	Capabilities    McpCapabilities
	Source          DefinitionSource
	ToolRegName     string
	InstallMethod   McpInstallMethod
	PackageName     string
	// Version is the exact release of PackageName that the lifecycle
	// installs; an unversioned package is never installed.
	Version string
	// Bin is the executable InstallMethod provides.
	Bin string
}

// HealthResult wraps a health check outcome with caching metadata.
type HealthResult struct {
	*mcphealth.ServerHealth
	CheckedAt time.Time
	Stale     bool
}
