# Phase 19: Ecosystem Onboarding Validation

## Goal

Validate that `gdev init` correctly detects, configures, and generates a working security-hardened development environment for every supported language ecosystem — both for new (greenfield) projects and existing (brownfield) projects. At the end of this phase, every Tier 1-2 ecosystem has verified new-project and existing-project test cases, with polyglot combinations tested.

## Dependencies

Phase 17 complete (test infrastructure). Phases 1-8 complete (all ecosystem modules, generation pipeline). Phase 6 complete (wizard, answers file). Phase 13 complete (project configuration — `.gdev.yaml` and Join mode for team standards).

## Phase Outputs

- Verified greenfield `gdev init` for all 27 ecosystems
- Verified brownfield `gdev init` against real open-source repos (Tier 1-2)
- Verified polyglot project detection and composition
- Verified generated file content (devenv.nix, security configs, pre-commit hooks, CI workflows)
- Detection accuracy test suite
- Per-ecosystem verification checklists executed

---

### Unit 19.1: Tier 1 Greenfield Validation — New Project Creation

**Description:** For each Tier 1 ecosystem, create a fresh project from scratch, run `gdev init --answers-file`, and verify all generated output matches the ecosystem's verification checklist. Each package manager variant is tested independently.

**Context:** Greenfield validation is the simplest path — no existing configs to merge, no detection ambiguity from co-located ecosystems. This is the baseline: if `gdev init` cannot produce correct output for a clean new project, nothing else works. The research artifact identifies exact project creation commands and verification checklists for every ecosystem, including per-package-manager variants within JS/TS (4 PMs) and Python (3 PMs). Each test uses `--answers-file` to run non-interactively with known-good wizard responses, producing deterministic output that can be asserted against.

**Desired Outcome:** Every Tier 1 ecosystem (8 ecosystems, 13 PM variants) passes greenfield validation with correct devenv.nix packages, security hardening configs, pre-commit hooks, CLAUDE.md sections, and CI workflow steps.

**Steps:**
1. Create testscript scripts for each Tier 1 ecosystem variant. Each script:
   a. Creates a temp directory and runs the ecosystem's project initialization commands.
   b. Writes a pre-built `--answers-file` YAML matching that ecosystem.
   c. Runs `gdev init --answers-file <path>`.
   d. Asserts file existence and content for all generated artifacts.
2. JS/TS — npm variant:
   - `npm init -y` then `gdev init`.
   - Verify `.npmrc` contains `ignore-scripts=true`, `save-exact=true`, `audit=true`, `min-release-age=3`.
   - Verify `devenv.nix` contains `languages.javascript.enable = true`, `languages.javascript.npm.enable = true`, `languages.javascript.package = pkgs.nodejs_22`.
   - Verify pre-commit hooks include `prettier`, `eslint`, `ripsecrets`.
   - Verify CI workflow includes `npm ci`.
3. JS/TS — pnpm variant:
   - `pnpm init` then `gdev init`.
   - Verify `pnpm-workspace.yaml` contains `onlyBuiltDependencies`, `strictDepBuilds: true`, `minimumReleaseAge: 4320`.
   - Verify `devenv.nix` contains `languages.javascript.pnpm.enable = true`.
   - Verify CI workflow includes `pnpm install --frozen-lockfile`.
4. JS/TS — yarn variant:
   - `yarn init -2` then `gdev init`.
   - Verify `.yarnrc.yml` contains `enableImmutableInstalls: true`, `enableHardenedMode: true`, `enableScripts: false`, `npmMinimalAgeGate: 7d`.
   - Verify CI workflow includes `yarn install --immutable`.
5. JS/TS — bun variant:
   - `bun init` then `gdev init`.
   - Verify `bunfig.toml` contains `[install]` with `minimumReleaseAge = "7d"`.
   - Verify `devenv.nix` contains `languages.javascript.bun.enable = true`.
   - Verify CI workflow includes `bun install --frozen-lockfile`.
6. Python — pip variant:
   - Create `requirements.txt` then `gdev init`.
   - Verify `pip.conf` contains `require-hashes = true`, `only-binary = :all:`.
   - Verify `devenv.nix` contains `languages.python.enable = true`, `languages.python.venv.enable = true`.
   - Verify pre-commit hooks include `ruff`, `mypy`, `bandit`, `ripsecrets`.
   - Verify CI workflow includes `pip install --require-hashes -r requirements.txt`.
7. Python — uv variant:
   - `uv init` then `gdev init`.
   - Verify `devenv.nix` contains `languages.python.uv.enable = true`.
   - Verify CI workflow includes `uv sync --locked`.
   - Verify CLAUDE.md documents `--exclude-newer` age-gating.
