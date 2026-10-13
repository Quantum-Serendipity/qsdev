package claudecode

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/trust"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/internal/validation"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// SettingsJSON is the top-level structure that marshals to .claude/settings.json.
type SettingsJSON struct {
	// Env holds the environment Claude Code sets for the session and the
	// hooks it runs; qsdev uses it to hand hooks their configured policy.
	Env         map[string]string        `json:"env,omitempty"`
	Permissions Permissions              `json:"permissions"`
	Sandbox     *SandboxConfig           `json:"sandbox,omitempty"`
	Hooks       map[string][]HookMatcher `json:"hooks,omitempty"`
}

// FileBoundaryExtraReadPathsEnv is the variable through which the
// file-boundary hook receives .qsdev.yaml hooks.file_boundary.extra_read_paths,
// comma-separated.
const FileBoundaryExtraReadPathsEnv = "FILE_BOUNDARY_EXTRA_READ_PATHS"

// FileBoundaryDenyPathsEnv is the variable through which the file-boundary
// hook receives the home credential stores (secrets.CredentialHomePaths),
// comma-separated. It is set exactly when FileBoundaryExtraReadPathsEnv is:
// the hook drops any extra read path that overlaps one of these stores, and
// grants no extra read path at all when the list is missing. System paths are
// left out; validation rejects extra read paths overlapping them at check time.
const FileBoundaryDenyPathsEnv = "FILE_BOUNDARY_DENY_PATHS"

// ToolGatesAllowedEnv and ToolGatesDeniedEnv are the variables through which
// the tool-gates hook receives .qsdev.yaml hooks.tool_gates.allowed and
// hooks.tool_gates.denied, comma-separated.
const (
	ToolGatesAllowedEnv = "TOOL_GATES_ALLOWED"
	ToolGatesDeniedEnv  = "TOOL_GATES_DENIED"
)

// PackageGuardMinAgeDaysEnv is the variable through which the package-guard
// hook receives the compliance level's release-age window, in whole days.
const PackageGuardMinAgeDaysEnv = claudesettings.EnvPackageGuardMinAgeDays

// Permissions defines the permission rules for Claude Code.
type Permissions struct {
	DefaultMode                  string   `json:"defaultMode,omitempty"`
	DisableBypassPermissionsMode string   `json:"disableBypassPermissionsMode,omitempty"`
	Allow                        []string `json:"allow"`
	Deny                         []string `json:"deny"`
	Ask                          []string `json:"ask,omitempty"`
}

// SandboxConfig is Claude Code's settings.json "sandbox" block: it turns on
// the Bash sandbox and restricts what sandboxed commands may read, write and
// reach on the network.
type SandboxConfig struct {
	Enabled    bool               `json:"enabled"`
	Filesystem *SandboxFilesystem `json:"filesystem,omitempty"`
	Network    *SandboxNetwork    `json:"network,omitempty"`
}

// SandboxFilesystem is the "sandbox.filesystem" block.
type SandboxFilesystem struct {
	AllowWrite []string `json:"allowWrite,omitempty"`
	DenyWrite  []string `json:"denyWrite,omitempty"`
	DenyRead   []string `json:"denyRead,omitempty"`
}

