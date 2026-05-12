# Nix Security Mechanisms Relevant to devenv.sh

## Overview

This report catalogs the security mechanisms that Nix provides at the package manager level and analyzes how devenv.sh interacts with, leverages, or bypasses each one. The goal is to identify which mechanisms can be configured into a security-hardened devenv.sh boilerplate and where gaps remain.

Nix's security model is layered: cryptographic store integrity at the bottom, build-time sandboxing in the middle, evaluation-time purity constraints at the top, and a daemon-based trust model governing who can do what. Devenv.sh sits atop all of these as an opinionated developer environment manager, but it does not add its own security layer -- it inherits Nix's protections and in some cases weakens them for ergonomic reasons.

---

## 1. Sandboxed Builds

### How It Works

Nix's build sandbox uses Linux kernel namespaces to isolate build processes. When `sandbox = true` (the default on Linux), each build runs in private PID, mount, network, IPC, and UTS namespaces. The build process sees only:

- **Nix store paths** declared as dependencies
- **Temporary build directory** (`/tmp/nix-build-*`)
- **Private device nodes**: `/proc`, `/dev`, `/dev/shm`, `/dev/pts`
- **Explicitly configured `sandbox-paths`**

Network access is completely blocked for normal derivations. The builder cannot see the host filesystem hierarchy, cannot communicate with other processes via IPC, and cannot even see other PIDs on the system.

**Fixed-output derivations (FODs)** are the exception: they bypass network isolation because they need to fetch content from the internet. Their integrity is guaranteed by a pre-declared content hash rather than by build isolation.

### What It Protects Against

- **Undeclared dependencies**: Builds cannot silently depend on `/usr/bin/curl` or other host tools
- **Build-time data exfiltration**: No network access means a compromised build script cannot phone home
- **Cross-build contamination**: PID/IPC isolation prevents builds from interfering with each other
- **Host filesystem manipulation**: Mount namespace prevents writes outside the sandbox

### Limitations

- **Not a security boundary against malicious derivation authors**: The Nix project explicitly states the sandbox is for reproducibility, not for containing malicious code. A compromised derivation can still produce malicious output that runs outside the sandbox.
- **FOD network exception**: Fixed-output derivations have full network access. A malicious FOD could contact a C2 server during the build phase.
- **`sandbox-fallback = true` by default**: If the kernel doesn't support sandboxing, Nix silently falls back to unsandboxed builds. This weakens guarantees without warning.
- **`sandbox = relaxed` mode**: Allows FODs and `__noChroot` derivations to skip sandboxing entirely.
- **Historical sandbox escapes**: CVE history includes at least 3 sandbox escape advisories (GHSA-g3g9-5vj6-r3gj critical, GHSA-q82p-44mg-mgh5 low, GHSA-wf4c-57rh-9pjg low on macOS), indicating this is not an impenetrable boundary.
- **macOS limitations**: macOS uses a different sandboxing mechanism (`sandbox-exec`) that lacks Linux namespace-level isolation. Multiple macOS-specific escapes have been reported.

### Devenv.sh Interaction

Devenv.sh does **not** modify sandbox settings. It inherits whatever the system's `nix.conf` configures. On NixOS, this means `sandbox = true` by default. On non-NixOS Linux with the Determinate Systems installer, sandboxing is also enabled. On macOS, sandboxing is present but weaker.

**Hardening opportunity**: A devenv boilerplate could enforce `sandbox-fallback = false` in the project's nix configuration to prevent silent fallback to unsandboxed builds. However, devenv cannot set daemon-level nix.conf options -- these require system-level configuration.

**Key constraint**: Devenv.sh shell environments themselves are NOT sandboxed. The sandbox only applies during `nix-build` / `nix build` operations. Once packages are built and installed into the Nix store, the developer's shell has full access to them and to the host system. There is no runtime sandboxing of developer tools.

---

## 2. Content-Addressed Derivations (CA)

### How It Works

Traditional Nix uses **input-addressed** store paths: the output path is computed from a hash of all inputs (source, dependencies, build script). This means any change to any input -- even a cosmetic one -- produces a different output path, even if the output bytes are identical.

**Content-addressed derivations** hash the output itself. The store path is `sha256:<hash-of-actual-output-bytes>`. This provides:

