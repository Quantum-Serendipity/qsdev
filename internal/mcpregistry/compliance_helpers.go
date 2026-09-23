package mcpregistry

import (
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// secretPrefixes are well-known prefixes that indicate a value is likely a
// secret token or API key.
var secretPrefixes = []string{
	"sk-",
	"sk_",
	"ghp_",
	"gho_",
	"token_",
	"key_",
	"secret_",
	"password_",
}

// sensitiveNamePattern matches env keys and CLI flag names whose literal value
// is a credential. metadataNameSuffix excludes names that merely describe where
// or what the credential is (TOKEN_FILE, --secret-name, API_KEY_URL).
var (
	sensitiveNamePattern = regexp.MustCompile(`(?i)(secret|token|passw(or)?d|api[_-]?key|credential)`)
	metadataNameSuffix   = regexp.MustCompile(`(?i)[_-](file|path|dir|url|uri|type|name|env|var|header)$`)
)

// varRefPattern matches a ${VAR} (or ${VAR:-default}) reference that the MCP
// client expands at launch; a reference is not a plaintext secret, but literal
// text next to one still is.
var varRefPattern = regexp.MustCompile(`\$\{[^}]*\}`)

// secretRedactor is the shared credential value-shape detector (AKIA…, ghp_…,
// github_pat_…, glpat-…, xox?-…, JWTs, PEM keys, URL userinfo, NAME=secret
// pairs). A value it would redact contains a secret.
var secretRedactor = sync.OnceValue(logging.NewRedactor)

// hasPlaintextSecrets returns true if any env value, argument, header value or
// the URL of the definition appears to carry a plaintext secret rather than a
// variable reference.
func hasPlaintextSecrets(def *McpServerDefinition) bool {
	for k, v := range def.Env {
		if looksLikeSecret(v) || isLiteralCredential(k, v) {
			return true
		}
	}
	if argsContainSecret(def.Args) {
		return true
	}
	for k, v := range def.Headers {
		if containsSecret(v) || isLiteralCredential(k, v) {
			return true
		}
	}
	return urlContainsSecret(def.URL)
}

// containsSecret reports whether any word of a composite value looks like a
// secret, so "--token=ghp_..." and "Bearer ghp_..." are caught as well as a
// bare token.
func containsSecret(value string) bool {
	if looksLikeSecret(value) {
		return true
	}
	words := strings.FieldsFunc(value, func(r rune) bool {
		return r == '=' || r == ' ' || r == '\t' || r == ','
	})
	return slices.ContainsFunc(words, looksLikeSecret)
}

// urlContainsSecret reports whether a server URL carries a secret: a
// credential-shaped value, userinfo, a secret-shaped path segment, or a
// credential-named query parameter (?api_key=...) with a literal value.
func urlContainsSecret(raw string) bool {
	if raw == "" {
		return false
	}
	if looksLikeSecret(raw) {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if _, hasPassword := u.User.Password(); hasPassword {
		return true
	}
	// A key embedded in the path (https://host/mcp/<key>/sse).
	if slices.ContainsFunc(strings.Split(u.Path, "/"), looksLikeSecret) {
		return true
	}
	for name, values := range u.Query() {
		for _, v := range values {
			if looksLikeSecret(v) || isLiteralCredential(name, v) {
				return true
			}
		}
	}
	return false
}

// argsContainSecret reports whether any argument looks like a secret, or a
// credential-named flag (--token X, --api-key=X) carries a literal value.
func argsContainSecret(args []string) bool {
	for i, arg := range args {
		if looksLikeSecret(arg) {
			return true
		}
		name, value, hasValue := strings.Cut(arg, "=")
		if !strings.HasPrefix(name, "-") {
			continue
		}
		if !hasValue {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				continue // a bare switch followed by another flag
			}
			value = args[i+1]
		}
		if isLiteralCredential(strings.TrimLeft(name, "-"), value) {
			return true
		}
	}
	return false
}

// isLiteralCredential reports whether name denotes a credential and value is a
// literal (not a variable reference, path or URL) long enough to be one.
func isLiteralCredential(name, value string) bool {
	if !sensitiveNamePattern.MatchString(name) || metadataNameSuffix.MatchString(name) {
		return false
	}
	lit := strings.TrimSpace(varRefPattern.ReplaceAllString(value, ""))
	if len(lit) < 8 {
		return false
	}
	return !looksLikePathOrURL(lit)
}

// looksLikeSecret applies a pragmatic heuristic to a single value: after
// removing ${...} references, the remaining literal text is suspicious if it
// matches a known credential shape, starts with a known secret prefix, or is
// long enough to be an opaque token.
func looksLikeSecret(value string) bool {
	lit := varRefPattern.ReplaceAllString(value, "")
	if lit == "" {
		return false
	}

	if secretRedactor().RedactString(lit) != lit {
		return true
	}

	// Short values are almost certainly configuration flags, not secrets.
	if len(lit) <= 20 {
		return false
	}

	lower := strings.ToLower(lit)
	for _, prefix := range secretPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	return looksLikeOpaqueToken(lit)
}

// looksLikeOpaqueToken reports whether v is a long, whitespace-free token. A
// value without '/' longer than 40 characters is treated as a token; a value
// of 40 or more characters that contains '/' (base64, e.g. an AWS secret
// access key) must mix upper case, lower case and digits and not look like a
// path, which keeps file paths and hex digests out.
func looksLikeOpaqueToken(v string) bool {
	if len(v) < 40 || strings.ContainsAny(v, " \t\r\n") || strings.Contains(v, "://") {
		return false
	}
	if !strings.Contains(v, "/") && len(v) > 40 {
		return true
	}
	return !looksLikePathOrURL(v) && hasMixedCharClasses(v)
}

func looksLikePathOrURL(v string) bool {
	return strings.Contains(v, "://") || strings.HasPrefix(v, "/") ||
		strings.HasPrefix(v, ".") || strings.HasPrefix(v, "~") || filepath.IsAbs(v)
}

func hasMixedCharClasses(v string) bool {
	var upper, lower, digit bool
	for _, r := range v {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	return upper && lower && digit
}

// launcherRule describes a package launcher that can fetch and run code from a
// registry at runtime.
type launcherRule struct {
	// fetchSubcommands are the subcommands that fetch; nil means every
	// invocation of the launcher may fetch.
	fetchSubcommands [][]string
	// offlineFlags forbid the launcher from reaching the network, so it can
	// only run what is already installed or cached.
	offlineFlags []string
	// pinned reports whether the invocation names an exact package version.
	pinned func(args []string) bool
}

// networkLaunchers maps a normalised launcher name to how it fetches. Package
// managers that also run local scripts (pnpm, yarn, bun, uv) only fetch for
// their dlx/x/create/run subcommands.
var networkLaunchers = map[string]launcherRule{
	"npx":    {offlineFlags: npmOfflineFlags, pinned: npmPackagePinned},
	"npm":    {offlineFlags: npmOfflineFlags, pinned: npmPackagePinned},
	"pnpx":   {offlineFlags: []string{"--offline"}, pinned: npmPackagePinned},
	"pnpm":   {fetchSubcommands: [][]string{{"dlx"}, {"create"}}, offlineFlags: []string{"--offline"}, pinned: npmPackagePinned},
	"yarn":   {fetchSubcommands: [][]string{{"dlx"}, {"create"}}, offlineFlags: []string{"--offline"}, pinned: npmPackagePinned},
	"bunx":   {pinned: npmPackagePinned},
	"bun":    {fetchSubcommands: [][]string{{"x"}, {"create"}}, pinned: npmPackagePinned},
	"deno":   {offlineFlags: []string{"--cached-only"}, pinned: npmPackagePinned},
	"uvx":    {offlineFlags: []string{"--offline"}, pinned: pythonPackagePinned},
	"uv":     {fetchSubcommands: [][]string{{"tool", "run"}, {"run"}}, offlineFlags: []string{"--offline"}, pinned: pythonPackagePinned},
	"pipx":   {pinned: pythonPackagePinned},
	"docker": {offlineFlags: []string{"--pull=never"}, pinned: imageDigestPinned},
	"podman": {offlineFlags: []string{"--pull=never"}, pinned: imageDigestPinned},
}

// npmOfflineFlags stop npx/npm exec from installing a missing package.
var npmOfflineFlags = []string{"--offline", "--no", "--no-install"}

// shellInterpreters run an arbitrary script, so what they launch cannot be
// inspected; graders fail closed on them.
var shellInterpreters = []string{"sh", "bash", "zsh", "dash", "ksh", "fish", "cmd", "powershell", "pwsh"}

// invocationKind classifies what a server command does at launch.
type invocationKind int

const (
	invocationLocal    invocationKind = iota // runs a local program
	invocationLauncher                       // package launcher that may fetch at runtime
	invocationOpaque                         // shell wrapper that cannot be inspected
)

// invocation is a server command normalised for the runtime-fetch criteria.
type invocation struct {
	kind invocationKind
	rule launcherRule
	args []string
}

// offline reports whether a launcher invocation forbids network access. A flag
// may be given as one argument (--pull=never) or two (--pull never).
func (inv invocation) offline() bool {
	for i, a := range inv.args {
		if slices.Contains(inv.rule.offlineFlags, a) {
			return true
		}
		if i+1 < len(inv.args) && slices.Contains(inv.rule.offlineFlags, a+"="+inv.args[i+1]) {
			return true
		}
	}
	return false
}

// classifyInvocation normalises the command (absolute paths, Windows .exe/.cmd
// shims, env wrappers) and classifies it as a local program, a network package
// launcher, or an opaque shell wrapper.
func classifyInvocation(command string, args []string) invocation {
	name := commandName(command)
	if name == "env" {
		inner, rest, split := unwrapEnv(args)
		switch {
		case split:
			// env -S runs a command line it splits itself, like a shell.
			return invocation{kind: invocationOpaque}
		case inner == "":
			return invocation{kind: invocationLocal}
		}
		return classifyInvocation(inner, rest)
	}
	if slices.Contains(shellInterpreters, name) {
		return invocation{kind: invocationOpaque}
	}
	rule, ok := networkLaunchers[name]
	if !ok {
		return invocation{kind: invocationLocal}
	}
	if rule.fetchSubcommands != nil && !hasAnySubcommand(args, rule.fetchSubcommands) {
		return invocation{kind: invocationLocal}
	}
	return invocation{kind: invocationLauncher, rule: rule, args: args}
}

// commandName returns the lower-cased base name of command without a Windows
// executable extension, so /usr/bin/npx and C:\nodejs\npx.cmd both yield "npx".
func commandName(command string) string {
	base := strings.ToLower(path.Base(strings.ReplaceAll(command, `\`, "/")))
	for _, ext := range []string{".exe", ".cmd", ".bat", ".ps1"} {
		if trimmed, ok := strings.CutSuffix(base, ext); ok {
			return trimmed
		}
	}
	return base
}

// unwrapEnv returns the command and arguments run by `env [OPTION]...
// [NAME=VALUE]... COMMAND [ARG]...` (an empty command when there is none).
// split reports a -S/--split-string option, whose value is a whole command
// line that cannot be inspected as separate arguments.
func unwrapEnv(args []string) (command string, rest []string, split bool) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--split-string" || strings.HasPrefix(a, "--split-string="),
			strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.Contains(a, "S"):
			return "", nil, true
		case a == "-u" || a == "--unset" || a == "-C" || a == "--chdir":
			i++ // option takes a value
		case strings.HasPrefix(a, "-") || strings.Contains(a, "="):
		default:
			return a, args[i+1:], false
		}
	}
	return "", nil, false
}

// hasAnySubcommand reports whether args contain any of the subcommand word
// sequences. It matches anywhere, so a flag before the subcommand
// (pnpm --silent dlx) cannot hide it; a spurious match fails closed.
func hasAnySubcommand(args []string, subcommands [][]string) bool {
	for _, sub := range subcommands {
		for i := 0; i+len(sub) <= len(args); i++ {
			if slices.Equal(args[i:i+len(sub)], sub) {
				return true
			}
		}
	}
	return false
}

// exactVersionPattern matches an exact semver-style version (1.2.3, 1.2.3-rc.1)
// but not a range, tag or dist-tag (^1.2, latest, next).
var exactVersionPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]+)?$`)

// pythonExactVersionPattern matches an exact PEP 440 release (1.2, 1.2.3,
// 1.2.3rc1, 1.2.3.post1) but not a wildcard (1.2.*).
var pythonExactVersionPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)*([a-z]+[0-9]*)?(\.(post|dev)[0-9]+)?$`)

