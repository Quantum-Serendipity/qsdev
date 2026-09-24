package toolreg

import (
	"path/filepath"
	"strings"

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

// DevenvNixFile is the project-relative path of the generated devenv.nix,
// the shared file whose tool sections come from SharedContent.
const DevenvNixFile = "devenv.nix"

// SharedSection identifies one tool section in one shared file. A tool often
// uses the same SectionID in several files (e.g. devenv.nix and CLAUDE.md),
// and each file needs content in its own format, so content is keyed by both.
type SharedSection struct {
	Path      string
	SectionID string
}

// SectionOf returns the SharedSection a shared FileOwnership entry declares.
func SectionOf(f FileOwnership) SharedSection {
	return SharedSection{Path: f.Path, SectionID: f.SectionID}
}

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

	// SharedContent maps a shared-file section (path + SectionID) to a
	// function that produces that section's content in the file's format.
	// The devenv generator renders the devenv.nix entries for enabled tools.
	SharedContent map[SharedSection]SharedContentFunc

	// SectionDataFunc provides template data for rendering section_content
	// templates in CLAUDE.md. Only needed for tools with dynamic content.
	SectionDataFunc SectionDataFunc
}

// IsAgentTool reports whether the tool configures the AI coding agent itself
// (catalog category ai-agent: skills, sub-agents, MCP servers). The claudecode
// addon generates these tools' files; every other tool's files belong to the
// project and are generated whether or not Claude Code is configured.
func (t *Tool) IsAgentTool() bool {
	return t.Category == CategoryAIAgent
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

// OwnsExclusively reports whether relPath is one of the tool's exclusive
// files, or lies beneath an exclusive entry that names a directory (e.g.
// ".opengrep/rules/core" owns every rule file generated under it).
func (t *Tool) OwnsExclusively(relPath string) bool {
	relPath = filepath.ToSlash(relPath)
	for _, f := range t.ExclusiveFiles() {
		p := strings.TrimSuffix(filepath.ToSlash(f.Path), "/")
		if relPath == p || strings.HasPrefix(relPath, p+"/") {
			return true
		}
	}
	return false
}