8. Python — poetry variant:
   - `poetry init -n` then `gdev init`.
   - Verify `devenv.nix` contains `languages.python.poetry.enable = true`.
   - Verify CI workflow includes `poetry install --no-interaction`.
9. Go:
   - `go mod init example.com/test` then `gdev init`.
   - Verify `devenv.nix` contains `languages.go.enable = true`.
   - Verify environment sets `GOFLAGS=-mod=readonly`.
   - Verify pre-commit hooks include `gofmt`, `go vet`, `staticcheck`, `govulncheck`.
   - Verify CI workflow includes `go mod verify` and `govulncheck ./...`.
10. Rust:
    - `cargo init` then `gdev init`.
    - Verify `devenv.nix` contains `languages.rust.enable = true` with `channel = "stable"`.
    - Verify `.cargo/config.toml` contains `[net] git-fetch-with-cli = true`.
    - Verify pre-commit hooks include `rustfmt`, `clippy`, `cargo audit`.
    - Verify CI workflow includes `cargo build --locked` and `cargo audit`.
11. Java (Maven):
    - `mvn archetype:generate` (non-interactive) then `gdev init`.
    - Verify `devenv.nix` contains `languages.java.enable = true`, `languages.java.jdk.package = pkgs.jdk21`, `languages.java.maven.enable = true`.
    - Verify `settings.xml` contains checksum policy `fail` and snapshots disabled.
    - Verify CI workflow includes `mvn verify --strict-checksums`.
12. Kotlin (Gradle):
    - `gradle init --type kotlin-application --dsl kotlin` then `gdev init`.
    - Verify `devenv.nix` contains `languages.kotlin.enable = true` and `languages.java.jdk.package = pkgs.jdk21`.
    - Verify `gradle.properties` contains `dependencyLocking.lockMode=STRICT` and distribution SHA256.
    - Verify pre-commit hooks include `ktlint`, `detekt`.
13. .NET:
    - `dotnet new console` then `gdev init`.
    - Verify `devenv.nix` contains `languages.dotnet.enable = true`.
    - Verify `nuget.config` contains `signatureValidationMode = require`, HTTPS sources, audit settings.
    - Verify `Directory.Build.props` contains `RestorePackagesWithLockFile = true`.
    - Verify CI workflow includes `dotnet restore --locked-mode` and `dotnet list package --vulnerable`.
14. Docker:
    - Create a `Dockerfile` then `gdev init`.
    - Verify `devenv.nix` adds `pkgs.hadolint` to packages.
    - Verify `.hadolint.yaml` generated with trusted registries and failure threshold.
    - Verify pre-commit hooks include `hadolint`.
    - Verify CLAUDE.md documents digest-pinning best practices.
15. Terraform:
    - Create `main.tf` with provider block then `gdev init`.
    - Verify `devenv.nix` contains `languages.terraform.enable = true`.
    - Verify pre-commit hooks include `terraform fmt`, `terraform validate`, `tflint`, `checkov`.
    - Verify CI workflow includes `terraform init -lockfile=readonly`.
    - Verify `.claude/settings.json` deny rules block `terraform apply` without plan.
16. For every ecosystem variant: verify `devenv shell --command "echo ok"` succeeds (generated Nix is valid).

**Acceptance Criteria:**
- [ ] All 4 JS/TS PM variants pass greenfield validation with correct security configs
- [ ] All 3 Python PM variants pass greenfield validation with correct security configs
- [ ] Go, Rust, Java/Maven, Kotlin/Gradle, .NET, Docker, Terraform each pass greenfield validation
- [ ] Every generated `devenv.nix` parses successfully (`devenv shell --command "echo ok"`)
- [ ] Security hardening configs match ecosystem research specifications
- [ ] Pre-commit hooks are appropriate per ecosystem
- [ ] CI workflow steps are correct per ecosystem (frozen installs, vuln scanning)
- [ ] CLAUDE.md sections generated for each ecosystem
- [ ] All tests run via testscript with `--answers-file` (fully non-interactive)

**Research Citations:**
- `artifacts/language-ecosystem-test-targets-research.md § Tier 1` — per-ecosystem creation commands, detection signals, verification checklists
- `artifacts/language-ecosystem-coverage.md` — devenv.nix module mappings, security config specifications
- `research-spikes/package-supply-chain-security/quarantine-gates-research.md` — age-gating config values for npm, pnpm, yarn, bun, pip

**Status:** Not Started

---

### Unit 19.2: Tier 1 Brownfield Validation — Existing Project Onboarding

