package posture

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// CheckName is a typed string identifying a conformance check.
type CheckName string

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

// CheckStatus is the outcome of a conformance check or level.
type CheckStatus string

const (
	// CheckPass: the requirement is met.
	CheckPass CheckStatus = "pass"
	// CheckFail: the requirement is not met.
	CheckFail CheckStatus = "fail"
	// CheckUnknown: the requirement could not be evaluated because the input
	// it reads was never measured (e.g. vulnerability counts without a
	// dependency scan). It does not count as a pass, and it is not a failure.
	CheckUnknown CheckStatus = "unknown"
)

// Label returns the status in upper case for summary output.
func (s CheckStatus) Label() string {
	return strings.ToUpper(string(s))
}

// statusRank orders statuses from best to worst for worstStatus.
var statusRank = map[CheckStatus]int{CheckPass: 0, CheckUnknown: 1, CheckFail: 2}

// worstStatus returns the worst of the given statuses (fail > unknown >
// pass); CheckPass for none.
func worstStatus(statuses ...CheckStatus) CheckStatus {
	worst := CheckPass
	for _, s := range statuses {
		if statusRank[s] > statusRank[worst] {
			worst = s
		}
	}
	return worst
}

// verdict returns status, or the pass/fail status Pass implies when status is
// unset (a report written before the status field existed).
func verdict(status CheckStatus, pass bool) CheckStatus {
	switch {
	case status != "":
		return status
	case pass:
		return CheckPass
	default:
		return CheckFail
	}
}

// NewCheck builds a conformance check with the given status; Pass is set only
// for CheckPass.
func NewCheck(name CheckName, status CheckStatus, reason string) ConformanceCheck {
	return ConformanceCheck{Name: name, Pass: status == CheckPass, Status: status, Reason: reason}
}

// passFail maps a boolean result onto CheckPass or CheckFail.
func passFail(ok bool) CheckStatus {
	if ok {
		return CheckPass
	}
	return CheckFail
}

// NewLevel builds a conformance level whose status is the worst of its
// checks' statuses and of floor (the status of a level it builds on; pass it
// CheckPass for a standalone level). Pass is set only for CheckPass.
func NewLevel(floor CheckStatus, checks []ConformanceCheck) ConformanceLevel {
	status := floor
	for _, c := range checks {
		status = worstStatus(status, c.Verdict())
	}
	return ConformanceLevel{Pass: status == CheckPass, Status: status, Checks: checks}
}

// EvaluateConformance checks baseline and enhanced conformance. genState must
// list only the generated files that are present on disk, so a deleted file
// never passes a presence check. Enhanced builds on baseline, so its status is
// never better than baseline's.
func EvaluateConformance(
	defense DefenseCoverage,
	deps DependencyHealth,
	enabledTools map[string]bool,
	genState types.GeneratedState,
) ConformanceResult {
	baseline := NewLevel(CheckPass, evaluateBaseline(defense, deps, genState))
	enhanced := NewLevel(baseline.Status, evaluateEnhanced(defense, deps, enabledTools, genState))
	return ConformanceResult{Baseline: baseline, Enhanced: enhanced}
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
	checks = append(checks, NewCheck(CheckLockFilesPresent, passFail(allLocked),
		boolReason(allLocked, "all detected ecosystems have lock files", "some ecosystems missing lock files")))

	// Only a conclusive scan can certify "no critical vulnerabilities": a failed
	// ecosystem or an unresolved-severity vuln leaves zero Totals meaning
	// "unknown", not "clean", so a non-certifiable result never passes, and
	// without a scan the check is unknown rather than passed.
	checks = append(checks, NewCheck(CheckNoCriticalVulns,
		vulnCheckStatus(deps, deps.Totals.Critical == 0),
		vulnCheckReason(deps, deps.Totals.Critical == 0,
			"no critical vulnerabilities", "critical vulnerabilities found")))

	_, hasClaudeMD := genState.Files["CLAUDE.md"]
	checks = append(checks, NewCheck(CheckClaudeMDPresent, passFail(hasClaudeMD),
		boolReason(hasClaudeMD, "CLAUDE.md present", "CLAUDE.md missing or not generated")))

	_, hasSettings := genState.Files[".claude/settings.json"]
	checks = append(checks, NewCheck(CheckSettingsJSONPresent, passFail(hasSettings),
		boolReason(hasSettings, "settings.json present", "settings.json missing or not generated")))

	// Baseline is a fixed floor, deliberately independent of the progressive
	// tier: every high/critical layer is required at every tier, even though
	// the tier-relative defense score leaves higher-tier layers out.
	highLayersOK := true
	for _, l := range defense.Layers {
		if l.Weight == WeightHigh || l.Weight == WeightCritical {
			if l.Status != LayerEnabled && l.Status != LayerNotApplicable {
				highLayersOK = false
				break
			}
		}
	}
	checks = append(checks, NewCheck(CheckHighWeightLayersOn, passFail(highLayersOK),
		boolReason(highLayersOK, "all high/critical defense layers enabled", "some high/critical defense layers not enabled")))

	hasPreCommit := false
	for path := range genState.Files {
		if path == ".pre-commit-config.yaml" || path == ".husky/pre-commit" || path == ".githooks/pre-commit" {
			hasPreCommit = true
			break
		}
	}
	checks = append(checks, NewCheck(CheckPreCommitHooks, passFail(hasPreCommit),
		boolReason(hasPreCommit, "pre-commit hooks configured", "no pre-commit hook configuration present")))

	return checks
}

