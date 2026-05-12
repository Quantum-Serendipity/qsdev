# Devenv.sh Configuration Options: Security Analysis

## Overview

This report catalogs every devenv.sh configuration knob relevant to security hardening, covering `devenv.nix` module options, `devenv.yaml` settings, Nix-level controls, pre-commit hooks, lock file pinning, container/process isolation, environment variable management, and shell hook execution. For each option: name, purpose, default, security implications, and concrete configuration examples.

**Key finding**: devenv provides a strong foundation for reproducible, auditable development environments, but most security-relevant features are **opt-in** rather than secure-by-default. A hardened boilerplate must explicitly enable clean environments, secret scanning hooks, license controls, input pinning discipline, and secretspec integration.

---

## 1. devenv.nix Options

### 1.1 packages

- **Option**: `packages`
- **Type**: list of derivation
- **Default**: `[]`
- **What it does**: Declares all packages available in the development environment's `$PATH`
- **Security implications**: This is the primary control over what binaries developers can execute. Every package added expands the attack surface. Packages come from the pinned nixpkgs input, so their provenance traces to a specific nixpkgs commit. Nix's content-addressed store means packages are integrity-verified, but the package set itself must be curated.
- **Hardening guidance**: Use an explicit allowlist of packages. Avoid `pkgs.buildEnv` patterns that pull in transitive dependencies blindly. Pin nixpkgs to a specific, audited commit.

```nix
{ pkgs, ... }: {
  packages = [
    pkgs.git
    pkgs.jq
    pkgs.curl
    # Explicit, minimal package set
  ];
}
```

### 1.2 env

- **Option**: `env`
- **Type**: attribute set (freeform submodule)
- **Default**: `{}`
- **What it does**: Sets environment variables in the shell environment
- **Security implications**: Environment variables are visible to all processes in the shell. Secrets placed here are exposed in `/proc/*/environ`, shell history, and process listings. Any package or script in the environment can read them.
- **Hardening guidance**: Never put secrets in `env`. Use secretspec (section 7) for credentials. Use `env` only for non-sensitive configuration like `EDITOR`, `LANG`, feature flags.

```nix
{
  env.EDITOR = "vim";
  env.NODE_ENV = "development";
  # NEVER: env.DATABASE_PASSWORD = "secret";
}
```

### 1.3 enterShell

- **Option**: `enterShell`
- **Type**: string (bash code)
- **Default**: `""`
- **What it does**: Bash code executed every time a developer enters the shell via `devenv shell` or direnv activation
- **Security implications**: Runs with full user permissions, no sandboxing. Has access to all environment variables, the full filesystem, and network. Malicious code here executes on every shell entry. Code is visible in `devenv.nix` (auditable if committed to git), but `devenv.local.nix` overrides are not committed and could contain anything.
- **Hardening guidance**: Keep `enterShell` minimal. Avoid fetching remote resources. Code review `devenv.nix` changes carefully. Consider that `devenv.local.nix` can override this without team visibility.

```nix
{
  enterShell = ''
    echo "Environment loaded: $(date)"
    # Verify expected tools are available
    command -v git >/dev/null || echo "WARNING: git not found"
  '';
}
```

### 1.4 enterTest

- **Option**: `enterTest`
- **Type**: string (bash code)
- **Default**: Detects and runs `.test.sh` if present
- **What it does**: Bash code executed during `devenv test`. Processes auto-start before tests and stop after.
- **Security implications**: Same execution model as `enterShell` -- no sandboxing, full user permissions. Tests run in the full environment. The `config.devenv.isTesting` flag allows conditional behavior, meaning the test environment can differ from development.
- **Hardening guidance**: Use `enterTest` to validate security controls are active (e.g., verify hooks are installed, check that forbidden packages are absent).

```nix
{
  enterTest = ''
    # Verify security hooks are installed
    test -f .git/hooks/pre-commit || exit 1
    # Verify no secrets in tracked files
    ripsecrets --strict .
  '';
}
```

### 1.5 scripts.*

- **Option**: `scripts.<name>.exec`, `scripts.<name>.packages`, `scripts.<name>.description`
- **Type**: string (script body), list of derivation, string
- **Default**: N/A (must be defined)
- **What it does**: Creates named executable scripts available in the shell. `scripts.<name>.packages` provides dependencies available only when that script runs.
- **Security implications**: Scripts run with full user permissions, no sandboxing. The `packages` attribute scopes dependencies per-script (good for least-privilege). Scripts are defined in `devenv.nix` and are auditable. Direct path references (`${pkgs.curl}/bin/curl`) prevent PATH hijacking.
- **Hardening guidance**: Use direct Nix store path references for security-critical tools. Use `scripts.<name>.packages` to limit what each script can access.