// SandboxNetwork is the "sandbox.network" block.
type SandboxNetwork struct {
	AllowedDomains []string `json:"allowedDomains,omitempty"`
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

// AllBaseDenyRules returns the full deny rule list. Exported for use by
// the check command to verify deny rule coverage.
func AllBaseDenyRules() []string {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	return append(withNestedSecretFileRules(cat.AllPermissionDenyRules()), homeCredentialReadDenyRules()...)
}

// HomeCredentialReadGlobs returns the home credential stores of the
// credential canon as read globs (secrets.CredentialReadGlobs), for the
// public framework adapters, which may not import internal packages.
func HomeCredentialReadGlobs() []string {
	return secrets.CredentialReadGlobs()
}

// homeCredentialReadDenyRules returns a Read deny rule for each home
// credential store of the canon (secrets.CredentialReadGlobs). Every preset
// carries them whatever the languages, so a registry token such as ~/.npmrc
// is out of the Read tool's reach in any project. Known limits:
//   - Shell reads (Bash(cat ~/.npmrc)) are not covered; the Bash read policy
//     belongs to the shell-read work (U15/XS-WS5).
//   - Claude Code matches Read rules case-sensitively, so ~/.NPMRC on a
//     case-insensitive filesystem slips past them; the file-boundary hook is
//     the backstop.
//   - With the opt-in Claude Code sandbox, Claude Code applies Read denies to
//     sandboxed Bash too, so ssh, gpg, npm, mvn and gh lose their user
//     config inside it.
func homeCredentialReadDenyRules() []string {
	return readDenyPermissionRules(secrets.CredentialReadGlobs())
}

// rootSecretsReadRule is the catalog rule that read-denies the project's
// top-level secrets/ directory. It is anchored at the project root because
// an unanchored secrets/** would also deny source directories such as
// internal/secrets/ (U15-14).
const rootSecretsReadRule = "Read(/secrets/**)"

// withNestedSecretFileRules adds, to a deny list that read-denies the
// top-level secrets/ directory, a rule for each secret-material file pattern
// of the secrets canon (secrets.SecretFilePatterns) inside a directory named
// secrets at any depth (services/api/secrets/prod.env, deploy/secrets/tls.key),
// so nested secret stores are guarded while source code beside them is not.
func withNestedSecretFileRules(deny []string) []string {
	if !slices.Contains(deny, rootSecretsReadRule) {
		return deny
	}
	for _, p := range secrets.SecretFilePatterns() {
		deny = append(deny, "Read(/**/secrets/**/"+p+")")
	}
	return deny
}

// AllBaseAskRules returns every rule the catalog gates behind ask. Exported
// for qsdev check, which verifies no skill pre-approves any of them.
func AllBaseAskRules() []string {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	return cat.AllPermissionAskRules()
}

// ---------------------------------------------------------------------------
// Permission preset builder
// ---------------------------------------------------------------------------

// buildPermissions constructs the Permissions struct based on the selected
// permission preset, wizard answers, ecosystem registry, and addon config.
func buildPermissions(preset PermissionPreset, answers types.WizardAnswers, registry *ecosystem.Registry, cfg Config) (Permissions, error) {
	cat, err := catalog.Default()
	if err != nil {
		return Permissions{}, fmt.Errorf("loading catalog for permissions: %w", err)
	}
	// The home credential stores of the canon come first, then the ecosystem
	// deny rules, which include Read(...) rules for the credential files
	// modules declare read-denied, so the agent's own Read tool is blocked from
	// them whether or not the Bash sandbox is enabled.
	ecosystemDeny := homeCredentialReadDenyRules()
	ecosystemDeny = append(ecosystemDeny, collectEcosystemDenyRules(answers, registry)...)
	ecosystemDeny = append(ecosystemDeny, readDenyPermissionRules(collectEcosystemReadDenyRules(answers, registry))...)

	// A malformed committed rule is an error rather than being dropped, so a
	// bad policy is reported instead of silently narrowing or widening access.
	if errs := validation.CheckClaudePermissions(answers.ClaudePermissions); len(errs) > 0 {
		return Permissions{}, errs[0]
	}

	presetName := string(preset)
	presetDef, ok := cat.PermissionPreset(presetName)
	if !ok {
		// Never guess: silently substituting another preset could grant far
		// more access (e.g. Write(*)) than the configuration asked for.
		return Permissions{}, fmt.Errorf("unknown permission preset %q (valid: %s)",
			presetName, strings.Join(cat.PermissionPresets(), ", "))
	}

	// Assemble allow rules from preset's allow sets.
	var allow []string
	for _, setName := range presetDef.AllowSets {
		allow = append(allow, cat.PermissionAllowRules(setName)...)
	}

	// Assemble deny rules from preset's deny sets.
	var deny []string
	for _, setName := range presetDef.DenySets {
		deny = append(deny, cat.PermissionDenyRules(setName)...)
	}
	deny = withNestedSecretFileRules(deny)

	// Assemble ask rules from preset's ask sets.
	var ask []string
	for _, setName := range presetDef.AskSets {
		ask = append(ask, cat.PermissionAskRules(setName)...)
	}

	// Always add the home credential and ecosystem deny rules.
	deny = append(deny, ecosystemDeny...)

	switch preset {
	case PermissionPresetPermissive:
		// Mirror the permissive docker allows for Podman when it is detected.
		// Never `podman *`: container daemon access is root-equivalent.
		if answers.Detected.ContainerRuntime == "podman-rootless" || answers.Detected.ContainerRuntime == "podman-rootful" {
			for _, rule := range cat.PermissionAllowRules("permissive_extra") {
				if podman, ok := strings.CutPrefix(rule, "Bash(docker "); ok {
					allow = append(allow, "Bash(podman "+podman)
				}
			}
		}

	}

	// Every preset, custom and supply-chain-only included, takes the extras
	// (addon config, then the committed claude_code.permissions block) and
	// its catalog modes: custom and supply-chain-only allow nothing of their
	// own, so custom's allow list is exactly the extras. The preset's ask and
	// deny rules still take precedence over an extra allow.
	allow = append(allow, cfg.ExtraAllowPatterns...)
	allow = append(allow, answers.ClaudePermissions.Allow...)
	deny = append(deny, cfg.ExtraDenyPatterns...)
	deny = append(deny, answers.ClaudePermissions.Deny...)

	perms := Permissions{
		Allow: sliceutil.Dedup(allow),
		Deny:  sliceutil.Dedup(deny),
	}
	if len(ask) > 0 {
		perms.Ask = sliceutil.Dedup(ask)
	}
	if presetDef.DefaultMode != "" {
		perms.DefaultMode = presetDef.DefaultMode
	}
	if presetDef.DisableBypassMode != "" {
		perms.DisableBypassPermissionsMode = presetDef.DisableBypassMode
	}
	return perms, nil
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
		drp, ok := mod.(ecosystem.DenyRuleProvider)
		if !ok {
			continue
		}
		cfg := ecosystem.ToModuleConfig(lang)
		rules = append(rules, drp.DenyRules(cfg)...)
	}
	return sliceutil.Dedup(rules)
}

