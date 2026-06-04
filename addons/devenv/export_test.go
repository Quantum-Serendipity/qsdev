package devenv

import (
	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ExportServiceToTemplateData exposes serviceToTemplateData for external tests.
var ExportServiceToTemplateData = func(svc types.ServiceChoice) (ServiceTemplateData, error) {
	return serviceToTemplateData(svc)
}

// ExportSaveAnswers exposes saveAnswers for external tests.
var ExportSaveAnswers = saveAnswers

// ExportLoadAnswers exposes loadAnswers for external tests.
var ExportLoadAnswers = loadAnswers

// ExportBuildAnswersFromFlags exposes buildAnswersFromFlags for external tests.
var ExportBuildAnswersFromFlags = buildAnswersFromFlags

// ExportValidServices exposes validServices for external tests.
var ExportValidServices = validServices

// ExportValidLanguages exposes validLanguages for external tests.
var ExportValidLanguages = validLanguages

// ExportContains exposes contains for external tests.
var ExportContains = func(slice []string, val string) bool {
	return sliceutil.Contains(slice, val)
}

// ExportAnswersPath exposes answersPath for external tests.
var ExportAnswersPath = func(projectRoot string) string {
	return answersPath(projectRoot)
}

// ExportDevenvCmd exposes devenvCmd for external tests.
var ExportDevenvCmd = devenvCmd

// ExportCompletionCmd exposes completionCmd for external tests.
var ExportCompletionCmd = completionCmd

// ExportDetectShell exposes detectShell for external tests.
var ExportDetectShell = detectShell

// ExportDefaultRCFile exposes defaultRCFile for external tests.
var ExportDefaultRCFile = defaultRCFile

// ExportNewDevenvGenerator re-exports NewDevenvGenerator for test clarity.
// (NewDevenvGenerator is already exported, but this pattern keeps the export
// file consistent.)
var _ types.Generator = (*DevenvGenerator)(nil)
