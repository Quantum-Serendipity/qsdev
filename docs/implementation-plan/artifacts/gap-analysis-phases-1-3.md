# Gap Analysis: Phases 1-3

Systematic review of internal consistency, cross-phase dependencies, missing units, ambiguities, acceptance criteria gaps, interface mismatches, testing gaps, contradictions, scalability concerns, and forward compatibility issues.

---

## Phase 1: Foundation & Shared Infrastructure

### 1. Unit Numbering Collision with Phase 3

Phase 3 numbers its units starting at 2.1 through 2.5 instead of 3.1 through 3.5. This collides with Phase 2's unit numbers (2.1 through 2.8). A sub-agent told to "implement Unit 2.2" would not know which phase is intended. Every cross-reference to Phase 3 units is ambiguous. Fix the numbering to 3.1-3.5.

### 2. EcosystemModule Interface Missing nix.conf Fragment Method (Unit 1.7)

The `EcosystemModule` interface defines `DevenvNixFragment()` and `DevenvYamlInputs()` but has no method for contributing nix.conf settings. Phase 5 (Security & Infrastructure Integration) includes nix.conf hardening, which may need per-ecosystem contributions (e.g., Go's GOFLAGS or Rust's sccache config are environment-level, not devenv.nix-level). If nix.conf generation is purely global, this is fine, but the boundary is not documented.

### 3. EcosystemModule Interface Missing Config Validation Method (Unit 1.7)

The interface has no method for an ecosystem module to validate its own generated config files. Validation is handled generically in the generation pipeline (Unit 1.5) by file extension, but ecosystem-specific validation (e.g., running `mvn --validate` on settings.xml, `terraform validate` on .tf files) is not supported. This matters because generic YAML validation will accept a settings.xml that is structurally invalid XML passed through the YAML validator (settings.xml is XML, not YAML).

### 4. EcosystemModule.SecurityConfigs Returns []GeneratedFile but Phase 3 Needs Merge Strategy

`SecurityConfigs(config ModuleConfig) []GeneratedFile` returns full files. But some security configs need to be merged with existing files rather than generated fresh. For example, `pnpm-workspace.yaml` additions (Unit 2.1 Step 3) need to merge with an existing workspace file, not overwrite it. The `GeneratedFile` struct includes a `MergeStrategy` field, but the Phase 2 unit descriptions don't specify which strategy each config file should use, and the Phase 3 composition logic doesn't describe how it handles different merge strategies from different modules targeting the same file path.

### 5. ModuleConfig Struct Undefined (Unit 1.7)

`ModuleConfig` is used as the parameter type for most `EcosystemModule` methods but its definition is listed only as a step description ("Define `ModuleConfig` struct — the ecosystem-specific portion of `WizardAnswers`"). There is no specification of its fields. Phase 2 modules need to know exactly what `ModuleConfig` contains — version string? Package manager choice? Profile reference? Extra settings map? Without this, every Phase 2 module will make different assumptions about what data is available.

### 6. DevenvInput Type Referenced but Not Defined (Unit 1.7)

`DevenvYamlInputs(config ModuleConfig) []DevenvInput` returns `[]DevenvInput`, but `DevenvInput` is not defined in Unit 1.2 (shared types) or Unit 1.7. It is defined in the research spike (`config-template-engine-design.md`) as a struct with `URL` and `Follows` fields, but the phase file omits it. The type needs to be in Unit 1.2's shared types or Unit 1.7.

### 7. HookConfig and CICommand Types Not Defined (Unit 1.7)

`PreCommitHooks() []HookConfig` and `CICommands() []CICommand` reference types that are neither defined in Unit 1.2 nor in Unit 1.7. What fields does `HookConfig` have? Name, repo URL, hook ID, arguments, language? What does `CICommand` contain — command string, description, when to run? These are interface contract types that must be specified.

### 8. WizardField Type Not Defined (Unit 1.7)

`WizardFields() []WizardField` is listed in the interface but the type is not defined anywhere in Phases 1-3. The wizard itself is Phase 6, but the type needs to exist in Phase 1 for the interface to compile.

### 9. PackageManagerInfo Type Not Defined (Unit 1.7)

`PackageManagers() []PackageManagerInfo` references an undefined type. This is metadata about supported package managers — but what metadata? Name, binary name, lockfile name, install command? Not specified.

### 10. Detection Engine Overlap Between Unit 1.3 and Unit 1.7