// imageDigestPattern matches a container image reference pinned by digest.
var imageDigestPattern = regexp.MustCompile(`@sha256:[0-9a-f]{64}$`)

// packageSpec returns the package argument of a launcher invocation: the value
// of a package flag (-p/--package/--from/--spec) when present, else the first
// argument that is not a flag, subcommand word or launcher keyword.
func packageSpec(args []string, skip ...string) string {
	for i, a := range args {
		for _, flag := range []string{"-p", "--package", "--from", "--spec"} {
			if a == flag && i+1 < len(args) {
				return args[i+1]
			}
			if v, ok := strings.CutPrefix(a, flag+"="); ok {
				return v
			}
		}
	}
	for _, a := range args {
		if !strings.HasPrefix(a, "-") && !slices.Contains(skip, a) {
			return a
		}
	}
	return ""
}

// npmPackagePinned reports whether an npm-style spec ([npm:]name@1.2.3 or
// [npm:]@scope/name@1.2.3) names an exact version.
func npmPackagePinned(args []string) bool {
	spec := strings.TrimPrefix(packageSpec(args, "exec", "x", "dlx", "create", "run"), "npm:")
	at := strings.LastIndex(spec, "@")
	if at <= 0 {
		return false
	}
	return exactVersionPattern.MatchString(spec[at+1:])
}

