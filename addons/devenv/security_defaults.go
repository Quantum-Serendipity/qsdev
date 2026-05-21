package devenv

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// defaultUnsetEnvVars returns the canonical list of credential-bearing
// environment variables stripped from the devenv shell.
func defaultUnsetEnvVars() []string {
	return catalog.Default().UnsetVars()
}

// defaultSecurityHooks returns the built-in git-hooks.nix hooks that are
// always enabled for security scanning.
func defaultSecurityHooks() []string {
	return catalog.Default().SecurityHooks()
}

// defaultBasePackages returns the minimal set of packages always included.
func defaultBasePackages() []string {
	return catalog.Default().BasePackages()
}

// defaultCleanKeep returns the allowlist of environment variables that pass
// through when clean.enabled is true.
func defaultCleanKeep() []string {
	return catalog.Default().KeepVars()
}

// defaultSpecializedHooks returns the specialized custom security hooks that
// are always present. These use custom Nix expressions for advanced checks.
func defaultSpecializedHooks() []CustomHookData {
	cat := catalog.Default()
	defs := cat.CustomHooks()

	hooks := make([]CustomHookData, 0, len(defs))
	for _, def := range defs {
		hook := CustomHookData{
			ID:            def.ID,
			Name:          def.Name,
			Description:   def.Description,
			Language:      def.Language,
			Files:         def.Files,
			PassFilenames: def.PassFilenames,
			Stages:        def.Stages,
		}

		switch def.ID {
		case "lock-file-audit":
			hook.Entry = fmt.Sprintf(`pkgs.writeShellScript "lock-audit" ''
        %s
      ''`, strings.TrimSpace(def.Entry))
			hook.RawEntry = true

		case "nix-secrets-check":
			hook.Entry = buildNixSecretsCheckEntry(def)
			hook.RawEntry = true

		default:
			if def.Entry != "" {
				hook.Entry = def.Entry
			}
		}

		hooks = append(hooks, hook)
	}

	return hooks
}

func buildNixSecretsCheckEntry(def catalog.CustomHookDef) string {
	envPattern := def.EnvPattern
	credPattern := "(" + strings.Join(def.CredentialPatterns, "|") + ")"

	return fmt.Sprintf(`pkgs.writeShellScript "nix-secrets-check" ''
        ret=0
        for f in "$@"; do
          if ${pkgs.gnugrep}/bin/grep -nP '%s' "$f" 2>/dev/null; then
            echo "ERROR: $f appears to set a secret via env.*"
            ret=1
          fi
          if ${pkgs.gnugrep}/bin/grep -nP '%s' "$f" 2>/dev/null; then
            echo "ERROR: $f appears to contain a hardcoded credential"
            ret=1
          fi
        done
        exit $ret
      ''`, envPattern, credPattern)
}

// buildEnterShellScript returns the shell script body for devenv.nix enterShell.
func buildEnterShellScript() string {
	prefix := branding.Get().EnvPrefix
	return fmt.Sprintf(`echo ""
echo "=== Security-Hardened Development Environment ==="

# Verify git hooks are installed
if [ -d .git ] && [ ! -f .git/hooks/pre-commit ]; then
  echo "  WARNING: Pre-commit hooks not installed."
  echo "           Run 'devenv shell' to install them."
else
  echo "  Pre-commit hooks: active"
fi

# Verify clean environment is working (spot-check)
if [ -n "${AWS_SECRET_ACCESS_KEY:-}" ]; then
  echo "  WARNING: AWS_SECRET_ACCESS_KEY is set in environment!"
  echo "           This should have been stripped by clean mode."
  echo "           Check devenv.yaml clean.keep settings."
fi

# Verify ripsecrets is available
if command -v ripsecrets >/dev/null 2>&1; then
  echo "  Secret scanning: available (ripsecrets)"
else
  echo "  WARNING: ripsecrets not found in PATH"
fi

echo "==================================================="
echo ""
echo "  ${%[1]sPROJECT_NAME:-unknown} | security: ${%[1]sSECURITY_PROFILE:-standard} | tools: ${%[1]sTOOL_COUNT:-0}"
echo ""

# Shell completions for qsdev
if command -v qsdev >/dev/null 2>&1; then
  if [ -n "${ZSH_VERSION:-}" ]; then
    eval "$(qsdev completion zsh)"
  elif [ -n "${BASH_VERSION:-}" ]; then
    eval "$(qsdev completion bash)"
  fi
fi`, prefix)
}

// buildEnterTestScript returns the shell script body for devenv.nix enterTest.
func buildEnterTestScript() string {
	return `echo "=== Security Validation ==="

# 1. Verify pre-commit hooks are installed
if [ -d .git ]; then
  test -f .git/hooks/pre-commit || {
    echo "FAIL: pre-commit hooks not installed"
    exit 1
  }
  echo "PASS: pre-commit hooks installed"
fi

# 2. Verify credential variables are not in environment
for var in AWS_SECRET_ACCESS_KEY GITHUB_TOKEN VAULT_TOKEN DATABASE_PASSWORD; do
  if printenv "$var" >/dev/null 2>&1; then
    echo "FAIL: $var is set in the environment"
    exit 1
  fi
done
echo "PASS: no credential variables in environment"

# 3. Verify ripsecrets finds no issues in tracked files
if command -v ripsecrets >/dev/null 2>&1; then
  if ripsecrets --strict-ignore . 2>/dev/null; then
    echo "PASS: no secrets detected in codebase"
  else
    echo "FAIL: ripsecrets found potential secrets"
    exit 1
  fi
fi

# 4. Verify DEVENV_SECURITY_HARDENED flag is set
test "${DEVENV_SECURITY_HARDENED:-}" = "true" || {
  echo "FAIL: DEVENV_SECURITY_HARDENED not set (security config may be overridden)"
  exit 1
}
echo "PASS: security-hardened flag present"

echo "=== All security checks passed ==="`
}
