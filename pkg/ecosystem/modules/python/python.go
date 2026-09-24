// Package python implements the Python ecosystem module for qsdev.
// It detects Python projects by scanning for pyproject.toml, requirements files,
// setup.py, setup.cfg, Pipfile and conda environment files, generates devenv.nix fragments with package manager integration, and
// provides security-hardened pip.conf, pre-commit hooks, CI commands, deny rules,
// and wizard fields for the Python toolchain.
package python

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)
var _ ecosystem.SASTModule = (*Module)(nil)
var _ ecosystem.DevenvYamlInputProvider = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)
var _ ecosystem.SetupWarner = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// pythonVersionRe is the accepted form of a Python version: major.minor with
// an optional patch component, as understood by nixpkgs-python. Versions are
// read from repo-controlled files and interpolated into devenv.nix, so any
// other text is rejected rather than escaped.
var pythonVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+(\.[0-9]+)?$`)

// defaultPythonVersion is used when no Python version is configured.
const defaultPythonVersion = "3.12"

// Module implements ecosystem.EcosystemModule for the Python programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return ecosystem.NamePython }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Python" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for Python ecosystem indicators and returns a DetectionResult.
// pyproject.toml is Certain; setup.py, setup.cfg, Pipfile, requirements*.txt,
// requirements*.in and conda environment files are Probable. The package
// manager is inferred from lockfiles (uv.lock -> uv, poetry.lock -> poetry,
// otherwise pip); projects managed by a tool qsdev does not support (pdm,
// pipenv, hatch, conda) are reported through Extras and evidence instead of
// being called pip, and SetupWarnings repeats that as an init warning. The
// version comes from .python-version, else from pyproject.toml's
// requires-python or Poetry python constraint.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	confidence, evidence := detectIndicators(projectRoot)
	if confidence == ecosystem.ConfidenceAbsent {
		return ecosystem.DetectionAbsent()
	}

	pp := readPyproject(projectRoot)
	extras := map[string]string{}
	pm, pmEvidence := detectPackageManager(projectRoot, pp, extras)
	version, versionEvidence := detectVersion(projectRoot, pp)
	evidence = append(evidence, pmEvidence...)
	evidence = append(evidence, versionEvidence...)
	evidence = append(evidence, detectToolConfig(projectRoot, pp, extras)...)
	if len(extras) == 0 {
		extras = nil
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Version:        version,
			PackageManager: pm,
			Extras:         extras,
		},
	}
}

// Extras keys recorded by Detect and read by the generators.
const (
	// extraProjectManager names an unsupported tool (pdm, pipenv, hatch,
	// conda) that manages the project's dependencies.
	extraProjectManager = "project_manager"
	// extraMypy is "true" when the project configures mypy.
	extraMypy = "mypy"
	// extraBanditConfig is the config file bandit must read (-c), if any.
	extraBanditConfig = "bandit_config"
	// extraUVExcludeNewer is "project" when the project sets its own uv
	// exclude-newer, which qsdev's cooldown must not override.
	extraUVExcludeNewer = "uv_exclude_newer"
)

// detectionIndicators are the file-name patterns that mark a Python project.
var detectionIndicators = []struct {
	pattern    string
	confidence ecosystem.Confidence
}{
	{"pyproject.toml", ecosystem.ConfidenceCertain},
	{"requirements*.txt", ecosystem.ConfidenceProbable},
	{"requirements*.in", ecosystem.ConfidenceProbable},
	{"setup.py", ecosystem.ConfidenceProbable},
	{"setup.cfg", ecosystem.ConfidenceProbable},
	{"Pipfile", ecosystem.ConfidenceProbable},
	{"environment.yml", ecosystem.ConfidenceProbable},
	{"environment.yaml", ecosystem.ConfidenceProbable},
}

// detectIndicators returns the highest confidence among the indicator files
// present in projectRoot, with one evidence entry per file found.
func detectIndicators(projectRoot string) (ecosystem.Confidence, []string) {
	entries, err := os.ReadDir(projectRoot)
	if err != nil {
		return ecosystem.ConfidenceAbsent, nil
	}
	confidence := ecosystem.ConfidenceAbsent
	var evidence []string
	for _, ind := range detectionIndicators {
		for _, e := range entries {
			if ok, _ := filepath.Match(ind.pattern, e.Name()); !ok || !fileutil.FileExists(projectRoot, e.Name()) {
				continue
			}
			confidence = max(confidence, ind.confidence)
			evidence = append(evidence, e.Name()+" found")
		}
	}
	return confidence, evidence
}

// unsupportedManager describes a Python project manager qsdev cannot
// configure: its markers, and the manifest and lockfile it maintains.
type unsupportedManager struct {
	name     string
	files    []string   // any of these files marks the manager
	tool     [][]string // any of these pyproject.toml [tool.*] tables marks it
	manifest ecosystem.ManifestFileInfo
}

// unsupportedManagers are checked in order when no uv or poetry lockfile is
// present. Build-backend tables ([tool.pdm.build], [tool.hatch.build]) are
// deliberately not markers: pip and uv projects use those backends too.
var unsupportedManagers = []unsupportedManager{
	{
		name: "pdm", files: []string{"pdm.lock"},
		tool: [][]string{{"pdm", "dev-dependencies"}, {"pdm", "scripts"}, {"pdm", "resolution"}, {"pdm", "source"}},
		manifest: ecosystem.ManifestFileInfo{Path: "pyproject.toml", Ecosystem: "pdm", VSSupported: true,
			LockFile: "pdm.lock", LockFilePolicy: ecosystem.LockFilePolicyRequired},
	},
	{
		name: "pipenv", files: []string{"Pipfile.lock", "Pipfile"},
		manifest: ecosystem.ManifestFileInfo{Path: "Pipfile", Ecosystem: "pipenv",
			LockFile: "Pipfile.lock", LockFilePolicy: ecosystem.LockFilePolicyRequired},
	},
	{
		name: "hatch", tool: [][]string{{"hatch", "envs"}, {"hatch", "env"}},
		manifest: ecosystem.ManifestFileInfo{Path: "pyproject.toml", Ecosystem: "hatch", VSSupported: true,
			LockFilePolicy: ecosystem.LockFilePolicyNone},
	},
	{
		name: "conda", files: []string{"environment.yml", "environment.yaml"},
		manifest: ecosystem.ManifestFileInfo{Path: "environment.yml", Ecosystem: "conda",
			LockFilePolicy: ecosystem.LockFilePolicyNone},
	},
}

// detectPackageManager infers the package manager from lockfiles. A project
// managed by an unsupported tool gets no package manager suggestion; the tool
// is recorded in extras[extraProjectManager] instead.
func detectPackageManager(projectRoot string, pp pyproject, extras map[string]string) (string, []string) {
	switch {
	case fileutil.FileExists(projectRoot, "uv.lock"):
		return "uv", []string{"uv.lock found"}
	case fileutil.FileExists(projectRoot, "poetry.lock"):
		return "poetry", []string{"poetry.lock found"}
	}
	if um, marker, ok := findUnsupportedManager(projectRoot, pp); ok {
		extras[extraProjectManager] = um.name
		return "", []string{unsupportedManagerMessage(um, marker)}
	}
	return "pip", nil
}

// findUnsupportedManager returns the first unsupported project manager whose
// marker file or pyproject.toml [tool.*] table is present in projectRoot,
// with a description of the marker that identified it.
func findUnsupportedManager(projectRoot string, pp pyproject) (unsupportedManager, string, bool) {
	for _, um := range unsupportedManagers {
		for _, f := range um.files {
			if fileutil.FileExists(projectRoot, f) {
				return um, f + " found", true
			}
		}
		for _, key := range um.tool {
			if pp.hasTool(key...) {
				return um, "[tool." + strings.Join(key, ".") + "] in pyproject.toml", true
			}
		}
	}
	return unsupportedManager{}, "", false
}

// unsupportedManagerMessage explains that the project is managed by um, which
// qsdev cannot configure. The supported managers are listed from
// PackageManagers so the message cannot drift from the catalog.
func unsupportedManagerMessage(um unsupportedManager, marker string) string {
	pms := (&Module{}).PackageManagers()
	supported := make([]string, 0, len(pms))
	for _, pm := range pms {
		supported = append(supported, pm.Name)
	}
	return fmt.Sprintf("%s: %s is not a supported package manager (supported: %s); only pip hardening is generated",
		marker, um.name, strings.Join(supported, ", "))
}

// detectVersion returns the Python version to pin: a valid .python-version
// entry, else the version chosen from the project's version constraint.
// Unusable .python-version values (pypy3.10, 3.13t, system, hostile text) are
// reported and ignored rather than rendered into devenv.nix.
func detectVersion(projectRoot string, pp pyproject) (string, []string) {
	var evidence []string
	if v, raw := readPythonVersionFile(filepath.Join(projectRoot, ".python-version")); v != "" {
		if pythonVersionRe.MatchString(v) {
			return v, []string{fmt.Sprintf("python version %s (from .python-version)", v)}
		}
		evidence = append(evidence, fmt.Sprintf("ignored .python-version entry %q: not a CPython MAJOR.MINOR[.PATCH] version", raw))
	}
	spec, source := pp.requiresPython()
	if spec == "" {
		return "", evidence
	}
	if v := versionForSpecifier(spec); v != "" {
		return v, append(evidence, fmt.Sprintf("python version %s (from %s %q)", v, source, spec))
	}
	return "", append(evidence, fmt.Sprintf("no supported python version satisfies %s %q", source, spec))
}

// detectToolConfig records which Python tools the project configures itself,
// so generated hooks only run tools the project has opted into and read the
// project's own settings.
func detectToolConfig(projectRoot string, pp pyproject, extras map[string]string) []string {
	var evidence []string
	switch {
	case pp.hasTool("mypy"):
		evidence = append(evidence, "mypy configured in pyproject.toml")
	case fileutil.FileExists(projectRoot, "mypy.ini"), fileutil.FileExists(projectRoot, ".mypy.ini"):
		evidence = append(evidence, "mypy configuration file found")
	case iniHasSection(filepath.Join(projectRoot, "setup.cfg"), "mypy"):
		evidence = append(evidence, "mypy configured in setup.cfg")
	}
	if len(evidence) > 0 {
		extras[extraMypy] = "true"
	}
	if pp.hasTool("bandit") {
		extras[extraBanditConfig] = "pyproject.toml"
		evidence = append(evidence, "bandit configured in pyproject.toml")
	}
	if pp.hasTool("uv", "exclude-newer") || tomlDefines(filepath.Join(projectRoot, "uv.toml"), "exclude-newer") {
		extras[extraUVExcludeNewer] = "project"
		evidence = append(evidence, "uv exclude-newer configured by the project")
	}
	return evidence
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Python language support with the configured package manager. It returns
// an error when the configured version is not a plain Python version number.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	version := config.Version
	if version == "" {
		version = defaultPythonVersion
	}
	if !pythonVersionRe.MatchString(version) {
		return "", fmt.Errorf("invalid python version %q: want MAJOR.MINOR or MAJOR.MINOR.PATCH", version)
	}

	props := []ecosystem.NixProperty{
		{Key: "version", Value: ecosystem.NixString(version)},
	}
	var envVars []ecosystem.NixEnvVar
	var extraBlocks []string

	switch pm := config.PM("pip"); pm {
	case "poetry":
		// devenv's poetry task activates poetry's own .venv. venv.enable
		// must stay off: its task runs after poetry's and would activate a
		// second, empty virtualenv over it. The task (and so activation)
		// only exists while install.enable is true, and installing is only
		// safe from a lockfile that matches pyproject.toml: without one,
		// `poetry install` would resolve fresh on every shell entry. The
		// lock check task enforces that when the shell loads rather than
		// through builtins.pathExists, whose result devenv's evaluation
		// cache keeps after poetry.lock is created.
		props = append(props,
			ecosystem.NixProperty{Key: "poetry.enable", Value: "true"},
			ecosystem.NixProperty{Key: "poetry.install.enable", Value: "true"},
			ecosystem.NixProperty{Key: "poetry.activate.enable", Value: "true"},
		)
		extraBlocks = append(extraBlocks, poetryCheckLockTask())
		envVars = append(envVars, ecosystem.NixEnvVar{
			Key:     "POETRY_INSTALLER_ONLY_BINARY",
			Value:   ecosystem.NixString(":all:"),
			Comment: "Refuse source distributions, whose builds run arbitrary code (Poetry >= 2.0; Poetry has no release-age cooldown)",
		})
	case "uv":
		props = append(props,
			ecosystem.NixProperty{Key: "uv.enable", Value: "true"},
			ecosystem.NixProperty{Key: "venv.enable", Value: "true"},
		)
		if config.Extra(extraUVExcludeNewer, "") != "project" {
			envVars = append(envVars, ecosystem.NixEnvVar{
				Key:     "UV_EXCLUDE_NEWER",
				Value:   ecosystem.NixString(uvCooldown),
				Comment: "Ignore releases newer than 7 days when resolving (uv >= 0.9.17)",
			})
		}
	default:
		props = append(props, ecosystem.NixProperty{Key: "venv.enable", Value: "true"})
		// pip only reads configuration from user, site, virtualenv or
		// PIP_CONFIG_FILE locations, never from the project root, so the
		// generated pip.conf must be wired in explicitly for its settings to
		// take effect. The value is a Nix antiquotation of the project root.
		if pm == "pip" {
			envVars = append(envVars, ecosystem.NixEnvVar{
				Key:     "PIP_CONFIG_FILE",
				Value:   `"${config.devenv.root}/` + pipConfigPath + `"`,
				Comment: "Apply the qsdev-managed pip.conf (binary-only installs from the index)",
			})
		}
	}

	return ecosystem.BuildLanguageFragment(ecosystem.NixLangConfig{
		EnablePath:  "languages.python",
		Properties:  props,
		EnvVars:     envVars,
		ExtraBlocks: extraBlocks,
	}), nil
}

// poetryDevenvTask is the devenv task that runs `poetry install` and
// activates poetry's .venv on shell entry. devenv defines it only while
// languages.python.poetry.install.enable is true.
const poetryDevenvTask = "devenv:python:poetry"

// poetryCheckLockTask returns a devenv task that runs before
// poetryDevenvTask, and exists exactly when that task does. It fails when
// pyproject.toml or poetry.lock is missing or `poetry check --lock` rejects
// the lockfile, so devenv skips the install and activation (their dependency
// failed) and prints why, instead of letting poetry resolve dependencies
// afresh or act on a lock that no longer matches pyproject.toml. The shell
// itself still loads, so `poetry lock` can be run from it.
func poetryCheckLockTask() string {
	return `  # Only install from a poetry.lock that matches pyproject.toml.
  tasks."` + branding.Get().AppName + `:python:poetry-check-lock" = lib.mkIf config.languages.python.poetry.install.enable {
    description = "Check that poetry.lock exists and matches pyproject.toml";
    exec = ''
      if [ ! -f pyproject.toml ]; then
        echo "pyproject.toml not found; skipping poetry install. Run 'poetry init' to create it." >&2
        exit 1
      fi
      if [ ! -f poetry.lock ]; then
        echo "poetry.lock not found; skipping poetry install. Run 'poetry lock', commit poetry.lock and reload the shell." >&2
        exit 1
      fi
      if ! ${config.languages.python.poetry.package}/bin/poetry check --lock --no-interaction; then
        echo "poetry check --lock failed (see above): poetry.lock is out of date with pyproject.toml or pyproject.toml is invalid; skipping poetry install. Run 'poetry lock', commit poetry.lock and reload the shell." >&2
        exit 1
      fi
    '';
    cwd = config.devenv.root;
    before = [ "` + poetryDevenvTask + `" ];
  };