Unit 1.3 implements a detection engine in `addons/devinit/detect.go` that scans for Go, Node, Python, and Rust. Unit 1.7 defines `EcosystemModule.Detect()` and `ModuleRegistry.DetectAll()` which delegates detection to registered modules. These are two parallel detection systems. The relationship is unclear:
- Does Unit 1.3's detection engine get replaced by Unit 1.7's module-based detection?
- Does Unit 1.3 become a thin wrapper that calls `ModuleRegistry.DetectAll()`?
- Or do they coexist with different purposes?

The Phase 1 completion criteria mention both: "Detection engine returns correct results for test fixture directories" and "Module registry discovers and dispatches to registered modules." A sub-agent implementing Unit 1.3 would not know about Unit 1.7's approach, and vice versa.

**Recommendation:** Unit 1.3 should be refactored to be the harness that calls `ModuleRegistry.DetectAll()`, with the actual detection logic living in ecosystem modules. Unit 1.3 would handle filesystem traversal, caching, and result aggregation. Unit 1.7's `Detect()` method on each module handles ecosystem-specific marker recognition. This should be stated explicitly.

### 11. Template Engine Package Location Ambiguity (Unit 1.4)

Step 1 creates `addons/devinit/tmpl/` as a shared template utilities package. But the config-template-engine-design research places templates in each addon's own package (`addons/devenv/templates/`, `addons/claudecode/templates/`). Unit 1.4 says "Create embed.FS declarations in devenv and claudecode addon packages" but the rendering functions `RenderNix` and `RenderMarkdown` are in `devinit/tmpl/`. This means the devenv addon must import devinit's template package to render its own templates — but devinit imports devenv. This creates a circular dependency: `devinit` -> `devenv` (for generation) and `devenv` -> `devinit/tmpl` (for rendering).

**Recommendation:** The template rendering package should be a standalone shared package (e.g., `internal/tmpl/` or `lib/tmpl/`) that neither addon owns, or it should live in each addon independently.

### 12. Hash Tracking State Location (Unit 1.6)

Unit 1.6 says state is stored in "gdev's config file" at `~/.config/<appname>.yaml`. But the migration-strategy-design research shows state stored per-addon (`devenv.generated.files` and `claudecode.generated.files`). Unit 1.6 doesn't clarify which addon's config namespace the state lives under, or whether there's a shared state store. Since devinit orchestrates both addons, it needs access to both addons' generated state. The ownership and access pattern for state is ambiguous.

### 13. Missing: Error Handling Strategy

No unit in Phase 1 defines a shared error handling strategy. What happens when:
- A template renders invalid Nix? (Unit 1.5 catches this, but what error type?)
- An ecosystem module's Detect() panics?
- Multiple modules claim the same file path with different merge strategies?
- gdev config YAML is corrupt?

There is no `errors` package, no error type hierarchy, no logging infrastructure. Phase 2 and 3 units will each invent their own error handling patterns.

### 14. Missing: Logging Infrastructure

No unit establishes structured logging. Detection engine performance (<100ms requirement), template rendering, file writing, and validation all need instrumentation. Without it, debugging generated configs will require adding logging ad-hoc in later phases.

### 15. Missing: go.mod / Go Module Setup

Unit 1.1 says "Create the three addon packages" but doesn't mention `go.mod` initialization, dependency management, or which Go version to target. The validation findings note Go 1.26.2 in gdev's `go.work`. The plan should state whether these addons live in gdev's existing module or in a separate module, and what the import path is.

### 16. InfraProfile.ConfigFiles() Generates renovate.json (Unit 1.8)

Unit 1.8 step 5 says `ConfigFiles()` generates `renovate.json` or `dependabot.yml`. But Renovate configuration generation is a Phase 5 concern (Security & Infrastructure Integration). Generating it in Phase 1 as part of the profile type means Phase 1 needs to know Renovate's config schema, which is premature. The profile type should define the configuration data; the actual file generation should be deferred to Phase 5.

### 17. Acceptance Criterion Imprecision (Unit 1.8)

"`InfraProfile` captures all infrastructure choices from artifacts/artifact-stores-caches-research.md" is not binary-verifiable. A sub-agent would need to enumerate every infrastructure choice in that artifact and check each one. The criterion should list the specific choices or provide a count.

---

## Phase 2: Ecosystem Modules — Tier 1

### 18. C/C++ Missing from Tier 1

