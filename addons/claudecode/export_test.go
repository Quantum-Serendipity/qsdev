package claudecode

import (
	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ExportMCPServerConfig re-exports MCPServerConfig for convenience in tests.
type ExportMCPServerConfig = MCPServerConfig

var (
	ExportLoadManifest = loadManifest
	ExportDeploySkills = deploySkills
	ExportDeployRules  = deployRules
)

// ExportSaveAnswers exposes saveAnswers for external tests.
var ExportSaveAnswers = saveAnswers

// ExportLoadAnswers exposes loadAnswers for external tests.
var ExportLoadAnswers = loadAnswers

// ExportBuildClaudeAnswersFromFlags exposes buildClaudeAnswersFromFlags for external tests.
//
// Parameters: projectRoot, preset string, skills, mcpServers []string, yes, noSafetyBlock bool
var ExportBuildClaudeAnswersFromFlags = buildClaudeAnswersFromFlags

// ExportValidPermissionPresets exposes validPermissionPresets for external tests.
var ExportValidPermissionPresets = validPermissionPresets

// ExportValidHookPresets exposes validHookPresets for external tests.
var ExportValidHookPresets = validHookPresets

// ExportClaudeCmd exposes claudeCmd for external tests.
var ExportClaudeCmd = claudeCmd

// ExportHookPresetToChoices exposes hookPresetToChoices for external tests.
var ExportHookPresetToChoices = func(name string, hooks *types.HookChoices) {
	hookPresetToChoices(name, hooks)
}

// ExportAnswersPath exposes answersPath for external tests.
var ExportAnswersPath = func(projectRoot string) string {
	return answersPath(projectRoot)
}

var (
	ExportComputeTemplateVersion     = ComputeTemplateVersion
	ExportComputeSkillLibraryVersion = ComputeSkillLibraryVersion
	ExportCompareVersions            = CompareVersions
	ExportBuildUpdateSummary         = BuildUpdateSummary
	ExportIsLibrarySkill             = IsLibrarySkill
	ExportIsLibraryRule              = IsLibraryRule
)

type ExportVersionDiff = VersionDiff
type ExportUpdateSummary = UpdateSummary

// ExportContains exposes the contains helper for external tests.
var ExportContains = sliceutil.Contains

var ExportLoadQsdevOpsManifest = loadQsdevOpsManifest
var ExportDeployOperationSkills = deployOperationSkills
var ExportLoadAgentManifest = loadAgentManifest
var ExportDeployAgents = deployAgents
var ExportLoadConsultingSkillManifest = loadConsultingSkillManifest
var ExportDeployWorkflowSkills = deployWorkflowSkills

type ExportQsdevOpsManifest = QsdevOpsManifest
type ExportQsdevOpsEntry = QsdevOpsEntry
type ExportAgentManifest = AgentManifest
type ExportAgentEntry = AgentEntry
type ExportConsultingSkillManifest = ConsultingSkillManifest
type ExportConsultingSkillEntry = ConsultingSkillEntry

// Hook registry exports for external tests.
type ExportHookDefinition = HookDefinition
type ExportHookStatus = HookStatus

// ExportDefaultSecretPatterns exposes DefaultSecretPatterns for external tests.
var ExportDefaultSecretPatterns = DefaultSecretPatterns

// ExportPlaceholderIndicators exposes PlaceholderIndicators for external tests.
var ExportPlaceholderIndicators = PlaceholderIndicators

const (
	ExportTierProject = TierProject
	ExportTierTeam    = TierTeam
	ExportTierOrg     = TierOrg
)

var (
	ExportNewHookRegistry     = NewHookRegistry
	ExportDefaultHookRegistry = defaultHookRegistry
	ExportBuildHookStatuses   = buildHookStatuses
	ExportHooksCmd            = hooksCmd
)
