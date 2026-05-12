# Phase 2: Ecosystem Modules — Tier 1

## Goal

Implement the 8 highest-priority ecosystem modules that a software engineering consulting firm encounters on the majority of engagements. Each module implements the `EcosystemModule` interface from Phase 1, providing detection, devenv.nix templates, security hardening configs, pre-commit hooks, deny rules, and CI commands. At the end of this phase, `gdev init` can generate security-hardened environments for the core ecosystems.

## Dependencies

Phase 1 complete (module interface, detection engine, template engine, generation pipeline).

## Phase Outputs

- 8 ecosystem modules implementing `EcosystemModule`: JavaScript/TypeScript, Python, Go, Rust, Java/Kotlin, C#/.NET, Docker, Terraform
- Per-ecosystem security hardening config files (`.npmrc`, `pip.conf`, `pnpm-workspace.yaml`, `.yarnrc.yml`, `bunfig.toml`, `settings.xml`, `gradle.properties`, `nuget.config`, `Directory.Build.props`, `.cargo/config.toml`, `.hadolint.yaml`, `.terraformrc`)
- devenv.nix template fragments for each ecosystem
- Pre-commit hook definitions for each ecosystem
- Claude Code deny rules for each ecosystem's package managers
- CI frozen-install and audit commands for each ecosystem

---

### Unit 2.1: JavaScript/TypeScript Module

**Description:** The most complex module — covers npm, pnpm, yarn, and bun with 4 separate security config generators, plus devenv.nix support for both `languages.javascript` and `languages.typescript`.

**Context:** JS/TS is the most threatened ecosystem (highest attack surface per supply-chain-security spike). Age-gating alone catches 92% of PyPI malware — npm has similar exposure. Must support 4 package managers with distinct config formats. pnpm v10+ blocks install scripts by default (strongest native defense after Composer 2.9).

**Desired Outcome:** Module detects JS/TS projects (package.json, lockfiles), determines package manager, and generates all security configs + devenv.nix fragment.

**Steps:**
1. Implement `Detect`: check for `package.json`, determine package manager from lockfiles (`package-lock.json`→npm, `pnpm-lock.yaml`→pnpm, `yarn.lock`→yarn, `bun.lock`→bun), parse version from `.nvmrc`/`package.json engines`.
2. Implement `DevenvNixFragment`: `languages.javascript.enable = true` or `languages.typescript.enable = true`, with version, `languages.javascript.npm.enable/pnpm.enable/yarn.enable/bun.enable` as appropriate.
3. Implement `SecurityConfigs` — 4 generators:
   - **npm**: `.npmrc` with `save-exact=true`, `ignore-scripts=true`, `min-release-age=3`
   - **pnpm**: additions to `pnpm-workspace.yaml` with `strictDepBuilds: true`, `minimumReleaseAge: 4320`, `trustPolicy: no-downgrade`, `blockExoticSubdeps: true`
   - **yarn**: `.yarnrc.yml` with `enableImmutableInstalls: true`, `enableHardenedMode: true`, `enableScripts: false`, `npmMinimalAgeGate: 7d`
   - **bun**: `bunfig.toml` with `minimumReleaseAge = "7d"`
4. Implement `PreCommitHooks`: `prettier` (formatter), `eslint` (linter).
5. Implement `DenyRules`: patterns for `npm install`, `npm i`, `npm add`, `npx`, `yarn add`, `pnpm add`, `pnpm i`, `bun add`, `bun install`, plus pipe-to-shell patterns.
6. Implement `CICommands`: `npm ci --ignore-scripts`, `pnpm install --frozen-lockfile`, `yarn install --immutable`, `bun install --frozen-lockfile`.
7. Include version requirement comments in each config file.

**Acceptance Criteria:**
- [ ] Detects npm/pnpm/yarn/bun from lockfiles
- [ ] Generates correct `.npmrc` for npm with age-gating + script blocking
- [ ] Generates correct `pnpm-workspace.yaml` additions (minutes format, not milliseconds)
- [ ] Generates correct `.yarnrc.yml` with hardened mode
- [ ] Generates correct `bunfig.toml` with age-gating
- [ ] devenv.nix fragment renders valid Nix with correct package manager enabled
- [ ] Deny rules cover all 4 package managers
- [ ] Each config includes inline comments explaining security purpose

**Research Citations:**
- `artifacts/language-ecosystem-coverage.md § JavaScript/TypeScript` — all config contents
- `research-spikes/package-supply-chain-security/quarantine-gates-research.md` — age-gating specifics
- `research-spikes/package-supply-chain-security/install-sandboxing-research.md` — script blocking
- Validation: pnpm uses minutes in workspace.yaml, not milliseconds in .npmrc

