package aiframework

import (
	"context"
	"fmt"

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
	Detection   *FrameworkDetection
	Permissions *PermissionPolicy
	MCPServers  []MCPServerSpec
	Sandbox     *SandboxPolicy
	Model       *ModelPreferences
	Hooks       *HookConfiguration
}

// ValidationSeverity indicates the severity of a validation issue.
type ValidationSeverity int

const (
	SeverityWarning ValidationSeverity = iota
	SeverityError
)

var validationSeverityNames = [...]string{
	SeverityWarning: "warning",
	SeverityError:   "error",
}

func (s ValidationSeverity) String() string {
	if int(s) >= 0 && int(s) < len(validationSeverityNames) {
		return validationSeverityNames[s]
	}
	return "unknown"
}

func (s ValidationSeverity) MarshalText() ([]byte, error) {
	str := s.String()
	if str == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown ValidationSeverity value %d", int(s))
	}
	return []byte(str), nil
}

func (s *ValidationSeverity) UnmarshalText(text []byte) error {
	for i, name := range validationSeverityNames {
		if name == string(text) {
			*s = ValidationSeverity(i)
			return nil
		}
	}
	return fmt.Errorf("unknown validation severity: %q", string(text))
}

// ValidationIssue reports a problem found during config validation.
type ValidationIssue struct {
	Path     string
	Message  string
	Severity ValidationSeverity
}
