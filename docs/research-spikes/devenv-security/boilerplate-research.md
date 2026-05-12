# Devenv.sh Security-Hardened Boilerplate: Design & Rationale

## Overview

This document presents a production-ready, security-hardened devenv.sh boilerplate consisting of four files: `devenv.yaml`, `devenv.nix`, `devenv.local.nix.example`, and `.envrc`. Each setting is annotated with the threat it mitigates (referencing `security-surface-research.md` vector numbers), the tradeoff it imposes, and its classification as **MUST-HAVE**, **RECOMMENDED**, or **OPTIONAL**.

The boilerplate is designed as a copy-paste starting point for new projects. It embodies a "secure by default, weaken explicitly" philosophy: the committed configuration enforces the security baseline, and any weakening requires a visible code-review-gated change to a tracked file.

### Design Principles

1. **Defense in depth.** No single control is the only barrier. Clean environments, secret scanning, input pinning, and license controls each address different vectors.
2. **Visible weakening.** Settings that CANNOT be weakened without a tracked, reviewable change are marked MUST-HAVE. They live in committed files (`devenv.yaml`, `devenv.nix`), not in local overrides.
3. **Minimal friction.** Security controls that developers never notice (clean env, ripsecrets hook) are preferred over controls that require daily interaction.
4. **Layered overrides.** `devenv.local.nix` exists for developer comfort (editor preferences, extra debug tools) but cannot weaken the security model without being visible in `devenv.nix` assertions.

---

## File 1: devenv.yaml

```yaml
# =============================================================================
# devenv.yaml — Security-Hardened Configuration
# =============================================================================
# This file controls input sources, environment isolation, and package policy.
# Changes to this file affect the entire team and MUST be code-reviewed.
# =============================================================================

# ---------------------------------------------------------------------------
# INPUT PINNING
# ---------------------------------------------------------------------------
# [MUST-HAVE] Use upstream NixOS nixpkgs, not devenv-nixpkgs/rolling.
#
# Threat mitigated: Vector 1b (nixpkgs compromise), Vector 10c (devenv supply chain).
# devenv-nixpkgs/rolling is a Cachix-maintained fork with additional patches
# reviewed only by the Cachix team (~3 people). Upstream nixpkgs has ~139
# committers, Hydra CI gating, and broader community review.
#
# Tradeoff: Some packages may need local compilation instead of cache hits from
# devenv.cachix.org (which only caches devenv-nixpkgs/rolling builds). The
# official cache.nixos.org covers upstream nixpkgs extensively.
#
# To use the latest stable channel with security patches, pin to nixos-25.11.
# Update this when a new stable release ships (~May and November each year).
# Review the diff in devenv.lock after any update.
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixos-25.11

# ---------------------------------------------------------------------------
# ENVIRONMENT ISOLATION
# ---------------------------------------------------------------------------
# [MUST-HAVE] Strip inherited environment variables on shell entry.
#
# Threat mitigated: Vector 8a (secrets in environment), Vector 8c (impureEnvVars
# leakage). Without clean mode, the devenv shell inherits everything from the
# parent shell: AWS_SECRET_ACCESS_KEY, GITHUB_TOKEN, cloud provider credentials,
# CI tokens, and any other ambient secrets. These leak into all processes,
# build logs, and potentially the Nix store.
#
# Tradeoff: Some tools require specific env vars to function. Add them to
# clean.keep explicitly — each addition is a conscious, reviewable decision.
# The list below covers common necessities for terminal, display, and SSH.
clean:
  enabled: true
  keep:
    # Terminal rendering
    - TERM
    - COLORTERM
    # User identity (required by many tools)
    - HOME
    - USER
    - LOGNAME
    # Display server (required for GUI tools, browsers, electron apps)
    - DISPLAY
    - WAYLAND_DISPLAY
    - XDG_RUNTIME_DIR
    - XDG_SESSION_TYPE
    # SSH agent forwarding (required for git over SSH)
    - SSH_AUTH_SOCK
    # Locale (prevents encoding errors)
    - LANG
    - LC_ALL
    # Nix daemon communication (required for nix operations)
    - NIX_SSL_CERT_FILE
    - SSL_CERT_FILE

# [MUST-HAVE] Do not relax environment hermeticity.
#
# Threat mitigated: Vector 8c (impureEnvVars leakage). Impure mode allows
# the environment to depend on host system state, breaking reproducibility
# and making security posture vary per machine.
#
# Tradeoff: Some workflows (e.g., GPU development, system-level tooling) need
# host state. Use `devenv shell --impure` per-invocation rather than setting
# this globally.
impure: false

# ---------------------------------------------------------------------------
# PACKAGE POLICY
# ---------------------------------------------------------------------------
# [MUST-HAVE] Deny unfree packages by default.
#
# Threat mitigated: Vector 1a (malicious packages). Unfree packages cannot be
# source-audited. Their build processes are opaque.
#
# Tradeoff: If your project requires specific unfree packages (e.g., vscode,
# nvidia-x11), add them to permitted_unfree_packages with a comment explaining
# the business justification. Do NOT set allow_unfree: true.
nixpkgs:
  allow_unfree: false
  allow_broken: false

  # [RECOMMENDED] Allowlist specific unfree packages if needed.
  # Each entry is a conscious, reviewable decision.
  permitted_unfree_packages: []
    # Example:
    # - "vscode"

  # [MUST-HAVE] Never permit insecure packages without documented risk acceptance.
  # Nix marks packages as insecure when they have known, unpatched CVEs.
  # Adding a package here is accepting known vulnerability exposure.
  permitted_insecure_packages: []

  # [RECOMMENDED] Block licenses incompatible with your project.
  # Prevents accidental introduction of packages with problematic licenses.
  #
  # Threat mitigated: Compliance risk, not a security vector per se, but
  # license violations can force emergency dependency removal.
  #
  # Common blocklist entries (uncomment as appropriate):
  blocklisted_licenses: []
    # - "bsl11"          # Business Source License (time-delayed open source)
    # - "sspl"           # Server Side Public License (MongoDB)
    # - "elastic-2.0"    # Elastic License 2.0

# ---------------------------------------------------------------------------
# SECRETS MANAGEMENT
# ---------------------------------------------------------------------------
# [MUST-HAVE] Enable SecretSpec for declarative secrets.
#
# Threat mitigated: Vector 8a (secrets in store), Vector 8d (.env file leakage).
# SecretSpec separates secret declaration (committed secretspec.toml) from
# secret provisioning (provider-specific, not committed). Runtime loading via
# `secretspec run -- <command>` keeps secrets out of the shell environment
# entirely.
#
# Tradeoff: Developers must set up their secret provider (keyring, 1Password,
# etc.) once. This is a one-time cost vs. ongoing .env file risk.
#
# Provider options: keyring, 1password, dotenv, env, lastpass, gcloud-secret-manager
# Choose based on your team's existing secret management infrastructure.
secretspec:
  enable: true
  provider: keyring
  profile: development

# ---------------------------------------------------------------------------
# VERSION ENFORCEMENT
# ---------------------------------------------------------------------------
# [MUST-HAVE] Require minimum devenv version.
#
# Threat mitigated: Prevents environments from being built with outdated devenv
# versions that may lack security features (e.g., SecretSpec requires 1.8+,
# native activation requires 2.0+, shell support requires 2.1+).
#
# Tradeoff: Developers must keep devenv updated. This is desirable.
require_version: ">=2.1"
```

