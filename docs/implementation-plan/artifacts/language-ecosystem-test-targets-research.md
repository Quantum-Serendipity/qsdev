# Language Ecosystem Test Targets Research

End-to-end validation targets for gdev `qsdev init` across 27 language/platform ecosystems. Each ecosystem has two test types: **Type A** (new project creation / greenfield) and **Type B** (existing project onboarding / brownfield).

**Last updated:** 2026-05-12

---

## Table of Contents

1. [Tier 1 — Must Ship](#tier-1--must-ship)
2. [Tier 2 — Should Ship](#tier-2--should-ship)
3. [Tier 3 — Nice to Have](#tier-3--nice-to-have)
4. [Tier 4 — Reference Docs Only](#tier-4--reference-docs-only)
5. [Polyglot Combo Test Targets](#polyglot-combo-test-targets)

---

## Tier 1 — Must Ship

### 1. JavaScript/TypeScript

JavaScript and TypeScript share detection but diverge on package managers. Test each PM variant independently.

#### 1a. npm

**Type A — New Project Creation:**
```bash
mkdir test-npm && cd test-npm
npm init -y
npm install express
# Creates: package.json, package-lock.json, node_modules/
```
Files created: `package.json`, `package-lock.json`, `node_modules/`

**Type B — Existing Project Onboarding:**
- **Primary:** [expressjs/express](https://github.com/expressjs/express) — MIT, actively maintained, npm-based, has CI (.github/workflows/), package-lock.json committed. ~180 files. The canonical Node.js web framework.
- **Alternative:** [sindresorhus/meow](https://github.com/sindresorhus/meow) — MIT, npm-based CLI helper, ~3.7K stars, small focused project. Good for testing pure npm without framework noise.

**Detection Signals:**
- `package.json` (primary — present in all JS/TS projects)
- `package-lock.json` (npm-specific lockfile)
- `.npmrc` (existing npm config — must not clobber)
- `tsconfig.json` (TypeScript indicator)
- Edge cases: `workspaces` field in package.json (npm workspaces), absence of lockfile (bare init)

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.javascript.enable = true` and `languages.javascript.npm.enable = true`
- [ ] `devenv.nix` sets `languages.javascript.package = pkgs.nodejs_22` (or appropriate LTS)
- [ ] `.npmrc` generated with `ignore-scripts=true`, `save-exact=true`, `audit=true`, `min-release-age=3`
- [ ] Pre-commit hooks include `prettier`, `eslint`, `ripsecrets`
- [ ] `.envrc` generated with `use devenv`
- [ ] CI workflow includes `npm ci` (frozen install)
- [ ] If `tsconfig.json` exists, `languages.typescript.enable = true` added

#### 1b. pnpm

**Type A — New Project Creation:**
```bash
mkdir test-pnpm && cd test-pnpm
pnpm init
pnpm add express
# Creates: package.json, pnpm-lock.yaml, node_modules/
```
Files created: `package.json`, `pnpm-lock.yaml`, `node_modules/`

**Type B — Existing Project Onboarding:**
- **Primary:** [honojs/hono](https://github.com/honojs/hono) — MIT, pnpm-based (pnpm-workspace.yaml), actively maintained (updated May 2026), ~22K stars. Web framework built on Web Standards. Has CI, workspace structure.
- **Alternative:** [drizzle-team/drizzle-orm](https://github.com/drizzle-team/drizzle-orm) — Apache-2.0, pnpm monorepo, ~30K stars. Tests workspace detection.

**Detection Signals:**
- `pnpm-lock.yaml` (pnpm-specific lockfile)
- `pnpm-workspace.yaml` (workspace root)
- `.npmrc` with `shamefully-hoist` or pnpm-specific settings
- Edge cases: workspace packages vs root, `packageManager` field in package.json

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.javascript.pnpm.enable = true`
- [ ] `pnpm-workspace.yaml` security settings generated: `onlyBuiltDependencies`, `strictDepBuilds: true`, `minimumReleaseAge: 4320`
- [ ] CI workflow includes `pnpm install --frozen-lockfile`
- [ ] Existing `pnpm-workspace.yaml` merged, not replaced

#### 1c. yarn (Berry/v4)

**Type A — New Project Creation:**
```bash
mkdir test-yarn && cd test-yarn
yarn init -2
yarn add express
# Creates: package.json, yarn.lock, .yarnrc.yml, .yarn/
```
Files created: `package.json`, `yarn.lock`, `.yarnrc.yml`, `.yarn/`

**Type B — Existing Project Onboarding:**
- **Primary:** [yarnpkg/berry](https://github.com/yarnpkg/berry) — BSD-2-Clause, the Yarn v4 monorepo itself. Tests yarn workspace detection and `.yarnrc.yml` merge behavior.
- **Alternative:** [redwoodjs/redwood](https://github.com/redwoodjs/redwood) — MIT, yarn workspace monorepo, full-stack JS framework. ~17K stars. Good polyglot test (JS + Prisma + GraphQL).

**Detection Signals:**
- `yarn.lock` (yarn-specific lockfile)
- `.yarnrc.yml` (Yarn Berry config — must merge, not replace)
- `.yarn/` directory (PnP cache, releases)
- `packageManager` field in package.json with `yarn@4.x`
- Edge cases: Yarn Classic (v1) vs Berry (v2+) — different config formats

**Verification Checklist:**
- [ ] `.yarnrc.yml` generated/merged with `enableImmutableInstalls: true`, `enableHardenedMode: true`, `enableScripts: false`, `npmMinimalAgeGate: 7d`
- [ ] CI workflow includes `yarn install --immutable`

#### 1d. bun

**Type A — New Project Creation:**
```bash
mkdir test-bun && cd test-bun
bun init
bun add express
# Creates: package.json, bun.lock, node_modules/, tsconfig.json, index.ts
```
Files created: `package.json`, `bun.lock`, `node_modules/`, `tsconfig.json`, `index.ts`

**Type B — Existing Project Onboarding:**
- **Primary:** [elysiajs/elysia](https://github.com/elysiajs/elysia) — MIT, Bun-first web framework, bun.lock present, actively maintained, ~12K stars. Uses Bun native APIs.

**Detection Signals:**
- `bun.lock` (Bun lockfile — text format since Bun 1.2)
- `bun.lockb` (legacy binary lockfile — older projects)
- `bunfig.toml` (Bun config)
- Edge cases: `bun.lock` vs `bun.lockb` format transition; projects that have both `package-lock.json` and `bun.lock`

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.javascript.bun.enable = true`
- [ ] `bunfig.toml` generated with `[install] minimumReleaseAge = "7d"`
- [ ] CI workflow includes `bun install --frozen-lockfile`

---

### 2. Python

#### 2a. pip

**Type A — New Project Creation:**
```bash
mkdir test-pip && cd test-pip
python -m venv .venv
echo "flask==3.1.0" > requirements.txt
pip install -r requirements.txt
# Creates: requirements.txt, .venv/
```
Files created: `requirements.txt`, `.venv/`

**Type B — Existing Project Onboarding:**
- **Primary:** [pallets/flask](https://github.com/pallets/flask) — BSD-3-Clause, uses requirements files for dev dependencies, pyproject.toml for package definition. ~70K stars. Has CI, CONTRIBUTING guide.
- **Alternative:** [psf/requests](https://github.com/psf/requests) — Apache-2.0, uses pyproject.toml + requirements files. ~52K stars. Canonical HTTP library.

**Detection Signals:**
- `requirements.txt` (pip primary)
- `requirements/*.txt` (split requirements — dev, prod, test)
- `setup.py` / `setup.cfg` (legacy packaging)
- `pyproject.toml` without `[tool.poetry]` or `[tool.uv]` sections
- Edge cases: multiple requirements files, constraints files, `-r` includes

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.python.enable = true` and `languages.python.venv.enable = true`
- [ ] `pip.conf` generated with `require-hashes = true`, `only-binary = :all:`
- [ ] Pre-commit hooks include `ruff`, `mypy`, `bandit`, `ripsecrets`
- [ ] CI workflow includes `pip install --require-hashes -r requirements.txt`

#### 2b. uv

**Type A — New Project Creation:**
```bash
uv init test-uv
cd test-uv
uv add flask
# Creates: pyproject.toml, uv.lock, .python-version, hello.py
```
Files created: `pyproject.toml`, `uv.lock`, `.python-version`

**Type B — Existing Project Onboarding:**
- **Primary:** [astral-sh/ruff](https://github.com/astral-sh/ruff) — MIT, uses uv for Python tooling (though Ruff itself is Rust). ~40K stars. Has pyproject.toml, CI workflows.
- **Alternative:** [owenlamont/uv-secure](https://github.com/owenlamont/uv-secure) — MIT, pure Python project using uv with uv.lock, 150 stars. Smaller but cleanly demonstrates uv workflow.

**Detection Signals:**
- `uv.lock` (uv-specific lockfile)
- `pyproject.toml` with `[tool.uv]` section
- `.python-version` file
- Edge cases: uv workspaces (`[tool.uv.workspace]`), projects migrating from pip/poetry

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.python.uv.enable = true`
- [ ] CI workflow includes `uv sync --locked`
- [ ] Age-gating via `uv pip install --exclude-newer` documented in CLAUDE.md

#### 2c. poetry

**Type A — New Project Creation:**
```bash
poetry new test-poetry
cd test-poetry
poetry add flask
# Creates: pyproject.toml, poetry.lock, tests/, README.md
```
Files created: `pyproject.toml` (with `[tool.poetry]`), `poetry.lock`, `tests/`

**Type B — Existing Project Onboarding:**
- **Primary:** [python-poetry/poetry](https://github.com/python-poetry/poetry) — MIT, uses poetry itself (poetry.lock present), ~33K stars. Has CI. The canonical poetry project.
- **Alternative:** [tiangolo/fastapi](https://github.com/tiangolo/fastapi) — MIT, uses poetry for development. ~85K stars. Real-world API framework.

**Detection Signals:**
- `poetry.lock` (poetry-specific lockfile)
- `pyproject.toml` with `[tool.poetry]` section
- Edge cases: poetry v1 vs v2 format differences, `poetry.toml` config

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.python.poetry.enable = true`
- [ ] CI workflow includes `poetry install --no-interaction`

---

### 3. Go

**Type A — New Project Creation:**
```bash
mkdir test-go && cd test-go
go mod init example.com/test
cat > main.go << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("hello")
}
EOF
go mod tidy
# Creates: go.mod, go.sum (if dependencies), main.go
```
Files created: `go.mod`, `main.go` (optionally `go.sum` once deps added)

**Type B — Existing Project Onboarding:**
- **Primary:** [axllent/mailpit](https://github.com/axllent/mailpit) — MIT, Go project with go.sum, CI workflows, ~9.4K stars. Email testing tool. Clean Go module structure, moderate size.
- **Alternative:** [ThreeDotsLabs/watermill](https://github.com/ThreeDotsLabs/watermill) — MIT, event-driven Go library, ~9.7K stars. Has go.sum, CI, good module structure.

**Detection Signals:**
- `go.mod` (primary — defines module path and Go version)
- `go.sum` (checksum database)
- `*.go` files
- Edge cases: Go workspace (`go.work`), multi-module repos, vendor directory (`vendor/`)

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.go.enable = true`
- [ ] Environment sets `GOFLAGS=-mod=readonly`
- [ ] Pre-commit hooks include `gofmt`, `go vet`, `staticcheck`, `govulncheck`
- [ ] CI workflow includes `go mod verify` and `govulncheck ./...`

---

### 4. Rust (Cargo)

**Type A — New Project Creation:**
```bash
cargo init test-rust
cd test-rust
cargo add serde --features derive
# Creates: Cargo.toml, Cargo.lock, src/main.rs
```
Files created: `Cargo.toml`, `Cargo.lock`, `src/main.rs`

**Type B — Existing Project Onboarding:**
- **Primary:** [bensadeh/tailspin](https://github.com/bensadeh/tailspin) — MIT, Rust CLI tool for log highlighting, Cargo.lock present, ~7.8K stars. Clean single-crate structure with CI.
- **Alternative:** [ducaale/xh](https://github.com/ducaale/xh) — MIT, HTTP client (like httpie), ~7.8K stars. Cargo-based with CI workflows and release automation.

**Detection Signals:**
- `Cargo.toml` (primary — package manifest)
- `Cargo.lock` (lockfile — committed for binaries, sometimes for libraries)
- `src/main.rs` or `src/lib.rs`
- `.cargo/config.toml` (existing Cargo config — must merge)
- Edge cases: Cargo workspace (`[workspace]` in root Cargo.toml), multiple crates, `rust-toolchain.toml`

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.rust.enable = true` with `channel = "stable"`
- [ ] `.cargo/config.toml` generated with `[net] git-fetch-with-cli = true`
- [ ] Pre-commit hooks include `rustfmt`, `clippy`, `cargo audit`
- [ ] CI workflow includes `cargo build --locked` and `cargo audit`

---

### 5. Java/Kotlin (Maven, Gradle)

#### 5a. Java — Maven

**Type A — New Project Creation:**
```bash
mvn archetype:generate \
  -DgroupId=com.example \
  -DartifactId=test-maven \
  -DarchetypeArtifactId=maven-archetype-quickstart \
  -DinteractiveMode=false
cd test-maven
# Creates: pom.xml, src/main/java/, src/test/java/
```
Files created: `pom.xml`, `src/main/java/`, `src/test/java/`

**Type B — Existing Project Onboarding:**
- **Primary:** [karatelabs/karate](https://github.com/karatelabs/karate) — MIT, Maven-based Java project, ~8.9K stars. Test automation framework with pom.xml, CI workflows.
- **Alternative:** [graphql-java/graphql-java](https://github.com/graphql-java/graphql-java) — MIT, Gradle-based Java library (tests Gradle path), ~6.2K stars.

**Detection Signals:**
- `pom.xml` (Maven primary)
- `mvnw` / `.mvn/` (Maven wrapper)
- `.mvn/maven.config` (existing Maven config)
- Edge cases: multi-module Maven (parent pom with `<modules>`), Maven vs Gradle coexistence

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.java.enable = true`, `languages.java.jdk.package = pkgs.jdk21`, `languages.java.maven.enable = true`
- [ ] `settings.xml` generated with checksum policy `fail`, snapshots disabled
- [ ] CI workflow includes `mvn verify --strict-checksums`
- [ ] Pre-commit hooks include `google-java-format`, `ripsecrets`

#### 5b. Kotlin — Gradle

**Type A — New Project Creation:**
```bash
mkdir test-kotlin && cd test-kotlin
gradle init --type kotlin-application --dsl kotlin
# Creates: build.gradle.kts, settings.gradle.kts, gradlew, gradle/, src/
```
Files created: `build.gradle.kts`, `settings.gradle.kts`, `gradlew`, `gradle/`, `src/`

**Type B — Existing Project Onboarding:**
- **Primary:** [InsertKoinIO/koin](https://github.com/InsertKoinIO/koin) — Apache-2.0, Kotlin DI framework, Gradle-based with build.gradle.kts, ~10K stars. Has CI workflows.
- **Alternative:** [javalin/javalin](https://github.com/javalin/javalin) — Apache-2.0, Kotlin/Java web framework, Maven-based, ~8.3K stars. Tests Java+Kotlin combo detection.

**Detection Signals:**
- `build.gradle.kts` or `build.gradle` (Gradle primary)
- `settings.gradle.kts` or `settings.gradle`
- `gradlew` / `gradle/` (Gradle wrapper)
- `gradle.properties`
- `gradle.lockfile` (opt-in lockfile)
- `*.kt` files (Kotlin source)
- Edge cases: Kotlin Multiplatform (`KotlinMultiplatformExtension`), Gradle version catalogs (`gradle/libs.versions.toml`), composite builds

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.kotlin.enable = true` and `languages.java.jdk.package = pkgs.jdk21`
- [ ] `gradle.properties` generated with `dependencyLocking.lockMode=STRICT`, distribution SHA256
- [ ] `settings.gradle.kts` security additions: `repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)`
- [ ] CI workflow includes `./gradlew dependencies --write-locks` check
- [ ] Pre-commit hooks include `ktlint`, `detekt`

---

### 6. C#/.NET (NuGet, dotnet)

**Type A — New Project Creation:**
```bash
dotnet new console -n test-dotnet
cd test-dotnet
dotnet add package Newtonsoft.Json
dotnet restore
# Creates: test-dotnet.csproj, Program.cs, obj/
```
Files created: `test-dotnet.csproj`, `Program.cs`, `obj/`

**Type B — Existing Project Onboarding:**
- **Primary:** [litedb-org/LiteDB](https://github.com/litedb-org/LiteDB) — MIT, .NET NoSQL database, ~9.4K stars. Has .sln, .csproj files, NuGet packages, CI workflows.
- **Alternative:** [ThreeMammals/Ocelot](https://github.com/ThreeMammals/Ocelot) — MIT, .NET API Gateway, ~8.7K stars. Multi-project solution, NuGet dependencies, CI.

**Detection Signals:**
- `*.csproj` or `*.fsproj` (project files)
- `*.sln` (solution file)
- `nuget.config` (existing NuGet config — must merge)
- `packages.lock.json` (NuGet lockfile — opt-in)
- `Directory.Build.props` (MSBuild properties — must merge)
- `Directory.Packages.props` (central package management)
- `global.json` (.NET SDK version pinning)
- Edge cases: multi-project solutions, .NET Aspire projects, Paket alternative PM

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.dotnet.enable = true` with SDK package
- [ ] `nuget.config` generated with `signatureValidationMode = require`, HTTPS sources, audit settings
- [ ] `Directory.Build.props` generated with `RestorePackagesWithLockFile = true`, `RestoreLockedMode` for CI
- [ ] CI workflow includes `dotnet restore --locked-mode` and `dotnet list package --vulnerable`
- [ ] Pre-commit hooks include `dotnet format`, `ripsecrets`

---

### 7. Docker/Containerfiles

**Type A — New Project Creation:**
```bash
mkdir test-docker && cd test-docker
cat > Dockerfile << 'EOF'
FROM node:22-alpine
WORKDIR /app
COPY . .
CMD ["node", "index.js"]
EOF
# Creates: Dockerfile
```
Files created: `Dockerfile`

**Type B — Existing Project Onboarding:**
- **Primary:** [linuxserver/Heimdall](https://github.com/linuxserver/Heimdall) — MIT, PHP app with Dockerfile, docker-compose, ~9.2K stars. Real containerized application.
- **Alternative:** [mlocati/docker-php-extension-installer](https://github.com/mlocati/docker-php-extension-installer) — MIT, Docker-focused project, ~4.9K stars. Has Dockerfile, CI.

**Detection Signals:**
- `Dockerfile` (primary)
- `Containerfile` (Podman alternative name)
- `docker-compose.yml` / `docker-compose.yaml` / `compose.yml` / `compose.yaml`
- `.dockerignore`
- Edge cases: multi-stage builds, multiple Dockerfiles (`Dockerfile.dev`, `Dockerfile.prod`), buildx bake files

**Verification Checklist:**
- [ ] `devenv.nix` adds `pkgs.hadolint` to packages
- [ ] `.hadolint.yaml` generated with trusted registries and failure threshold
- [ ] Pre-commit hooks include `hadolint`
- [ ] CLAUDE.md documents digest-pinning best practices
- [ ] CI workflow includes Hadolint and image scanning steps

---

### 8. Terraform/OpenTofu

**Type A — New Project Creation:**
```bash
mkdir test-terraform && cd test-terraform
cat > main.tf << 'EOF'
terraform {
  required_version = ">= 1.9.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}
EOF
terraform init
# Creates: main.tf, .terraform/, .terraform.lock.hcl
```
Files created: `main.tf`, `.terraform/`, `.terraform.lock.hcl`

**Type B — Existing Project Onboarding:**
- **Primary:** [poseidon/typhoon](https://github.com/poseidon/typhoon) — MIT, Terraform-based Kubernetes distribution, ~2K stars. Has .terraform.lock.hcl, versions.tf, CI workflows. Clean HCL module structure.
- **Alternative:** [terraform-aws-modules/terraform-aws-vpc](https://github.com/terraform-aws-modules/terraform-aws-vpc) — Apache-2.0, canonical Terraform module, ~3K stars. Tests module-style (not root-module) Terraform.

**Detection Signals:**
- `*.tf` files (primary)
- `.terraform.lock.hcl` (provider lockfile)
- `.terraformrc` / `terraform.rc` (CLI config)
- `versions.tf` (version constraints convention)
- `backend.tf` (state backend config)
- Edge cases: Terragrunt wrapper (`terragrunt.hcl`), OpenTofu vs Terraform, workspace-based layouts, module vs root distinction

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.terraform.enable = true` (or `languages.opentofu`)
- [ ] Pre-commit hooks from `pre-commit-terraform`: `terraform fmt`, `terraform validate`, `tflint`, `tfsec`/`checkov`
- [ ] CI workflow includes `terraform init -lockfile=readonly`
- [ ] `.claude/settings.json` deny rules block `terraform apply` without plan

---

## Tier 2 — Should Ship

### 9. PHP (Composer)

**Type A — New Project Creation:**
```bash
mkdir test-php && cd test-php
composer init --no-interaction --name=test/project --type=project
composer require guzzlehttp/guzzle
# Creates: composer.json, composer.lock, vendor/
```
Files created: `composer.json`, `composer.lock`, `vendor/`

**Type B — Existing Project Onboarding:**
- **Primary:** [kanboard/kanboard](https://github.com/kanboard/kanboard) — MIT, Kanban project management, ~9.6K stars. Has composer.json, composer.lock, CI, PHP CS config. Moderate size.
- **Alternative:** [bobthecow/psysh](https://github.com/bobthecow/psysh) — MIT, PHP REPL, ~9.8K stars. Composer-based, CI workflows.

**Detection Signals:**
- `composer.json` (primary)
- `composer.lock` (lockfile)
- `vendor/` directory
- `phpunit.xml` or `phpunit.xml.dist`
- `.php-cs-fixer.php` or `.php-cs-fixer.dist.php`
- Edge cases: WordPress projects (different structure), Laravel vs Symfony conventions

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.php.enable = true` with PHP version
- [ ] `composer.json` config section generated with `secure-http: true`, `audit.block-insecure: true`
- [ ] CI workflow includes `composer install --no-scripts --no-interaction` and `composer audit`
- [ ] Pre-commit hooks include `php-cs-fixer`, `phpstan`, `ripsecrets`

---

### 10. Ruby (Bundler)

**Type A — New Project Creation:**
```bash
mkdir test-ruby && cd test-ruby
bundle init
echo 'gem "sinatra"' >> Gemfile
bundle install
# Creates: Gemfile, Gemfile.lock
```
Files created: `Gemfile`, `Gemfile.lock`

**Type B — Existing Project Onboarding:**
- **Primary:** [ruby-grape/grape](https://github.com/ruby-grape/grape) — MIT, REST API framework for Ruby, ~10K stars. Has Gemfile.lock, CI, .rubocop.yml. Clean gem structure.
- **Alternative:** [ankane/pghero](https://github.com/ankane/pghero) — MIT, Postgres performance dashboard, ~8.9K stars. Rails engine, Gemfile.lock, CI.

**Detection Signals:**
- `Gemfile` (primary)
- `Gemfile.lock` (lockfile)
- `.ruby-version` (Ruby version pinning)
- `*.gemspec` (gem packaging)
- `.rubocop.yml` (existing linting config)
- `Rakefile`
- Edge cases: Rails vs non-Rails detection, engine vs application

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.ruby.enable = true` and `languages.ruby.bundler.enable = true`
- [ ] `.bundle/config` generated with `BUNDLE_FROZEN: "true"`
- [ ] CI workflow includes `bundle install --frozen` and `bundle audit check --update`
- [ ] Pre-commit hooks include `rubocop`, `bundler-audit`, `ripsecrets`

---

### 11. Scala (sbt)

**Type A — New Project Creation:**
```bash
sbt new scala/scala3.g8 --name=test-scala
cd test-scala
# Creates: build.sbt, project/, src/
```
Files created: `build.sbt`, `project/build.properties`, `project/plugins.sbt`, `src/`

**Type B — Existing Project Onboarding:**
- **Primary:** [zio/zio](https://github.com/zio/zio) — Apache-2.0, Scala async/concurrent library, ~4.4K stars. sbt-based, CI, moderate size. Canonical Scala library.
- **Alternative:** [JohnSnowLabs/spark-nlp](https://github.com/JohnSnowLabs/spark-nlp) — Apache-2.0, NLP on Spark, ~4.1K stars. sbt + Maven dual build.

**Detection Signals:**
- `build.sbt` (sbt primary)
- `project/build.properties` (sbt version)
- `project/plugins.sbt` (sbt plugins)
- `*.scala` files
- Edge cases: Mill build tool (`build.sc`), sbt multi-project builds, Scala.js cross-compilation

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.scala.enable = true` and `languages.scala.sbt.enable = true`
- [ ] `project/plugins.sbt` gets `sbt-dependency-lock` and `sbt-dependency-check` additions
- [ ] Pre-commit hooks include `scalafmt`, `scalafix`
- [ ] CI workflow includes `sbt dependencyLockCheck` and `sbt dependencyCheck`

---

### 12. Helm

**Type A — New Project Creation:**
```bash
helm create test-helm
cd test-helm
# Creates: Chart.yaml, values.yaml, templates/, charts/, .helmignore
```
Files created: `Chart.yaml`, `values.yaml`, `templates/`, `charts/`, `.helmignore`

**Type B — Existing Project Onboarding:**
- **Primary:** [traefik/traefik-helm-chart](https://github.com/traefik/traefik-helm-chart) — Apache-2.0, Traefik proxy Helm chart, ~1.4K stars. Has Chart.yaml, Chart.lock, CI, values.yaml.
- **Alternative:** [8gears/n8n-helm-chart](https://github.com/8gears/n8n-helm-chart) — Apache-2.0, n8n workflow automation Helm chart, ~700 stars. Clean single-chart structure.

**Detection Signals:**
- `Chart.yaml` (primary)
- `Chart.lock` (dependency lockfile)
- `values.yaml` (configuration values)
- `templates/` directory
- `charts/` directory (subchart dependencies)
- `.helmignore`
- Edge cases: Helmfile (`helmfile.yaml`), OCI-based chart references, library charts vs application charts

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.helm.enable = true`
- [ ] Pre-commit hooks include `helm lint`, `checkov`, `kubeconform`
- [ ] CI workflow includes `helm dependency build` and `helm template | kubeconform`

---

### 13. Ansible (Galaxy)

**Type A — New Project Creation:**
```bash
mkdir test-ansible && cd test-ansible
ansible-galaxy init test-role --init-path=roles/
cat > playbook.yml << 'EOF'
- hosts: all
  roles:
    - test-role
EOF
cat > requirements.yml << 'EOF'
collections:
  - name: community.general
    version: "9.0.0"
EOF
# Creates: playbook.yml, requirements.yml, roles/
```
Files created: `playbook.yml`, `requirements.yml`, `roles/`

**Type B — Existing Project Onboarding:**
- **Primary:** [ansistrano/deploy](https://github.com/ansistrano/deploy) — MIT, Ansible deployment role, ~2.4K stars. Has meta/main.yml, tasks/, CI. Clean role structure.
- **Alternative:** [ansible-lockdown/RHEL8-CIS](https://github.com/ansible-lockdown/RHEL8-CIS) — MIT, CIS benchmark role, ~325 stars. Security-focused, CI, tests.

**Detection Signals:**
- `playbook.yml` / `site.yml` (playbook primary)
- `ansible.cfg` (existing config — must merge)
- `requirements.yml` (Galaxy dependencies)
- `roles/` directory
- `inventory/` or `hosts` file
- `group_vars/`, `host_vars/`
- Edge cases: collection layout vs role layout, molecule testing structure

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.ansible.enable = true`
- [ ] `ansible.cfg` generated with GPG signature verification settings
- [ ] Pre-commit hooks include `ansible-lint`, `yamllint`, `ripsecrets`

---

### 14. Bash/Shell

**Type A — New Project Creation:**
```bash
mkdir test-shell && cd test-shell
cat > script.sh << 'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "Hello, world!"
EOF
chmod +x script.sh
# Creates: script.sh
```
Files created: `script.sh`

**Type B — Existing Project Onboarding:**
- **Primary:** [wfxr/forgit](https://github.com/wfxr/forgit) — MIT, interactive git CLI using fzf, ~5K stars. Shell scripts, CI workflows. Good shell project structure.
- **Alternative:** [tfutils/tfenv](https://github.com/tfutils/tfenv) — MIT, Terraform version manager written in Bash, ~4.9K stars. Pure shell, CI.

**Detection Signals:**
- `*.sh` files
- `#!/usr/bin/env bash` or `#!/bin/bash` shebangs
- `Makefile` with shell commands (partial signal)
- Edge cases: projects that are "mostly shell" vs polyglot projects with a few helper scripts; `.envrc` is shell but shouldn't trigger shell ecosystem

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.shell.enable = true` and adds `shellcheck`, `shfmt`
- [ ] Pre-commit hooks include `shellcheck` (with `--severity=warning`), `shfmt`, `ripsecrets`
- [ ] No package manager configs generated (shell has no PM)

---

### 15. C/C++ (Conan, vcpkg, CMake)

**Type A — New Project Creation:**
```bash
# CMake-based C++ project
mkdir test-cpp && cd test-cpp
cat > CMakeLists.txt << 'EOF'
cmake_minimum_required(VERSION 3.20)
project(test_cpp LANGUAGES CXX)
set(CMAKE_CXX_STANDARD 20)
add_executable(main src/main.cpp)
EOF
mkdir src
cat > src/main.cpp << 'EOF'
#include <iostream>
int main() { std::cout << "Hello\n"; }
EOF
# Creates: CMakeLists.txt, src/main.cpp
```
Files created: `CMakeLists.txt`, `src/main.cpp`

**Type B — Existing Project Onboarding:**
- **Primary:** [erincatto/box2d](https://github.com/erincatto/box2d) — MIT, 2D physics engine (C), CMake-based, ~9.7K stars. Has CMakeLists.txt, CI, moderate size. Well-structured C project.
- **Alternative:** [flightlessmango/MangoHud](https://github.com/flightlessmango/MangoHud) — MIT, Vulkan/OpenGL overlay (C++), Meson-based, ~8.6K stars. Tests Meson detection path.

**Detection Signals:**
- `CMakeLists.txt` (CMake primary)
- `meson.build` (Meson build)
- `conanfile.py` or `conanfile.txt` (Conan PM)
- `vcpkg.json` (vcpkg manifest)
- `conan.lock` (Conan lockfile)
- `Makefile` (generic make — weak signal)
- `*.c`, `*.cpp`, `*.h`, `*.hpp` files
- Edge cases: Conan vs vcpkg coexistence, FetchContent (no lockfile), Meson WrapDB

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.c.enable = true` and/or `languages.cplusplus.enable = true`
- [ ] Packages include `cmake`, `ninja`, and detected PM (`conan_2` or `vcpkg`)
- [ ] If Conan: lockfile enforcement in CI (`conan install . --lockfile=conan.lock`)
- [ ] Pre-commit hooks include `clang-format`, `clang-tidy`, `ripsecrets`

---

## Tier 3 — Nice to Have

### 16. Elixir (Mix/Hex)

**Type A — New Project Creation:**
```bash
mix new test_elixir
cd test_elixir
# Add deps to mix.exs, then:
mix deps.get
# Creates: mix.exs, mix.lock, lib/, test/
```
Files created: `mix.exs`, `mix.lock`, `lib/`, `test/`

**Type B — Existing Project Onboarding:**
- **Primary:** [philss/floki](https://github.com/philss/floki) — MIT, HTML parser for Elixir, ~2.1K stars. Has mix.exs, mix.lock, CI. Clean OTP structure, moderate size.
- **Alternative:** [elixir-tesla/tesla](https://github.com/elixir-tesla/tesla) — MIT, HTTP client for Elixir, ~2.1K stars. mix.lock, CI, middleware architecture.

**Detection Signals:**
- `mix.exs` (primary)
- `mix.lock` (lockfile with Hex content hashes)
- `lib/` directory
- `config/` directory
- Edge cases: Phoenix framework projects (additional detection via `config/dev.exs`, `priv/`), umbrella applications

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.elixir.enable = true`
- [ ] `mix.exs` gets `mix_audit` dependency addition suggestion
- [ ] CI workflow includes `mix deps.get --check-locked` and `mix hex.audit`
- [ ] Pre-commit hooks include `mix format`, `credo`

---

### 17. Dart/Flutter (pub)

**Type A — New Project Creation:**
```bash
# Pure Dart
dart create test_dart
cd test_dart
# Creates: pubspec.yaml, pubspec.lock, lib/, test/, bin/, analysis_options.yaml

# Flutter
flutter create test_flutter
cd test_flutter
# Creates: pubspec.yaml, pubspec.lock, lib/, test/, android/, ios/, web/
```
Files created: `pubspec.yaml`, `pubspec.lock`, `lib/`, `test/`, `analysis_options.yaml`

**Type B — Existing Project Onboarding:**
- **Primary:** [simolus3/drift](https://github.com/simolus3/drift) — MIT, reactive persistence library for Dart/Flutter, ~3.2K stars. pubspec.yaml, pubspec.lock, CI. Multi-package Dart repo.
- **Alternative:** [gskinnerTeam/flutter-wonderous-app](https://github.com/gskinnerTeam/flutter-wonderous-app) — MIT, Flutter showcase app, ~4.5K stars. Full Flutter project with pubspec.lock.

**Detection Signals:**
- `pubspec.yaml` (primary)
- `pubspec.lock` (lockfile with SHA256 content hashes)
- `analysis_options.yaml` (linting config)
- `android/`, `ios/`, `web/` directories (Flutter indicator)
- Edge cases: Pure Dart vs Flutter (different toolchain), Dart workspace (`dart_workspace.yaml`)

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.dart.enable = true`, optionally `pkgs.flutter`
- [ ] CI workflow includes `dart pub get --enforce-lockfile` (or `flutter pub get --enforce-lockfile`)
- [ ] Pre-commit hooks include `dart format`, `dart analyze`

---

### 18. Swift (SPM)

**Type A — New Project Creation:**
```bash
mkdir test-swift && cd test-swift
swift package init --type executable
swift package resolve
# Creates: Package.swift, Package.resolved, Sources/, Tests/
```
Files created: `Package.swift`, `Package.resolved`, `Sources/`, `Tests/`

**Type B — Existing Project Onboarding:**
- **Primary:** [EFPrefix/EFQRCode](https://github.com/EFPrefix/EFQRCode) — MIT, QR code library for Swift, ~4.7K stars. Package.swift, Package.resolved, CI.
- **Alternative:** [devicekit/DeviceKit](https://github.com/devicekit/DeviceKit) — MIT, device detection for Apple platforms, ~4.7K stars. SPM-based, CI.

**Detection Signals:**
- `Package.swift` (SPM primary)
- `Package.resolved` (SPM lockfile)
- `*.xcodeproj` or `*.xcworkspace` (Xcode project)
- `Podfile` (CocoaPods — legacy)
- `Cartfile` (Carthage — legacy)
- Edge cases: Xcode-only vs SPM, CocoaPods migration projects, mixed Objective-C/Swift

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.swift.enable = true`
- [ ] Pre-commit hooks include `swiftformat`, `swiftlint`
- [ ] SPM trust policy documented in CLAUDE.md

---

### 19. Haskell (Cabal, Stack)

**Type A — New Project Creation:**
```bash
# Cabal
mkdir test-haskell && cd test-haskell
cabal init --minimal --non-interactive
cabal build
# Creates: test-haskell.cabal, cabal.project (optional), app/ or src/

# Stack
stack new test-haskell
cd test-haskell
stack build
# Creates: package.yaml, stack.yaml, stack.yaml.lock, src/, app/
```
Files created (Cabal): `*.cabal`, `CHANGELOG.md`, `app/` or `src/`
Files created (Stack): `package.yaml`, `stack.yaml`, `stack.yaml.lock`, `src/`

**Type B — Existing Project Onboarding:**
- **Primary:** [kmonad/kmonad](https://github.com/kmonad/kmonad) — MIT, keyboard manager, ~5K stars. Stack-based (stack.yaml, stack.yaml.lock), CI. Good moderate-size Haskell project.
- **Alternative:** [nmattia/niv](https://github.com/nmattia/niv) — MIT, Nix dependency management tool written in Haskell, ~1.8K stars. Cabal-based, CI.

**Detection Signals:**
- `*.cabal` (Cabal primary)
- `cabal.project` (Cabal project config)
- `cabal.project.freeze` (Cabal pseudo-lockfile)
- `stack.yaml` (Stack primary)
- `stack.yaml.lock` (Stack lockfile)
- `package.yaml` (hpack — generates .cabal)
- Edge cases: Cabal vs Stack detection priority, hpack-generated .cabal files, Nix-based Haskell builds

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.haskell.enable = true`
- [ ] If Stack: `languages.haskell.stack.enable = true`, CI uses `stack build --locked`
- [ ] If Cabal: `cabal.project` gets `index-state` pinning suggestion
- [ ] Pre-commit hooks include `ormolu` or `fourmolu`, `hlint`

---

### 20. Clojure (deps.edn)

**Type A — New Project Creation:**
```bash
mkdir test-clojure && cd test-clojure
cat > deps.edn << 'EOF'
{:deps {org.clojure/clojure {:mvn/version "1.12.0"}}
 :paths ["src"]}
EOF
mkdir src
# Creates: deps.edn, src/

# Or with Leiningen:
lein new app test-clojure
cd test-clojure
# Creates: project.clj, src/, test/, resources/
```
Files created (tools.deps): `deps.edn`, `src/`
Files created (Leiningen): `project.clj`, `src/`, `test/`, `resources/`

**Type B — Existing Project Onboarding:**
- **Primary:** [babashka/babashka](https://github.com/babashka/babashka) — EPL-1.0, Clojure scripting runtime, ~4.5K stars. deps.edn, CI workflows. Major Clojure project.
- **Alternative:** [ring-clojure/ring](https://github.com/ring-clojure/ring) — MIT, HTTP server abstraction, ~3.9K stars. Leiningen-based (project.clj), CI.

**Detection Signals:**
- `deps.edn` (tools.deps primary)
- `project.clj` (Leiningen primary)
- `bb.edn` (Babashka tasks)
- `*.clj`, `*.cljs`, `*.cljc` files
- Edge cases: Clojure vs ClojureScript, Babashka-only projects, deps.edn + Leiningen coexistence

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.clojure.enable = true`
- [ ] Pre-commit hooks include `cljfmt`, `clj-kondo`
- [ ] CLAUDE.md documents version pinning best practices (no lockfile in Clojure ecosystem)

---

### 21. Bazel (bzlmod)

**Type A — New Project Creation:**
```bash
mkdir test-bazel && cd test-bazel
cat > MODULE.bazel << 'EOF'
module(
    name = "test_bazel",
    version = "0.1.0",
)
bazel_dep(name = "rules_go", version = "0.50.1")
EOF
cat > BUILD.bazel << 'EOF'
# Empty root BUILD
EOF
bazel mod deps
# Creates: MODULE.bazel, MODULE.bazel.lock, BUILD.bazel
```
Files created: `MODULE.bazel`, `MODULE.bazel.lock`, `BUILD.bazel`

**Type B — Existing Project Onboarding:**
- **Primary:** [aspect-build/bazel-examples](https://github.com/aspect-build/bazel-examples) — Apache-2.0, official Bazel examples with MODULE.bazel, multi-language. Tests Bazel detection with various rule sets.
- **Alternative:** [bazelbuild/rules_go](https://github.com/bazelbuild/rules_go) — Apache-2.0, Go rules for Bazel, ~1.4K stars. MODULE.bazel, CI. Tests Bazel+Go polyglot.

**Detection Signals:**
- `MODULE.bazel` (bzlmod primary)
- `MODULE.bazel.lock` (bzlmod lockfile)
- `WORKSPACE` or `WORKSPACE.bazel` (legacy)
- `BUILD.bazel` or `BUILD` files
- `.bazelrc` (existing config — must merge)
- Edge cases: bzlmod vs WORKSPACE migration, custom registries, remote execution configs

**Verification Checklist:**
- [ ] `devenv.nix` adds `pkgs.bazel_7` to packages
- [ ] `.bazelrc` generated with sandbox settings, `--lockfile_mode=error` for CI
- [ ] Pre-commit hooks include `buildifier`

---

### 22. Nix (flake inputs)

**Type A — New Project Creation:**
```bash
mkdir test-nix && cd test-nix
nix flake init
# Creates: flake.nix, flake.lock (after first eval)
```
Files created: `flake.nix`, `flake.lock`

**Type B — Existing Project Onboarding:**
- **Primary:** [nmattia/niv](https://github.com/nmattia/niv) — MIT, Nix dependency management, ~1.8K stars. flake.nix, flake.lock, CI. Pure Nix project.
- **Alternative:** [nix-community/home-manager](https://github.com/nix-community/home-manager) — MIT, Nix-based dotfile manager. flake.nix, large module system. Tests existing Nix flake merge.

**Detection Signals:**
- `flake.nix` (Nix flake primary)
- `flake.lock` (flake lockfile)
- `default.nix` (legacy Nix expression)
- `shell.nix` (legacy dev shell)
- `devenv.nix` + `devenv.yaml` (already using devenv — special case)
- Edge cases: project already using devenv (no-op or upgrade), nix-darwin configs, NixOS modules

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.nix.enable = true`
- [ ] Pre-commit hooks include `statix`, `nixfmt` or `alejandra`, `deadnix`, `flake-checker`
- [ ] Existing `flake.nix` / `devenv.nix` detected and NOT overwritten
- [ ] If devenv already present, gdev offers upgrade path only

---

## Tier 4 — Reference Docs Only

### 23. Perl (Carton)

**Type A — New Project Creation:**
```bash
mkdir test-perl && cd test-perl
cat > cpanfile << 'EOF'
requires 'Mojolicious', '== 9.35';
requires 'DBI', '== 1.643';
EOF
carton install
# Creates: cpanfile, cpanfile.snapshot, local/
```
Files created: `cpanfile`, `cpanfile.snapshot`, `local/`

**Type B — Existing Project Onboarding:**
- **Suggested:** [mojolicious/mojo](https://github.com/mojolicious/mojo) — Artistic-2.0, Perl web framework. The canonical Perl project. Has cpanfile, CI. Note: Artistic license (not MIT/Apache/BSD) — may need to find alternative or use for reference only.

**Detection Signals:**
- `cpanfile` (Carton primary)
- `cpanfile.snapshot` (lockfile)
- `Makefile.PL` or `Build.PL` (module build)
- `*.pm`, `*.pl` files
- `lib/` directory (Perl convention)

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.perl.enable = true`
- [ ] CI workflow includes `carton install --deployment`
- [ ] Pre-commit hooks include `perltidy`, `perlcritic`

---

### 24. R (renv)

**Type A — New Project Creation:**
```r
# In R console:
renv::init()
install.packages("ggplot2")
renv::snapshot()
# Creates: renv.lock, .Rprofile, renv/
```
Files created: `renv.lock`, `.Rprofile`, `renv/settings.json`

**Type B — Existing Project Onboarding:**
- **Suggested:** [tidyverse/ggplot2](https://github.com/tidyverse/ggplot2) — MIT, plotting library for R, ~6.6K stars. Has DESCRIPTION file (R package format), CI. Though it's a library not an app, it demonstrates R ecosystem detection.

**Detection Signals:**
- `renv.lock` (renv lockfile)
- `.Rprofile` (R profile config)
- `DESCRIPTION` (R package metadata)
- `NAMESPACE` (R namespace)
- `*.R`, `*.Rmd` files
- `renv/` directory

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.r.enable = true`
- [ ] `.Rprofile` forces HTTPS CRAN mirror
- [ ] Pre-commit hooks include `styler`, `lintr`

---

### 25. Lua (LuaRocks)

**Type A — New Project Creation:**
```bash
mkdir test-lua && cd test-lua
cat > test-lua-1.0-1.rockspec << 'EOF'
package = "test-lua"
version = "1.0-1"
source = { url = "..." }
dependencies = { "lua >= 5.4", "lpeg >= 1.1" }
build = { type = "builtin", modules = { ["test"] = "src/test.lua" } }
EOF
luarocks install --local lpeg
# Creates: *.rockspec, src/
```
Files created: `*.rockspec`, `src/`

**Type B — Existing Project Onboarding:**
- **Suggested:** [lunarmodules/luacheck](https://github.com/lunarmodules/luacheck) — MIT, Lua linter, ~1.9K stars. Has rockspec, CI. Pure Lua project.

**Detection Signals:**
- `*.rockspec` (LuaRocks primary)
- `lux.toml` (Lux PM — newer alternative)
- `lux.lock` (Lux lockfile)
- `*.lua` files
- `.luacheckrc` (luacheck config)

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.lua.enable = true`
- [ ] Pre-commit hooks include `stylua`, `luacheck`

---

### 26. Zig

**Type A — New Project Creation:**
```bash
mkdir test-zig && cd test-zig
zig init
# Creates: build.zig, build.zig.zon, src/main.zig, src/root.zig
```
Files created: `build.zig`, `build.zig.zon`, `src/main.zig`

**Type B — Existing Project Onboarding:**
- **Suggested:** [zigtools/zls](https://github.com/zigtools/zls) — MIT, Zig Language Server, ~3.3K stars. Has build.zig, build.zig.zon with dependency hashes, CI.

**Detection Signals:**
- `build.zig` (Zig build script)
- `build.zig.zon` (Zig package manifest with SHA256 hashes)
- `*.zig` files
- Edge cases: dependencies use content-addressed hashing (no separate lockfile)

**Verification Checklist:**
- [ ] `devenv.nix` contains `languages.zig.enable = true`
- [ ] Pre-commit hooks include `zig fmt`
- [ ] No separate security config needed (Zig uses content-addressed hashing by design)

---

### 27. PowerShell (PSGallery)

**Type A — New Project Creation:**
```powershell
mkdir test-powershell && cd test-powershell
New-ModuleManifest -Path .\TestModule.psd1 -RootModule TestModule.psm1
# Or with requirements file:
@{
    'Pester' = @{ Version = '5.6.1'; Repository = 'PSGallery' }
    'PSScriptAnalyzer' = @{ Version = '1.22.0'; Repository = 'PSGallery' }
} | Export-Clixml requirements.psd1
# Creates: *.psd1, *.psm1
```
Files created: `*.psd1` (module manifest), `*.psm1` (module script)

**Type B — Existing Project Onboarding:**
- **Suggested:** [PowerShell/PSScriptAnalyzer](https://github.com/PowerShell/PSScriptAnalyzer) — MIT, PowerShell static analysis tool, ~1.9K stars. Has .psd1, CI.

**Detection Signals:**
- `*.psd1` (module manifest)
- `*.psm1` (module script)
- `*.ps1` (PowerShell scripts)
- `requirements.psd1` (requirements file)
- Edge cases: C# hybrid modules (.csproj + .psd1)

**Verification Checklist:**
- [ ] `devenv.nix` adds `pkgs.powershell` to packages
- [ ] Pre-commit hooks include PSScriptAnalyzer invocation

---

## Polyglot Combo Test Targets

These repos combine multiple ecosystems and test gdev's composition logic — detecting all ecosystems present and generating unified config.

### Combo 1: Go + TypeScript + Docker + Terraform

**Target:** [rcourtman/Pulse](https://github.com/rcourtman/Pulse)
- **License:** MIT
- **Stars:** ~5.7K
- **Ecosystems:** TypeScript frontend, Node.js backend, Docker (Dockerfile + compose), possible Terraform deployment
- **Why:** Tests the most common full-stack web app pattern with containerization. Validates that `qsdev init` detects multiple ecosystems and generates a unified devenv.nix.

**Verification:**
- [ ] All detected ecosystems appear in devenv.nix
- [ ] Single `.envrc` for all ecosystems
- [ ] Pre-commit hooks cover all detected languages
- [ ] No conflicting security configs between ecosystems

### Combo 2: Go + Docker + Helm + Terraform

**Target:** [gruntwork-io/terragrunt](https://github.com/gruntwork-io/terragrunt)
- **License:** MIT
- **Stars:** ~9.6K
- **Ecosystems:** Go (go.mod), Docker (Dockerfile), Terraform/HCL (test fixtures)
- **Why:** Go CLI tool with IaC testing infrastructure. Tests Go + Terraform + Docker detection in a DevOps tool context.

**Verification:**
- [ ] Go and Terraform both detected
- [ ] Docker config detected from Dockerfile
- [ ] Security configs generated for all three ecosystems
- [ ] CI workflow covers Go tests, Terraform validation, and Docker linting

### Combo 3: Python + TypeScript + Docker

**Target:** [sequinstream/sequin](https://github.com/sequinstream/sequin)
- **License:** MIT
- **Stars:** ~2.1K
- **Ecosystems:** Elixir (primary), TypeScript (dashboard), Docker (deployment)
- **Why:** Tests Elixir + TypeScript + Docker composition. Exercises Tier 3 ecosystem (Elixir) alongside Tier 1.

**Alternative:** [tiangolo/full-stack-fastapi-template](https://github.com/tiangolo/full-stack-fastapi-template)
- **License:** MIT
- **Ecosystems:** Python (FastAPI backend), TypeScript (React frontend), Docker (docker-compose), potentially PostgreSQL
- **Why:** Canonical Python + TypeScript + Docker full-stack template. Tests the most common Python web app polyglot pattern.

### Combo 4: Java + Kotlin + Docker + Helm

**Target:** [testcontainers/testcontainers-java](https://github.com/testcontainers/testcontainers-java)
- **License:** MIT
- **Stars:** ~8.6K
- **Ecosystems:** Java (Gradle), Kotlin (Gradle), Docker (Docker client integration)
- **Why:** Java/Kotlin multi-module Gradle project with Docker integration. Tests JVM ecosystem detection with Docker.

**Verification:**
- [ ] Java and Kotlin both detected
- [ ] Gradle build system detected (not Maven)
- [ ] Docker tooling detected
- [ ] JDK version set appropriately in devenv.nix

### Combo 5: Rust + TypeScript + Docker

**Target:** [astral-sh/ruff](https://github.com/astral-sh/ruff)
- **License:** MIT
- **Stars:** ~40K
- **Ecosystems:** Rust (Cargo.toml), Python (pyproject.toml for the Python package), potentially Docker
- **Why:** Rust core with Python packaging layer. Tests Rust + Python ecosystem detection and correct security configs for both.

**Alternative for Rust + TS + Docker:** [nicholasgasior/gcloud-pubsub-emulator](https://github.com/nicholasgasior/gcloud-pubsub-emulator) or similar Rust web service with TypeScript client.

---

## Detection Signal Summary Table

Quick reference for all primary detection files across ecosystems:

| Ecosystem | Primary Detection File(s) | Lockfile | PM-Specific Config |
|-----------|--------------------------|----------|-------------------|
| JS/npm | `package.json` + `package-lock.json` | `package-lock.json` | `.npmrc` |
| JS/pnpm | `package.json` + `pnpm-lock.yaml` | `pnpm-lock.yaml` | `pnpm-workspace.yaml` |
| JS/yarn | `package.json` + `yarn.lock` | `yarn.lock` | `.yarnrc.yml` |
| JS/bun | `package.json` + `bun.lock` | `bun.lock` | `bunfig.toml` |
| Python/pip | `requirements.txt` | `requirements.txt` (manual) | `pip.conf` |
| Python/uv | `pyproject.toml` + `uv.lock` | `uv.lock` | `[tool.uv]` in pyproject.toml |
| Python/poetry | `pyproject.toml` + `poetry.lock` | `poetry.lock` | `[tool.poetry]` in pyproject.toml |
| Go | `go.mod` | `go.sum` | `GOFLAGS` env |
| Rust | `Cargo.toml` | `Cargo.lock` | `.cargo/config.toml` |
| Java/Maven | `pom.xml` | N/A (version pins in pom) | `settings.xml` |
| Java/Gradle | `build.gradle(.kts)` | `gradle.lockfile` (opt-in) | `gradle.properties` |
| Kotlin/Gradle | `build.gradle.kts` + `*.kt` | `gradle.lockfile` (opt-in) | `gradle.properties` |
| C#/.NET | `*.csproj` / `*.sln` | `packages.lock.json` (opt-in) | `nuget.config` |
| Docker | `Dockerfile` / `Containerfile` | N/A | `.hadolint.yaml` |
| Terraform | `*.tf` | `.terraform.lock.hcl` | `.terraformrc` |
| PHP | `composer.json` | `composer.lock` | `composer.json` config section |
| Ruby | `Gemfile` | `Gemfile.lock` | `.bundle/config` |
| Scala | `build.sbt` | N/A (plugin-based) | `project/plugins.sbt` |
| Helm | `Chart.yaml` | `Chart.lock` | N/A |
| Ansible | `playbook.yml` + `requirements.yml` | N/A | `ansible.cfg` |
| Bash/Shell | `*.sh` files | N/A | N/A |
| C/C++ | `CMakeLists.txt` / `conanfile.*` / `vcpkg.json` | `conan.lock` | Conan profile |
| Elixir | `mix.exs` | `mix.lock` | N/A |
| Dart/Flutter | `pubspec.yaml` | `pubspec.lock` | N/A |
| Swift | `Package.swift` | `Package.resolved` | SPM trust config |
| Haskell/Stack | `stack.yaml` | `stack.yaml.lock` | N/A |
| Haskell/Cabal | `*.cabal` | `cabal.project.freeze` | `cabal.project` |
| Clojure | `deps.edn` / `project.clj` | N/A | N/A |
| Bazel | `MODULE.bazel` | `MODULE.bazel.lock` | `.bazelrc` |
| Nix | `flake.nix` | `flake.lock` | N/A |
| Perl | `cpanfile` | `cpanfile.snapshot` | N/A |
| R | `renv.lock` / `DESCRIPTION` | `renv.lock` | `.Rprofile` |
| Lua | `*.rockspec` / `lux.toml` | `lux.lock` | N/A |
| Zig | `build.zig.zon` | Inline hashes | N/A |
| PowerShell | `*.psd1` / `*.psm1` | N/A | N/A |

---

## Detection Priority / Conflict Resolution

When multiple ecosystems are detected, gdev must resolve conflicts:

1. **PM detection within JS/TS**: Priority order for lockfile presence: `bun.lock` > `pnpm-lock.yaml` > `yarn.lock` > `package-lock.json`. If multiple lockfiles exist, warn user and pick highest-priority. Respect `packageManager` field in package.json if present.

2. **PM detection within Python**: Priority: `uv.lock` > `poetry.lock` > `Pipfile.lock` > `requirements.txt`. Check `pyproject.toml` for `[tool.uv]` vs `[tool.poetry]` to disambiguate.

3. **PM detection within Haskell**: Stack (`stack.yaml`) vs Cabal (`*.cabal` + `cabal.project`). If both present, prefer Stack (more common in practice).

4. **PM detection within C/C++**: Check for Conan (`conanfile.*`), vcpkg (`vcpkg.json`), Meson (`meson.build`), CMake (`CMakeLists.txt`). Multiple can coexist.

5. **Build tool within JVM**: Maven (`pom.xml`) vs Gradle (`build.gradle*`). Both can coexist in transitioning projects — generate configs for whichever is primary (detect from CI workflow if ambiguous).

6. **Terraform vs OpenTofu**: If `.terraformrc` references OpenTofu, or if `tofu` binary is in PATH, prefer OpenTofu module.

7. **Already-devenv projects**: If `devenv.nix` + `devenv.yaml` exist, do NOT overwrite. Offer merge/upgrade mode only.

---

## Test Execution Strategy

### Phase 1: Greenfield (Type A) — Automated
For each ecosystem, the test harness should:
1. Create a temp directory
2. Run the new-project commands listed above
3. Run `qsdev init --yes`
4. Verify all checklist items programmatically (file existence, content assertions)
5. Run `devenv shell --command "echo ok"` to validate the generated devenv.nix parses

### Phase 2: Brownfield (Type B) — Semi-Automated
For each ecosystem, the test harness should:
1. `git clone --depth=1` the target repo into a temp directory
2. Run `qsdev init` (interactive or with `--yes`)
3. Verify checklist items
4. Verify NO clobbering: diff existing config files to ensure user content preserved
5. Verify merge behavior: security additions appear alongside existing content

### Phase 3: Polyglot (Combo) — Manual + Automated
For each combo repo:
1. Clone the repo
2. Run `qsdev init --yes`
3. Verify ALL detected ecosystems appear in devenv.nix
4. Verify security configs generated for each detected ecosystem
5. Verify no conflicts between ecosystem configs
6. Run `devenv shell` to validate combined config works
