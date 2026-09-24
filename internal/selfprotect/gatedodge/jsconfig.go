package gatedodge

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// JavaScript package-manager hardening: qsdev writes pnpm-workspace.yaml,
// .yarnrc.yml, .yarnrc (Yarn Classic) and bunfig.toml (and .npmrc, see
// checkNpmrcResult) to keep dependency lifecycle scripts off and a release-age
// gate on. These rules compare the parsed file before and after a change, so
// deleting a setting, changing its value or spelling it differently is caught,
// not only a known bad literal.

// settingKind is how a hardening setting may change without weakening it.
type settingKind int

const (
	// keepTrue: once true it must stay true, and it may never be set false.
	keepTrue settingKind = iota
	// keepFalse: once false it must stay false, and it may never be set true.
	keepFalse
	// neverTrue: it may never be set true.
	neverTrue
	// minAge: a release-age gate that may not be removed or lowered.
	minAge
	// noNewEntries: an allowlist that may not gain entries.
	noNewEntries
	// keepValue: once set, it may not be removed or changed. Registries are
	// kept this way: a registry qsdev set is the organization's vetting proxy,
	// and replacing or dropping it sends installs past that proxy's checks.
	keepValue
)

// setting is one hardening key (dotted for nested tables) of a config file.
type setting struct {
	key  string
	kind settingKind
	// ageUnit is the unit of a bare number for minAge settings.
	ageUnit time.Duration
}

// configFormat parses a config file into flattened (dotted) keys.
type configFormat func(content string) (map[string]any, error)

// hardenedConfig describes one guarded package-manager config file.
type hardenedConfig struct {
	parse    configFormat
	settings []setting
}

var (
	pnpmWorkspaceConfig = hardenedConfig{parse: parseYAMLConfig, settings: []setting{
		{key: "strictDepBuilds", kind: keepTrue},
		{key: "blockExoticSubdeps", kind: keepTrue},
		{key: "dangerouslyAllowAllBuilds", kind: neverTrue},
		{key: "minimumReleaseAge", kind: minAge, ageUnit: time.Minute},
		{key: "minimumReleaseAgeExclude", kind: noNewEntries},
		{key: "onlyBuiltDependencies", kind: noNewEntries},
		{key: "allowBuilds", kind: noNewEntries},
		{key: "trustPolicy", kind: keepValue},
		{key: "trustPolicyExclude", kind: noNewEntries},
		{key: "npmRegistryServer", kind: keepValue},
		{key: "registry", kind: keepValue},
	}}
	yarnBerryConfig = hardenedConfig{parse: parseYAMLConfig, settings: []setting{
		{key: "enableScripts", kind: keepFalse},
		{key: "enableHardenedMode", kind: keepTrue},
		{key: "enableImmutableInstalls", kind: keepTrue},
		{key: "enableStrictSsl", kind: keepTrue},
		{key: "npmMinimalAgeGate", kind: minAge, ageUnit: time.Minute},
		{key: "npmPreapprovedPackages", kind: noNewEntries},
		{key: "npmRegistryServer", kind: keepValue},
	}}
	yarnClassicConfig = hardenedConfig{parse: parseYarnrcConfig, settings: []setting{
		{key: "ignore-scripts", kind: keepTrue},
		{key: "registry", kind: keepValue},
	}}
	bunConfig = hardenedConfig{parse: parseTOMLConfig, settings: []setting{
		{key: "install.minimumReleaseAge", kind: minAge, ageUnit: time.Second},
		{key: "install.minimumReleaseAgeExcludes", kind: noNewEntries},
		{key: "install.ignoreScripts", kind: keepTrue},
		{key: "install.registry", kind: keepValue},
	}}

	pnpmWorkspaceResult = &ResultRule{ID: "GD-004", check: pnpmWorkspaceConfig.check}
	yarnBerryResult     = &ResultRule{ID: "GD-004", check: yarnBerryConfig.check}
	yarnClassicResult   = &ResultRule{ID: "GD-004", check: yarnClassicConfig.check}
	bunResult           = &ResultRule{ID: "GD-004", check: bunConfig.check}
)

