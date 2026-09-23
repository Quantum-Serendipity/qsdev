package devenv

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// defaultUnsetEnvVars returns the canonical list of credential-bearing
// environment variables stripped from the devenv shell.
func defaultUnsetEnvVars() []string {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	return cat.UnsetVars()
}

// defaultSecurityHooks returns the built-in git-hooks.nix hooks that are
// always enabled for security scanning.
func defaultSecurityHooks() []string {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	return cat.SecurityHooks()
}

// defaultBasePackages returns the minimal set of packages always included.
func defaultBasePackages() []string {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	return cat.BasePackages()
}

// defaultCleanKeep returns the allowlist of environment variables that pass
// through when clean.enabled is true.
func defaultCleanKeep() []string {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	return cat.KeepVars()
}

// defaultToolNixPackages returns the tool→Nix package map from the catalog.
func defaultToolNixPackages() map[string]string {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	return cat.ToolNixPackages()
}

// defaultToolNixExprs returns the tool→Nix expression map from the catalog.
func defaultToolNixExprs() map[string]string {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	return cat.ToolNixExprs()
}

// defaultSpecializedHooks returns the specialized custom security hooks that
// are always present. These use custom Nix expressions for advanced checks.
func defaultSpecializedHooks() []CustomHookData {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
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
			hook.Entry = fmt.Sprintf("pkgs.writeShellScript \"lock-audit\" ''\n%s\n      ''",
				indentBlock(strings.TrimSpace(def.Entry), "        "))
			hook.RawEntry = true
			hook.NeedsToString = true

		case "nix-secrets-check":
			hook.Entry = buildNixSecretsCheckEntry(def)
			hook.RawEntry = true
			hook.NeedsToString = true

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
	// The patterns are PCRE regexes full of backslashes (\., \w, \s). Inside a
	// Nix double-quoted string an unescaped "\w" evaluates to a plain "w",
	// which silently turns the regex into one that matches nothing, so every
	// piece is Nix-escaped before it is placed between quotes.
	envPattern := nixStr(def.EnvPattern)

	// Build credential pattern using Nix string concatenation so the
	// generated devenv.nix doesn't contain literal credential prefixes
	// that would trigger the hook's own scanner.
	var nixFragments []string
	for _, p := range def.CredentialPatterns {
		mid := len(p) / 2
		if mid == 0 {
			mid = 1
		}
		nixFragments = append(nixFragments, nixStr(p[:mid])+" + "+nixStr(p[mid:]))
	}
	credPatternExpr := `"(" + ` + strings.Join(nixFragments, ` + "|" + `) + ` + ")"`

	return fmt.Sprintf(`let
          envPattern = %s;
          credPattern = %s;
        in
        pkgs.writeShellScript "nix-secrets-check" ''
          ret=0
          for f in "$@"; do
            if ${pkgs.gnugrep}/bin/grep -nP -e ${lib.escapeShellArg envPattern} "$f" 2>/dev/null; then
              echo "ERROR: $f appears to set a secret via env.*"
              ret=1
            fi
            if ${pkgs.gnugrep}/bin/grep -nP -e ${lib.escapeShellArg credPattern} "$f" 2>/dev/null; then
              echo "ERROR: $f appears to contain a hardcoded credential"
              ret=1
            fi
          done
          exit $ret
        ''`, envPattern, credPatternExpr)
}

// leakSpotCheckVars are the credential variables enterShell and enterTest
// probe to confirm the unset list took effect. Only those still in the
// generated unset list are probed: a variable a service sets on purpose (e.g.
// MinIO's AWS_SECRET_ACCESS_KEY) is expected to be present.
var leakSpotCheckVars = []string{"AWS_SECRET_ACCESS_KEY", "VAULT_TOKEN", "DATABASE_PASSWORD"}

// spotCheckVars returns the leakSpotCheckVars that appear in unset.
func spotCheckVars(unset []string) []string {
	var out []string
	for _, v := range leakSpotCheckVars {
		if slices.Contains(unset, v) {
			out = append(out, v)
		}
	}
	return out
}

// hooksStateShell resolves the hooks directory the way git does, so the check
// works in linked worktrees and submodules (where .git is a file) and honours
// core.hooksPath. It sets hooks_state to "active", "missing", or "unknown"
// when the directory is not inside a git repository.
const hooksStateShell = `hooks_state=unknown
if command -v git >/dev/null 2>&1 && hooks_dir="$(git rev-parse --git-path hooks 2>/dev/null)"; then
  if [ -f "$hooks_dir/pre-commit" ]; then
    hooks_state=active
  else
    hooks_state=missing
  fi
fi
unset hooks_dir`