```nix
{
  scripts.check-secrets = {
    exec = ''
      ${pkgs.ripsecrets}/bin/ripsecrets --strict "$@"
    '';
    description = "Scan for leaked secrets";
  };
}
```

### 1.6 dotenv.*

- **Option**: `dotenv.enable`, `dotenv.filename`, `dotenv.disableHint`
- **Type**: boolean, string, boolean
- **Default**: `false`, `".env"`, `false`
- **What it does**: Loads environment variables from a `.env` file into the shell
- **Security implications**: `.env` files are a common vector for credential leakage (accidental git commits). When enabled, all variables from the file are injected into the shell environment, visible to all processes. The file is read at shell entry time, so any process can access the values via the environment.
- **Hardening guidance**: Prefer secretspec over dotenv. If dotenv must be used, ensure `.env` is in `.gitignore`, enable the `ripsecrets` pre-commit hook, and consider using `dotenv.filename` to use a non-standard name that's harder to accidentally commit.

```nix
{
  # PREFERRED: disable dotenv, use secretspec instead
  dotenv.enable = false;

  # IF dotenv is required:
  # dotenv.enable = true;
  # dotenv.filename = ".env.local";  # Non-standard name
}
```

### 1.7 git-hooks.*

- **Option**: `git-hooks.enable`, `git-hooks.hooks.<name>.enable`, `git-hooks.hooks.<name>.entry`, plus `files`, `types`, `excludes`, `language`, `pass_filenames`, `stages`, `settings`
- **Type**: boolean, string, etc.
- **Default**: `false` for `git-hooks.enable`
- **What it does**: Integrates pre-commit framework. Hooks run at commit time and during `devenv test`. The `.pre-commit-config.yaml` is auto-generated and not committed.
- **Security implications**: Hooks are the primary mechanism for automated security scanning. They run with full user permissions. The generated config is symlinked, not committed -- so hook configuration is reproducible via `devenv.nix`. Custom hooks can execute arbitrary code.
- **Security-relevant built-in hooks**:
  - **`ripsecrets`** -- Detects secrets and credentials in code before commit. **Critical for preventing credential leakage.**
  - **`reuse`** -- REUSE license compliance checking. Ensures all files have proper license headers.
  - **`check-added-large-files`** -- Prevents accidentally committing large files (binary blobs, data dumps).
  - **`no-commit-to-branch`** -- Branch protection (prevent direct commits to main/master).
  - **`shellcheck`** -- Shell script linting catches common security mistakes (unquoted variables, command injection patterns).
  - **`phpstan`**, **`psalm`** -- PHP static analysis (catches type errors, some security issues).
  - **`mypy`**, **`pyright`** -- Python type checking (catches type-related bugs).
  - **`clippy`** -- Rust linting (catches unsafe code patterns).
  - **`statix`** -- Nix anti-pattern detection.
  - **`typos`** -- Spell checker (catches typos in variable names that could cause bugs).

```nix
{
  git-hooks.enable = true;
  git-hooks.hooks = {
    # Secret detection (CRITICAL)
    ripsecrets.enable = true;

    # License compliance
    reuse.enable = true;

    # Prevent large file commits
    check-added-large-files.enable = true;

    # Branch protection
    no-commit-to-branch.enable = true;

    # Code quality that catches security issues
    shellcheck.enable = true;

    # Custom: dependency audit
    dependency-audit = {
      enable = true;
      name = "Dependency audit";
      entry = "${pkgs.writeShellScript "dep-audit" ''
        # Custom dependency checking logic
        echo "Checking dependencies..."
      ''}";
      language = "system";
      pass_filenames = false;
      stages = [ "pre-push" ];
    };
  };
}
```

### 1.8 processes.*

- **Option**: `processes.<name>.exec`, `.env`, `.cwd`, `.before`, `.after`, plus restart policies, readiness probes, socket activation, file watching, watchdog
- **Type**: various
- **Default**: N/A
- **What it does**: Defines long-running processes managed by a process supervisor (default: process-compose)
- **Security implications**: **No namespace or cgroup isolation.** Processes share the same user/environment space. All processes can see each other's environment variables, file descriptors, and network ports. Socket activation passes file descriptors via `LISTEN_FDS` (systemd-compatible but unauthenticated). `NOTIFY_SOCKET` paths are shared.
- **Hardening guidance**: Do not pass secrets via process environment variables -- use secretspec runtime loading instead. Be aware that all processes in a devenv share the same trust boundary.