// check blocks a change that weakens any hardening setting. A result that
// cannot be parsed is blocked when the current file holds a setting to weaken.
func (c hardenedConfig) check(before, after string) (bool, string) {
	prev, err := c.parse(before)
	if err != nil {
		prev = nil // an unreadable current file has nothing established
	}
	next, err := c.parse(after)
	if err != nil {
		for _, s := range c.settings {
			if _, ok := prev[s.key]; ok {
				return true, "leaves an unreadable config (" + err.Error() + ")"
			}
		}
		return false, ""
	}
	for _, s := range c.settings {
		if reason := s.weakened(prev, next); reason != "" {
			return true, reason
		}
	}
	return false, ""
}

// weakened describes how next weakens s relative to prev ("" when it does not).
func (s setting) weakened(prev, next map[string]any) string {
	pv, pset := prev[s.key]
	nv, nset := next[s.key]
	switch s.kind {
	case keepTrue, keepFalse:
		want := s.kind == keepTrue
		if nset && !isBool(nv, want) {
			return fmt.Sprintf("%s set to %v", s.key, nv)
		}
		if pset && isBool(pv, want) && !nset {
			return s.key + " removed"
		}
	case neverTrue:
		if nset && !isBool(nv, false) {
			return fmt.Sprintf("%s set to %v", s.key, nv)
		}
	case minAge:
		if !pset {
			return ""
		}
		old, ok := parseAge(pv, s.ageUnit)
		if !ok {
			return ""
		}
		if !nset {
			return s.key + " removed"
		}
		cur, ok := parseAge(nv, s.ageUnit)
		if !ok || cur < old {
			return fmt.Sprintf("%s lowered from %v to %v", s.key, pv, nv)
		}
	case noNewEntries:
		if added := newEntries(pv, nv); len(added) > 0 {
			return fmt.Sprintf("%s gains %s: exempting dependencies from the install hardening is for a person to do", s.key, strings.Join(added, ", "))
		}
	case keepValue:
		if pset && (!nset || !reflect.DeepEqual(pv, nv)) {
			return fmt.Sprintf("%s changed from %v to %v", s.key, pv, valueOrRemoved(nv, nset))
		}
	}
	return ""
}

// valueOrRemoved renders a setting's value for a reason, or "(unset)".
func valueOrRemoved(v any, set bool) any {
	if !set {
		return "(unset)"
	}
	return v
}

// isBool reports whether v is the boolean want, written as a YAML/TOML bool
// or as a (case-insensitive) string.
func isBool(v any, want bool) bool {
	switch x := v.(type) {
	case bool:
		return x == want
	case string:
		b, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(x)))
		return err == nil && b == want
	}
	return false
}

// parseAge converts an age setting to a duration: a bare number is in unit,
// and a string may carry a unit suffix (s, m, h, d, w) as Yarn's
// npmMinimalAgeGate does ("7d").
func parseAge(v any, unit time.Duration) (time.Duration, bool) {
	switch x := v.(type) {
	case int:
		return time.Duration(x) * unit, true
	case int64:
		return time.Duration(x) * unit, true
	case float64:
		return time.Duration(x * float64(unit)), true
	case string:
		s := strings.TrimSpace(x)
		if n, err := strconv.ParseFloat(s, 64); err == nil {
			return time.Duration(n * float64(unit)), true
		}
		suffixes := map[byte]time.Duration{'s': time.Second, 'm': time.Minute, 'h': time.Hour, 'd': 24 * time.Hour, 'w': 7 * 24 * time.Hour}
		if s == "" {
			return 0, false
		}
		mult, ok := suffixes[s[len(s)-1]]
		if !ok {
			return 0, false
		}
		n, err := strconv.ParseFloat(s[:len(s)-1], 64)
		if err != nil {
			return 0, false
		}
		return time.Duration(n * float64(mult)), true
	}
	return 0, false
}

