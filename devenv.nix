{ pkgs, lib, config, ... }:
{
  # Local customizations (not managed by qsdev). Create devenv.local.nix to
  # add packages, overlays, env vars, or any other devenv config for this machine.
  imports = lib.optional (builtins.pathExists ./devenv.local.nix) ./devenv.local.nix;

  # Nixpkgs overlays
  overlays = [
    (import ./nix/go-overlay.nix)
  ];

  # Base packages
  packages = [ pkgs.git pkgs.jq pkgs.curl pkgs.coreutils pkgs.go-tools pkgs.govulncheck pkgs.gopls pkgs.golangci-lint pkgs.delve pkgs.goreleaser pkgs.gitleaks ] ++ [ (pkgs.writeShellScriptBin "semgrep" "exec -a osemgrep ''${pkgs.semgrep-core}/bin/semgrep-core --experimental \"$@\"") ];
  env = {
    DEVENV_SECURITY_HARDENED = "true";
    QSDEV_ECOSYSTEMS = "go";
    QSDEV_PROJECT_NAME = "qsdev";
    QSDEV_SECURITY_PROFILE = "enhanced";
    QSDEV_TOOL_COUNT = "22";
    QSDEV_VERSION = "v0.7.2-0.20260521220409-c97fcb68bdaa";
  };

  # Credential-bearing variables stripped from the shell
  unsetEnvVars = [ "AWS_ACCESS_KEY_ID" "AWS_SECRET_ACCESS_KEY" "AWS_SESSION_TOKEN" "AWS_SECURITY_TOKEN" "AWS_DEFAULT_REGION" "GITHUB_TOKEN" "GH_TOKEN" "GITHUB_PAT" "GITLAB_TOKEN" "GL_TOKEN" "GOOGLE_APPLICATION_CREDENTIALS" "GCLOUD_PROJECT" "CLOUDSDK_CORE_PROJECT" "AZURE_CLIENT_ID" "AZURE_CLIENT_SECRET" "AZURE_TENANT_ID" "AZURE_SUBSCRIPTION_ID" "NPM_TOKEN" "PYPI_TOKEN" "DOCKER_PASSWORD" "DOCKER_AUTH_CONFIG" "CACHIX_AUTH_TOKEN" "DATABASE_URL" "DATABASE_PASSWORD" "PGPASSWORD" "MYSQL_PWD" "REDIS_PASSWORD" "VAULT_TOKEN" "SENTRY_DSN" "STRIPE_SECRET_KEY" "SENDGRID_API_KEY" "SLACK_TOKEN" "SLACK_WEBHOOK_URL" "API_KEY" "API_SECRET" "SECRET_KEY" "PRIVATE_KEY" "ENCRYPTION_KEY" ];

  # Disable .env file loading for security
  dotenv.enable = false;

  # Go
  languages.go = {
    enable = true;
    package = pkgs.go_1_26;
  };

  # Enforce module-aware mode — prevents unvetted dependency additions
  env.GOFLAGS = "-mod=readonly";
  # Ensure all modules are verified via the Go checksum database
  env.GONOSUMCHECK = "";
  # Ensure all modules use the Go notary for transparency
  env.GONOSUMDB = "";

  # Git hooks — managed by prek (devenv 1.11+ default hook runner).
  # Tiers: baseline (always-on), enhanced (language-aware), specialized (custom)
  git-hooks.hooks = {
    # Baseline security hooks (always enabled)
    ripsecrets.enable = true;
    check-added-large-files.enable = true;
    no-commit-to-branch.enable = true;
    check-merge-conflicts.enable = true;
    shellcheck = {
      enable = true;
      excludes = [ "vendor/" ];
    };
    statix.enable = true;
    # Enhanced hooks (language-aware, from ecosystem modules)
    gofmt = {
      enable = true;
      excludes = [ "rules/core/testdata/" "vendor/" ];
    };
    govet = {
      enable = true;
      excludes = [ "rules/core/testdata/" "vendor/" ];
    };
    # Specialized hooks (custom definitions)
    staticcheck = {
      enable = true;
      name = "staticcheck";
      description = "Run staticcheck for advanced static analysis";
      entry = "${pkgs.go-tools}/bin/staticcheck ./...";
      language = "system";
      types = [ "go" ];
      stages = [ "pre-commit" ];
      pass_filenames = false;
      excludes = [ "vendor/" ];
    };
    govulncheck = {
      enable = true;
      name = "govulncheck";
      description = "Check for known vulnerabilities in Go dependencies";
      entry = "${pkgs.govulncheck}/bin/govulncheck ./...";
      language = "system";
      types = [ "go" ];
      stages = [ "pre-commit" ];
      pass_filenames = false;
      excludes = [ "vendor/" ];
    };
    lock-file-audit = {
      enable = true;
      name = "Lock file change audit";
      description = "Flag lock file changes for review — lock files redirect package sources";
      entry = toString (pkgs.writeShellScript "lock-audit" ''
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
      '');
      language = "system";
      stages = [ "pre-commit" ];
      files = "(devenv|flake)\\.lock$";
      pass_filenames = true;
    };
    nix-secrets-check = {
      enable = true;
      name = "Nix file secrets check";
      description = "Detect hardcoded secrets and credential patterns in .nix files";
      entry = toString (let
          envPattern = "env\.\w*(SECRET|TOKEN|PASSWORD|KEY|CREDENTIAL|API_KEY)\w*\s*=";
          credPattern = "(" + "sk_l" + "ive_" + "|" + "sk_t" + "est_" + "|" + "gh" + "p_" + "|" + "gh" + "o_" + "|" + "glp" + "at-" + "|" + "AKIA[A-Z" + "0-9]{16}" + ")";
        in
        pkgs.writeShellScript "nix-secrets-check" ''
          ret=0
          for f in "$@"; do
            if ${pkgs.gnugrep}/bin/grep -nP '${envPattern}' "$f" 2>/dev/null; then
              echo "ERROR: $f appears to set a secret via env.*"
              ret=1
            fi
            if ${pkgs.gnugrep}/bin/grep -nP '${credPattern}' "$f" 2>/dev/null; then
              echo "ERROR: $f appears to contain a hardcoded credential"
              ret=1
            fi
          done
          exit $ret
        '');
      language = "system";
      stages = [ "pre-commit" ];
      files = "\\.nix$";
      pass_filenames = true;
    };
  };

  enterShell = ''
    # Development task functions (generated by qsdev)
    qsdev-build() {
      go build ./...
    }
    qsdev-test() {
      go test ./...
    }
    qsdev-lint() {
      go vet ./...
      golangci-lint run
    }
    qsdev-format() {
      gofmt -l .
    }
    qsdev-security-scan() {
      semgrep --config auto --error .
      gitleaks detect --no-banner
    }

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

    echo "==================================================="
    echo ""
    echo "  ''${QSDEV_PROJECT_NAME:-unknown} | security: ''${QSDEV_SECURITY_PROFILE:-standard} | tools: ''${QSDEV_TOOL_COUNT:-0}"
    echo ""

    # Shell completions for qsdev
    if command -v qsdev >/dev/null 2>&1; then
      if [ -n "''${ZSH_VERSION:-}" ]; then
        eval "$(qsdev completion zsh)"
      elif [ -n "''${BASH_VERSION:-}" ]; then
        eval "$(qsdev completion bash)"
      fi
    fi
  '';

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
}
