# Lock File Integrity and Reproducible Installs as Supply Chain Defense

## Executive Summary

Lock files and hash-pinned dependencies are the single most impactful "configure-once, invisible-in-operation" defense against package supply chain attacks. When enforced correctly, they eliminate an entire class of attacks where malicious code enters through version resolution drift — an attacker publishes a compromised package version, and the next `install` silently picks it up because the manifest allows a range. Lock file enforcement freezes the dependency graph at a known-good state and (in most ecosystems) cryptographically verifies that each downloaded artifact matches its recorded hash, detecting registry compromise and MITM attacks.

However, lock files also introduce their own attack surface: lockfile poisoning, where an attacker submits a PR that subtly modifies the lock file to redirect dependencies to malicious sources. Mitigating this requires tooling (lockfile linting, hardened modes) and process (CI enforcement, code review policies).

This report covers lock file mechanics across nine ecosystems, hash verification capabilities, CI enforcement patterns, reproducible build strategies, known attacks against lock files, and the tradeoffs of strict enforcement.

---

## 1. Lock File Mechanics Per Ecosystem

### 1.1 npm (Node.js)

**Lock file**: `package-lock.json`

**What it records**: Complete dependency tree with exact versions, resolved URLs, and SHA-512 integrity hashes for every package. Also includes license and funding metadata (which bloats diffs and complicates review).

**Key commands**:
- `npm install` — Resolves dependencies within semver ranges; creates/updates lock file. **Never use in CI.**
- `npm ci` — Installs exactly what is in the lock file. Deletes `node_modules/` first. Fails if `package.json` and lock file diverge. 2-10x faster than `npm install`.

**Hash verification**: Built-in. Every entry includes an `integrity` field with a `sha512-...` subresource integrity hash. npm verifies downloaded tarballs against these hashes at install time. If a registry-side replacement attack changes a package's contents under the same version, the hash mismatch causes installation to fail.

**CI enforcement**: Use `npm ci` in all CI pipelines. No additional flags needed — it is inherently frozen.

**Configuration for "configure-once"**:
```ini
# .npmrc
save-exact=true          # Pin versions without ^ or ~ ranges
ignore-scripts=true      # Disable lifecycle scripts
```

### 1.2 Yarn

**Lock file**: `yarn.lock`

**Yarn Classic (v1)**:
- `yarn install --frozen-lockfile` — Fails if the lock file needs updating
- Does NOT include integrity hashes in `yarn.lock` (hashes are verified via a separate cache mechanism)

**Yarn Berry (v2+/v4)**:
- `yarn install --immutable` — Equivalent of frozen-lockfile; prevents lock file modification
- `yarn install --immutable --immutable-cache` — Also prevents cache modifications
- **Hardened Mode** (v4.0+): Automatically enables `--check-resolutions` and `--refresh-lockfile`, verifying that every resolution in the lock file is valid for the original range and that metadata matches the registry. Automatically activated on GitHub PRs for public repos. Can be manually enabled via `enableHardenedMode: true` in `.yarnrc.yml` or `YARN_ENABLE_HARDENED_MODE=1`.
- **Age Gate** (v4.12+): `npmMinimalAgeGate` setting refuses packages published fewer than N days ago. `npmPreapprovedPackages` allows exceptions for trusted packages.
- **Scripts disabled by default** (v4.14+): Postinstall scripts require explicit opt-in per package.

**CI enforcement**: Use `yarn install --immutable` in CI. Enable hardened mode for open-source projects.

### 1.3 pnpm

**Lock file**: `pnpm-lock.yaml`

**Key behaviors**:
- `pnpm install --frozen-lockfile` — Fails if the lock file is out of date. **Enabled by default when `CI` environment variable is set.**
- pnpm does NOT store tarball source URLs in the lock file (unlike npm), which means lockfile poisoning via URL replacement is not possible. This is a significant structural advantage.
- Content-addressable storage means pnpm verifies package content by hash intrinsically.

