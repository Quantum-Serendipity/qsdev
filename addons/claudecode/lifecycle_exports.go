package claudecode

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GeneratePostmortemSkill is an exported wrapper around generatePostmortemSkill
// for use by the tool lifecycle system (qsdev enable/disable).
func GeneratePostmortemSkill(answers types.WizardAnswers, registry *ecosystem.Registry) (*types.GeneratedFile, error) {
	return generatePostmortemSkill(answers, registry)
}

// GenerateVersionSentinelFiles is an exported wrapper around generateVersionSentinelFiles
// for use by the tool lifecycle system.
func GenerateVersionSentinelFiles(answers types.WizardAnswers, registry *ecosystem.Registry) ([]types.GeneratedFile, error) {
	return generateVersionSentinelFiles(answers, registry)
}

// GenerateSembleFiles is an exported wrapper around generateSembleConfig
// for use by the tool lifecycle system.
func GenerateSembleFiles(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	sr, err := generateSembleConfig(answers)
	if err != nil {
		return nil, err
	}
	if sr == nil {
		return nil, nil
	}
	return sr.Files, nil
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