---

## File 2: devenv.nix

```nix
# =============================================================================
# devenv.nix — Security-Hardened Environment Definition
# =============================================================================
# This file defines the development environment: packages, hooks, scripts,
# and security controls. Changes MUST be code-reviewed.
#
# Security model: This file is the primary trust boundary. Everything defined
# here executes with the developer's full privileges (no sandbox at runtime).
# The controls below are compensating measures for that architectural reality.
# =============================================================================
{ pkgs, lib, config, ... }:

{
  # -------------------------------------------------------------------------
  # PACKAGES — Explicit, Minimal Package Set
  # -------------------------------------------------------------------------
  # [MUST-HAVE] Declare every package explicitly.
  #
  # Threat mitigated: Vector 1a (malicious packages). An explicit package list
  # is auditable. Each package traces to the pinned nixpkgs commit in
  # devenv.lock. Avoid wildcard patterns or large meta-packages.
  #
  # Tradeoff: Developers must add packages here (and get them reviewed) rather
  # than installing ad-hoc. This is the point — every binary in PATH is
  # intentional.
  packages = [
    pkgs.git
    pkgs.jq
    pkgs.curl
    pkgs.coreutils
    # === Add project-specific packages below this line ===

  ];

  # -------------------------------------------------------------------------
  # ENVIRONMENT VARIABLES — Non-Sensitive Only
  # -------------------------------------------------------------------------
  # [MUST-HAVE] Never put secrets in env.*
  #
  # Threat mitigated: Vector 8a (secrets in Nix store). Values set here become
  # part of the Nix store path, which is world-readable (dr-xr-xr-x). Any user
  # on the system can read them. They persist indefinitely.
  #
  # Use env.* ONLY for non-sensitive configuration.
  env = {
    # Project metadata
    DEVENV_SECURITY_HARDENED = "true";

    # Example non-sensitive settings:
    # EDITOR = "vim";
    # NODE_ENV = "development";
  };

  # -------------------------------------------------------------------------
  # UNSET DANGEROUS ENVIRONMENT VARIABLES
  # -------------------------------------------------------------------------
  # [MUST-HAVE] Remove credential-bearing variables from the environment.
  #
  # Threat mitigated: Vector 8a (secrets in environment), Vector 8c
  # (impureEnvVars leakage). Even with clean.enabled=true, if a variable is
  # in the clean.keep list OR is set by a Nix derivation/overlay, it could
  # appear. unsetEnvVars provides a second layer of defense: these variables
  # are explicitly removed after all other environment setup completes.
  #
  # The default unsetEnvVars already removes 26+ Nix build internals
  # (buildInputs, shellHook, stdenv, etc.). We ADD credential variables.
  #
  # Tradeoff: If your project legitimately needs AWS/GCP/Azure credentials in
  # the shell (not recommended), you must remove them from this list AND add
  # them to clean.keep in devenv.yaml. Both changes are reviewable.
  #
  # If you need cloud credentials, use secretspec runtime loading instead:
  #   secretspec run -- aws s3 ls
  unsetEnvVars = [
    # --- AWS ---
    "AWS_ACCESS_KEY_ID"
    "AWS_SECRET_ACCESS_KEY"
    "AWS_SESSION_TOKEN"
    "AWS_SECURITY_TOKEN"
    "AWS_DEFAULT_REGION"
    # --- GitHub ---
    "GITHUB_TOKEN"
    "GH_TOKEN"
    "GITHUB_PAT"
    # --- GitLab ---
    "GITLAB_TOKEN"
    "GL_TOKEN"
    # --- GCP ---
    "GOOGLE_APPLICATION_CREDENTIALS"
    "GCLOUD_PROJECT"
    "CLOUDSDK_CORE_PROJECT"
    # --- Azure ---
    "AZURE_CLIENT_ID"
    "AZURE_CLIENT_SECRET"
    "AZURE_TENANT_ID"
    "AZURE_SUBSCRIPTION_ID"
    # --- Generic secrets ---
    "NPM_TOKEN"
    "PYPI_TOKEN"
    "DOCKER_PASSWORD"
    "DOCKER_AUTH_CONFIG"
    "CACHIX_AUTH_TOKEN"
    "DATABASE_URL"
    "DATABASE_PASSWORD"
    "PGPASSWORD"
    "MYSQL_PWD"
    "REDIS_PASSWORD"
    "VAULT_TOKEN"
    "SENTRY_DSN"
    "STRIPE_SECRET_KEY"
    "SENDGRID_API_KEY"
    "SLACK_TOKEN"
    "SLACK_WEBHOOK_URL"
    "API_KEY"
    "API_SECRET"
    "SECRET_KEY"
    "PRIVATE_KEY"
    "ENCRYPTION_KEY"
  ];

  # -------------------------------------------------------------------------
  # DOTENV — Disabled
  # -------------------------------------------------------------------------
  # [MUST-HAVE] Disable .env file loading.
  #
  # Threat mitigated: Vector 8d (.env file leakage). .env files are the #1
  # cause of accidental credential commits. They are unencrypted, easy to
  # mishandle, and every tool loads them differently.
  #
  # Use SecretSpec instead (configured in devenv.yaml above).
  #
  # Tradeoff: Teams migrating from .env workflows need to set up a SecretSpec
  # provider. The dotenv provider exists as a migration bridge if needed,
  # but should be configured via SecretSpec, not via devenv's dotenv option.
  dotenv.enable = false;

  # -------------------------------------------------------------------------
  # PRE-COMMIT SECURITY HOOKS
  # -------------------------------------------------------------------------
  # [MUST-HAVE] Enable git hooks for automated security scanning.
  #
  # Threat mitigated: Vector 8a (secrets in code), Vector 7a (lock file
  # tampering). Pre-commit hooks are the last line of defense before secrets
  # or dangerous changes reach version control.
  #
  # Tradeoff: Hooks add ~1-3 seconds to each commit. This is negligible
  # compared to the cost of a credential leak.
  git-hooks.enable = true;
  git-hooks.hooks = {

    # --- Secret Detection (CRITICAL) ---
    # [MUST-HAVE] Scan for leaked secrets before every commit.
    #
    # ripsecrets is a Rust-based secret scanner that detects API keys, tokens,
    # passwords, and private keys using pattern matching. It is the only
    # secret scanner built into git-hooks.nix (gitleaks, trufflehog, and
    # detect-secrets require custom hook definitions).
    #
    # Threat mitigated: Vector 8a (secrets in code/store).
    ripsecrets.enable = true;

    # --- Large File Prevention ---
    # [RECOMMENDED] Prevent accidental commits of large files.
    #
    # Blocks binary blobs, data dumps, and other large files that shouldn't
    # be in version control. These can contain embedded credentials, inflate
    # repo size, and resist code review.
    check-added-large-files.enable = true;

    # --- Branch Protection ---
    # [RECOMMENDED] Prevent direct commits to protected branches.
    #
    # Forces all changes through pull requests where code review (the primary
    # defense for devenv.nix changes) can occur.
    #
    # Tradeoff: Developers cannot push hotfixes directly to main. This is
    # the point.
    no-commit-to-branch.enable = true;

    # --- Shell Script Security ---
    # [RECOMMENDED] Lint shell scripts for injection vulnerabilities.
    #
    # ShellCheck catches unquoted variables, command injection patterns,
    # and other common shell scripting mistakes that create security holes.
    shellcheck.enable = true;

    # --- Nix Anti-Pattern Detection ---
    # [RECOMMENDED] Catch Nix evaluation issues that could affect security.
    #
    # Statix identifies anti-patterns in .nix files including deprecated
    # constructs and patterns that could lead to unexpected evaluation
    # behavior.
    statix.enable = true;

    # --- Custom: Lock File Change Audit ---
    # [OPTIONAL] Flag devenv.lock changes for extra review attention.
    #
    # Threat mitigated: Vector 7a (lock file tampering). Lock file changes
    # redirect the entire package source. They deserve explicit review
    # attention. This hook prints a warning, not a block — the reviewer
    # decides.
    #
    # Tradeoff: Minor noise on legitimate lock updates. Worth it.
    lock-file-audit = {
      enable = true;
      name = "Lock file change audit";
      entry = "${pkgs.writeShellScript "lock-audit" ''
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
      ''}";
      language = "system";
      files = "(devenv|flake)\\.lock$";
      pass_filenames = true;
      stages = [ "pre-commit" ];
    };

    # --- Custom: Secrets in Nix Files ---
    # [OPTIONAL] Extra check for secrets patterns in .nix files specifically.
    #
    # ripsecrets covers general secret patterns, but this hook adds
    # devenv-specific checks: env.* assignments that look like secrets,
    # hardcoded URLs with credentials, etc.
    nix-secrets-check = {
      enable = true;
      name = "Nix file secrets check";
      entry = "${pkgs.writeShellScript "nix-secrets-check" ''
        ret=0
        for f in "$@"; do
          # Check for env.* assignments that look like secrets
          if ${pkgs.gnugrep}/bin/grep -nP 'env\.\w*(SECRET|TOKEN|PASSWORD|KEY|CREDENTIAL|API_KEY)\w*\s*=' "$f" 2>/dev/null; then
            echo "ERROR: $f appears to set a secret via env.*"
            echo "  Use SecretSpec instead: https://devenv.sh/integrations/secretspec/"
            ret=1
          fi
          # Check for hardcoded credential-like strings
          if ${pkgs.gnugrep}/bin/grep -nP '(sk_live_|sk_test_|ghp_|gho_|glpat-|AKIA[A-Z0-9]{16})' "$f" 2>/dev/null; then
            echo "ERROR: $f appears to contain a hardcoded credential"
            ret=1
          fi
        done
        exit $ret
      ''}";
      language = "system";
      files = "\\.nix$";
      pass_filenames = true;
      stages = [ "pre-commit" ];
    };
  };

  # -------------------------------------------------------------------------
  # SECRETSPEC DECLARATION
  # -------------------------------------------------------------------------
  # [MUST-HAVE] Declare required secrets in secretspec.toml (not here).
  #
  # The secretspec.toml file should be committed to git. It declares WHAT
  # secrets exist, not their values. Example secretspec.toml:
  #
  #   [profiles.development]
  #   DATABASE_URL = { description = "PostgreSQL connection string" }
  #   API_KEY = { description = "External API key (test mode)" }
  #
  # Developers provision values via their chosen provider:
  #   secretspec set DATABASE_URL --provider keyring
  #
  # Processes access secrets via runtime loading:
  #   secretspec run -- npm start
  #
  # This keeps secrets out of the shell environment entirely.

  # -------------------------------------------------------------------------
  # ENTER SHELL — Security Checks & Warnings
  # -------------------------------------------------------------------------
  # [RECOMMENDED] Print security posture on shell entry.
  #
  # This is NOT a security control (enterShell runs unsandboxed with full
  # user privileges — see Vector 3a). It is an awareness mechanism: developers
  # see the security status every time they enter the environment.
  #
  # Keep this minimal. Do not fetch remote resources. Do not modify files
  # tracked by git (causes re-evaluation loops — Vector 3d).
  enterShell = ''
    echo ""
    echo "=== Security-Hardened Development Environment ==="

    # Verify git hooks are installed
    if [ -d .git ] && [ ! -f .git/hooks/pre-commit ]; then
      echo "  WARNING: Pre-commit hooks not installed."
      echo "           Run 'devenv shell' to install them."
    else
      echo "  Pre-commit hooks: active"
    fi

    # Verify clean environment is working (spot-check)
    if [ -n "''${AWS_SECRET_ACCESS_KEY:-}" ]; then
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

    # SecretSpec status
    if command -v secretspec >/dev/null 2>&1; then
      echo "  Secret management: available (secretspec)"
    else
      echo "  INFO: secretspec CLI not found (install for runtime secret loading)"
    fi

    echo "================================================="
    echo ""
  '';

  # -------------------------------------------------------------------------
  # ENTER TEST — Security Validation
  # -------------------------------------------------------------------------
  # [RECOMMENDED] Validate security controls in CI via `devenv test`.
  #
  # These assertions verify the security model is intact. Run `devenv test`
  # in CI pipelines to catch misconfigurations.
  enterTest = ''
    echo "=== Security Validation ==="

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
    test "''${DEVENV_SECURITY_HARDENED:-}" = "true" || {
      echo "FAIL: DEVENV_SECURITY_HARDENED not set (security config may be overridden)"
      exit 1
    }
    echo "PASS: security-hardened flag present"

    echo "=== All security checks passed ==="
  '';

  # -------------------------------------------------------------------------
  # GENERATED FILES — Security-Critical Defaults
  # -------------------------------------------------------------------------
  # [MUST-HAVE] Generate .gitignore with credential patterns.
  #
  # Threat mitigated: Vector 8d (.env file leakage). Even if dotenv is
  # disabled, developers may create .env files out of habit. This .gitignore
  # prevents them from being committed.
  #
  # WARNING: This overwrites any existing .gitignore on every shell entry.
  # If your project needs a custom .gitignore, merge the security patterns
  # below into it and remove this files.* block.
  #
  # Tradeoff: Projects with existing .gitignore files must merge content.
  # Consider using files.".gitignore".text with lib.concatStringsSep to
  # append rather than replace.

  # Uncomment the block below if you want devenv to manage .gitignore.
  # Most projects will want to manage .gitignore manually and include
  # these patterns themselves.
  #
  # files.".gitignore".text = ''
  #   # === Security: credential and secret patterns ===
  #   .env
  #   .env.*
  #   !.env.example
  #   *.key
  #   *.pem
  #   *.p12
  #   *.pfx
  #   *.jks
  #   secrets/
  #   credentials/
  #   .secret
  #   .secrets
  #
  #   # === Devenv: local overrides (not committed) ===
  #   devenv.local.nix
  #   devenv.local.yaml
  #   .devenv/
  #   .devenv.flake.nix
  #
  #   # === Direnv cache ===
  #   .direnv/
  # '';

  # -------------------------------------------------------------------------
  # SCRIPTS — Security Utilities
  # -------------------------------------------------------------------------
  # [OPTIONAL] Convenience scripts for security operations.
  #
  # These use Nix store path references (${pkgs.foo}/bin/foo) to prevent
  # PATH hijacking (Vector 3b). A malicious PATH entry cannot override these.
  scripts = {
    # Scan for secrets in the entire repository
    check-secrets = {
      exec = ''
        echo "Scanning for secrets..."
        ${pkgs.ripsecrets}/bin/ripsecrets --strict-ignore "''${1:-.}"
      '';
      description = "Scan for leaked secrets in the repository";
    };

    # Audit the lock file inputs
    audit-inputs = {
      exec = ''
        echo "=== devenv.lock Input Audit ==="
        if [ -f devenv.lock ]; then
          echo "Lock file last modified: $(stat -c '%y' devenv.lock 2>/dev/null || stat -f '%Sm' devenv.lock 2>/dev/null)"
          echo ""
          echo "Pinned inputs:"
          ${pkgs.jq}/bin/jq -r '.nodes | to_entries[] | select(.key != "root") | "  \(.key): \(.value.locked.owner // "local")/\(.value.locked.repo // "N/A") @ \(.value.locked.rev // "N/A" | .[:12])"' devenv.lock 2>/dev/null || echo "  (could not parse lock file)"
        else
          echo "WARNING: No devenv.lock found. Run 'devenv update' to create one."
        fi
      '';
      description = "Audit pinned inputs in devenv.lock";
    };

    # Show current security posture
    security-status = {
      exec = ''
        echo "=== Devenv Security Status ==="
        echo ""
        echo "Environment:"
        echo "  Clean mode: ''${DEVENV_SECURITY_HARDENED:+enabled}"
        echo "  Impure: false (enforced via devenv.yaml)"
        echo ""
        echo "Credential variables (should all be empty):"
        for var in AWS_SECRET_ACCESS_KEY AWS_ACCESS_KEY_ID GITHUB_TOKEN GH_TOKEN VAULT_TOKEN DATABASE_URL DATABASE_PASSWORD NPM_TOKEN; do
          val="$(printenv "$var" 2>/dev/null || true)"
          if [ -n "$val" ]; then
            echo "  $var = [SET - WARNING]"
          else
            echo "  $var = [not set - OK]"
          fi
        done
        echo ""
        echo "Git hooks:"
        if [ -f .git/hooks/pre-commit ]; then
          echo "  pre-commit: installed"
        else
          echo "  pre-commit: NOT INSTALLED"
        fi
        echo ""
        echo "SecretSpec:"
        if command -v secretspec >/dev/null 2>&1; then
          echo "  CLI: available"
          if [ -f secretspec.toml ]; then
            echo "  Config: secretspec.toml present"
          else
            echo "  Config: secretspec.toml NOT FOUND"
          fi
        else
          echo "  CLI: not installed"
        fi
      '';
      description = "Display current security posture of the environment";
    };
  };

  # -------------------------------------------------------------------------
  # LANGUAGE / SERVICE / PROCESS CONFIGURATION
  # -------------------------------------------------------------------------
  # Add your project-specific language, service, and process configuration
  # below this line. Security notes:
  #
  # - languages.*: Generally safe. Language modules add packages from the
  #   pinned nixpkgs input.
  #
  # - services.*: Run unsandboxed as your user (Vector 9a). Ensure services
  #   bind to 127.0.0.1 only. Do not pass production credentials.
  #
  # - processes.*: No isolation (Vector 9a). Use `secretspec run --` for
  #   processes that need credentials. Do not put secrets in processes.*.env.
  #
  # - overlays: Each overlay can replace ANY package (Vector 1a). Audit
  #   carefully. Document why each overlay exists.

}
```

