package toolreg

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// DefaultPolicy controls when a tool is enabled during initial project setup.
type DefaultPolicy int

const (
	AlwaysOn       DefaultPolicy = iota // Enabled in all projects.
	OnWhenDetected                      // Enabled when DetectFunc returns true.
	OptIn                               // Disabled by default; user must explicitly enable.
	AlwaysOff                           // Never auto-enabled (internal/deprecated tools).
)

func (d DefaultPolicy) String() string {
	switch d {
	case AlwaysOn:
		return "always-on"
	case OnWhenDetected:
		return "on-when-detected"
	case OptIn:
		return "opt-in"
	case AlwaysOff:
		return "always-off"
	default:
		return "unknown"
	}
}

// OwnershipType distinguishes exclusive vs shared file ownership.
type OwnershipType int

const (
	Exclusive OwnershipType = iota // Tool owns the entire file.
	Shared                         // Tool contributes a section/entry to a file other tools also write to.
)

func (o OwnershipType) String() string {
	switch o {
	case Exclusive:
		return "exclusive"
	case Shared:
		return "shared"
	default:
		return "unknown"
	}
}

// ToolCategory groups tools for display and filtering.
type ToolCategory string

const (
	CategorySecurity       ToolCategory = "security"
	CategoryAIAgent        ToolCategory = "ai-agent"
	CategoryDevEx          ToolCategory = "devex"
	CategoryInfrastructure ToolCategory = "infrastructure"
)

// DisplayName returns a human-friendly label for the category.
func (c ToolCategory) DisplayName() string {
	switch c {
	case CategorySecurity:
		return "Security"
	case CategoryAIAgent:
		return "AI Agent"
	case CategoryDevEx:
		return "Developer Experience"
	case CategoryInfrastructure:
		return "Infrastructure"
	default:
		return string(c)
	}
}

// FileOwnership maps a tool to a file it creates or contributes to.
type FileOwnership struct {
	Path           string        // Relative path from project root.
	Ownership      OwnershipType // Exclusive or Shared.
	SectionID      string        // For shared files: section identifier used in markers.
	SectionContent string        // Go template or static content for this section.
}

// DetectFunc determines whether a tool should be auto-enabled based on project state.
type DetectFunc func(detected types.DetectedProject) bool

// EnableFunc modifies WizardAnswers to reflect a tool being enabled.
type EnableFunc func(answers *types.WizardAnswers)

// DisableFunc modifies WizardAnswers to reflect a tool being disabled.
type DisableFunc func(answers *types.WizardAnswers)

// GenerateFunc produces the exclusive files for a tool given the current answers.
type GenerateFunc func(answers types.WizardAnswers) ([]types.GeneratedFile, error)

// SectionDataFunc provides template data for rendering section_content templates.
type SectionDataFunc func(answers types.WizardAnswers, ecoReg *ecosystem.Registry) map[string]any

// SharedContentFunc produces the content to insert into a shared file section.
type SharedContentFunc func(answers types.WizardAnswers) ([]byte, error)

// Tool defines a toggleable tool in the lifecycle system.
type Tool struct {
	Name          string
	DisplayName   string
	Category      ToolCategory
	Description   string
	Default       DefaultPolicy
	DetectFunc    DetectFunc
	Prerequisites []string // Tool names that must be enabled first.
	Conflicts     []string // Tool names that cannot coexist.
	OwnedFiles    []FileOwnership
	EnableFunc    EnableFunc
	DisableFunc   DisableFunc
	GenerateFunc  GenerateFunc // Produces exclusive files.

	// SharedContent maps SectionID to a function that produces the content
	// to insert into the shared file. Used during enable operations.
	SharedContent map[string]SharedContentFunc

	// SectionDataFunc provides template data for rendering section_content
	// templates in CLAUDE.md. Only needed for tools with dynamic content.
	SectionDataFunc SectionDataFunc
}

// ExclusiveFiles returns all files this tool exclusively owns.
func (t *Tool) ExclusiveFiles() []FileOwnership {
	var result []FileOwnership
	for _, f := range t.OwnedFiles {
		if f.Ownership == Exclusive {
			result = append(result, f)
		}
	}
	return result
}

// SharedFiles returns all files this tool contributes sections to.
func (t *Tool) SharedFiles() []FileOwnership {
	var result []FileOwnership
	for _, f := range t.OwnedFiles {
		if f.Ownership == Shared {
			result = append(result, f)
		}
	}
	return result
}
