# Language Ecosystem Security Coverage

Comprehensive reference for the gdev secure development environment bootstrap addon. Covers package managers, security hardening configurations, vulnerability scanning tools, devenv.sh module support, and pre-commit hooks for every language/platform a consulting firm encounters.

**Last updated:** 2026-05-12

---

## Table of Contents

1. [JVM Ecosystems (Java, Kotlin, Scala, Groovy, Clojure)](#1-jvm-ecosystems)
2. [.NET Ecosystems (C#, F#)](#2-net-ecosystems)
3. [JavaScript/TypeScript (npm, yarn, pnpm, bun)](#3-javascripttypescript)
4. [PHP (Composer)](#4-php)
5. [Ruby (Bundler, RubyGems)](#5-ruby)
6. [Swift & Objective-C](#6-swift--objective-c)
7. [Dart/Flutter](#7-dartflutter)
8. [Python (pip, uv, poetry)](#8-python)
9. [Go](#9-go)
10. [Rust](#10-rust)
11. [Elixir/Erlang](#11-elixirerlang)
12. [Haskell](#12-haskell)
13. [Perl](#13-perl)
14. [R](#14-r)
15. [Lua](#15-lua)
16. [Zig](#16-zig)
17. [C/C++](#17-cc)
18. [Bash/Shell](#18-bashshell)
19. [PowerShell](#19-powershell)
20. [Terraform](#20-terraform)
21. [Pulumi](#21-pulumi)
22. [Ansible](#22-ansible)
23. [Helm](#23-helm)
24. [Docker/Containerfiles](#24-dockercontainerfiles)
25. [Bazel](#25-bazel)
26. [Nix](#26-nix)
27. [WASM](#27-wasm)
28. [Cross-Ecosystem Tools](#28-cross-ecosystem-tools)

---

## 1. JVM Ecosystems

Covers Java, Kotlin, Scala, Groovy, and Clojure. These share the JVM dependency infrastructure (Maven Central, Gradle Plugin Portal) but have ecosystem-specific tooling.

### 1.1 Java

#### Package Managers / Build Tools

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **Maven** (official) | `pom.xml` pins versions; no native lockfile | `settings.xml`, `.mvn/maven.config` |
| **Gradle** (dominant for Android, common elsewhere) | `gradle.lockfile` (opt-in), `gradle/verification-metadata.xml` | `gradle.properties`, `settings.gradle.kts` |
| **Ant** (legacy) | `ivy.xml` with Ivy | `build.xml`, `ivysettings.xml` |
| **Bazel** (monorepo) | `MODULE.bazel.lock` | `MODULE.bazel`, `WORKSPACE` |

#### Security Hardening Configs

**Maven `settings.xml` — Hardened:**
```xml
<settings xmlns="http://maven.apache.org/SETTINGS/1.2.0">
  <profiles>
    <profile>
      <id>security-hardened</id>
      <repositories>
        <repository>
          <id>central</id>
          <url>https://repo.maven.apache.org/maven2</url>
          <releases>
            <checksumPolicy>fail</checksumPolicy>
          </releases>
          <snapshots>
            <enabled>false</enabled>
            <checksumPolicy>fail</checksumPolicy>
          </snapshots>
        </repository>
      </repositories>
      <pluginRepositories>
        <pluginRepository>
          <id>central</id>
          <url>https://repo.maven.apache.org/maven2</url>
          <releases>
            <checksumPolicy>fail</checksumPolicy>
          </releases>
          <snapshots>
            <enabled>false</enabled>
          </snapshots>
        </pluginRepository>
      </pluginRepositories>
    </profile>
  </profiles>
  <activeProfiles>
    <activeProfile>security-hardened</activeProfile>
  </activeProfiles>
  <!-- Block snapshot repositories (mutable artifacts = supply chain risk) -->
  <mirrors>
    <mirror>
      <id>block-snapshots</id>
      <mirrorOf>*,!central</mirrorOf>
      <url>https://repo.maven.apache.org/maven2</url>
    </mirror>
  </mirrors>
</settings>
```

**Maven CI command:**
```bash
# Strict checksum verification on CLI
mvn verify --strict-checksums
```

**Gradle `gradle/verification-metadata.xml` — Bootstrap:**
```bash
# Generate initial verification metadata (review before committing!)
./gradlew --write-verification-metadata sha256,pgp
```

**Gradle `gradle.properties` — Hardened:**
```properties
# Enable dependency locking globally
dependencyLocking.lockMode=STRICT

# Use HTTPS for wrapper downloads
distributionUrl=https\://services.gradle.org/distributions/gradle-8.12-bin.zip
distributionSha256Sum=<pin-sha256-here>

# Require verification metadata
systemProp.org.gradle.dependency.verification=strict
```

**Gradle `settings.gradle.kts` — Repository content filtering:**
```kotlin
dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        mavenCentral {
            content {
                // Block internal group IDs from resolving on public repos
                excludeGroupByRegex("com\\.yourcompany\\..*")
            }
        }
    }
}
```

**Gradle lockfile generation:**
```bash
# Generate lockfiles for all configurations
./gradlew dependencies --write-locks
# CI: fail if lockfile is stale
./gradlew dependencies --update-locks '*:*' && git diff --exit-code gradle.lockfile
```

#### Vulnerability Scanning

| Tool | Type | Ecosystems | Notes |
|------|------|-----------|-------|
| **OSV-Scanner** | Free CLI | Maven, Gradle | Google-backed, uses OSV DB |
| **Snyk** | Freemium CLI+SaaS | Maven, Gradle, Ant | 16 languages, SAST+SCA |
| **Socket.dev** | SaaS+CLI | Maven | Behavioral analysis beyond CVEs |
| **OWASP Dependency-Check** | Free CLI | Maven (plugin), Gradle (plugin) | NVD-backed, mature |
| **Trivy** | Free CLI | Maven, Gradle | Note: March 2026 supply chain compromise — verify provenance |
| **sbt-dependency-check** | Free sbt plugin | sbt (Scala/Java) | OWASP wrapper for sbt |

#### devenv.sh Module

```nix
languages.java = {
  enable = true;
  jdk.package = pkgs.jdk21;      # Pin JDK version
  maven.enable = true;            # Adds mvn to PATH
  maven.package = pkgs.maven;
  gradle.enable = true;           # Adds gradle to PATH
  # gradle.package inherits JDK automatically
  lsp.enable = true;              # JDT language server
  lsp.package = pkgs.jdt-language-server;
};
```

**Module name:** `languages.java`
**Key options:** `enable`, `jdk.package`, `maven.enable`, `maven.package`, `gradle.enable`, `gradle.package`, `lsp.enable`, `lsp.package`

#### Pre-Commit Hooks

| Hook | Purpose | Repo |
|------|---------|------|
| `google-java-format` | Formatting | `https://github.com/macisamuele/language-formatters-pre-commit-hooks` |
| `checkstyle` | Style enforcement | `https://github.com/cristianoliveira/java-checkstyle` |
| `spotbugs` | Bug detection (via Gradle/Maven task) | Run as build step |
| `pmd` | Static analysis | Run as build step |
| `ripsecrets` | Secrets detection | `https://github.com/sirwart/ripsecrets` |

---

### 1.2 Kotlin

#### Package Managers / Build Tools

Same as Java (Maven, Gradle, Bazel). Kotlin projects overwhelmingly use Gradle with `build.gradle.kts`.

#### Security Hardening

Same Gradle hardening as Java (verification-metadata.xml, lockfiles, repository content filtering). All Java configs apply.

#### Kotlin-Specific Scanning

| Tool | Notes |
|------|-------|
| **Detekt** | Static analysis for Kotlin (like ESLint for JS) |
| **Snyk** | Full Kotlin/Gradle support |
| **OSV-Scanner** | Parses Gradle lockfiles |

#### devenv.sh Module

```nix
languages.kotlin = {
  enable = true;
  lsp.enable = true;
  # lsp.package = pkgs.kotlin-language-server;
};
# Kotlin inherits JDK from languages.java.jdk.package
languages.java.jdk.package = pkgs.jdk21;
```

**Module name:** `languages.kotlin`
**Key options:** `enable`, `lsp.enable`, `lsp.package`

#### Pre-Commit Hooks

| Hook | Purpose | Repo |
|------|---------|------|
| `ktlint` | Formatting + linting | `https://github.com/JLLeitworst/ktlint-gradle` (via Gradle task) |
| `detekt` | Static analysis | Via Gradle: `./gradlew detekt` |
| `diktat` | Strict Kotlin style | Via Gradle plugin |

---

### 1.3 Scala

#### Package Managers / Build Tools

| Tool | Lockfile | Notes |
|------|----------|-------|
| **sbt** (dominant) | `build.sbt.lock` (via sbt-dependency-lock plugin) | Not built-in — requires plugin |
| **Mill** | No native lockfile | Newer, faster builds |
| **Maven** | Same as Java | Less common for Scala |

#### Security Hardening

**sbt — Dependency locking:**
```scala
// project/plugins.sbt
addSbtPlugin("software.purpledragon" % "sbt-dependency-lock" % "1.6.0")

// build.sbt — CI check
// Run: sbt dependencyLockCheck
```

**sbt — Vulnerability scanning plugin:**
```scala
// project/plugins.sbt
addSbtPlugin("net.vonbuchholtz" % "sbt-dependency-check" % "5.0.0")
// Run: sbt dependencyCheck
```

#### devenv.sh Module

```nix
languages.scala = {
  enable = true;
  package = pkgs.scala_3;
  sbt.enable = true;               # Adds sbt to PATH
  # sbt.package = pkgs.sbt;
  # mill.enable = true;            # Alternative build tool
  lsp.enable = true;               # Metals language server
};
languages.java.jdk.package = pkgs.jdk21; # Scala tools inherit JDK
```

**Module name:** `languages.scala`
**Key options:** `enable`, `package`, `sbt.enable`, `sbt.package`, `mill.enable`, `mill.package`, `lsp.enable`, `lsp.package`

#### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `scalafmt` | Formatting (via sbt task or standalone) |
| `scalafix` | Linting/refactoring |
| `sbt-dependency-check` | OWASP vuln scanning (CI) |

---

### 1.4 Groovy

#### Package Managers / Build Tools

Groovy projects use **Gradle** (dominant) or **Maven**. Groovy scripts can use **Grape** (`@Grab` annotations) for ad-hoc dependency resolution from Maven Central.

#### Security Hardening

- All Gradle hardening from Java section applies
- **Grape** has no lockfile, no signing, no security controls — avoid in production; use only for standalone scripts
- For Gradle-built Groovy projects, use `verification-metadata.xml` and lockfiles identically to Java

#### devenv.sh Module

No dedicated Groovy module. Use `languages.java` with Groovy added to packages:

```nix
languages.java.enable = true;
packages = [ pkgs.groovy ];
```

#### Pre-Commit Hooks

Same as Java/Gradle. CodeNarc is the Groovy-specific static analysis tool (run via Gradle task).

---

### 1.5 Clojure

#### Package Managers / Build Tools

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **tools.deps** (official CLI) | No native lockfile | `deps.edn` |
| **Leiningen** | No native lockfile | `project.clj` |

Neither tools.deps nor Leiningen has native lockfile support. Dependency resolution is based on Maven coordinates with version pinning in config files.

#### Security Hardening

**deps.edn — Pin all versions explicitly:**
```clojure
{:deps {org.clojure/clojure {:mvn/version "1.12.0"}
        ;; Always pin explicit versions — no ranges
        }}
```

**Vulnerability scanning:**
```bash
# nvd-clojure — NVD database checker for Clojure
clojure -Tclj-watson scan
# or with Leiningen:
lein nvd check
```

| Tool | Type | Notes |
|------|------|-------|
| **nvd-clojure** | Free CLI/Lein plugin | NVD database |
| **clj-watson** | Free CLI | SCA scanner for deps.edn |
| **OSV-Scanner** | Free CLI | Parses pom.xml (transitive) |

#### devenv.sh Module

```nix
languages.clojure = {
  enable = true;
  lsp.enable = true;  # clojure-lsp
};
```

**Module name:** `languages.clojure`
**Key options:** `enable`, `lsp.enable`, `lsp.package`

#### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `cljfmt` | Formatting |
| `clj-kondo` | Linting/static analysis |
| `nvd-clojure` | Vulnerability scanning (CI) |

---

## 2. .NET Ecosystems

Covers C# and F#. Both use the same package manager (NuGet) and build system (MSBuild/dotnet CLI).

### Package Managers / Build Tools

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **NuGet** (official, via dotnet CLI) | `packages.lock.json` (opt-in) | `nuget.config`, `.csproj`/`.fsproj` |
| **MSBuild** | N/A (build system) | `.csproj`, `Directory.Build.props` |
| **Paket** (alternative) | `paket.lock` | `paket.dependencies` |

### Security Hardening Configs

**nuget.config — Hardened:**
```xml
<?xml version="1.0" encoding="utf-8"?>
<configuration>
  <!-- Require signed packages -->
  <config>
    <add key="signatureValidationMode" value="require" />
  </config>

  <!-- Define trusted signers -->
  <trustedSigners>
    <repository name="nuget.org" serviceIndex="https://api.nuget.org/v3/index.json">
      <certificate fingerprint="0E5F38F57DC1BCC806D8494F4F90FBCEDD988B46760709CBEEC6F4219AA6157D"
                   hashAlgorithm="SHA256"
                   allowUntrustedRoot="false" />
    </repository>
  </trustedSigners>

  <!-- Package sources — HTTPS only -->
  <packageSources>
    <clear />
    <add key="nuget.org" value="https://api.nuget.org/v3/index.json" protocolVersion="3" />
  </packageSources>

  <!-- Audit settings -->
  <config>
    <add key="audit-level" value="moderate" />
    <add key="audit-mode" value="all" />
  </config>
</configuration>
```

**Directory.Build.props — Enable lockfile + Central Package Management:**
```xml
<Project>
  <PropertyGroup>
    <!-- Enable lockfile generation -->
    <RestorePackagesWithLockFile>true</RestorePackagesWithLockFile>
    <!-- CI: locked mode prevents resolution drift -->
    <RestoreLockedMode Condition="'$(CI)' != ''">true</RestoreLockedMode>
    <!-- Enable Central Package Management -->
    <ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>
  </PropertyGroup>
</Project>
```

**Directory.Packages.props — Central version management:**
```xml
<Project>
  <ItemGroup>
    <!-- All package versions pinned centrally -->
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageVersion Include="Microsoft.Extensions.Logging" Version="8.0.1" />
  </ItemGroup>
</Project>
```

**CI commands:**
```bash
# Locked restore (fails if lockfile is stale)
dotnet restore --locked-mode
# Built-in vulnerability audit
dotnet list package --vulnerable --include-transitive
```

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **dotnet list package --vulnerable** | Built-in | Free, ships with SDK |
| **Snyk** | Freemium | Full .NET/NuGet support |
| **OSV-Scanner** | Free | Parses packages.lock.json |
| **Socket.dev** | SaaS | NuGet ecosystem support |
| **NuGet Audit** | Built-in | Auto-runs during restore (SDK 8.0+) |

### devenv.sh Module

```nix
languages.dotnet = {
  enable = true;
  package = pkgs.dotnet-sdk_8;    # Pin SDK version
  lsp.enable = true;               # csharp-ls
  lsp.package = pkgs.csharp-ls;
};
```

**Module name:** `languages.dotnet`
**Key options:** `enable`, `package`, `lsp.enable`, `lsp.package`

F# uses the same module — there is no separate `languages.fsharp`. F# is included in the .NET SDK.

### Pre-Commit Hooks

| Hook | Purpose | Tool |
|------|---------|------|
| `dotnet format` | Formatting | Built-in dotnet CLI |
| `CSharpier` | Opinionated formatting | `dotnet csharpier` |
| `Roslynator` | Static analysis | Roslyn analyzers via `.editorconfig` |
| `SonarAnalyzer` | Security + quality | NuGet analyzer package |
| `Husky.NET` | Hook runner | .NET-native alternative to husky |
| `ripsecrets` | Secrets detection | Language-agnostic |

**.editorconfig — Enforce analyzer severity:**
```ini
[*.cs]
# Roslyn analyzer rules
dotnet_diagnostic.CA2100.severity = error   # SQL injection
dotnet_diagnostic.CA2301.severity = error   # Insecure deserialization
dotnet_diagnostic.CA5350.severity = error   # Weak crypto
dotnet_diagnostic.CA5351.severity = error   # Broken crypto
dotnet_diagnostic.CA5359.severity = error   # Certificate validation
```

---

## 3. JavaScript/TypeScript

Already partially covered in the existing plan. This section provides complete coverage.

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **npm** | `package-lock.json` | `.npmrc` |
| **pnpm** | `pnpm-lock.yaml` | `pnpm-workspace.yaml`, `.npmrc` |
| **yarn** (Berry/v4) | `yarn.lock` | `.yarnrc.yml` |
| **bun** | `bun.lock` | `bunfig.toml` |

### Security Hardening Configs

**.npmrc — Hardened (npm 11+):**
```ini
# Age-gating: block packages published < 3 days ago
# Requires npm 11+ (use time[dist-tags.latest] for version publish time)
min-release-age=3

# Block install scripts (primary supply chain vector)
ignore-scripts=true

# Pin exact versions
save-exact=true

# Enforce lockfile in CI
# CI command: npm ci (inherently frozen)

# Audit settings
audit=true
audit-level=moderate
```

**pnpm-workspace.yaml — Hardened (pnpm 10+):**
```yaml
onlyBuiltDependencies:
  # Explicit allowlist of packages permitted to run install scripts
  - esbuild
  - sharp
strictDepBuilds: true
# Age-gating (value in MINUTES — not milliseconds)
minimumReleaseAge: 4320  # 3 days
trustPolicy: no-downgrade
blockExoticSubdeps: true
```

**.yarnrc.yml — Hardened:**
```yaml
enableImmutableInstalls: true
enableHardenedMode: true
enableScripts: false
npmMinimalAgeGate: 7d
```

**bunfig.toml — Hardened:**
```toml
[install]
minimumReleaseAge = "7d"
```

**CI commands:**
```bash
# npm: frozen install
npm ci
# pnpm: frozen lockfile
pnpm install --frozen-lockfile
# yarn: immutable
yarn install --immutable
# bun: frozen lockfile
bun install --frozen-lockfile
```

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **npm audit** | Built-in | Free, ships with npm |
| **Socket.dev** | SaaS+CLI | Behavioral analysis, npm/yarn/pnpm |
| **OSV-Scanner** | Free CLI | npm, yarn, pnpm lockfile parsing |
| **Snyk** | Freemium | Full JS/TS ecosystem |
| **Renovate/Dependabot** | Free | Auto-PR with age-gating support |

### devenv.sh Module

```nix
languages.javascript = {
  enable = true;
  package = pkgs.nodejs_22;
  npm.enable = true;
  npm.install.enable = true;        # Auto-run npm install
  pnpm.enable = true;
  pnpm.install.enable = true;
  yarn.enable = true;
  bun.enable = true;
  corepack.enable = true;           # Node.js corepack for package manager version pinning
  lsp.enable = true;
};

# TypeScript extends JavaScript
languages.typescript = {
  enable = true;
  lsp.enable = true;
};
```

**Module names:** `languages.javascript`, `languages.typescript`
**Key options (JS):** `enable`, `package`, `npm.enable`, `npm.install.enable`, `pnpm.enable`, `pnpm.install.enable`, `yarn.enable`, `bun.enable`, `corepack.enable`, `lsp.enable`
**Key options (TS):** `enable`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `prettier` | Formatting |
| `eslint` | Linting + security rules (eslint-plugin-security) |
| `tsc --noEmit` | Type checking |
| `ripsecrets` | Secrets detection |

---

## 4. PHP

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **Composer** (dominant) | `composer.lock` | `composer.json`, `auth.json` |
| **PEAR/PECL** (legacy extensions) | N/A | N/A |

### Security Hardening Configs

**composer.json — Hardened (Composer 2.9+):**
```json
{
    "config": {
        "lock": true,
        "preferred-install": "dist",
        "secure-http": true,
        "audit": {
            "block-insecure": true,
            "block-abandoned": "report",
            "ignore-severity": ["info"]
        },
        "allow-plugins": {
            "dealerdirect/phpcodesniffer-composer-installer": true
        }
    }
}
```

**CI commands:**
```bash
# Locked install (fails if lockfile stale)
composer install --no-dev --no-scripts --no-interaction
# Vulnerability audit
composer audit
# Verify lockfile integrity
composer validate --strict
```

Key: Composer 2.9 (Nov 2025) introduced automatic security blocking — vulnerable packages are blocked during dependency resolution by default.

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **composer audit** | Built-in | Free, ships with Composer 2.4+ |
| **Snyk** | Freemium | Composer support |
| **OSV-Scanner** | Free CLI | Parses composer.lock |
| **Socket.dev** | SaaS | Limited PHP support |
| **Roave Security Advisories** | Composer plugin | Blocks known-vulnerable installs |

### devenv.sh Module

```nix
languages.php = {
  enable = true;
  package = pkgs.php83;            # Pin PHP version
  # PHP extensions via packages
  packages = with pkgs.php83Extensions; [
    xdebug
    redis
  ];
  lsp.enable = true;
};
```

**Module name:** `languages.php`
**Key options:** `enable`, `package`, `packages` (extensions), `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `php-cs-fixer` | Formatting |
| `phpstan` | Static analysis (levels 0-9) |
| `psalm` | Static analysis with security focus |
| `phpcs` | Code standards (PSR-12) |
| `composer audit` | Vulnerability check (CI) |
| `ripsecrets` | Secrets detection |

---

## 5. Ruby

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **Bundler** (dominant) | `Gemfile.lock` | `Gemfile`, `.bundle/config` |
| **RubyGems** (underlying) | N/A | `~/.gemrc` |

### Security Hardening Configs

**Gemfile — Enforce HTTPS sources:**
```ruby
source "https://rubygems.org"
# Never use http:// or git:// sources
```

**.bundle/config — Hardened:**
```yaml
---
BUNDLE_FROZEN: "true"        # Equivalent to --frozen in CI
BUNDLE_DISABLE_EXEC_LOAD: "true"
BUNDLE_JOBS: "4"
```

**.gemrc — Hardened:**
```yaml
---
:sources:
  - https://rubygems.org
gem: --no-document
```

**CI commands:**
```bash
# Frozen install (fails if lockfile stale)
bundle install --frozen
# Vulnerability audit
bundle audit check --update
# Check for insecure sources
bundle audit check --gemfile-lock Gemfile.lock
```

**.bundler-audit.yml:**
```yaml
---
ignore:
  # Only add advisories you've manually verified as non-applicable
  # - GHSA-xxxx-yyyy-zzzz
```

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **bundler-audit** | Free CLI | Checks Gemfile.lock against ruby-advisory-db |
| **Snyk** | Freemium | RubyGems support |
| **OSV-Scanner** | Free CLI | Parses Gemfile.lock |
| **Socket.dev** | SaaS | RubyGems ecosystem |
| **brakeman** | Free CLI | Rails-specific security scanner |

### devenv.sh Module

```nix
languages.ruby = {
  enable = true;
  package = pkgs.ruby_3_3;
  bundler.enable = true;
  # bundler.package = pkgs.bundler;
  lsp.enable = true;                # Solargraph
};
```

**Module name:** `languages.ruby`
**Key options:** `enable`, `package`, `bundler.enable`, `bundler.package`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `rubocop` | Linting + formatting |
| `bundler-audit` | Vulnerability scanning |
| `brakeman` | Rails security (if applicable) |
| `ripsecrets` | Secrets detection |

---

## 6. Swift & Objective-C

### Package Managers

| Tool | Lockfile | Config File | Status |
|------|----------|-------------|--------|
| **Swift Package Manager** (SPM) | `Package.resolved` | `Package.swift` | Recommended for new projects |
| **CocoaPods** | `Podfile.lock` | `Podfile` | Legacy, maintenance mode |
| **Carthage** | `Cartfile.resolved` | `Cartfile` | Largely abandoned |

### Security Hardening Configs

**Swift Package Manager — Trust policy (SE-0378, SE-0391):**

SPM supports package signing and Trust-On-First-Use (TOFU) for source archives and manifests. Configure in `~/.swiftpm/configuration/registries.json`:

```json
{
  "security": {
    "default": {
      "signing": {
        "onUnsigned": "warn",
        "onUntrustedCertificate": "error",
        "validationChecks": {
          "certificateExpiration": "enabled",
          "certificateRevocation": "strict"
        }
      }
    }
  }
}
```

**Package.swift — Pin versions explicitly:**
```swift
dependencies: [
    .package(url: "https://github.com/example/lib.git", exact: "1.2.3"),
]
```

**CI commands:**
```bash
# Resolve and verify
swift package resolve
# Check for updates (manual review)
swift package show-dependencies
```

**CocoaPods (legacy) — Lockfile enforcement:**
```bash
# CI: install from lockfile only
pod install --deployment
```

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **Snyk** | Freemium | Swift/CocoaPods support |
| **OSV-Scanner** | Free CLI | Limited Swift support |
| **swift-dependency-audit** | Community | Emerging tool |

### devenv.sh Module

```nix
languages.swift = {
  enable = true;
  package = pkgs.swift;
  lsp.enable = true;               # sourcekit-lsp
};
```

**Module name:** `languages.swift`
**Key options:** `enable`, `package`, `lsp.enable`, `lsp.package`

No dedicated Objective-C module — use `languages.c` or `languages.cplusplus` with Xcode toolchain packages.

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `swiftlint` | Linting |
| `swiftformat` | Formatting |
| `ripsecrets` | Secrets detection |

---

## 7. Dart/Flutter

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **pub** (official) | `pubspec.lock` | `pubspec.yaml` |

### Security Hardening Configs

**pubspec.lock enforcement:**

Pub maintains SHA256 content hashes in `pubspec.lock`. If a published package's hash differs from the lockfile, pub warns and blocks.

```bash
# CI: ensure lockfile is committed and up to date
dart pub get --enforce-lockfile
# Flutter equivalent
flutter pub get --enforce-lockfile
```

**pubspec.yaml — Best practices:**
```yaml
# Pin exact versions for applications
dependencies:
  http: 1.2.0          # exact, not ^1.2.0

# Use verified publishers — look for blue badge on pub.dev
# Prefer packages from verified publishers (domain-verified identity)
```

pub.dev verified publishers use DNS domain verification (no GPG signing). TOFU is implicit via content hashes in pubspec.lock.

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **Snyk** | Freemium | Dart/Flutter support |
| **OSV-Scanner** | Free CLI | Parses pubspec.lock |
| **dart pub outdated** | Built-in | Shows outdated/vulnerable deps |

### devenv.sh Module

```nix
languages.dart = {
  enable = true;
  package = pkgs.dart;
  lsp.enable = true;
};
# Flutter requires separate package
packages = [ pkgs.flutter ];
```

**Module name:** `languages.dart`
**Key options:** `enable`, `package`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `dart format` | Formatting |
| `dart analyze` | Static analysis |
| `dart pub outdated` | Dependency freshness (CI) |
| `ripsecrets` | Secrets detection |

---

## 8. Python

Already covered in existing plan. Summary for completeness:

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **pip** | `requirements.txt` (manual) | `pip.conf` |
| **uv** (Astral) | `uv.lock` | `pyproject.toml` |
| **poetry** | `poetry.lock` | `pyproject.toml` |
| **pipenv** | `Pipfile.lock` | `Pipfile` |
| **conda** | `conda-lock.yml` (via conda-lock) | `environment.yml` |

### Security Hardening Configs

**pip.conf — Hardened:**
```ini
[global]
require-hashes = true
only-binary = :all:
```

**uv — Age-gating:**
```bash
uv pip install --exclude-newer 2026-05-05  # 7-day age gate
```

**CI commands:**
```bash
# pip: hash-verified install
pip install --require-hashes -r requirements.txt
# uv: locked install
uv sync --locked
# poetry: locked install
poetry install --no-interaction
```

### Vulnerability Scanning

| Tool | Type |
|------|------|
| **pip-audit** | Free CLI (PyPI advisory DB) |
| **Safety** | Freemium CLI |
| **Snyk** | Freemium |
| **OSV-Scanner** | Free CLI |
| **Socket.dev** | SaaS (PyPI support) |
| **Bandit** | SAST for Python code |

### devenv.sh Module

**Module name:** `languages.python`
**Key options:** `enable`, `package`, `version`, `poetry.enable`, `poetry.activate.enable`, `uv.enable`, `uv.sync.enable`, `venv.enable`, `lsp.enable`

### Pre-Commit Hooks

`ruff` (linting+formatting), `mypy` (type checking), `bandit` (security SAST), `ripsecrets`

---

## 9. Go

Already covered in existing plan. Summary for completeness:

### Security Hardening

```bash
# Lockfile enforcement
GOFLAGS=-mod=readonly
# Verify module checksums
go mod verify
# Vulnerability scanning
govulncheck ./...
```

### devenv.sh Module

**Module name:** `languages.go`
**Key options:** `enable`, `package`

### Pre-Commit Hooks

`gofmt`, `go vet`, `staticcheck`, `govulncheck`, `ripsecrets`

---

## 10. Rust

Already covered in existing plan. Summary for completeness:

### Security Hardening

**Cargo config (.cargo/config.toml):**
```toml
[net]
git-fetch-with-cli = true

[install]
# No native age-gating in Cargo
```

```bash
# Lockfile enforcement
cargo build --locked
# Vulnerability scanning
cargo audit
```

### devenv.sh Module

**Module name:** `languages.rust`
**Key options:** `enable`, `channel` (stable/nightly), `components`, `targets`, `lsp.enable`

### Pre-Commit Hooks

`rustfmt`, `clippy`, `cargo audit`, `ripsecrets`

---

## 11. Elixir/Erlang

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **Mix** (official, built into Elixir) | `mix.lock` | `mix.exs` |
| **Hex** (package registry) | Content hashes in mix.lock | `~/.hex/hex.config` |
| **rebar3** (Erlang) | `rebar.lock` | `rebar.config` |

### Security Hardening Configs

**mix.lock** contains content hashes from Hex.pm by default. Always commit it.

```bash
# CI: ensure lockfile is respected
mix deps.get --check-locked
# Retired package audit
mix hex.audit
# Vulnerability audit
mix deps.audit  # requires mix_audit
```

**mix.exs — Add mix_audit dependency:**
```elixir
defp deps do
  [
    {:mix_audit, "~> 2.1", only: [:dev, :test], runtime: false}
  ]
end
```

Hex 2.4 added OAuth device flow + 2FA for publishing, significantly improving publisher authentication.

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **mix hex.audit** | Built-in | Checks for retired packages |
| **mix_audit** | Free library | GitHub advisory DB |
| **Snyk** | Freemium | Hex/Elixir support |
| **OSV-Scanner** | Free CLI | Parses mix.lock |
| **Sobelow** | Free CLI | Phoenix-specific security scanner |

### devenv.sh Module

```nix
languages.elixir = {
  enable = true;
  package = pkgs.elixir_1_17;
  lsp.enable = true;
};
```

**Module name:** `languages.elixir`
**Key options:** `enable`, `package`, `lsp.enable`, `lsp.package`

Erlang is available via `languages.erlang` with similar options.

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `mix format` | Formatting |
| `credo` | Static analysis |
| `dialyxir` | Type checking (Dialyzer) |
| `sobelow` | Security (Phoenix apps) |
| `mix deps.audit` | Vulnerability scanning |
| `ripsecrets` | Secrets detection |

---

## 12. Haskell

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **Cabal** | `cabal.project.freeze` (not a true lockfile — see limitations) | `cabal.project`, `*.cabal` |
| **Stack** | `stack.yaml.lock` | `stack.yaml` |

### Security Hardening Configs

**Cabal limitations:** `cabal freeze` does NOT produce a true lockfile — it pins direct dependencies but may not be exhaustive for all transitive dependencies. This is a known limitation actively discussed in the Haskell community (Feb 2025).

**cabal.project — Hardened:**
```
-- Pin index state for reproducibility
index-state: hackage.haskell.org 2026-05-01T00:00:00Z

-- Require hashes when available
-- (limited support — Hackage doesn't universally provide content hashes)
```

**Stack — Better lockfile support:**
```bash
# stack.yaml.lock is auto-generated and comprehensive
# CI: use locked resolver
stack build --locked
```

### Vulnerability Scanning

Limited ecosystem-specific tooling. Use:

| Tool | Type | Notes |
|------|------|-------|
| **OSV-Scanner** | Free CLI | Limited Hackage support |
| **Snyk** | Freemium | No direct Haskell support |
| **Manual audit** | N/A | Hackage security advisories list |

### devenv.sh Module

```nix
languages.haskell = {
  enable = true;
  package = pkgs.ghc;
  stack.enable = true;
  lsp.enable = true;    # haskell-language-server
};
```

**Module name:** `languages.haskell`
**Key options:** `enable`, `package`, `stack.enable`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `ormolu` or `fourmolu` | Formatting |
| `hlint` | Linting |
| `stan` | Static analysis |
| `ripsecrets` | Secrets detection |

---

## 13. Perl

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **CPAN** | N/A | `CPAN/Config.pm` |
| **cpanm** (cpanminus) | N/A | N/A (CLI flags) |
| **Carton** (Bundler for Perl) | `cpanfile.snapshot` | `cpanfile` |

### Security Hardening Configs

**cpanfile — Pin versions:**
```perl
requires 'Mojolicious', '== 9.35';
requires 'DBI', '== 1.643';
```

**Carton — Lockfile enforcement:**
```bash
# Install from snapshot (like --frozen)
carton install --deployment
# Bundle for air-gapped deployments
carton bundle
```

No native signing or age-gating in CPAN/Carton. Security relies on:
- Pinning exact versions in `cpanfile`
- Committing `cpanfile.snapshot`
- Using `--deployment` flag in CI

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **CPAN::Audit** | Free CLI | `cpan-audit installed` checks installed modules |
| **OSV-Scanner** | Free CLI | Limited CPAN support |
| **Snyk** | No direct Perl support | — |

### devenv.sh Module

```nix
languages.perl = {
  enable = true;
  # package = pkgs.perl;
  lsp.enable = true;
};
```

**Module name:** `languages.perl`
**Key options:** `enable`, `package`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `perltidy` | Formatting |
| `perlcritic` | Static analysis |
| `cpan-audit` | Vulnerability scanning (CI) |
| `ripsecrets` | Secrets detection |

---

## 14. R

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **renv** (dominant) | `renv.lock` | `.Rprofile`, `renv/settings.json` |
| **CRAN** (repository) | N/A | `Rprofile.site` |
| **pak** (modern alternative) | `pkg.lock` | N/A |

### Security Hardening Configs

**renv — Lockfile enforcement:**
```r
# Restore from lockfile exactly
renv::restore()
# Validate lockfile schema
renv::lockfile_validate()
# Check for vulnerabilities (requires Posit Package Manager)
renv::vulns()
```

**.Rprofile — Hardened:**
```r
# Force HTTPS for CRAN
options(repos = c(CRAN = "https://cloud.r-project.org/"))
# Disable sourcing from .Rprofile in untrusted directories
# (handled by renv's project-level isolation)
```

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **renv::vulns()** | Built-in | Requires Posit Package Manager |
| **oysteR** | CRAN package | Scans renv.lock against OSS Index |
| **rosv** | CRAN package | OSV database for CRAN packages |
| **OSV-Scanner** | Free CLI | R/CRAN ecosystem support |

### devenv.sh Module

```nix
languages.r = {
  enable = true;
  package = pkgs.R;
  lsp.enable = true;
};
```

**Module name:** `languages.r`
**Key options:** `enable`, `package`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `styler` | Formatting |
| `lintr` | Linting |
| `oysteR::audit_renv_lock()` | Vulnerability scanning (CI) |
| `ripsecrets` | Secrets detection |

---

## 15. Lua

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **LuaRocks** | No native lockfile (basic version pinning) | `.luarocks/config-5.4.lua` |
| **Lux** (newer alternative) | `lux.lock` (with source + rockspec hashes) | `lux.toml` |

### Security Hardening Configs

LuaRocks has **no package signing** and limited security controls. After a 2019 security incident, package verification was noted as high priority but remains unimplemented.

**LuaRocks — Best practices:**
```bash
# Pin exact versions in rockspec
luarocks install lpeg 1.1.0-1
# Use --local to avoid system-wide installs
luarocks install --local lpeg 1.1.0-1
```

**Lux (recommended for new projects):**
- `lux.lock` stores source and rockspec hashes
- NAR hashes for git dependencies (compatible with Nix fixed-output derivations)
- Announced April 2025

### Vulnerability Scanning

No Lua-specific vulnerability scanning tools. Use:

| Tool | Type | Notes |
|------|------|-------|
| **Manual audit** | N/A | Check luarocks.org advisories |
| **OSV-Scanner** | Free CLI | No LuaRocks support currently |
| **General SAST** | Semgrep | Custom rules for Lua patterns |

### devenv.sh Module

```nix
languages.lua = {
  enable = true;
  # package = pkgs.lua5_4;
  lsp.enable = true;     # lua-language-server
};
```

**Module name:** `languages.lua`
**Key options:** `enable`, `package`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `stylua` | Formatting |
| `luacheck` | Linting/static analysis |
| `ripsecrets` | Secrets detection |

---

## 16. Zig

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **Zig build system** (built-in) | Hashes embedded in `build.zig.zon` | `build.zig.zon` |

### Security Hardening Configs

Zig's package manager has strong integrity verification by design:

- Every dependency requires a SHA256 hash in `build.zig.zon`
- **Hash is the source of truth** — the URL is just a mirror. If content changes, hash mismatch fails the build
- No mutable references (unlike npm tags or Docker :latest)

**build.zig.zon example:**
```zig
.{
    .name = "myproject",
    .version = "0.1.0",
    .dependencies = .{
        .@"dep-name" = .{
            .url = "https://github.com/example/dep/archive/v1.0.0.tar.gz",
            .hash = "1220abc123...",  // SHA256 — mandatory
        },
    },
}
```

No age-gating or signing — security relies entirely on content-addressed hashing. This is architecturally similar to Nix's fixed-output derivations.

### Vulnerability Scanning

No Zig-specific vulnerability scanning tools exist. Use:

| Tool | Type | Notes |
|------|------|-------|
| **Manual audit** | N/A | Check dependency source repos |
| **General SAST** | Semgrep | Custom rules |

### devenv.sh Module

```nix
languages.zig = {
  enable = true;
  package = pkgs.zig;
  lsp.enable = true;    # zls
};
```

**Module name:** `languages.zig`
**Key options:** `enable`, `package`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `zig fmt` | Formatting (built-in) |
| `ripsecrets` | Secrets detection |

---

## 17. C/C++

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **Conan** (dominant) | `conan.lock` | `conanfile.py` / `conanfile.txt`, `profiles/` |
| **vcpkg** (Microsoft) | `vcpkg.json` manifest (baseline pinning) | `vcpkg.json`, `vcpkg-configuration.json` |
| **CMake FetchContent** | No lockfile | `CMakeLists.txt` |
| **Meson WrapDB** | `.wrap` files with hashes | `subprojects/*.wrap` |
| **Bazel** | `MODULE.bazel.lock` | `MODULE.bazel` |

### Security Hardening Configs

**Conan — Lockfile enforcement:**
```bash
# Generate lockfile
conan lock create .
# CI: locked build
conan install . --lockfile=conan.lock
# Configuration package management (reproducible config)
conan config install-pkg myorg/config@1.0
```

**Conan profile — Hardened:**
```ini
[settings]
os=Linux
compiler=gcc
compiler.version=13
build_type=Release

[conf]
# Require lockfile in CI
tools.graph:lockfile_policy=require
```

**vcpkg — Baseline pinning:**
```json
{
    "dependencies": ["fmt", "spdlog"],
    "builtin-baseline": "a1b2c3d4...",
    "overrides": [
        { "name": "fmt", "version": "10.2.1" }
    ]
}
```

**Meson — Hash-verified wraps:**
```ini
# subprojects/zlib.wrap
[wrap-file]
directory = zlib-1.3.1
source_url = https://zlib.net/zlib-1.3.1.tar.gz
source_hash = 9a93b2b7dfdac77ceba5a558a580e74667dd6fede4585b91eefb60f03b72df23
```

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **OSV-Scanner** | Free CLI | C/C++ support (via lockfiles) |
| **Snyk** | Freemium | C/C++ support |
| **SonarQube** | Freemium | Beta Conan/vcpkg support (2025) |
| **cppcheck** | Free CLI | Static analysis (bugs, not CVEs) |
| **Coverity** | Commercial | Deep static analysis |

### devenv.sh Module

```nix
languages.c = {
  enable = true;
  lsp.enable = true;    # clangd
};
languages.cplusplus = {
  enable = true;
  lsp.enable = true;    # clangd
};
# Add build tools via packages
packages = with pkgs; [ cmake meson ninja conan_2 vcpkg ];
```

**Module names:** `languages.c`, `languages.cplusplus`
**Key options:** `enable`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `clang-format` | Formatting |
| `clang-tidy` | Static analysis + security checks |
| `cppcheck` | Bug detection |
| `include-what-you-use` | Header hygiene |
| `ripsecrets` | Secrets detection |

---

## 18. Bash/Shell

### Package Managers

No package manager. Shell scripts depend on system binaries.

### Security Hardening / Scanning

**shellcheck** is the primary security tool — catches quoting bugs, injection risks, undefined variables, and common POSIX pitfalls.

**shfmt** enforces consistent formatting.

### devenv.sh Module

```nix
languages.shell = {
  enable = true;
  lsp.enable = true;    # bash-language-server
};
packages = [ pkgs.shellcheck pkgs.shfmt ];
```

**Module name:** `languages.shell`
**Key options:** `enable`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose | Repo |
|------|---------|------|
| `shellcheck` | Linting/security | `https://github.com/shellcheck-py/shellcheck-py` |
| `shfmt` | Formatting | `https://github.com/scop/pre-commit-shfmt` |
| `ripsecrets` | Secrets detection | `https://github.com/sirwart/ripsecrets` |

**.pre-commit-config.yaml snippet:**
```yaml
- repo: https://github.com/shellcheck-py/shellcheck-py
  rev: v0.10.0.1
  hooks:
    - id: shellcheck
      args: ["--severity=warning"]
- repo: https://github.com/scop/pre-commit-shfmt
  rev: v3.8.0-1
  hooks:
    - id: shfmt
      args: ["-i", "2", "-ci"]
```

---

## 19. PowerShell

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **PSGallery** (PowerShell Gallery) | No lockfile | `Install-Module` / `Save-Module` |
| **NuGet** (underlying) | `packages.lock.json` | `nuget.config` |

### Security Hardening Configs

**PowerShell execution policy:**
```powershell
# Require signed scripts
Set-ExecutionPolicy AllSigned -Scope CurrentUser
# Or RemoteSigned for locally-authored scripts
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
```

**Module installation verification:**
```powershell
# Only install from trusted repositories
Set-PSRepository -Name PSGallery -InstallationPolicy Trusted
# Verify module signatures
Install-Module -Name ModuleName -RequiredVersion 1.2.3
# Skip publisher check only when explicitly needed (avoid by default)
```

**Pin module versions in requirements file:**
```
# requirements.psd1
@{
    'Pester' = @{ Version = '5.6.1'; Repository = 'PSGallery' }
    'PSScriptAnalyzer' = @{ Version = '1.22.0'; Repository = 'PSGallery' }
}
```

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **PSScriptAnalyzer** | Free CLI | Static analysis, security rules |
| **ScriptBlockLogging** | Built-in | Runtime script auditing |
| **Snyk** | No direct PS support | — |

### devenv.sh Module

No dedicated PowerShell module. Add via packages:

```nix
packages = [ pkgs.powershell ];
```

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `PSScriptAnalyzer` | Linting + security rules |
| `ripsecrets` | Secrets detection |

PSScriptAnalyzer includes built-in security rules — all packages published to PSGallery must pass its checks.

---

## 20. Terraform

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **Terraform Registry** | `.terraform.lock.hcl` | `main.tf`, `versions.tf`, `.terraformrc` |
| **OpenTofu** (fork) | `.terraform.lock.hcl` | Same as Terraform |

### Security Hardening Configs

**versions.tf — Pin provider versions:**
```hcl
terraform {
  required_version = ">= 1.9.0, < 2.0.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "= 5.82.2"    # Exact pin
    }
    random = {
      source  = "hashicorp/random"
      version = "= 3.6.3"
    }
  }
}
```

**.terraform.lock.hcl** — Auto-generated, contains provider hashes (h1: and zh: schemes). Always commit this file.

```bash
# CI: init with lockfile enforcement
terraform init -lockfile=readonly
# Update lockfile for multiple platforms
terraform providers lock -platform=linux_amd64 -platform=darwin_arm64
```

**.terraformrc — Security settings:**
```hcl
# Disable automatic provider installation from registry
# (use explicit init instead)
disable_checkpoint = true

# Use private registry mirror
provider_installation {
  network_mirror {
    url = "https://registry.internal.company.com/v1/providers/"
  }
  direct {
    exclude = ["registry.terraform.io/*/*"]
  }
}
```

Terraform providers are signed with GPG keys by HashiCorp. The lock file records hash digests for all platforms.

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **tfsec** | Free CLI | Terraform-specific security scanner (now part of Trivy) |
| **Checkov** | Free CLI | Policy-as-code for Terraform, CloudFormation, K8s |
| **TFLint** | Free CLI | Linting + best practices |
| **Trivy** | Free CLI | IaC scanning mode (includes tfsec rules) |
| **Snyk IaC** | Freemium | Terraform scanning |
| **OPA/Conftest** | Free CLI | Custom policy enforcement |

### devenv.sh Module

```nix
languages.terraform = {
  enable = true;
  version = "1.9.8";     # Auto-selects package via nixpkgs-terraform
  lsp.enable = true;      # terraform-ls
};
# Or use OpenTofu
languages.opentofu = {
  enable = true;
};
```

**Module names:** `languages.terraform`, `languages.opentofu`
**Key options:** `enable`, `version`, `package`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose | Repo |
|------|---------|------|
| `terraform fmt` | Formatting | `https://github.com/antonbabenko/pre-commit-terraform` |
| `terraform validate` | Syntax validation | Same |
| `tfsec` | Security scanning | Same |
| `tflint` | Linting | Same |
| `checkov` | Policy compliance | Same |
| `infracost_breakdown` | Cost estimation | Same |
| `ripsecrets` | Secrets detection | — |

The `pre-commit-terraform` meta-repo provides hooks for all major Terraform tools.

---

## 21. Pulumi

### Package Managers

Pulumi uses **language-specific package managers** (npm for TS, pip for Python, Go modules for Go, NuGet for .NET). No Pulumi-specific package manager.

### Security Hardening

**Pulumi CrossGuard — Policy-as-Code:**
```typescript
// policy-pack/index.ts
import * as policy from "@pulumi/policy";

new policy.PolicyPack("security-policies", {
    policies: [
        {
            name: "no-public-s3",
            description: "S3 buckets must not be public",
            enforcementLevel: "mandatory",
            validateResource: (args, reportViolation) => {
                if (args.type === "aws:s3/bucket:Bucket") {
                    if (args.props.acl === "public-read") {
                        reportViolation("S3 buckets must not be public");
                    }
                }
            },
        },
    ],
});
```

**Pre-built compliance packs:** CIS, PCI DSS, ISO 27001, SOC 2 — available for AWS, Azure, GCP, K8s.

```bash
# Run policy check during preview
pulumi preview --policy-pack ./policy-pack
# Enforce on every deployment
pulumi up --policy-pack ./policy-pack
```

The security hardening for Pulumi's dependencies follows the language ecosystem being used (see relevant section above). Apply npm/pip/Go/NuGet hardening configs to Pulumi projects accordingly.

### devenv.sh Module

No dedicated Pulumi module. Add via packages:

```nix
packages = [ pkgs.pulumi pkgs.pulumiPackages.pulumi-aws ];
# Plus the language runtime module (e.g., languages.typescript for TS)
```

### Pre-Commit Hooks

Use the hooks from the underlying language ecosystem, plus:

| Hook | Purpose |
|------|---------|
| `pulumi preview` | Drift detection (CI) |
| `checkov` | IaC policy scanning (supports Pulumi) |
| `ripsecrets` | Secrets detection |

---

## 22. Ansible

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **Ansible Galaxy** | `requirements.yml` (version pins) | `ansible.cfg` |
| **Collections** | N/A (no lockfile) | `requirements.yml` |

### Security Hardening Configs

**requirements.yml — Pin versions:**
```yaml
collections:
  - name: community.general
    version: ">=8.0.0,<9.0.0"
  - name: ansible.posix
    version: "1.5.4"         # Exact pin for production
```

**ansible.cfg — Signature verification:**
```ini
[galaxy]
# Require valid GPG signatures on collections
gpg_keyring = ~/.ansible/keyring.gpg
required_valid_signature_count = 1
# Ignore specific signature errors (use sparingly)
# ignore_signature_status_codes = EXPKEYSIG

[defaults]
# Disable cowsay, enable logging
log_path = /var/log/ansible.log
```

**Collection signature verification:**
```bash
# Import trusted signing keys
gpg --import ansible-signing-key.asc
# Install with signature verification
ansible-galaxy collection install community.general \
  --keyring ~/.ansible/keyring.gpg \
  --required-valid-signature-count 1
# Verify installed collections
ansible-galaxy collection verify community.general \
  --keyring ~/.ansible/keyring.gpg
```

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **ansible-lint** | Free CLI | Best practices + security rules |
| **Checkov** | Free CLI | Ansible playbook scanning |
| **Snyk IaC** | Freemium | Ansible support |
| **KICS** | Free CLI | IaC scanning (Ansible support) |

### devenv.sh Module

```nix
languages.ansible = {
  enable = true;
  package = pkgs.ansible;
  lsp.enable = true;
};
```

**Module name:** `languages.ansible`
**Key options:** `enable`, `package`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose | Repo |
|------|---------|------|
| `ansible-lint` | Linting + security | `https://github.com/ansible/ansible-lint` |
| `yamllint` | YAML syntax | `https://github.com/adrienverge/yamllint` |
| `ripsecrets` | Secrets detection | — |

---

## 23. Helm

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **Helm** | `Chart.lock` | `Chart.yaml` |
| **OCI registries** | Digest-pinned references | N/A |

### Security Hardening Configs

**Chart.yaml — Pin dependency versions:**
```yaml
dependencies:
  - name: postgresql
    version: "15.5.10"        # Exact version pin
    repository: "oci://registry-1.docker.io/bitnamicharts"
```

**Provenance and signing:**
```bash
# GPG-based provenance (traditional)
helm package --sign --key 'mykey' --keyring ~/.gnupg/secring.gpg mychart/
helm verify mychart-0.1.0.tgz

# Cosign/Sigstore (modern, for OCI charts)
cosign sign --key cosign.key oci://registry.example.com/charts/mychart:0.1.0
cosign verify --key cosign.pub oci://registry.example.com/charts/mychart:0.1.0
```

**CI commands:**
```bash
# Update dependencies from lockfile
helm dependency build
# Template rendering for validation
helm template mychart/ --values values.yaml | kubeval --strict
```

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **Checkov** | Free CLI | Helm chart scanning |
| **Trivy** | Free CLI | Helm chart + container scanning |
| **Snyk IaC** | Freemium | Helm support |
| **Pluto** | Free CLI | Detects deprecated K8s APIs |
| **kubeaudit** | Free CLI | K8s manifest security audit |
| **kubeval/kubeconform** | Free CLI | Schema validation |

### devenv.sh Module

```nix
languages.helm = {
  enable = true;
  # plugins = [ "helm-diff" "helm-secrets" ];
  lsp.enable = true;
};
```

**Module name:** `languages.helm`
**Key options:** `enable`, `plugins`, `lsp.enable`, `lsp.package`

### Pre-Commit Hooks

| Hook | Purpose | Repo |
|------|---------|------|
| `helmlint` | Chart linting | `https://github.com/gruntwork-io/pre-commit` |
| `checkov` | Security policies | `https://github.com/bridgecrewio/checkov` |
| `kubeconform` | Schema validation | — |
| `ripsecrets` | Secrets detection | — |

---

## 24. Docker/Containerfiles

### Security Hardening Configs

**Dockerfile — Hardened patterns:**
```dockerfile
# Pin base image by digest (not tag)
FROM node:22-alpine@sha256:a1b2c3d4e5f6...

# Run as non-root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Multi-stage build (minimize attack surface)
FROM node:22-alpine@sha256:a1b2c3d4... AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --ignore-scripts
COPY . .
RUN npm run build

FROM node:22-alpine@sha256:a1b2c3d4... AS runtime
COPY --from=builder /app/dist /app
USER appuser
CMD ["node", "/app/index.js"]
```

**Key principles:**
- Pin by digest (`@sha256:...`), never `:latest`
- Multi-stage builds to minimize final image size
- Non-root USER directive
- COPY specific files, not entire directories
- No secrets in build args or layers

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **Hadolint** | Free CLI | Dockerfile linting (uses ShellCheck for RUN) |
| **Dockle** | Free CLI | Container image best practices |
| **Trivy** | Free CLI | Image vulnerability scanning (verify provenance — March 2026 incident) |
| **Grype** | Free CLI | Image vulnerability scanning (Anchore) |
| **Docker Scout** | Built-in | Docker Desktop integration |
| **Cosign** | Free CLI | Image signature verification |
| **Syft** | Free CLI | SBOM generation |
| **Dive** | Free CLI | Layer analysis (detect secrets in layers) |

### devenv.sh Module

No dedicated Docker module. Container tools added via packages:

```nix
packages = with pkgs; [ docker hadolint dive cosign syft grype ];
```

### Pre-Commit Hooks

| Hook | Purpose | Repo |
|------|---------|------|
| `hadolint` | Dockerfile linting | `https://github.com/hadolint/hadolint` |
| `docker-compose-check` | Compose file validation | `https://github.com/IamTheFij/docker-pre-commit` |
| `ripsecrets` | Secrets detection | — |

**.hadolint.yaml:**
```yaml
ignored:
  - DL3008    # Pin versions in apt-get (noisy but consider enabling)
trustedRegistries:
  - docker.io
  - gcr.io
  - ghcr.io
failure-threshold: warning
```

---

## 25. Bazel

### Package Manager

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **Bzlmod** (MODULE.bazel) | `MODULE.bazel.lock` | `MODULE.bazel`, `.bazelrc` |
| **WORKSPACE** (legacy) | N/A | `WORKSPACE`, `WORKSPACE.bazel` |

### Security Hardening Configs

**MODULE.bazel — Pin versions:**
```starlark
bazel_dep(name = "rules_go", version = "0.50.1")
bazel_dep(name = "rules_python", version = "0.36.0")
bazel_dep(name = "protobuf", version = "29.0")
```

**.bazelrc — Hardened:**
```
# Strict lockfile mode
common --lockfile_mode=update
# In CI:
# common --lockfile_mode=error

# Sandbox all builds
build --spawn_strategy=sandboxed
build --sandbox_default_allow_network=false

# Require hashes for downloads
build --experimental_repository_hash_file=resolved.bzl
```

**MODULE.bazel.lock** contains `registryFileHashes` with SHA256 hashes for all registry files. Commit this file.

BCR (Bazel Central Registry) provenance attestations are upcoming but not yet shipped.

### Vulnerability Scanning

| Tool | Type | Notes |
|------|------|-------|
| **OSV-Scanner** | Free CLI | Parses MODULE.bazel |
| **Snyk** | Freemium | Bazel support |
| **Gazelle** | Free CLI | Dependency management for Bazel |

### devenv.sh Module

No dedicated Bazel module. Add via packages:

```nix
packages = [ pkgs.bazel_7 ];
```

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `buildifier` | Starlark formatting/linting |
| `unused_deps` | Detect unnecessary dependencies |
| `ripsecrets` | Secrets detection |

---

## 26. Nix

Already partially covered in existing plan. Summary:

### Security Hardening

Covered extensively in `research-spikes/devenv-security/`. Key configs:

**nix.conf — Hardened:**
```ini
sandbox = true
sandbox-fallback = false
require-sigs = true
trusted-users = root
filter-syscalls = true
accept-flake-config = false
```

**flake.lock** is auto-generated and contains content hashes for all inputs. Always commit.

### devenv.sh Module

```nix
languages.nix = {
  enable = true;
  lsp.enable = true;     # nil or nixd
};
```

**Module name:** `languages.nix`

### Pre-Commit Hooks

| Hook | Purpose |
|------|---------|
| `statix` | Nix anti-pattern detection |
| `nixfmt` or `alejandra` | Formatting |
| `deadnix` | Dead code detection |
| `flake-checker` | Flake input audit |
| `ripsecrets` | Secrets detection |

---

## 27. WASM

### Package Managers

| Tool | Lockfile | Config File |
|------|----------|-------------|
| **wasm-pack** | Uses Cargo.lock (Rust-based) | `Cargo.toml` |
| **wasi-sdk** | N/A (C/C++ toolchain) | Makefile/CMake |
| **wasmtime/wasmer** | N/A (runtimes) | N/A |

### Security Hardening

WASM/WASI has capability-based security by design — modules are granted explicit capabilities rather than ambient authority.

**wasm-pack:** Uses Rust's Cargo ecosystem — apply all Rust hardening (see section 10). wasm-pack fixed a tar vulnerability (CVE for arbitrary file overwrite/symlink poisoning) in 2025.

**WASI security model:**
- Explicit filesystem capabilities
- No ambient network access
- No ambient environment variable access
- Sandboxed execution by default

Package management for WASM components is still maturing. No established package registry with signing/verification yet.

### devenv.sh Module

No dedicated WASM module. Add via packages:

```nix
packages = with pkgs; [ wasm-pack wasmtime wasmer wasi-sdk ];
# Rust module for wasm-pack projects
languages.rust = {
  enable = true;
  targets = [ "wasm32-unknown-unknown" "wasm32-wasi" ];
};
```

### Pre-Commit Hooks

Use language-specific hooks (Rust/C++ depending on source language). No WASM-specific hooks.

---

## 28. Cross-Ecosystem Tools

These tools work across multiple language ecosystems and should be included in every generated configuration.

### Universal Vulnerability Scanners

| Tool | Languages | Type | CI Integration |
|------|-----------|------|---------------|
| **OSV-Scanner v2** | 15+ ecosystems (npm, pip, Maven, Go, Cargo, NuGet, Composer, Gem, CRAN, pub, Hex, etc.) | Free CLI | GitHub Actions: `google/osv-scanner-action` |
| **Snyk** | 16 languages, IaC, containers | Freemium | GitHub Actions: `snyk/actions` |
| **Socket.dev** | 9 ecosystems (npm, PyPI, Maven, Cargo, Gem, NuGet, Go, Conda, OpenVSX) | SaaS+CLI | GitHub App, MCP server |
| **Trivy** | Multi-language, containers, IaC | Free CLI | GitHub Actions: `aquasecurity/trivy-action` (verify provenance — March 2026 incident) |
| **Grype** | Multi-language, containers | Free CLI | GitHub Actions: `anchore/scan-action` |
| **Semgrep** | 30+ languages (SAST, not SCA) | Freemium | GitHub Actions: `semgrep/semgrep-action` |

### Universal Pre-Commit Hooks (Always Enable)

| Hook | Purpose | Repo |
|------|---------|------|
| **ripsecrets** | Secrets detection (fast, Rust-based) | `https://github.com/sirwart/ripsecrets` |
| **gitleaks** | Secrets detection (regex-based) | `https://github.com/gitleaks/gitleaks` |
| **check-added-large-files** | Prevent accidental binary commits | `https://github.com/pre-commit/pre-commit-hooks` |
| **no-commit-to-branch** | Protect main/master | Same |
| **check-merge-conflict** | Detect unresolved conflicts | Same |
| **detect-private-key** | Block private keys | Same |
| **mixed-line-ending** | Normalize line endings | Same |
| **trailing-whitespace** | Clean whitespace | Same |

### CI Security Workflow Template

**`.github/workflows/security-scan.yml`:**
```yaml
name: Security Scan
on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read
  security-events: write

jobs:
  osv-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: google/osv-scanner-action@v2
        with:
          scan-args: |-
            --lockfile=./package-lock.json
            --lockfile=./go.sum
            --recursive
            ./

  harden-runner:
    runs-on: ubuntu-latest
    steps:
      - uses: step-security/harden-runner@v2
        with:
          egress-policy: audit
      - uses: actions/checkout@v4
      # ... build steps
```

### Renovate Config with Age-Gating

**renovate.json:**
```json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": ["config:recommended"],
  "minimumReleaseAge": "3 days",
  "vulnerabilityAlerts": {
    "labels": ["security"],
    "minimumReleaseAge": "0 days"
  },
  "packageRules": [
    {
      "matchUpdateTypes": ["major"],
      "minimumReleaseAge": "7 days"
    }
  ]
}
```

---

## devenv.sh Module Coverage Summary

Complete mapping of requested languages to devenv.sh module support:

| Language | Module Name | Has Module | Key Features |
|----------|-------------|-----------|--------------|
| Java | `languages.java` | Yes | JDK, Maven, Gradle, LSP |
| Kotlin | `languages.kotlin` | Yes | LSP (inherits Java JDK) |
| C# / F# | `languages.dotnet` | Yes | SDK package, LSP |
| JavaScript | `languages.javascript` | Yes | npm, pnpm, yarn, bun, corepack |
| TypeScript | `languages.typescript` | Yes | LSP |
| PHP | `languages.php` | Yes | PHP version, extensions, LSP |
| Ruby | `languages.ruby` | Yes | Bundler, LSP |
| Swift | `languages.swift` | Yes | Package, LSP |
| Dart | `languages.dart` | Yes | Package, LSP |
| Python | `languages.python` | Yes | Version, poetry, uv, venv, LSP |
| Go | `languages.go` | Yes | Package |
| Rust | `languages.rust` | Yes | Channel, components, targets, LSP |
| Scala | `languages.scala` | Yes | sbt, Mill, LSP |
| Elixir | `languages.elixir` | Yes | Package, LSP |
| Erlang | `languages.erlang` | Yes | Package, LSP |
| Haskell | `languages.haskell` | Yes | Stack, LSP |
| Clojure | `languages.clojure` | Yes | LSP |
| Perl | `languages.perl` | Yes | LSP |
| R | `languages.r` | Yes | Package, LSP |
| Lua | `languages.lua` | Yes | LSP |
| Zig | `languages.zig` | Yes | Package, LSP |
| C | `languages.c` | Yes | LSP (clangd) |
| C++ | `languages.cplusplus` | Yes | LSP (clangd) |
| Shell/Bash | `languages.shell` | Yes | LSP |
| Nix | `languages.nix` | Yes | LSP |
| Terraform | `languages.terraform` | Yes | Version auto-select, LSP |
| OpenTofu | `languages.opentofu` | Yes | — |
| Helm | `languages.helm` | Yes | Plugins, LSP |
| Ansible | `languages.ansible` | Yes | Package, LSP |
| Groovy | N/A | No | Use `languages.java` + `pkgs.groovy` |
| PowerShell | N/A | No | Use `pkgs.powershell` |
| Objective-C | N/A | No | Use `languages.c` / `languages.cplusplus` |
| Bazel | N/A | No | Use `pkgs.bazel_7` |
| Docker | N/A | No | Use `pkgs.docker` + `pkgs.hadolint` |
| Pulumi | N/A | No | Use `pkgs.pulumi` |
| WASM | N/A | No | Use `pkgs.wasm-pack` + language module |

devenv.sh supports **59 language modules** total (as of 2026-05). The table above covers the 23 requested languages that have dedicated modules plus 7 that require manual package configuration.

---

## Implementation Priority Matrix

For the gdev addon, ecosystems are prioritized by consulting firm encounter frequency:

### Tier 1 — Must Ship (most client engagements)

| Ecosystem | Config Files to Generate | Complexity |
|-----------|------------------------|------------|
| JavaScript/TypeScript | `.npmrc`, `pnpm-workspace.yaml`, `.yarnrc.yml`, `bunfig.toml` | Already planned |
| Python | `pip.conf`, `pyproject.toml` settings | Already planned |
| Go | `GOFLAGS` env var, CI commands | Already planned |
| Rust | `.cargo/config.toml`, CI commands | Already planned |
| Java/Kotlin | `settings.xml`, `gradle.properties`, `verification-metadata.xml` | New — medium |
| C#/.NET | `nuget.config`, `Directory.Build.props`, `Directory.Packages.props` | New — medium |
| Docker | `.hadolint.yaml`, Dockerfile template | New — small |
| Terraform | `versions.tf` template, `.terraformrc` | New — small |

### Tier 2 — Should Ship (common in enterprise)

| Ecosystem | Config Files to Generate | Complexity |
|-----------|------------------------|------------|
| PHP | `composer.json` config section | New — small |
| Ruby | `.bundle/config`, `.gemrc`, `.bundler-audit.yml` | New — small |
| Scala | `project/plugins.sbt` additions | New — small |
| Helm | `Chart.yaml` template | New — small |
| Ansible | `ansible.cfg` signature settings, `requirements.yml` | New — small |
| Bash/Shell | (hooks only — no config files) | New — trivial |

### Tier 3 — Nice to Have (specialized clients)

| Ecosystem | Config Files to Generate | Complexity |
|-----------|------------------------|------------|
| Elixir | `mix.exs` additions for mix_audit | New — small |
| Dart/Flutter | CI commands reference | New — trivial |
| Swift | SPM trust config | New — small |
| Haskell | `cabal.project` settings | New — small |
| C/C++ | Conan profile, vcpkg manifest | New — medium |
| Clojure | deps.edn pinning guidance | New — trivial |
| Bazel | `.bazelrc` hardening | New — small |
| Nix | nix.conf (already planned) | Already planned |

### Tier 4 — Rare (include as templates only)

| Ecosystem | Approach |
|-----------|----------|
| Perl | Reference doc with Carton config |
| R | Reference doc with renv config |
| Lua | Reference doc only |
| Zig | Reference doc only |
| PowerShell | Reference doc only |
| Groovy | Covered by Java/Gradle |
| F# | Covered by .NET |
| Objective-C | Covered by Swift section |
| WASM | Covered by Rust/C++ sections |
| Pulumi | Covered by underlying language |

---

## Appendix: Pre-Commit Hook Tier Mapping

Extension of the existing 3-tier hook system from devenv-security spike to cover all ecosystems:

### Baseline (always enabled, language-agnostic)

```yaml
# git-hooks.hooks in devenv.nix
ripsecrets.enable = true;
check-added-large-files.enable = true;
no-commit-to-branch.enable = true;
check-merge-conflict.enable = true;
detect-private-key.enable = true;
```

### Enhanced (per-language formatters and linters)

| Language | Formatter Hook | Linter Hook |
|----------|---------------|-------------|
| Go | `gofmt` | `go-vet`, `staticcheck` |
| TypeScript/JS | `prettier` | `eslint` |
| Python | `ruff-format` | `ruff` |
| Rust | `rustfmt` | `clippy` |
| Java | `google-java-format` | `checkstyle` |
| Kotlin | `ktlint` | `detekt` |
| C#/.NET | `csharpier` | `dotnet-format` |
| Scala | `scalafmt` | `scalafix` |
| PHP | `php-cs-fixer` | `phpstan` |
| Ruby | `rubocop` | `rubocop` (combined) |
| Swift | `swiftformat` | `swiftlint` |
| Dart | `dart-format` | `dart-analyze` |
| Elixir | `mix-format` | `credo` |
| Haskell | `ormolu` | `hlint` |
| Clojure | `cljfmt` | `clj-kondo` |
| Perl | `perltidy` | `perlcritic` |
| R | `styler` | `lintr` |
| Lua | `stylua` | `luacheck` |
| Zig | `zig-fmt` | N/A |
| C/C++ | `clang-format` | `clang-tidy` |
| Shell | `shfmt` | `shellcheck` |
| Nix | `nixfmt` | `statix` |
| Terraform | `terraform-fmt` | `tflint` |
| Ansible | N/A | `ansible-lint` |
| Docker | N/A | `hadolint` |
| Helm | N/A | `helm-lint` |
| Bazel | `buildifier` | N/A |

### Specialized (security-focused, opt-in)

| Hook | Applicable Ecosystems |
|------|-----------------------|
| `govulncheck` | Go |
| `cargo-audit` | Rust |
| `bundler-audit` | Ruby |
| `mix-deps-audit` | Elixir |
| `composer-audit` | PHP |
| `pip-audit` | Python |
| `npm-audit` | JavaScript |
| `tfsec` / `checkov` | Terraform, Helm, Ansible, K8s |
| `bandit` | Python (SAST) |
| `brakeman` | Ruby (Rails SAST) |
| `sobelow` | Elixir (Phoenix SAST) |
| `psalm` | PHP (security-focused SAST) |
| `lock-file-audit` | All (custom — checks lockfile freshness) |
| `nix-secrets-check` | Nix (custom — scans .nix files) |