---

## File 3: devenv.local.nix.example

```nix
# =============================================================================
# devenv.local.nix.example — Developer Local Overrides
# =============================================================================
# Copy this file to devenv.local.nix for personal customizations.
# devenv.local.nix is NOT committed to version control.
#
# IMPORTANT: devenv.local.nix can override ANY setting from devenv.nix.
# This is powerful but dangerous. The security model depends on certain
# settings remaining intact.
#
# === SAFE TO OVERRIDE (developer comfort) ===
# - Adding packages (extra editors, debug tools, personal utilities)
# - Setting env.EDITOR, env.PAGER, or other preference variables
# - Adding personal scripts
# - Enabling additional language tools or services for local testing
#
# === BREAKS THE SECURITY MODEL (do NOT override) ===
# - dotenv.enable = true          → Re-enables .env file loading
# - git-hooks.enable = false      → Disables ALL pre-commit security scanning
# - git-hooks.hooks.ripsecrets.enable = false → Disables secret scanning
# - unsetEnvVars = []             → Re-exposes ALL credential variables
# - Overriding enterTest          → Disables CI security validation
#
# If you need to weaken a security control, propose the change in devenv.nix
# so the team can review it. Do not silently bypass controls locally.
# =============================================================================
{ pkgs, config, ... }:

{
  # --- SAFE: Additional packages for your workflow ---
  packages = [
    # pkgs.vim
    # pkgs.ripgrep
    # pkgs.fd
    # pkgs.htop
  ];

  # --- SAFE: Editor and tool preferences ---
  env = {
    # EDITOR = "nvim";
    # PAGER = "less -R";
  };

  # --- SAFE: Personal convenience scripts ---
  # scripts.my-tool = {
  #   exec = ''echo "Hello from my local script"'';
  #   description = "My personal helper script";
  # };

  # --- SAFE: Additional services for local testing ---
  # services.redis.enable = true;

  # ===========================================================================
  # WARNING: The settings below BREAK the security model.
  # Do NOT uncomment them without team discussion.
  # ===========================================================================

  # # DANGEROUS: Re-enables .env file loading (Vector 8d)
  # dotenv.enable = true;

  # # DANGEROUS: Disables pre-commit hooks (Vector 8a)
  # git-hooks.enable = false;

  # # DANGEROUS: Disables secret scanning (Vector 8a)
  # git-hooks.hooks.ripsecrets.enable = false;

  # # DANGEROUS: Clears credential variable blocklist (Vector 8a, 8c)
  # unsetEnvVars = [];
}
```