- **Early cutoff**: If an input changes but the output is byte-identical, downstream rebuilds are skipped
- **Trustless sharing**: Two users can verify they have the same output without trusting each other or a shared signer, because the path itself encodes the content hash

### Security Properties

The key security benefit is a **modified trust model**: CA paths don't need signatures from a trusted key because anyone can verify the path by hashing its contents. This eliminates the need for `trusted-public-keys` for CA store objects -- `require-sigs` does not apply to content-addressed paths.

This means CA derivations could enable trustless binary sharing across untrusted environments without the key management overhead of traditional signature verification.

### Current State

CA derivations remain **experimental** as of Nix 2.28 (May 2026). The `ca-derivations` stabilization milestone is approximately 65% complete. Activation requires:

```nix
# In nix.conf or NixOS configuration
experimental-features = ca-derivations
```

Individual derivations must opt in with `__contentAddressed = true`, or globally via `config.contentAddressedByDefault = true` in nixpkgs.

### Devenv.sh Interaction

Devenv.sh does **not** enable CA derivations and does not use `__contentAddressed` in any of its derivations. Since the feature is experimental, this is unsurprising. A hardened boilerplate could enable the experimental feature, but the ecosystem-wide adoption is insufficient for this to provide meaningful benefit in a typical devenv setup today -- most packages from nixpkgs do not set `__contentAddressed`.

**Future relevance**: When CA derivations stabilize, they could meaningfully reduce the trust surface of binary caches in devenv setups by eliminating dependence on cache signing keys for verified packages.

---

## 3. Binary Cache Signature Verification

### How It Works

When Nix downloads a pre-built package (a "substitution") from a binary cache, it verifies a cryptographic signature before accepting the store path. The verification chain:

1. **Cache serves a `.narinfo` file** per store path, containing: the NAR hash, NAR size, references, and one or more Ed25519 signatures
2. **Nix computes a fingerprint**: `store-path;nar-hash;nar-size;refs` (canonicalized)
3. **Signature is verified** against keys listed in `trusted-public-keys`
4. **If verification fails**: the substitution is rejected and Nix falls back to building from source

### Default Configuration

```
trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=
substituters = https://cache.nixos.org/
require-sigs = true  # default
```

The `require-sigs = true` default means **all non-content-addressed paths must have a valid signature**. Disabling this (`require-sigs = false`) would accept any binary from any cache without verification -- a catastrophic security regression.

### Trust Configuration

- **`trusted-public-keys`**: Keys that can sign store paths. System-wide in `/etc/nix/nix.conf`.
- **`trusted-substituters`**: Cache URLs that unprivileged users are allowed to add. Without being in this list, only trusted users can add new caches.
- **`extra-trusted-public-keys`** / **`extra-substituters`**: Additive variants that don't replace the system defaults -- they append to them.

### What Happens on Mismatch

If a signature doesn't match any trusted key, the substitution fails silently and Nix rebuilds from source. This is secure but can cause surprising build-from-source behavior when cache keys are misconfigured.

### Devenv.sh Interaction

Devenv.sh adds two additional cache sources via its `nixConfig` in `flake.nix`:

```nix
nixConfig = {
  extra-trusted-public-keys = "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw= cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM=";
  extra-substituters = "https://devenv.cachix.org https://cachix.cachix.org";
};
```

This means using devenv.sh trusts **three** signing keys (the two above plus the default `cache.nixos.org` key) and downloads from **three** cache sources. Each additional cache is an additional trust surface.

**Critical detail**: For `extra-substituters` from a flake's `nixConfig` to be used by an untrusted user, either:
- The user must be in `trusted-users` (equivalent to root access), OR
- The substituter URL must be in `trusted-substituters` in the system `nix.conf`

If neither condition is met, the user is prompted for confirmation. This is a friction point that devenv's docs address by recommending users add themselves to `trusted-users` -- which is **equivalent to granting root access** per Nix's own documentation.

**Hardening opportunity**: Instead of adding users to `trusted-users`, the system `nix.conf` should add devenv's caches to `trusted-substituters` and their keys to `trusted-public-keys`. This grants cache access without granting root-equivalent privileges.

---

## 4. Restricted and Pure Eval Modes

### `--pure-eval`

Pure evaluation mode prevents Nix expressions from depending on external state:

- **Disables**: `builtins.currentSystem`, `builtins.currentTime`, `builtins.nixPath`, `builtins.storePath`, `builtins.getEnv`
- **Restricts**: File access to only cryptographically-hashed store paths and the flake's own source tree
- **Prevents**: Environment variable reads, home directory expansion, system-dependent behavior

This ensures the same expression evaluates identically on any machine with the same inputs.

### `--restrict-eval`

Restricted evaluation mode is stricter about filesystem and network access:

- **Filesystem**: Only paths in `builtins.nixPath` are accessible
- **Network**: Only URIs matching prefixes in `allowed-uris` can be fetched
- **Effect**: Prevents evaluation from reading arbitrary files (like `/etc/passwd`) or fetching from arbitrary URLs

### What They Protect Against

- **Eval-time side channels**: A malicious Nix expression cannot read environment variables, access files outside its inputs, or contact external servers during evaluation
- **Non-reproducible builds**: Eliminates time-dependent, system-dependent, and environment-dependent behavior
- **Data exfiltration during eval**: `restrict-eval` prevents `builtins.fetchurl` from contacting arbitrary servers

### Limitations

- These only apply during **evaluation** (the Nix language interpreter phase). They do not restrict what build scripts do during the build phase (that's the sandbox's job).
- `restrict-eval` can be cumbersome -- it requires explicitly whitelisting every URI prefix that inputs might need.
- Neither mode prevents a derivation's build script from doing malicious things after evaluation completes.

### Devenv.sh Interaction

**Flakes enable `--pure-eval` by default.** When devenv.sh is used with flakes (either native flake mode or its internal flake generation), pure evaluation is active. This is a significant security benefit that comes "for free."

However, devenv.sh also supports `--no-pure-eval` via the `impure` option in `devenv.yaml`:

```yaml
impure: true  # Relaxes pure evaluation
```

This is sometimes needed because pure eval prevents devenv from querying the working directory or reading environment variables. The devenv CLI itself sometimes uses `--impure` to function (e.g., for `devenv shell` to detect the current directory).

**Hardening opportunity**: A boilerplate should keep `impure: false` (the default) and document which specific use cases genuinely require impurity, rather than blanket-enabling it.

Devenv.sh does **not** use `--restrict-eval`. This mode is primarily used by Hydra (the NixOS build farm) and is too restrictive for typical development workflows.

---

## 5. Flake Lock Files

### How It Works

A `flake.lock` file pins every transitive dependency of a flake to:

- **`rev`**: The exact Git commit hash
- **`narHash`**: SHA-256 hash (in SRI format) of the NAR serialization of the input's source tree
- **`lastModified`**: Timestamp of the pinned revision
- **Type-specific fields**: `owner`, `repo`, `ref`, etc.

When Nix evaluates a locked flake, it:
1. Fetches the input at the pinned `rev`
2. Serializes it to NAR format
3. Computes SHA-256 of the NAR
4. **Compares against `narHash`** -- fails if mismatch
5. Uses the fetched content only if hashes match

### Reproducibility Guarantees

- **Exact version pinning**: Every dependency is locked to a specific commit, not a branch or tag
- **Content integrity**: The `narHash` ensures the fetched content is byte-identical to what was locked
- **Transitive locking**: Lock files lock indirect dependencies too -- Nix ignores sub-flakes' own lock files

### Tampering Considerations

- **Lock file in version control**: The `flake.lock` is a JSON file committed to Git. Tampering with it requires write access to the repository (or a compromised PR merge).
- **`narHash` is the integrity check**: Even if `rev` is tampered with to point to a different commit, the `narHash` mismatch would cause a failure. An attacker would need to find a collision in SHA-256 NAR hashing -- computationally infeasible.
- **`nix flake update` is the attack surface**: Running `nix flake update` replaces pinned hashes with new ones. A developer running this unknowingly after a dependency is compromised would lock in the compromised version. This is a TOCTOU-style risk: the update is correct at the time it runs, but the upstream may have been compromised between the last review and the update.
- **No signature on the lock file itself**: The lock file relies on Git's own integrity (commit signing, if used). There is no Nix-level signature on `flake.lock`.

### Devenv.sh Interaction

