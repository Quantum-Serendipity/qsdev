package mcpregistry

import (
	"maps"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

// LaunchSpec is a server definition qsdev vouches for: a stdio command line or
// a remote endpoint, with its ${VAR} templates unexpanded.
type LaunchSpec struct {
	Command string
	Args    []string
	Env     map[string]string
	Headers map[string]string
	URL     string
}

// MatchesTrusted reports whether cfg is exactly one of specs, comparing the
// templates before expansion. A stdio entry must match the command, args and
// env; a remote entry (URL set) must match the URL, headers and env, since
// those decide where the expanded references are sent.
func MatchesTrusted(cfg mcphealth.ServerConfig, specs []LaunchSpec) bool {
	for _, s := range specs {
		if !maps.Equal(s.Env, cfg.Env) {
			continue
		}
		if cfg.URL != "" {
			if s.URL == cfg.URL && maps.Equal(s.Headers, cfg.Headers) {
				return true
			}
			continue
		}
		if s.Command != "" && s.Command == cfg.Command && slices.Equal(s.Args, cfg.Args) {
			return true
		}
	}
	return false
}

// LauncherSpec is the definition generation writes for def when no installed
// binary is used: the remote endpoint of an HTTP server, otherwise the pinned
// stdio launcher.
func LauncherSpec(def catalog.MCPServerDef) LaunchSpec {
	if def.Transport == "http" && def.URL != "" {
		return LaunchSpec{URL: def.URL, Env: def.Env}
	}
	return LaunchSpec{Command: def.Command, Args: def.Args, Env: def.Env}
}

// InstalledSpec is the definition generation writes for def once `qsdev mcp
// install` has installed its pinned release. It reports false when def's
// install method provides no binary or pins no version.
func InstalledSpec(def catalog.MCPServerDef) (LaunchSpec, bool) {
	if def.Bin == "" || def.Version == "" {
		return LaunchSpec{}, false
	}
	return LaunchSpec{Command: def.Bin, Args: def.BinArgs, Env: def.Env}, true
}

// LaunchVariants returns every definition generation may write for def: the
// launcher and, when available, the installed binary, each as is and with
// each of def.OptionalArgs appended.
func LaunchVariants(def catalog.MCPServerDef) []LaunchSpec {
	bases := []LaunchSpec{LauncherSpec(def)}
	if spec, ok := InstalledSpec(def); ok {
		bases = append(bases, spec)
	}
	variants := make([]LaunchSpec, 0, len(bases)*(1+len(def.OptionalArgs)))
	for _, base := range bases {
		variants = append(variants, base)
		if base.Command == "" {
			continue // optional arguments apply to stdio servers only
		}
		for _, opt := range def.OptionalArgs {
			v := base
			v.Args = slices.Concat(base.Args, opt)
			variants = append(variants, v)
		}
	}
	return variants
}

// TrustedDefinitions returns the server definitions qsdev itself vouches for,
// keyed by server name: every LaunchVariants of the embedded catalog plus the
// user's organization overlay, followed by extra (the servers configured into
// the binary). The project catalog overlay is deliberately excluded — like
// .mcp.json, it is repository content. A catalog that fails to load
// contributes nothing, so its servers are treated as untrusted.
func TrustedDefinitions(extra map[string][]LaunchSpec) map[string][]LaunchSpec {
	trusted := make(map[string][]LaunchSpec)
	var opts []catalog.LoadOption
	if org := catalog.OrgConfigFile(); org != "" {
		opts = append(opts, catalog.WithOrgConfigFile(org))
	}
	if cat, err := catalog.Load(opts...); err == nil {
		for name, def := range cat.MCPServers() {
			trusted[name] = LaunchVariants(def)
		}
	}
	for name, specs := range extra {
		trusted[name] = append(trusted[name], specs...)
	}
	return trusted
}