```nix
{
  process.manager.implementation = "process-compose";
  processes.api = {
    exec = "secretspec run -- node server.js";  # Runtime secret injection
    cwd = "./backend";
  };
}
```

### 1.9 containers.*

- **Option**: `containers.<name>.name`, `.entrypoint`, `.copyToRoot`, `.registry`, `.defaultCopyArgs`, plus `container.isBuilding`
- **Type**: various
- **Default**: N/A
- **What it does**: Generates OCI container images from the devenv environment using Nix. Uses `skopeo` for registry operations.
- **Security implications**: Container images are built reproducibly via Nix (good for auditability). `copyToRoot` controls exactly what enters the image -- minimal images reduce attack surface. `container.isBuilding` allows conditional behavior (e.g., excluding dev tools from production images). However, no runtime security policies (seccomp, capabilities, read-only rootfs) are configured at the devenv level -- those are concerns for the container runtime.
- **Hardening guidance**: Use `container.isBuilding` to strip dev tools, debug utilities, and unnecessary packages from production images. Use explicit `copyToRoot` rather than copying the entire environment.

```nix
{
  containers.production = {
    name = "myapp";
    entrypoint = [ "${pkgs.nodejs}/bin/node" "server.js" ];
    copyToRoot = pkgs.buildEnv {
      name = "production-root";
      paths = [ pkgs.nodejs pkgs.cacert ];  # Minimal
    };
  };

  # Strip dev tools from container builds
  packages = lib.optionals (!config.container.isBuilding) [
    pkgs.vim
    pkgs.htop
  ];
}
```

### 1.10 files.*

- **Option**: `files.<name>.text`, `.json`, `.yaml`, `.toml`, `.ini`, `.executable`
- **Type**: string, attrs, attrs, attrs, attrs, boolean
- **Default**: N/A
- **What it does**: Generates files in the project directory with declarative content
- **Security implications**: Generated files are placed in the project directory with specified permissions. The `.executable` flag controls whether files are chmod +x. File contents are defined in `devenv.nix` (auditable). These files are regenerated on each shell entry, so they cannot be tampered with persistently if the Nix config is trusted.
- **Hardening guidance**: Use `files` to generate security-critical configs (`.gitignore` with `.env`, credential patterns). Set `.executable = false` unless explicitly needed.

```nix
{
  files.".gitignore".text = ''
    .env
    .env.*
    *.key
    *.pem
    secrets/
  '';
}
```

### 1.11 services.*

- **Option**: `services.<name>.enable` plus service-specific settings (40+ services available including postgres, mysql, redis, vault, nginx, etc.)
- **Type**: various
- **Default**: `false`
- **What it does**: Configures and runs development services
- **Security implications**: Services run as the current user with no additional isolation. Database services bind to localhost by default (good). Services like Vault provide security infrastructure that can be used for secret management during development. No TLS is configured by default for most services.
- **Hardening guidance**: Verify services bind only to localhost. Use Vault service for development secret management. Do not use production credentials with development services.

### 1.12 overlays

- **Option**: `overlays`
- **Type**: list of overlay functions
- **Default**: `[]`
- **What it does**: Applies Nix overlays to modify the package set (requires devenv 1.4.2+)
- **Security implications**: Overlays can replace any package in nixpkgs with modified versions. A malicious overlay could substitute compromised binaries. Overlays are defined in `devenv.nix` (auditable) but imported overlays from external inputs need trust verification.
- **Hardening guidance**: Audit all overlays. Prefer overlays from trusted, pinned inputs. Document why each overlay exists.

### 1.13 unsetEnvVars

- **Option**: `unsetEnvVars`
- **Type**: list of string
- **Default**: 25+ build-related variables (`buildInputs`, `shellHook`, `strictDeps`, etc.)
- **What it does**: Removes Nix build-related variables from the shell environment
- **Security implications**: Reduces information leakage about the build system. Default list removes internal Nix variables that could be confusing or exploitable.
- **Hardening guidance**: Leave defaults in place. Add additional variables if custom build systems leak sensitive paths.

---

## 2. devenv.yaml Options

### 2.1 inputs

- **Option**: `inputs`, `inputs.<name>.url`, `inputs.<name>.flake`, `inputs.<name>.follows`, `inputs.<name>.overlays`
- **Type**: attribute set
- **Default**: `inputs.nixpkgs.url: github:cachix/devenv-nixpkgs/rolling`
- **What it does**: Declares Nix inputs (sources of packages and modules). The nixpkgs input determines the entire package set available.
- **Security implications**: **This is the single most important security control.** The input URL determines which package repository is trusted. `github:cachix/devenv-nixpkgs/rolling` is a Cachix-maintained fork that tracks nixpkgs-unstable. Using `follows` to share inputs reduces the number of independent trust decisions. External inputs from third-party flakes expand the trust boundary.
- **Hardening guidance**: Pin inputs to specific commits (done automatically via `devenv.lock`). Audit input URLs. Minimize third-party inputs. Use `follows` to reduce independent trust decisions.

