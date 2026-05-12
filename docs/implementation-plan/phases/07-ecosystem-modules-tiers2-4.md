# Phase 7: Ecosystem Modules — Tiers 2-4

## Goal

Implement the remaining 19 ecosystem modules covering languages and platforms commonly encountered by a software engineering consulting firm. Tier 2 (commonly encountered) modules get full config generators; Tier 3 (specialized) get detection + devenv.nix + basic security; Tier 4 (rare) get detection + reference documentation only.

## Dependencies

Phase 2 complete (Tier 1 modules establish the pattern). Module interface proven.

## Phase Outputs

- 7 Tier 2 modules: PHP (Composer), Ruby (Bundler), Scala (sbt), C/C++ (Conan, vcpkg, CMake), Helm, Ansible (Galaxy), Bash/Shell
- 7 Tier 3 modules: Elixir (Mix/Hex), Dart/Flutter (pub), Swift (SPM), Haskell (Cabal/Stack), Clojure (deps.edn), Bazel (bzlmod), Nix (flake inputs)
- 5 Tier 4 modules: Perl (Carton), R (renv), Lua (LuaRocks), Zig, PowerShell (PSGallery)

---

### Unit 7.1: Tier 2 Modules — PHP, Ruby, Scala

**Description:** Full config generators for 3 commonly-encountered ecosystems.

**Steps:**
1. **PHP (Composer)**: Detect `composer.json`/`composer.lock`. Note Composer 2.9+ blocks vulnerable packages by default (strongest built-in defense). Generate `composer.json` config section with `secure-http: true`, `lock: true`, `audit.block-insecure: true`, `allow-plugins` whitelist. devenv.nix: `languages.php.enable = true`. Hooks: `phpcs`, `phpstan`.
2. **Ruby (Bundler)**: Detect `Gemfile`/`Gemfile.lock`. Generate `.bundle/config` with `BUNDLE_FROZEN: true`, `BUNDLE_DISABLE_EXEC_LOAD: true`. Generate `.gemrc` with https-only sources. devenv.nix: `languages.ruby.enable = true`. Hooks: `rubocop`. CI: `bundle install --frozen`, `bundle-audit check`.
3. **Scala (sbt)**: Detect `build.sbt`/`project/`. Generate `project/plugins.sbt` with `sbt-dependency-lock` and `sbt-dependency-check` plugins. devenv.nix: `languages.scala.enable = true`. Hooks: `scalafmt`.

**Acceptance Criteria:**
- [ ] PHP module documents Composer 2.9 built-in defense
- [ ] Ruby frozen bundle in CI
- [ ] Scala dependency locking plugin added

**Status:** Not Started

---

### Unit 7.2: Tier 2 Modules — C/C++, Helm, Ansible, Bash/Shell

**Description:** Full config generators for 4 more commonly-encountered ecosystems.

**Steps:**
1. **C/C++**: Detect `CMakeLists.txt`, `Makefile`, `conanfile.py`/`conanfile.txt`, `vcpkg.json`, `meson.build`. Generate Conan profile with `tools.graph:lockfile_policy=require`, vcpkg manifest with baseline pinning, Meson `.wrap` files with SHA256 hashes. devenv.nix: `languages.c.enable = true` or `languages.cplusplus.enable = true`, `pkgs.cmake`/`pkgs.meson`. sccache/ccache integration from infrastructure profile. Hooks: `clang-format`, `cppcheck`.
2. **Helm**: Detect `Chart.yaml`/`Chart.lock`. Generate Chart.yaml template with exact version pins for dependencies. OCI registry configuration. devenv.nix: `languages.helm.enable = true`. Hooks: `helm-lint`. CI: `helm dependency build`, `cosign verify` for OCI charts.
3. **Ansible**: Detect `ansible.cfg`, `playbooks/`, `roles/`, `requirements.yml`. Generate `ansible.cfg` with GPG signature verification, `required_valid_signature_count = 1`. Generate `requirements.yml` template with version pins. devenv.nix: `languages.ansible.enable = true`. Hooks: `ansible-lint`.
4. **Bash/Shell**: Detect `*.sh`, `Makefile`, `.envrc`. No package manager configs. devenv.nix: `languages.shell.enable = true`. Hooks: `shellcheck`, `shfmt`. Security: `bash -n` validation, ShellCheck severity levels.

**Acceptance Criteria:**
- [ ] C/C++ supports Conan, vcpkg, and Meson build systems
- [ ] Helm uses OCI registry and cosign verification
- [ ] Ansible has GPG signature verification
- [ ] Shell module adds shellcheck and shfmt hooks

