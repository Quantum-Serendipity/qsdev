package claudecode

import "fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"

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
var ExportContains = contains
