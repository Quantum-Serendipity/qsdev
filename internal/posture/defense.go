package posture

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// DefenseLayerNames lists the canonical names of all 10 defense layers,
// derived from layerTable in init() to prevent drift.
var DefenseLayerNames [10]string

func init() {
	if len(layerTable) != len(DefenseLayerNames) {
		panic(fmt.Sprintf("posture: layerTable has %d entries but DefenseLayerNames expects %d",
			len(layerTable), len(DefenseLayerNames)))
	}
	for i, spec := range layerTable {
		DefenseLayerNames[i] = spec.Name
	}
}

// assessmentInput bundles all inputs needed by layer assessment functions.
type assessmentInput struct {
	EnabledTools map[string]bool
	Detected     types.DetectedProject
	GenState     types.GeneratedState
}

// layerSpec defines one defense layer's metadata and assessment logic.
type layerSpec struct {
	Name    string
	Weight  LayerWeight
	MinTier int
	Assess  func(input assessmentInput) (status LayerStatus, score int, reason string)
}

// layerTable is the data-driven table of all 10 defense layers.
var layerTable = []layerSpec{
	{
		Name:    "pretooluse-hooks",
		Weight:  WeightCritical,
		MinTier: 1,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			attachGuardEnabled := input.EnabledTools["attach-guard"]
			_, hasPackageGuard := input.GenState.Files[".claude/hooks/package-guard.py"]

			if attachGuardEnabled && hasPackageGuard {
				return LayerEnabled, 0, "attach-guard enabled and package-guard.py present"
			}
			if attachGuardEnabled || hasPackageGuard {
				if !attachGuardEnabled {
					return LayerPartial, 5, "package-guard.py present but attach-guard not enabled"
				}
				return LayerPartial, 5, "attach-guard enabled but package-guard.py not in state"
			}
			return LayerDisabled, 0, "attach-guard not enabled"
		},
	},
	{
		Name:    "age-gating",
		Weight:  WeightHigh,
		MinTier: 2,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			if !input.EnabledTools["attach-guard"] {
				return LayerDisabled, 0, "attach-guard not enabled; age-gating requires it"
			}
			// Age-gating is built into package-guard.py (MIN_AGE_DAYS). When the
			// guard script is present and attach-guard is enabled, age-gating is active.
			_, hasPackageGuard := input.GenState.Files[".claude/hooks/package-guard.py"]
			if hasPackageGuard {
				return LayerEnabled, 0, "package-guard.py enforces publication age checks"
			}
			return LayerDisabled, 0, "package-guard.py not found in generated state"
		},
	},
	{
		Name:    "install-script-blocking",
		Weight:  WeightHigh,
		MinTier: 1,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			if input.EnabledTools["attach-guard"] {
				return LayerEnabled, 0, "attach-guard blocks unverified install scripts"
			}
			return LayerDisabled, 0, "attach-guard not enabled"
		},
	},
	{
		Name:    "lock-file-enforcement",
		Weight:  WeightHigh,
		MinTier: 1,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			// Check if lock file enforcement is configured through generated state.
			// Look for pre-commit config or lock-related configs.
			hasLockEnforcement := false
			for path := range input.GenState.Files {
				if strings.Contains(path, "lock") || strings.Contains(path, ".pre-commit-config") {
					hasLockEnforcement = true
					break
				}
			}

			if input.EnabledTools["attach-guard"] && hasLockEnforcement {
				return LayerEnabled, 0, "lock file enforcement configured"
			}
			if input.EnabledTools["attach-guard"] || hasLockEnforcement {
				return LayerPartial, 5, "partial lock file enforcement"
			}
			return LayerDisabled, 0, "no lock file enforcement configured"
		},
	},
	{
		Name:    "vulnerability-scanning",
		Weight:  WeightHigh,
		MinTier: 1,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			// Check for vulnerability scanning configs (grype, socket-dev, etc.)
			hasVulnConfig := false
			for path := range input.GenState.Files {
				if strings.Contains(path, ".grype") || strings.Contains(path, "vuln") {
					hasVulnConfig = true
					break
				}
			}

			if input.EnabledTools["container-security"] || input.EnabledTools["socket-dev-mcp"] || hasVulnConfig {
				return LayerEnabled, 0, "vulnerability scanning configured"
			}
			return LayerDisabled, 0, "no vulnerability scanning configured"
		},
	},
	{
		Name:    "nix-hardening",
		Weight:  WeightMedium,
		MinTier: 3,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			// Check if devenv.nix exists in generated state (implies NixHardeningGuide was applied).
			_, hasDevenvNix := input.GenState.Files["devenv.nix"]
			if hasDevenvNix {
				return LayerEnabled, 0, "devenv.nix present with hardening configuration"
			}
			return LayerDisabled, 0, "devenv.nix not in generated state"
		},
	},
	{
		Name:    "sast",
		Weight:  WeightMedium,
		MinTier: 3,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			semgrepEnabled := input.EnabledTools["semgrep"]
			_, hasSemgrepYml := input.GenState.Files[".semgrep.yml"]

			if semgrepEnabled && hasSemgrepYml {
				return LayerEnabled, 0, "semgrep enabled and .semgrep.yml present"
			}
			if semgrepEnabled || hasSemgrepYml {
				if !semgrepEnabled {
					return LayerPartial, 5, ".semgrep.yml present but semgrep not enabled"
				}
				return LayerPartial, 5, "semgrep enabled but .semgrep.yml not in state"
			}
			return LayerDisabled, 0, "semgrep not enabled"
		},
	},
	{
		Name:    "secrets-scanning",
		Weight:  WeightMedium,
		MinTier: 2,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			gitleaksEnabled := input.EnabledTools["gitleaks"]
			ripsecrets := input.EnabledTools["ripsecrets"]
			if !ripsecrets {
				_, hasPreCommit := input.GenState.Files[".pre-commit-config.yaml"]
				ripsecrets = hasPreCommit
			}

			if gitleaksEnabled && ripsecrets {
				return LayerEnabled, 10, "both gitleaks and ripsecrets enabled"
			}
			if gitleaksEnabled || ripsecrets {
				if gitleaksEnabled {
					return LayerPartial, 5, "gitleaks enabled; ripsecrets not enabled"
				}
				return LayerPartial, 5, "ripsecrets enabled; gitleaks not enabled"
			}
			return LayerDisabled, 0, "no secrets scanning enabled"
		},
	},
	{
		Name:    "container-security",
		Weight:  WeightMedium,
		MinTier: 3,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			if !input.Detected.HasDockerfile {
				return LayerNotApplicable, 0, "no Dockerfile detected"
			}
			if input.EnabledTools["container-security"] {
				return LayerEnabled, 0, "container security scanning enabled"
			}
			return LayerDisabled, 0, "container security not enabled despite Dockerfile present"
		},
	},
	{
		Name:    "license-compliance",
		Weight:  WeightLow,
		MinTier: 3,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			if input.EnabledTools["license-compliance"] {
				return LayerEnabled, 0, "license compliance scanning enabled"
			}
			return LayerDisabled, 0, "license compliance not enabled"
		},
	},
}

// assessLayer evaluates a single defense layer from its spec and input.
func assessLayer(spec layerSpec, input assessmentInput) DefenseLayer {
	status, score, reason := spec.Assess(input)
	return DefenseLayer{
		Name:    spec.Name,
		Weight:  spec.Weight,
		MinTier: spec.MinTier,
		Status:  status,
		Score:   score,
		Reason:  reason,
	}
}

// AssessDefenseLayers evaluates all 10 defense layers.
func AssessDefenseLayers(enabledTools map[string]bool, detected types.DetectedProject, genState types.GeneratedState, currentTier int) DefenseCoverage {
	input := assessmentInput{
		EnabledTools: enabledTools,
		Detected:     detected,
		GenState:     genState,
	}

	layers := make([]DefenseLayer, len(layerTable))
	for i, spec := range layerTable {
		layers[i] = assessLayer(spec, input)
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