---

## File 4: .envrc

```bash
#!/usr/bin/env bash
# =============================================================================
# .envrc — Direnv Integration for devenv
# =============================================================================
# This file is the entry point for direnv-based shell activation.
#
# SECURITY NOTES:
# - direnv hashes this file and requires `direnv allow` on first use or change.
# - However, changes to devenv.nix do NOT require re-approval (Vector 5a).
# - Any change to devenv.nix takes effect on the next shell prompt render.
# - The primary defense is code review of devenv.nix changes.
#
# If using devenv 2.0+ native activation instead of direnv, this file is
# not needed. Native activation uses `devenv allow` / `devenv revoke`.
# =============================================================================

# Load devenv's direnv integration
eval "$(devenv direnvrc)"

# Activate the devenv environment
use devenv

# ---- Optional: Additional direnv security ----

# [OPTIONAL] Warn if devenv.nix was modified since last review.
# This adds a visual cue, but does not block activation.
# Uncomment if your team wants extra visibility.
#
# if [ -f .devenv-reviewed-hash ]; then
#   current_hash=$(sha256sum devenv.nix 2>/dev/null | cut -d' ' -f1)
#   reviewed_hash=$(cat .devenv-reviewed-hash 2>/dev/null)
#   if [ "$current_hash" != "$reviewed_hash" ]; then
#     log_status "WARNING: devenv.nix has changed since last review"
#     log_status "  Run: sha256sum devenv.nix > .devenv-reviewed-hash"
#   fi
# fi
```