**Description:** Clone real open-source repos and run `gdev init` to test detection accuracy, existing config handling, and merge behavior against production codebases.

**Context:** Brownfield validation is harder than greenfield because real projects have existing configs that must not be clobbered, CI workflows that must be preserved, and detection signals that may be ambiguous (e.g., both `package-lock.json` and `pnpm-lock.yaml` in a migrating project). The research artifact identifies 14 real open-source repos (all MIT/BSD/Apache licensed) selected for clean structure, active maintenance, committed lockfiles, and existing CI. Each tests a different ecosystem's detection path and merge behavior. These tests run on nightly CI only — repos are large and cloning is slow.

**Desired Outcome:** All 14 Tier 1 brownfield repos onboard without errors. Detection signals are correct, existing configs are preserved, merge behavior works (section markers in CLAUDE.md, three-way merge for settings.json), and generated devenv.nix produces a valid Nix environment for each repo.

**Steps:**
1. Create a nightly CI job that clones each test repo (shallow clone, `--depth=1`) into an isolated temp directory.
2. npm brownfield — `expressjs/express`:
   - Clone, run `gdev init --yes`.
   - Verify npm detected (not pnpm/yarn/bun) from `package-lock.json`.
   - Verify existing `.github/workflows/` preserved (not overwritten).
   - Verify `.npmrc` generated without clobbering any existing npm config.
   - Verify devenv.nix contains correct JS/TS module with npm.
3. pnpm brownfield — `honojs/hono`:
   - Clone, run `gdev init --yes`.
   - Verify pnpm detected from `pnpm-lock.yaml` and `pnpm-workspace.yaml`.
   - Verify existing `pnpm-workspace.yaml` merged (security additions alongside existing workspace config).
   - Verify workspace-level detection (monorepo root, not per-package).
4. yarn brownfield — `yarnpkg/berry`:
   - Clone, run `gdev init --yes`.
   - Verify yarn detected from `yarn.lock` and `.yarnrc.yml`.
   - Verify existing `.yarnrc.yml` merged (security additions alongside existing yarn config).
5. bun brownfield — `elysiajs/elysia`:
   - Clone, run `gdev init --yes`.
   - Verify bun detected from `bun.lock`.
   - Verify `bunfig.toml` generated with age-gating.
6. Python/pip brownfield — `pallets/flask`:
   - Clone, run `gdev init --yes`.
   - Verify pip detected (not uv/poetry) from `requirements.txt` / `pyproject.toml` without `[tool.uv]` or `[tool.poetry]`.
   - Verify existing CI workflows preserved.
7. Python/uv brownfield — `owenlamont/uv-secure`:
   - Clone, run `gdev init --yes`.
   - Verify uv detected from `uv.lock` and `[tool.uv]` in `pyproject.toml`.
8. Python/poetry brownfield — `python-poetry/poetry`:
   - Clone, run `gdev init --yes`.
   - Verify poetry detected from `poetry.lock` and `[tool.poetry]` in `pyproject.toml`.
9. Go brownfield — `axllent/mailpit`:
   - Clone, run `gdev init --yes`.
   - Verify Go detected from `go.mod`.
   - Verify existing CI workflows preserved.
10. Rust brownfield — `bensadeh/tailspin`:
    - Clone, run `gdev init --yes`.
    - Verify Rust detected from `Cargo.toml` and `Cargo.lock`.
    - Verify `.cargo/config.toml` merged if existing.
11. Java/Maven brownfield — `karatelabs/karate`:
    - Clone, run `gdev init --yes`.
    - Verify Maven detected from `pom.xml`.
    - Verify `settings.xml` generated without conflicting with existing Maven wrapper.
12. Kotlin/Gradle brownfield — `InsertKoinIO/koin`:
    - Clone, run `gdev init --yes`.
    - Verify Gradle detected from `build.gradle.kts` and Kotlin from `*.kt` files.
    - Verify existing `gradle.properties` merged.
13. .NET brownfield — `litedb-org/LiteDB`:
    - Clone, run `gdev init --yes`.
    - Verify .NET detected from `*.csproj` and `*.sln`.
    - Verify existing `nuget.config` merged if present.
14. Terraform brownfield — `poseidon/typhoon`:
    - Clone, run `gdev init --yes`.
    - Verify Terraform detected from `*.tf` files and `.terraform.lock.hcl`.
15. Docker brownfield — `linuxserver/Heimdall`:
    - Clone, run `gdev init --yes`.
    - Verify Docker detected from `Dockerfile`.
    - Verify PHP also detected (Heimdall is a PHP app) — polyglot validation.
