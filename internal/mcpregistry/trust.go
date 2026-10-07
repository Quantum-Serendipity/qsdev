package mcpregistry

import (
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/installer"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
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
//
// The release cutoff of a hardened launcher (see HardenedLauncherSpec)
// depends on the day and compliance level it was generated for, so an entry
// that matches no spec exactly is compared again with its cutoffs replaced by
// the placeholder LaunchVariants uses: any RFC 3339 cutoff at least the
// manager's baseline floor old then matches. A malformed cutoff, or one
// recent enough to loosen the baseline gate, matches nothing.
func MatchesTrusted(cfg mcphealth.ServerConfig, specs []LaunchSpec) bool {
	if anyMatch(cfg, specs, false, sameDefinition) {
		return true
	}
	now := time.Now()
	args, ok := canonicalArgs(cfg.Args, now)
	if !ok {
		return false
	}
	env, ok := canonicalEnv(cfg.Env, now)
	if !ok {
		return false
	}
	cfg.Args, cfg.Env = args, env
	return anyMatch(cfg, specs, true, sameDefinition)
}

// MatchesTrustedLaunch reports whether cfg launches what one of specs
// launches: the same command and args, or the same URL. Unlike
// MatchesTrusted it ignores env and headers, so it answers "is this the
// registry's server" rather than "may qsdev expand this entry's references".
// Launcher cutoffs in args are canonicalised as MatchesTrusted does.
func MatchesTrustedLaunch(cfg mcphealth.ServerConfig, specs []LaunchSpec) bool {
	if anyMatch(cfg, specs, false, sameLaunch) {
		return true
	}
	args, ok := canonicalArgs(cfg.Args, time.Now())
	if !ok {
		return false
	}
	cfg.Args = args
	return anyMatch(cfg, specs, true, sameLaunch)
}

// anyMatch reports whether cfg equals, under eq, one of the specs that do
// (placeholder) or do not carry the cutoff placeholder. Keeping the passes
// apart means an entry holding the placeholder literally never matches.
func anyMatch(cfg mcphealth.ServerConfig, specs []LaunchSpec, placeholder bool, eq func(mcphealth.ServerConfig, LaunchSpec) bool) bool {
	for _, s := range specs {
		if hasCutoffPlaceholder(s) == placeholder && eq(cfg, s) {
			return true
		}
	}
	return false
}

// sameDefinition is MatchesTrusted's equality: command, args and env for a
// stdio entry; URL, headers and env for a remote one.
func sameDefinition(cfg mcphealth.ServerConfig, s LaunchSpec) bool {
	if !maps.Equal(s.Env, cfg.Env) {
		return false
	}
	if cfg.URL != "" {
		return s.URL == cfg.URL && maps.Equal(s.Headers, cfg.Headers)
	}
	return s.Command != "" && s.Command == cfg.Command && slices.Equal(s.Args, cfg.Args)
}

// sameLaunch is MatchesTrustedLaunch's equality: command, args and URL.
func sameLaunch(cfg mcphealth.ServerConfig, s LaunchSpec) bool {
	if s.Command == "" && s.URL == "" {
		return false
	}
	return s.Command == cfg.Command && slices.Equal(s.Args, cfg.Args) && s.URL == cfg.URL
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

// Hardening applied to package launchers. npm reads npm_config_* environment
// variables as configuration, so an npx launcher needs no extra arguments.
const (
	npmIgnoreScriptsEnv = "npm_config_ignore_scripts"
	npmBeforeEnv        = "npm_config_before"
	// uvExcludeNewerFlag takes its cutoff in the same argument, so the cutoff
	// is never mistaken for the package spec of a launcher without --from.
	uvExcludeNewerFlag = "--exclude-newer="
	// releaseCutoffPlaceholder stands for the cutoff in the hardened variant
	// LaunchVariants trusts; MatchesTrusted maps a valid cutoff onto it.
	releaseCutoffPlaceholder = "<release-cutoff>"
)

// npmReleaseAge and uvReleaseAge return the release-age gate for packages
// npm and uv fetch on qsdev's behalf: the project's gate minAge (zero means
// the default), never below that manager's baseline floor.
func npmReleaseAge(minAge time.Duration) time.Duration {
	return ecosystem.EffectiveReleaseAge(minAge, installer.NpmMinReleaseAge)
}

func uvReleaseAge(minAge time.Duration) time.Duration {
	return ecosystem.EffectiveReleaseAge(minAge, installer.UvMinReleaseAge)
}

// HardenedLauncherSpec is LauncherSpec(def) with its package launcher
// age-gated and, for npm, stripped of lifecycle scripts: an npx launcher gets
// npm_config_before (and npm_config_ignore_scripts=true unless
// def.AllowInstallScripts), a uvx launcher a leading --exclude-newer=. The
// cutoff is minAge (raised to the manager's floor) before the UTC midnight
// of now, so regenerating on the same day writes the same entry, and the
// truncation only ever makes the gate stricter. Any other command, and a
// remote server, is returned as LauncherSpec has it.
func HardenedLauncherSpec(def catalog.MCPServerDef, minAge time.Duration, now time.Time) LaunchSpec {
	day := now.UTC().Truncate(24 * time.Hour)
	spec, _ := hardenLauncher(def,
		installer.ReleaseCutoff(day, npmReleaseAge(minAge)),
		installer.ReleaseCutoff(day, uvReleaseAge(minAge)))
	return spec
}

// hardenLauncher returns LauncherSpec(def) hardened with the given cutoffs,
// and whether def's launcher is one that hardening changes. The hardened
// spec owns its Env and Args.
func hardenLauncher(def catalog.MCPServerDef, npmCutoff, uvCutoff string) (LaunchSpec, bool) {
	spec := LauncherSpec(def)
	if spec.Command == "" {
		return spec, false
	}
	launcher := commandName(spec.Command)
	if launcher != "npx" && launcher != "uvx" {
		return spec, false
	}
	spec.Env = maps.Clone(spec.Env)
	spec.Args = slices.Clone(spec.Args)
	if launcher == "uvx" {
		spec.Args = slices.Insert(spec.Args, 0, uvExcludeNewerFlag+uvCutoff)
		return spec, true
	}
	if spec.Env == nil {
		spec.Env = make(map[string]string, 2)
	}
	if !def.AllowInstallScripts {
		spec.Env[npmIgnoreScriptsEnv] = "true"
	}
	spec.Env[npmBeforeEnv] = npmCutoff
	return spec, true
}

// hasCutoffPlaceholder reports whether s is a hardened variant whose cutoff
// LaunchVariants left as the placeholder.
func hasCutoffPlaceholder(s LaunchSpec) bool {
	return s.Env[npmBeforeEnv] == releaseCutoffPlaceholder ||
		slices.Contains(s.Args, uvExcludeNewerFlag+releaseCutoffPlaceholder)
}

// canonicalArgs returns a copy of args with each --exclude-newer= cutoff
// replaced by releaseCutoffPlaceholder. It reports false when a cutoff is not
// one generation writes (see validCutoff).
func canonicalArgs(args []string, now time.Time) ([]string, bool) {
	out := slices.Clone(args)
	for i, a := range out {
		v, ok := strings.CutPrefix(a, uvExcludeNewerFlag)
		if !ok {
			continue
		}
		if !validCutoff(v, now, installer.UvMinReleaseAge) {
			return nil, false
		}
		out[i] = uvExcludeNewerFlag + releaseCutoffPlaceholder
	}
	return out, true
}

// canonicalEnv returns env with its npm_config_before cutoff replaced by
// releaseCutoffPlaceholder, copying env rather than changing it. It reports
// false when the cutoff is not one generation writes (see validCutoff).
func canonicalEnv(env map[string]string, now time.Time) (map[string]string, bool) {
	v, ok := env[npmBeforeEnv]
	if !ok {
		return env, true
	}
	if !validCutoff(v, now, installer.NpmMinReleaseAge) {
		return nil, false
	}
	out := maps.Clone(env)
	out[npmBeforeEnv] = releaseCutoffPlaceholder
	return out, true
}

// validCutoff reports whether v is an RFC 3339 instant no later than floor
// before now; a later cutoff would admit releases the baseline gate refuses.
func validCutoff(v string, now time.Time, floor time.Duration) bool {
	t, err := time.Parse(time.RFC3339, v)
	return err == nil && !t.After(now.Add(-floor))
}

// LaunchVariants returns every definition generation may write for def: the
// launcher, its hardened form (HardenedLauncherSpec, with the cutoff as a
// placeholder MatchesTrusted resolves) and, when available, the installed
// binary, each as is and with each of def.OptionalArgs appended. The
// unhardened launcher stays trusted for entries generated before hardening.
func LaunchVariants(def catalog.MCPServerDef) []LaunchSpec {
	bases := []LaunchSpec{LauncherSpec(def)}
	if spec, ok := hardenLauncher(def, releaseCutoffPlaceholder, releaseCutoffPlaceholder); ok {
		bases = append(bases, spec)
	}
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
// keyed by server name: every LaunchVariants of the user-scope catalog
// (catalog.LoadUserScope: the embedded catalog plus the user's organization
// overlay), followed by extra (the servers configured into the binary). The
// project catalog overlay is deliberately excluded — like .mcp.json, it is
// repository content. An org overlay that fails to load contributes nothing,
// so its servers are treated as untrusted.
func TrustedDefinitions(extra map[string][]LaunchSpec) map[string][]LaunchSpec {
	trusted := make(map[string][]LaunchSpec)
	if cat, err := catalog.LoadUserScope(); err == nil {
		for name, def := range cat.MCPServers() {
			trusted[name] = LaunchVariants(def)
		}
	}
	for name, specs := range extra {
		trusted[name] = append(trusted[name], specs...)
	}
	return trusted
}