---

## Supplementary: secretspec.toml (Example)

This file should be committed to version control. It declares WHAT secrets the project needs, not their values.

```toml
# =============================================================================
# secretspec.toml — Secret Declaration
# =============================================================================
# This file declares what secrets the project requires.
# Actual values are provisioned via the configured provider (keyring,
# 1password, etc.) and are NEVER stored in this file or version control.
#
# Set secrets:    secretspec set DATABASE_URL
# Use at runtime: secretspec run -- npm start
# =============================================================================

[profiles.development]
DATABASE_URL = { description = "PostgreSQL connection string", default = "postgresql://localhost/myapp_dev" }
# API_KEY = { description = "External API key (use test/sandbox key)" }
# SMTP_PASSWORD = { description = "Email service password" }

[profiles.test]
DATABASE_URL = { description = "Test database connection string", default = "postgresql://localhost/myapp_test" }

[profiles.production]
DATABASE_URL = { description = "Production PostgreSQL connection string" }
# API_KEY = { description = "External API key (production)" }
```

---

## Setting-by-Setting Rationale Summary

### devenv.yaml Settings

| Setting | Classification | Threat Mitigated | Tradeoff |
|---------|---------------|------------------|----------|
| `inputs.nixpkgs.url: github:NixOS/nixpkgs/nixos-25.11` | MUST-HAVE | V1b, V10c: upstream has broader review than devenv-nixpkgs/rolling | Some packages may need local compilation; no devenv.cachix.org cache hits |
| `clean.enabled: true` | MUST-HAVE | V8a, V8c: strips ambient credentials from parent shell | Must explicitly allowlist needed variables |
| `clean.keep: [TERM, HOME, ...]` | MUST-HAVE | N/A (companion to clean.enabled) | Each addition widens the pass-through; review carefully |
| `impure: false` | MUST-HAVE | V8c: prevents host state dependency | Some GPU/system workflows need `--impure` flag |
| `nixpkgs.allow_unfree: false` | MUST-HAVE | V1a: unfree packages cannot be source-audited | Must allowlist specific unfree packages if needed |
| `nixpkgs.allow_broken: false` | MUST-HAVE | Known-broken packages may have unpatched issues | None for most projects |
| `nixpkgs.permitted_insecure_packages: []` | MUST-HAVE | Prevents use of packages with known CVEs | Must explicitly accept risk for each exception |
| `nixpkgs.blocklisted_licenses: []` | RECOMMENDED | License compliance risk | Requires knowing which licenses to block |
| `secretspec.enable: true` | MUST-HAVE | V8a, V8d: separates secret declaration from provisioning | One-time provider setup per developer |
| `require_version: ">=2.1"` | MUST-HAVE | Outdated devenv may lack security features | Developers must keep devenv updated |