16. For every repo: run `git diff` post-init and verify no existing user files were modified outside of gdev-managed sections. Verify CLAUDE.md uses section markers. Verify devenv.nix parses.
17. Test Join mode (existing `.gdev.yaml`): create a brownfield fixture with a pre-existing `.gdev.yaml` team configuration file. Run `gdev init --yes`. Verify that Join mode is detected (existing `.gdev.yaml` drives configuration instead of the wizard), team standards from `.gdev.yaml` are applied, and the onboarding experience respects the committed project configuration.

**Acceptance Criteria:**
- [ ] All 14 brownfield repos onboard without errors or crashes
- [ ] Join mode correctly detected and applied when `.gdev.yaml` is present
- [ ] Detection correctly identifies the primary package manager for each repo
- [ ] Existing config files are not clobbered (diff shows only additions in gdev-managed sections)
- [ ] CLAUDE.md uses section markers (`<!-- gdev:* -->` / `<!-- /gdev:* -->`)
- [ ] Existing CI workflows are preserved (gdev generates separate workflow files or merges safely)
- [ ] Generated `devenv.nix` is valid Nix for every repo
- [ ] Merge behavior preserves existing user content in shared files (settings.json, .yarnrc.yml, pnpm-workspace.yaml, nuget.config)
- [ ] Tests run on nightly schedule (not every PR)
- [ ] Each test captures and reports detection signals found vs expected

**Research Citations:**
- `artifacts/language-ecosystem-test-targets-research.md § Tier 1` — repo selections, detection signals, verification checklists
- `research-spikes/gdev-extension-design/migration-strategy-design.md § Section Markers` — CLAUDE.md merge strategy
- `research-spikes/gdev-extension-design/migration-strategy-design.md § Three-Way Merge` — settings.json merge behavior

**Status:** Not Started

---

### Unit 19.3: Tier 2-4 Greenfield Validation

**Description:** Create new projects and verify `gdev init` for all Tier 2-4 ecosystems. Tier 2 gets full security config testing. Tier 3-4 verify detection and basic devenv.nix generation only.

**Context:** Tiers 2-4 represent 19 additional ecosystems beyond Tier 1. Tier 2 ecosystems (PHP, Ruby, Scala, Helm, Ansible, Bash/Shell, C/C++) are commonly encountered on client engagements and need full security config validation — pre-commit hooks, CI workflows, and package manager hardening. Tier 3 ecosystems (Elixir, Dart/Flutter, Swift, Haskell, Clojure, Bazel, Nix) are specialized and need detection + devenv.nix validation. Tier 4 ecosystems (Perl, R, Lua, Zig, PowerShell) are rare and need detection + reference docs only. This tiered validation approach matches the tiered implementation in Phase 7.

**Desired Outcome:** All 7 Tier 2 ecosystems pass full greenfield validation with security configs. All 7 Tier 3 ecosystems pass detection and devenv.nix validation. All 5 Tier 4 ecosystems pass detection and basic generation validation.

**Steps:**
1. Tier 2 — PHP (Composer):
   - `composer init --no-interaction` then `gdev init`.
   - Verify `devenv.nix` contains `languages.php.enable = true`.
   - Verify `composer.json` config section contains `secure-http: true`, `audit.block-insecure: true`.
   - Verify CI workflow includes `composer install --no-scripts --no-interaction` and `composer audit`.
   - Verify pre-commit hooks include `php-cs-fixer`, `phpstan`, `ripsecrets`.
2. Tier 2 — Ruby (Bundler):
   - `bundle init` then `gdev init`.
   - Verify `devenv.nix` contains `languages.ruby.enable = true`, `languages.ruby.bundler.enable = true`.
   - Verify `.bundle/config` contains `BUNDLE_FROZEN: "true"`.
   - Verify CI workflow includes `bundle install --frozen` and `bundle audit check --update`.
   - Verify pre-commit hooks include `rubocop`, `bundler-audit`, `ripsecrets`.
3. Tier 2 — Scala (sbt):
   - `sbt new scala/scala3.g8` then `gdev init`.
   - Verify `devenv.nix` contains `languages.scala.enable = true`, `languages.scala.sbt.enable = true`.
   - Verify `project/plugins.sbt` additions for dependency lock and check.
   - Verify pre-commit hooks include `scalafmt`, `scalafix`.
4. Tier 2 — Helm:
   - `helm create test-helm` then `gdev init`.
   - Verify `devenv.nix` contains `languages.helm.enable = true`.
   - Verify pre-commit hooks include `helm lint`, `checkov`, `kubeconform`.
   - Verify CI workflow includes `helm dependency build` and `helm template | kubeconform`.
