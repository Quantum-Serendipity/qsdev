package mcpregistry

import (
	"net"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
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

// networkCommands are package-manager launchers that fetch packages from the
// network at runtime.
var networkCommands = []string{"npx", "npm", "pnpx", "bunx", "uvx", "pipx"}

// hasPlaintextSecrets returns true if any configured value appears to be a
// plaintext secret rather than a variable reference: an env value, an arg, a
// header value, or a credential embedded in the URL.
func hasPlaintextSecrets(def *McpServerDefinition) bool {
	for _, v := range def.Env {
		if looksLikeSecret(v) {
			return true
		}
	}
	for _, a := range def.Args {
		if containsSecret(a) {
			return true
		}
	}
	for _, v := range def.Headers {
		if containsSecret(v) {
			return true
		}
	}
	return urlHasSecret(def.URL)
}

// containsSecret reports whether any word of a composite value looks like a
// secret, so "--token=ghp_..." and "Bearer ghp_..." are caught as well as a
// bare token.
func containsSecret(value string) bool {
	words := strings.FieldsFunc(value, func(r rune) bool {
		return r == '=' || r == ' ' || r == '\t' || r == ','
	})
	return slices.ContainsFunc(words, looksLikeSecret)
}

// urlHasSecret reports whether a URL embeds a credential: a userinfo password,
// a path segment (as in https://host/mcp/<key>/sse) or a query parameter value.
func urlHasSecret(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		// Unparseable: fall back to scanning the raw string.
		return containsSecret(raw)
	}
	if pw, ok := u.User.Password(); ok && pw != "" && !strings.Contains(pw, "${") {
		return true
	}
	if slices.ContainsFunc(strings.Split(u.Path, "/"), looksLikeSecret) {
		return true
	}
	for _, vals := range u.Query() {
		if slices.ContainsFunc(vals, containsSecret) {
			return true
		}
	}
	return false
}

// looksLikeSecret applies a pragmatic heuristic: a value is suspicious if it
// is not a ${...} variable reference, exceeds a minimum length, and either
// starts with a known secret prefix or is long enough to be an opaque token.
func looksLikeSecret(value string) bool {
	// Variable references are not plaintext secrets.
	if strings.Contains(value, "${") {
		return false
	}

	// Short values are almost certainly configuration flags, not secrets.
	if len(value) <= 20 {
		return false
	}

	// Check for well-known secret prefixes.
	lower := strings.ToLower(value)
	for _, prefix := range secretPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	// Long opaque strings without path separators are likely tokens.
	if !strings.Contains(value, "/") && len(value) > 40 {
		return true
	}

	return false
}

// isLocalOnly returns true when the server runs locally: a command that does
// not fetch packages from the network at runtime, or a URL on the loopback
// interface. A remote endpoint is never local.
func isLocalOnly(def *McpServerDefinition) bool {
	if def.URL != "" {
		return isLoopbackURL(def.URL)
	}
	return !LaunchesFromNetwork(def.Command)
}

// LaunchesFromNetwork reports whether command is a package-manager launcher
// (npx, uvx, ...) that downloads and executes a package from the network when
// started. It matches on the binary's name (see launcherName), so an absolute
// path such as /usr/bin/npx or a Windows shim such as npx.cmd is classified the
// same as the bare name. Callers that only want to observe a server, such as
// health diagnostics, must not start such a command: doing so installs and runs
// whatever version of the package is currently published.
func LaunchesFromNetwork(command string) bool {
	return slices.Contains(networkCommands, launcherName(command))
}

// windowsExecExts are the executable extensions Windows resolves a bare command
// name to, so "npx.cmd" and "uvx.exe" name the same launchers as "npx"/"uvx".
var windowsExecExts = []string{".exe", ".cmd", ".bat", ".ps1"}

// launcherName reduces a command to the binary name it runs: the final path
// element (splitting on both / and \ regardless of the host OS, since configs
// are shared across platforms), lower-cased, without a Windows executable
// extension.
func launcherName(command string) string {
	name := strings.ToLower(command[strings.LastIndexAny(command, `/\`)+1:])
	if ext := filepath.Ext(name); slices.Contains(windowsExecExts, ext) {
		name = strings.TrimSuffix(name, ext)
	}
	return name
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

// hasNpxDashY returns true when the command is "npx" and the arguments include
// the -y or --yes flag, which enables automatic installation of unreviewed
// packages.
func hasNpxDashY(def *McpServerDefinition) bool {
	if launcherName(def.Command) != "npx" {
		return false
	}
	return slices.Contains(def.Args, "-y") || slices.Contains(def.Args, "--yes")
}

// hasVerifiedProvenance returns true when the command binary has verifiable
// provenance — currently Nix store paths (content-addressed) and the qsdev
// binary itself (built from source or Nix).
func hasVerifiedProvenance(def *McpServerDefinition) bool {
	if strings.HasPrefix(def.Command, "/nix/store/") {
		return true
	}
	if def.Command == "qsdev" {
		return true
	}
	return false
}

// AttestationChecker reports whether a server's command binary has a verified
// external attestation (a trusted Minisign signature). It defaults to a no-op
// that returns false; the claudecode addon overrides it at startup with a
// contentsign-backed implementation (mcpregistry must not import contentsign).
var AttestationChecker = func(*McpServerDefinition) bool { return false }

func hasExternalAttestation(def *McpServerDefinition) bool {
	return AttestationChecker(def)
}