### devenv.nix Settings

| Setting | Classification | Threat Mitigated | Tradeoff |
|---------|---------------|------------------|----------|
| Explicit `packages` list | MUST-HAVE | V1a: auditable, minimal attack surface | Developers must request new packages via PR |
| `unsetEnvVars` (credential list) | MUST-HAVE | V8a, V8c: second-layer credential stripping | Must use secretspec for legitimate credential needs |
| `dotenv.enable = false` | MUST-HAVE | V8d: prevents .env file credential leakage | Must migrate to SecretSpec |
| `git-hooks.enable = true` | MUST-HAVE | V8a, V7a: automated security scanning gate | ~1-3s added to each commit |
| `ripsecrets.enable = true` | MUST-HAVE | V8a: last line of defense before secrets reach git | Very rare false positives |
| `check-added-large-files.enable = true` | RECOMMENDED | Prevents binary blob commits | May need size threshold tuning |
| `no-commit-to-branch.enable = true` | RECOMMENDED | Forces changes through code review | Cannot push directly to main |
| `shellcheck.enable = true` | RECOMMENDED | Catches shell injection patterns | May flag intentional patterns |
| `statix.enable = true` | RECOMMENDED | Catches Nix anti-patterns | May flag uncommon-but-valid patterns |
| `lock-file-audit` custom hook | OPTIONAL | V7a: flags lock file changes for review | Minor noise on legitimate updates |
| `nix-secrets-check` custom hook | OPTIONAL | V8a: devenv-specific secret pattern detection | Regex-based, may have false positives |
| `enterShell` security warnings | RECOMMENDED | Awareness of security posture | Adds ~5 lines of output on shell entry |
| `enterTest` security assertions | RECOMMENDED | CI validation of security controls | Requires running `devenv test` in CI |
| Security utility scripts | OPTIONAL | Developer convenience for security tasks | Adds commands to PATH |

