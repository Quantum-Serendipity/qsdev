package mcpregistry

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/installer"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpconfig"
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

// MatchesTrusted reports whether cfg is exactly one of specs; see TrustMismatch.
func MatchesTrusted(cfg mcphealth.ServerConfig, specs []LaunchSpec) bool {
	return TrustMismatch(cfg, specs) == ""
}

// TrustMismatch returns "" when cfg is exactly one of specs, comparing the
// templates before expansion. The whole launch tuple — command, args, URL,
// headers and env — must be equal whatever the transport: an entry carrying
// both a command and a trusted URL is launched as stdio by Claude Code (an
// explicit "type":"stdio", or a schema that reads the command), so a remote
// spec only matches an entry with no command, and a stdio spec only matches
// one with no URL or headers.
//
// The release cutoff of a hardened launcher (see HardenedLauncherSpec)
// depends on the day and compliance level it was generated for, so an entry
// that matches no spec exactly is compared again with its cutoffs replaced by
// the placeholder LaunchVariants uses: any RFC 3339 cutoff at least the
// manager's baseline floor old then matches. A malformed cutoff, or one
// recent enough to loosen the baseline gate, matches nothing.
//
// Otherwise it names the cutoff that cannot be trusted, or the first field
// that differs from the closest spec (one of the same hardening form with the
// same launch line or endpoint, else the same command or URL, else the
// first). Repository strings are quoted, and env and header values are never
// included.
func TrustMismatch(cfg mcphealth.ServerConfig, specs []LaunchSpec) string {
	if len(specs) == 0 {
		return fmt.Sprintf("no trusted definition named %q", cfg.Name)
	}
	if anyMatch(cfg, specs, false) {
		return ""
	}
	canon, reason := canonicalCutoffs(cfg, time.Now())
	if reason != "" {
		return reason
	}
	if anyMatch(canon, specs, true) {
		return ""
	}
	return specDiff(canon, closestSpec(canon, specs))
}

// anyMatch reports whether cfg is exactly one of the specs that do
// (placeholder) or do not carry the cutoff placeholder. Keeping the passes
// apart means an entry holding the placeholder literally never matches.
func anyMatch(cfg mcphealth.ServerConfig, specs []LaunchSpec, placeholder bool) bool {
	return slices.ContainsFunc(specs, func(s LaunchSpec) bool {
		return hasCutoffPlaceholder(s) == placeholder && specMatches(cfg, s)
	})
}

// canonicalCutoffs returns cfg with its release cutoffs replaced by the
// placeholder (canonicalArgs, canonicalEnv), on copies of its args and env.
// When a cutoff is not one generation writes it returns the reason instead,
// naming the argument or the env key but never an env value.
func canonicalCutoffs(cfg mcphealth.ServerConfig, now time.Time) (mcphealth.ServerConfig, string) {
	args, ok := canonicalArgs(cfg.Args, now)
	if !ok {
		return cfg, fmt.Sprintf("args %q carry a release cutoff that is malformed or newer than the baseline release-age gate", cfg.Args)
	}
	env, ok := canonicalEnv(cfg.Env, now)
	if !ok {
		return cfg, fmt.Sprintf("env %q is a release cutoff that is malformed or newer than the baseline release-age gate", npmBeforeEnv)
	}
	cfg.Args, cfg.Env = args, env
	return cfg, ""
}

// specMatches reports whether cfg is exactly s: every launch field is equal.
func specMatches(cfg mcphealth.ServerConfig, s LaunchSpec) bool {
	return (s.Command != "" || s.URL != "") &&
		s.Command == cfg.Command && slices.Equal(s.Args, cfg.Args) &&
		s.URL == cfg.URL && maps.Equal(s.Headers, cfg.Headers) &&
		maps.Equal(s.Env, cfg.Env)
}

// sameTarget reports whether cfg dials s's URL or, without a URL, runs s's
// command. It only picks the spec a diff is reported against.
func sameTarget(cfg mcphealth.ServerConfig, s LaunchSpec) bool {
	if cfg.URL != "" {
		return s.URL == cfg.URL
	}
	return s.Command != "" && s.Command == cfg.Command
}