The plan.md Phase Index and Phase 2 file list 8 Tier 1 ecosystems: JS/TS, Python, Go, Rust, Java/Kotlin, .NET, Docker, Terraform. But plan.md's Tier 2 list includes "C/C++ (Conan, vcpkg, CMake)" and `language-ecosystem-coverage.md` classifies C/C++ as Tier 3 complexity "New -- medium". The tier assignment is internally consistent (C/C++ is Tier 2 in plan.md), but the Phase 2 heading says "Tier 1" and includes Docker and Terraform which are infrastructure tools, not languages. This is fine as-is but worth noting that "Tier 1" here means "must-ship" not "programming language."

### 19. pnpm-workspace.yaml Merge Problem (Unit 2.1)

Step 3 says to generate "additions to `pnpm-workspace.yaml`" but `SecurityConfigs()` returns `[]GeneratedFile`. A `GeneratedFile` represents a complete file, not additions to an existing file. If the project already has a `pnpm-workspace.yaml` with workspace definitions, the module must merge its security additions, not overwrite. The `MergeStrategy` enum includes `Merge` but the merge semantics for YAML are not defined anywhere in Phase 1.

The same problem applies to `composer.json` (Tier 2, but the pattern should be established now), `gradle.properties` (Unit 2.5), and `Directory.Build.props` (Unit 2.6).

### 20. Version-Specific Config Comments Not in Acceptance Criteria (Units 2.1-2.8)

The validation findings say "Version-specific configs should include minimum tool version comments" and Unit 2.1 Step 7 says "Include version requirement comments in each config file." But this requirement is not in any acceptance criterion for Units 2.1-2.8. Only Unit 2.1's general narrative mentions it, and the acceptance criteria say "Each config includes inline comments explaining security purpose" — which is different from version requirement comments. Add a criterion like "Each config file includes minimum tool version requirements (e.g., npm 11+, pnpm 10+) as inline comments."

### 21. npm Age Check Bug Not Addressed in Unit 2.1

Validation finding #5 says "Fix npm age check to use `time[dist-tags.latest]` instead of `time.modified`." This affects the npm `.npmrc` config generation in Unit 2.1 — the generated `.npmrc` comment should explain the `min-release-age` field uses npm's internal semantics. But more critically, this is a PreToolUse hook fix (Phase 4), not a config file fix. The `.npmrc` `min-release-age=3` is correct as-is (npm handles the age check internally). The validation finding should be directed at Phase 4, not Phase 2. However, Unit 2.1 should include a comment in the generated `.npmrc` explaining that `min-release-age` uses `time[dist-tags.latest]` semantics as of npm 11+.

### 22. Missing: Test Infrastructure for Ecosystem Modules

Phase 2 completion criteria say "Unit tests pass for all 8 modules" but there is no unit in Phase 1 or Phase 2 that establishes the test infrastructure. Each module needs:
- Fixture directories with marker files for detection testing
- Template rendering tests with snapshot comparison
- Config validation tests (is the generated .npmrc valid INI? Is settings.xml valid XML?)

A shared test helper package would prevent 8 modules from each reinventing fixture creation, temp directory management, and snapshot testing. This should be a unit in Phase 1 or early Phase 2.

### 23. XML Generation Strategy Unspecified

Unit 2.5 (Java/Kotlin) generates Maven `settings.xml` and Unit 2.6 (.NET) generates `nuget.config` — both XML files. The plan.md design principle #5 says "Struct marshaling for YAML/JSON/XML." But no Go XML marshaling infrastructure is established in Phase 1. The template engine (Unit 1.4) handles Nix and Markdown via text/template. The generation pipeline (Unit 1.5) validates `.yaml` and `.json` but has no `.xml` validation. XML is a distinct format requiring its own marshaling approach (`encoding/xml`) and validation. This is a gap in Phase 1 infrastructure.

### 24. TOML Generation Strategy Unspecified

Unit 2.1 generates `bunfig.toml` and Unit 2.4 generates `.cargo/config.toml`. No TOML marshaling or validation is established in Phase 1. Go's standard library doesn't include TOML; a third-party library (e.g., `github.com/BurntSushi/toml`) is needed. The generation pipeline (Unit 1.5) has no `.toml` validation step.

### 25. INI Generation Strategy Unspecified

Unit 2.1 generates `.npmrc` (INI format) and Unit 2.2 generates `pip.conf` (INI format). No INI marshaling is established in Phase 1. These are simple enough that string templates could work, but the approach should be explicit.

### 26. HCL Generation Strategy Unspecified