// newEntries lists the allowlist entries in next that prev lacks. A list names
// its entries; a map (pnpm's allowBuilds) names the packages mapped to a value
// other than false.
func newEntries(prev, next any) []string {
	have := allowEntries(prev)
	var added []string
	for _, e := range allowEntries(next) {
		if !slices.Contains(have, e) {
			added = append(added, e)
		}
	}
	return added
}

func allowEntries(v any) []string {
	var out []string
	switch x := v.(type) {
	case []any:
		for _, e := range x {
			out = append(out, fmt.Sprint(e))
		}
	case map[string]any:
		for _, k := range slices.Sorted(maps.Keys(x)) {
			if !isBool(x[k], false) {
				out = append(out, k)
			}
		}
	}
	return out
}

// parseYAMLConfig parses a YAML mapping's top-level keys.
func parseYAMLConfig(content string) (map[string]any, error) {
	m := map[string]any{}
	if err := yaml.Unmarshal([]byte(content), &m); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	return m, nil
}

// parseTOMLConfig parses TOML, flattening tables into dotted keys.
func parseTOMLConfig(content string) (map[string]any, error) {
	var doc map[string]any
	if _, err := toml.Decode(content, &doc); err != nil {
		return nil, fmt.Errorf("parsing TOML: %w", err)
	}
	flat := map[string]any{}
	var walk func(prefix string, m map[string]any)
	walk = func(prefix string, m map[string]any) {
		for k, v := range m {
			if sub, ok := v.(map[string]any); ok {
				walk(prefix+k+".", sub)
				continue
			}
			flat[prefix+k] = v
		}
	}
	walk("", doc)
	return flat, nil
}

// parseYarnrcConfig parses Yarn Classic's .yarnrc: one `key value` per line,
// '#' comments, optionally quoted keys and values, the last assignment wins.
// A key with no value is true.
func parseYarnrcConfig(content string) (map[string]any, error) {
	m := map[string]any{}
	for line := range strings.Lines(content) {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		key, value, hasValue := strings.Cut(line, " ")
		key = strings.Trim(key, `"'`)
		if !hasValue || strings.TrimSpace(value) == "" {
			m[key] = true
			continue
		}
		m[key] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return m, nil
}

// ConfigCommandTarget returns the guarded file that a package-manager command
// rewrites without naming it, or "": `yarn config set enableScripts true`
// edits the project's .yarnrc.yml, `npm config set --location=project`
// .npmrc, and `pnpm config set --location project` and `pnpm approve-builds`
// pnpm-workspace.yaml. Like a shell rewrite, that change skips the
// before/after check the Edit and Write tools go through. name is the
// command's base name.
func ConfigCommandTarget(name string, args []string) string {
	words := positionalArgs(args)
	if len(words) == 0 {
		return ""
	}
	configWrite := len(words) > 1 && (words[0] == "config" || words[0] == "c") &&
		slices.Contains([]string{"set", "delete", "unset", "edit", "fix"}, words[1])
	switch name {
	case "npm":
		if (configWrite || words[0] == "set") && projectLocation(args) {
			return ".npmrc"
		}
	case "pnpm":
		if words[0] == "approve-builds" || (configWrite && projectLocation(args)) {
			return "pnpm-workspace.yaml"
		}
	case "yarn":
		// Yarn Berry writes the project .yarnrc.yml unless -H/--home is given.
		if configWrite && !slices.Contains(args, "-H") && !slices.Contains(args, "--home") {
			return ".yarnrc.yml"
		}
	}
	return ""
}

// positionalArgs returns args without options and the value of a separate
// --location/-L option.
func positionalArgs(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--location" || a == "-L":
			i++
		case !strings.HasPrefix(a, "-"):
			out = append(out, a)
		}
	}
	return out
}

// projectLocation reports whether an npm/pnpm config command targets the
// project's config file.
func projectLocation(args []string) bool {
	for i, a := range args {
		if a == "--location=project" || ((a == "--location" || a == "-L") && i+1 < len(args) && args[i+1] == "project") {
			return true
		}
	}
	return false
}