```yaml
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixos-24.11  # Stable, security-patched channel
  # Minimize third-party inputs
```

### 2.2 nixpkgs.allow_unfree / permit controls

- **Option**: `nixpkgs.allow_unfree`, `nixpkgs.allow_broken`, `nixpkgs.allow_unsupported_system`, `nixpkgs.permitted_unfree_packages`, `nixpkgs.permitted_insecure_packages`, `nixpkgs.allowlisted_licenses`, `nixpkgs.blocklisted_licenses`
- **Type**: boolean / list of string
- **Default**: all `false` / all `[]`
- **What it does**: Controls which categories of packages are allowed to be installed
- **Security implications**:
  - `allow_unfree = true` permits proprietary packages where source code cannot be audited
  - `allow_broken = true` permits packages marked as broken (may have known issues)
  - `permitted_insecure_packages` explicitly allows packages with known CVEs -- Nix will refuse to install these without explicit permission
  - License allowlisting/blocklisting enforces organizational license compliance
- **Hardening guidance**: Keep all defaults (`false`/`[]`). If unfree packages are needed, use `permitted_unfree_packages` to allowlist specific packages rather than blanket `allow_unfree = true`. Never use `permitted_insecure_packages` without documenting the risk acceptance. Use `blocklisted_licenses` to enforce organizational policy.

```yaml
nixpkgs:
  allow_unfree: false
  allow_broken: false
  permitted_unfree_packages:
    - "vscode"  # Specific allowlist, not blanket permission
  blocklisted_licenses:
    - "unfreeRedistributable"
    - "bsl11"
```

### 2.3 clean.*

- **Option**: `clean.enabled`, `clean.keep`
- **Type**: boolean, list of string
- **Default**: `false`, `[]`
- **What it does**: When enabled, strips all inherited environment variables when entering the shell, creating a hermetic environment. Only variables in `clean.keep` survive.
- **Security implications**: **Critical for environment isolation.** Without `clean`, the devenv shell inherits all variables from the parent shell, including potentially sensitive credentials (`AWS_SECRET_ACCESS_KEY`, `GITHUB_TOKEN`, etc.) and configuration that could affect build behavior (`CC`, `CFLAGS`, `LD_LIBRARY_PATH`). With `clean.enabled = true`, the environment is deterministic and free from ambient credential leakage.
- **Hardening guidance**: Enable `clean` in all security-conscious environments. Explicitly list only the variables that need to pass through (e.g., `TERM`, `HOME`, `USER`, `DISPLAY`, `WAYLAND_DISPLAY` for GUI apps).

```yaml
clean:
  enabled: true
  keep:
    - TERM
    - HOME
    - USER
    - DISPLAY
    - WAYLAND_DISPLAY
    - XDG_RUNTIME_DIR
    - SSH_AUTH_SOCK  # Only if needed
```

### 2.4 impure

- **Option**: `impure`
- **Type**: boolean
- **Default**: `false`
- **What it does**: Relaxes hermeticity of the environment, allowing access to host system state
- **Security implications**: When `true`, the environment can depend on host state (installed packages, system configuration). This breaks reproducibility and means the security posture varies per machine. Builds may succeed on one machine but fail on another, making security auditing unreliable.
- **Hardening guidance**: Keep `false`. If impure mode is needed for specific workflows, use the `--impure` CLI flag rather than setting it in config, so it's a conscious per-invocation decision.

```yaml
impure: false  # Default, but make it explicit
```

### 2.5 imports

- **Option**: `imports`
- **Type**: list of string (paths)
- **Default**: `[]`
- **What it does**: Imports additional `devenv.nix` and `devenv.yaml` from relative paths, absolute paths, or input references
- **Security implications**: Imported configurations merge into the current environment. They can add packages, scripts, hooks, and environment variables. Currently remote imports are not supported (only local paths and input references), which limits the remote code execution surface. However, input references (e.g., `my-input/path/to/module`) execute Nix code from those inputs.
- **Hardening guidance**: Audit all imported modules. Pin inputs that provide importable modules. Review what packages and hooks imported modules add.

```yaml
imports:
  - ./security   # Local security hardening module
  - ./frontend
```

