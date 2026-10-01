package claudecode

import (
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ExportMCPServerConfig re-exports MCPServerConfig for convenience in tests.
type ExportMCPServerConfig = MCPServerConfig

var (
	ExportLoadManifest        = loadManifest
	ExportDeploySkills        = deploySkills
	ExportDeployRules         = deployRules
	ExportLegacyFlatSkillPath = legacyFlatSkillPath
)

// ExportSaveAnswers exposes saveAnswers for external tests.
var ExportSaveAnswers = saveAnswers

// ExportLoadAnswers exposes loadAnswers for external tests.
var ExportLoadAnswers = loadAnswers

// ExportBuildClaudeAnswersFromFlags exposes buildClaudeAnswersFromFlags for external tests.
//
// Parameters: projectRoot, preset string, skills, mcpServers []string, yes bool
var ExportBuildClaudeAnswersFromFlags = buildClaudeAnswersFromFlags

// ExportValidPermissionPresets exposes the permission presets init accepts for external tests.
var ExportValidPermissionPresets = validation.PermissionPresets()

// ExportClaudeCmd exposes claudeCmd for external tests.
var ExportClaudeCmd = claudeCmd

// ExportContentCmd exposes contentCmd for external tests.
var ExportContentCmd = contentCmd

// ExportDocsCmd exposes docsCmd for external tests.
var ExportDocsCmd = docsCmd

// ExportVerifyDocSet exposes verifyDocSet for external tests.
var ExportVerifyDocSet = verifyDocSet

// ExportDocVerifyResult re-exports docVerifyResult for external tests.
type ExportDocVerifyResult = docVerifyResult

// ExportAnswersPath exposes answersPath for external tests.
var ExportAnswersPath = func(projectRoot string) string {
	return answersPath(projectRoot)
}

var (
	ExportComputeTemplateVersion     = ComputeTemplateVersion
	ExportComputeSkillLibraryVersion = ComputeSkillLibraryVersion
	ExportCompareVersions            = CompareVersions
	ExportBuildUpdateSummary         = BuildUpdateSummary
)

type ExportVersionDiff = VersionDiff
type ExportUpdateSummary = UpdateSummary

// ExportContains exposes the contains helper for external tests.
var ExportContains = slices.Contains[[]string, string]

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

// ExportConfigSecretPatterns exposes ConfigSecretPatterns for external tests.
var ExportConfigSecretPatterns = ConfigSecretPatterns

// ExportPlaceholderIndicators exposes PlaceholderIndicators for external tests.
var ExportPlaceholderIndicators = PlaceholderIndicators

var (
	ExportNewHookRegistry     = NewHookRegistry
	ExportDefaultHookRegistry = defaultHookRegistry
	ExportBuildHookStatuses   = buildHookStatuses
	ExportHooksCmd            = hooksCmd
	ExportTemplateFS          = templateFS
	ExportGenerateHookFiles   = GenerateHookFiles
	ExportFailClosedCommand   = failClosedCommand
	ExportBuildHooks          = buildHooks
)

// ExportEmittedCommand exposes HookDefinition.emittedCommand for external tests.
func ExportEmittedCommand(h HookDefinition, answers types.WizardAnswers, app string) string {
	return h.emittedCommand(answers, app)
}

var ExportIsTemplateTestFixture = isTemplateTestFixture

// ExportClaudeSpec exposes claudeSpec for external tests.
var ExportClaudeSpec = claudeSpec