5. Tier 2 — Ansible (Galaxy):
   - Create `playbook.yml` and `requirements.yml` then `gdev init`.
   - Verify `devenv.nix` contains `languages.ansible.enable = true`.
   - Verify `ansible.cfg` contains GPG signature verification settings.
   - Verify pre-commit hooks include `ansible-lint`, `yamllint`, `ripsecrets`.
6. Tier 2 — Bash/Shell:
   - Create `script.sh` with bash shebang then `gdev init`.
   - Verify `devenv.nix` contains `languages.shell.enable = true` and adds `shellcheck`, `shfmt`.
   - Verify pre-commit hooks include `shellcheck` (with `--severity=warning`), `shfmt`, `ripsecrets`.
   - Verify no package manager configs generated (shell has no PM).
7. Tier 2 — C/C++ (CMake):
   - Create `CMakeLists.txt` and `src/main.cpp` then `gdev init`.
   - Verify `devenv.nix` contains `languages.cplusplus.enable = true`.
   - Verify packages include `cmake`, `ninja`.
   - Verify pre-commit hooks include `clang-format`, `clang-tidy`, `ripsecrets`.
8. Tier 3 — Elixir (Mix):
   - `mix new test_elixir` then `gdev init`.
   - Verify `devenv.nix` contains `languages.elixir.enable = true`.
   - Verify pre-commit hooks include `mix format`, `credo`.
9. Tier 3 — Dart/Flutter (pub):
   - `dart create test_dart` then `gdev init`.
   - Verify `devenv.nix` contains `languages.dart.enable = true`.
   - Verify pre-commit hooks include `dart format`, `dart analyze`.
10. Tier 3 — Swift (SPM):
    - `swift package init --type executable` then `gdev init`.
    - Verify `devenv.nix` contains `languages.swift.enable = true`.
    - Verify pre-commit hooks include `swiftformat`, `swiftlint`.
11. Tier 3 — Haskell (Cabal):
    - `cabal init --minimal --non-interactive` then `gdev init`.
    - Verify `devenv.nix` contains `languages.haskell.enable = true`.
    - Verify pre-commit hooks include `ormolu` or `fourmolu`, `hlint`.
12. Tier 3 — Clojure (deps.edn):
    - Create `deps.edn` then `gdev init`.
    - Verify `devenv.nix` contains `languages.clojure.enable = true`.
    - Verify pre-commit hooks include `cljfmt`, `clj-kondo`.
13. Tier 3 — Bazel (bzlmod):
    - Create `MODULE.bazel` and `BUILD.bazel` then `gdev init`.
    - Verify `devenv.nix` adds `pkgs.bazel_7` to packages.
    - Verify pre-commit hooks include `buildifier`.
14. Tier 3 — Nix (flake):
    - `nix flake init` then `gdev init`.
    - Verify `devenv.nix` contains `languages.nix.enable = true`.
    - Verify existing `flake.nix` NOT overwritten.
    - Verify pre-commit hooks include `statix`, `nixfmt`, `deadnix`, `flake-checker`.
15. Tier 4 — Perl (Carton):
    - Create `cpanfile` then `gdev init`.
    - Verify `devenv.nix` contains `languages.perl.enable = true`.
    - Verify detection succeeds and basic generation works.
16. Tier 4 — R (renv):
    - Create `renv.lock` and `DESCRIPTION` then `gdev init`.
    - Verify `devenv.nix` contains `languages.r.enable = true`.
17. Tier 4 — Lua (LuaRocks):
    - Create `*.rockspec` then `gdev init`.
    - Verify `devenv.nix` contains `languages.lua.enable = true`.
18. Tier 4 — Zig:
    - `zig init` then `gdev init`.
    - Verify `devenv.nix` contains `languages.zig.enable = true`.
19. Tier 4 — PowerShell (PSGallery):
    - Create `*.psd1` and `*.psm1` then `gdev init`.
    - Verify `devenv.nix` adds `pkgs.powershell` to packages.
20. For every Tier 2 ecosystem: verify `devenv shell --command "echo ok"` succeeds. For Tier 3-4: verify devenv.nix parses without errors.

**Acceptance Criteria:**
- [ ] All 7 Tier 2 ecosystems pass greenfield validation with full security configs (pre-commit, CI, PM hardening)
- [ ] All 7 Tier 3 ecosystems pass detection and devenv.nix generation validation
- [ ] All 5 Tier 4 ecosystems pass detection and basic generation validation
- [ ] Tier 2 pre-commit hooks are appropriate per ecosystem
- [ ] Tier 2 CI workflow steps are correct per ecosystem
- [ ] Every generated `devenv.nix` parses without Nix evaluation errors
- [ ] All tests run via testscript with `--answers-file`