### 2.6 secretspec.*

- **Option**: `secretspec.enable`, `secretspec.provider`, `secretspec.profile`
- **Type**: boolean, string, string
- **Default**: `false`, N/A, N/A
- **What it does**: Enables SecretSpec integration for declarative secrets management. Secrets are declared in `secretspec.toml` and sourced from configurable providers (keyring, 1Password, dotenv, env vars, LastPass, Google Cloud Secret Manager).
- **Security implications**: **This is devenv's recommended approach for secrets.** Runtime loading via `secretspec run -- <command>` keeps secrets out of the shell environment entirely. Provider abstraction means development can use keyring/dotenv while CI/production uses cloud secret managers. Secret declarations are committed to git (what secrets are needed) while values stay in providers (where secrets are stored).
- **Hardening guidance**: Enable secretspec, use runtime loading (`secretspec run`), prefer keyring or 1Password over dotenv provider.

```yaml
secretspec:
  enable: true
  provider: keyring
  profile: development
```

### 2.7 require_version

- **Option**: `require_version`
- **Type**: boolean or string
- **Default**: N/A (not set)
- **What it does**: Enforces a minimum devenv CLI version. `true` requires CLI-module compatibility. A string like `">=2.1"` applies explicit version constraints.
- **Security implications**: Prevents environments from being built with outdated devenv versions that may have known vulnerabilities or missing security features.
- **Hardening guidance**: Set to a version that includes all security features you depend on.

```yaml
require_version: ">=2.1"
```

### 2.8 reload

- **Option**: `reload`
- **Type**: boolean
- **Default**: `true`
- **What it does**: Auto-reloads the shell when devenv files change
- **Security implications**: Means changes to `devenv.nix` take effect automatically. If a developer modifies their `devenv.local.nix` with malicious content, it executes on next file save rather than requiring explicit re-entry.
- **Hardening guidance**: Leave enabled (the benefits of live-reload outweigh the risk, since `devenv.local.nix` is already a local trust boundary).

---

## 3. Nix-Level Settings

### 3.1 Can devenv.nix Set Nix Settings?

**No, not directly.** `devenv.nix` configures the development environment, not the Nix daemon. Nix security settings like `sandbox`, `restrict-eval`, and `trusted-substituters` are daemon-level configurations that must be set in:

- `/etc/nix/nix.conf` (system-wide, requires root)
- `~/.config/nix/nix.conf` (user-level, limited scope)
- `flake.nix` `nixConfig` attribute (when using flakes integration -- prompts user for approval)
- `--nix-option <key> <value>` CLI flag (per-invocation)

The devenv CLI passes `--nix-option` flags to underlying Nix commands, so you can influence Nix behavior per-invocation but cannot override daemon-level security settings without elevated privileges.

### 3.2 Key Nix Security Settings

**sandbox** (Boolean, default: `true` on Linux)
- Isolates builds in private namespaces. Only Nix store dependencies and temp dirs are accessible.
- **Hardening**: Ensure `sandbox = true` in system nix.conf. Never set `sandbox = false`.

**restrict-eval** (Boolean, default: `false`)
- Prevents Nix evaluator from accessing files outside `builtins.nixPath` or URIs outside `allowed-uris`.
- **Hardening**: Enable for CI/production builds. May break interactive development if too restrictive.

**allowed-uris** (List, default: empty)
- URI prefixes permitted in restricted evaluation mode.
- **Hardening**: Allowlist only trusted sources (`github:NixOS`, `github:cachix`).

**trusted-substituters** (List, default: empty)
- Binary caches that unprivileged users can enable.
- **Hardening**: Explicitly list only trusted caches. Default `cache.nixos.org` plus `devenv.cachix.org` if using devenv's binary cache.

**trusted-public-keys** (List, default: `cache.nixos.org-1:...`)
- Public keys for verifying binary cache signatures.
- **Hardening**: Only add keys for caches you explicitly trust. Each key grants that cache the ability to provide binaries that will be executed on your system.

**require-sigs** (Boolean, default: `true`)
- Mandates cryptographic signatures on store paths.
- **Hardening**: Never set to `false`. This is the primary defense against tampered binary caches.

**trusted-users** (List, default: `root`)
- Users who can bypass security restrictions (set substituters, import unsigned NARs).
- **Hardening**: "Adding a user to trusted-users is essentially equivalent to giving that user root access." Minimize this list.

**allowed-users** (List, default: `*`)
- Users who can connect to the Nix daemon.
- **Hardening**: Restrict to specific users/groups on shared systems.

