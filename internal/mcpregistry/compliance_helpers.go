package mcpregistry

import (
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
var networkCommands = []string{"npx", "npm", "uvx", "pipx"}

// hasPlaintextSecrets returns true if any value in def.Env appears to be a
// plaintext secret rather than a variable reference.
func hasPlaintextSecrets(def *McpServerDefinition) bool {
	for _, v := range def.Env {
		if looksLikeSecret(v) {
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

// isLocalOnly returns true when the server command runs locally without
// fetching packages from the network at runtime.
func isLocalOnly(def *McpServerDefinition) bool {
	return !slices.Contains(networkCommands, def.Command)
}

// hasNpxDashY returns true when the command is "npx" and the arguments include
// the -y or --yes flag, which enables automatic installation of unreviewed
// packages.
func hasNpxDashY(def *McpServerDefinition) bool {
	if def.Command != "npx" {
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

// hasExternalAttestation is a placeholder for P30 Content Signing. It always
// returns false until external attestation verification is implemented.
func hasExternalAttestation(_ *McpServerDefinition) bool {
	return false
}
