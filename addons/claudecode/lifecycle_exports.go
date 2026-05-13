package claudecode

import (
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// GeneratePostmortemSkill is an exported wrapper around generatePostmortemSkill
// for use by the tool lifecycle system (gdev enable/disable).
func GeneratePostmortemSkill(answers types.WizardAnswers, registry *ecosystem.Registry) (*types.GeneratedFile, error) {
	return generatePostmortemSkill(answers, registry)
}

// GenerateVersionSentinelFiles is an exported wrapper around generateVersionSentinelFiles
// for use by the tool lifecycle system.
func GenerateVersionSentinelFiles(answers types.WizardAnswers, registry *ecosystem.Registry) ([]types.GeneratedFile, error) {
	return generateVersionSentinelFiles(answers, registry)
}

// GenerateSembleFiles is an exported wrapper around generateSembleFiles
// for use by the tool lifecycle system.
func GenerateSembleFiles(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	return generateSembleFiles(answers)
}
