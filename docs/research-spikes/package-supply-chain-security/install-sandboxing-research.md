# Install Script Sandboxing & Runtime Protections

## Executive Summary

Install-time code execution is the single most exploited attack vector in package supply chain compromises. Most major ecosystems allow packages to run arbitrary code during installation — npm via lifecycle scripts, Python via `setup.py`, Rust via `build.rs` and proc macros, Ruby via `extconf.rb`, and JVM via build plugins. Go is the notable exception, having made the explicit design decision that "neither fetching nor building code will let that code execute." The defenses available range from blunt script-disabling flags to sophisticated allowlist-based systems, but no ecosystem has yet shipped true OS-level sandboxing of install scripts as a default. The most practical configure-once defenses today are: npm's `ignore-scripts=true` with `@lavamoat/allow-scripts` for allowlisting; pnpm v10+'s `allowBuilds` with default script blocking; Python's `--only-binary :all:` to avoid `setup.py` execution; and Deno's permission model which blocks scripts by default. For defense-in-depth, these should be layered with network-level tools like Socket Firewall and environment-level isolation via containers or microVMs.

---

## 1. The Attack Vector: Per-Ecosystem Analysis

### 1.1 npm (JavaScript/TypeScript)

**Mechanism**: npm supports lifecycle scripts defined in `package.json`: `preinstall`, `install`, `postinstall`, `prepare`, and others. When you run `npm install`, these scripts execute with the full privileges of the installing user — file system access, network access, environment variable access, subprocess spawning. Any package in the dependency tree (including transitive dependencies) can define these hooks.

**Scale of the problem**: Approximately 2.2% of npm packages use install scripts. While this seems small, in a typical large project with hundreds of transitive dependencies, the probability of encountering at least one is high.

**Real-world attacks**:
- **Shai-Hulud worm (September 2025)**: Compromised 796 packages with 132 million monthly downloads via cascading postinstall script infections.
- **"qix" compromise (September 2025)**: Cascaded into 18 packages including `chalk`, `debug`, `ansi-styles`, and `strip-ansi` — collectively 2.6 billion weekly downloads.
- **eslint-scope**: Maintainer account breach led to a malicious postinstall script that stole npm credentials.
- **crossenv**: Typosquatting package that exfiltrated environment variables via a postinstall hook.
- **Axios compromise (March 2026)**: Malicious dependency injected through updated npm packages for Axios.

**What scripts can do**: Read/write any file the user can access, make network requests, access credentials and tokens, scan for SSH keys and cloud credentials, spawn processes, and exfiltrate data — all before the developer has reviewed any code.

### 1.2 Python (PyPI)

**Mechanism**: Python's supply chain attack surface centers on source distributions (sdists) vs. built distributions (wheels).

- **sdists with `setup.py`**: When pip installs a package that only provides a source distribution, it must execute `setup.py` to build the package and extract metadata. This is arbitrary code execution by design. The code runs during `pip install`, `pip download`, and even during dependency resolution when pip needs metadata.
- **Wheels**: Pre-built binary distributions that install without executing any code. The `.whl` format is just a zip file that gets extracted — no build step required.
- **PEP 517/518 and `pyproject.toml`**: Modern build backends replace `setup.py` with declarative `pyproject.toml` configuration. However, backwards compatibility means the specification still permits fallback to `setup.py` for older metadata formats. An attacker needs only a single source distribution anywhere in the dependency graph to trigger code execution.
- **The transitive problem**: Even if you install a wheel, its transitive dependencies may include sdists. One malicious sdist anywhere in the tree compromises the install.

**Real-world attacks**:
- **LiteLLM PyPI compromise (2025)**: 119,000+ compromised downloads in 2.5 hours before detection.
- **PyPI phishing campaign (July 2025)**: Targeted maintainers with proxy credential harvesters.

### 1.3 Rust (Cargo)

**Mechanism**: Cargo has two code execution vectors at build time:

- **`build.rs` (build scripts)**: Compiled and executed before the crate itself is compiled. Can perform arbitrary operations including network requests, file system access, and subprocess execution. Commonly used by `-sys` crates to find and link system libraries (e.g., `openssl-sys` probes for OpenSSL). Build scripts "can do literally anything from network requests to executing arbitrary binaries."
- **Procedural macros**: Rust code that runs at compile time to transform the token stream. Like build scripts, they execute arbitrary code but within the `rustc` process.