// sameLaunch reports whether cfg is s's launch line, or s's URL and headers.
// It only picks the spec a diff is reported against.
func sameLaunch(cfg mcphealth.ServerConfig, s LaunchSpec) bool {
	if cfg.URL != "" {
		return sameTarget(cfg, s) && maps.Equal(s.Headers, cfg.Headers)
	}
	return sameTarget(cfg, s) && slices.Equal(s.Args, cfg.Args)
}

// closestSpec returns the spec cfg is best compared against: one with the
// same launch line (or URL and headers), else the same command (or URL), else
// the first, looking first at the specs of cfg's hardening form (whether it
// carries the cutoff placeholder), so a hardened entry is compared with the
// hardened launcher. specs must not be empty.
func closestSpec(cfg mcphealth.ServerConfig, specs []LaunchSpec) LaunchSpec {
	hardened := hasCutoffPlaceholder(LaunchSpec{Args: cfg.Args, Env: cfg.Env})
	sameForm := func(s LaunchSpec) bool { return hasCutoffPlaceholder(s) == hardened }
	for _, same := range []func(mcphealth.ServerConfig, LaunchSpec) bool{sameLaunch, sameTarget} {
		for _, pass := range []func(LaunchSpec) bool{sameForm, func(LaunchSpec) bool { return true }} {
			if i := slices.IndexFunc(specs, func(s LaunchSpec) bool { return pass(s) && same(cfg, s) }); i >= 0 {
				return specs[i]
			}
		}
	}
	return specs[0]
}

// specDiff names the first field in which cfg differs from s: the launch
// line or endpoint, then headers, then env.
func specDiff(cfg mcphealth.ServerConfig, s LaunchSpec) string {
	if d := launchDiff(cfg, s); d != "" {
		return d
	}
	if d := mapDiff("header", cfg.Headers, s.Headers); d != "" {
		return d
	}
	if d := mapDiff("env", cfg.Env, s.Env); d != "" {
		return d
	}
	return "matches no trusted definition"
}

// launchDiff names how cfg's command, args or URL differ from s's, or "".
func launchDiff(cfg mcphealth.ServerConfig, s LaunchSpec) string {
	switch {
	case cfg.URL != "" && s.URL == "":
		return fmt.Sprintf("url %q where the trusted definition runs command %q", displayURL(cfg.URL), s.Command)
	case cfg.Command != "" && s.Command == "":
		return fmt.Sprintf("command %q where the trusted definition is remote url %q", cfg.Command, s.URL)
	case cfg.URL == "" && s.URL != "":
		return fmt.Sprintf("no url where the trusted definition is remote url %q", s.URL)
	case cfg.URL != s.URL:
		return fmt.Sprintf("url %q is not %q", displayURL(cfg.URL), s.URL)
	case cfg.Command != s.Command:
		return fmt.Sprintf("command %q is not %q", cfg.Command, s.Command)
	case !slices.Equal(cfg.Args, s.Args):
		return fmt.Sprintf("args %q differ from %q", cfg.Args, s.Args)
	}
	return ""
}

// displayURL returns the scheme and host of a configured URL, the part safe to
// print. Like header and env values, the rest (userinfo, path, query,
// fragment) may carry a literal credential such as ?apiKey=..., so it is
// replaced by "/…". A URL without a host prints as "<unparseable>".
func displayURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "<unparseable>"
	}
	shown := u.Scheme + "://" + u.Host
	if u.User != nil || strings.Trim(u.EscapedPath(), "/") != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		shown += "/…"
	}
	return shown
}

// mapDiff names the first key (in sorted order) in which got differs from
// want, without either value, or returns "".
func mapDiff(kind string, got, want map[string]string) string {
	for _, k := range slices.Sorted(maps.Keys(got)) {
		w, ok := want[k]
		switch {
		case !ok:
			return fmt.Sprintf("%s %q not in trusted definition", kind, k)
		case got[k] != w:
			return fmt.Sprintf("%s %q differs from trusted definition", kind, k)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(want)) {
		if _, ok := got[k]; !ok {
			return fmt.Sprintf("%s %q missing from configuration", kind, k)
		}
	}
	return ""
}

// EntrySpec is the definition an .mcp.json entry declares, such as one in
// output qsdev has just generated.
func EntrySpec(s mcpconfig.Server) LaunchSpec {
	return LaunchSpec{Command: s.Command, Args: s.Args, Env: s.Env, Headers: s.Headers, URL: s.URL}
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