// mcpSecretsShell re-injects MCP credentials after clean-mode stripping, but
// only variables the project's .mcp.json references as ${NAME}: a project
// without a server that needs GITHUB_TOKEN never gets the user's gh token in
// its environment. .qsdev/mcp-secrets.env is parsed as KEY=VALUE data, never
// sourced as shell code. It can likewise only supply referenced names, never
// overrides a variable that is already set (PATH, HOME, ...), and never sets
// dynamic-loader or shell-startup variables (LD_PRELOAD, BASH_ENV, ...).
const mcpSecretsShell = `# MCP credential provisioning — re-inject after clean mode stripping, limited
# to the variables .mcp.json references as placeholders.
mcp_refs() {
  [ -f "$PWD/.mcp.json" ] && grep -Eq "[$][{]$1(:-[^}]*)?[}]" "$PWD/.mcp.json"
}

# GITHUB_TOKEN: sourced from gh CLI keyring (never touches disk).
if mcp_refs GITHUB_TOKEN && command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
  export GITHUB_TOKEN="$(gh auth token 2>/dev/null)"
fi

# Project-scoped MCP secrets (.qsdev/ is gitignored), read as KEY=VALUE lines.
if [ -f "$PWD/.qsdev/mcp-secrets.env" ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%$'\r'}"
    case "$line" in ""|"#"*) continue ;; esac
    line="${line#export }"
    key="${line%%=*}"
    val="${line#*=}"
    if [ "$key" = "$line" ] || ! [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
      echo "  WARNING: ignoring malformed line in .qsdev/mcp-secrets.env"
      continue
    fi
    if ! mcp_refs "$key"; then
      echo "  WARNING: ignoring $key from .qsdev/mcp-secrets.env (not referenced by .mcp.json)"
      continue
    fi
    case "$key" in
      LD_*|DYLD_*|BASH_ENV|ENV|BASH_FUNC_*|PROMPT_COMMAND)
        echo "  WARNING: ignoring $key from .qsdev/mcp-secrets.env (affects program loading)"
        continue ;;
    esac
    if [ -n "${!key+x}" ]; then
      echo "  WARNING: ignoring $key from .qsdev/mcp-secrets.env (already set)"
      continue
    fi
    case "$val" in
      \"*\"|\'*\') val="${val:1:${#val}-2}" ;;
    esac
    export "$key=$val"
  done < "$PWD/.qsdev/mcp-secrets.env"
  unset line key val
fi
unset -f mcp_refs`

// buildEnterShellScript returns the shell script body for devenv.nix
// enterShell. unset is the generated unsetEnvVars list.
func buildEnterShellScript(unset []string) string {
	prefix := branding.Get().EnvPrefix

	var spotCheck strings.Builder
	for _, v := range spotCheckVars(unset) {
		fmt.Fprintf(&spotCheck, `if [ -n "${%[1]s:-}" ]; then
  echo "  WARNING: %[1]s is set in environment!"
  echo "           This should have been stripped by clean mode."
  echo "           Check devenv.yaml clean.keep settings."
fi
`, v)
	}

	return `echo ""
echo "=== Security-Hardened Development Environment ==="

# Verify git hooks are installed
` + hooksStateShell + `
case "$hooks_state" in
  active) echo "  Pre-commit hooks: active" ;;
  missing)
    echo "  WARNING: Pre-commit hooks not installed."
    echo "           Run 'devenv shell' to install them."
    ;;
  *) echo "  Pre-commit hooks: unknown (not a git repository)" ;;
esac
unset hooks_state

# Verify clean environment is working (spot-check)
` + spotCheck.String() + `
# Verify ripsecrets is available
if command -v ripsecrets >/dev/null 2>&1; then
  echo "  Secret scanning: available (ripsecrets)"
else
  echo "  WARNING: ripsecrets not found in PATH"
fi

` + mcpSecretsShell + `

echo "==================================================="
echo ""
` + fmt.Sprintf(`echo "  ${%[1]sPROJECT_NAME:-unknown} | security: ${%[1]sSECURITY_PROFILE:-standard} | tools: ${%[1]sTOOL_COUNT:-0}"`, prefix) + `
echo ""

# Shell completions for qsdev
if command -v qsdev >/dev/null 2>&1; then
  if [ -n "${ZSH_VERSION:-}" ]; then
    eval "$(qsdev completion zsh)"
  elif [ -n "${BASH_VERSION:-}" ]; then
    eval "$(qsdev completion bash)"
  fi
fi`
}

// buildEnterTestScript returns the shell script body for devenv.nix
// enterTest. unset is the generated unsetEnvVars list.
func buildEnterTestScript(unset []string) string {
	leakCheck := `echo "PASS: no ambient credential leakage"`
	if vars := spotCheckVars(unset); len(vars) > 0 {
		leakCheck = `for var in ` + strings.Join(vars, " ") + `; do
  if printenv "$var" >/dev/null 2>&1; then
    echo "FAIL: $var is set in the environment"
    exit 1
  fi
done
` + leakCheck
	}

	return `echo "=== Security Validation ==="

# 1. Verify pre-commit hooks are installed (worktree- and hooksPath-aware)
` + hooksStateShell + `
case "$hooks_state" in
  active) echo "PASS: pre-commit hooks installed" ;;
  missing)
    echo "FAIL: pre-commit hooks not installed"
    exit 1
    ;;
  *) echo "SKIP: not a git repository; pre-commit hook check skipped" ;;
esac

# 2. Verify ambient credential leakage is prevented.
# Variables .mcp.json references (e.g. GITHUB_TOKEN) are re-injected on purpose.
` + leakCheck + `

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

// indentBlock prepends prefix to every line of text.
func indentBlock(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