// pythonPackagePinned reports whether a Python spec (name==1.2.3 or
// name@1.2.3) names an exact version.
func pythonPackagePinned(args []string) bool {
	spec := packageSpec(args, "tool", "run")
	if name, version, ok := strings.Cut(spec, "=="); ok && name != "" {
		return pythonExactVersionPattern.MatchString(version)
	}
	at := strings.LastIndex(spec, "@")
	if at <= 0 {
		return false
	}
	return exactVersionPattern.MatchString(spec[at+1:])
}

// imageDigestPinned reports whether a container invocation runs an image
// pinned by digest.
func imageDigestPinned(args []string) bool {
	return slices.ContainsFunc(args, imageDigestPattern.MatchString)
}

// isLocalOnly returns true when the server runs locally: a URL on the loopback
// interface (a remote endpoint is never local), or a command that does not
// fetch packages from the network at runtime: it is not a package launcher, or
// it is one explicitly forbidden from reaching the network. Shell wrappers
// cannot be inspected and fail closed.
func isLocalOnly(def *McpServerDefinition) bool {
	if def.URL != "" {
		return isLoopbackURL(def.URL)
	}
	inv := classifyInvocation(def.Command, def.Args)
	switch inv.kind {
	case invocationLocal:
		return true
	case invocationLauncher:
		return inv.offline()
	default:
		return false
	}
}

