package gatedodge

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Pre-compiled regexes for the devenv.nix detector: a module or git hook that
// is switched on (`name.enable = true;`, or a `name = {` block whose first
// attribute is `enable = true;`) or switched off (including via mkForce).
var (
	reNixEnableTrue  = regexp.MustCompile(`(?m)^\s*([A-Za-z_][\w.-]*)\.enable\s*=\s*true\s*;`)
	reNixEnableBlock = regexp.MustCompile(`(?m)^\s*([A-Za-z_][\w.-]*)\s*=\s*\{\s*\n\s*enable\s*=\s*true\s*;`)
	reNixEnableFalse = regexp.MustCompile(`(?m)^\s*([A-Za-z_][\w.-]*)\.enable\s*=\s*(?:lib\.)?(?:mkForce\s+)?false\s*;`)
)

// Pre-compiled regex for .pre-commit-config.yaml detector.
var rePrecommitEmpty = regexp.MustCompile(`stages:\s*\[\s*\]`)

// Pre-compiled regex for .npmrc detector.
var reNpmrcScripts = regexp.MustCompile(`(?i)ignore-scripts\s*=\s*false`)

// Pre-compiled regex for CLAUDE.md detector.
var reClaudeMDOverride = regexp.MustCompile(`(?i)(ignore\s+(?:all\s+)?security|disable\s+(?:all\s+)?hooks?|skip\s+(?:all\s+)?validation|never\s+block)`)

// securityToolCategory is the catalog tool category of the security tools
// (scanners, secret detection, guards) whose disabling is a downgrade.
const securityToolCategory = "security"

// qsdevConfigDowngrade is the GD-001 Change check of .qsdev.yaml. It
// describes how an updated .qsdev.yaml weakens the project's security
// posture ("" when it does not): security.level drops, a security.* control
// that is on (explicitly, or by default when unset) turns off,
// security.credential_vend is widened, a security-category tool is newly
// disabled, or claude_code.enabled turns off (the next regeneration would
// drop the Claude Code settings and the self-protection hook they register).
// A missing or unparseable current file is compared as the organization
// defaults.
func qsdevConfigDowngrade(before, after string) (string, error) {
	var next types.QsdevConfig
	if err := yaml.Unmarshal([]byte(after), &next); err != nil {
		return "", fmt.Errorf("parsing proposed .qsdev.yaml: %w", err)
	}
	defaults := config.DefaultQsdevConfig()
	current := *defaults
	if strings.TrimSpace(before) != "" {
		var parsed types.QsdevConfig
		if err := yaml.Unmarshal([]byte(before), &parsed); err == nil {
			current = parsed
		}
	}

	curLevel := levelOrDefault(current.Security.Level, defaults.Security.Level)
	nextLevel := levelOrDefault(next.Security.Level, defaults.Security.Level)
	if config.CompareComplianceLevels(nextLevel, curLevel) < 0 {
		return fmt.Sprintf("security.level lowered from %q to %q", curLevel, nextLevel), nil
	}

	controls := []struct {
		name      string
		cur, next *bool
		byDefault *bool
	}{
		{"age_gating", current.Security.AgeGating, next.Security.AgeGating, defaults.Security.AgeGating},
		{"script_blocking", current.Security.ScriptBlocking, next.Security.ScriptBlocking, defaults.Security.ScriptBlocking},
		{"lock_enforcement", current.Security.LockEnforcement, next.Security.LockEnforcement, defaults.Security.LockEnforcement},
		{"vuln_scanning", current.Security.VulnScanning, next.Security.VulnScanning, defaults.Security.VulnScanning},
	}
	for _, c := range controls {
		if boolOr(c.cur, c.byDefault) && !boolOr(c.next, c.byDefault) {
			return "security." + c.name + " disabled", nil
		}
	}

	if reason := credentialVendWidened(current.Security.CredentialVend, next.Security.CredentialVend); reason != "" {
		return reason, nil
	}

	if config.ClaudeCodeEnabled(&current) && !config.ClaudeCodeEnabled(&next) {
		return "claude_code.enabled turned off: regeneration would drop the Claude Code guardrails, including the self-protection hook", nil
	}

	return disabledSecurityTool(current.Tools.Disabled, next.Tools.Disabled)
}