**Status:** Not Started

---

### Unit 7.3: Tier 3 Modules — Elixir, Dart, Swift, Haskell, Clojure, Bazel, Nix

**Description:** Detection + devenv.nix + basic security configs for 7 specialized ecosystems.

**Steps:**
1. **Elixir**: Detect `mix.exs`/`mix.lock`. devenv.nix: `languages.elixir.enable = true`. Add `mix_audit` dependency recommendation. Hooks: `mix format`. CI: `mix deps.audit`.
2. **Dart/Flutter**: Detect `pubspec.yaml`/`pubspec.lock`. devenv.nix: `languages.dart.enable = true`. Recommend exact version pins (not `^`). Hooks: `dart format`.
3. **Swift**: Detect `Package.swift`/`Package.resolved`. devenv.nix: `languages.swift.enable = true`. Note SE-0391 package signing (TOFU model). Hooks: `swift-format`.
4. **Haskell**: Detect `*.cabal`, `stack.yaml`, `cabal.project`. devenv.nix: `languages.haskell.enable = true`. Note cabal freeze limitation (not true lockfile). Recommend Stack for lockfile support. Hooks: `ormolu`/`fourmolu`.
5. **Clojure**: Detect `deps.edn`, `project.clj`. devenv.nix: `languages.clojure.enable = true`. Note: no native lockfile support — version pin in config only. Hooks: `cljfmt`.
6. **Bazel**: Detect `MODULE.bazel`, `WORKSPACE`, `.bazelrc`. devenv.nix: add `pkgs.bazel_7`. Generate `.bazelrc` with `--lockfile_mode=update` (dev) / `--lockfile_mode=error` (CI), `--spawn_strategy=sandboxed`, `--sandbox_default_allow_network=false`. Remote cache config from infrastructure profile.
7. **Nix**: Detect `flake.nix`, `flake.lock`. devenv.nix: `languages.nix.enable = true`. Hooks: `statix` (linter), `deadnix` (dead code), `nixfmt` (formatter). Security: flake input pinning, `flake-checker` for outdated inputs.

**Acceptance Criteria:**
- [ ] All 7 modules detect their ecosystem markers
- [ ] All 7 generate valid devenv.nix fragments
- [ ] Known limitations documented (Haskell cabal freeze, Clojure no lockfile)
- [ ] Bazel sandboxed builds and network blocking configured

**Status:** Not Started

---

### Unit 7.4: Tier 4 Modules — Perl, R, Lua, Zig, PowerShell

**Description:** Detection + minimal devenv.nix + reference documentation for 5 rare ecosystems.

**Steps:**
1. **Perl**: Detect `cpanfile`, `Makefile.PL`. devenv.nix: `languages.perl.enable = true`. Note: no signing, no age-gating — version pin in cpanfile only. CI: `carton install --deployment`.
2. **R**: Detect `DESCRIPTION`, `renv.lock`, `.Rprofile`. devenv.nix: `languages.r.enable = true`. renv lockfile is the primary defense. CI: `renv::restore()`.
3. **Lua**: Detect `*.rockspec`, `.luarocks/`. devenv.nix: `languages.lua.enable = true`. Note: no package signing (documented gap since 2019 incident). Recommend Lux over LuaRocks when available.
4. **Zig**: Detect `build.zig`, `build.zig.zon`. devenv.nix: `languages.zig.enable = true`. Content-addressed dependencies (SHA256 mandatory in build.zig.zon) — strongest integrity model after Nix.
5. **PowerShell**: Detect `*.ps1`, `*.psm1`, `requirements.psd1`. devenv.nix: add `pkgs.powershell`. Hooks: `PSScriptAnalyzer`. Note: PSGallery has no age-gating or install script blocking.

**Acceptance Criteria:**
- [ ] All 5 modules detect their ecosystem markers
- [ ] All 5 generate valid devenv.nix fragments
- [ ] Security limitations clearly documented for each
- [ ] Zig's content-addressed model noted as exemplary

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All 19 modules implement `EcosystemModule` interface
- [ ] All 19 register in module registry
- [ ] Detection works for each ecosystem
- [ ] Tier 2 modules have full security config generators
- [ ] Tier 3 modules have detection + devenv.nix + basic security
- [ ] Tier 4 modules have detection + devenv.nix + reference docs
- [ ] Known ecosystem limitations documented in generated output
- [ ] Unit tests pass for all 19 modules
