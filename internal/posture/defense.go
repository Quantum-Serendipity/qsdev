package posture

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
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

	// securityScanScript is the devenv script that runs the security-scan
	// task (e.g. "qsdev-security-scan").
	securityScanScript = ecosystem.TaskScriptPrefix + ecosystem.SecurityScanTask
	// securityScanExecRe matches the start of the security-scan script's
	// indented-string exec body in devenv.nix. The devenv addon groups the
	// scripts into one `scripts = { "qsdev-security-scan" = { ... exec = ''`
	// attrset; the dotted forms (`scripts."qsdev-security-scan".exec = ''`)
	// are accepted too. The key may be quoted or bare.
	securityScanExecRe = regexp.MustCompile(`(?:^|[\s{;.])(?:"` + regexp.QuoteMeta(securityScanScript) + `"|` +
		regexp.QuoteMeta(securityScanScript) + `)\s*(?:=\s*\{[^}]*?\bexec|\.exec)\s*=\s*''`)
	// semgrepInvocationRe matches a script line that runs semgrep with an
	// explicit rule config.
	semgrepInvocationRe = regexp.MustCompile(`(?m)^\s*semgrep\s(?:.*\s)?--config\s`)
	// scancodePolicyInvocationRe matches a script line that runs a ScanCode
	// license scan with a license policy applied.
	scancodePolicyInvocationRe = regexp.MustCompile(`(?m)^\s*scancode\s(?:.*\s)?--license\s(?:.*\s)?--license-policy\s`)

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
	// claudeSettings returns the effective Claude settings of ProjectPath,
	// read once on first use; nil reads as an empty view.
	claudeSettings func() (claudesettings.Effective, error)
	// guardOnce returns the package guard's state, judged once on first use.
	guardOnce func() guardState
	// userSettingsRead records that claudeSettings includes the user file.
	userSettingsRead bool
	// packageGuard is the generator's package-guard.py (see
	// AssessOptions.PackageGuard).
	packageGuard []byte
}

// guardState is whether the package guard is in force, and why.
type guardState struct {
	Status LayerStatus
	Reason string
}

