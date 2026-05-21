package posture

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// DefenseLayerNames lists the canonical names of all 10 defense layers.
var DefenseLayerNames = [...]string{
	"age-gating",
	"install-script-blocking",
	"lock-file-enforcement",
	"vulnerability-scanning",
	"pretooluse-hooks",
	"nix-hardening",
	"sast",
	"secrets-scanning",
	"container-security",
	"license-compliance",
}

// AssessDefenseLayers evaluates all 10 defense layers.
func AssessDefenseLayers(enabledTools map[string]bool, detected types.DetectedProject, genState types.GeneratedState, currentTier int) DefenseCoverage {
	layers := []DefenseLayer{
		assessPreToolUseHooks(enabledTools, genState),
		assessAgeGating(enabledTools, genState),
		assessInstallScriptBlocking(enabledTools),
		assessLockFileEnforcement(enabledTools, genState),
		assessVulnScanning(enabledTools, genState),
		assessNixHardening(genState),
		assessSAST(enabledTools, genState),
		assessSecretsScanning(enabledTools),
		assessContainerSecurity(enabledTools, detected),
		assessLicenseCompliance(enabledTools),
	}

	score := ComputeTierRelativeDefenseScore(layers, currentTier)

	enabled := 0
	total := 0
	for _, l := range layers {
		if l.Status == LayerNotApplicable {
			continue
		}
		total++
		if l.Status == LayerEnabled {
			enabled++
		}
	}

	return DefenseCoverage{
		Layers:  layers,
		Score:   score,
		Enabled: enabled,
		Total:   total,
	}
}

func assessPreToolUseHooks(enabledTools map[string]bool, genState types.GeneratedState) DefenseLayer {
	layer := DefenseLayer{
		Name:    "pretooluse-hooks",
		Weight:  WeightCritical,
		MinTier: 1,
	}

	attachGuardEnabled := enabledTools["attach-guard"]
	_, hasPackageGuard := genState.Files[".claude/hooks/package-guard.py"]

	if attachGuardEnabled && hasPackageGuard {
		layer.Status = LayerEnabled
		layer.Reason = "attach-guard enabled and package-guard.py present"
	} else if attachGuardEnabled || hasPackageGuard {
		layer.Status = LayerPartial
		layer.Score = 5
		if !attachGuardEnabled {
			layer.Reason = "package-guard.py present but attach-guard not enabled"
		} else {
			layer.Reason = "attach-guard enabled but package-guard.py not in state"
		}
	} else {
		layer.Status = LayerDisabled
		layer.Reason = "attach-guard not enabled"
	}
	return layer
}

func assessAgeGating(enabledTools map[string]bool, genState types.GeneratedState) DefenseLayer {
	layer := DefenseLayer{
		Name:    "age-gating",
		Weight:  WeightHigh,
		MinTier: 2,
	}

	// Age-gating is configured when the attach-guard is enabled and there are
	// per-ecosystem config files (age-gate-*.yaml) in the generated state.
	if !enabledTools["attach-guard"] {
		layer.Status = LayerDisabled
		layer.Reason = "attach-guard not enabled; age-gating requires it"
		return layer
	}

	count := 0
	for path := range genState.Files {
		if strings.Contains(path, "age-gate") || strings.Contains(path, "age_gate") {
			count++
		}
	}

	if count > 0 {
		layer.Status = LayerEnabled
		layer.Reason = "age-gating configuration files present"
	} else {
		layer.Status = LayerDisabled
		layer.Reason = "no age-gating configuration files found in state"
	}
	return layer
}

func assessInstallScriptBlocking(enabledTools map[string]bool) DefenseLayer {
	layer := DefenseLayer{
		Name:    "install-script-blocking",
		Weight:  WeightHigh,
		MinTier: 1,
	}

	if enabledTools["attach-guard"] {
		layer.Status = LayerEnabled
		layer.Reason = "attach-guard blocks unverified install scripts"
	} else {
		layer.Status = LayerDisabled
		layer.Reason = "attach-guard not enabled"
	}
	return layer
}

func assessLockFileEnforcement(enabledTools map[string]bool, genState types.GeneratedState) DefenseLayer {
	layer := DefenseLayer{
		Name:    "lock-file-enforcement",
		Weight:  WeightHigh,
		MinTier: 1,
	}

	// Check if lock file enforcement is configured through generated state.
	// Look for pre-commit config or lock-related configs.
	hasLockEnforcement := false
	for path := range genState.Files {
		if strings.Contains(path, "lock") || strings.Contains(path, ".pre-commit-config") {
			hasLockEnforcement = true
			break
		}
	}

	if enabledTools["attach-guard"] && hasLockEnforcement {
		layer.Status = LayerEnabled
		layer.Reason = "lock file enforcement configured"
	} else if enabledTools["attach-guard"] || hasLockEnforcement {
		layer.Status = LayerPartial
		layer.Score = 5
		layer.Reason = "partial lock file enforcement"
	} else {
		layer.Status = LayerDisabled
		layer.Reason = "no lock file enforcement configured"
	}
	return layer
}