**Research Citations:**
- `artifacts/language-ecosystem-test-targets-research.md § Tier 2-4` — per-ecosystem creation commands, detection signals, verification checklists
- `artifacts/language-ecosystem-coverage.md` — devenv.nix module mappings, tier classification
- `phases/07-ecosystem-modules-tiers2-4.md` — Tier 2-4 module implementation

**Status:** Not Started

---

### Unit 19.4: Polyglot Project Validation

**Description:** Test multi-ecosystem detection and composition by running `gdev init` against real polyglot repositories that combine two or more language ecosystems.

**Context:** Real projects rarely use a single ecosystem. A typical web application has a backend language, a TypeScript frontend, Docker for containerization, and possibly Terraform for infrastructure. gdev must detect all ecosystems present, generate a unified `devenv.nix` that composes the correct package sets, produce security configs for each ecosystem, and avoid conflicts between ecosystem modules. The research artifact identifies 5 polyglot repos covering common combinations. This unit also validates detection priority and conflict resolution when multiple lockfiles coexist (e.g., a project migrating from npm to pnpm that has both `package-lock.json` and `pnpm-lock.yaml`).

**Desired Outcome:** All 5 polyglot repos produce correct multi-ecosystem output. Detection correctly identifies every ecosystem present, devenv.nix composes packages from all ecosystems without conflicts, and security configs are generated for each detected ecosystem.

**Steps:**
1. Polyglot combo 1 — Go + TypeScript + Docker: `rcourtman/Pulse`
   - Clone, run `gdev init --yes`.
   - Verify TypeScript detected (check which JS PM from lockfile).
   - Verify Docker detected from Dockerfile and/or compose file.
   - Verify `devenv.nix` contains both `languages.javascript.enable = true` and Docker packages (`pkgs.hadolint`).
   - Verify pre-commit hooks cover both JS/TS and Docker (prettier, eslint, hadolint).
   - Verify single `.envrc` serves all ecosystems.
   - Verify no conflicting security configs between ecosystems.
2. Polyglot combo 2 — Go + Docker + Terraform: `gruntwork-io/terragrunt`
   - Clone, run `gdev init --yes`.
   - Verify Go detected from `go.mod`.
   - Verify Terraform/HCL detected from `*.tf` test fixtures.
   - Verify Docker detected from Dockerfile.
   - Verify `devenv.nix` contains `languages.go.enable = true` and `languages.terraform.enable = true` and Docker packages.
   - Verify CI workflow covers Go tests, Terraform validation, and Docker linting.
3. Polyglot combo 3 — Elixir + TypeScript + Docker: `sequinstream/sequin`
   - Clone, run `gdev init --yes`.
   - Verify Elixir detected from `mix.exs` (Tier 3 ecosystem alongside Tier 1).
   - Verify TypeScript detected from lockfile.
   - Verify Docker detected.
   - Verify `devenv.nix` contains `languages.elixir.enable = true` alongside JS/TS modules.
   - Verify Tier 3 ecosystem (Elixir) composes correctly with Tier 1 ecosystems.
4. Polyglot combo 4 — Java + Kotlin + Docker: `testcontainers/testcontainers-java`
   - Clone, run `gdev init --yes`.
   - Verify Java and Kotlin both detected.
   - Verify Gradle build system detected (not Maven) from `build.gradle.kts`.
   - Verify Docker tooling detected.
   - Verify JDK version set appropriately in devenv.nix (single JDK for both Java and Kotlin).
   - Verify `devenv.nix` does not duplicate JDK entries.
5. Polyglot combo 5 — Rust + Python: `astral-sh/ruff`
   - Clone, run `gdev init --yes`.
   - Verify Rust detected from `Cargo.toml`.
   - Verify Python detected from `pyproject.toml`.
   - Verify correct Python PM detected (uv if `[tool.uv]` present, else pip).
   - Verify `devenv.nix` contains both `languages.rust.enable = true` and `languages.python.enable = true`.
   - Verify security configs generated for both Rust and Python (cargo audit + Python vuln scanning).
6. Detection priority / conflict resolution tests:
   - Create a synthetic project with both `package-lock.json` and `pnpm-lock.yaml`. Verify pnpm wins per detection priority order. Verify user is warned about conflicting lockfiles.
   - Create a synthetic project with `packageManager` field in `package.json` set to `yarn@4.x`. Verify yarn detected regardless of which lockfile is present.
   - Create a synthetic project with both `requirements.txt` and `uv.lock`. Verify uv wins per detection priority order.
   - Create a synthetic project with both `pom.xml` and `build.gradle.kts`. Verify correct build tool detected (Gradle preferred if both present, or determined from CI workflow).