**filter-syscalls** (Boolean, default: `true`)
- Prevents setuid/setgid file creation and ACL/xattr manipulation in builds.
- **Hardening**: Leave enabled.

**sandbox-paths** (List, default: empty)
- Additional paths bind-mounted into sandboxed builds.
- **Hardening**: Minimize. Each added path expands what builds can access.

### 3.3 NixOS Configuration Example

For NixOS systems, these settings go in the system configuration:

```nix
# /etc/nixos/configuration.nix
{
  nix.settings = {
    sandbox = true;
    require-sigs = true;
    trusted-substituters = [
      "https://cache.nixos.org"
      "https://devenv.cachix.org"
    ];
    trusted-public-keys = [
      "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
      "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw="
    ];
    trusted-users = [ "root" ];  # NOT "@wheel"
    allowed-users = [ "@developers" ];
    filter-syscalls = true;
  };
}
```

---

## 4. Pre-commit Hooks for Security

devenv integrates with [cachix/git-hooks.nix](https://github.com/cachix/git-hooks.nix), which provides 120+ hooks. The security-relevant subset:

### 4.1 Secret Detection

| Hook | Purpose | Notes |
|------|---------|-------|
| `ripsecrets` | Detect secrets/credentials in code | **Only built-in secret scanner.** Fast, Rust-based. Detects API keys, tokens, passwords. |

Notable absences: `gitleaks`, `trufflehog`, `detect-secrets` are NOT built-in hooks. They must be configured as custom hooks.

```nix
{
  git-hooks.hooks = {
    ripsecrets.enable = true;

    # Custom gitleaks hook (not built-in)
    gitleaks = {
      enable = true;
      name = "gitleaks";
      entry = "${pkgs.gitleaks}/bin/gitleaks git --staged --verbose";
      language = "system";
      pass_filenames = false;
      stages = [ "pre-commit" ];
    };
  };
}
```

### 4.2 License Compliance

| Hook | Purpose |
|------|---------|
| `reuse` | REUSE specification compliance -- ensures every file has a license header |

### 4.3 Commit Hygiene (Security-Adjacent)

| Hook | Purpose |
|------|---------|
| `check-added-large-files` | Prevent large file commits (blocks binary blobs, data dumps) |
| `no-commit-to-branch` | Prevent direct commits to protected branches |
| `check-case-conflicts` | Detect filename case conflicts (prevents cross-platform issues) |

### 4.4 Static Analysis (Language-Specific)

| Hook | Purpose | Security relevance |
|------|---------|-------------------|
| `clippy` | Rust linting | Catches unsafe code, memory issues |
| `shellcheck` | Shell script linting | Catches injection vulnerabilities, unquoted vars |
| `phpstan` / `psalm` | PHP static analysis | Type errors, some security patterns |
| `mypy` / `pyright` | Python type checking | Type safety reduces bug classes |
| `eslint` / `oxlint` | JS/TS linting | Configurable security rules |
| `statix` | Nix anti-patterns | Catches Nix evaluation issues |

### 4.5 Custom Security Hooks

For security tools not in the built-in list, use custom hook definitions:

```nix
{
  git-hooks.hooks = {
    # Dependency audit (example for Node.js)
    npm-audit = {
      enable = true;
      name = "npm audit";
      entry = "npm audit --audit-level=high";
      language = "system";
      files = "package-lock\\.json$";
      pass_filenames = false;
    };

    # SAST with semgrep
    semgrep = {
      enable = true;
      name = "semgrep";
      entry = "${pkgs.semgrep}/bin/semgrep scan --config auto --error";
      language = "system";
      pass_filenames = false;
      stages = [ "pre-push" ];  # Slower, run on push not commit
    };

    # Terraform security
    tfsec = {
      enable = true;
      name = "tfsec";
      entry = "${pkgs.tfsec}/bin/tfsec .";
      language = "system";
      files = "\\.tf$";
      pass_filenames = false;
    };
  };
}
```

---

## 5. devenv.lock: Pinning Granularity

### What It Pins

`devenv.lock` pins **input sources**, not individual packages. Each input is locked to a specific:

- **Git commit hash** (for GitHub/GitLab/Git inputs)
- **Content hash** (for tarball inputs)
- **Timestamp** of when the lock was created

Example lock entry (conceptual):
```
nixpkgs → github:NixOS/nixpkgs?rev=238b18d7b2c8239f676358634bfb32693d3706f3
```

### What It Does NOT Pin

- Individual package versions within nixpkgs (you get whatever version is in that nixpkgs commit)
- Runtime dependencies fetched by packages (e.g., npm packages, pip packages)
- Container base images referenced by URL

### Individual Package Version Control

You cannot pin `pkgs.nodejs` to a specific version independently of the nixpkgs commit. Strategies for version control:

1. **Use a nixpkgs commit that contains your desired version** -- search with `devenv search <name>`
2. **Use multiple nixpkgs inputs** -- pin a second input to a different commit for specific packages
3. **Use overlays** -- override package definitions to pin versions
4. **Use language-level lock files** -- `package-lock.json`, `Cargo.lock`, etc. are orthogonal to devenv.lock

### Security Implications

- `devenv.lock` provides **reproducibility** (same inputs = same packages), which is a security property (you know what you're running)
- `devenv update` refreshes the lock file -- this should be a deliberate, reviewed action (similar to dependabot PRs)
- Lock files should be committed to git so the whole team uses identical inputs
- **No automatic vulnerability scanning** of locked inputs -- you must manually track nixpkgs security advisories or use tooling like `vulnix`

---

## 6. Container and Process Isolation

### Processes: No Isolation

devenv processes provide **no security isolation**:

- All processes share the same user identity
- All processes share the same environment variables
- All processes can access the same filesystem
- No namespace isolation (PID, network, mount)
- No cgroup resource limits
- Socket activation passes file descriptors without authentication
- Process environment variables are visible to all other processes

**The process manager is for convenience (supervision, restart), not security.**

### Containers: Build-Time Only

devenv containers are an **image build tool**, not a runtime isolation mechanism:

- `devenv container build <name>` produces an OCI image via Nix
- The image is built reproducibly (good for supply chain)
- `copyToRoot` controls exactly what enters the image (good for minimal images)
- No runtime security policies are set at the devenv level
- Container runtime security (seccomp, capabilities, network policies) is the responsibility of whatever runs the image (Docker, Kubernetes, etc.)

### Isolation Gap

devenv provides no mechanism to run development processes with reduced privileges, namespace isolation, or capability restrictions. All processes run with the developer's full permissions. For development environments that need isolation, consider:

- Using the container build to test in an isolated runtime
- Using systemd-run or firejail wrappers in scripts
- Running services in actual containers via `docker-compose` alongside devenv

---

## 7. Environment Variable Management

### Layered Model

Environment variables in devenv come from multiple sources, applied in this order:

1. **Host environment** (inherited unless `clean.enabled = true`)
2. **Nix build environment** (`unsetEnvVars` removes build-related vars)
3. **`env` attribute** (explicit declarations in devenv.nix)
4. **`dotenv`** (loaded from `.env` file if enabled)
5. **`secretspec`** (injected from secret provider if enabled)
6. **`enterShell`** (can export additional vars)
7. **Process-specific `env`** (per-process overrides)

### Security Analysis by Source

| Source | Visibility | Committed to git? | Risk level |
|--------|-----------|-------------------|------------|
| Host env | All processes | N/A | High -- ambient credentials leak in |
| `env` in devenv.nix | All processes | Yes | Low -- auditable, no secrets |
| `dotenv` | All processes | Should not be | Medium -- file can leak |
| `secretspec` (shell) | All processes | Declaration only | Medium -- values in environment |
| `secretspec` (runtime) | Target process only | Declaration only | **Low -- best option** |
| `enterShell` exports | All processes | Yes (except local.nix) | Medium |
| Process `env` | That process | Yes | Medium |

### Recommended Architecture

```
secretspec.toml (committed)     → Declares WHAT secrets exist
secretspec provider (not committed) → Provides WHERE values come from
secretspec run -- <command>     → Injects secrets at runtime only
```

---

## 8. Scripts and Shell Hooks: Execution Model

### No Sandboxing

All execution contexts in devenv run with the developer's full permissions:

| Context | When it runs | Sandboxed? | Can access network? | Can access filesystem? |
|---------|-------------|------------|--------------------|-----------------------|
| `enterShell` | Every shell entry | No | Yes | Yes |
| `enterTest` | `devenv test` | No | Yes | Yes |
| `scripts.*` | On invocation | No | Yes | Yes |
| `git-hooks.*` | On git commit | No | Yes | Yes |
| `processes.*` | `devenv up` | No | Yes | Yes |

### Trust Model

The trust boundary in devenv is the Nix configuration files:

- `devenv.nix` -- committed, code-reviewed, shared by team
- `devenv.local.nix` -- **NOT committed**, per-developer, not reviewed
- `devenv.yaml` -- committed, declares inputs
- `devenv.local.yaml` -- **NOT committed**, per-developer

**Security implication**: `devenv.local.nix` can override any setting from `devenv.nix`, including adding malicious packages, scripts, or shell hooks. This is a feature (developer customization) but means the team cannot enforce security controls if a developer's local machine is compromised.

### Mitigation

For environments where enforcement matters:

1. Use `devenv test` in CI to verify the committed configuration passes security checks
2. Use `enterTest` to assert security controls are active
3. Audit `devenv.lock` changes in code review
4. Rely on Nix daemon settings (section 3) for system-level enforcement that local overrides cannot bypass

---

## 9. Hardened Boilerplate Summary

A security-hardened devenv configuration combines these settings:

```yaml
# devenv.yaml
clean:
  enabled: true
  keep:
    - TERM
    - HOME
    - USER
    - DISPLAY
    - WAYLAND_DISPLAY
    - XDG_RUNTIME_DIR
    - SSH_AUTH_SOCK
impure: false
secretspec:
  enable: true
  provider: keyring
  profile: development
nixpkgs:
  allow_unfree: false
  allow_broken: false
  blocklisted_licenses:
    - "bsl11"
require_version: ">=2.1"
inputs:
  nixpkgs:
    url: github:NixOS/nixpkgs/nixos-24.11  # Stable channel
```

```nix
# devenv.nix
{ pkgs, config, ... }: {
  # Minimal, explicit package set
  packages = [
    pkgs.git
    pkgs.jq
    pkgs.curl
  ];

  # No secrets in env
  env.EDITOR = "vim";

  # Disable dotenv (use secretspec instead)
  dotenv.enable = false;

  # Security hooks
  git-hooks.enable = true;
  git-hooks.hooks = {
    ripsecrets.enable = true;
    check-added-large-files.enable = true;
    no-commit-to-branch.enable = true;
    shellcheck.enable = true;
  };

  # Security validation in tests
  enterTest = ''
    # Verify hooks are installed
    test -f .git/hooks/pre-commit
    # Verify clean environment (no leaked credentials)
    test -z "''${AWS_SECRET_ACCESS_KEY:-}"
    # Verify ripsecrets finds no issues
    ripsecrets --strict .
  '';

  # Generate security-critical files
  files.".gitignore".text = ''
    .env
    .env.*
    *.key
    *.pem
    secrets/
    devenv.local.nix
    devenv.local.yaml
  '';
}
```

---

## Sources

- [devenv.nix options reference](https://devenv.sh/reference/options/) -> `docs/devenv-nix-options-reference.md`
- [devenv.yaml options reference](https://devenv.sh/reference/yaml-options/) -> `docs/devenv-yaml-options-reference.md`
- [devenv pre-commit hooks](https://devenv.sh/pre-commit-hooks/) -> `docs/devenv-pre-commit-hooks.md`
- [devenv git hooks](https://devenv.sh/git-hooks/) -> `docs/devenv-git-hooks-configuration.md`
- [devenv inputs](https://devenv.sh/inputs/) -> `docs/devenv-inputs-configuration.md`
- [devenv processes](https://devenv.sh/processes/) -> `docs/devenv-processes-configuration.md`
- [devenv containers](https://devenv.sh/containers/) -> `docs/devenv-containers-configuration.md`
- [devenv scripts](https://devenv.sh/scripts/) -> `docs/devenv-scripts-configuration.md`
- [devenv files and variables](https://devenv.sh/files-and-variables/) -> `docs/devenv-files-and-variables.md`
- [devenv testing](https://devenv.sh/tests/) -> `docs/devenv-testing.md`
- [devenv packages](https://devenv.sh/packages/) -> `docs/devenv-packages.md`
- [devenv imports/composition](https://devenv.sh/composing-using-imports/) -> `docs/devenv-imports-composition.md`
- [devenv flakes integration](https://devenv.sh/guides/using-with-flakes/) -> `docs/devenv-flakes-integration.md`
- [devenv top-level module source](https://raw.githubusercontent.com/cachix/devenv/main/src/modules/top-level.nix) -> `docs/devenv-top-level-module.md`
- [SecretSpec integration](https://devenv.sh/integrations/secretspec/) -> `docs/secretspec-integration.md`
- [SecretSpec announcement](https://devenv.sh/blog/2025/07/21/announcing-secretspec-declarative-secrets-management/) -> `docs/secretspec-announcement.md`
- [nix.conf security settings](https://nix.dev/manual/nix/2.19/command-ref/conf-file) -> `docs/nix-conf-security-settings.md`
- [git-hooks.nix complete hook list](https://github.com/cachix/git-hooks.nix/blob/master/modules/hooks.nix) -> `docs/git-hooks-nix-complete-list.md`
