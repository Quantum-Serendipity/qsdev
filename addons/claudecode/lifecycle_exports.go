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

// DeployOperationSkills is an exported wrapper around deployOperationSkills
// for use by the tool lifecycle system.
func DeployOperationSkills(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	return deployOperationSkills(answers)
}

// DeployAgents is an exported wrapper around deployAgents
// for use by the tool lifecycle system.
func DeployAgents(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	return deployAgents(answers)
}

// DeployWorkflowSkills is an exported wrapper around deployWorkflowSkills
// for use by the tool lifecycle system.
func DeployWorkflowSkills(answers types.WizardAnswers, registry *ecosystem.Registry) ([]types.GeneratedFile, error) {
	return deployWorkflowSkills(answers, registry)
}
