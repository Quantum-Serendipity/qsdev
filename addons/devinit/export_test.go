package devinit

import "github.com/spf13/cobra"

// ExportLanguageSpec re-exports LanguageSpec for external tests.
type ExportLanguageSpec = LanguageSpec

// ExportProfile re-exports Profile for external tests.
type ExportProfile = Profile

// ExportProfileSummary re-exports ProfileSummary for external tests.
type ExportProfileSummary = ProfileSummary

// ExportProjectProfileRegistry re-exports ProjectProfileRegistry for external tests.
type ExportProjectProfileRegistry = ProjectProfileRegistry

// ExportExistingConfig re-exports ExistingConfig for external tests.
type ExportExistingConfig = ExistingConfig

// ExportLanguageOption re-exports LanguageOption for external tests.
type ExportLanguageOption = LanguageOption

var (
	// ExportNewProjectProfileRegistry exposes NewProjectProfileRegistry for external tests.
	ExportNewProjectProfileRegistry = NewProjectProfileRegistry

	// ExportDefaultProjectProfileRegistry exposes DefaultProjectProfileRegistry for external tests.
	ExportDefaultProjectProfileRegistry = DefaultProjectProfileRegistry

	// ExportProfileToAnswers exposes ProfileToAnswers for external tests.
	ExportProfileToAnswers = ProfileToAnswers

	// ExportMergeProfileWithFlags exposes MergeProfileWithFlags for external tests.
	ExportMergeProfileWithFlags = MergeProfileWithFlags

	// ExportHooksFromStrings exposes hooksFromStrings for external tests.
	ExportHooksFromStrings = hooksFromStrings

	// ExportGoWeb exposes the GoWeb built-in profile.
	ExportGoWeb = GoWeb

	// ExportTSFullstack exposes the TSFullstack built-in profile.
	ExportTSFullstack = TSFullstack

	// ExportPythonData exposes the PythonData built-in profile.
	ExportPythonData = PythonData

	// ExportRustCLI exposes the RustCLI built-in profile.
	ExportRustCLI = RustCLI

	// ExportJavaWeb exposes the JavaWeb built-in profile.
	ExportJavaWeb = JavaWeb

	// ExportPythonWeb exposes the PythonWeb built-in profile.
	ExportPythonWeb = PythonWeb

	// ExportTSBackend exposes the TSBackend built-in profile.
	ExportTSBackend = TSBackend

	// ExportElixirWeb exposes the ElixirWeb built-in profile.
	ExportElixirWeb = ElixirWeb

	// ExportRustWeb exposes the RustWeb built-in profile.
	ExportRustWeb = RustWeb

	// ExportDotnetWeb exposes the DotnetWeb built-in profile.
	ExportDotnetWeb = DotnetWeb

	// ExportMapDetectionToDefaults exposes MapDetectionToDefaults for external tests.
	ExportMapDetectionToDefaults = MapDetectionToDefaults

	// ExportDetectExistingConfig exposes DetectExistingConfig for external tests.
	ExportDetectExistingConfig = DetectExistingConfig

	// ExportBuildLanguageOptions exposes BuildLanguageOptions for external tests.
	ExportBuildLanguageOptions = BuildLanguageOptions

	// ExportDetectionAnnotation exposes DetectionAnnotation for external tests.
	ExportDetectionAnnotation = DetectionAnnotation

	// ExportPreSelectedLanguages exposes PreSelectedLanguages for external tests.
	ExportPreSelectedLanguages = PreSelectedLanguages

	// ExportQuickPathSummary exposes QuickPathSummary for external tests.
	ExportQuickPathSummary = QuickPathSummary

	// ExportExtractRepoName exposes extractRepoName for external tests.
	ExportExtractRepoName = extractRepoName

	// ExportAnswersFromFlags exposes AnswersFromFlags for external tests.
	ExportAnswersFromFlags = AnswersFromFlags

	// ExportValidateAnswers exposes ValidateAnswers for external tests.
	ExportValidateAnswers = ValidateAnswers

	// ExportLoadAnswersFile exposes LoadAnswersFile for external tests.
	ExportLoadAnswersFile = LoadAnswersFile

	// ExportLoadAnswersFromReader exposes LoadAnswersFromReader for external tests.
	ExportLoadAnswersFromReader = LoadAnswersFromReader

	// ExportValidateAnswersFileCompleteness exposes ValidateAnswersFileCompleteness for external tests.
	ExportValidateAnswersFileCompleteness = ValidateAnswersFileCompleteness

	// ExportRegisterInitFlags exposes RegisterInitFlags for external tests.
	ExportRegisterInitFlags = RegisterInitFlags

	// ExportNewFlagSet exposes NewFlagSet for external tests.
	ExportNewFlagSet = func(cmd *cobra.Command) *FlagSet {
		return NewFlagSet(cmd)
	}
)