**Unique feature**: `minimumReleaseAge` (v10.16+) — refuses to install package versions published less than a configurable number of minutes ago. This creates a time-based quarantine buffer.

```ini
# .npmrc
minimum-release-age=86400000  # 24 hours in milliseconds
```

**CI enforcement**: Automatic when `CI=true` is set (frozen-lockfile by default).

### 1.4 pip / Python

**Lock files**: Python has historically lacked a standard lock file format. Multiple approaches exist:

**requirements.txt with hashes** (pip-tools / uv):
```
flask==3.1.1 \
    --hash=sha256:d667207822abcdef...
```
- `pip install --require-hashes -r requirements.txt` — Fails if any package lacks a hash or if downloaded content doesn't match.
- **Critical gotcha**: pip only enforces hash checking when EVERY requirement has a hash, OR when `--require-hashes` is passed explicitly. A single unhashed line silently disables verification for that package.

**uv** (recommended modern tooling):
- `uv lock` — Generates `uv.lock` with full hash pinning
- `uv sync` — Installs from lock file
- `uv pip compile --generate-hashes` — Generates hashed requirements.txt
- `uv pip compile --exclude-newer <date>` — Time-based quarantine

**Poetry**:
- `poetry.lock` — Contains SHA-256 hashes for all packages
- `poetry install` — Strict enforcement by default; halts if lock file conflicts with `pyproject.toml`
- Hashes are verified during installation

**Pipenv**:
- `Pipfile.lock` — JSON format with SHA-256 hashes
- `pipenv install --deploy` — Fails if `Pipfile.lock` is out of date or missing
- Standard `pipenv install` silently updates the lock file (insecure default)

**PEP 751 / pylock.toml** (accepted March 2025, early adoption phase):
- New standard lock file format for Python
- **Mandatory hashes** — security by default, not opt-in
- No resolver needed at install time — fully deterministic
- `pip lock` command being added to pip 25.1+
- Supported by pip-audit 2.9.0+ for vulnerability scanning

**CI enforcement**: Use `pip install --require-hashes -r requirements.txt` or `uv sync --frozen`. For Poetry: default behavior is already strict.

### 1.5 Cargo (Rust)

**Lock file**: `Cargo.lock`

**What it records**: Exact versions, source registry/git URLs, and SHA-256 checksums for all crate dependencies. The checksum is computed by crates.io before adding to the registry index.

**Key behaviors**:
- `cargo build` — Respects existing `Cargo.lock` but will silently update it if new dependencies are added. **This is insecure for CI.**
- `cargo build --locked` — Fails if `Cargo.lock` is missing or would need updating. **Use this in CI.**
- `cargo build --frozen` — Like `--locked` but also prevents network access entirely.

**When to commit**: Always commit `Cargo.lock` for applications/binaries. For libraries, the Rust community increasingly recommends committing it too (reversal from earlier guidance), as it enables accurate vulnerability scanning.

**Hash verification**: Cargo verifies downloaded crate tarballs against the checksum recorded in both the registry index and `Cargo.lock`. A registry compromise that replaces a crate with different content would be detected.

**cargo-vet**: Mozilla's tool for auditing third-party crate usage. Enforces that every new dependency has been reviewed by a trusted entity. Results can be shared across organizations.

**CI enforcement**: `cargo build --locked` (or `--frozen` for air-gapped builds).

### 1.6 Go Modules

**Files**: `go.mod` + `go.sum`

**Critical distinction**: `go.sum` is NOT a lock file. It has zero semantic effect on version resolution. `go.mod` is the actual version-pinning mechanism — since Go 1.17 it lists all transitive dependencies needed for building.

**How `go.sum` works**: It is a local cache of cryptographic hashes (SHA-256) for every module version the project has encountered. These hashes are verified against the **Go Checksum Database** (`sum.golang.org`), a transparency log that ensures the entire Go ecosystem shares the same contents for any given module version.