Devenv.sh generates a `devenv.lock` file (or `flake.lock` when used in flake mode) that pins all inputs. By default, the primary input is `github:cachix/devenv-nixpkgs/rolling` -- a Cachix-maintained fork of nixpkgs that tracks the rolling release.

**Security implications**:
- Devenv pins to a specific fork (`devenv-nixpkgs`) rather than upstream `NixOS/nixpkgs`. This adds Cachix as a trust dependency.
- The `rolling` branch reference is resolved to a specific commit at lock time, but the branch name suggests frequent updates.
- Lock files provide strong integrity guarantees but only if developers review lock file changes in PRs.

**Hardening opportunity**: Pin to specific nixpkgs releases (e.g., `nixos-24.11`) rather than `rolling`. Require PR review for any `flake.lock` / `devenv.lock` changes. Use `nix flake lock --no-update-lock-file` in CI to fail if the lock is out of date rather than silently updating.

---

## 6. Store Path Integrity

### How It Works

Every path in `/nix/store` follows the pattern `/nix/store/<32-char-hash>-<name>-<version>`. The hash is derived through a multi-step process:

1. **NAR serialization**: File/directory contents are serialized into a canonical NAR format (sorted directory entries, normalized metadata)
2. **SHA-256 hash**: Computed over the NAR serialization
3. **Fingerprint construction**: `"source:sha256:<hash>:/nix/store:<name>"` (for source paths) or `"output:out:<drv-hash>:/nix/store:<name>"` (for derivation outputs)
4. **Path generation**: First 160 bits of SHA-256 of the fingerprint, base-32 encoded

### Integrity Properties

- **Immutability**: Store paths are read-only. The Nix store directory (`/nix/store`) is owned by root with `0555` permissions on store paths. Modification requires root access.
- **Content verification**: `nix store verify --check-contents` recomputes hashes and compares against the Nix database. Detects bit-rot, tampering, or corruption.
- **Referential integrity**: Nix tracks all references between store paths. A path cannot reference a non-existent path.
- **Collision resistance**: The fingerprint format separates source paths (`source:`) from derivation outputs (`output:out:`), preventing an attacker from crafting a source path that collides with a derivation output.
- **Database-backed verification**: Store path hashes are recorded in a SQLite database (`/nix/var/nix/db/db.sqlite`). The database is owned by root and not world-writable.

### Limitations

- **No runtime integrity monitoring**: Nix does not continuously verify store path contents. A root-level compromise could modify store contents and update the database. Verification is on-demand via `nix store verify`.
- **Database is the weak point**: The integrity guarantee depends on the database being trustworthy. A compromised root could update both the store path and the database hash.
- **No remote attestation**: There is no mechanism for a remote party to verify the integrity of a local Nix store.

### Devenv.sh Interaction

Devenv.sh inherits store path integrity without modification. All packages devenv installs are standard Nix store paths with the same integrity properties. The store is shared across all devenv environments on the same machine.

**Hardening opportunity**: Periodic `nix store verify --check-contents` as a cron job or CI step can detect store corruption. On NixOS, this can be configured as a systemd timer.

---

## 7. User/Daemon Trust Model

### How It Works

In multi-user Nix (the standard installation), the architecture is:

1. **Nix daemon** (`nix-daemon`): Runs as root, owns the Nix store and database
2. **Build users** (`nixbld1`-`nixbld10` typically): Unprivileged UIDs used to execute builds, members of the `nixbld` group with no other group memberships
3. **Regular users**: Connect to the daemon via Unix socket at `/nix/var/nix/daemon-socket/socket`

### Trust Levels

**`allowed-users`** (default: `*`):
- Users allowed to connect to the daemon at all
- Can request builds from arbitrary Nix expressions
- Can install packages from the default substituters
- Cannot add new substituters or modify daemon settings

**`trusted-users`** (default: `root`):
- Everything `allowed-users` can do, plus:
- Can specify arbitrary binary caches (substituters)
- Can import unsigned NARs
- Can set any nix.conf option via `--option`
- Can set `sandbox-paths` (gaining read access to otherwise inaccessible directories)
- **"Essentially equivalent to giving that user root access to the system"** -- Nix documentation

### Security Boundaries

- **Daemon socket**: Access controlled by filesystem permissions. Users not in the appropriate group cannot connect.
- **Build isolation**: Builds run as separate `nixbld` users in sandboxed namespaces, preventing builds from accessing each other's data or the invoking user's data.
- **The daemon is NOT a security boundary against malicious Nix language code**: The Nix project explicitly states this. Sharing a daemon with potentially malicious users is not recommended.

