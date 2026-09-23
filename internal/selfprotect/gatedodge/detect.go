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

// Detect checks whether a Write/Edit operation to the given file introduces
// content that would weaken security configuration.
// filePath is the target file path (not necessarily canonical).
// content is the text the operation introduces.
// Returns (blocked, ruleID, reason).
//
// Files whose protection depends on what the change removes as well as what
// it adds (.qsdev.yaml, devenv.nix) are checked by DetectChange instead.
func Detect(filePath string, content string) (bool, string, string) {
	if content == "" {
		return false, "", ""
	}

	if strings.HasSuffix(filePath, ".pre-commit-config.yaml") {
		return detectPrecommitConfig(content)
	}

	if strings.HasSuffix(filePath, ".npmrc") {
		return detectNpmrc(content)
	}

	if strings.HasSuffix(filePath, "CLAUDE.md") {
		return detectClaudeMD(content)
	}

	return false, "", ""
}

// FileChange returns a file's content before and after a Write/Edit/MultiEdit.
type FileChange func() (before, after string, err error)

// DetectChange checks whether a Write/Edit/MultiEdit weakens a security
// configuration file by comparing the file before and after the change:
// .qsdev.yaml (GD-001: a lower security level, a disabled security control,
// or a disabled security tool) and devenv.nix (GD-002: a module or git hook
// switched off, or a hardening switch turned back on). change is called only
// for these files; when it fails the change cannot be verified and is
// blocked (fail closed). Returns (blocked, ruleID, reason).
func DetectChange(filePath string, change FileChange) (bool, string, string) {
	var ruleID string
	var detect func(before, after string) (string, error)
	switch {
	case strings.HasSuffix(filePath, ".qsdev.yaml"):
		ruleID, detect = "GD-001", qsdevConfigDowngrade
	case strings.HasSuffix(filePath, "devenv.nix"):
		ruleID, detect = "GD-002", devenvNixDowngrade
	default:
		return false, "", ""
	}
	before, after, err := change()
	if err != nil {
		return true, ruleID, "cannot verify change to security configuration: " + err.Error()
	}
	reason, err := detect(before, after)
	switch {
	case err != nil:
		return true, ruleID, "cannot verify change to security configuration: " + err.Error()
	case reason != "":
		return true, ruleID, reason
	default:
		return false, "", ""
	}
}

// qsdevConfigDowngrade describes how an updated .qsdev.yaml weakens the
// project's security posture ("" when it does not): security.level drops,
// a security.* control that is on (explicitly, or by default when unset)
// turns off, or a security-category tool is newly disabled. A missing or
// unparseable current file is compared as the organization defaults.
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

	return disabledSecurityTool(current.Tools.Disabled, next.Tools.Disabled)
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

// devenvNixDowngrade describes how an updated devenv.nix weakens the
// environment ("" when it does not): a module or git hook that is enabled now
// is removed or switched off (directly or through a mkForce override), or a
// switch that is explicitly off (such as dotenv.enable, which keeps .env
// secrets out of the shell) is turned on. A new file has nothing to weaken.
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

// detectPrecommitConfig checks for patterns that neutralize pre-commit hooks.
func detectPrecommitConfig(content string) (bool, string, string) {
	if rePrecommitEmpty.MatchString(content) {
		return true, "GD-003", "empty stages array disables all hooks"
	}
	return false, "", ""
}

// detectNpmrc checks for patterns that re-enable npm install scripts.
func detectNpmrc(content string) (bool, string, string) {
	if reNpmrcScripts.MatchString(content) {
		return true, "GD-004", "ignore-scripts set to false re-enables install scripts"
	}
	return false, "", ""
}

// detectClaudeMD checks for patterns that override security directives.
func detectClaudeMD(content string) (bool, string, string) {
	if reClaudeMDOverride.MatchString(content) {
		return true, "GD-005", "content attempts to override security directives"
	}
	return false, "", ""
}