**Key behaviors**:
- Since Go 1.16, the default behavior is `-mod=readonly` — the go command reports an error if `go.mod` would need changes
- `go mod verify` — Checks that downloaded modules match their checksums in `go.sum`
- `go mod download` — Downloads modules and records hashes

**Environment variables**:
- `GOFLAGS=-mod=readonly` — Enforces that go.mod cannot be modified (default since 1.16)
- `GONOSUMDB=<patterns>` — Skip checksum database lookups for matching modules (useful for private modules)
- `GOPROXY=off` — Disable all network access; use only cached modules

**CI enforcement**: `go mod verify` in CI to confirm downloaded modules match committed `go.sum`. The readonly default since Go 1.16 means CI already fails if `go.mod` would need updating.

**Go's design advantage**: By separating version pinning (go.mod) from integrity verification (go.sum + checksum database), Go achieves ecosystem-wide tamper detection — not just per-project, but globally. If a module's content changes for a given version anywhere, sum.golang.org detects it.

### 1.7 Maven (Java)

**Lock file**: None built-in. Maven has no native lock file support.

**Third-party solution**: `maven-lockfile` plugin (chains-project)
- `mvn io.github.chains-project:maven-lockfile:generate` — Creates `lockfile.json` with SHA-256 checksums of all direct and transitive dependencies, POM files, and optionally Maven plugins
- `mvn io.github.chains-project:maven-lockfile:validate` — Fails if dependencies have changed since lock file generation
- `mvn io.github.chains-project:maven-lockfile:freeze` — Generates `pom.lockfile.xml` with all versions pinned

**Maven Enforcer Plugin**: Can enforce version convergence and ban specific dependencies, but does not provide hash verification.

**CI enforcement**: Run `mvn io.github.chains-project:maven-lockfile:validate` in CI. GitHub Action available for automated lockfile management.

**Warning**: Results can be platform-dependent — some artifacts have platform-specific checksums.

### 1.8 Gradle (Java/Kotlin)

**Lock file**: `gradle.lockfile`

**Critical gap**: Gradle lock files do NOT include checksums. They only record `group:artifact:version` — no hash verification. This is a significant security weakness compared to other ecosystems.

**Enabling** (opt-in, not default):
```kotlin
dependencyLocking {
    lockAllConfigurations()
}
```

**Generating**: `./gradlew dependencies --write-locks`

**Lock modes**:
- **Default**: Validates entries match and no extras exist
- **Strict**: Fails if any locked configuration lacks lock state
- **Lenient**: Pins dynamic versions but allows additions/removals

**Adoption**: Only 0.9% of Gradle projects on GitHub include lock files (per the arXiv empirical study), reflecting the opt-in approach and suggesting most Java/Kotlin projects using Gradle lack this protection entirely.

**CI enforcement**:
```kotlin
dependencyLocking {
    lockAllConfigurations()
    lockMode = LockMode.STRICT
}
```

### 1.9 NuGet (.NET)

**Lock file**: `packages.lock.json`

**What it records**: Direct and transitive dependencies with requested vs. resolved versions, SHA-512 content hashes, and dependency chains organized by target framework.

**Enabling** (opt-in, not default):
```xml
<PropertyGroup>
    <RestorePackagesWithLockFile>true</RestorePackagesWithLockFile>
</PropertyGroup>
```

**Enforcement**:
```xml
<PropertyGroup>
    <RestoreLockedMode>true</RestoreLockedMode>
</PropertyGroup>
```

When `RestoreLockedMode=true`, restore fails if the lock file is out of sync — preventing unintended package updates.

**CI enforcement**: Set `RestoreLockedMode=true` in CI or pass `--locked-mode` to `dotnet restore`.

**CLI equivalents**: `--use-lock-file`, `--locked-mode`, `--force-evaluate`