### Devenv.sh Interaction

This is one of the most significant security friction points with devenv.sh. To use devenv's Cachix integration seamlessly, devenv's documentation recommends adding the user to `trusted-users`:

```
# /etc/nix/nix.conf
trusted-users = root <username>
```

This grants root-equivalent access to the Nix daemon. On a shared system, this is a significant privilege escalation.

**Hardening opportunity**: Instead of `trusted-users`, configure `trusted-substituters` and `trusted-public-keys` at the system level:

```nix
# NixOS configuration
nix.settings = {
  trusted-substituters = [
    "https://devenv.cachix.org"
    "https://cachix.cachix.org"
  ];
  trusted-public-keys = [
    "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw="
    "cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM="
  ];
};
```

This allows unprivileged users to pull from devenv's caches without granting them root-equivalent daemon access.

---

## 8. Relationship with NixOS Security Features

### Available on NixOS

**Systemd service hardening**: NixOS services can use `PrivateNetwork`, `ProtectHome`, `NoNewPrivileges`, `ProtectSystem`, `ReadOnlyPaths`, `SystemCallFilter`, etc. These apply to services defined in NixOS configuration, not to developer shell environments.

**Stateful firewall**: NixOS enables a firewall by default that blocks unexpected incoming connections. Relevant for devenv processes that bind to ports (databases, dev servers).

**LUKS disk encryption**: Protects the Nix store at rest.

### Not Yet Properly Integrated on NixOS

**SELinux**: "Proper integration does not exist" on NixOS despite being technically possible. Cannot be leveraged.

**AppArmor**: "Available but not yet been properly integrated" as of April 2026. Partial support exists but is not production-ready for constraining arbitrary packages.

**Seccomp**: Not mentioned in NixOS security documentation. Individual applications can use seccomp, but NixOS does not provide a framework for applying seccomp profiles to Nix-installed packages.

### Applicability to Non-NixOS devenv Setups

On non-NixOS Linux (e.g., Ubuntu with Nix installed):
- **Build sandbox**: Works the same way (Linux namespaces) regardless of host distro
- **AppArmor**: The host distro's AppArmor can constrain Nix-installed binaries, but profiles must be written manually for each tool
- **Seccomp**: Available at the application level; no Nix-specific integration
- **Firejail/Bubblewrap**: Could wrap devenv shell invocations in additional sandboxing, but this is outside devenv's scope
- **Namespace isolation for dev shells**: Not available. Devenv shells are normal shell sessions with no additional isolation.

### Devenv.sh Interaction

Devenv.sh provides **container support** that can leverage some NixOS isolation features:

```nix
containers.myapp = {
  # Generates OCI-compatible container images
  layers.*.perms = { ... };
  layers.*.reproducible = true;
};
```

And **process management** with Linux capabilities:

```nix
processes.myservice.linux.capabilities = [ ... ];
```

However, these are for containerized deployments, not for the developer's local shell environment. The developer's shell itself has no additional isolation beyond what the host OS provides.

---

## 9. Recent Security Improvements

### Security Advisories (2024-2026)

The Nix project has addressed 9 security advisories, including:

| Advisory | Severity | Category |
|----------|----------|----------|
| GHSA-g3g9-5vj6-r3gj (Apr 2026) | Critical | Sandbox escape via symlink in FOD |
| GHSA-h4vv-h3jq-v493 (Sep 2024) | Critical | Unsafe NAR unpacking |
| GHSA-vh5x-56v6-4368 (May 2026) | High | Stack overflow in NAR parser |
| GHSA-qc7j-jgf3-qmhg (Jul 2025) | High | macOS privilege drop failure |
| GHSA-6fjr-mq49-mm2c (Sep 2024) | Moderate | Credential leak in fetchurl |
| GHSA-2ffj-w4mj-pg37 (Mar 2024) | Moderate | FOD corruption |
| GHSA-gr92-w2r5-qw5p (May 2026) | Moderate | Path traversal in archive unpacking |

Recurring themes: **sandbox escapes** (especially via fixed-output derivations and symlinks), **NAR parsing bugs**, and **macOS-specific issues**.