`
}

// SetupWarnings reports project files that the configured package manager
// does not match. A poetry-mode project is warned about missing Poetry files:
// the generated shell installs dependencies and activates poetry's .venv only
// once both pyproject.toml (absent when a requirements.txt project picks
// poetry) and poetry.lock exist (see poetryCheckLockTask). A pip-mode project
// that pdm, pipenv, hatch or conda manages is warned that qsdev cannot
// configure that tool, so the gap is visible rather than the project being
// silently treated as a pip project. uv needs no check.
func (m *Module) SetupWarnings(projectRoot string, config ecosystem.ModuleConfig) []string {
	switch config.PM("pip") {
	case "poetry":
		return poetrySetupWarnings(projectRoot)
	case "pip":
		if um, marker, ok := findUnsupportedManager(projectRoot, readPyproject(projectRoot)); ok {
			return []string{unsupportedManagerMessage(um, marker) +
				", and qsdev does not configure " + um.name + " itself; " +
				"choose a supported package manager with --python-pkg-mgr if the project can use one"}
		}
	}
	return nil
}

// poetrySetupWarnings reports the Poetry files a poetry-mode project lacks.
func poetrySetupWarnings(projectRoot string) []string {
	switch {
	case !fileutil.FileExists(projectRoot, "pyproject.toml"):
		return []string{"poetry is the package manager but pyproject.toml is missing, so the devenv shell cannot set up poetry's .venv; " +
			"run 'poetry init' and 'poetry lock', or choose another package manager with --python-pkg-mgr"}
	case !fileutil.FileExists(projectRoot, "poetry.lock"):
		return []string{"poetry.lock is missing, so the devenv shell will not install dependencies or activate poetry's .venv; " +
			"run 'poetry lock', commit poetry.lock and reload the shell"}
	}
	return nil
}

// pipConfigPath is the project-relative path of the generated pip.conf.
const pipConfigPath = "pip.conf"

// pipLockedInstallCommand installs a pip project's pinned, hashed
// requirements. It is the only place hash checking is required: pip.conf
// cannot set require-hashes globally (see SecurityConfigs).
const pipLockedInstallCommand = "pip install --require-hashes --only-binary :all: -r requirements.txt"

// uvCooldown is the uv exclude-newer duration (ISO 8601, 7 days) applied to
// uv projects that do not set their own. The same value must be used
// everywhere: uv records it in uv.lock and treats a different setting as a
// stale lockfile.
const uvCooldown = "P7D"

// DevenvYamlInputs returns the extra flake input required for Python.
//
// DevenvNixFragment always pins languages.python.version (defaulting to "3.12"
// when unset), and devenv >=2.1 refuses to evaluate languages.python.version
// unless the nixpkgs-python input is present. The input is therefore returned
// unconditionally whenever the Python module is active — the invariant
// "input present ⟺ languages.python.version emitted" must not depend on whether
// the user pinned a version.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return []ecosystem.DevenvInput{
		{URL: "github:cachix/nixpkgs-python", Follows: "nixpkgs"},
	}
}

// SecurityConfigs returns generated security configuration files.
// For pip, it generates a security-hardened pip.conf. uv and poetry are
// hardened through environment variables in the devenv.nix fragment
// (UV_EXCLUDE_NEWER, POETRY_INSTALLER_ONLY_BINARY): uv.toml would shadow a
// project's [tool.uv] table, and poetry.toml is user-owned.
//
// The pip.conf applies to every pip run in the devenv shell (it is wired in
// through PIP_CONFIG_FILE), so it holds only settings that are safe for all
// of them. only-binary = :all: is: pip applies it to packages it resolves
// from an index, never to a local project directory, so `pip install -e .`
// still builds the project itself. require-hashes is not: hash-checking mode
// rejects the unpinned `pip install --upgrade pip` that devenv's virtualenv
// task runs (venv --upgrade-deps) and every local-directory or editable
// install. Hash checking is instead applied to the locked install, where the
// requirements are pinned and hashed: see CICommands.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	pm := config.PM("pip")

	if pm != "pip" {
		return nil
	}

	var b strings.Builder
	b.WriteString("# Security-hardened pip configuration\n")
	b.WriteString("# " + branding.GeneratedBy() + "\n")
	b.WriteString("# Note: age-gating via uploaded-prior-to requires pip >= 26.0 (Jan 2026)\n")
	b.WriteString("#\n")
	b.WriteString("# only-binary: Blocks source distributions from the index, whose builds run\n")
	b.WriteString("#   arbitrary code. The local project itself (pip install -e .) still builds.\n")
	b.WriteString("# Hash checking is not set here: it would break the virtualenv's pip upgrade\n")
	b.WriteString("#   and editable installs. Install locked dependencies with:\n")
	b.WriteString("#   " + pipLockedInstallCommand + "\n")
	b.WriteString("\n")
	b.WriteString("[global]\n")
	if config.RegistryProxy != "" {
		fmt.Fprintf(&b, "index-url = %s\n", ecosystem.INIEscapeValue(config.RegistryProxy))
	}
	b.WriteString("only-binary = :all:\n")
	content := b.String()

	return []types.GeneratedFile{
		{
			Path:     pipConfigPath,
			Content:  []byte(content),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Skip,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for the Python ecosystem.
// mypy runs only for projects that configure it: the git-hooks.nix hook uses
// its own interpreter, cannot see the project's venv, and so fails on every
// third-party import unless the project has set mypy up for that. bandit
// reads the project's [tool.bandit] settings when present and otherwise
// reports only medium and high severity issues, skipping assert_used (B101),
// which fires on every pytest assert.
func (m *Module) PreCommitHooks(config ecosystem.ModuleConfig) []ecosystem.HookConfig {
	hooks := []ecosystem.HookConfig{
		{
			ID:            "ruff",
			Name:          "ruff",
			Description:   "Run ruff linter and formatter for Python",
			Entry:         "ruff check --fix",
			Language:      "python",
			Types:         []string{"python"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
	}
	if config.Extra(extraMypy, "") == "true" {
		hooks = append(hooks, ecosystem.HookConfig{
			ID:            "mypy",
			Name:          "mypy",
			Description:   "Run mypy type checker for Python",
			Entry:         "mypy",
			Language:      "python",
			Types:         []string{"python"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		})
	}

	banditArgs := "-ll --skip B101"
	if cfg := config.Extra(extraBanditConfig, ""); cfg == "pyproject.toml" {
		banditArgs = "-c " + cfg
	}
	return append(hooks, ecosystem.HookConfig{
		ID:          "bandit",
		Name:        "bandit",
		Description: "Run bandit security SAST scanner for Python",
		Entry:       "bandit " + banditArgs + " -r",
		// Language "system" runs the nix-provided binary directly; "python"
		// would make git-hooks.nix build a redundant venv for the hook. Match
		// the staticcheck/govulncheck reference for NixPackage-backed hooks.
		Language:      "system",
		Types:         []string{"python"},
		Stages:        []string{"pre-commit"},
		PassFilenames: true,
		// git-hooks.nix has no built-in "bandit" hook; render it as a
		// custom hook so an `entry` is always emitted (via NixPackage).
		BuiltIn:    false,
		NixPackage: "bandit",
	})
}

// CICommands returns CI pipeline commands for the Python ecosystem.
// Commands vary based on the configured package manager.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	pm := config.PM("pip")

	var cmds []ecosystem.CICommand

	switch pm {
	case "pip":
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "pip-install",
			Command:     pipLockedInstallCommand,
			Description: "Install Python dependencies with hash verification and binary-only constraint",
			Phase:       ecosystem.CIPhaseInstall,
		})
	case "uv":
		// --locked (unlike --frozen) resolves and fails when uv.lock is out
		// of date, so the cooldown applies; it must match the devenv shell's
		// UV_EXCLUDE_NEWER or uv treats the lockfile as stale.
		command := "uv sync --locked"
		if config.Extra(extraUVExcludeNewer, "") != "project" {
			command += " --exclude-newer " + uvCooldown
		}
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "uv-sync",
			Command:     command,
			Description: "Install Python dependencies from an up-to-date uv lockfile with a 7-day age gate",
			Phase:       ecosystem.CIPhaseInstall,
		})
	case "poetry":
		cmds = append(cmds,
			ecosystem.CICommand{
				Name:        "poetry-check-lock",
				Command:     "poetry check --lock",
				Description: "Fail when poetry.lock is missing or out of date with pyproject.toml",
				Phase:       ecosystem.CIPhaseInstall,
			},
			ecosystem.CICommand{
				Name:        "poetry-install",
				Command:     "poetry install --no-interaction",
				Description: "Install Python dependencies from poetry lockfile",
				Phase:       ecosystem.CIPhaseInstall,
			},
		)
	}

	return append(cmds, ecosystem.CICommand{
		Name:        "pip-audit",
		Command:     "pip-audit",
		Description: "Audit Python dependencies for known vulnerabilities",
		Phase:       ecosystem.CIPhaseScan,
	})
}

// DevenvPackages returns pip-audit, which the generated CI's scan step runs
// in the devenv shell for every package manager.
func (m *Module) DevenvPackages(_ ecosystem.ModuleConfig) []string {
	return []string{"pip-audit"}
}

// PackageManagers returns metadata about Python's package managers.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:           "pip",
			LockFile:       "requirements.txt",
			InstallCommand: "pip install -r requirements.txt",
		},
		{
			Name:           "uv",
			LockFile:       "uv.lock",
			InstallCommand: "uv sync",
		},
		{
			Name:           "poetry",
			LockFile:       "poetry.lock",
			InstallCommand: "poetry install",
		},
	}
}

// WizardFields returns additional wizard form fields for Python configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         types.SettingVersion,
			Label:       "Python version",
			Description: "The Python version (MAJOR.MINOR); leave empty for the detected version or " + defaultPythonVersion,
			Type:        ecosystem.FieldTypeInput,
			Placeholder: "3.13",
		},
		{
			Key:         types.SettingPackageManager,
			Label:       "Package manager",
			Description: "Select the Python package manager to use",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "pip", Value: "pip"},
				{Label: "uv", Value: "uv"},
				{Label: "poetry", Value: "poetry"},
			},
			Default:  "pip",
			Required: true,
		},
	}
}

// VerificationCommands returns project verification commands for the Python ecosystem.
// mypy is only listed for projects that configure it (see PreCommitHooks).
func (m *Module) VerificationCommands(config ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	vc := ecosystem.VerificationCommands{
		Test:   []string{"python -m pytest"},
		Lint:   []string{"ruff check ."},
		Format: []string{"ruff format --check ."},
	}
	if config.Extra(extraMypy, "") == "true" {
		vc.TypeCheck = []string{"mypy ."}
	}
	return vc
}

// ManifestFiles returns manifest file metadata for the Python ecosystem.
// A pip-mode project that Detect found to be managed by an unsupported tool
// (pdm, pipenv, hatch, conda) reports that tool's manifest and lockfile.
func (m *Module) ManifestFiles(config ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	pm := config.PM("pip")
	if pm == "pip" {
		for _, um := range unsupportedManagers {
			if um.name == config.Extra(extraProjectManager, "") {
				return []ecosystem.ManifestFileInfo{um.manifest}
			}
		}
	}
	var manifests []ecosystem.ManifestFileInfo
	switch pm {
	case "uv":
		manifests = append(manifests, ecosystem.ManifestFileInfo{
			Path:           "pyproject.toml",
			Ecosystem:      "uv",
			VSSupported:    true,
			LockFile:       "uv.lock",
			LockFilePolicy: ecosystem.LockFilePolicyRequired,
		})
	case "poetry":
		manifests = append(manifests, ecosystem.ManifestFileInfo{
			Path:           "pyproject.toml",
			Ecosystem:      "poetry",
			VSSupported:    true,
			LockFile:       "poetry.lock",
			LockFilePolicy: ecosystem.LockFilePolicyRequired,
		})
	default:
		manifests = append(manifests, ecosystem.ManifestFileInfo{
			Path:           "requirements.txt",
			Ecosystem:      "pip",
			VSSupported:    true,
			LockFilePolicy: ecosystem.LockFilePolicyNone,
		})
	}
	return manifests
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to Python projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/python", "p/django", "p/flask", "p/owasp-top-ten"}
}
