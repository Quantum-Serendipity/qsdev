package devenv

import (
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
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
	return ecosystem.ContainsStr(slice, val)
}

// ExportAnswersPath exposes answersPath for external tests.
var ExportAnswersPath = func(projectRoot string) string {
	return answersPath(projectRoot)
}

// ExportDevenvCmd exposes devenvCmd for external tests.
var ExportDevenvCmd = devenvCmd

// ExportNewDevenvGenerator re-exports NewDevenvGenerator for test clarity.
// (NewDevenvGenerator is already exported, but this pattern keeps the export
// file consistent.)
var _ types.Generator = (*DevenvGenerator)(nil)