### 1.10 Bundler (Ruby)

**Lock file**: `Gemfile.lock`

**Hash verification** (Bundler 2.6+, December 2024):
- `bundle lock --add-checksums` — Adds a `CHECKSUMS` section to `Gemfile.lock` with SHA-256 hashes
- `bundle config lockfile_checksums true` — Always include checksums in new lock files
- Bundler verifies checksums on every install; mismatches block installation

**Frozen install**:
- `bundle install --frozen` — Fails if `Gemfile.lock` would need updating
- `bundle config set --local deployment true` — Equivalent persistent setting

**CI enforcement**: `bundle install --frozen` in CI pipelines.

**Note**: Bundler 4 (2025) does NOT automatically add checksums to existing lock files — you must explicitly request them. New projects should configure `lockfile_checksums true` from the start.

---

## 2. Hash Verification Comparison

| Ecosystem | Lock File | Hash Algorithm | Hashes Included | Verified Against |
|-----------|-----------|---------------|-----------------|-----------------|
| npm | package-lock.json | SHA-512 | Yes (integrity field) | Downloaded tarball |
| Yarn Classic | yarn.lock | N/A | No (cache-based) | Cache mechanism |
| Yarn Berry | yarn.lock | SHA-512 | Yes | Downloaded tarball |
| pnpm | pnpm-lock.yaml | SHA-512 | Yes | Content-addressable store |
| pip (hashed) | requirements.txt | SHA-256 | Yes (opt-in) | Downloaded wheel/sdist |
| Poetry | poetry.lock | SHA-256 | Yes | Downloaded wheel/sdist |
| uv | uv.lock | SHA-256 | Yes | Downloaded wheel/sdist |
| pylock.toml | pylock.toml | SHA-256+ | Yes (mandatory) | Downloaded archive |
| Cargo | Cargo.lock | SHA-256 | Yes (checksum field) | Registry index + tarball |
| Go | go.sum | SHA-256 | Yes | Checksum database (global) |
| Maven (plugin) | lockfile.json | SHA-256 | Yes | Downloaded artifact |
| Gradle | gradle.lockfile | N/A | **No** | N/A |
| NuGet | packages.lock.json | SHA-512 | Yes | Downloaded nupkg |
| Bundler 2.6+ | Gemfile.lock | SHA-256 | Yes (opt-in CHECKSUMS) | Downloaded gem |

### What Hashes Prevent

1. **Registry compromise**: An attacker who gains access to a registry backend and replaces a package tarball with malicious content under the same version number. The hash in the lock file won't match the tampered content.

2. **MITM attacks**: An attacker intercepting network traffic between the developer/CI and the registry and substituting a malicious package. Hash verification catches the substitution.

3. **Mirror/proxy tampering**: A compromised internal mirror serving altered packages. Hash verification at the client detects modifications.

4. **Silent re-publication**: A maintainer (or compromised maintainer account) replacing existing published content. Some registries allow this; hash verification detects it client-side.

### What Hashes Do NOT Prevent

- **Initial trust**: If you first install a malicious package and record its hash, the hash will faithfully verify the malicious content forever.
- **Typosquatting**: Hashes verify content integrity, not package identity.
- **Lockfile poisoning**: An attacker can change both the resolved URL AND the hash in a lockfile PR.

---

## 3. CI Enforcement Quick Reference

### Configure-Once Settings (per ecosystem)

**npm**: Use `npm ci` in all CI scripts. Add to `.npmrc`:
```ini
save-exact=true
ignore-scripts=true
```

**Yarn Berry**: Add to `.yarnrc.yml`:
```yaml
enableImmutableInstalls: true
enableHardenedMode: true
enableScripts: false
```

**pnpm**: Automatic — `--frozen-lockfile` is default when `CI=true`. Add to `.npmrc`:
```ini
minimum-release-age=86400000
```