// credentialVendWidened describes how next widens security.credential_vend,
// the opt-in that lets the MCP server hand cloud credentials to an agent (""
// when it does not): vending switched on, GetSessionToken allowed, or an
// identity or scope added to an allow-list.
func credentialVendWidened(cur, next types.CredentialVendConfig) string {
	const field = "security.credential_vend."
	if next.Enabled && !cur.Enabled {
		return field + "enabled turned on"
	}
	if next.AWS.AllowSessionToken && !cur.AWS.AllowSessionToken {
		return field + "aws.allow_session_token turned on"
	}
	lists := []struct {
		name      string
		cur, next []string
	}{
		{"aws.role_arns", cur.AWS.RoleARNs, next.AWS.RoleARNs},
		{"gcp.service_accounts", cur.GCP.ServiceAccounts, next.GCP.ServiceAccounts},
		{"azure.scopes", cur.Azure.Scopes, next.Azure.Scopes},
		{"azure.identities", cur.Azure.Identities, next.Azure.Identities},
	}
	for _, l := range lists {
		for _, v := range l.next {
			if !slices.Contains(l.cur, v) {
				return fmt.Sprintf("%q added to %s%s", v, field, l.name)
			}
		}
	}
	return ""
}

// disabledSecurityTool reports the first security-category tool (per the
// catalog) that next disables and current does not.
func disabledSecurityTool(current, next []string) (string, error) {
	var added []string
	for _, name := range next {
		if !slices.Contains(current, name) {
			added = append(added, name)
		}
	}
	if len(added) == 0 {
		return "", nil
	}
	cat, err := catalog.Default()
	if err != nil {
		return "", fmt.Errorf("loading tool catalog: %w", err)
	}
	for _, name := range added {
		if tool, ok := cat.Tool(name); ok && tool.Category == securityToolCategory {
			return fmt.Sprintf("security tool %q added to tools.disabled", name), nil
		}
	}
	return "", nil
}

func levelOrDefault(level, fallback string) string {
	if level == "" {
		return fallback
	}
	return level
}

func boolOr(v, fallback *bool) bool {
	if v != nil {
		return *v
	}
	return fallback != nil && *fallback
}

// devenvNixDowngrade is the GD-002 Change check of devenv.nix. It describes
// how an updated devenv.nix weakens the environment ("" when it does not): a
// module or git hook that is enabled now is removed or switched off
// (directly or through a mkForce override), or a switch that is explicitly
// off (such as dotenv.enable, which keeps .env secrets out of the shell) is
// turned on. A new file has nothing to weaken.
func devenvNixDowngrade(before, after string) (string, error) {
	enabledBefore, enabledAfter := nixEnabled(before), nixEnabled(after)
	disabledBefore, disabledAfter := nixDisabled(before), nixDisabled(after)

	for _, name := range sortedKeys(enabledBefore) {
		if !enabledAfter[name] {
			return fmt.Sprintf("devenv.nix no longer enables %q", name), nil
		}
	}
	enabledLeaves := make(map[string]bool)
	for name := range enabledBefore {
		enabledLeaves[nixLeaf(name)] = true
	}
	for _, name := range sortedKeys(disabledAfter) {
		if !disabledBefore[name] && enabledLeaves[nixLeaf(name)] {
			return fmt.Sprintf("devenv.nix overrides %q to disabled", name), nil
		}
	}
	for _, name := range sortedKeys(disabledBefore) {
		if enabledAfter[name] {
			return fmt.Sprintf("devenv.nix enables %q, which is disabled for security", name), nil
		}
	}
	return "", nil
}

// nixEnabled returns the attribute names switched on in a devenv.nix.
func nixEnabled(src string) map[string]bool {
	names := make(map[string]bool)
	for _, re := range []*regexp.Regexp{reNixEnableTrue, reNixEnableBlock} {
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			names[m[1]] = true
		}
	}
	return names
}

// nixDisabled returns the attribute names switched off in a devenv.nix.
func nixDisabled(src string) map[string]bool {
	names := make(map[string]bool)
	for _, m := range reNixEnableFalse.FindAllStringSubmatch(src, -1) {
		names[m[1]] = true
	}
	return names
}

// nixLeaf returns the last component of a dotted attribute path, so
// `git-hooks.hooks.ripsecrets` and `ripsecrets` (inside the git-hooks.hooks
// block) are recognised as the same hook.
func nixLeaf(name string) string {
	return name[strings.LastIndexByte(name, '.')+1:]
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// detectPrecommitConfig is the GD-003 Introduced check: introduced text that
// neutralizes pre-commit hooks.
func detectPrecommitConfig(content string) string {
	if rePrecommitEmpty.MatchString(content) {
		return "empty stages array disables all hooks"
	}
	return ""
}

// detectNpmrc is the GD-004 Introduced check of .npmrc: introduced text that
// re-enables npm install scripts.
func detectNpmrc(content string) string {
	if reNpmrcScripts.MatchString(content) {
		return "ignore-scripts set to false re-enables install scripts"
	}
	return ""
}

// detectClaudeMD is the GD-005 Introduced check: introduced text that
// overrides security directives.
func detectClaudeMD(content string) string {
	if reClaudeMDOverride.MatchString(content) {
		return "content attempts to override security directives"
	}
	return ""
}