### Experimental Security Features

**`verified-fetches`** (experimental, milestone 48): Enables verification of Git commit signatures through `fetchGit`. When stable, this would allow devenv flake inputs to be verified against GPG/SSH signatures, adding a layer of authenticity verification beyond hash pinning.

**`configurable-impure-env`** (experimental, milestone 37): Allows administrators to control which environment variables are exposed during builds, reducing information leakage.

**`fetch-closure`** (experimental, milestone 40): Controlled retrieval of pre-built store objects with explicit provenance tracking.

### Ecosystem Developments

**Nixpkgs supply chain security project**: A funded initiative (supported by Sovereign Tech Agency) to improve supply chain security across the Nix ecosystem, including SBOM generation and vulnerability tracking.

**Nixpkgs GitHub Actions hardening**: Following the "Pwning the Nix Ecosystem" disclosure, nixpkgs workflows were hardened against `pull_request_target` abuse, credential leakage, and symlink attacks.

**RFC 0100 (Sign Commits)**: Under discussion for requiring signed commits to nixpkgs, though debate continues about implementation approach.

### Devenv.sh Interaction

Devenv.sh benefits from upstream Nix security fixes automatically when users update their Nix installation. However, devenv does not expose or enable any of the experimental security features. A hardened boilerplate could:

- Enable `verified-fetches` when it stabilizes
- Track Nix version requirements to ensure known-vulnerable versions are rejected (via `require_version` in `devenv.yaml`)
- Document minimum Nix versions that include critical security fixes

---

## Summary: Security Mechanism Interaction Matrix

| Mechanism | Devenv Default Behavior | Can Devenv Configure? | Hardening Action |
|-----------|------------------------|----------------------|-----------------|
| Build sandbox | Inherits system default (on on Linux) | No (system-level) | Set `sandbox-fallback = false` in system nix.conf |
| CA derivations | Not used | Could enable experimentally | Wait for stabilization |
| Binary cache sigs | Adds 2 Cachix caches + keys | Yes (cachix.pull/push) | Use `trusted-substituters` instead of `trusted-users` |
| Pure eval | On by default with flakes | Yes (`impure: true` disables) | Keep `impure: false`, document exceptions |
| Restrict eval | Not used | Not configurable via devenv | N/A for dev environments |
| Flake locks | Auto-generated, pins all inputs | Yes (input pinning) | Pin to release branches, review lock changes in PRs |
| Store integrity | Inherited from Nix | No | Run periodic `nix store verify` |
| Daemon trust | Recommends `trusted-users` | No (system-level) | Use `trusted-substituters` instead |
| NixOS security | N/A for non-NixOS | Partial (containers, capabilities) | Apply systemd hardening to devenv processes |
| Recent features | Benefits from Nix updates | No direct exposure | Track Nix version, enable verified-fetches when stable |

---

## Sources

All raw sources are saved in `docs/`:

- `docs/nix-conf-security-options.md` -- Full nix.conf security option reference
- `docs/ca-derivations-wiki.md` -- CA derivations wiki page
- `docs/nix-multi-user-mode.md` -- Multi-user mode documentation
- `docs/nix-sandboxing-discourse.md` -- Sandbox discussion and details
- `docs/nix-store-path-computation.md` -- Store path hash computation (Nix Pills)
- `docs/nix-security-advisories.md` -- All 9 security advisories
- `docs/nix-experimental-features.md` -- Experimental features reference
- `docs/nix-flake-lock-format.md` -- Flake lock file structure
- `docs/nixos-security-wiki.md` -- NixOS security wiki page
- `docs/determinate-nix-security.md` -- Determinate Systems security documentation
- `docs/tweag-untrusted-ci-binary-cache.md` -- Tweag untrusted CI caching article
- `docs/devenv-flake-nix-config.md` -- devenv's own flake.nix nixConfig
- `docs/devenv-binary-caching.md` -- devenv binary caching documentation
- `docs/devenv-nix-options-reference.md` -- devenv.nix security-relevant options
- `docs/devenv-yaml-options-reference.md` -- devenv.yaml full option reference
- `docs/nixcademy-secure-supply-chain.md` -- Secure supply chains with Nix
- `docs/pwning-nix-ecosystem.md` -- Nixpkgs GitHub Actions vulnerability