Unit 2.8 generates `.terraformrc` (HCL format). HCL has its own syntax distinct from JSON/YAML/Nix. The template engine could handle it via text/template (similar to Nix), but this is not mentioned. Validation would need `terraform validate` or the `hclwrite` Go library.

### 27. Deny Rule Format Not Specified

Units 2.1-2.8 each implement `DenyRules()` returning `[]string`. But what format are these strings? Claude Code deny rules use glob patterns in a specific JSON schema. Are these the raw glob patterns? The full JSON objects? The deny rule format must be documented in the `EcosystemModule` interface specification so all 8 modules produce compatible output.

### 28. Pre-commit Hook Integration with prek Not Addressed

Validation finding #3 says "prek replaces pre-commit as default hook runner in devenv 1.11+." All Phase 2 units describe pre-commit hooks, and Phase 3 composes them into devenv.nix. But the hook configuration format for prek vs pre-commit is not distinguished. The HookConfig type (undefined, per gap #7) needs to support both runners, or the plan should commit to one.

### 29. Acceptance Criteria Not Covering Multi-Ecosystem Interaction

No Phase 2 acceptance criterion tests what happens when multiple modules generate configs for the same project. For example: a project with both Go and TypeScript — do the devenv.nix fragments compose correctly? Do the deny rules merge without duplicates? Do the pre-commit hooks from both modules coexist? This is tested in Phase 3's completion criteria but not in Phase 2.

### 30. Docker Module devenv.nix Fragment Contradicts Ecosystem Coverage Research

Unit 2.7 Step 2 says "add `pkgs.docker`, `pkgs.hadolint`, `pkgs.dive` to packages." But `language-ecosystem-coverage.md` Section 24 says "No dedicated Docker module" in devenv.sh and adds packages directly. Unit 2.7's `DevenvNixFragment()` should add to the `packages` list, not use `languages.docker.enable` (which doesn't exist). The step is correct but the interface design doesn't account for modules that add packages without using a `languages.*` module. The `DevenvNixFragment()` return type is `string` (a Nix fragment) — it can work, but it requires the fragment to be a `packages = [ ... ];` line rather than a `languages.docker.enable = true;` line. This pattern difference should be documented.

### 31. Terraform Module DenyRules Block terraform init (Unit 2.8)