// hasRuntimeAutoInstall returns true when the server command can install a
// package at launch. Any package launcher can: npx and friends assume --yes
// when stdin is not a TTY, as it never is for a stdio MCP server, so -y makes
// no difference. A launcher invocation passes only when it is offline and pins
// an exact version (or image digest), so it can only run that already-present
// release. Shell wrappers cannot be inspected and fail closed.
func hasRuntimeAutoInstall(def *McpServerDefinition) bool {
	inv := classifyInvocation(def.Command, def.Args)
	switch inv.kind {
	case invocationLocal:
		return false
	case invocationLauncher:
		return !inv.offline() || inv.rule.pinned == nil || !inv.rule.pinned(inv.args)
	default:
		return true
	}
}

// nixStorePathPattern matches a path inside a Nix store object:
// /nix/store/<32-char nix-base32 hash>-<name>[/...]. Nix base32 omits e, o, t
// and u.
var nixStorePathPattern = regexp.MustCompile(`^/nix/store/[0-9a-df-np-sv-z]{32}-[^/]+(/|$)`)

// provenanceResolver holds the filesystem lookups the verified-provenance
// criterion needs, so tests can substitute them.
type provenanceResolver struct {
	lookPath     func(file string) (string, error)
	evalSymlinks func(path string) (string, error)
	executable   func() (string, error)
}