**Status:** Not Started

---

### Unit 2.2: Python Module

**Description:** Covers pip, uv, and poetry with security configs for age-gating, binary-only installs, and hash verification.

**Context:** Python is the second-most-targeted ecosystem. `pip install` runs arbitrary setup.py code by default. Key defenses: `--only-binary :all:` (blocks source distributions that execute setup.py), `--require-hashes` (integrity verification), uv's `--exclude-newer` (age-gating). poetry and uv have separate lockfile formats.

**Desired Outcome:** Module detects Python projects and generates security configs for the detected package manager.

**Steps:**
1. Implement `Detect`: check for `pyproject.toml`, `requirements.txt`, `setup.py`, `Pipfile`. Determine package manager from lockfiles (`poetry.lock`→poetry, `uv.lock`→uv, else pip). Parse version from `.python-version` or `pyproject.toml`.
2. Implement `DevenvNixFragment`: `languages.python.enable = true`, version, `languages.python.uv.enable`/`languages.python.poetry.enable` (mutually exclusive), `languages.python.venv.enable = true`.
3. Implement `SecurityConfigs`:
   - **pip**: `pip.conf` with `require-hashes = true`, `only-binary = :all:` (dev relaxation via devenv env var override)
   - **uv**: CI commands reference with `uv --exclude-newer=7d`, `uv sync --frozen`
   - **poetry**: `poetry.lock` commit enforcement, `poetry install --no-root` in CI
4. Implement `PreCommitHooks`: `ruff` (linter+formatter), `mypy` (type checker), `bandit` (security scanner).
5. Implement `DenyRules`: patterns for `pip install`, `pip3 install`, `python -m pip install`, `python3 -m pip install`, `uv pip install`, `uv add`, `poetry add`.
6. Implement `CICommands`: `pip install --require-hashes --only-binary :all:`, `uv sync --frozen`, `poetry install --no-root`.

**Acceptance Criteria:**
- [ ] Detects pip/uv/poetry from lockfiles
- [ ] Generates `pip.conf` with hash requirement and binary-only
- [ ] poetry and uv mutually exclusive in devenv.nix
- [ ] Deny rules cover all Python package managers
- [ ] Pre-commit hooks include ruff and bandit

**Research Citations:**
- `artifacts/language-ecosystem-coverage.md § Python` — configs
- `research-spikes/package-supply-chain-security/install-sandboxing-research.md § Python` — setup.py risks

**Status:** Not Started

---

### Unit 2.3: Go Module

**Description:** Go has the strongest security posture by design — no install hooks, checksum transparency via sum.golang.org, MVS. Config is minimal but important.

**Steps:**
1. Implement `Detect`: check for `go.mod`, parse Go version.
2. Implement `DevenvNixFragment`: `languages.go.enable = true`, version, `languages.go.delve.enable = true`.
3. Implement `SecurityConfigs`: Set `GOFLAGS=-mod=readonly` in devenv.nix env vars, `GONOSUMCHECK` empty (ensure sumdb is active), `GONOSUMDB` empty.
4. Implement `PreCommitHooks`: `gofmt`, `govet`, `staticcheck`, `govulncheck`.
5. Implement `DenyRules`: `go get *`, `go install *`.
6. Implement `CICommands`: `go mod verify`, `govulncheck ./...`.

**Acceptance Criteria:**
- [ ] Detects Go from go.mod with version
- [ ] `GOFLAGS=-mod=readonly` in devenv.nix env vars
- [ ] govulncheck in pre-commit hooks
- [ ] Deny rules cover `go get` and `go install`

**Research Citations:**
- `artifacts/language-ecosystem-coverage.md § Go`
- `research-spikes/package-supply-chain-security/attack-surface-landscape-research.md § Go` — most secure ecosystem

**Status:** Not Started

---

### Unit 2.4: Rust Module

**Description:** Cargo with locked builds, cargo-audit, sccache integration.

**Steps:**
1. Implement `Detect`: check for `Cargo.toml`, parse toolchain from `rust-toolchain.toml`.
2. Implement `DevenvNixFragment`: `languages.rust.enable = true`, channel (stable/nightly), `languages.rust.components = ["rustfmt" "clippy"]`.
3. Implement `SecurityConfigs`: `.cargo/config.toml` with `net.git-fetch-with-cli = true`, `build.rustc-wrapper = "sccache"` (when build cache configured).
4. Implement `PreCommitHooks`: `rustfmt`, `clippy`.
5. Implement `DenyRules`: `cargo add *`, `cargo install *`.
6. Implement `CICommands`: `cargo build --locked`, `cargo audit`.