// ExportUpdateOptions re-exports UpdateOptions for external tests.
type ExportUpdateOptions = UpdateOptions

// ExportUpdateAction re-exports UpdateAction for external tests.
type ExportUpdateAction = UpdateAction

// ExportFileUpdatePlan re-exports FileUpdatePlan for external tests.
type ExportFileUpdatePlan = FileUpdatePlan

// ExportUpdatePlan re-exports UpdatePlan for external tests.
type ExportUpdatePlan = UpdatePlan

// ExportInitOptions re-exports InitOptions for external tests.
type ExportInitOptions = InitOptions

// ExportFlagSet re-exports FlagSet for external tests.
type ExportFlagSet = FlagSet

// ExportRunUpdate exposes runUpdate for external tests.
var ExportRunUpdate = runUpdate

// ExportRunCreate exposes runCreate for external tests.
var ExportRunCreate = runCreate

// ExportRunJoin exposes runJoin for external tests.
var ExportRunJoin = runJoin

// ExportRunRepair exposes runRepair for external tests.
var ExportRunRepair = runRepair

// ExportDetectOnboardingMode exposes DetectOnboardingMode for external tests.
var ExportDetectOnboardingMode = DetectOnboardingMode

// ExportOverrideMode exposes overrideMode for external tests.
var ExportOverrideMode = overrideMode

// ExportCheckPrerequisites exposes CheckPrerequisites for external tests.
var ExportCheckPrerequisites = CheckPrerequisites

// ExportEnsureGitignoreEntry exposes EnsureGitignoreEntry for external tests.
var ExportEnsureGitignoreEntry = EnsureGitignoreEntry

// ExportGenerateLocalConfigTemplate exposes GenerateLocalConfigTemplate for external tests.
var ExportGenerateLocalConfigTemplate = GenerateLocalConfigTemplate

// ExportConfigToAnswers exposes configToAnswers for external tests.
var ExportConfigToAnswers = configToAnswers

// ExportOnboardingMode re-exports OnboardingMode for external tests.
type ExportOnboardingMode = OnboardingMode

// ExportModeDetectionResult re-exports ModeDetectionResult for external tests.
type ExportModeDetectionResult = ModeDetectionResult

// ExportDriftReport re-exports DriftReport for external tests.
type ExportDriftReport = DriftReport

// ExportPrerequisiteStatus re-exports PrerequisiteStatus for external tests.
type ExportPrerequisiteStatus = PrerequisiteStatus

// ExportPrerequisiteResult re-exports PrerequisiteResult for external tests.
type ExportPrerequisiteResult = PrerequisiteResult

const (
	ExportModeCreate = ModeCreate
	ExportModeJoin   = ModeJoin
	ExportModeUpdate = ModeUpdate
	ExportModeRepair = ModeRepair
)

// ExportBuildUpdatePlan exposes buildUpdatePlan for external tests.
var ExportBuildUpdatePlan = buildUpdatePlan

// ExportPreviewUpdatePlan exposes previewUpdatePlan for external tests.
var ExportPreviewUpdatePlan = previewUpdatePlan

// ExportExecuteUpdatePlan exposes executeUpdatePlan for external tests.
var ExportExecuteUpdatePlan = executeUpdatePlan

// ExportDispatchMerge exposes dispatchMerge for external tests.
var ExportDispatchMerge = dispatchMerge

const (
	// ExportUpdateActionRegenerate exposes UpdateActionRegenerate for external tests.
	ExportUpdateActionRegenerate = UpdateActionRegenerate
	// ExportUpdateActionMerge exposes UpdateActionMerge for external tests.
	ExportUpdateActionMerge = UpdateActionMerge
	// ExportUpdateActionSkip exposes UpdateActionSkip for external tests.
	ExportUpdateActionSkip = UpdateActionSkip
	// ExportUpdateActionCreate exposes UpdateActionCreate for external tests.
	ExportUpdateActionCreate = UpdateActionCreate
	// ExportUpdateActionSidecar exposes UpdateActionSidecar for external tests.
	ExportUpdateActionSidecar = UpdateActionSidecar
)

