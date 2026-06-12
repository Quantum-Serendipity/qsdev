package posture

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// CheckName is a typed string identifying a conformance check.
type CheckName = string

// Baseline conformance check names.
const (
	CheckLockFilesPresent    CheckName = "lock-files-present"
	CheckNoCriticalVulns     CheckName = "no-critical-vulns"
	CheckClaudeMDPresent     CheckName = "claude-md-present"
	CheckSettingsJSONPresent CheckName = "settings-json-present"
	CheckHighWeightLayersOn  CheckName = "high-weight-layers-enabled"
	CheckPreCommitHooks      CheckName = "pre-commit-hooks"
)

// Enhanced conformance check names.
const (
	CheckNoHighVulns            CheckName = "no-high-vulns"
	CheckSASTEnabled            CheckName = "sast-enabled"
	CheckSecretsScanningEnabled CheckName = "secrets-scanning-enabled"
	CheckLicenseComplianceOn    CheckName = "license-compliance-enabled"
	CheckAgeGatingConfigured    CheckName = "age-gating-configured"
	CheckCIWorkflowsGenerated   CheckName = "ci-workflows-generated"
)

// EvaluateConformance checks baseline and enhanced conformance.
func EvaluateConformance(
	defense DefenseCoverage,
	deps DependencyHealth,
	enabledTools map[string]bool,
	genState types.GeneratedState,
) ConformanceResult {
	baselineChecks := evaluateBaseline(defense, deps, genState)
	enhancedChecks := evaluateEnhanced(defense, deps, enabledTools, genState)

	baselinePass := true
	for _, c := range baselineChecks {
		if !c.Pass {
			baselinePass = false
			break
		}
	}

	enhancedPass := baselinePass
	if enhancedPass {
		for _, c := range enhancedChecks {
			if !c.Pass {
				enhancedPass = false
				break
			}
		}
	}

	return ConformanceResult{
		Baseline: ConformanceLevel{
			Pass:   baselinePass,
			Checks: baselineChecks,
		},
		Enhanced: ConformanceLevel{
			Pass:   enhancedPass,
			Checks: enhancedChecks,
		},
	}
}

func evaluateBaseline(
	defense DefenseCoverage,
	deps DependencyHealth,
	genState types.GeneratedState,
) []ConformanceCheck {
	var checks []ConformanceCheck

	allLocked := true
	for _, eco := range deps.Ecosystems {
		if eco.Detected && eco.LockFile == "missing" {
			allLocked = false
			break
		}
	}
	checks = append(checks, ConformanceCheck{
		Name:   CheckLockFilesPresent,
		Pass:   allLocked,
		Reason: boolReason(allLocked, "all detected ecosystems have lock files", "some ecosystems missing lock files"),
	})

	noCritical := deps.Totals.Critical == 0
	checks = append(checks, ConformanceCheck{
		Name:   CheckNoCriticalVulns,
		Pass:   noCritical,
		Reason: boolReason(noCritical, "no critical vulnerabilities", "critical vulnerabilities found"),
	})

	_, hasClaudeMD := genState.Files["CLAUDE.md"]
	checks = append(checks, ConformanceCheck{
		Name:   CheckClaudeMDPresent,
		Pass:   hasClaudeMD,
		Reason: boolReason(hasClaudeMD, "CLAUDE.md present in generated state", "CLAUDE.md not found in generated state"),
	})

	_, hasSettings := genState.Files[".claude/settings.json"]
	checks = append(checks, ConformanceCheck{
		Name:   CheckSettingsJSONPresent,
		Pass:   hasSettings,
		Reason: boolReason(hasSettings, "settings.json present in generated state", "settings.json not found in generated state"),
	})

	highLayersOK := true
	for _, l := range defense.Layers {
		if l.Weight == WeightHigh || l.Weight == WeightCritical {
			if l.Status != LayerEnabled && l.Status != LayerNotApplicable {
				highLayersOK = false
				break
			}
		}
	}
	checks = append(checks, ConformanceCheck{
		Name:   CheckHighWeightLayersOn,
		Pass:   highLayersOK,
		Reason: boolReason(highLayersOK, "all high/critical defense layers enabled", "some high/critical defense layers not enabled"),
	})

	hasPreCommit := false
	for path := range genState.Files {
		if path == ".pre-commit-config.yaml" || path == ".husky/pre-commit" || path == ".githooks/pre-commit" {
			hasPreCommit = true
			break
		}
	}
	checks = append(checks, ConformanceCheck{
		Name:   CheckPreCommitHooks,
		Pass:   hasPreCommit,
		Reason: boolReason(hasPreCommit, "pre-commit hooks configured", "no pre-commit hooks found in generated state"),
	})

	return checks
}

func evaluateEnhanced(
	defense DefenseCoverage,
	deps DependencyHealth,
	enabledTools map[string]bool,
	genState types.GeneratedState,
) []ConformanceCheck {
	var checks []ConformanceCheck

	noHighVulns := deps.Totals.High == 0
	checks = append(checks, ConformanceCheck{
		Name:   CheckNoHighVulns,
		Pass:   noHighVulns,
		Reason: boolReason(noHighVulns, "no high vulnerabilities", "high vulnerabilities found"),
	})

	semgrepEnabled := enabledTools["semgrep"]
	checks = append(checks, ConformanceCheck{
		Name:   CheckSASTEnabled,
		Pass:   semgrepEnabled,
		Reason: boolReason(semgrepEnabled, "semgrep SAST enabled", "semgrep not enabled"),
	})

	gitleaksEnabled := enabledTools["gitleaks"]
	checks = append(checks, ConformanceCheck{
		Name:   CheckSecretsScanningEnabled,
		Pass:   gitleaksEnabled,
		Reason: boolReason(gitleaksEnabled, "gitleaks secrets scanning enabled", "gitleaks not enabled"),
	})

	licenseEnabled := enabledTools["license-compliance"]
	checks = append(checks, ConformanceCheck{
		Name:   CheckLicenseComplianceOn,
		Pass:   licenseEnabled,
		Reason: boolReason(licenseEnabled, "license compliance enabled", "license compliance not enabled"),
	})

	ageGating := FindLayerByName(defense.Layers, "age-gating")
	ageGatingOK := ageGating != nil && ageGating.Status == LayerEnabled
	checks = append(checks, ConformanceCheck{
		Name:   CheckAgeGatingConfigured,
		Pass:   ageGatingOK,
		Reason: boolReason(ageGatingOK, "age-gating configured", "age-gating not configured"),
	})

	ciGenerated := false
	for path := range genState.Files {
		if strings.HasPrefix(path, ".github/workflows/") {
			ciGenerated = true
			break
		}
	}
	checks = append(checks, ConformanceCheck{
		Name:   CheckCIWorkflowsGenerated,
		Pass:   ciGenerated,
		Reason: boolReason(ciGenerated, "CI workflows generated", "no CI workflows in generated state"),
	})

	return checks
}

// FindLayerByName returns a pointer to the layer with the given name,
// or nil if not found.
func FindLayerByName(layers []DefenseLayer, name string) *DefenseLayer {
	for i := range layers {
		if layers[i].Name == name {
			return &layers[i]
		}
	}
	return nil
}

func boolReason(ok bool, pass, fail string) string {
	if ok {
		return pass
	}
	return fail
}
