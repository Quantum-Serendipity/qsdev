package gatedodge

import (
	"regexp"
	"strings"
)

// Pre-compiled regexes for .qsdev.yaml detector.
var (
	reQsdevCompliance = regexp.MustCompile(`(?i)compliance_level:\s*(none|minimal|low)`)
	reQsdevSelfProt   = regexp.MustCompile(`(?i)self_protection:\s*false`)
	reQsdevSecEnf     = regexp.MustCompile(`(?i)security_enforcement:\s*false`)
	reQsdevHooksOff   = regexp.MustCompile(`(?i)hooks:.*enabled:\s*false`)
)

// Pre-compiled regex for devenv.nix detector.
var reDevenvDisable = regexp.MustCompile(`(?i)(QSDEV_DISABLE_HOOKS|GDEV_DISABLE_HOOKS)`)

// Pre-compiled regex for .pre-commit-config.yaml detector.
var rePrecommitEmpty = regexp.MustCompile(`stages:\s*\[\s*\]`)

// Pre-compiled regex for .npmrc detector.
var reNpmrcScripts = regexp.MustCompile(`(?i)ignore-scripts\s*=\s*false`)

// Pre-compiled regex for CLAUDE.md detector.
var reClaudeMDOverride = regexp.MustCompile(`(?i)(ignore\s+(?:all\s+)?security|disable\s+(?:all\s+)?hooks?|skip\s+(?:all\s+)?validation|never\s+block)`)

// Detect checks whether a Write/Edit operation to the given file contains
// content that would weaken security configuration.
// filePath is the target file path (not necessarily canonical).
// content is the proposed file content.
// Returns (blocked, ruleID, reason).
func Detect(filePath string, content string) (bool, string, string) {
	if content == "" {
		return false, "", ""
	}

	if strings.HasSuffix(filePath, ".qsdev.yaml") {
		return detectQsdevYaml(content)
	}

	if strings.HasSuffix(filePath, "devenv.nix") {
		return detectDevenvNix(content)
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

// detectQsdevYaml checks for patterns that weaken security in .qsdev.yaml.
func detectQsdevYaml(content string) (bool, string, string) {
	if reQsdevCompliance.MatchString(content) {
		return true, "GD-001", "compliance level set to a weak value"
	}
	if reQsdevSelfProt.MatchString(content) {
		return true, "GD-001", "self-protection disabled"
	}
	if reQsdevSecEnf.MatchString(content) {
		return true, "GD-001", "security enforcement disabled"
	}
	if reQsdevHooksOff.MatchString(content) {
		return true, "GD-001", "hooks disabled"
	}
	return false, "", ""
}

// detectDevenvNix checks for hook-disabling environment variables in devenv.nix.
func detectDevenvNix(content string) (bool, string, string) {
	if reDevenvDisable.MatchString(content) {
		return true, "GD-002", "hook-disabling environment variable in devenv.nix"
	}
	return false, "", ""
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
