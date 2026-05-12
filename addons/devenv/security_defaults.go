package devenv

// defaultUnsetEnvVars is the canonical list of credential-bearing environment
// variables that are explicitly stripped from the devenv shell. This provides
// a second layer of defense beyond clean.enabled in devenv.yaml.
var defaultUnsetEnvVars = []string{
	// AWS
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"AWS_SECURITY_TOKEN",
	"AWS_DEFAULT_REGION",
	// GitHub
	"GITHUB_TOKEN",
	"GH_TOKEN",
	"GITHUB_PAT",
	// GitLab
	"GITLAB_TOKEN",
	"GL_TOKEN",
	// GCP
	"GOOGLE_APPLICATION_CREDENTIALS",
	"GCLOUD_PROJECT",
	"CLOUDSDK_CORE_PROJECT",
	// Azure
	"AZURE_CLIENT_ID",
	"AZURE_CLIENT_SECRET",
	"AZURE_TENANT_ID",
	"AZURE_SUBSCRIPTION_ID",
	// Package registries
	"NPM_TOKEN",
	"PYPI_TOKEN",
	// Docker
	"DOCKER_PASSWORD",
	"DOCKER_AUTH_CONFIG",
	// Nix
	"CACHIX_AUTH_TOKEN",
	// Databases
	"DATABASE_URL",
	"DATABASE_PASSWORD",
	"PGPASSWORD",
	"MYSQL_PWD",
	"REDIS_PASSWORD",
	// Secrets management
	"VAULT_TOKEN",
	// Third-party services
	"SENTRY_DSN",
	"STRIPE_SECRET_KEY",
	"SENDGRID_API_KEY",
	// Communication
	"SLACK_TOKEN",
	"SLACK_WEBHOOK_URL",
	// Generic
	"API_KEY",
	"API_SECRET",
	"SECRET_KEY",
	"PRIVATE_KEY",
	"ENCRYPTION_KEY",
}

// defaultSecurityHooks lists the built-in git-hooks.nix hooks that are always
// enabled for security scanning.
var defaultSecurityHooks = []string{
	"ripsecrets",
	"check-added-large-files",
	"no-commit-to-branch",
	"check-merge-conflict",
	"shellcheck",
	"statix",
}

// defaultBasePackages is the minimal set of packages always included.
var defaultBasePackages = []string{
	"git",
	"jq",
	"curl",
	"coreutils",
}

// defaultCleanKeep is the allowlist of environment variables that pass through
// when clean.enabled is true.
var defaultCleanKeep = []string{
	"TERM",
	"COLORTERM",
	"HOME",
	"USER",
	"LOGNAME",
	"DISPLAY",
	"WAYLAND_DISPLAY",
	"XDG_RUNTIME_DIR",
	"XDG_SESSION_TYPE",
	"SSH_AUTH_SOCK",
	"LANG",
	"LC_ALL",
	"NIX_SSL_CERT_FILE",
	"SSL_CERT_FILE",
}

// defaultSpecializedHooks returns the specialized custom security hooks that are
// always present. These go beyond the baseline built-in hooks and use custom
// Nix expressions for advanced checks.
func defaultSpecializedHooks() []CustomHookData {
	return []CustomHookData{
		{
			ID:          "lock-file-audit",
			Name:        "Lock file change audit",
			Description: "Flag lock file changes for review — lock files redirect package sources",
			Entry: `${pkgs.writeShellScript "lock-audit" ''
        for f in "$@"; do
          case "$f" in
            devenv.lock|flake.lock)
              echo "WARNING: $f has been modified."
              echo "  Lock file changes redirect package sources."
              echo "  Verify the diff carefully during code review."
              echo "  Run 'nix flake metadata' to inspect resolved inputs."
              ;;
          esac
        done
      ''}`,
			RawEntry:      true,
			Language:      "system",
			Files:         `(devenv|flake)\.lock$`,
			PassFilenames: true,
			Stages:        []string{"pre-commit"},
		},
		{
			ID:          "nix-secrets-check",
			Name:        "Nix file secrets check",
			Description: "Detect hardcoded secrets and credential patterns in .nix files",
			Entry: `${pkgs.writeShellScript "nix-secrets-check" ''
        ret=0
        for f in "$@"; do
          if ${pkgs.gnugrep}/bin/grep -nP 'env\.\w*(SECRET|TOKEN|PASSWORD|KEY|CREDENTIAL|API_KEY)\w*\s*=' "$f" 2>/dev/null; then
            echo "ERROR: $f appears to set a secret via env.*"
            ret=1
          fi
          if ${pkgs.gnugrep}/bin/grep -nP '(sk_live_|sk_test_|ghp_|gho_|glpat-|AKIA[A-Z0-9]{16})' "$f" 2>/dev/null; then
            echo "ERROR: $f appears to contain a hardcoded credential"
            ret=1
          fi
        done
        exit $ret
      ''}`,
			RawEntry:      true,
			Language:      "system",
			Files:         `\.nix$`,
			PassFilenames: true,
			Stages:        []string{"pre-commit"},
		},
	}
}

// buildEnterShellScript returns the shell script body for devenv.nix enterShell.
// This runs on every shell entry and provides security posture awareness.
// Shell variable references use ${VAR} syntax; the caller applies nixMultiline
// escaping before embedding in a Nix '' ... '' string.
func buildEnterShellScript() string {
	return `echo ""
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
echo ""`
}

// buildEnterTestScript returns the shell script body for devenv.nix enterTest.
// This runs during 'devenv test' and validates security controls in CI.
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
