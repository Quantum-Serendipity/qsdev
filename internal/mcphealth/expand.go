package mcphealth

import (
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
)

// envRefPattern matches the variable references Claude Code expands in
// .mcp.json: ${VAR} and ${VAR:-default}. Group 1 is the variable name and
// group 2, when present, is the ":-default" suffix.
var envRefPattern = regexp.MustCompile(`\$\{([^}:]+)(:-[^}]*)?\}`)

// envLookup resolves a variable name, reporting whether it is set.
type envLookup func(string) (string, bool)

// expandVars expands ${VAR} and ${VAR:-default} in s the way Claude Code does:
// a set variable expands to its value, an unset one to its default. An unset
// variable with no default is left as the literal reference so the probe (and
// ValidateConfig) can surface it rather than silently substituting "".
func expandVars(s string, lookup envLookup) string {
	if !strings.Contains(s, "${") {
		return s
	}
	return envRefPattern.ReplaceAllStringFunc(s, func(ref string) string {
		m := envRefPattern.FindStringSubmatch(ref)
		if val, ok := lookup(m[1]); ok {
			return val
		}
		if def, ok := strings.CutPrefix(m[2], ":-"); ok {
			return def
		}
		return ref
	})
}

// ProbeTarget returns cfg as CheckServer will start or dial it: with the
// references expanded from the current environment exactly as the probe
// expands them (see expandConfig). A probe gate inspects this, not the
// templates, so an environment value cannot smuggle in what the gate refuses.
func ProbeTarget(cfg ServerConfig) ServerConfig {
	return expandConfig(cfg, os.LookupEnv)
}

// expandConfig returns a copy of cfg with variable references expanded in
// the fields Claude Code expands: command and args always, url and header
// values only when cfg.ExpandEnv is set, since expanding those discloses the
// host environment to the remote endpoint. Env values are expanded by
// buildProcessEnv when the process environment is assembled.
func expandConfig(cfg ServerConfig, lookup envLookup) ServerConfig {
	cfg.Command = expandVars(cfg.Command, lookup)

	cfg.Args = slices.Clone(cfg.Args)
	for i, a := range cfg.Args {
		cfg.Args[i] = expandVars(a, lookup)
	}

	if !cfg.ExpandEnv {
		return cfg
	}
	cfg.URL = expandVars(cfg.URL, lookup)
	cfg.Headers = maps.Clone(cfg.Headers)
	for k, v := range cfg.Headers {
		cfg.Headers[k] = expandVars(v, lookup)
	}

	return cfg
}
