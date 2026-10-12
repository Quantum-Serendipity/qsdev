package gatedodge

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// The Change checks below compare the whole content a Write/Edit/MultiEdit
// leaves in a file against the file's current content. An Introduced check
// only scans the text a call introduces, so it cannot see a protective
// setting that is deleted (an Edit whose new_string is empty, a Write that
// omits the line) or changed to a value its patterns do not list.

// npmrcSettings are the .npmrc hardening settings besides ignore-scripts:
// npm's release-age gate (in days) and the registry packages come from.
var npmrcSettings = []setting{
	{key: "min-release-age", kind: minAge, ageUnit: 24 * time.Hour},
	{key: "registry", kind: keepValue},
}

// checkNpmrcResult is the GD-004 Change check of .npmrc. It blocks a change
// that turns npm install scripts back on (setting ignore-scripts to anything
// but true, or dropping an effective ignore-scripts=true), removes or lowers
// the min-release-age gate, or removes or changes the registry.
func checkNpmrcResult(before, after string) (string, error) {
	afterSet, afterOn := npmrcIgnoreScripts(after)
	if afterSet && !afterOn {
		return "ignore-scripts set to a value other than true re-enables install scripts", nil
	}
	if _, beforeOn := npmrcIgnoreScripts(before); beforeOn && !afterOn {
		return "removing ignore-scripts=true re-enables install scripts", nil
	}
	prev, next := npmrcValues(before), npmrcValues(after)
	for _, s := range npmrcSettings {
		if reason := s.weakened(prev, next); reason != "" {
			return reason, nil
		}
	}
	return "", nil
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

// checkPrecommitResult is the GD-003 Change check. It blocks a
// .pre-commit-config.yaml change that removes a configured hook, or leaves a
// file that pre-commit cannot read in place of one that had hooks.
func checkPrecommitResult(before, after string) (string, error) {
	beforeHooks, err := precommitHooks(before)
	if err != nil || len(beforeHooks) == 0 {
		return "", nil // nothing established to remove
	}
	afterHooks, err := precommitHooks(after)
	if err != nil {
		return "leaves an unreadable pre-commit config (" + err.Error() + ")", nil
	}
	var removed []string
	for _, h := range beforeHooks {
		if !slices.Contains(afterHooks, h) {
			removed = append(removed, h)
		}
	}
	if len(removed) > 0 {
		return "removes pre-commit hooks: " + strings.Join(removed, ", "), nil
	}
	return "", nil
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