**Python (uv)**: `uv sync --frozen` in CI. Generate with `uv lock`.

**Python (pip)**: `pip install --require-hashes -r requirements.txt` in CI.

**Rust**: `cargo build --locked` in CI.

**Go**: Already readonly by default since Go 1.16. Add `go mod verify` to CI.

**Gradle**:
```kotlin
dependencyLocking {
    lockAllConfigurations()
    lockMode = LockMode.STRICT
}
```

**NuGet**: Add to `.csproj`:
```xml
<RestorePackagesWithLockFile>true</RestorePackagesWithLockFile>
<RestoreLockedMode>true</RestoreLockedMode>
```

**Bundler**: `bundle install --frozen` in CI. Configure `bundle config lockfile_checksums true`.

**Maven**: `mvn io.github.chains-project:maven-lockfile:validate` in CI.

**Bazel**: `--lockfile_mode=error` in CI `.bazelrc`:
```
build --lockfile_mode=error
```

### Dockerfile Patterns

```dockerfile
# Node.js
COPY package.json package-lock.json ./
RUN npm ci --production

# Python
COPY requirements.txt ./
RUN pip install --no-cache-dir --require-hashes -r requirements.txt

# Rust
COPY Cargo.toml Cargo.lock ./
RUN cargo build --release --locked

# Go
COPY go.mod go.sum ./
RUN go mod download -x && go mod verify

# Ruby
COPY Gemfile Gemfile.lock ./
RUN bundle install --frozen
```

Pin base images by digest, not tag:
```dockerfile
FROM node:20-alpine@sha256:a1b2c3d4e5f6...
```

---

## 4. Reproducible Builds

### Bazel

Bazel provides the strongest reproducible build guarantees through its lockfile system:

- `MODULE.bazel.lock` records resolved dependency versions, registry file hashes, and module extension results
- `--lockfile_mode=error` in CI prevents any network requests during resolution and fails if the lockfile is stale
- `rules_jvm_external` integrates with Maven Central and generates `maven_install.json` pinning artifacts with SHA-256 checksums
- The content-addressable download cache ensures bit-for-bit identical artifacts

### Container-Based Reproducibility

Full reproducibility in Docker requires discipline at every layer:

1. **Base image**: Pin by digest (`@sha256:...`), not tag
2. **System packages**: Pin to exact versions (`curl=7.88.1-10+deb12u5`)
3. **Application dependencies**: Use ecosystem lock files with hash verification (all the CI patterns above)
4. **Timestamps**: Use `SOURCE_DATE_EPOCH` with BuildKit for deterministic timestamps
5. **Build context**: Use `.dockerignore` to exclude variable files

Verification: build twice with `--no-cache` and compare image digests.

### Key Insight

True reproducibility requires pinning at EVERY layer — a locked `package-lock.json` inside a `FROM node:latest` container is still non-reproducible because the base image can change. The entire dependency chain (OS, runtime, libraries, application dependencies) must be frozen.

---

## 5. Lock File Attacks

### 5.1 Lockfile Poisoning (via PR Manipulation)

**How it works**: An attacker submits a pull request that modifies the lock file to redirect a dependency to a malicious source. The PR appears to make a minor code change, but the lock file diff (often thousands of lines, hidden by default on GitHub/GitLab) contains a subtle resolution URL change pointing to an attacker-controlled registry or git repository, with a matching hash for the malicious payload.

**Why it succeeds**:
- GitHub collapses diffs exceeding a few hundred lines by default
- Lock file diffs are machine-generated and rarely reviewed character-by-character
- Both the artifact URL and integrity hash travel through the same channel (the lock file), so an attacker can update both consistently

