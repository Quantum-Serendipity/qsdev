package gatedodge

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ResultRule checks the whole content a Write/Edit/MultiEdit leaves in a file
// against the file's current content. Detect only scans the text a call
// introduces, so it cannot see a protective setting that is deleted (an Edit
// whose new_string is empty, a Write that omits the line) or changed to a value
// its patterns do not list; a ResultRule compares before and after instead.
type ResultRule struct {
	// ID is the gate-dodge rule ID reported on a block.
	ID    string
	check func(before, after string) (blocked bool, reason string)
}

// Check reports whether changing the file from before to after weakens it, and
// why.
func (r *ResultRule) Check(before, after string) (bool, string) {
	return r.check(before, after)
}

var (
	npmrcResultRule = &ResultRule{ID: "GD-004", check: checkNpmrcResult}
	precommitResult = &ResultRule{ID: "GD-003", check: checkPrecommitResult}
)

// resultRules maps each guarded file's lower-cased base name to its rule.
var resultRules = map[string]*ResultRule{
	".npmrc":                  npmrcResultRule,
	".pre-commit-config.yaml": precommitResult,
	"pnpm-workspace.yaml":     pnpmWorkspaceResult,
	".yarnrc.yml":             yarnBerryResult,
	".yarnrc":                 yarnClassicResult,
	"bunfig.toml":             bunResult,
}

// ResultRuleFor returns the ResultRule guarding filePath's resulting content,
// or nil when the file has none. The name matches regardless of case: on a
// case-insensitive filesystem (macOS, Windows) .NPMRC opens .npmrc, and
// guarding an unrelated differently-cased file elsewhere is harmless.
func ResultRuleFor(filePath string) *ResultRule {
	return resultRules[strings.ToLower(filepath.Base(filePath))]
}

// GuardedFileNames returns the lower-cased base names of the files that have
// a ResultRule, sorted. A shell command that rewrites one of them bypasses the
// before/after comparison, so callers deny those instead.
func GuardedFileNames() []string {
	return slices.Sorted(maps.Keys(resultRules))
}

// npmrcSettings are the .npmrc hardening settings besides ignore-scripts:
// npm's release-age gate (in days) and the registry packages come from.
var npmrcSettings = []setting{
	{key: "min-release-age", kind: minAge, ageUnit: 24 * time.Hour},
	{key: "registry", kind: keepValue},
}

// checkNpmrcResult blocks an .npmrc change that turns npm install scripts back
// on (setting ignore-scripts to anything but true, or dropping an effective
// ignore-scripts=true), removes or lowers the min-release-age gate, or
// removes or changes the registry.
func checkNpmrcResult(before, after string) (bool, string) {
	afterSet, afterOn := npmrcIgnoreScripts(after)
	if afterSet && !afterOn {
		return true, "ignore-scripts set to a value other than true re-enables install scripts"
	}
	if _, beforeOn := npmrcIgnoreScripts(before); beforeOn && !afterOn {
		return true, "removing ignore-scripts=true re-enables install scripts"
	}
	prev, next := npmrcValues(before), npmrcValues(after)
	for _, s := range npmrcSettings {
		if reason := s.weakened(prev, next); reason != "" {
			return true, reason
		}
	}
	return false, ""
}

// npmrcValues parses an .npmrc as ini (the last assignment wins; ';' and '#'
// start comments; values may be quoted).
func npmrcValues(content string) map[string]any {
	m := map[string]any{}
	for line := range strings.Lines(content) {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		m[strings.ToLower(strings.TrimSpace(key))] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return m
}

// npmrcIgnoreScripts reports whether an .npmrc assigns ignore-scripts and
// whether its effective (last) assignment is true. npm reads .npmrc as ini: ';'
// and '#' start comments, a bare key means true, and values may be quoted.
func npmrcIgnoreScripts(content string) (set, on bool) {
	for line := range strings.Lines(content) {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		key, value, hasValue := strings.Cut(line, "=")
		if !strings.EqualFold(strings.TrimSpace(key), "ignore-scripts") {
			continue
		}
		set = true
		if !hasValue {
			on = true
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		on = strings.EqualFold(value, "true") || value == "1"
	}
	return set, on
}

// precommitConfig is the part of .pre-commit-config.yaml that says which hooks
// run.
type precommitConfig struct {
	Repos []struct {
		Repo  string `yaml:"repo"`
		Hooks []struct {
			ID string `yaml:"id"`
		} `yaml:"hooks"`
	} `yaml:"repos"`
}

// checkPrecommitResult blocks a .pre-commit-config.yaml change that removes a
// configured hook, or leaves a file that pre-commit cannot read in place of
// one that had hooks.
func checkPrecommitResult(before, after string) (bool, string) {
	beforeHooks, err := precommitHooks(before)
	if err != nil || len(beforeHooks) == 0 {
		return false, "" // nothing established to remove
	}
	afterHooks, err := precommitHooks(after)
	if err != nil {
		return true, "leaves an unreadable pre-commit config (" + err.Error() + ")"
	}
	var removed []string
	for _, h := range beforeHooks {
		if !slices.Contains(afterHooks, h) {
			removed = append(removed, h)
		}
	}
	if len(removed) > 0 {
		return true, "removes pre-commit hooks: " + strings.Join(removed, ", ")
	}
	return false, ""
}

// precommitHooks lists the configured hooks as "repo#id".
func precommitHooks(content string) ([]string, error) {
	var cfg precommitConfig
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("parsing pre-commit config: %w", err)
	}
	var hooks []string
	for _, r := range cfg.Repos {
		for _, h := range r.Hooks {
			hooks = append(hooks, r.Repo+"#"+h.ID)
		}
	}
	return hooks, nil
}