// ExportIsMachineReadableFormat exposes isMachineReadableFormat for external tests.
var ExportIsMachineReadableFormat = isMachineReadableFormat

// ExportCheckCmd exposes checkCmd for external tests.
var ExportCheckCmd = checkCmd

// ExportInitCmd exposes initCmd for external tests.
var ExportInitCmd = initCmd

// ExportSaveAnswers exposes saveAnswers for external tests.
var ExportSaveAnswers = saveAnswers

// ExportLoadAnswers exposes loadAnswers for external tests.
var ExportLoadAnswers = loadAnswers

// ExportAnswersPath exposes answersPath for external tests.
var ExportAnswersPath = func(projectRoot string) string {
	return answersPath(projectRoot)
}

// ExportPostGenerationMessage exposes postGenerationMessage for external tests.
var ExportPostGenerationMessage = postGenerationMessage

// ExportFormState re-exports formState for external tests.
type ExportFormState = formState

// ExportMapFormToAnswers exposes mapFormToAnswers for external tests.
var ExportMapFormToAnswers = mapFormToAnswers

// ExportParseExtraPackages exposes parseExtraPackages for external tests.
var ExportParseExtraPackages = parseExtraPackages

// ExportIsAccessible exposes isAccessible for external tests.
var ExportIsAccessible = isAccessible

// ExportBuildWizardForm exposes buildWizardForm for external tests.
var ExportBuildWizardForm = buildWizardForm

// ExportBuildPlanPreview exposes buildPlanPreview for external tests.
var ExportBuildPlanPreview = buildPlanPreview

// NewExportFormState constructs a formState with the given options for external tests.
func NewExportFormState(opts ...func(*formState)) *formState {
	fs := &formState{}
	for _, opt := range opts {
		opt(fs)
	}
	return fs
}

// WithQuickChoice sets quickChoice on formState.
func WithQuickChoice(v string) func(*formState) {
	return func(fs *formState) { fs.quickChoice = v }
}

// WithSelectedLanguages sets selectedLanguages on formState.
func WithSelectedLanguages(v []string) func(*formState) {
	return func(fs *formState) { fs.selectedLanguages = v }
}

// WithGoVersion sets goVersion on formState.
func WithGoVersion(v string) func(*formState) {
	return func(fs *formState) { fs.goVersion = v }
}

// WithJSVersion sets jsVersion on formState.
func WithJSVersion(v string) func(*formState) {
	return func(fs *formState) { fs.jsVersion = v }
}

// WithPythonVersion sets pythonVersion on formState.
func WithPythonVersion(v string) func(*formState) {
	return func(fs *formState) { fs.pythonVersion = v }
}

// WithSelectedServices sets selectedServices on formState.
func WithSelectedServices(v []string) func(*formState) {
	return func(fs *formState) { fs.selectedServices = v }
}

// WithDirenv sets direnv on formState.
func WithDirenv(v bool) func(*formState) {
	return func(fs *formState) { fs.direnv = v }
}

// WithGitHooks sets gitHooks on formState.
func WithGitHooks(v []string) func(*formState) {
	return func(fs *formState) { fs.gitHooks = v }
}

// WithExtraPackagesStr sets extraPackages on formState.
func WithExtraPackagesStr(v string) func(*formState) {
	return func(fs *formState) { fs.extraPackages = v }
}

// WithClaudeCode sets claudeCode on formState.
func WithClaudeCode(v bool) func(*formState) {
	return func(fs *formState) { fs.claudeCode = v }
}

// WithPermissionLevel sets permissionLevel on formState.
func WithPermissionLevel(v string) func(*formState) {
	return func(fs *formState) { fs.permissionLevel = v }
}

// WithSkills sets skills on formState.
func WithSkills(v []string) func(*formState) {
	return func(fs *formState) { fs.skills = v }
}

// WithAutoFormat sets autoFormat on formState.
func WithAutoFormat(v bool) func(*formState) {
	return func(fs *formState) { fs.autoFormat = v }
}

// WithSafetyBlock sets safetyBlock on formState.
func WithSafetyBlock(v bool) func(*formState) {
	return func(fs *formState) { fs.safetyBlock = v }
}

// WithMCPServers sets mcpServers on formState.
func WithMCPServers(v []string) func(*formState) {
	return func(fs *formState) { fs.mcpServers = v }
}

// WithConfirmed sets confirmed on formState.
func WithConfirmed(v bool) func(*formState) {
	return func(fs *formState) { fs.confirmed = v }
}