func assessVulnScanning(enabledTools map[string]bool, genState types.GeneratedState) DefenseLayer {
	layer := DefenseLayer{
		Name:    "vulnerability-scanning",
		Weight:  WeightHigh,
		MinTier: 1,
	}

	// Check for vulnerability scanning configs (grype, socket-dev, etc.)
	hasVulnConfig := false
	for path := range genState.Files {
		if strings.Contains(path, ".grype") || strings.Contains(path, "vuln") {
			hasVulnConfig = true
			break
		}
	}

	if enabledTools["container-security"] || enabledTools["socket-dev-mcp"] || hasVulnConfig {
		layer.Status = LayerEnabled
		layer.Reason = "vulnerability scanning configured"
	} else {
		layer.Status = LayerDisabled
		layer.Reason = "no vulnerability scanning configured"
	}
	return layer
}

func assessNixHardening(genState types.GeneratedState) DefenseLayer {
	layer := DefenseLayer{
		Name:    "nix-hardening",
		Weight:  WeightMedium,
		MinTier: 3,
	}

	// Check if devenv.nix exists in generated state (implies NixHardeningGuide was applied).
	_, hasDevenvNix := genState.Files["devenv.nix"]
	if hasDevenvNix {
		layer.Status = LayerEnabled
		layer.Reason = "devenv.nix present with hardening configuration"
	} else {
		layer.Status = LayerDisabled
		layer.Reason = "devenv.nix not in generated state"
	}
	return layer
}

func assessSAST(enabledTools map[string]bool, genState types.GeneratedState) DefenseLayer {
	layer := DefenseLayer{
		Name:    "sast",
		Weight:  WeightMedium,
		MinTier: 3,
	}

	semgrepEnabled := enabledTools["semgrep"]
	_, hasSemgrepYml := genState.Files[".semgrep.yml"]

	if semgrepEnabled && hasSemgrepYml {
		layer.Status = LayerEnabled
		layer.Reason = "semgrep enabled and .semgrep.yml present"
	} else if semgrepEnabled || hasSemgrepYml {
		layer.Status = LayerPartial
		layer.Score = 5
		if !semgrepEnabled {
			layer.Reason = ".semgrep.yml present but semgrep not enabled"
		} else {
			layer.Reason = "semgrep enabled but .semgrep.yml not in state"
		}
	} else {
		layer.Status = LayerDisabled
		layer.Reason = "semgrep not enabled"
	}
	return layer
}

func assessSecretsScanning(enabledTools map[string]bool) DefenseLayer {
	layer := DefenseLayer{
		Name:    "secrets-scanning",
		Weight:  WeightMedium,
		MinTier: 2,
	}

	gitleaksEnabled := enabledTools["gitleaks"]
	ripsecrets := enabledTools["ripsecrets"]

	if gitleaksEnabled && ripsecrets {
		layer.Status = LayerEnabled
		layer.Score = 10
		layer.Reason = "both gitleaks and ripsecrets enabled"
	} else if gitleaksEnabled || ripsecrets {
		layer.Status = LayerPartial
		layer.Score = 5
		if gitleaksEnabled {
			layer.Reason = "gitleaks enabled; ripsecrets not enabled"
		} else {
			layer.Reason = "ripsecrets enabled; gitleaks not enabled"
		}
	} else {
		layer.Status = LayerDisabled
		layer.Reason = "no secrets scanning enabled"
	}
	return layer
}

func assessContainerSecurity(enabledTools map[string]bool, detected types.DetectedProject) DefenseLayer {
	layer := DefenseLayer{
		Name:    "container-security",
		Weight:  WeightMedium,
		MinTier: 3,
	}

	if !detected.HasDockerfile {
		layer.Status = LayerNotApplicable
		layer.Reason = "no Dockerfile detected"
		return layer
	}

	if enabledTools["container-security"] {
		layer.Status = LayerEnabled
		layer.Reason = "container security scanning enabled"
	} else {
		layer.Status = LayerDisabled
		layer.Reason = "container security not enabled despite Dockerfile present"
	}
	return layer
}

func assessLicenseCompliance(enabledTools map[string]bool) DefenseLayer {
	layer := DefenseLayer{
		Name:    "license-compliance",
		Weight:  WeightLow,
		MinTier: 3,
	}

	if enabledTools["license-compliance"] {
		layer.Status = LayerEnabled
		layer.Reason = "license compliance scanning enabled"
	} else {
		layer.Status = LayerDisabled
		layer.Reason = "license compliance not enabled"
	}
	return layer
}
