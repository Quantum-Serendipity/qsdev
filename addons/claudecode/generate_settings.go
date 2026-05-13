package claudecode

import (
	"encoding/json"
	"fmt"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// SettingsJSON is the top-level structure that marshals to .claude/settings.json.
type SettingsJSON struct {
	Permissions Permissions              `json:"permissions"`
	Sandbox     *SandboxConfig           `json:"sandbox,omitempty"`
	Hooks       map[string][]HookMatcher `json:"hooks,omitempty"`
}

// Permissions defines the permission rules for Claude Code.
type Permissions struct {
	DefaultMode                  string   `json:"defaultMode,omitempty"`
	DisableBypassPermissionsMode string   `json:"disableBypassPermissionsMode,omitempty"`
	Allow                        []string `json:"allow"`
	Deny                         []string `json:"deny"`
	Ask                          []string `json:"ask,omitempty"`
}

// SandboxConfig defines filesystem and network sandbox restrictions.
type SandboxConfig struct {
	WriteDeny  []string `json:"writeDeny,omitempty"`
	WriteAllow []string `json:"writeAllow,omitempty"`
	ReadDeny   []string `json:"readDeny,omitempty"`
	NetAllow   []string `json:"netAllow,omitempty"`
}

// HookMatcher defines a matcher and its associated hooks within a hook event.
type HookMatcher struct {
	Matcher string      `json:"matcher"`
	Hooks   []HookEntry `json:"hooks"`
}

// HookEntry defines a single hook command within a HookMatcher.
type HookEntry struct {
	Type          string `json:"type"`
	Command       string `json:"command"`
	Timeout       int    `json:"timeout,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

// ---------------------------------------------------------------------------
// Base deny rules — categorized slices from the reference deny rules document.
// These are the "fast catch" layer blocking obvious package install patterns.
// ---------------------------------------------------------------------------

// denyJSPackageManagers blocks npm, npx, yarn, pnpm, and bun install commands.
var denyJSPackageManagers = []string{
	`Bash(npm install *)`,
	`Bash(npm install)`,
	`Bash(npm i *)`,
	`Bash(npm i)`,
	`Bash(npm add *)`,
	`Bash(npm add)`,
	`Bash(npm ci *)`,
	`Bash(npm ci)`,
	`Bash(npm update *)`,
	`Bash(npm update)`,
	`Bash(npm uninstall *)`,
	`Bash(npm uninstall)`,
	`Bash(npm remove *)`,
	`Bash(npm remove)`,
	`Bash(npx *)`,
	`Bash(yarn add *)`,
	`Bash(yarn install *)`,
	`Bash(yarn install)`,
	`Bash(yarn upgrade *)`,
	`Bash(yarn remove *)`,
	`Bash(pnpm add *)`,
	`Bash(pnpm install *)`,
	`Bash(pnpm install)`,
	`Bash(pnpm update *)`,
	`Bash(pnpm update)`,
	`Bash(pnpm remove *)`,
	`Bash(bun add *)`,
	`Bash(bun install *)`,
	`Bash(bun install)`,
	`Bash(bun remove *)`,
}

// denyPython blocks pip, pip3, python -m pip, pipx, and uv install commands.
var denyPython = []string{
	`Bash(pip install *)`,
	`Bash(pip install)`,
	`Bash(pip3 install *)`,
	`Bash(pip3 install)`,
	`Bash(pip uninstall *)`,
	`Bash(pip3 uninstall *)`,
	`Bash(python -m pip install *)`,
	`Bash(python3 -m pip install *)`,
	`Bash(python -m pip uninstall *)`,
	`Bash(python3 -m pip uninstall *)`,
	`Bash(pipx install *)`,
	`Bash(pipx uninstall *)`,
	`Bash(uv pip install *)`,
	`Bash(uv pip install)`,
	`Bash(uv add *)`,
	`Bash(uv sync *)`,
	`Bash(uv sync)`,
	`Bash(uv remove *)`,
}

// denyRust blocks cargo add and cargo install.
var denyRust = []string{
	`Bash(cargo add *)`,
	`Bash(cargo install *)`,
	`Bash(cargo install)`,
}

// denyGo blocks go get and go install.
var denyGo = []string{
	`Bash(go get *)`,
	`Bash(go install *)`,
}

// denyRuby blocks gem install and bundle install/add/update.
var denyRuby = []string{
	`Bash(gem install *)`,
	`Bash(bundle install *)`,
	`Bash(bundle install)`,
	`Bash(bundle add *)`,
	`Bash(bundle update *)`,
	`Bash(bundle update)`,
}

// denyPHP blocks composer require/install/update.
var denyPHP = []string{
	`Bash(composer require *)`,
	`Bash(composer install *)`,
	`Bash(composer install)`,
	`Bash(composer update *)`,
	`Bash(composer update)`,
}

// denyNix blocks nix-env imperative installs, nix profile, and cachix use.
var denyNix = []string{
	`Bash(nix-env -i *)`,
	`Bash(nix-env --install *)`,
	`Bash(nix-env -e *)`,
	`Bash(nix-env --erase *)`,
	`Bash(nix-env --uninstall *)`,
	`Bash(nix profile install *)`,
	`Bash(nix profile remove *)`,
	`Bash(cachix use *)`,
}

// denySystem blocks apt, brew, pacman, dnf, yum, apk, and snap.
var denySystem = []string{
	`Bash(apt install *)`,
	`Bash(apt-get install *)`,
	`Bash(apt remove *)`,
	`Bash(apt-get remove *)`,
	`Bash(brew install *)`,
	`Bash(brew uninstall *)`,
	`Bash(pacman -S *)`,
	`Bash(pacman -R *)`,
	`Bash(dnf install *)`,
	`Bash(yum install *)`,
	`Bash(apk add *)`,
	`Bash(snap install *)`,
}

// denyPipeToShell blocks curl|bash, curl|sh, wget|bash, wget|sh patterns.
var denyPipeToShell = []string{
	`Bash(curl * | bash *)`,
	`Bash(curl * | bash)`,
	`Bash(curl * | sh *)`,
	`Bash(curl * | sh)`,
	`Bash(wget * | bash *)`,
	`Bash(wget * | bash)`,
	`Bash(wget * | sh *)`,
	`Bash(wget * | sh)`,
}

// denyShellWrapping blocks bash -c, sh -c, zsh -c, dash -c wrapping of install commands.
var denyShellWrapping = []string{
	`Bash(bash -c *npm install*)`,
	`Bash(bash -c *pip install*)`,
	`Bash(bash -c *cargo install*)`,
	`Bash(bash -c *go get*)`,
	`Bash(bash -c *gem install*)`,
	`Bash(bash -c *nix-env*)`,
	`Bash(sh -c *npm install*)`,
	`Bash(sh -c *pip install*)`,
	`Bash(sh -c *cargo install*)`,
	`Bash(sh -c *go get*)`,
	`Bash(sh -c *gem install*)`,
	`Bash(sh -c *nix-env*)`,
	`Bash(zsh -c *npm install*)`,
	`Bash(zsh -c *pip install*)`,
	`Bash(zsh -c *cargo install*)`,
	`Bash(zsh -c *go get*)`,
	`Bash(zsh -c *gem install*)`,
	`Bash(zsh -c *nix-env*)`,
	`Bash(dash -c *npm install*)`,
	`Bash(dash -c *pip install*)`,
	`Bash(dash -c *cargo install*)`,
	`Bash(dash -c *go get*)`,
	`Bash(dash -c *gem install*)`,
	`Bash(dash -c *nix-env*)`,
}

// denyEnvCommandPrefix blocks env and command prefix bypasses.
var denyEnvCommandPrefix = []string{
	`Bash(env npm install *)`,
	`Bash(env pip install *)`,
	`Bash(env pip3 install *)`,
	`Bash(env cargo install *)`,
	`Bash(env nix-env *)`,
	`Bash(command npm install *)`,
	`Bash(command pip install *)`,
	`Bash(command pip3 install *)`,
	`Bash(command cargo install *)`,
	`Bash(command nix-env *)`,
}

// denySudoPrefix blocks sudo-prefixed install commands.
var denySudoPrefix = []string{
	`Bash(sudo npm install *)`,
	`Bash(sudo pip install *)`,
	`Bash(sudo pip3 install *)`,
	`Bash(sudo apt install *)`,
	`Bash(sudo apt-get install *)`,
	`Bash(sudo pacman -S *)`,
	`Bash(sudo nix-env *)`,
	`Bash(sudo gem install *)`,
}

// denySubprocessEscape blocks interpreter-based subprocess escapes.
var denySubprocessEscape = []string{
	`Bash(python -c *subprocess*)`,
	`Bash(python3 -c *subprocess*)`,
	`Bash(python -c *import os*)`,
	`Bash(python3 -c *import os*)`,
	`Bash(node -e *child_process*)`,
	`Bash(node -e *execSync*)`,
	`Bash(node -e *spawn*)`,
	`Bash(ruby -e *system*)`,
	`Bash(perl -e *system*)`,
}

// denyEvalXargs blocks eval and xargs-based indirect execution.
var denyEvalXargs = []string{
	`Bash(eval *npm install*)`,
	`Bash(eval *pip install*)`,
	`Bash(eval *cargo*)`,
	`Bash(eval *nix-env*)`,
	`Bash(xargs npm install *)`,
	`Bash(xargs pip install *)`,
	`Bash(xargs cargo install *)`,
}

// denyDestructiveOps blocks dangerous git, filesystem, and secret-reading operations.
var denyDestructiveOps = []string{
	`Bash(git push --force *)`,
	`Bash(git push * --force)`,
	`Bash(git reset --hard *)`,
	`Bash(rm -rf *)`,
	`Read(./.env)`,
	`Read(./.env.*)`,
	`Read(./secrets/**)`,
}

// allBaseDenyRules concatenates all categorized deny rule slices into a single
// comprehensive list. This is the full "fast catch" layer.
func allBaseDenyRules() []string {
	var rules []string
	rules = append(rules, denyJSPackageManagers...)
	rules = append(rules, denyPython...)
	rules = append(rules, denyRust...)
	rules = append(rules, denyGo...)
	rules = append(rules, denyRuby...)
	rules = append(rules, denyPHP...)
	rules = append(rules, denyNix...)
	rules = append(rules, denySystem...)
	rules = append(rules, denyPipeToShell...)
	rules = append(rules, denyShellWrapping...)
	rules = append(rules, denyEnvCommandPrefix...)
	rules = append(rules, denySudoPrefix...)
	rules = append(rules, denySubprocessEscape...)
	rules = append(rules, denyEvalXargs...)
	rules = append(rules, denyDestructiveOps...)
	return rules
}

// ---------------------------------------------------------------------------
// Allow rules for Standard and Permissive presets.
// ---------------------------------------------------------------------------

// allowMinimal is the baseline allow set for the minimal preset.
var allowMinimal = []string{
	`Read(*)`,
}

// allowMinimalBashBuildTest provides Bash-wrapped build/test commands for minimal.
var allowMinimalBashBuildTest = []string{
	`Bash(go build *)`,
	`Bash(go build)`,
	`Bash(go test *)`,
	`Bash(go test)`,
	`Bash(cargo build *)`,
	`Bash(cargo build)`,
	`Bash(cargo test *)`,
	`Bash(cargo test)`,
	`Bash(npm test *)`,
	`Bash(npm test)`,
	`Bash(npm run build *)`,
}

// allowStandardBase provides the standard preset's core allow rules.
var allowStandardBase = []string{
	`Read(*)`,
	`Edit(*)`,
	`Write(*)`,
	`Bash(git *)`,
}

// allowStandardBuildTestLint provides build, test, lint, and run commands.
var allowStandardBuildTestLint = []string{
	// JS script runners
	`Bash(npm run *)`,
	`Bash(npm test *)`,
	`Bash(npm test)`,
	`Bash(npm start *)`,
	`Bash(npm start)`,
	`Bash(npm run build *)`,
	`Bash(yarn run *)`,
	`Bash(pnpm run *)`,
	`Bash(bun run *)`,
	// Go
	`Bash(go build *)`,
	`Bash(go build)`,
	`Bash(go test *)`,
	`Bash(go test)`,
	`Bash(go run *)`,
	// Rust
	`Bash(cargo build *)`,
	`Bash(cargo build)`,
	`Bash(cargo test *)`,
	`Bash(cargo test)`,
	`Bash(cargo run *)`,
	`Bash(cargo run)`,
	// Ruby / PHP
	`Bash(bundle exec *)`,
	`Bash(composer run-script *)`,
	// Nix development
	`Bash(nix develop *)`,
	`Bash(nix develop)`,
	`Bash(nix build *)`,
	`Bash(nix build)`,
	`Bash(nix run *)`,
	`Bash(nix shell *)`,
	`Bash(nix flake check *)`,
	`Bash(nix flake show *)`,
	`Bash(devenv shell *)`,
	`Bash(devenv shell)`,
	// Read-only / informational
	`Bash(npm list *)`,
	`Bash(npm ls *)`,
	`Bash(npm outdated *)`,
	`Bash(npm audit *)`,
	`Bash(npm view *)`,
	`Bash(npm info *)`,
	`Bash(pip list *)`,
	`Bash(pip show *)`,
	`Bash(pip freeze *)`,
	`Bash(pip-audit *)`,
	`Bash(cargo audit *)`,
	`Bash(vulnix *)`,
	`Bash(nix flake info *)`,
	`Bash(nix flake metadata *)`,
}

// allowPermissiveExtra provides the additional rules for the permissive preset.
var allowPermissiveExtra = []string{
	`Bash(make *)`,
	`Bash(docker *)`,
}

// askStandard provides the ask rules for standard and permissive presets.
var askStandard = []string{
	`Bash(nix flake update *)`,
	`Bash(nix flake update)`,
	`Bash(pip install -r requirements.txt *)`,
	`Bash(pip install -r requirements.txt)`,
	`Bash(pip install -e . *)`,
	`Bash(pip install -e .)`,
}

// askMinimal provides the ask rules for the minimal preset.
var askMinimal = []string{
	`Bash(nix flake update *)`,
	`Bash(nix flake update)`,
}

// ---------------------------------------------------------------------------
// Permission preset builder
// ---------------------------------------------------------------------------

// buildPermissions constructs the Permissions struct based on the selected
// permission preset, wizard answers, ecosystem registry, and addon config.
func buildPermissions(preset PermissionPreset, answers types.WizardAnswers, registry *ecosystem.Registry, cfg Config) Permissions {
	baseDeny := allBaseDenyRules()
	ecosystemDeny := collectEcosystemDenyRules(answers, registry)

	var allow []string
	var deny []string
	var ask []string
	var defaultMode string
	var disableBypass string

	switch preset {
	case PermissionPresetMinimal:
		allow = append(allow, allowMinimal...)
		allow = append(allow, allowMinimalBashBuildTest...)
		deny = append(deny, baseDeny...)
		deny = append(deny, ecosystemDeny...)
		ask = append(ask, askMinimal...)

	case PermissionPresetStandard:
		allow = append(allow, allowStandardBase...)
		allow = append(allow, allowStandardBuildTestLint...)
		deny = append(deny, baseDeny...)
		deny = append(deny, ecosystemDeny...)
		ask = append(ask, askStandard...)
		defaultMode = "default"
		disableBypass = "disable"

	case PermissionPresetPermissive:
		allow = append(allow, allowStandardBase...)
		allow = append(allow, allowStandardBuildTestLint...)
		allow = append(allow, allowPermissiveExtra...)
		deny = append(deny, baseDeny...)
		deny = append(deny, ecosystemDeny...)
		ask = append(ask, askStandard...)
		defaultMode = "default"
		disableBypass = "disable"

	case PermissionPresetCustom:
		// Custom: allow only what's in ExtraAllowPatterns.
		allow = append(allow, cfg.ExtraAllowPatterns...)
		deny = append(deny, baseDeny...)
		deny = append(deny, ecosystemDeny...)
		deny = append(deny, cfg.ExtraDenyPatterns...)
		// Return early — don't append extras again below.
		return Permissions{
			Allow: dedup(allow),
			Deny:  dedup(deny),
		}
	}

	// For non-custom presets, append extra patterns from config.
	allow = append(allow, cfg.ExtraAllowPatterns...)
	deny = append(deny, cfg.ExtraDenyPatterns...)

	perms := Permissions{
		Allow: dedup(allow),
		Deny:  dedup(deny),
	}
	if len(ask) > 0 {
		perms.Ask = dedup(ask)
	}
	if defaultMode != "" {
		perms.DefaultMode = defaultMode
	}
	if disableBypass != "" {
		perms.DisableBypassPermissionsMode = disableBypass
	}
	return perms
}

// collectEcosystemDenyRules iterates the selected languages, looks up each
// module in the registry, and aggregates their DenyRules output.
func collectEcosystemDenyRules(answers types.WizardAnswers, registry *ecosystem.Registry) []string {
	if registry == nil {
		return nil
	}
	var rules []string
	for _, lang := range answers.Languages {
		mod, ok := registry.ByName(lang.Name)
		if !ok {
			continue
		}
		cfg := toModuleConfig(lang)
		rules = append(rules, mod.DenyRules(cfg)...)
	}
	return dedup(rules)
}

// buildSandbox returns a SandboxConfig when sandbox is enabled, or nil otherwise.
func buildSandbox(cfg Config) *SandboxConfig {
	if !cfg.SandboxEnabled {
		return nil
	}
	sandbox := &SandboxConfig{
		WriteDeny: []string{"/etc", "/usr"},
	}
	if len(cfg.AllowedDomains) > 0 {
		sandbox.NetAllow = cfg.AllowedDomains
	}
	return sandbox
}

// buildHooks returns the hooks map based on enabled hook presets.
func buildHooks(answers types.WizardAnswers) map[string][]HookMatcher {
	hooks := make(map[string][]HookMatcher)

	if answers.Hooks.SafetyBlock {
		hooks["PreToolUse"] = append(hooks["PreToolUse"], HookMatcher{
			Matcher: "Bash",
			Hooks: []HookEntry{
				{
					Type:          "command",
					Command:       `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py`,
					Timeout:       30,
					StatusMessage: "Checking package install safety...",
				},
			},
		})
	}

	if answers.Hooks.AuditLog {
		hooks["PostToolUse"] = append(hooks["PostToolUse"], HookMatcher{
			Matcher: "*",
			Hooks: []HookEntry{
				{
					Type:          "command",
					Command:       `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/audit-log.sh`,
					Timeout:       5,
					StatusMessage: "Logging tool action...",
				},
			},
		})
	}

	if len(hooks) == 0 {
		return nil
	}
	return hooks
}

// GenerateSettings produces a .claude/settings.json file from the wizard
// answers, ecosystem registry, and addon configuration. It returns a
// GeneratedFile ready for the generation pipeline.
func GenerateSettings(answers types.WizardAnswers, registry *ecosystem.Registry, cfg Config) (*types.GeneratedFile, error) {
	preset := cfg.DefaultPermissions
	if preset == "" {
		preset = PermissionPresetStandard
	}

	// Override from wizard answers if set.
	if answers.PermissionLevel != "" {
		preset = PermissionPreset(answers.PermissionLevel)
	}

	settings := SettingsJSON{
		Permissions: buildPermissions(preset, answers, registry, cfg),
		Sandbox:     buildSandbox(cfg),
		Hooks:       buildHooks(answers),
	}

	jsonBytes, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling settings.json: %w", err)
	}

	// Append trailing newline for POSIX compliance.
	jsonBytes = append(jsonBytes, '\n')

	return &types.GeneratedFile{
		Path:     ".claude/settings.json",
		Content:  jsonBytes,
		Mode:     0o644,
		Strategy: types.ThreeWayMerge,
	}, nil
}