**Current state**: No sandboxing exists in mainline Cargo. The Rust project has an active initiative ("Explore Sandboxed Build Scripts," 2024 H2 project goal) exploring WebAssembly (WASI) and Cackle-based sandboxing. The vision is for sandboxed builds to become the default by the next Rust Edition, but this remains in-progress.

**Proposed mitigations**:
- **Build script allowlist mode** (Cargo issue #13681): Would block build script execution unless crates are explicitly allowlisted in `Cargo.toml`. Status: open proposal, needs design work, no active development.
- **Cackle (cargo-acl)**: Third-party tool that provides API-level ACLs for Rust dependencies. Runs build scripts in Bubblewrap sandboxes with network/filesystem restrictions. Analyzes compiled binaries to detect which APIs each dependency uses (net, fs, process, unsafe). Configured via `cackle.toml`. Available today but requires manual setup.
- **cargo-vet**: Mozilla's supply chain audit tool. Tracks manual code review status of dependencies. Doesn't sandbox, but ensures human review of changes.

### 1.4 Go

**Mechanism**: Go has **no install scripts by design**. This is an explicit, intentional security decision.

"It is an explicit security design goal of the Go toolchain that neither fetching nor building code will let that code execute, even if it is untrusted and malicious. This is different from most other ecosystems, many of which have first-class support for running code at package fetch time."

Key design features that reinforce this:
- No post-install hooks exist in the toolchain.
- `go.mod` is a unified constraints+lock file — no separate lock file that can diverge.
- The Go Module Mirror (proxy.golang.org) runs VCS tools in a sandbox when fetching modules.
- The Checksum Database (sum.golang.org) provides a global, append-only, cryptographically-verifiable record of module hashes — every consumer of a given version gets identical content.
- `init()` functions execute at runtime but only for packages that contribute code to the specific build, not for unused transitive dependencies.

Go's cultural emphasis on minimal dependencies ("a little copying is better than a little dependency") further reduces attack surface.

**Comparison value**: Go demonstrates that a major ecosystem can function effectively without install-time code execution. It is the gold standard for this specific threat vector.

### 1.5 Ruby (RubyGems)

**Mechanism**: RubyGems supports native C extensions via `extconf.rb` files. When a gem includes this file, RubyGems runs it automatically during `gem install` — before the developer imports or uses the package. This is functionally equivalent to npm's postinstall scripts.

**Real-world attacks**:
- **BufferZoneCorp campaign (May 2026)**: Gems with hidden scripts in the native extension build process swept CI/CD environments for SSH keys, AWS credentials, GitHub CLI tokens, npm configs, and RubyGems credentials, exfiltrating everything to attacker-controlled endpoints.
- **Cryptocurrency hijacking (2020)**: `ruby-bitcoin-0.0.20.gem` contained an obfuscated payload in `extconf.rb`.

**Mitigations**: RubyGems does not currently offer an `ignore-scripts` equivalent or allowlist mechanism. The primary defense is avoiding gems with native extensions where possible, and running `gem install` in isolated environments.

### 1.6 Maven/Gradle (JVM)

**Mechanism**: Maven and Gradle build plugins execute arbitrary Java/Groovy/Kotlin code during the build lifecycle. Unlike npm-style install scripts, this code runs during `mvn compile` / `gradle build` rather than during dependency download. However, the distinction is academic — developers routinely run builds immediately after dependency resolution.

**Attack vectors**:
- **Malicious build plugins**: Can download and execute arbitrary code from C2 servers.
- **MavenGate**: Exploits abandoned library domain takeovers. Since Maven Central doesn't verify domain ownership after initial publication, attackers can acquire expired domains and publish malicious updates. All Maven-based technologies including Gradle are vulnerable.
- **Maven-Hijack**: Exploits classpath ordering — classes with the same name in two JARs allow build/classpath order to determine which code runs.
- **Repository plugin attacks**: Gradle and Maven plugins sourced from repositories can be compromised.

**Mitigations**: Maven Enforcer Plugin provides build-time validation. Gradle's Dependency Analysis Plugin can detect Maven-Hijack-style risks. The primary recommendation is running builds in ephemeral, isolated environments (containers or VMs).

### 1.7 NuGet (.NET)

**Mechanism**: NuGet historically supported `install.ps1`, `uninstall.ps1`, and `init.ps1` PowerShell scripts that ran during package installation in Visual Studio.

**Current state**: `install.ps1` and `uninstall.ps1` are **deprecated** and not executed in the modern PackageReference format (which replaced `packages.config`). However:
- `init.ps1` is **still honored** by Visual Studio and runs without any warning when installing a NuGet package.
- Legacy `packages.config` projects still execute `install.ps1`.
- The migration tool warns about script incompatibility but doesn't prevent use of older formats.

**Real-world attacks**: Malicious NuGet packages have used `tools/init.ps1` to achieve code execution without triggering warnings, as documented by JFrog in 2023.

**Mitigations**: Migrating to PackageReference format eliminates `install.ps1`/`uninstall.ps1` execution. No mechanism exists to disable `init.ps1` execution in Visual Studio.

---

## 2. npm `--ignore-scripts` Deep Dive

### 2.1 How It Works

The `ignore-scripts` flag prevents execution of all lifecycle hooks during package installation. It can be configured at three levels:

1. **Per-command**: `npm install --ignore-scripts`
2. **Per-project** (`.npmrc` in project root): `ignore-scripts=true`
3. **System-wide**: `npm config set ignore-scripts true`

When enabled, npm skips `preinstall`, `install`, `postinstall`, `prepare`, and all other lifecycle scripts for all packages in the dependency tree.

### 2.2 What Breaks

Only ~2.2% of npm packages use install scripts, but those that do include widely-used packages:
- **bcrypt**: Compiles native C++ addon
- **node-sass** (deprecated but still used): Downloads platform-specific binary
- **sharp**: Downloads/compiles libvips for image processing
- **esbuild**: Downloads platform-specific binary
- **sqlite3**: Compiles native SQLite binding
- **canvas**: Compiles native Cairo binding

These packages will install their JavaScript files but will not have their native binaries compiled or downloaded, causing runtime failures.

### 2.3 Allowlisting with @lavamoat/allow-scripts

`@lavamoat/allow-scripts` provides selective script execution on top of global `ignore-scripts=true`:

**Setup**:
1. Install globally: `npm i -g @lavamoat/allow-scripts`
2. Run `allow-scripts setup` — this adds `ignore-scripts=true` to `.npmrc` and installs `@lavamoat/preinstall-always-fail` as a failsafe
3. Run `allow-scripts auto` to generate an initial allowlist

**Configuration** in `package.json`:
```json
{
  "lavamoat": {
    "allowScripts": {
      "sharp#0.33.2": true,
      "esbuild#0.20.1": true,
      "some-untrusted-package": false
    }
  }
}
```

**Key design decisions**:
- Allowed packages **require version pinning** (e.g., `sharp#0.33.2` not just `sharp`) — protects against maintainer compromise pushing a new malicious version
- Denied packages can omit versions to block across all versions
- Missing entries generate warnings, forcing explicit decisions
- The `can-i-ignore-scripts` tool helps evaluate which scripts are genuinely necessary

**Workflow integration**:
```json
{
  "scripts": {
    "setup": "npm install && npm exec allow-scripts && tsc -b"
  }
}
```

**Experimental bin script protection**: The `--experimental-bins` flag addresses a separate attack vector where malicious packages install bin scripts that shadow system executables in PATH.

### 2.4 pnpm's Superior Approach

pnpm v10+ provides a more integrated solution:

- **`strictDepBuilds: true`** (default since late 2025): Blocks all lifecycle scripts by default
- **`allowBuilds`** in `.npmrc` or `pnpm-workspace.yaml`:
  ```yaml
  allowBuilds:
    esbuild: true
    sharp: true
  ```
- No third-party tooling required — built into the package manager itself
- Combined with `minimumReleaseAge` (default 24 hours in pnpm v11) and `trustPolicy: no-downgrade`

### 2.5 Yarn Berry

Yarn Berry implements `enableHardenedMode` (enabled by default on GitHub PRs), which validates lockfile integrity against the registry. It also supports `dependenciesMeta` for per-package script configuration.

### 2.6 Bun

Bun blocks lifecycle scripts by default and supports per-package allowlists, similar to pnpm's model.

### 2.7 Cross-Manager Comparison

| Feature | npm CLI | pnpm v11 | Yarn Berry | Bun | Deno |
|---------|---------|----------|-----------|-----|------|
| Script blocking default | No | Yes | Yes | Yes | Yes |
| Per-package allowlist | No (needs @lavamoat) | Yes (built-in) | Yes | Yes | Yes |
| Version-pinned allowlist | Yes (@lavamoat) | No | No | No | Yes |
| Release cooldown default | No | Yes (1 day) | Yes (3 days) | Opt-in | No |

npm CLI has the fewest consumer-side protections of any major JS package manager.

---

## 3. Deno's Permission Model

Deno represents the most comprehensive runtime-level approach to package installation security.

### 3.1 Core Security Model

Deno is **secure by default** — code executes in a sandbox with zero OS access unless explicitly permitted. All permissions must be granted via command-line flags:

- `--allow-read[=path]` / `-R`: File system read access
- `--allow-write[=path]` / `-W`: File system write access
- `--allow-net[=host]` / `-N`: Network access
- `--allow-env[=var]` / `-E`: Environment variable access
- `--allow-run[=cmd]`: Subprocess execution
- `--allow-ffi`: Foreign function interface (native libraries)
- `--allow-sys`: System information access

Permissions can be granular: `--allow-read=./config.json` or `--allow-net=api.stripe.com`.

### 3.2 Deny Flags

All permissions support `--deny-*` counterparts that take precedence over allow flags:
```
deno run --allow-read=/etc --deny-read=/etc/hosts script.ts
```

### 3.3 Install Script Handling

Unlike npm, Deno does not automatically execute postinstall scripts. The `--allow-scripts` flag must be explicitly provided, and it supports per-package granularity:
```
deno install --allow-scripts=npm:sqlite3
```
This allows only sqlite3's postinstall script while blocking all others.

### 3.4 No Installation Phase

Deno's import-based dependency model means there is no separate "install" phase for most packages. Dependencies are fetched and cached on first import. This eliminates the install-time hook attack surface entirely for pure JavaScript/TypeScript packages.

### 3.5 Audit and Monitoring

- `DENO_AUDIT_PERMISSIONS=/path/to/log`: Writes JSONL audit logs of every permission check
- `DENO_TRACE_PERMISSIONS=1`: Generates stack traces for permission requests
- `DENO_PERMISSION_BROKER_PATH`: Delegates all permission decisions to an external process

### 3.6 Limitations

- All code on the same thread operates at identical privilege levels — you cannot restrict individual packages differently within a single execution context
- `--allow-run` subprocesses run with unrestricted access, bypassing the sandbox
- `--allow-ffi` native libraries execute outside the sandbox
- Network permissions are all-or-nothing per host (no protocol/port granularity)

---

## 4. Network Isolation During Builds

The two-phase install pattern — download dependencies with network access, then build/install with network blocked — is a powerful defense that works across all ecosystems.

### 4.1 The Pattern

1. **Phase 1 (online)**: Download all package archives/tarballs. Verify checksums and signatures. Scan with tools like Socket Firewall.
2. **Phase 2 (offline)**: Extract, build, and install from local cache with network access blocked. Any install script attempting to download additional payloads or exfiltrate data will fail.

This is how Codex Cloud implements package security: "setup runs before the agent phase and can access the network to install specified dependencies, then the agent phase runs offline by default."

### 4.2 Implementation Approaches

**Bubblewrap (bwrap)**:
```bash
bwrap --unshare-net --ro-bind / / --bind $PWD $PWD \
  npm install --ignore-scripts && npm run postinstall-allowed
```
Bubblewrap is the preferred tool for lightweight process-level isolation on Linux. It uses kernel namespaces (`--unshare-user`, `--unshare-pid`, `--unshare-net`) to create an isolated environment. Used by Flatpak, OpenAI Codex, and various CI/CD systems.

**Firejail**:
```bash
firejail --net=none npm install
```
Firejail v0.9.74+ includes experimental Landlock support. V0.9.80 (March 2026) added a new seccomp-bpf engine. Easier to use than bubblewrap for ad-hoc sandboxing but uses SUID binary which has a larger attack surface.

**Docker/OCI containers**:
```dockerfile
# Phase 1: Download with network
FROM node:20 AS deps
COPY package*.json ./
RUN npm ci

# Phase 2: Build without network (network disabled in build stage)
FROM node:20
COPY --from=deps /node_modules ./node_modules
COPY . .
RUN npm run build
```
Docker's `--network=none` flag can be used for build stages. This is the most common approach in CI/CD pipelines.

**Firecracker microVMs**:
For maximum isolation, run package installs in throwaway Firecracker microVMs. Boot time ~125ms, <5 MiB overhead per VM, up to 150 VMs/second/host. The hardware virtualization boundary (Intel VT-x / AMD-V) prevents entire classes of kernel-based escapes that container sandboxes cannot.

**gVisor**:
User-space kernel that intercepts syscalls. Middle ground between containers (shared kernel) and full VMs (hardware boundary). Lower overhead than Firecracker but stronger isolation than containers.

### 4.3 Per-Ecosystem Offline Build Support

- **npm**: `npm ci --cache .npm-cache --prefer-offline`; firejail/bubblewrap for network blocking
- **pip**: `pip install --no-index --find-links ./wheels/` after pre-downloading wheels
- **Cargo**: `cargo fetch` to download, then build with network blocked
- **Maven**: `mvn dependency:go-offline` then `mvn -o` (offline mode)
- **Go**: `GOFLAGS=-mod=vendor go build` with vendored dependencies

---

## 5. Sandboxing Tools and Approaches

### 5.1 Linux Kernel Security Primitives

**Landlock** (Linux 5.13+):
- Stackable Linux Security Module for unprivileged filesystem sandboxing
- Capability-based: processes can restrict their own access without root
- Used by OpenAI Codex as supplementary restriction alongside Bubblewrap
- Restricts filesystem access using `AccessFs` rules
- Cannot restrict network access (as of current kernel versions — network support in development)

**seccomp-BPF** (Linux 3.17+):
- Filters system calls at the kernel level using BPF programs
- Can block specific syscalls: `ptrace`, `io_uring_*`, network operations (`connect`, `bind`, `listen`)
- Used by Codex, Firejail, and Chrome/Firefox for sandbox enforcement
- Very low overhead — filter runs in kernel space

**AppArmor**:
- Mandatory Access Control framework
- Profile-based: define per-application filesystem, network, and capability restrictions
- Could be used to create profiles for package managers, but no pre-built profiles exist for npm/pip/cargo
- Requires root to load profiles

### 5.2 Container-Based Isolation

Running `npm install` or `pip install` inside a container provides:
- Filesystem isolation (changes don't persist to host)
- Optional network isolation (`--network=none`)
- User namespace isolation
- Resource limits (CPU, memory)

**Limitations**: Containers share the host kernel. A kernel exploit in a build script could escape the container. This is the "minimum viable" isolation level.

### 5.3 VM-Based Approaches

**Throwaway VMs**: Run package installs in ephemeral VMs that are destroyed after build artifacts are extracted.

**Firecracker microVMs**: Purpose-built for multi-tenant workloads. Hardware virtualization boundary. ~125ms boot, <5 MiB overhead. Used by AWS Lambda and Fly.io. The strongest isolation available, but adds operational complexity.

**Kata Containers**: OCI-compatible containers backed by lightweight VMs. Drop-in replacement for standard containers with VM-level isolation.

### 5.4 Cackle (Rust-specific)

Cackle provides the most sophisticated per-ecosystem sandboxing available today:
- Runs build scripts in individual Bubblewrap sandboxes with configurable permissions
- Analyzes compiled binaries to detect API usage (net, fs, process, unsafe)
- Each build script can have different sandbox permissions
- Proc macros are sandboxed by running `rustc` in a sandbox (limitation: all proc macros share the same permissions)
- Configured via `cackle.toml` with CI integration (`cargo acl -n`)
- Empirical finding: ~50% of dependencies require no special permissions

### 5.5 Codex Sandbox Architecture (Reference Implementation)

OpenAI's Codex provides a well-documented reference for multi-platform sandboxing:

**Linux**: Bubblewrap (primary) + Landlock (fallback) + seccomp
- Isolated user, PID, and network namespaces
- Read-only filesystem by default with layered writable roots
- Three network modes: Isolated, ProxyOnly, FullAccess

**macOS**: Seatbelt (`sandbox-exec`)
- Apple's native sandboxing framework
- Dynamically generated Sandbox Profile Language scripts
- All-or-nothing network access (no domain granularity)

**Windows**: Restricted tokens + ACLs
- Preflight audit for world-writable directories
- Process execution with restricted token

---

## 6. Emerging Solutions

### 6.1 Socket Firewall (`sfw`)

Socket Firewall is a zero-configuration install-time firewall that wraps package manager commands:
```bash
npm i -g sfw
sfw npm install      # JavaScript
sfw uv pip install flask   # Python
sfw cargo build      # Rust
```

**How it works**: Operates as an ephemeral HTTP proxy intercepting package manager traffic. Checks packages against Socket's API for known malware, obfuscated code, typosquatting, and suspicious network behavior before allowing download.

**Coverage**: npm, yarn, pnpm, pip, uv, cargo (free). Maven, Gradle, gem, Bundler, NuGet (enterprise).

**Limitations of free tier**: No custom registry support, AI-detected malware generates warnings not blocks, no allow-listing, no policy configuration.

### 6.2 Rust Sandboxed Build Scripts (In Progress)

The Rust project's 2024 H2 goal to sandbox build scripts is exploring:
- **WASI (WebAssembly System Interface)**: Run build scripts as WASM modules with capability-based permissions
- **Declarative permission grants**: Configuration in `Cargo.toml` specifying what each build script needs
- **crates.io integration**: Displaying permission requirements on package pages
- Targeted for default-on by the next Rust Edition

### 6.3 pip `--only-binary` and Time-Based Defenses

Python's emerging defensive stack:
- **`pip install --only-binary :all:`** or **`uv pip install --only-binary :all:`**: Refuses to install any source distribution, eliminating `setup.py` execution entirely. Environment variable: `PIP_ONLY_BINARY=:all:`
- **`uv --exclude-newer "7 days"`**: Only resolves packages published before a specified time window
- **`pip --uploaded-prior-to`** (v26+): Equivalent time-based filtering
- **`pip install --require-hashes`**: Requires cryptographic hash verification for every package

### 6.4 Birdcage (Phylum)

Open-source sandbox from Phylum, baked into the Phylum CLI. Restricts filesystem and network operations during package installation. Available as extensions for pip and Poetry. Designed specifically for neutralizing malicious install-time code.

### 6.5 pnpm's Integrated Security Stack

pnpm v10-v11 represents the most mature "configure-once" approach in any JavaScript package manager:
- `strictDepBuilds: true` (default) — blocks all lifecycle scripts
- `allowBuilds` — explicit per-package allowlist
- `minimumReleaseAge: 1440` (default in v11) — 24-hour quarantine on new versions
- `trustPolicy: no-downgrade` — blocks packages with weaker auth than previous versions
- `blockExoticSubdeps` — prevents transitive deps from using git repos or tarballs

All configured in `.npmrc` or `pnpm-workspace.yaml` and enforced automatically.

---

## 7. Tradeoffs: What Breaks With Sandboxing

### 7.1 Native Module Compilation

The most common legitimate use of install scripts is compiling native C/C++ addons:
- **npm**: `bcrypt`, `sharp`, `canvas`, `sqlite3`, `node-sass` all require postinstall compilation
- **Python**: Any package without a pre-built wheel for the target platform requires `setup.py` execution
- **Rust**: `-sys` crates (e.g., `openssl-sys`, `libz-sys`) use `build.rs` to find and link system libraries
- **Ruby**: Gems with C extensions require `extconf.rb`

**Mitigation**: Use pre-built binaries where available. npm packages increasingly ship platform-specific binaries (esbuild, SWC, Prisma). Python's wheel ecosystem covers most platforms for popular packages. The `--only-binary` flag makes this explicit.

### 7.2 Binary Downloads

Many tools download platform-specific binaries during postinstall:
- **esbuild**: Downloads Go-compiled binary
- **Prisma**: Downloads query engine binary
- **Playwright**: Downloads browser binaries
- **Puppeteer**: Downloads Chromium

These require both script execution AND network access. With `ignore-scripts=true`, they must be handled via allowlists or separate installation steps.

### 7.3 Code Generation and Setup

Some packages perform necessary setup during install:
- **Prisma**: Generates client code from schema
- **protobuf**: Generates language bindings
- **Various**: Run `prepare` scripts to compile TypeScript to JavaScript

**Mitigation**: Move these to explicit build steps rather than install hooks. Use `npm run setup` instead of `postinstall`.

### 7.4 How Teams Handle This in Practice

1. **Start with `ignore-scripts=true` globally** — catches the 97.8% of packages that don't need scripts
2. **Use an allowlist tool** (`@lavamoat/allow-scripts` for npm, `allowBuilds` for pnpm) to selectively enable the 2.2% that do
3. **Audit allowed packages** — review what their scripts actually do. Tools like `can-i-ignore-scripts` help evaluate necessity
4. **Pin allowed versions** — if using `@lavamoat/allow-scripts`, pin to specific audited versions
5. **Run in CI with network isolation** — even for allowed scripts, block outbound network in the build phase
6. **Prefer pre-built binaries** — choose packages that ship platform-specific binaries over those that compile from source
7. **Document exceptions** — every allowlist entry should have a rationale for audit trails

### 7.5 The Fundamental Tension

The core tradeoff is between security and convenience. Go proves that an ecosystem can function without install-time code execution, but it was designed this way from the start. Retrofitting this into ecosystems with decades of install-script-dependent packages requires the allowlist approach — deny by default, permit by exception.

The second tension is between process-level sandboxing (bubblewrap, Landlock) and ecosystem-level controls (ignore-scripts, allowBuilds). Process-level sandboxing is more secure but harder to configure correctly and may break legitimate operations. Ecosystem-level controls are easier to deploy but can be bypassed by vulnerabilities in the package manager itself.

The practical recommendation is to layer both: ecosystem-level script blocking as the primary defense, with process/container-level isolation as defense-in-depth.

---

## 8. Ecosystem Maturity Comparison

| Ecosystem | Install-time code execution | Default protection | Best available defense | Maturity |
|-----------|---------------------------|-------------------|----------------------|----------|
| **Go** | None (by design) | Full | N/A — solved at design level | Gold standard |
| **Deno** | Blocked by default | Full | `--allow-scripts=npm:pkg` per-package | Excellent |
| **pnpm** | Blocked by default (v10+) | High | `allowBuilds` + `minimumReleaseAge` + `trustPolicy` | Excellent |
| **Yarn Berry** | Blocked by default | High | `dependenciesMeta` + `enableHardenedMode` | Good |
| **Bun** | Blocked by default | High | Per-package allowlist | Good |
| **npm CLI** | Allowed by default | None | `ignore-scripts=true` + `@lavamoat/allow-scripts` | Adequate (requires setup) |
| **Python/pip** | sdists execute code | None | `--only-binary :all:` + `--require-hashes` | Adequate (requires discipline) |
| **Python/uv** | sdists execute code | None | `--only-binary :all:` + `--exclude-newer` | Better than pip |
| **Rust/Cargo** | `build.rs` + proc macros | None | Cackle (third-party) | Poor (active work) |
| **Ruby** | `extconf.rb` executes | None | Container isolation | Poor |
| **Maven/Gradle** | Build plugins execute | None | Ephemeral build environments | Poor |
| **NuGet** | `init.ps1` in VS | Partial (PackageReference removes `install.ps1`) | Migrate to PackageReference | Adequate |

---

## Sources

All raw source material saved to `docs/`:

- `docs/npm-ignore-scripts-best-practices.md` — npm ignore-scripts configuration guide
- `docs/npm-supply-chain-defenses-2026.md` — Cross-manager comparison of JS ecosystem defenses
- `docs/lavamoat-allow-scripts.md` — @lavamoat/allow-scripts configuration and workflow
- `docs/pnpm-supply-chain-security.md` — pnpm security features overview
- `docs/pnpm-build-script-security.md` — pnpm build script security architecture
- `docs/deno-protects-npm-exploits.md` — Deno's sandbox-first approach to npm packages
- `docs/deno-security-permissions.md` — Deno permission system reference
- `docs/python-package-installation-attacks.md` — Python sdist/wheel attack vectors
- `docs/python-supply-chain-defense-guide.md` — Python defensive measures guide
- `docs/pypi-security-best-practices.md` — 17 PyPI security best practices
- `docs/rust-sandboxed-build-scripts.md` — Rust project goals for build script sandboxing
- `docs/rust-build-security-supply-chain.md` — Rust build environment security guide
- `docs/cargo-build-script-allowlist-issue.md` — Cargo allowlist proposal (issue #13681)
- `docs/cackle-rust-supply-chain.md` — Cackle ACL checker for Rust
- `docs/go-supply-chain-mitigations.md` — Go's design-level supply chain protections
- `docs/socket-firewall-overview.md` — Socket Firewall install-time protection
- `docs/codex-sandboxing-implementation.md` — Codex sandbox architecture reference
- `docs/agent-sandbox-deep-dive.md` — Agent sandbox techniques and limitations