7. For every polyglot repo: verify `devenv shell --command "echo ok"` succeeds with the combined devenv.nix.

**Acceptance Criteria:**
- [ ] All 5 polyglot repos produce correct multi-ecosystem devenv.nix
- [ ] Every detected ecosystem appears in generated config (no missing ecosystems)
- [ ] Package sets compose correctly (no duplicate entries, no conflicts)
- [ ] Security configs generated for each detected ecosystem independently
- [ ] CI workflow covers all detected ecosystems
- [ ] Pre-commit hooks cover all detected languages
- [ ] Single `.envrc` serves all ecosystems
- [ ] Detection priority resolves conflicting lockfiles correctly (pnpm > yarn > npm for JS, uv > poetry > pip for Python)
- [ ] `packageManager` field in package.json overrides lockfile-based detection
- [ ] User warned when conflicting lockfiles detected
- [ ] Combined `devenv.nix` is valid Nix for every polyglot repo

**Research Citations:**
- `artifacts/language-ecosystem-test-targets-research.md § Polyglot Combo Test Targets` — repo selections, expected ecosystem detection
- `artifacts/language-ecosystem-test-targets-research.md § Detection Priority / Conflict Resolution` — PM priority order, disambiguation rules
- `artifacts/language-ecosystem-coverage.md` — per-ecosystem devenv.nix modules and package sets

**Status:** Not Started

---

### Unit 19.5: Detection Accuracy & Edge Cases

**Description:** Test detection edge cases and accuracy with synthetic fixture directories that trigger each edge case, verifying graceful handling, correct tier-based behavior, and zero false negatives for Tier 1-2 primary detection signals.

**Context:** Real projects are messy. Monorepos have different ecosystems at different directory levels. Projects in migration have conflicting signals. Some projects have manifest files without lockfiles. Others have source files without any manifest. gdev must handle all of these gracefully — detect what it can, warn about ambiguity, and never crash. The detection signal summary table in the research artifact documents primary and secondary signals for all 27 ecosystems; this unit verifies the detection engine against every signal and its absence.

**Desired Outcome:** Zero false negatives for Tier 1-2 primary detection signals. Graceful handling of all edge cases (no crashes, clear warning messages). Tier-appropriate output levels (full security configs for Tier 1, full for Tier 2, detection + devenv.nix for Tier 3, detection + reference docs for Tier 4).

**Steps:**
1. Monorepo detection:
   - Create a fixture with `package.json` at root and `go.mod` in `backend/` subdirectory.
   - Verify gdev detects both ecosystems.
   - Create a pnpm workspace fixture with `pnpm-workspace.yaml` at root and `packages/a/package.json`, `packages/b/package.json`.
   - Verify workspace root detected (not individual packages).
   - Create a fixture with different ecosystems at root vs subdirectory.
   - Verify gdev reports all detected ecosystems with their locations.
2. Partial / incomplete projects:
   - `package.json` without `node_modules/` or any lockfile — verify detection succeeds, lockfile warning emitted.
   - `go.mod` without any `.go` files — verify Go detected (go.mod is authoritative).
   - `Cargo.toml` without `src/` directory — verify Rust detected.
   - `pyproject.toml` without any `[tool.*]` sections and no lockfile — verify Python detected with pip fallback.
   - `*.tf` files without `.terraform.lock.hcl` — verify Terraform detected, lockfile warning emitted.
3. Conflicting signals:
   - `Dockerfile` + `docker-compose.yml` both present — verify Docker detected once (not twice).
   - `Dockerfile` + `Containerfile` both present — verify no duplicate detection.
   - `setup.py` + `pyproject.toml` + `requirements.txt` all present — verify correct PM resolution.
   - `build.gradle.kts` + `pom.xml` in same directory — verify disambiguation works.
4. Multiple Python PMs:
   - `requirements.txt` + `pyproject.toml` (no tool sections) — verify pip detected.
   - `requirements.txt` + `pyproject.toml` with `[tool.poetry]` — verify poetry wins.
   - `requirements.txt` + `uv.lock` — verify uv wins.
   - `poetry.lock` + `uv.lock` — verify uv wins per priority order.
5. Lock file presence/absence:
   - Project with `package.json` but no lockfile — verify detection succeeds with warning "no lockfile found, consider running <pm> install".
   - Project with `go.mod` but no `go.sum` — verify detection succeeds (go.sum created on first build).
   - Project with `Gemfile` but no `Gemfile.lock` — verify detection succeeds with warning.
   - Verify warnings are informational, not errors — `gdev init` completes successfully.