func evaluateEnhanced(
	defense DefenseCoverage,
	deps DependencyHealth,
	enabledTools map[string]bool,
	genState types.GeneratedState,
) []ConformanceCheck {
	var checks []ConformanceCheck

	checks = append(checks, NewCheck(CheckNoHighVulns,
		vulnCheckStatus(deps, deps.Totals.High == 0),
		vulnCheckReason(deps, deps.Totals.High == 0,
			"no high vulnerabilities", "high vulnerabilities found")))

	semgrepEnabled := enabledTools["semgrep"]
	checks = append(checks, NewCheck(CheckSASTEnabled, passFail(semgrepEnabled),
		boolReason(semgrepEnabled, "semgrep SAST enabled", "semgrep not enabled")))

	gitleaksEnabled := enabledTools["gitleaks"]
	checks = append(checks, NewCheck(CheckSecretsScanningEnabled, passFail(gitleaksEnabled),
		boolReason(gitleaksEnabled, "gitleaks secrets scanning enabled", "gitleaks not enabled")))

	licenseEnabled := enabledTools["license-compliance"]
	checks = append(checks, NewCheck(CheckLicenseComplianceOn, passFail(licenseEnabled),
		boolReason(licenseEnabled, "license compliance enabled", "license compliance not enabled")))

	ageGating := FindLayerByName(defense.Layers, "age-gating")
	ageGatingOK := ageGating != nil && ageGating.Status == LayerEnabled
	checks = append(checks, NewCheck(CheckAgeGatingConfigured, passFail(ageGatingOK),
		boolReason(ageGatingOK, "age-gating configured", "age-gating not configured")))

	ciGenerated := false
	for path := range genState.Files {
		if strings.HasPrefix(path, ".github/workflows/") {
			ciGenerated = true
			break
		}
	}
	checks = append(checks, NewCheck(CheckCIWorkflowsGenerated, passFail(ciGenerated),
		boolReason(ciGenerated, "CI workflows generated", "no CI workflows in generated state")))

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

// vulnCheckStatus returns the status of a vulnerability-count conformance
// check whose count is zero when ok. Unscanned dependencies leave the count
// unmeasured, so the check is unknown; a scan that failed or left severities
// unresolved cannot certify clean, so it fails; a project with no dependency
// ecosystem has nothing vulnerable and passes.
func vulnCheckStatus(deps DependencyHealth, ok bool) CheckStatus {
	switch deps.ScanStatus() {
	case DepUnscanned:
		return CheckUnknown
	case DepNotApplicable:
		return CheckPass
	default:
		return passFail(deps.Certifiable() && ok)
	}
}

// vulnCheckReason renders the reason for a vulnerability-count conformance check.
// A zero count is a clean result only when the scan was conclusive. It reports,
// in priority order, an outright scan failure, unresolved-severity vulnerabilities
// (a scan that completed but could not certify clean), and a scan that never ran
// — each stated explicitly instead of claiming the absence of a given severity.
// This keeps the report honest: it reports what could not be confirmed rather
// than implying a clean bill of health.
func vulnCheckReason(deps DependencyHealth, ok bool, passReason, failReason string) string {
	switch {
	case deps.ScanFailed:
		return "dependency vulnerability scan failed; results unavailable " +
			"(vulnerability status unknown, not confirmed clean)"
	case deps.Totals.Unknown > 0:
		return "unresolved-severity vulnerabilities present; not confirmed clean"
	case deps.ScanStatus() == DepNotApplicable:
		return "no dependency ecosystems detected"
	case !deps.Scanned:
		return "dependencies not scanned for vulnerabilities; run 'qsdev status --scan' " +
			"(vulnerability status unknown, not confirmed clean)"
	default:
		return boolReason(ok, passReason, failReason)
	}
}