// collectEcosystemReadDenyRules iterates the selected languages, looks up each
// module in the registry, and aggregates their ReadDenyRules output.
func collectEcosystemReadDenyRules(answers types.WizardAnswers, registry *ecosystem.Registry) []string {
	if registry == nil {
		return nil
	}
	var rules []string
	for _, lang := range answers.Languages {
		mod, ok := registry.ByName(lang.Name)
		if !ok {
			continue
		}
		rdrp, ok := mod.(ecosystem.ReadDenyRuleProvider)
		if !ok {
			continue
		}
		cfg := ecosystem.ToModuleConfig(lang)
		rules = append(rules, rdrp.ReadDenyRules(cfg)...)
	}
	return sliceutil.Dedup(rules)
}

// readDenyPermissionRules converts module read-deny paths into Claude Code
// Read(...) permission deny rules. A trailing "/*" becomes "/**" so the rule
// also covers nested entries (e.g. ~/.aws/sso/cache/<dir>/<token>).
func readDenyPermissionRules(paths []string) []string {
	rules := make([]string, 0, len(paths))
	for _, p := range paths {
		if base, ok := strings.CutSuffix(p, "/*"); ok {
			p = base + "/**"
		}
		rules = append(rules, "Read("+p+")")
	}
	return rules
}