**Affected ecosystems**:
- **npm**: Most vulnerable — `package-lock.json` stores resolved URLs directly. An attacker can change the `resolved` field to point to any URL and update the `integrity` hash to match.
- **Yarn Classic**: Vulnerable — stores resolution URLs in `yarn.lock`
- **pnpm**: Structurally resistant — `pnpm-lock.yaml` does NOT store tarball source URLs, so there is no URL to redirect. This is a significant design advantage.
- **Other ecosystems**: Generally less vulnerable because they resolve against a fixed registry rather than storing per-package URLs

**Attack variants**:
1. **URL tampering**: Change `resolved` URL to attacker-controlled server, update hash to match malicious payload
2. **Phantom dependency injection**: Add entirely new dependencies to the lock file that aren't in the manifest — some package managers will install them anyway
3. **Version downgrade**: Change the locked version to an older, known-vulnerable version

### 5.2 Mitigations for Lockfile Poisoning

**Tooling**:
- **lockfile-lint** (Liran Tal / Snyk): CI tool that validates npm/Yarn lock files against security policies. Checks that all resolved URLs use HTTPS and point to allowed registries.
  ```bash
  npx lockfile-lint --path package-lock.json --type npm \
    --validate-https --allowed-hosts npm
  ```
- **Yarn Hardened Mode**: Automatically validates that lock file resolutions match registry metadata. Detects URL and version manipulation.
- **SafeDep vet**: Validates that package source URLs originate from trusted registries and that URL paths match expected structures.

**Process**:
- Require CODEOWNERS approval for lock file changes
- Configure CI to flag lock file modifications for manual review
- Use branch protection to prevent direct commits to lock files
- For open-source projects: never merge PRs from untrusted contributors without scrutinizing lock file changes