Step 5 says deny rules include `terraform init` with the note "ensure lockfile is committed first." But `terraform init` is required for normal development workflow (it's how you install providers). Denying it entirely would break development. The deny rule should be more nuanced — perhaps only blocking `terraform init -upgrade` or requiring the user to confirm. This is a security-vs-usability tradeoff that needs explicit resolution.

---

## Phase 3: devenv Addon — Core Generation

### 32. Unit Numbering: 2.1-2.5 Should Be 3.1-3.5

As noted in gap #1, Phase 3 uses unit numbers 2.1-2.5 which collide with Phase 2's unit numbers. Fix to 3.1-3.5.

### 33. Phase 3 Goal Says "Composing Fragments from Ecosystem Modules" but Unit 3.2 Hardcodes Language Sub-Templates

The Phase 3 goal statement explicitly says the devenv addon "queries registered ecosystem modules for their devenv.nix fragments" and "not hardcoded sub-templates." But Unit 3.2 (labeled "Unit 2.2") creates hardcoded language sub-templates: `templates/languages/go.nix.tmpl`, `typescript.nix.tmpl`, `python.nix.tmpl`, `rust.nix.tmpl`.

This directly contradicts the phase goal and plan.md design principle #7 ("New ecosystems are added by implementing the module interface — no changes to core code"). If language-specific Nix is in templates inside the devenv addon, then adding Tier 2 ecosystems (Phase 7) requires modifying the devenv addon's templates, not just implementing a new module.

The correct architecture: `DevenvNixFragment()` on each ecosystem module returns the Nix fragment. The devenv addon's main template iterates over detected modules and includes each fragment. No language-specific templates exist in the devenv addon.

**Recommendation:** Rewrite Unit 3.2 to compose `DevenvNixFragment()` outputs from registered modules. Language-specific Nix templates should live inside each ecosystem module's package (Phase 2), not in the devenv addon. The devenv addon's main template should have a loop like:
```
{{ range .LanguageFragments }}
{{ .Fragment }}
{{ end }}
```

### 34. Phase 3 Doesn't Call SecurityConfigs() from Ecosystem Modules

The Phase 3 goal says it generates "Per-ecosystem security config files generated by ecosystem modules." But no unit in Phase 3 actually calls `SecurityConfigs()` on the detected ecosystem modules. Unit 3.1 (labeled "Unit 2.1") generates devenv.yaml. Unit 3.2 generates devenv.nix. Unit 3.3 generates service templates. Unit 3.4 generates .envrc. Unit 3.5 registers CLI commands.

Where is the step that:
1. Iterates over detected ecosystem modules
2. Calls `SecurityConfigs(config)` on each
3. Collects the returned `[]GeneratedFile` files
4. Passes them through the generation pipeline

This is a missing unit. Without it, the `.npmrc`, `pip.conf`, `settings.xml`, `nuget.config`, `.cargo/config.toml`, `.hadolint.yaml`, and `.terraformrc` files designed in Phase 2 are never actually written to disk.

### 35. Phase 3 Doesn't Call PreCommitHooks() from Ecosystem Modules

Similarly, devenv.nix needs pre-commit hook configuration composed from ecosystem modules. The `git-hooks.hooks` section in devenv.nix should be populated by calling `PreCommitHooks()` on each detected module. Unit 3.2 doesn't mention this composition step. The security-hardened boilerplate from `boilerplate-research.md` includes baseline hooks (ripsecrets, check-added-large-files, etc.) but the per-ecosystem hooks (gofmt, prettier, ruff, etc.) need to come from the modules.

### 36. Phase 3 Doesn't Call DevenvYamlInputs() from Ecosystem Modules

Unit 3.1 (labeled "Unit 2.1") mentions "Include language-specific inputs (e.g., nixpkgs-python for Python version pinning)" in Step 4, but doesn't describe calling `DevenvYamlInputs()` on each ecosystem module to collect these inputs. The flow should be: iterate detected modules -> collect DevenvInput lists -> merge into devenv.yaml inputs map.

### 37. Phase 3 Doesn't Call CICommands() from Ecosystem Modules

CI commands (frozen-install, audit) per ecosystem are generated in Phase 2 modules but never composed or output in Phase 3. Where do these end up? In a generated CI workflow file? In CLAUDE.md? In a printed summary? Not specified.

### 38. Service Sub-Templates Are in Phase 3 but Not in Any Ecosystem Module

Unit 3.3 (labeled "Unit 2.3") defines service sub-templates (PostgreSQL, Redis, MySQL, MongoDB, Elasticsearch, RabbitMQ) inside the devenv addon. The phase description says "services remain in the devenv addon since they're devenv-specific, not ecosystem-specific." This is architecturally reasonable but creates an inconsistency: languages are (supposed to be) module-driven, services are template-driven. This dual pattern means:
- Adding a new language = implement EcosystemModule (no core changes)
- Adding a new service = add a template to the devenv addon (core change)

This is acceptable if documented, but it should be an explicit design decision, not an implicit one.

### 39. devenv CLI Commands Need WizardAnswers but Wizard Is Phase 6

Unit 3.5 (labeled "Unit 2.5") says `gdev devenv init` should "run wizard or accept flags." But the huh wizard is Phase 6 (Wizard & Orchestration). Phase 3 can only support flag-based (non-interactive) operation. The acceptance criterion "generates files without wizard" via `--lang go --yes` is fine, but the unit description should clarify that interactive mode is deferred to Phase 6 and Phase 3 only supports flag-driven operation.

### 40. Missing: nix.conf Generation

The devenv-security boilerplate includes a hardened `nix.conf` (10 security settings: sandbox, require-sigs, trusted-users, filter-syscalls, etc.). This is not generated in any Phase 3 unit. It appears to be deferred to Phase 5 (Security & Infrastructure Integration), but Phase 3's goal says it integrates "Security defaults integration (clean environment, unsetEnvVars, dotenv disabled, impure disabled)." The nix.conf settings are part of the security baseline but are missing from Phase 3.

### 41. Missing: .gitignore Generation

The research spike's generation pipeline (`config-template-engine-design.md`) shows `allFiles = append(allFiles, d.gitignoreFile(allFiles))` — a .gitignore entry generated from the list of generated files. No Phase 3 unit generates .gitignore entries. devenv.sh creates a `.devenv/` directory that should be gitignored, and `devenv.local.nix` should be gitignored. This is a small but missing piece.

### 42. Missing: devenv.local.nix.example Generation

The boilerplate-research.md describes a `devenv.local.nix.example` file that shows developers how to add personal overrides without affecting the team config. No Phase 3 unit generates this. It's a small file but important for the "secure by default, weaken explicitly" pattern.

### 43. Phase Completion Criterion "devenv shell Succeeds" Is Manual

Phase 3 completion criterion: "`devenv shell` succeeds in a test directory with generated config (manual verification)." This is the only manual criterion in the phase. For a plan that emphasizes automation, this should at minimum be a scripted integration test, even if it requires `nix` and `devenv` to be available (skip if not).

---

## Cross-Phase Findings

### 44. Phase 2 Modules Generate Files That Phase 3 Never Collects

This is the critical architectural gap. Phase 2 ecosystem modules implement `SecurityConfigs()`, `PreCommitHooks()`, `DevenvYamlInputs()`, `DenyRules()`, and `CICommands()`. Phase 3 is supposed to compose these into the final output. But Phase 3's units only cover devenv.yaml generation (without calling `DevenvYamlInputs()`), devenv.nix generation (without calling `DevenvNixFragment()` from modules or `PreCommitHooks()`), services, .envrc, and CLI commands. There is no "composition" unit that wires ecosystem module outputs into the generation pipeline.

**This means Phase 3 as written would produce a devenv.yaml and devenv.nix with hardcoded templates for 4 languages, but would NOT produce any security config files (.npmrc, pip.conf, etc.) or ecosystem-specific pre-commit hooks.**

Phase 3 needs a new unit (call it 3.6) that:
1. Calls `DetectAll()` or uses wizard answers to determine active modules
2. Calls `DevenvNixFragment()` on each and composes into devenv.nix
3. Calls `SecurityConfigs()` on each and adds to the GeneratedFile list
4. Calls `PreCommitHooks()` on each and composes into devenv.nix git-hooks section
5. Calls `DevenvYamlInputs()` on each and merges into devenv.yaml
6. Calls `CICommands()` on each and outputs them (destination TBD)

### 45. DenyRules() Output Has No Consumer in Phase 3

`DenyRules()` is a Claude Code concern, not a devenv concern. The deny rules should be consumed by Phase 4 (Claude Code Addon), which aggregates them into settings.json. Phase 4's description confirms this: "deny rules aggregated from all detected ecosystem modules." This is correct — but it means Phase 3 is not the consumer. The gap is that the plan doesn't make this cross-phase data flow explicit. Phase 4 needs to call the same ecosystem modules that Phase 3 uses. Both phases need access to the module registry and the wizard answers (or persisted config).

### 46. WizardAnswers vs ModuleConfig Translation Missing

Phase 1 defines `WizardAnswers` (the wizard output) and `ModuleConfig` (the ecosystem module input). Phase 3 needs to translate from `WizardAnswers.Languages[]` to the `ModuleConfig` for each active ecosystem module. No unit in any phase defines this translation. Each `LanguageChoice` in `WizardAnswers` needs to be mapped to the corresponding ecosystem module and converted to a `ModuleConfig`. This mapping function should be in Phase 1 or early Phase 3.

### 47. Profile System Not Wired Into Phase 3

Phase 1 Unit 1.8 defines `InfraProfile` with `EnvironmentVars()` and `ConfigFiles()`. Phase 3 generates devenv.nix and devenv.yaml. The profile's environment variables should appear in devenv.nix's `env` section, and the profile's registry proxy config should appear in ecosystem module security configs (e.g., `.npmrc registry=` line, `.terraformrc` network mirror). But Phase 3 has no step that passes the active profile to ecosystem modules or injects profile env vars into devenv.nix.

The `ModuleConfig` type (undefined, per gap #5) presumably needs a reference to the active `InfraProfile` so that `SecurityConfigs()` can generate registry-aware configs. This dependency chain is not documented.

### 48. Phase 2 Has No Integration Tests Across Modules

Phase 2 completion criteria require unit tests per module but no integration tests that exercise multiple modules together. The first time multiple modules are composed is Phase 3, but Phase 3 shouldn't be the first place that multi-module composition bugs surface. Phase 2 should have at least one integration test that registers 2-3 modules and calls `DetectAll()` on a multi-language fixture directory.

### 49. Persisted Config Shape Diverges Between Research and Phase 1

The devenv-addon-design research defines `DevenvPersistedConfig` with fields like `ProjectName`, `ProjectType`, `Languages []LanguageConfig`, `Services []ServiceConfig`, `GitHooks`, `DirenvEnabled`, etc. Phase 1 Unit 1.2 defines `WizardAnswers` with a different shape (e.g., `Languages []LanguageChoice` not `[]LanguageConfig`, `Direnv bool` not `DirenvEnabled`). Phase 3 Unit 3.5 (labeled "Unit 2.5") says `gdev devenv update` should "load saved config, regenerate files." But the saved config format (from gdev's config key system) is `DevenvPersistedConfig`, while the generation functions take `WizardAnswers`. There must be a conversion between the two, and neither Phase 1 nor Phase 3 defines it.

### 50. config-template-engine-design Shows devinit Calling Both Addons' Generate()

The research spike shows:
```go
devenvFiles, err := d.devenv.Generate(answers)
claudeFiles, err := d.claude.Generate(answers)
```

This means `devinit` calls `Generate()` on both addons. But Phase 3 only covers the devenv addon. Phase 4 covers claudecode. The orchestration that calls both is Phase 6. This is fine architecturally, but Phase 3's CLI command `gdev devenv init` (Unit 3.5) needs to work standalone — it should only generate devenv files, not claude files. The acceptance criteria confirm this, but the boundary should be explicit: Phase 3's `gdev devenv init` calls devenv's `Generate()` only; Phase 6's `gdev init` calls both.

---

## Contradictions with plan.md Design Principles

### 51. Principle #7 vs Phase 3 Hardcoded Templates

Design principle #7: "Each language/platform is a self-contained module... New ecosystems are added by implementing the module interface — no changes to core code." Phase 3 Unit 3.2 creates `templates/languages/go.nix.tmpl`, `typescript.nix.tmpl`, etc. inside the devenv addon. This is core code that must change for each new ecosystem. Contradiction.

### 52. Principle #5 vs Missing XML/TOML/INI/HCL Format Support

Design principle #5: "Format-matched generation. Struct marshaling for YAML/JSON/XML." Phase 1 infrastructure only covers text/template (Nix, Markdown) and struct marshaling (YAML, JSON). XML, TOML, INI, and HCL formats used by Phase 2 modules have no infrastructure. Partial contradiction.

### 53. Principle #3 vs Phase 3 Missing Wizard Stub

Design principle #3: "`gdev init` detects project type, generates all config files, and produces a working `devenv shell` in under 60 seconds. The wizard asks 1 question on the quick path." Phase 3 has no way to achieve this because the wizard is Phase 6. Phase 3 can only support flag-driven (non-interactive) operation. This is not a contradiction per se (the plan is phased), but Phase 3's completion criteria should explicitly note that the "one command, working environment" principle is only achievable after Phase 6.

---

## Scalability Concerns

### 54. 27 Ecosystem Modules All Self-Registering via init()

Unit 1.7 Step 5 says modules self-register via `init()` functions. With 27 modules, every module is initialized at startup even if the project only uses 1-2 ecosystems. This means 27 `Detect()` calls on `DetectAll()`. If each `Detect()` does filesystem scans, startup time may exceed the <100ms target. The registry should support lazy detection or the detection engine should batch filesystem operations (single directory listing, check against all modules' marker files).

### 55. Module init() Import Side Effects

Go's `init()` pattern means all modules must be imported somewhere (typically a `modules/all/all.go` that imports all module packages for their side effects). This is a standard Go pattern but creates a maintenance burden: adding a new module requires updating the import list. This should be documented explicitly.

---

## Forward Compatibility Concerns

### 56. EcosystemModule Interface Will Need Expansion for Phase 9

Phase 9 (Cross-Platform System Detection) adds OS detection, prerequisite mapping, and `gdev devenv doctor`/`gdev devenv setup`. Ecosystem modules may need methods like:
- `Prerequisites() []ToolPrerequisite` — what system tools does this ecosystem need (e.g., Java needs JDK, Terraform needs terraform binary)
- `DoctorChecks() []Check` — ecosystem-specific health checks

The `EcosystemModule` interface should either include these methods now (returning empty defaults) or be designed as a composition of smaller interfaces so that Phase 9 can add capabilities without breaking existing modules.

**Recommendation:** Use interface composition:
```go
type EcosystemModule interface {
    CoreModule       // Name, Tier, Detect
    DevenvGenerator  // DevenvNixFragment, DevenvYamlInputs
    SecurityProvider // SecurityConfigs, DenyRules
    HookProvider     // PreCommitHooks
    CIProvider       // CICommands
}
```
Phase 9 can add `PrerequisiteProvider` without modifying existing interfaces.

### 57. GeneratedFile.Strategy Will Need New Values for Phase 8

Phase 8 (Migration, Update & Polish) adds three-way merge, section markers for CLAUDE.md, and per-file merge strategies. The current `MergeStrategy` enum includes `ThreeWayMerge` and `SectionMarker` already, which is good forward planning. But the three-way merge implementation requires access to the "base" version (the previously generated version), which means the hash tracking system needs to store not just the hash but the actual content of the last generated version. Unit 1.6's `GeneratedState` only stores hashes, not content. Phase 8 will need to either:
- Store full generated content (storage cost for many files)
- Re-generate from saved config (requires config versioning)
- Accept that three-way merge needs the user to provide the base

This should be a noted future consideration in Unit 1.6.

### 58. No Interface for Cross-Platform Abstraction (Phase 9-10)

Phases 9-10 add cross-platform support. The current `EcosystemModule.SecurityConfigs()` returns files with hardcoded paths (e.g., `.npmrc`, `pip.conf`). On Windows, some of these paths differ (e.g., pip config lives at `%APPDATA%\pip\pip.ini`). The interface has no concept of platform-aware path resolution. This will need to be addressed when Phase 9 work begins, but the current interface design doesn't preclude it (paths can be made platform-aware in the module implementations).

---

## Prioritized Recommendations

### Critical (blocks correct implementation)

1. **Fix Phase 3 unit numbering** (gap #1/#32) — Rename units to 3.1-3.5 to avoid collision with Phase 2.

2. **Add Phase 3 ecosystem module composition unit** (gap #44) — New unit 3.6 that calls `DevenvNixFragment()`, `SecurityConfigs()`, `PreCommitHooks()`, `DevenvYamlInputs()` from all detected ecosystem modules and feeds results into the generation pipeline. This is the critical missing piece that connects Phase 2 modules to Phase 3 output.

3. **Remove hardcoded language templates from Phase 3** (gap #33/#51) — Unit 3.2 should compose `DevenvNixFragment()` output from modules, not maintain its own language templates. This aligns with design principle #7.

4. **Define all referenced types in Phase 1** (gaps #5/#6/#7/#8/#9) — `ModuleConfig`, `DevenvInput`, `HookConfig`, `CICommand`, `WizardField`, `PackageManagerInfo` must be fully specified with fields in Unit 1.2 or Unit 1.7.

### High (causes significant rework if not addressed)

5. **Resolve detection engine duplication** (gap #10) — Clarify that Unit 1.3 is the detection harness and Unit 1.7's `Detect()` is the per-module logic. Don't implement two parallel detection systems.

6. **Fix template package circular dependency** (gap #11) — Move template rendering to a standalone shared package.

7. **Add XML/TOML/INI/HCL format support to Phase 1** (gaps #23/#24/#25/#26) — Either add validation for these formats to the generation pipeline (Unit 1.5) or explicitly document that ecosystem modules handle their own format-specific generation and validation.

8. **Define deny rule format** (gap #27) — Specify whether `DenyRules()` returns glob pattern strings or structured objects. Phase 4 depends on this.

9. **Add WizardAnswers-to-ModuleConfig translation** (gap #46) — Define the mapping function in Phase 1 or early Phase 3.

10. **Wire InfraProfile into ecosystem module generation** (gap #47) — Ensure `ModuleConfig` includes profile data so security configs can be registry-aware.

### Medium (improves quality and maintainability)

11. **Add test infrastructure unit to Phase 1** (gap #22) — Shared test helpers for fixture directories, snapshot testing, and format-specific config validation.

12. **Add error handling and logging units to Phase 1** (gaps #13/#14) — Shared error types, structured logging.

13. **Add multi-module integration test to Phase 2** (gap #48) — Test composition before Phase 3.

14. **Add .gitignore and devenv.local.nix.example generation to Phase 3** (gaps #41/#42).

15. **Defer InfraProfile.ConfigFiles() to Phase 5** (gap #16) — Profile types define data; Phase 5 generates files.

16. **Clarify Phase 3 CLI commands are flag-only until Phase 6** (gap #39).

17. **Document pnpm-workspace.yaml merge semantics** (gap #19) — Define how `SecurityConfigs()` output merges with existing files.

### Low (polish and forward-thinking)

18. **Use interface composition for EcosystemModule** (gap #56) — Prepare for Phase 9 expansion.

19. **Document service vs language architecture asymmetry** (gap #38).

20. **Add lazy detection or batched filesystem scan** (gap #54) — Scalability for 27 modules.

21. **Note three-way merge content storage need in Unit 1.6** (gap #57).

22. **Specify go.mod setup in Unit 1.1** (gap #15).

23. **Revisit terraform init deny rule** (gap #31) — Too aggressive; blocks normal workflow.