// mcpToolDenyRules returns whole-tool deny entries for the path-bearing tools of
// MCP servers in the fallback trust tier, projected from the path rules in deny
// (see trust.GenerateDenyRuleProjections). A permission rule cannot scope an
// MCP tool by path, so an untrusted server's file tools are denied outright;
// trusted servers' tools stay available and are path-checked by the
// confused-deputy PreToolUse hook.
//
// Servers are scored as the enforce hook scores them: from their generated
// .mcp.json definition, enriched by the known-server database. A server qsdev
// does not configure (one added in a user-scope MCP config, say) carries no
// trust signals and scores into the fallback tier. Manual tier overrides in the
// user's trust config are not applied: they are per-machine, and the committed
// settings.json must be the same for every checkout.
func mcpToolDenyRules(deny []string, answers types.WizardAnswers, cfg Config) ([]string, error) {
	servers, err := buildMcpServers(answers, cfg)
	if err != nil {
		return nil, err
	}
	engine, err := trust.NewMcpTrustEngine("")
	if err != nil {
		return nil, fmt.Errorf("creating MCP trust engine: %w", err)
	}

	tierOf := func(name string) trust.TrustTier {
		var configured *trust.McpServerInfo
		if entry, ok := servers[name]; ok {
			configured = &trust.McpServerInfo{
				Name:    name,
				Command: entry.Command,
				Args:    entry.Args,
				Env:     entry.Env,
			}
		}
		info := trust.ResolveServerInfo(name, configured)
		return engine.ScoreServer(&info).Tier
	}
	return trust.GenerateDenyRuleProjections(deny, tierOf), nil
}

// buildSandbox returns a SandboxConfig when sandbox is enabled, or nil
// otherwise. Its denyRead is the canon's home credential stores, then the
// selected modules' read-deny paths.
func buildSandbox(cfg Config, answers types.WizardAnswers, registry *ecosystem.Registry) *SandboxConfig {
	if !cfg.SandboxEnabled {
		return nil
	}
	denyRead := append(secrets.CredentialHomePaths(), collectEcosystemReadDenyRules(answers, registry)...)
	sandbox := &SandboxConfig{
		Enabled: true,
		Filesystem: &SandboxFilesystem{
			DenyWrite: []string{"/etc", "/usr"},
			DenyRead:  sliceutil.Dedup(denyRead),
		},
	}
	if len(cfg.AllowedDomains) > 0 {
		sandbox.Network = &SandboxNetwork{AllowedDomains: cfg.AllowedDomains}
	}
	return sandbox
}

// buildHookEnv returns the settings.json env entries that configure the
// enabled hooks from the committed hook policy, or nil when there are none.
// An entry that fails validation is an error rather than being dropped, so a
// bad policy is reported instead of silently narrowing or widening access.
func buildHookEnv(answers types.WizardAnswers) (map[string]string, error) {
	if errs := validation.CheckHookPolicy(answers.HookPolicy); len(errs) > 0 {
		return nil, errs[0]
	}
	env := make(map[string]string)
	if extras := answers.HookPolicy.FileBoundary.ExtraReadPaths; answers.Hooks.FileBoundary && len(extras) > 0 {
		setListEnv(env, FileBoundaryExtraReadPathsEnv, extras)
		setListEnv(env, FileBoundaryDenyPathsEnv, secrets.CredentialHomePaths())
	}
	if answers.Hooks.ToolGates {
		setListEnv(env, ToolGatesAllowedEnv, answers.HookPolicy.ToolGates.Allowed)
		setListEnv(env, ToolGatesDeniedEnv, answers.HookPolicy.ToolGates.Denied)
	}
	if packageGuardEnabled(answers) {
		packageGuardEnv(answers, env)
	}
	if len(env) == 0 {
		return nil, nil
	}
	return env, nil
}

// packageGuardEnv adds the package-guard hook's env entries to env: the
// compliance level's release-age window, raised (never lowered, D18) by
// hooks.package_guard.min_age_days; the committed team lists and
// new-dependency gate; and the dependency-decision trail when the soc2-audit
// hook is chosen. The policy was validated by the caller.
func packageGuardEnv(answers types.WizardAnswers, env map[string]string) {
	policy := answers.HookPolicy.PackageGuard
	days := packageGuardMinAgeDays(catalog.EffectiveAgeGate(answers.ComplianceLevel))
	if policy.MinAgeDays != nil {
		days = max(days, *policy.MinAgeDays)
	}
	env[PackageGuardMinAgeDaysEnv] = strconv.Itoa(days)
	setListEnv(env, claudesettings.EnvPackageGuardTeamAllowlist, policy.Allowlist)
	setListEnv(env, claudesettings.EnvPackageGuardTeamDenylist, policy.Denylist)
	if policy.NewDepGate != "" {
		env[claudesettings.EnvPackageGuardNewDepGate] = policy.NewDepGate
	}
	if answers.Hooks.SOC2Audit {
		env[claudesettings.EnvPackageGuardSOC2Audit] = "true"
	}
}