**Structural**:
- Use pnpm (which doesn't store tarball URLs in the lock file)
- Use a private registry/proxy that all packages must come from (the lock file URL becomes irrelevant if all downloads are forced through a controlled proxy)

### 5.3 Lock File Confusion Attacks

Some less mature ecosystems have edge cases where lock file behavior is counterintuitive:

- **Cargo**: `cargo build` without `--locked` will silently update `Cargo.lock` if dependencies change — masking supply chain attacks in CI
- **Pipenv**: Standard `pipenv install` ignores the lock file and regenerates it — only `--deploy` enforces it
- **Gradle**: Without explicit locking configuration, no lock file exists at all — 99.1% of Gradle projects lack this protection

---

## 6. Tradeoffs and Operational Considerations

### What Breaks with Strict Lock File Enforcement

1. **Dependency updates require intentional action**: You cannot simply edit `package.json` and run the build — you must explicitly update the lock file first (`npm install`, `uv lock`, `cargo update`, etc.)

2. **Yanked/deleted versions**: If a dependency version is yanked from the registry, the lock file still points to it but installation may fail. This requires regenerating the lock file.

3. **Platform-specific dependencies**: Some lock files (notably Maven, npm) record platform-specific artifacts. A lock file generated on macOS may not work on Linux if the dependencies include platform-specific binaries.

4. **Merge conflicts**: Lock file changes create frequent merge conflicts, especially in projects with many active contributors. Large lock files (npm, Poetry) generate enormous diffs.

5. **CI performance**: Yarn's Hardened Mode is "significantly slower" because it verifies every resolution against the registry. Bazel's `--lockfile_mode=error` prevents any network requests, so the lock file must be comprehensive.

6. **Learning curve**: npm's dependency resolution is described by developers as "still kind of a mystery" with "cryptic" error messages. Lock file problems can be difficult to debug.

### How Teams Handle Legitimate Updates

**Automated dependency updates** (Dependabot, Renovate, Mend):
- These tools submit PRs that update both the manifest and lock file together
- CI validates the new lock file, runs tests, and merges automatically if all checks pass
- This separates "intentional update" from "accidental drift"

**Scheduled update cadence**:
- Many teams designate a weekly or monthly window for dependency updates
- Lock file changes only happen in dedicated update PRs, making review tractable
- Security updates may bypass the cadence via automated scanning

**Separation of environments**:
- Development uses flexible ranges (`pyproject.toml` with `>=` constraints)
- Deployment uses locked/hashed output (`requirements.txt` with `--require-hashes`)
- `uv export`, `pip-compile`, or Poetry's lock mechanism bridges the gap

**Lock file regeneration on demand**:
- When a lock file becomes too stale or conflicted, teams delete and regenerate it
- This is safe IF the result is reviewed and tested before merging
- Automated CI verification (tests, vulnerability scanning, lockfile-lint) provides the safety net

---

## 7. Ecosystem Maturity Ranking

Ranking ecosystems by the strength of their lock file + hash verification as a supply chain defense, from strongest to weakest:

1. **Go** — Unique: ecosystem-wide checksum transparency log (`sum.golang.org`) means the entire community shares hash verification, not just individual projects. Version pinning in `go.mod` is deterministic by default since Go 1.16.

2. **pnpm** — Strongest in JavaScript: frozen-lockfile by default in CI, content-addressable storage, no tarball URLs in lock file (immune to lockfile poisoning URL attacks), built-in `minimumReleaseAge`.

3. **Cargo/Rust** — Strong: SHA-256 checksums in both registry index and `Cargo.lock`, `--locked`/`--frozen` flags, cargo-vet for audit sharing. Weakness: default `cargo build` silently updates lock file.

4. **Yarn Berry** — Strong: Hardened Mode auto-detects PR context and validates resolutions, age gate, scripts disabled by default. Performance cost for hardened mode.

5. **npm** — Moderate: Good hash verification, `npm ci` is excellent, but `package-lock.json` stores resolved URLs (vulnerable to lockfile poisoning). Requires discipline to use `npm ci` instead of `npm install`.

6. **Poetry** — Moderate: Strict enforcement by default, SHA-256 hashes. Python ecosystem still fragmented across tools.

7. **uv** — Moderate-to-strong: Modern tooling with hash generation, `--exclude-newer` for time-based quarantine, `--frozen` flag. Rapidly maturing.

8. **NuGet** — Moderate: SHA-512 hashes, `RestoreLockedMode` for CI. Weakness: opt-in, not default. Cross-tool compatibility issues between `nuget.exe` and Visual Studio.

9. **Bundler** — Moderate: Checksum verification added in v2.6 (late 2024), but opt-in. `--frozen` flag for CI.

10. **Pipenv** — Weak: Hashes present in `Pipfile.lock` but standard install ignores the lock file. `--deploy` flag required but easy to forget.

11. **Maven** — Weak: No native lock file support. Third-party plugin required. Platform-dependent checksums.

12. **Gradle** — Weakest: Opt-in locking, no checksums, 0.9% adoption rate. Most Gradle projects have zero lock file protection.

---

## 8. Recommendations: Configure-Once Defenses

For an organization seeking to set up lock file integrity as a "configure-once, invisible-in-operation" defense:

### Immediate Actions (Per-Project)

1. **Commit lock files** for every project, including libraries
2. **Enable hash verification** where it's opt-in (Bundler `lockfile_checksums`, pip `--require-hashes`, NuGet `RestorePackagesWithLockFile`)
3. **Use strict install commands in CI**: `npm ci`, `--frozen-lockfile`, `--locked`, `--immutable`
4. **Add lockfile-lint** to JavaScript CI pipelines to detect lockfile poisoning

### Organizational Policies

5. **CODEOWNERS for lock files**: Require approval from security-aware reviewers for any lock file change
6. **Automated dependency updates**: Dependabot/Renovate for controlled, reviewable updates
7. **Private registry/proxy**: Route all package installations through a controlled proxy (Artifactory, Verdaccio, etc.) — makes lockfile URL manipulation irrelevant
8. **CI gates**: Every CI pipeline should fail on lock file drift; no exceptions

### Ecosystem-Specific Setup Files

Create standardized dotfiles that enforce these defaults across all projects:

```ini
# .npmrc (npm/pnpm)
save-exact=true
ignore-scripts=true
```

```yaml
# .yarnrc.yml (Yarn Berry)
enableImmutableInstalls: true
enableHardenedMode: true
enableScripts: false
npmMinimalAgeGate: 7
```

```toml
# pyproject.toml or tool config
[tool.uv]
require-hashes = true
```

```xml
<!-- Directory.Build.props (NuGet, solution-wide) -->
<PropertyGroup>
    <RestorePackagesWithLockFile>true</RestorePackagesWithLockFile>
    <RestoreLockedMode>true</RestoreLockedMode>
</PropertyGroup>
```

---

## Sources

- [Supply-Chain Guardrails for npm, pnpm, and Yarn](https://www.coinspect.com/blog/supply-chain-guardrails/) → `docs/coinspect-supply-chain-guardrails-npm-pnpm-yarn.md`
- [npm Lockfiles as Security Blindspot (Snyk)](https://snyk.io/blog/why-npm-lockfiles-can-be-a-security-blindspot-for-injecting-malicious-modules/) → `docs/snyk-lockfile-security-blindspot.md`
- [Lockfile Poisoning Attack Vector (SafeDep)](https://safedep.substack.com/p/lockfile-poisoning-an-attack-vector) → `docs/safedep-lockfile-poisoning-attack.md`
- [Python Supply Chain Security: Defense in Depth](https://bernat.tech/posts/securing-python-supply-chain/) → `docs/bernat-python-supply-chain-defense-in-depth.md`
- [Design Space of Lockfiles Across Package Managers (arXiv)](https://arxiv.org/html/2505.04834v2) → `docs/arxiv-lockfile-design-space-across-package-managers.md`
- [go.sum Is Not a Lockfile (Filippo Valsorda)](https://words.filippo.io/gosum/) → `docs/filippo-go-sum-not-lockfile.md`
- [Gradle Dependency Locking Docs](https://docs.gradle.org/current/userguide/dependency_locking.html) → `docs/gradle-dependency-locking-docs.md`
- [Maven Lockfile Plugin](https://github.com/chains-project/maven-lockfile) → `docs/maven-lockfile-plugin.md`
- [NuGet Lock File Wiki](https://github.com/NuGet/Home/wiki/Enable-repeatable-package-restore-using-lock-file) → `docs/nuget-lock-file-repeatable-restore.md`
- [Bazel Lockfile Docs](https://bazel.build/external/lockfile) → `docs/bazel-lockfile-docs.md`
- [npm CI/CD Locked Dependencies](https://charlesjones.dev/blog/npm-supply-chain-attacks-ci-cd-locked-dependencies) → `docs/charlesjones-npm-ci-cd-locked-dependencies.md`
- [Yarn Security Features / Hardened Mode](https://yarnpkg.com/features/security) → `docs/yarn-security-features-hardened-mode.md`
- [PEP 751 pylock.toml](https://peps.python.org/pep-0751/) → `docs/pep-751-pylock-toml-format.md`
- [Reproducible Docker Images with Locked Dependencies](https://oneuptime.com/blog/post/2026-02-08-how-to-build-reproducible-docker-images-with-locked-dependencies/view) → `docs/reproducible-docker-images-locked-dependencies.md`
- [Bundler v2.6: Lockfile Checksums](https://bundler.io/blog/2024/12/19/bundler-v2-6.html)
- [lockfile-lint (Liran Tal)](https://github.com/lirantal/lockfile-lint)
- [npm package-lock.json docs](https://docs.npmjs.com/cli/v9/configuring-npm/package-lock-json/)
- [Cargo FAQ](https://doc.rust-lang.org/cargo/faq.html)
- [Go Modules Reference](https://go.dev/ref/mod)
- [Go Checksum Database Proposal](https://go.googlesource.com/proposal/+/master/design/25530-sumdb.md)