// settings returns the effective Claude settings of the project.
func (in assessmentInput) settings() (claudesettings.Effective, error) {
	if in.claudeSettings == nil {
		return claudesettings.Read("")
	}
	return in.claudeSettings()
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

// guard returns whether package-guard.py is verifiably in force.
func (in assessmentInput) guard() guardState {
	return in.guardOnce()
}

// judgeGuard judges the package guard from the effective Claude settings, the
// enabled tools and the guard script on disk. The first check that fails
// decides; the guard is in force only when none does. Managed settings are
// never read, user settings only when the assessment was asked to.
func (in assessmentInput) judgeGuard() guardState {
	disabled := func(reason string) guardState { return guardState{Status: LayerDisabled, Reason: reason} }
	settings, err := in.settings()
	switch {
	case err != nil:
		return disabled(fmt.Sprintf("Claude settings unreadable: %v", err))
	case settings.DisableAllHooks:
		return disabled(fmt.Sprintf("hooks disabled (%s in %s)",
			claudesettings.KeyDisableAllHooks, strings.Join(settings.Sources[claudesettings.KeyDisableAllHooks], ", ")))
	case !in.EnabledTools["attach-guard"]:
		return disabled("attach-guard not enabled")
	case !in.has(packageGuardPath):
		return disabled("package-guard.py not present")
	}
	if reason := in.guardModified(); reason != "" {
		return disabled(reason)
	}
	if prefix := settings.Env[claudesettings.EnvShellPrefix]; prefix != "" {
		return disabled(fmt.Sprintf("hook commands are prefixed with %s=%q (set in %s), so the hook may never run package-guard.py",
			claudesettings.EnvShellPrefix, prefix,
			strings.Join(settings.Sources[claudesettings.EnvSourceKey(claudesettings.EnvShellPrefix)], ", ")))
	}
	if !settings.RunsScript(claudesettings.EventPreToolUse, "Bash", packageGuardPath, branding.Get().AppName, claudesettings.GuardHookTimeout) {
		return disabled(fmt.Sprintf("package-guard.py not registered as a blocking PreToolUse hook matching Bash "+
			"(not async, timeout at least %ds) in %s or %s",
			claudesettings.GuardHookTimeout, claudesettings.ProjectRelPath, claudesettings.LocalRelPath))
	}
	unread := "user/managed"
	if in.userSettingsRead {
		unread = "managed"
	}
	return guardState{Status: LayerEnabled, Reason: "package-guard.py unmodified and registered as a PreToolUse hook " +
		"matching Bash (" + unread + " settings not inspected)"}
}

// guardModified returns why package-guard.py on disk cannot be credited as the
// generated version, or "" when it is the content the generator writes (see
// AssessOptions.PackageGuard), line endings aside, with its recorded mode.
// The local state and the committed manifest are not consulted for the
// content: either can be re-hashed along with an edited guard.
func (in assessmentInput) guardModified() string {
	switch {
	case in.ProjectPath == "":
		return "package-guard.py content not inspected"
	case in.packageGuard == nil:
		return "package-guard.py content not verified: no generated version to compare it with"
	}
	fs := state.CheckContent(in.ProjectPath, packageGuardPath, in.packageGuard, in.GenState.Files[packageGuardPath].Mode)
	switch fs.Status {
	case types.Unmodified:
		return ""
	case types.Deleted:
		return "package-guard.py not present"
	case types.Modified:
		return "package-guard.py modified from the generated version this qsdev writes; run " +
			"'qsdev update --configs-only --overwrite-modified' to restore " + packageGuardPath
	default:
		return fmt.Sprintf("package-guard.py unreadable: %v", fs.Error)
	}
}

// guardLayer reports the package guard's state as a layer result.
func guardLayer(input assessmentInput) (LayerStatus, int, string) {
	g := input.guard()
	return g.Status, 0, g.Reason
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

// securityScanRuns reports whether devenv.nix defines the security-scan task
// script and a line of that script matches invocation. A scanner tool on its
// own only installs the binary and writes its config or policy file; scanning
// happens only where the task runs it.
func (in assessmentInput) securityScanRuns(invocation *regexp.Regexp) bool {
	data := in.content(devenvNixPath)
	if data == nil {
		return false
	}
	loc := securityScanExecRe.FindIndex(data)
	if loc == nil {
		return false
	}
	body, ok := nixIndentedStringBody(string(data[loc[1]:]))
	return ok && invocation.MatchString(body)
}

// nixIndentedStringBody returns the raw body of a Nix indented string, which
// is delimited by two single quotes, when s starts right after its opening
// delimiter. It stops at the closing delimiter and skips the escapes formed by
// the delimiter followed by a single quote, a dollar sign or a backslash. ok is
// false when the string is unterminated.
func nixIndentedStringBody(s string) (body string, ok bool) {
	for i := 0; i+1 < len(s); i++ {
		if s[i] != '\'' || s[i+1] != '\'' {
			continue
		}
		if i+2 < len(s) && strings.ContainsRune(`'$\`, rune(s[i+2])) {
			i += 2 // escape sequence: skip it whole
			continue
		}
		return s[:i], true
	}
	return "", false
}

// GuardAgeCheckedLanguages are the ecosystems whose registry publication age
// package-guard.py checks (npm, PyPI, crates.io, the Go module proxy,
// RubyGems, Packagist, NuGet's registration hive and pub.dev; dart/flutter
// installs are currently denied outright, which is stricter still). The
// claudecode addon's tests keep this list in sync with the hook template.
var GuardAgeCheckedLanguages = []string{
	ecosystem.NameJavaScript, ecosystem.NamePython, ecosystem.NameRust, ecosystem.NameGo,
	ecosystem.NameRuby, ecosystem.NamePHP, ecosystem.NameDart, ecosystem.NameDotnet,
}

// ageUngatedLanguages lists the detected ecosystems that install packages (their
// module declares package managers) but whose publication age package-guard.py
// does not check.
func (in assessmentInput) ageUngatedLanguages() []string {
	registry := ecosystem.DefaultRegistry()
	var out []string
	for _, lc := range in.Detected.LanguageChoices() {
		if slices.Contains(GuardAgeCheckedLanguages, lc.Name) || slices.Contains(out, lc.Name) {
			continue
		}
		if mod, ok := registry.ByName(lc.Name); ok && len(mod.PackageManagers()) > 0 {
			out = append(out, lc.Name)
		}
	}
	return out
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
		// The layer is the package guard itself.
		Assess: guardLayer,
	},
	{
		Name:    "age-gating",
		Weight:  WeightHigh,
		MinTier: 2,
		Assess: func(input assessmentInput) (LayerStatus, int, string) {
			// Age-gating is built into package-guard.py (MIN_AGE_DAYS), which
			// checks publication age only for some registries: a project using
			// another package ecosystem is only partly gated.
			if g := input.guard(); g.Status != LayerEnabled {
				return g.Status, 0, g.Reason
			}
			if uncovered := input.ageUngatedLanguages(); len(uncovered) > 0 {
				return LayerPartial, 5, fmt.Sprintf("package-guard.py checks publication age only for %s packages; not for: %s",
					strings.Join(GuardAgeCheckedLanguages, ", "), strings.Join(uncovered, ", "))
			}
			return LayerEnabled, 0, "package-guard.py enforces publication age checks"
		},
	},
	{
		Name:    "install-script-blocking",
		Weight:  WeightHigh,
		MinTier: 1,
		// package-guard.py blocks unverified install scripts.
		Assess: guardLayer,
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
			guard := input.guard()

			switch {
			case guard.Status == LayerEnabled && hasAuditHook:
				return LayerEnabled, 0, "package guard in force and " + lockFileAuditHookID + " hook configured"
			case hasAuditHook:
				return LayerPartial, 5, lockFileAuditHookID + " hook configured but package guard not in force: " + guard.Reason
			case guard.Status == LayerEnabled:
				return LayerPartial, 5, "package guard in force but " + lockFileAuditHookID + " hook not configured"
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
			if input.guard().Status == LayerEnabled {
				return LayerEnabled, 0, "package-guard.py checks package installs against OSV.dev"
			}
			if input.EnabledTools["container-security"] {
				return LayerPartial, 5, "container-security enabled but .grype.yaml not present"
			}
			if input.EnabledTools["socket-dev-mcp"] {
				return LayerPartial, 5, "socket-dev-mcp provides on-demand lookups only; no scanner configured"
			}
			if input.EnabledTools["attach-guard"] {
				return LayerDisabled, 0, "no vulnerability scanning in force: " + input.guard().Reason
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
			// SAST counts only when something runs it: the security-scan
			// task script in devenv.nix must invoke semgrep with its rule
			// packs, and the tool must be enabled so the binary is installed.
			semgrepEnabled := input.EnabledTools["semgrep"]
			wired := input.securityScanRuns(semgrepInvocationRe)

			switch {
			case semgrepEnabled && wired:
				return LayerEnabled, 0, "semgrep enabled and run by the " + securityScanScript + " task"
			case semgrepEnabled:
				return LayerPartial, 5, "semgrep enabled but the " + securityScanScript + " task in devenv.nix does not run it (run qsdev update)"
			case wired:
				return LayerPartial, 5, securityScanScript + " task runs semgrep but semgrep not enabled"
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
			// The policy file alone enforces nothing: licenses are checked
			// only where the security-scan task runs ScanCode with it.
			enabled := input.EnabledTools["license-compliance"]
			wired := input.securityScanRuns(scancodePolicyInvocationRe)

			switch {
			case enabled && wired:
				return LayerEnabled, 0, "license-compliance enabled and its ScanCode policy scan run by the " + securityScanScript + " task"
			case enabled:
				return LayerPartial, 5, "license-compliance enabled but the " + securityScanScript + " task in devenv.nix does not run the ScanCode policy scan (run qsdev update)"
			case wired:
				return LayerPartial, 5, securityScanScript + " task runs a ScanCode policy scan but license-compliance not enabled"
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
//
// opts.ClaudeUserDir also reads the user Claude settings, so a user-level
// disableAllHooks disables the guard layers; opts.PackageGuard is what the
// guard on disk is judged against. opts.FreshScan is not used here.
func AssessDefenseLayers(projectPath string, opts AssessOptions, enabledTools map[string]bool,
	detected types.DetectedProject, genState types.GeneratedState, currentTier int) DefenseCoverage {
	settingsOpts := claudesettings.ReadOptions{UserDir: opts.ClaudeUserDir}
	input := assessmentInput{
		ProjectPath:  projectPath,
		EnabledTools: enabledTools,
		Detected:     detected,
		GenState:     genState,
		claudeSettings: sync.OnceValues(func() (claudesettings.Effective, error) {
			return claudesettings.ReadWith(projectPath, settingsOpts)
		}),
		userSettingsRead: settingsOpts.UserDir != "",
		packageGuard:     opts.PackageGuard,
	}
	input.guardOnce = sync.OnceValue(input.judgeGuard)

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