// setListEnv sets env[key] to values, deduplicated and comma-separated, when
// there are any. The entries were validated to contain no comma.
func setListEnv(env map[string]string, key string, values []string) {
	if len(values) > 0 {
		env[key] = strings.Join(sliceutil.Dedup(values), ",")
	}
}

// buildHooks returns the hooks map based on enabled hook presets, as the
// default HookRegistry emits it (see HookDefinition.emittedCommand).
func buildHooks(answers types.WizardAnswers) map[string][]HookMatcher {
	return defaultHookRegistry().BuildHooksMap(answers)
}

// sandboxHookCommand wraps a hook command to run inside the sandbox under the
// given category ("linter" when empty), invoking the branded binary appName.
func sandboxHookCommand(appName, category, command string) string {
	if category == "" {
		category = "linter"
	}
	return fmt.Sprintf(`%s sandbox exec --category %s -- %s`, appName, category, command)
}

// GenerateSettings produces a .claude/settings.json file from the wizard
// answers, ecosystem registry, and addon configuration. It returns a
// GeneratedFile ready for the generation pipeline.
func GenerateSettings(answers types.WizardAnswers, registry *ecosystem.Registry, cfg Config) (*types.GeneratedFile, error) {
	preset := cfg.DefaultPermissions
	if preset == "" {
		preset = PermissionPresetStandard
	}

	// Override from wizard answers. PermissionLevel is the most specific knob
	// (user-facing --claude-permissions flag), so it takes precedence when set
	// to something other than the tier's own default. When only Tier is set
	// (and PermissionLevel is empty because FillDefaults didn't populate it),
	// derive the preset from the tier.
	if answers.PermissionLevel != "" {
		preset = PermissionPreset(answers.PermissionLevel)
	} else if answers.Tier != "" {
		t := resolveTier(answers)
		preset = PermissionPreset(t.DefaultPermissionPreset())
	}

	perms, err := buildPermissions(preset, answers, registry, cfg)
	if err != nil {
		return nil, fmt.Errorf("building permissions: %w", err)
	}
	mcpDeny, err := mcpToolDenyRules(perms.Deny, answers, cfg)
	if err != nil {
		return nil, fmt.Errorf("projecting deny rules onto MCP tools: %w", err)
	}
	perms.Deny = sliceutil.Dedup(append(perms.Deny, mcpDeny...))

	env, err := buildHookEnv(answers)
	if err != nil {
		return nil, fmt.Errorf("building hook environment: %w", err)
	}

	settings := SettingsJSON{
		Env:         env,
		Permissions: perms,
		Sandbox:     buildSandbox(cfg, answers, registry),
		Hooks:       buildHooks(answers),
	}

	raw, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("marshaling settings.json: %w", err)
	}
	// Emit the same canonical form the three-way merge writes, so the first
	// update after init is not a key-reorder rewrite.
	jsonBytes, err := merge.CanonicalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("canonicalizing settings.json: %w", err)
	}

	return &types.GeneratedFile{
		Path:     ".claude/settings.json",
		Content:  jsonBytes,
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.ThreeWayMerge,
	}, nil
}

// packageGuardMinAgeDays returns the package guard's release-age gate in
// days for a compliance window: the same gate npm and pnpm enforce, the
// window raised to the 3-day floor the guard shipped with (D18), so a catalog
// overlay with a shorter window never loosens it.
func packageGuardMinAgeDays(window time.Duration) int {
	return ecosystem.ReleaseAgeDays(ecosystem.EffectiveReleaseAge(window, ecosystem.DefaultMinReleaseAge))
}