**Acceptance Criteria:**
- [ ] Detects Rust from Cargo.toml with toolchain
- [ ] sccache integration when build cache profile configured
- [ ] cargo-audit in CI commands
- [ ] `--locked` flag in CI build commands

**Research Citations:**
- `artifacts/language-ecosystem-coverage.md § Rust`
- `artifacts/artifact-stores-caches-research.md § sccache` — Rust compilation caching

**Status:** Not Started

---

### Unit 2.5: Java/Kotlin Module (JVM)

**Description:** The most complex enterprise ecosystem — must generate both Maven `settings.xml` and Gradle configs since projects use either or both. Includes signature verification, checksum enforcement, and snapshot blocking.

**Context:** JVM needs dual config generators. Maven uses `settings.xml` with `checksumPolicy=fail` and snapshot blocking. Gradle uses `verification-metadata.xml` with SHA256+PGP, `dependencyLocking.lockMode=STRICT`, and repository content filtering. Kotlin uses the same build tools.

**Steps:**
1. Implement `Detect`: check for `pom.xml` (Maven), `build.gradle`/`build.gradle.kts` (Gradle), `settings.gradle.kts`. Parse Java version from `.java-version`, `pom.xml`, or gradle toolchain config.
2. Implement `DevenvNixFragment`: `languages.java.enable = true` (or `languages.kotlin.enable = true`), JDK version, `languages.java.maven.enable = true` or `languages.java.gradle.enable = true`.
3. Implement `SecurityConfigs` — dual generators:
   - **Maven**: `settings.xml` with `checksumPolicy=fail`, snapshot blocking, single-mirror configuration, blocked repositories
   - **Gradle**: `gradle.properties` with `dependencyLocking.lockMode=STRICT`, distribution SHA256 pin. `settings.gradle.kts` snippet with `repositoriesMode.set(FAIL_ON_PROJECT_REPOS)`. Bootstrap command for `verification-metadata.xml`.
4. Implement `PreCommitHooks`: `checkstyle`/`ktlint`, `spotbugs`.
5. Implement `DenyRules`: `mvn install`, `mvn dependency:resolve`, `gradle dependencies`, `./gradlew dependencies`.
6. Implement `CICommands`: `mvn verify -DskipTests`, `mvn io.github.chains-project:maven-lockfile:validate`, `./gradlew build --write-verification-metadata sha256,pgp` (bootstrap), `./gradlew build` (with verification active).

**Acceptance Criteria:**
- [ ] Detects Maven vs Gradle (or both)
- [ ] Maven `settings.xml` has checksumPolicy=fail and snapshot blocking
- [ ] Gradle has strict dependency locking and verification-metadata bootstrap command
- [ ] Kotlin detection maps to same JVM build tools
- [ ] Repository content filtering prevents internal group IDs from resolving on public repos

**Research Citations:**
- `artifacts/language-ecosystem-coverage.md § Java/Kotlin/Gradle` — settings.xml, gradle.properties, verification-metadata
- `artifacts/language-ecosystem-coverage.md § .NET` — Central Package Management pattern (analogous to Gradle centralization)

**Status:** Not Started

---

### Unit 2.6: C#/.NET Module

**Description:** NuGet with signature validation, Central Package Management, locked restore, and built-in vulnerability auditing.

**Context:** .NET has strong native security: `signatureValidationMode=require` in nuget.config, `packages.lock.json` with locked-mode restore, Central Package Management via `Directory.Packages.props`, and built-in `dotnet list package --vulnerable`.

**Steps:**
1. Implement `Detect`: check for `*.csproj`, `*.fsproj`, `*.sln`, `Directory.Build.props`. Parse .NET version from `global.json`.
2. Implement `DevenvNixFragment`: `languages.dotnet.enable = true`, version.
3. Implement `SecurityConfigs`:
   - `nuget.config` with `signatureValidationMode=require`, trusted signers (nuget.org certificate), `audit-level=moderate`, `audit-mode=all`, cleared + explicit package sources
   - `Directory.Build.props` with `RestorePackagesWithLockFile=true`, `RestoreLockedMode` in CI, `ManagePackageVersionsCentrally=true`
4. Implement `PreCommitHooks`: `dotnet-format`.
5. Implement `DenyRules`: `dotnet add package *`, `nuget install *`.
6. Implement `CICommands`: `dotnet restore --locked-mode`, `dotnet list package --vulnerable`.