var defaultProvenance = provenanceResolver{
	lookPath:     exec.LookPath,
	evalSymlinks: filepath.EvalSymlinks,
	executable:   os.Executable,
}

// verified reports whether command has verifiable provenance: a well-formed,
// existing Nix store path (content-addressed) whose symlinks stay inside the
// store, or the bare qsdev command when it resolves on PATH to the running
// qsdev binary itself.
func (r provenanceResolver) verified(command string) bool {
	if command == branding.Get().AppName {
		return r.isSelf(command)
	}
	if !isNixStorePath(command) {
		return false
	}
	resolved, err := r.evalSymlinks(command)
	if err != nil {
		return false
	}
	return isNixStorePath(filepath.ToSlash(resolved))
}

// isSelf reports whether the bare command name resolves on PATH to the running
// executable, so a same-named binary planted earlier on PATH is not trusted.
func (r provenanceResolver) isSelf(command string) bool {
	found, err := r.lookPath(command)
	if err != nil {
		return false
	}
	resolved, err := r.evalSymlinks(found)
	if err != nil {
		return false
	}
	exe, err := r.executable()
	if err != nil {
		return false
	}
	self, err := r.evalSymlinks(exe)
	if err != nil {
		return false
	}
	return resolved == self
}

// isNixStorePath reports whether p is a clean (no ".." or duplicate
// separators) path inside a well-formed Nix store object.
func isNixStorePath(p string) bool {
	return path.Clean(p) == p && nixStorePathPattern.MatchString(p)
}

// AttestationChecker reports whether a server's command binary has a verified
// external attestation (a trusted Minisign signature). It defaults to a no-op
// that returns false; the claudecode addon overrides it at startup with a
// contentsign-backed implementation (mcpregistry must not import contentsign).
var AttestationChecker = func(*McpServerDefinition) bool { return false }

func hasExternalAttestation(def *McpServerDefinition) bool {
	return AttestationChecker(def)
}

// isLoopbackURL reports whether raw points at localhost or a loopback address.
func isLoopbackURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// LaunchesFromNetwork reports whether command is a package launcher (see
// networkLaunchers) that downloads and executes a package from the network
// whenever it is started, whatever its arguments. It matches on the binary's
// name (see commandName), so an absolute path or a Windows .cmd shim is
// classified the same as the bare name. Launchers that fetch only for some
// subcommands need their arguments; see launcherFetches. Callers that only
// want to observe a server, such as health diagnostics, must not start such a
// command: doing so installs and runs whatever version of the package is
// currently published.
func LaunchesFromNetwork(command string) bool {
	rule, ok := networkLaunchers[commandName(command)]
	return ok && rule.fetchSubcommands == nil
}

// launcherFetches reports whether command run with args is a package-launcher
// invocation that may fetch from the network at start (including subcommand
// launchers and env-wrapped launchers) and is not forced offline.
func launcherFetches(command string, args []string) bool {
	inv := classifyInvocation(command, args)
	return inv.kind == invocationLauncher && !inv.offline()
}
