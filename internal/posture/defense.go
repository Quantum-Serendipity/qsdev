package posture

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// Artifacts whose presence or content the defense layers inspect.
const (
	packageGuardPath    = ".claude/hooks/package-guard.py"
	preCommitConfigPath = ".pre-commit-config.yaml"
	devenvNixPath       = "devenv.nix"
	grypeConfigPath     = ".grype.yaml"
	semgrepConfigPath   = ".semgrep.yml"

	// lockFileAuditHookID is the id of the catalog's lock file audit custom
	// hook, which devenv.nix registers with git-hooks.nix and which is rendered
	// into .pre-commit-config.yaml.
	lockFileAuditHookID = "lock-file-audit"
)

var (
	// lockAuditNixRe matches an enabled lock-file-audit hook definition in
	// devenv.nix (`lock-file-audit = { enable = true; ...`).
	lockAuditNixRe = regexp.MustCompile(regexp.QuoteMeta(lockFileAuditHookID) + `\s*=\s*\{\s*enable\s*=\s*true\s*;`)
	// lockAuditPreCommitRe matches the lock-file-audit hook id entry in a
	// rendered .pre-commit-config.yaml.
	lockAuditPreCommitRe = regexp.MustCompile(`(?m)^\s*(?:-\s*)?id:\s*["']?` + regexp.QuoteMeta(lockFileAuditHookID) + `["']?\s*$`)

	// nixHardeningSettings are the hardening settings qsdev renders into
	// devenv.nix. A devenv.nix that is merely present, without them, provides
	// no hardening.
	nixHardeningSettings = []struct {
		desc string
		re   *regexp.Regexp
	}{
		{"DEVENV_SECURITY_HARDENED", regexp.MustCompile(`DEVENV_SECURITY_HARDENED\s*=\s*"true"`)},
		{"unsetEnvVars", regexp.MustCompile(`unsetEnvVars\s*=\s*\[\s*"`)},
		{"dotenv disabled", regexp.MustCompile(`dotenv\.enable\s*=\s*false`)},
	}
)

// assessmentInput bundles all inputs needed by layer assessment functions.
type assessmentInput struct {
	// ProjectPath is the project root used to read the content of generated
	// artifacts. Empty disables content inspection (content reads as absent).
	ProjectPath  string
	EnabledTools map[string]bool
	Detected     types.DetectedProject
	// GenState lists the generated files that are present on disk.
	GenState types.GeneratedState
}

// has reports whether the generated file at rel is present.
func (in assessmentInput) has(rel string) bool {
	_, ok := in.GenState.Files[rel]
	return ok
}