6. Nested projects:
   - Root has `package.json` (TypeScript), `services/api/` has `go.mod` (Go), `infra/` has `main.tf` (Terraform).
   - Verify all three ecosystems detected.
   - Verify devenv.nix includes all three.
   - Create a deeply nested ecosystem (`a/b/c/d/Cargo.toml`) — verify detection depth limit (if any) is documented and reasonable.
7. Hidden and edge-case files:
   - `.go` files (dot-prefixed) — verify not detected as Go (Go convention doesn't use dot-prefixed source).
   - `.py` files without `__init__.py`, `setup.py`, `pyproject.toml`, or `requirements.txt` — verify Python NOT detected from source files alone (requires manifest).
   - `.envrc` file present (shell script) — verify does NOT trigger Bash/Shell ecosystem detection.
   - `Makefile` only — verify does NOT trigger any specific ecosystem (weak signal).
8. Empty project:
   - Completely empty directory — verify `gdev init` handles gracefully.
   - Verify clear message: "No ecosystems detected. Use the wizard to select ecosystems manually."
   - Verify wizard still functions (manual ecosystem selection).
   - Verify `gdev init --yes` on empty project produces minimal valid output (base devenv.nix with no ecosystem modules).
9. Tier-based output verification:
   - Create a Tier 1 project (Go) — verify full security configs, pre-commit hooks, CI workflow, CLAUDE.md section.
   - Create a Tier 2 project (Ruby) — verify full security configs, pre-commit hooks, CI workflow, CLAUDE.md section.
   - Create a Tier 3 project (Elixir) — verify detection, devenv.nix generation, basic pre-commit hooks.
   - Create a Tier 4 project (Zig) — verify detection, devenv.nix generation, reference documentation in CLAUDE.md only.
   - Verify tier boundaries are correct — no Tier 3 ecosystem gets CI workflow generation, no Tier 4 ecosystem gets pre-commit hooks beyond ripsecrets.
10. Already-devenv projects:
    - Create a project with existing `devenv.nix` + `devenv.yaml` — verify gdev does NOT overwrite.
    - Verify gdev offers upgrade/merge mode only.
    - Verify clear message explaining existing devenv detected.

**Acceptance Criteria:**
- [ ] Zero false negatives for all Tier 1 primary detection signals
- [ ] Zero false negatives for all Tier 2 primary detection signals
- [ ] Monorepo detection finds ecosystems in subdirectories
- [ ] Workspace root detection works for pnpm, npm, and yarn workspaces
- [ ] Partial projects (manifest without lockfile) detect successfully with warning
- [ ] Conflicting signals resolved per documented priority order
- [ ] Multiple Python PM coexistence resolved correctly
- [ ] Nested projects detect all ecosystems across directory tree
- [ ] Empty project handled gracefully (no crash, clear message, wizard still works)
- [ ] Tier-based output levels enforced (Tier 1-2 get full configs, Tier 3 get basic, Tier 4 get reference docs)
- [ ] Already-devenv projects are not overwritten
- [ ] No detection edge case causes a crash or panic
- [ ] All edge case tests implemented as testscript scripts with synthetic fixture directories

**Research Citations:**
- `artifacts/language-ecosystem-test-targets-research.md § Detection Signal Summary` — primary/secondary signals for all 27 ecosystems
- `artifacts/language-ecosystem-test-targets-research.md § Detection Priority / Conflict Resolution` — PM disambiguation rules
- `artifacts/language-ecosystem-coverage.md` — tier classification, ecosystem module interface
- `research-spikes/gdev-extension-design/wizard-flow-integration-design.md` — detection engine design, manual override via wizard

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All five units pass acceptance criteria
- [ ] All 8 Tier 1 ecosystems (with PM variants = 13 test cases) pass greenfield validation
- [ ] All 14 Tier 1 brownfield repos onboard without errors
- [ ] All 7 Tier 2 ecosystems pass greenfield validation with full security configs
- [ ] All 7 Tier 3 ecosystems pass detection and devenv.nix validation
- [ ] All 5 Tier 4 ecosystems pass detection and basic generation validation
- [ ] Polyglot composition correct for all 5 combo repos
- [ ] Detection accuracy: zero false negatives for Tier 1-2 primary detection signals
- [ ] Edge cases handled gracefully (no crashes, clear messages)
- [ ] Generated `devenv.nix` valid Nix for every ecosystem combination
- [ ] Security configs correct per ecosystem (hardening configs, pre-commit hooks, CI workflows)
- [ ] Brownfield merge behavior preserves existing user configs
- [ ] Detection priority resolves conflicting lockfiles per documented rules
- [ ] Tier-based output levels enforced across all ecosystems