### .envrc Settings

| Setting | Classification | Threat Mitigated | Tradeoff |
|---------|---------------|------------------|----------|
| `eval "$(devenv direnvrc)"` + `use devenv` | MUST-HAVE (if using direnv) | Standard integration | Changes to devenv.nix bypass direnv approval (V5a) |
| Optional hash-check warning | OPTIONAL | V5a, V5d: visual cue for devenv.nix changes | Manual hash update step after review |

---

## What This Boilerplate Does NOT Protect Against

Understanding limitations is critical to a honest security model:

1. **Compromised developer machine.** If an attacker has shell access to a developer's workstation, `devenv.local.nix` can override everything. The boilerplate cannot protect against this — it is an endpoint security problem, not a devenv problem.

2. **Malicious code passing code review.** The entire devenv security model converges on code review of `devenv.nix`, `devenv.yaml`, and `devenv.lock` changes. If a malicious change passes review, it executes with the developer's full privileges.

3. **Runtime process isolation.** Devenv provides NO sandboxing for `enterShell`, scripts, processes, or services (Vectors 3a, 9a). All runtime code executes as the developer user with full filesystem, network, and process access. The experimental bubblewrap sandbox (PR #2427) is not yet merged.

4. **Nix evaluation-time attacks.** `devenv.nix` is evaluated unsandboxed. A malicious Nix expression can use `builtins.readFile` to read any file the developer can read. This is a fundamental Nix architecture limitation.

5. **Binary cache provenance.** Cache signatures verify that a binary was signed by a trusted key, but do not prove it was built from the claimed source derivation (Vector 2c). This is a Nix ecosystem gap.

6. **Per-package CVE scanning.** `devenv.lock` pins nixpkgs commits, not individual packages. There is no built-in mechanism to scan for known vulnerabilities in the resolved package set. Tools like `vulnix` and `sbomnix` can be integrated but are not part of this boilerplate (see P2-T5 for CI integration guidance).

---

## Companion Requirements: System-Level nix.conf

The boilerplate above hardens the devenv project configuration. Several critical security settings must be enforced at the Nix daemon level (system-wide) because devenv cannot set them:

```nix
# NixOS configuration.nix (or /etc/nix/nix.conf for non-NixOS)
{
  nix.settings = {
    # [MUST-HAVE] Enable build sandbox
    sandbox = true;

    # [MUST-HAVE] Require cryptographic signatures on all substitutions
    require-sigs = true;

    # [MUST-HAVE] Only trust root for daemon operations
    # Do NOT add regular users or @wheel — trusted-users is root-equivalent
    trusted-users = [ "root" ];

    # [MUST-HAVE] Allowlist binary caches that unprivileged users may enable
    trusted-substituters = [
      "https://cache.nixos.org"
      # Add devenv.cachix.org ONLY if you use devenv-nixpkgs/rolling:
      # "https://devenv.cachix.org"
    ];

    # [MUST-HAVE] Allowlist signing keys
    trusted-public-keys = [
      "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
      # "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw="
    ];

    # [RECOMMENDED] Filter dangerous syscalls in sandbox
    filter-syscalls = true;

    # [RECOMMENDED] Do not fall back to unsandboxed builds
    sandbox-fallback = false;
  };
}
```

See P2-T2 (system-level nix.conf hardening guide) for the complete treatment.

---

## Deployment Checklist

When adopting this boilerplate in a new project:

1. Copy `devenv.yaml`, `devenv.nix`, `.envrc` to the project root
2. Copy `devenv.local.nix.example` to the project root
3. Create `secretspec.toml` with your project's secret declarations
4. Run `devenv shell` to generate `devenv.lock` (commit it)
5. Run `devenv test` to verify security controls pass
6. Add `devenv test` to your CI pipeline
7. Verify system-level `nix.conf` settings on all developer machines
8. Brief the team on the trust model (see P2-T4)

---

## Sources

- Phase 1 research reports: `architecture-research.md`, `security-surface-research.md`, `config-options-research.md`, `nix-security-mechanisms-research.md`, `prior-art-research.md`
- `docs/devenv-top-level-nix-unsetenvvars.md` — unsetEnvVars defaults
- `docs/devenv-yaml-options-complete-2026.md` — devenv.yaml options reference
- `docs/secretspec-integration.md` — SecretSpec integration docs
- `docs/secretspec-announcement.md` — SecretSpec architecture and providers
- `docs/nix-flake-checker-determinate.md` — Flake Checker validation checks
- `docs/devenv-direnv-integration-docs.md` — direnv integration setup
- NixOS 25.11 stable channel: [NixOS Status](https://status.nixos.org/)
- devenv 2.1 release: [devenv 2.1 blog post](https://devenv.sh/blog/2026/05/07/devenv-21-nix-with-zsh-fish-and-nushell-via-libghostty/)