**Acceptance Criteria:**
- [ ] Detects .NET from csproj/sln/global.json
- [ ] nuget.config has signature validation required
- [ ] Central Package Management enabled
- [ ] Locked restore in CI mode
- [ ] Built-in vulnerability audit in CI commands

**Research Citations:**
- `artifacts/language-ecosystem-coverage.md § .NET (C#/F#)` — nuget.config, Directory.Build.props, Directory.Packages.props

**Status:** Not Started

---

### Unit 2.7: Docker/Containerfiles Module

**Description:** Dockerfile hardening patterns, base image pinning by digest, Hadolint configuration, multi-stage build templates.

**Steps:**
1. Implement `Detect`: check for `Dockerfile`, `Containerfile`, `docker-compose.yml`, `.dockerignore`.
2. Implement `DevenvNixFragment`: add `pkgs.docker`, `pkgs.hadolint`, `pkgs.dive` to packages.
3. Implement `SecurityConfigs`:
   - `.hadolint.yaml` with `trustedRegistries` (docker.io, gcr.io, ghcr.io), `failure-threshold: warning`
   - Dockerfile template/reference with: digest-pinned base images, non-root USER, multi-stage builds, `npm ci --ignore-scripts` in COPY+RUN pattern
4. Implement `PreCommitHooks`: `hadolint` (Dockerfile linter).
5. Implement `DenyRules`: `docker pull *` (route through registry proxy when configured).
6. Implement `CICommands`: `hadolint Dockerfile`, `docker build --no-cache`, `trivy image` (with trivy compromise warning), `cosign verify` for base images.

**Acceptance Criteria:**
- [ ] Detects Docker projects from Dockerfile/Containerfile
- [ ] Hadolint config has trusted registries
- [ ] Dockerfile template demonstrates digest pinning and non-root patterns
- [ ] Trivy compromise warning included in CI docs

**Research Citations:**
- `artifacts/language-ecosystem-coverage.md § Docker/Containerfiles` — Hadolint, digest pinning
- `artifacts/artifact-stores-caches-research.md § Sigstore/cosign` — container signing

**Status:** Not Started

---

### Unit 2.8: Terraform/OpenTofu Module

**Description:** Provider version pinning, lockfile enforcement, network mirror configuration for registry proxy, and IaC security scanning.

**Steps:**
1. Implement `Detect`: check for `*.tf`, `*.tf.json`, `.terraform/`, `.terraform.lock.hcl`. Detect OpenTofu from `tofu` binary or `.opentofu/`.
2. Implement `DevenvNixFragment`: `languages.terraform.enable = true` (or `languages.opentofu.enable = true`), version.
3. Implement `SecurityConfigs`:
   - `.terraformrc` with `disable_checkpoint = true`, network mirror configuration when registry proxy configured, `provider_installation` block
   - `versions.tf` template with `required_version` constraint and exact provider version pins
4. Implement `PreCommitHooks`: `terraform_fmt`, `terraform_validate`, `tflint`, `tfsec`/`trivy config`.
5. Implement `DenyRules`: `terraform init` (ensure lockfile is committed first), `terraform providers` manipulation.
6. Implement `CICommands`: `terraform init -backend=false`, `terraform validate`, `terraform plan`, `tflint`, `tfsec .` or `trivy config .`.

**Acceptance Criteria:**
- [ ] Detects Terraform vs OpenTofu
- [ ] .terraformrc disables checkpoint and configures network mirror when proxy available
- [ ] Provider version pinning in versions.tf template
- [ ] tflint and tfsec/trivy in pre-commit and CI
- [ ] Lockfile `.terraform.lock.hcl` commit enforcement documented

**Research Citations:**
- `artifacts/language-ecosystem-coverage.md § Terraform` — .terraformrc, versions.tf

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All 8 ecosystem modules implement `EcosystemModule` interface
- [ ] All 8 modules register in the module registry
- [ ] Detection works for each ecosystem's marker files
- [ ] devenv.nix fragments render valid Nix for each ecosystem
- [ ] Security configs are valid per-format (YAML/JSON/XML/TOML/INI/HCL)
- [ ] At least 48 deny rules covering all Tier 1 package managers
- [ ] Pre-commit hooks cover formatting + linting + security for each ecosystem
- [ ] CI commands include frozen-install + audit for each ecosystem
- [ ] Each security config has inline comments explaining the security purpose
- [ ] Unit tests pass for all 8 modules