// content returns the content of the present generated file at rel, or nil
// when it is not present or cannot be read.
func (in assessmentInput) content(rel string) []byte {
	if in.ProjectPath == "" || !in.has(rel) {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(in.ProjectPath, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	return data
}

// hasLockFileAuditHook reports whether the lock file audit hook is configured
// in devenv.nix or the rendered pre-commit config.
func (in assessmentInput) hasLockFileAuditHook() bool {
	if data := in.content(devenvNixPath); data != nil && lockAuditNixRe.Match(data) {
		return true
	}
	if data := in.content(preCommitConfigPath); data != nil && lockAuditPreCommitRe.Match(data) {
		return true
	}
	return false
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
			hasPackageGuard := input.has(packageGuardPath)

			if attachGuardEnabled && hasPackageGuard {
				return LayerEnabled, 0, "attach-guard enabled and package-guard.py present"
			}
			if attachGuardEnabled || hasPackageGuard {
				if !attachGuardEnabled {
					return LayerPartial, 5, "package-guard.py present but attach-guard not enabled"
				}
				return LayerPartial, 5, "attach-guard enabled but package-guard.py not present"
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
			if input.has(packageGuardPath) {
				return LayerEnabled, 0, "package-guard.py enforces publication age checks"
			}
			return LayerDisabled, 0, "package-guard.py not present"
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
			// Lock file enforcement is the lock file audit pre-commit hook plus
			// the package guard, which checks installs against the lockfile. It
			// is judged from the hook's actual configuration, not from file
			// names that merely contain "lock".
			hasAuditHook := input.hasLockFileAuditHook()
			attachGuard := input.EnabledTools["attach-guard"]

			if attachGuard && hasAuditHook {
				return LayerEnabled, 0, "attach-guard enabled and " + lockFileAuditHookID + " hook configured"
			}
			if hasAuditHook {
				return LayerPartial, 5, lockFileAuditHookID + " hook configured but attach-guard not enabled"
			}
			if attachGuard {
				return LayerPartial, 5, "attach-guard enabled but " + lockFileAuditHookID + " hook not configured"
			}
			return LayerDisabled, 0, "no lock file enforcement configured"
		},
	},
	{
		Name:    "vulnerability-scanning",
		Weight:  WeightHigh,
		MinTier: 1,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			// Only a configured scanner counts as enabled: Grype via
			// container-security, or package-guard.py checking every package
			// install against OSV.dev. The Socket MCP server only offers the
			// agent on-demand lookups, so on its own it is partial.
			if input.EnabledTools["container-security"] && input.has(grypeConfigPath) {
				return LayerEnabled, 0, "container-security enabled and .grype.yaml present"
			}
			if input.EnabledTools["attach-guard"] && input.has(packageGuardPath) {
				return LayerEnabled, 0, "package-guard.py checks package installs against OSV.dev"
			}
			if input.EnabledTools["container-security"] {
				return LayerPartial, 5, "container-security enabled but .grype.yaml not present"
			}
			if input.EnabledTools["socket-dev-mcp"] {
				return LayerPartial, 5, "socket-dev-mcp provides on-demand lookups only; no scanner configured"
			}
			return LayerDisabled, 0, "no vulnerability scanning configured"
		},
	},
	{
		Name:    "nix-hardening",
		Weight:  WeightMedium,
		MinTier: 3,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			// devenv.nix must actually carry the hardening settings; its mere
			// presence says nothing about the environment's configuration.
			if !input.has(devenvNixPath) {
				return LayerDisabled, 0, "devenv.nix not present"
			}
			data := input.content(devenvNixPath)
			var missing []string
			for _, setting := range nixHardeningSettings {
				if data == nil || !setting.re.Match(data) {
					missing = append(missing, setting.desc)
				}
			}
			switch len(missing) {
			case 0:
				return LayerEnabled, 0, "devenv.nix present with hardening configuration"
			case len(nixHardeningSettings):
				return LayerDisabled, 0, "devenv.nix present but has no hardening configuration"
			default:
				return LayerPartial, 5, "devenv.nix missing hardening settings: " + strings.Join(missing, ", ")
			}
		},
	},
	{
		Name:    "sast",
		Weight:  WeightMedium,
		MinTier: 3,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			semgrepEnabled := input.EnabledTools["semgrep"]
			hasSemgrepYml := input.has(semgrepConfigPath)

			if semgrepEnabled && hasSemgrepYml {
				return LayerEnabled, 0, "semgrep enabled and .semgrep.yml present"
			}
			if semgrepEnabled || hasSemgrepYml {
				if !semgrepEnabled {
					return LayerPartial, 5, ".semgrep.yml present but semgrep not enabled"
				}
				return LayerPartial, 5, "semgrep enabled but .semgrep.yml not present"
			}
			return LayerDisabled, 0, "semgrep not enabled"
		},
	},
	{
		Name:    "secrets-scanning",
		Weight:  WeightMedium,
		MinTier: 2,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			// A scanner counts when it is enabled or when the pre-commit config
			// runs it as a hook (Assess folds those hook ids into EnabledTools);
			// the config file merely existing credits nothing.
			gitleaksEnabled := input.EnabledTools["gitleaks"]
			ripsecrets := input.EnabledTools["ripsecrets"]

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

// AssessDefenseLayers evaluates all 10 defense layers. genState must list only
// the generated files that are present on disk; projectPath is the project
// root used to inspect their content (empty disables content inspection).
//
// Enabled and Total count only the layers in scope at currentTier, matching
// the tier-relative Score, so the "N/M layers" summary never contradicts it.
func AssessDefenseLayers(projectPath string, enabledTools map[string]bool, detected types.DetectedProject, genState types.GeneratedState, currentTier int) DefenseCoverage {
	input := assessmentInput{
		ProjectPath:  projectPath,
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
		if l.MinTier > currentTier || l.Status == LayerNotApplicable {
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
