# Runtime Isolation Options for devenv.sh Developer Environments

## Executive Summary

Devenv.sh shell sessions have **zero runtime isolation** -- the Nix build sandbox only applies during `nix-build`, not during `devenv shell` or `devenv up`. All processes share the user's filesystem, network, and environment. This report evaluates seven approaches to adding runtime isolation, from native devenv PRs to manual wrapping techniques.

**Bottom line**: No production-ready, integrated runtime isolation exists for devenv today. The pragmatic path is a layered approach: (1) devcontainer generation for teams needing isolation now, (2) manual bubblewrap wrapping as an opt-in hardening script in the boilerplate, and (3) tracking the native sandbox PRs for future integration. Landlock via `landrun` is the most promising lightweight option for per-executable sandboxing on Linux 5.13+.

---

## 1. Devenv Sandbox PR #2427 (Bubblewrap Approach)

### Status
- **PR**: [cachix/devenv#2427](https://github.com/cachix/devenv/pull/2427)
- **Author**: zaytsev
- **Created**: January 23, 2026
- **Last Updated**: February 21, 2026
- **State**: Draft, no reviews, not merged

### What It Sandboxes
The **entire shell process** runs inside a bubblewrap container. This is whole-shell isolation, not per-executable. Everything executed within `devenv shell` -- enterShell hooks, scripts, package binaries, user commands -- runs inside the sandbox.

### Implementation Approach
YAML-based configuration in `devenv.yaml`:

```yaml
sandbox:
  enable: true
  network:
    enable: true
  mounts:
    - path: /nix/store
    - path: /dev
      mode: dev
    - path: $HOME
      mode: overlay
```

Bubblewrap flags used:
- Filesystem namespace isolation via selective bind mounts
- `/nix/store` mounted read-only for package access
- `/dev` mounted with device mode
- `/etc/passwd` and `/etc/group` for user identity
- `$HOME` with overlay mode (writes go to a tmpfs layer, not the real home)
- `/run/current-system/sw` for NixOS system packages
- Optional network namespace isolation
- User/IPC namespace unsharing

Mount modes supported:
- **standard**: Direct bind mount (default)
- **overlay**: Overlay filesystem -- reads from real path, writes to tmpfs
- **dev**: Device node mounting
- **temporary**: tmpfs

### Why It Hasn't Merged

1. **Draft/WiP status**: Author explicitly marked as early proof-of-concept seeking feedback
2. **Unresolved design debate**: Competing with the fundamentally different Landlock approach in PR #1783
3. **UX degradation**: Shell customization (zsh, starship prompt) doesn't work inside the sandbox
4. **Platform limitation**: Linux-only, NixOS-only tested -- devenv supports macOS
5. **No maintainer review**: Zero reviews from the devenv core team
6. **Home directory tension**: To be useful, `$HOME` must be mounted (overlay mode), which reduces the isolation benefit significantly -- SSH keys, browser profiles, and credentials become readable

### How a User Would Enable It
If merged, adding `sandbox.enable: true` to `devenv.yaml` would activate it. No code changes to `devenv.nix` needed.

### Assessment
**Maturity**: Experimental (draft PR, 4 months without merge progress)
**Likelihood of merging as-is**: Low. The design debate with PR #1783 is unresolved, and the UX tradeoffs are significant.

---

## 2. Devenv Sandbox PR #1783 (Landlock Approach)

### Status
- **PR**: [cachix/devenv#1783](https://github.com/cachix/devenv/pull/1783)
- **Author**: LorenzBischof
- **Created**: March 19, 2025
- **Last Updated**: February 9, 2026
- **State**: Draft, author stated they lack capacity to continue

### What It Sandboxes
**Individual Nix-provided executables**, not the shell itself. Every binary added to `$PATH` by devenv gets wrapped with Landlock restrictions. The user's shell remains unrestricted -- they can still access all files, run global commands, etc. Only the devenv-provided tools (compilers, linters, package managers) are confined.

### Implementation Approach
Uses Linux Landlock LSM to restrict filesystem access on a per-process basis:
- Each wrapped executable can only access the project directory and its Nix closure (dependencies)
- Sandboxing is "transparent" -- the developer doesn't notice it during normal use
- Best-effort: gracefully disables on unsupported kernels
- No network isolation (Landlock added network controls only in kernel 6.7, ABI v4)

### Key Technical Properties of Landlock
- **Kernel requirements**: Linux 5.13+ (ABI v1 for filesystem), 6.7+ for network
- **Unprivileged**: No root or SUID needed -- processes sandbox themselves
- **Monotonic**: Restrictions can only increase, never decrease. Once applied, a process cannot remove its own restrictions
- **Stackable**: Works alongside AppArmor, SELinux, and other LSMs
- **NixOS kernel 7.0.2**: Full Landlock support including ABI v5 (IOCTL, kernel 6.10+)

### Why It Hasn't Merged
The author explicitly stated: "This definitely needs more work and was just an experiment. I wont have any time to develop or think about this until next year." Additionally:
- direnv integration is challenging (Landlock's monotonic property conflicts with dynamic environment switching)
- The author created a separate PoC called "peninsula" using seccomp-unotify to work around direnv issues, adding complexity

### Comparison: Bubblewrap vs Landlock for devenv

| Aspect | PR #2427 (Bubblewrap) | PR #1783 (Landlock) |
|--------|----------------------|---------------------|
| Scope | Entire shell session | Per-executable |
| User shell | Restricted (no zsh/starship) | Unrestricted |
| Filesystem | Whitelist via mounts | Confined to project + closure |
| Network | Optional isolation | No (requires kernel 6.7+) |
| UX impact | High (breaks shell customization) | Low (transparent) |
| Bypass risk | Low (everything sandboxed) | Medium (unsandboxed shell, PATH manipulation) |
| direnv compat | Unknown | Problematic (monotonic restrictions) |
| Root required | No (unprivileged namespaces) | No (Landlock is unprivileged) |

### Assessment
**Maturity**: Experimental (abandoned draft, author unavailable)
**Design merit**: Higher than #2427 for developer experience -- transparent sandboxing is the right UX. But the monotonic restriction problem with direnv is a real blocker.

---

## 3. Manual Bubblewrap Wrapping (Available Today)

### Can You Wrap `devenv shell` in bwrap Today?
**Yes**, with significant caveats. A working wrapper script can isolate a devenv shell session without the PR.

### Working Example

```bash
#!/usr/bin/env bash
# devenv-sandboxed.sh -- wrap devenv shell in bubblewrap
# Requires: bubblewrap (bwrap) in PATH

PROJECT_DIR="$(pwd)"

bwrap \
  --ro-bind /nix /nix \
  --ro-bind /run/current-system /run/current-system \
  --ro-bind /etc /etc \
  --proc /proc \
  --dev /dev \
  --tmpfs /tmp \
  --tmpfs /run/user \
  --bind "$PROJECT_DIR" "$PROJECT_DIR" \
  --ro-bind ~/.config/git ~/.config/git \
  --unshare-pid \
  --unshare-ipc \
  --clearenv \
  --setenv HOME "$HOME" \
  --setenv USER "$USER" \
  --setenv PATH "/run/current-system/sw/bin:/nix/var/nix/profiles/default/bin" \
  --setenv NIX_SSL_CERT_FILE /etc/ssl/certs/ca-certificates.crt \
  --chdir "$PROJECT_DIR" \
  --die-with-parent \
  -- devenv shell
```

### What Breaks

1. **direnv integration**: `direnv allow` state is stored in `~/.local/share/direnv/allow/` -- if home is not mounted, direnv won't recognize the project. Fix: `--ro-bind ~/.local/share/direnv ~/.local/share/direnv`
2. **Shell customization**: zsh/bash configs in `$HOME` won't load unless explicitly bind-mounted read-only
3. **SSH/GPG**: No access to `~/.ssh` or `~/.gnupg` unless explicitly mounted. Git operations requiring SSH auth will fail
4. **Process management**: `devenv up` inside bwrap works but process-compose's TUI may have terminal issues
5. **File watching**: inotify works within the mount namespace, but watching files outside the bind mounts obviously fails
6. **Nix evaluation cache**: `.devenv/` needs to be writable. If project dir is bind-mounted rw, this works
7. **Native auto-activation**: devenv's v2.0 native shell activation (hook-based) may conflict with being inside bwrap
8. **Network**: If `--unshare-net` is used, all network access is blocked. Most dev workflows need network (package downloads, API calls). Omit `--unshare-net` for usability
9. **NixOS-specific paths**: The example above uses `/run/current-system` which only exists on NixOS. Other distros need different system path mounts

### Reference Implementations

Two existing projects demonstrate this pattern:
- **[nix-sandbox](https://github.com/fabian-thomas/nix-sandbox)**: Shell script wrapping CLI tools in bwrap with Nix dependencies. Mounts CWD read-write, .git read-only, minimal system dirs
- **[bubblewrap-claude](https://github.com/matgawin/bubblewrap-claude)**: Nix flake wrapping Claude Code in bwrap with `--unshare-all`, project dir rw, network via proxy allowlist

### Assessment
**Maturity**: Functional but fragile. Requires per-system tuning and breaks common workflows.
**Recommendation**: Include as an opt-in hardening script in the boilerplate with clear documentation of what breaks. Not suitable as a default.

---

## 4. Devcontainer Generation

### How It Works
Devenv can generate `.devcontainer.json` files via a single configuration line:

```nix
{ pkgs, ... }: {
  devcontainer.enable = true;
}
```

Running `devenv shell` auto-generates `.devcontainer.json`. The generated config:
- Uses `ghcr.io/cachix/devenv/devcontainer:latest` as the base image (Nix + devenv pre-installed)
- Installs the `mkhl.direnv` VS Code extension
- Runs `devenv test` as the `updateContentCommand`
- Supports freeform settings (any devcontainer.json property can be added)

### What Isolation Does It Provide?

The devcontainer itself provides **container-level isolation** -- but the isolation quality depends entirely on the container runtime:

| Runtime | Isolation Level |
|---------|----------------|
| Docker (default) | Linux namespaces + cgroups. Shared kernel with host. Container escapes are documented |
| Docker + Enhanced Container Isolation (ECI) | Sysbox runtime, user namespaces, stronger boundaries |
| Podman (rootless) | User namespace isolation, no daemon |
| GitHub Codespaces | VM-level isolation (Azure VMs). Strongest boundary |

**What's isolated** (with standard Docker):
- Filesystem: Container has its own root filesystem. Project mounted via volume
- Network: Bridge network by default. Configurable
- PID: Separate PID namespace
- User: Runs as non-root inside container (configurable)

**What's NOT isolated by the generated config**:
- No `securityOpt` directives (no seccomp, no AppArmor profile)
- No `capDrop` (all default Docker capabilities retained)
- No network restrictions
- Project directory is mounted read-write from host

### Is This the Pragmatic Answer?

**Partially**. Devcontainers provide meaningful isolation that works today, but with tradeoffs:

**Advantages**:
- Works now with zero custom code
- Familiar to developers who use VS Code/Codespaces
- Container boundary prevents filesystem access to host (except mounted volumes)
- Reproducible across teams

**Disadvantages**:
- Requires Docker/Podman running (not always available, especially on CI)
- Performance overhead of containerization
- Nix-in-Docker has known friction (store volume management, daemon setup)
- devenv's key selling point is "not being in a container" -- this negates it
- The generated config has no security hardening beyond default Docker isolation
- Nested containers (CI running in Docker, devcontainer inside) cause issues

### Assessment
**Maturity**: Production (devcontainer spec is mature; devenv's integration is minimal but functional)
**Recommendation**: Document as an option for teams that need isolation today. Enhance the generated `.devcontainer.json` with security hardening (capDrop, seccomp profile, read-only root filesystem) in the boilerplate.

---

## 5. Systemd-Based Isolation

### The Idea
Use `systemd-run --user` to wrap devenv processes in transient systemd units with security directives.

### Critical Limitation: User Services Cannot Sandbox

The systemd documentation is explicit: **"The various settings requiring file system namespacing support (such as ProtectSystem=) are not available for services run by the per-user service manager."**

This means the most useful security directives do NOT work with `--user`:

| Directive | System Service | User Service (--user) |
|-----------|---------------|----------------------|
| PrivateTmp | Yes | **No** |
| ProtectSystem | Yes | **No** |
| ProtectHome | Yes | **No** |
| PrivateDevices | Yes | **No** |
| ProtectKernelTunables | Yes | **No** |
| ProtectKernelModules | Yes | **No** |
| NoNewPrivileges | Yes | Yes |
| SystemCallFilter | Yes | Yes |
| MemoryDenyWriteExecute | Yes | Yes |
| RestrictNamespaces | Yes | Partial |
| CapabilityBoundingSet | Yes | Yes (limited) |

### Workaround: PrivateUsers=true

The systemd docs note: "Most namespacing settings, that will not work on their own in user services, will work when used in conjunction with PrivateUsers=true." However, this creates a new user namespace which can break Nix store access and other functionality.

### What Actually Works for devenv

```bash
# These directives DO work with --user:
systemd-run --user --pty --same-dir --wait --collect \
  --service-type=exec \
  --property="NoNewPrivileges=yes" \
  --property="SystemCallFilter=@system-service" \
  --property="MemoryDenyWriteExecute=yes" \
  -- devenv shell
```

This provides:
- **NoNewPrivileges**: Prevents SUID escalation from any devenv-installed binary
- **SystemCallFilter**: Restricts available syscalls (but cannot block execve)
- **MemoryDenyWriteExecute**: Prevents JIT-based exploits (breaks some runtimes like Node.js)

It does NOT provide:
- Filesystem isolation (no PrivateTmp, no ProtectHome)
- Network isolation
- Device isolation

### Compatibility with process-compose

`devenv up` launches processes via process-compose (or the native Rust PM). Wrapping `devenv up` in `systemd-run --user` creates a transient scope for all child processes, but:
- Process-compose manages its own process tree -- systemd's cgroup tracking may conflict
- process-compose's TUI needs a proper terminal (`--pty` flag)
- Health checks and readiness probes should work (they're internal to process-compose)

### Assessment
**Maturity**: Production (systemd is battle-tested), but the useful security features are unavailable for user services
**Recommendation**: Include `NoNewPrivileges=yes` wrapping in the boilerplate as a lightweight defense. Do NOT rely on systemd for filesystem/network isolation -- it doesn't work without root.

---

## 6. Namespace-Based Manual Isolation (unshare)

### What's Achievable Without Root

On NixOS, `security.unprivilegedUsernsClone` defaults to `true` (sets `kernel.unprivileged_userns_clone=1`), which means unprivileged user namespaces are available. This is required for bubblewrap to work.

Available namespaces for unprivileged users (via `unshare` or `bwrap`):

| Namespace | Unprivileged? | What It Isolates |
|-----------|--------------|-----------------|
| User | Yes (NixOS default) | UID/GID mapping |
| Mount | Yes (with user ns) | Filesystem view |
| PID | Yes (with user ns) | Process visibility |
| Network | Yes (with user ns) | Network stack (but no external connectivity without setup) |
| UTS | Yes (with user ns) | Hostname |
| IPC | Yes (with user ns) | System V IPC, POSIX message queues |
| Cgroup | Yes (with user ns) | Cgroup hierarchy view |
| Time | No (requires CAP_SYS_TIME) | System clock |

### Practical Example with unshare

```bash
# Create isolated mount+PID namespace with project dir access
unshare --user --map-root-user --mount --pid --fork -- bash -c '
  mount --bind /nix/store /nix/store
  mount -t proc proc /proc
  exec devenv shell
'
```

This is essentially what bubblewrap does, but bwrap is more ergonomic and handles edge cases (like `/dev`, `/proc` setup) correctly.

### Network Namespace Limitation

Creating a network namespace is straightforward (`unshare --net`), but the result is a loopback-only interface with no external connectivity. Setting up veth pairs and routing requires root (or complex `slirp4netns` setups). For development workflows that need network access, network namespace isolation is impractical without additional tooling.

### Assessment
**Maturity**: Production (kernel primitives are stable)
**Recommendation**: Don't use `unshare` directly -- use bubblewrap, which is a well-audited wrapper around the same primitives. The manual approach adds complexity with no benefit over bwrap.

---

## 7. Firejail

### How It Works
Firejail is a SUID sandbox using Linux namespaces, seccomp-bpf, and capabilities. On NixOS:

```nix
programs.firejail = {
  enable = true;
  wrappedBinaries = {
    devenv = {
      executable = "${pkgs.devenv}/bin/devenv";
      profile = "${pkgs.firejail}/etc/firejail/default.profile";
    };
  };
};
```

### Why It's Not Suitable for devenv Boilerplate

1. **SUID requirement**: Firejail needs `programs.firejail.enable = true` in the NixOS system configuration. A devenv boilerplate should work without system-level changes
2. **System-level config**: wrappedBinaries is a NixOS module option, not a per-project setting
3. **Profile complexity**: Firejail profiles need careful tuning per application. A generic profile for devenv would be either too restrictive (breaking workflows) or too permissive (not providing meaningful isolation)
4. **Security concerns**: Firejail's SUID design has been a target for privilege escalation CVEs. bubblewrap's unprivileged design is considered more secure

### Assessment
**Maturity**: Production (widely used for desktop app sandboxing)
**Recommendation**: Not suitable for the boilerplate. Requires system-level NixOS changes and SUID binary.

---

## 8. Landlock via landrun (Available Today, No devenv Changes Required)

### What Is landrun?

[landrun](https://github.com/Zouuup/landrun) is a standalone CLI tool that uses Landlock LSM to sandbox any process. Unlike PR #1783 (which requires devenv integration), landrun can be used today to wrap devenv commands.

### Working Example for devenv

```bash
# Sandbox devenv shell with filesystem restrictions
landrun \
  --rox /nix/store \
  --ro /etc \
  --ro /run/current-system \
  --rwx "$(pwd)" \
  --rw /tmp \
  --best-effort \
  --unrestricted-network \
  -- devenv shell
```

This restricts the shell to:
- Read-only + execute access to the Nix store (packages)
- Read-only access to system configuration
- Read-write + execute access to the project directory
- Read-write access to /tmp
- Full network access (unrestricted)

### Advantages Over Bubblewrap for This Use Case

1. **No namespace creation**: Landlock doesn't use namespaces, so it works even when `kernel.unprivileged_userns_clone=0`
2. **Transparent**: No mount namespace means tools see the real filesystem paths -- no path translation issues
3. **Monotonic security**: Sandboxed processes cannot remove restrictions, even if they exec new binaries
4. **Kernel-native**: No SUID, no daemon, no container runtime
5. **Graceful degradation**: `--best-effort` disables restrictions on unsupported kernels instead of failing

### Limitations

1. **Linux 5.13+ required**: Won't work on older kernels (NixOS 7.0.2 is fine)
2. **No PID isolation**: Sandboxed process can see all host processes
3. **No environment variable isolation**: Process inherits the full environment
4. **Network restrictions require kernel 6.7+**: Only TCP bind/connect filtering, no UDP
5. **Cannot restrict already-open file descriptors**: Only affects new file operations
6. **Home directory**: If `$HOME` is not explicitly granted, shell configs won't load

### Assessment
**Maturity**: Beta (landrun is relatively new, but Landlock itself is kernel-stable since 5.13)
**Recommendation**: Strong candidate for inclusion in the boilerplate as an opt-in hardening wrapper. Add landrun to `packages` and provide a convenience script. Document the limitations clearly.

---

## Comparison Matrix

| Approach | Maturity | FS Isolation | Network Isolation | PID Isolation | IPC Isolation | Env Isolation | What Breaks | Performance | Boilerplate Now? |
|----------|----------|-------------|-------------------|---------------|---------------|---------------|-------------|-------------|-----------------|
| **PR #2427 (bwrap native)** | Experimental | Yes (mount ns) | Optional | Yes | Yes | Yes (clearenv) | zsh/starship, SSH, shell configs | Negligible | No (not merged) |
| **PR #1783 (Landlock native)** | Experimental | Per-executable | No | No | No | No | direnv compat | Negligible | No (abandoned) |
| **Manual bwrap** | Functional | Yes (mount ns) | Optional | Yes | Yes | Yes | direnv, SSH, shell configs, portability | Negligible | Opt-in script |
| **Devcontainer** | Production | Yes (container) | Configurable | Yes | Yes | Yes | Docker required, perf overhead, Nix-in-Docker friction | Moderate | Document as option |
| **systemd --user** | Limited | **No** | **No** | **No** | **No** | No | Nothing (but provides almost nothing) | None | NoNewPrivileges only |
| **unshare (manual)** | Production | Yes (with user ns) | Impractical | Yes | Yes | No | Same as bwrap, less ergonomic | Negligible | No (use bwrap) |
| **Firejail** | Production | Yes | Yes | Yes | Yes | Yes | Requires NixOS system config, SUID | Negligible | No (system-level) |
| **landrun (Landlock)** | Beta | Yes (Landlock) | Partial (TCP, 6.7+) | No | No | No | No PID/IPC isolation, no env cleaning | Negligible | Opt-in script |

---

## Recommendation: What the Boilerplate Should Include

### Include TODAY

1. **Devcontainer generation with security hardening**
   ```nix
   devcontainer.enable = true;
   devcontainer.settings = {
     # Add security hardening to generated .devcontainer.json
     "runArgs" = ["--cap-drop=ALL" "--security-opt=no-new-privileges:true"];
     "containerUser" = "vscode";
   };
   ```
   Document as the primary isolation option for teams that can use Docker/Podman. Enhance the default generated config with capability dropping and no-new-privileges.

2. **Manual bubblewrap wrapper script**
   Include a `scripts.devenv-sandboxed` in the boilerplate that wraps `devenv shell` in bwrap with sensible defaults (project dir rw, nix store ro, no home access, PID isolation). Document what breaks and how to customize mounts.

3. **landrun wrapper script**
   Include `landrun` in packages and provide a `scripts.devenv-landlock` that restricts filesystem access via Landlock. Better UX than bwrap (no namespace issues, transparent paths) but weaker isolation (no PID/env).

4. **NoNewPrivileges via systemd** for `devenv up` processes
   If processes run via process-compose, wrap in `systemd-run --user --property=NoNewPrivileges=yes`. This is a minimal defense that prevents SUID-based privilege escalation from any devenv-installed package.

### Document as "Enable When Available"

5. **Native devenv sandbox** (PR #2427 or #1783)
   Track both PRs. When either merges, add `sandbox.enable: true` to the boilerplate. The Landlock approach (#1783) is architecturally better for developer experience but has the direnv blocker. The bubblewrap approach (#2427) provides stronger isolation but worse UX.

6. **Enhanced Landlock with network restrictions**
   When the boilerplate targets kernel 6.7+ consistently, add TCP network restrictions to the landrun wrapper (restrict outbound connections to known-good destinations).

### Do NOT Include

7. **Firejail**: Requires NixOS system-level configuration, SUID binary. Not portable.
8. **Raw unshare**: Use bubblewrap instead -- it's the same primitives with better ergonomics and security.
9. **systemd filesystem isolation**: Does not work with `--user` services. Would require root-level systemd units, which defeats the purpose of a portable devenv boilerplate.

---

## NixOS-Specific Considerations

The target system runs NixOS with kernel 7.0.2, which means:
- **Unprivileged user namespaces**: Enabled by default (`security.unprivilegedUsernsClone = true`). Bubblewrap works out of the box
- **Landlock**: Full ABI v5 support (filesystem, network, IOCTL). All Landlock features available
- **systemd**: User service manager available but with namespace limitations
- **Bubblewrap**: Available in nixpkgs (`pkgs.bubblewrap`)
- **landrun**: May need packaging or fetching from GitHub

For non-NixOS Linux distributions:
- Unprivileged user namespaces may be restricted (Ubuntu restricts since 23.10 with AppArmor)
- Landlock availability depends on kernel version and LSM configuration
- Bubblewrap may need the kernel sysctl set

For macOS:
- None of these approaches work. macOS has `sandbox-exec` (deprecated) and no Landlock/namespace support
- Devcontainers (running a Linux VM via Docker Desktop) are the only viable option

---

## Open Questions

1. **Will the devenv team settle on bubblewrap or Landlock?** The two PRs represent fundamentally different philosophies (whole-shell vs per-executable). A hybrid using Landlock for per-executable sandboxing plus bubblewrap for the shell when maximum isolation is needed would cover both use cases
2. **Can Landlock's monotonic restriction property be worked around for direnv?** The "peninsula" PoC using seccomp-unotify suggests a path but adds significant complexity
3. **What's the performance impact of Nix-in-Docker for devcontainer workflows?** Nix store volume management and evaluation caching inside containers needs benchmarking
4. **Should the boilerplate include landrun as a package?** It may not be in nixpkgs yet -- may need a flake input or custom derivation

---

## Sources

All source documents are saved in `docs/`. Key sources for this report:

- `docs/devenv-sandbox-pr-2427-detailed.md` -- PR #2427 full analysis
- `docs/devenv-landlock-pr-1783.md` -- PR #1783 full analysis
- `docs/nix-sandbox-bubblewrap-tool.md` -- nix-sandbox reference implementation
- `docs/bubblewrap-sandboxing-shell-tutorial.md` -- bwrap tutorial with working examples
- `docs/bubblewrap-claude-code-sandbox.md` -- bubblewrap-claude reference implementation
- `docs/devenv-containers-docs-detailed.md` -- devenv container generation
- `docs/devenv-codespaces-devcontainer-integration.md` -- devcontainer integration
- `docs/devenv-devcontainer-module-source.md` -- devcontainer module source analysis
- `docs/landlock-unprivileged-sandboxing.md` -- Landlock overview
- `docs/landrun-landlock-sandbox-tool.md` -- landrun CLI tool
- `docs/systemd-sandboxing-redhat.md` -- systemd security directives
- `docs/systemd-zero-code-sandboxing-cloudflare.md` -- systemd-run ad-hoc sandboxing
- `docs/firejail-nixos-wiki.md` -- Firejail on NixOS
