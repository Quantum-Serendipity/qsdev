package aiframework

import (
	"context"

	"github.com/Quantum-Serendipity/qsdev/internal/enumtext"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ConfigRenderer generates framework-specific configuration files from
// framework-agnostic policy input.
type ConfigRenderer interface {
	FrameworkID() FrameworkID
	Capabilities() ConfigCapabilities
	Render(ctx context.Context, input *PolicyInput) ([]types.GeneratedFile, error)
	Validate(ctx context.Context, files []types.GeneratedFile) []ValidationIssue
	Format() string
}

// ConfigCapabilities reports which configuration aspects a renderer handles.
type ConfigCapabilities struct {
	RendersPermissions bool
	RendersMCP         bool
	RendersHooks       bool
	RendersSandbox     bool
	RendersIgnore      bool
}

// PolicyInput aggregates all framework-agnostic policy declarations that a
// ConfigRenderer translates into framework-specific files.
type PolicyInput struct {
	ProjectRoot string
	Permissions *PermissionPolicy
	MCPServers  []MCPServerSpec
	Sandbox     *SandboxPolicy
	Model       *ModelPreferences
	Hooks       *HookConfiguration
}

// ValidationSeverity indicates the severity of a validation issue.
//
// The zero value is SeverityUnknown, which never marshals and must be treated
// as an error, so an issue whose severity was forgotten never downgrades to a
// warning.
type ValidationSeverity int

const (
	SeverityUnknown ValidationSeverity = iota // Unset; treat as error, never marshalled.
	SeverityWarning
	SeverityError
)

var validationSeverityNames = [...]string{
	SeverityUnknown: "",
	SeverityWarning: "warning",
	SeverityError:   "error",
}

var validationSeverityText = enumtext.New[ValidationSeverity]("ValidationSeverity", "validation severity", "unknown", validationSeverityNames[:])

func (s ValidationSeverity) String() string { return validationSeverityText.String(s) }

func (s ValidationSeverity) MarshalText() ([]byte, error) {
	return validationSeverityText.MarshalText(s)
}

func (s *ValidationSeverity) UnmarshalText(text []byte) error {
	return validationSeverityText.UnmarshalText(text, s)
}

// ValidationIssue reports a problem found during config validation.
type ValidationIssue struct {
	Path     string
	Message  string
	Severity ValidationSeverity
}
