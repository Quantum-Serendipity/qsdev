# Gap Analysis: Phases 7 and 8

Analysis of Phase 7 (Ecosystem Modules — Tiers 2-4) and Phase 8 (Migration, Update & Polish) against the master plan, Phase 1 interface definitions, Phase 2 quality bar, and the language-ecosystem-coverage artifact.

---

## Phase 7 Findings

### 1. Tier 4 Ecosystem Count Mismatch (Critical)

The master plan lists **10** Tier 4 ecosystems: Perl, R, Lua, Zig, PowerShell, Groovy, F#, Objective-C, WASM, Pulumi.

Phase 7 Unit 7.4 lists only **5**: Perl, R, Lua, Zig, PowerShell.

The missing 5 are Groovy, F#, Objective-C, WASM, and Pulumi. The language-ecosystem-coverage artifact (Implementation Priority Matrix, Tier 4 table) marks these as "covered by" other ecosystems:

- Groovy: "Covered by Java/Gradle"
- F#: "Covered by .NET"
- Objective-C: "Covered by Swift section"
- WASM: "Covered by Rust/C++ sections"
- Pulumi: "Covered by underlying language"

**The gap**: Phase 7 silently drops these 5 without explanation. Even if they are subsumed by other modules, this needs to be explicit. There should be either:
- Stub modules that delegate detection to the parent ecosystem module and generate a reference note, or
- An explicit "Not Implemented — See Module X" mapping in the registry so `DetectAll()` handles these correctly.

Without this, a user with a `Pulumi.yaml` or `*.fs` file gets no detection, no guidance. The detection engine will miss these ecosystems entirely. The registry also won't reach the plan's stated "27 ecosystem modules" — it will be 24 at most.

**Recommendation**: Add a Unit 7.5 for alias/delegation modules, or explicitly document in Phase 7 that these 5 are handled as sub-cases of their parent modules and ensure detection markers (e.g., `*.fsx`, `*.groovy`, `Pulumi.yaml`, `*.wasm`, `*.m`) are registered in the parent modules.

### 2. C/C++ Is Listed as Tier 2 in Phase 7 but Tier 3 in the Coverage Artifact (Inconsistency)

The language-ecosystem-coverage artifact lists C/C++ under Tier 3 ("Nice to Have — specialized clients"), not Tier 2. The master plan Tier 2 list does not include C/C++ either — it lists: PHP, Ruby, Scala, Helm, Ansible, Bash/Shell.

But Phase 7 places C/C++ in Unit 7.2 (Tier 2), giving it full config generators. This means:
- Phase 7 claims 7 Tier 2 modules (PHP, Ruby, Scala, C/C++, Helm, Ansible, Bash) but the plan says Tier 2 has 6 ecosystems.
- C/C++ in the artifact is listed with "New — medium" complexity in Tier 3, not Tier 2.

**Impact**: The tier mismatch changes the expected scope for C/C++. Tier 2 gets "full config generators" while Tier 3 gets "detection + devenv.nix + basic security." C/C++ with Conan, vcpkg, and Meson support is genuinely medium complexity — it probably deserves Tier 2 treatment. But the plan, Phase 7, and the artifact need to agree.

**Recommendation**: Either promote C/C++ to Tier 2 in the master plan and artifact (reflecting Phase 7's decision), or demote it to Unit 7.3 with Tier 3 treatment. The former is better — C/C++ is common enough in consulting.

### 3. EcosystemModule Interface Insufficiency for Unusual Ecosystems

Phase 1 Unit 1.7 defines the `EcosystemModule` interface with these methods:

```
Name(), DisplayName(), Tier(), Detect(), DevenvNixFragment(), DevenvYamlInputs(),
SecurityConfigs(), PreCommitHooks(), DenyRules(), CICommands(),
PackageManagers(), WizardFields()
```

Several Tier 2-4 ecosystems have characteristics that don't map cleanly:

**Bazel (Tier 3)**: Bazel's hermetic build system is its own ecosystem that subsumes other language ecosystems. A Bazel project might build Go, Java, and Python code, but dependency management happens through `MODULE.bazel`, not `go.mod` or `pom.xml`. The interface assumes one ecosystem = one set of config files. But Bazel's `DevenvNixFragment` would need to suppress other detected ecosystems' fragments (e.g., don't add `languages.go.enable = true` when the project uses `rules_go` through Bazel). There is no interface method for "I override these other modules." The `.bazelrc` with `--sandbox_default_allow_network=false` is security-critical but doesn't map to `SecurityConfigs()` (which returns `[]GeneratedFile`) — it maps to modifying an existing file's content.

**Nix (Tier 3)**: Nix flake inputs are the "dependencies" — but they don't have a package manager in the traditional sense. `SecurityConfigs()` for Nix would return nix.conf settings, but those are system-level (per Phase 5 Unit 4.3, they're generated as a recommendation document, not a project-level config). The `DenyRules()` method is awkward — what Claude Code deny rules apply to Nix? `nix profile install`? `nix flake update`? These aren't well-defined in the existing research.

**Bash/Shell (Tier 2)**: Has no package manager at all. `PackageManagers()` returns empty. `SecurityConfigs()` returns empty. `CICommands()` returns empty. The module exists purely for detection + devenv.nix + pre-commit hooks. This works within the interface but feels like a degenerate case.

**Docker (already Tier 1)**: The artifact notes Docker has no dedicated devenv.sh module — it uses packages. But `DevenvNixFragment()` is supposed to return a Nix fragment. Docker's fragment would be `packages = [ pkgs.docker pkgs.hadolint pkgs.dive ];` which is a different shape from `languages.go.enable = true`. The interface doesn't distinguish between `languages.X.enable` style and `packages +=` style.

**Recommendation**: Add an optional interface method `OverrideModules() []string` for ecosystems like Bazel that subsume others. Consider a `DevenvPackages()` method separate from `DevenvNixFragment()` for ecosystems that add packages rather than enabling language modules. Or, define the Nix fragment contract more precisely to handle both patterns.

### 4. Phase 11 Interface Extension Not Planned in Phase 7

Phase 11 (Unit 11.5) extends `EcosystemModule` with two new methods:
```go
VerificationCommands() VerificationCommands
ManifestFiles() []ManifestFileInfo
```

Phase 7 implements 19 modules. Phase 11 is listed as depending on Phase 2 (Tier 1 only). This means:
- The 19 Tier 2-4 modules from Phase 7 will NOT implement `VerificationCommands()` or `ManifestFiles()` unless someone retrofits them.
- If these are added to the `EcosystemModule` interface directly (not as a supplementary interface), all 19 Phase 7 modules will need to be updated.
- If they use a supplementary `VerifiableModule` interface (Phase 11 Step 4 mentions this as an option), modules can opt-in, but then the aggregation functions need type assertions.

**Recommendation**: Phase 7 should either implement these methods (even with empty/minimal returns for Tier 3-4) or the plan should explicitly note that Phase 11 will retrofit them. The cleaner approach is to include them in Phase 7 from the start, since the module author already has all the domain context.

### 5. Unit Granularity — 19 Ecosystems in 4 Units

Unit 7.1 (3 ecosystems), Unit 7.2 (4 ecosystems), Unit 7.3 (7 ecosystems), Unit 7.4 (5 ecosystems).

**C/C++ deserves its own unit.** It has 3 build systems (Conan, vcpkg, Meson) plus CMake detection, plus sccache/ccache integration from infrastructure profiles. The complexity is comparable to the JVM module (Phase 2 Unit 2.5) which gets its own unit.

**Bazel deserves its own unit.** It has hermetic build semantics, remote cache integration from the infrastructure profile, and the override problem described above.

**The rest are fine grouped.** PHP, Ruby, Scala are small. Helm, Ansible, Bash are small. Elixir, Dart, Swift, Haskell, Clojure, Nix are straightforward Tier 3 modules. Perl, R, Lua, Zig, PowerShell are trivial Tier 4 modules.

**Recommendation**: Split Unit 7.2 — extract C/C++ into its own Unit 7.2a. Move Bazel from Unit 7.3 to its own Unit 7.3a. Keep the rest as grouped.

### 6. Acceptance Criteria Specificity

Compare Phase 2 acceptance criteria (e.g., Unit 2.1 has 8 specific criteria checking `.npmrc` content, package manager detection, deny rules, inline comments) to Phase 7 acceptance criteria.

Phase 7 Unit 7.1 has only 3 criteria:
- "PHP module documents Composer 2.9 built-in defense"
- "Ruby frozen bundle in CI"
- "Scala dependency locking plugin added"

These are extremely thin. They don't verify:
- Detection heuristics work (what files are checked?)
- devenv.nix fragments render valid Nix
- Security configs are valid per-format
- Deny rules are generated for each ecosystem's package manager
- Pre-commit hooks are configured
- CI commands are generated
- Inline comments explain security purpose (required by Phase 2)

Phase 7 Unit 7.3 is even worse — 4 criteria for 7 ecosystems, and they're all generic ("All 7 modules detect their ecosystem markers").

**Recommendation**: Expand acceptance criteria to match Phase 2's specificity. At minimum, each Tier 2 module needs: detection works, devenv.nix valid, security configs valid, deny rules present, hooks configured, CI commands present, inline comments present. Tier 3 modules need: detection works, devenv.nix valid, known limitations documented.

### 7. Missing Security Research for Several Tier 2 Ecosystems

Phase 2 modules have detailed security configs from the package-supply-chain-security spike. Phase 7 Tier 2 modules reference some security settings but lack the depth:

- **PHP**: Mentions `audit.block-insecure: true` and `allow-plugins` whitelist, but the artifact also lists `preferred-install: dist`, `secure-http: true`, and the `Roave Security Advisories` Composer plugin. Phase 7 doesn't mention Roave at all.
- **Ruby**: Mentions `.bundle/config` and `.gemrc` but not `.bundler-audit.yml` (which the artifact includes). Also doesn't mention `brakeman` for Rails projects.
- **Scala**: Only mentions `sbt-dependency-lock` and `sbt-dependency-check`. The artifact also notes that Scala shares the full JVM security stack (Maven Central settings, checksum policy). Phase 7 doesn't generate any Scala-specific `settings.xml` or connect to the JVM module's security configs.
- **Helm**: Mentions OCI registry and cosign verification, but doesn't address the `helm-secrets` plugin for handling secrets in values files, which is a critical security concern.
- **Ansible**: Mentions GPG signature verification but doesn't mention `ansible-vault` for secrets management, which is arguably the most security-critical Ansible feature.

**Recommendation**: Each Tier 2 module's Steps section should cross-reference the language-ecosystem-coverage artifact and include all listed security features, not a subset.

### 8. Missing Deny Rules and CI Commands for Tier 2-3 Ecosystems

Phase 7's ecosystem descriptions mention some security configs but are inconsistent about deny rules and CI commands:

- **PHP**: No deny rules listed. Should have `composer require *`, `composer install` (without `--no-dev`), etc.
- **Ruby**: No deny rules listed. Should have `gem install *`, `bundle add *`.
- **Scala**: No deny rules listed. Should have `sbt update`, `sbt +update`, etc.
- **Helm**: No deny rules listed. Should have `helm install *`, `helm upgrade *`.
- **Ansible**: No deny rules listed. Should have `ansible-galaxy collection install *`.
- **Bash/Shell**: No deny rules (makes sense — no package manager).
- **C/C++**: No deny rules listed. Should have `conan install *`, `vcpkg install *`.
- **Tier 3** modules: None list deny rules. Even Tier 3 modules should return deny rules for Claude Code (Elixir: `mix deps.get`, Dart: `dart pub add`, etc.).
- **Tier 4** modules: Only Perl lists CI commands (`carton install --deployment`). R lists `renv::restore()`. Others are silent.

**Recommendation**: Add deny rules and CI commands to each ecosystem's Steps section, at least for Tier 2. Reference the `EcosystemModule` interface — `DenyRules()` and `CICommands()` are required methods, so every module must implement them.

### 9. No Testing Strategy for 19 Modules

Phase 7's completion criteria include "Unit tests pass for all 19 modules" but no unit describes how testing works:

- No fixture directories defined for detection testing.
- No template rendering tests described.
- No security config validation described (is the generated `.bundle/config` valid YAML? Is the generated `project/plugins.sbt` valid Scala?).
- No Nix syntax validation for devenv.nix fragments.
- Phase 2 has more testing detail (round-trip tests, `nix-instantiate --parse`), but Phase 7 has zero.

**Recommendation**: Add a testing step to each unit or a dedicated testing unit (Unit 7.5) that defines: fixture directories per ecosystem, template rendering validation, config format validation, and Nix syntax checks.

---

## Phase 8 Findings

### 10. Unit Numbering Error (Cosmetic but Confusing)

Phase 8 units are numbered 6.1 through 6.7, not 8.1 through 8.7. This suggests the file was written when Phase 8 was Phase 6 in an earlier plan version and was renumbered without updating internal unit IDs. This is confusing when cross-referencing — "Unit 6.1" could mean Phase 6 Unit 5.1 or Phase 8 Unit 6.1.

**Recommendation**: Renumber to 8.1 through 8.7.

### 11. Multi-Addon File Conflict Not Addressed

Both the devenv addon and the claudecode addon can write to the same files:

- **`.gitignore`**: devenv needs entries (`.devenv*`, `.direnv/`), Claude Code might need entries (`.claude/`, agent artifacts). Neither Phase 3, Phase 4, nor Phase 8 describes who owns `.gitignore` or how entries from multiple addons are merged.
- **`.pre-commit-config.yaml`**: If the user has an existing pre-commit config, and both devenv (hooks) and claudecode (hook scripts) need entries, how are they merged?
- **`devenv.nix`**: The devenv addon generates this, but ecosystem modules from Phase 7 contribute fragments. The composition model is described in Phase 3, but Phase 8's update strategy for `devenv.nix` treats it as a single file. What if one ecosystem module is added to a project between `gdev init` runs? The `.devenv.nix.new` diff would show the new ecosystem's fragment, but the user would need to manually merge it — fine for small changes, confusing for large ones.

**Recommendation**: Define a `.gitignore` merge strategy (section markers or append-only with dedup). Clarify ownership for shared files.

### 12. Three-Way Merge Scope Is Too Narrow

Unit 6.2 (Phase 8) defines three-way merge only for `settings.json` and `.mcp.json`. But several other generated files need similar treatment:

- **`.hadolint.yaml`** (Docker module): User adds trusted registries. Update needs to preserve user additions.
- **`nuget.config`** (XML): User adds package sources. Update needs to preserve them.
- **`settings.xml`** (Maven, XML): User adds mirrors or profiles. Update needs to preserve them.
- **`gradle.properties`**: User adds build properties. Update needs to preserve them.
- **`composer.json`**: User adds `config` entries. Update needs to preserve them.
- **`.npmrc`**: User adds scoped registry entries (`@company:registry=...`). Update needs to preserve them.
- **`pip.conf`**: User adds `extra-index-url`. Update needs to preserve it.

Phase 8 Unit 6.1 classifies each file as unmodified/modified/deleted/new and routes to merge strategies. But the only merge strategies defined are: Overwrite (unmodified files), ThreeWayMerge (settings.json, .mcp.json), SectionMarker (CLAUDE.md), and .new+diff (devenv.nix). What strategy applies to `.npmrc`, `nuget.config`, `composer.json`, etc.?

Looking at Phase 1 Unit 1.2, `MergeStrategy` has: `Overwrite`, `Append`, `Merge`, `Skip`, `SectionMarker`, `ThreeWayMerge`, `LibraryManaged`. Phase 8 only implements three of these seven. `Append`, `Merge`, `Skip`, and `LibraryManaged` are undefined.

**Recommendation**: Define merge strategies for every generated file type. At minimum:
- `.npmrc`, `pip.conf`, `.bundle/config` (INI-like): line-based merge or section markers.
- `nuget.config`, `settings.xml` (XML): either three-way merge on the parsed tree, or hash-based overwrite (simpler but lossy).
- `composer.json` (JSON): three-way merge like settings.json.
- `.hadolint.yaml`, `Chart.yaml` (YAML): three-way merge on the parsed tree.
- `.cargo/config.toml`, `bunfig.toml`, `.bazelrc` (TOML/custom): hash-based overwrite or section markers.

### 13. `gdev init --update` Behavior Undefined for New Ecosystem Addition

What happens when:
1. User runs `gdev init` selecting Go.
2. User adds a `package.json` to the project.
3. User runs `gdev init --update`.

Does `--update` re-run detection and add JavaScript/TypeScript configs? Or does it only regenerate from the saved `WizardAnswers`?

Phase 8 Unit 6.1 says: "reads stored `GeneratedState`" and "load saved config." This implies `--update` regenerates from the original wizard answers, NOT from fresh detection. But the Steps section says "load saved config -> load GeneratedState -> generate new files" — the "generate new files" could mean "from current config" or "from fresh detection."

If `--update` doesn't re-detect, users must run `gdev init` again (full wizard) to add ecosystems. But `gdev init` on an existing project triggers merge mode (Phase 6 Unit 5.3). The interaction between "full init on existing project" and "--update" is not clearly distinguished.

**Recommendation**: Define two distinct paths:
- `gdev init --update`: Regenerate from saved config with current templates (propagate team standards, no detection).
- `gdev init`: Full init flow — re-detect, re-wizard, handle existing files via merge.
- Document the distinction clearly.

### 14. Team Standards Versioning Lacks Rollback

Unit 6.5 (Phase 8) describes propagating template version bumps. But there is no rollback mechanism:

- What if a new template introduces a bug in generated configs?
- What if a team needs to pin to a specific template version while investigating an issue?
- What if `--update` applies changes and something breaks — can the user `gdev init --rollback`?

The GeneratedState stores template versions, but there's no "previous state" for comparison or revert.

**Recommendation**: At minimum, `gdev init --update` should generate a backup of changed files before overwriting (e.g., `settings.json.bak` or `.gdev/backups/<timestamp>/`). Ideally, since this is a git-tracked project, the recommendation should be "commit before running `gdev init --update`" — but this should be enforced or warned, not just documented.

### 15. Integration Tests Are Underspecified

Unit 6.6 (Phase 8) defines 8 integration tests. But:

- **No Tier 2-4 ecosystem tests.** All tests reference Go, TypeScript, or generic files. None test PHP, Ruby, C/C++, or any Tier 2-4 ecosystem from Phase 7. Phase 8 depends on Phases 2-5 but not Phase 7 — yet the integration tests should cover the full ecosystem matrix.
- **No multi-ecosystem conflict test.** What happens when Go + Python + Rust are all detected? Do their devenv.nix fragments compose correctly? Do deny rules accumulate without duplicates?
- **No performance test.** The plan states `gdev init` should complete in <60 seconds (Design Principle 3). The tests should verify this.
- **No error path tests.** What if detection finds a corrupt `package.json`? What if the template engine fails for one ecosystem but succeeds for others?
- **No cross-platform test.** Phase 9 introduces cross-platform support. Integration tests need to at least mock different OS environments.
- **Fixture management is undefined.** The tests "create temp directories with fixture files" but there's no shared fixture library or test helper.

**Recommendation**: Expand integration tests significantly. Add tests for: at least 2 Tier 2 ecosystems, multi-ecosystem composition, error handling, and performance. Define a test fixture library in Phase 1 or Phase 7.

### 16. Documentation Gaps

Unit 6.7 (Phase 8) lists documentation targets but is missing:

- **Per-ecosystem documentation.** Each ecosystem module generates security configs with specific settings. Users need to understand what each setting does and how to customize it. The unit says "Configuration reference covers all generated files" but doesn't describe how this scales to 27 ecosystems.
- **Troubleshooting guide.** What if `devenv shell` fails after `gdev init`? What if a pre-commit hook fails? What if age-gating blocks a legitimate package?
- **Escape hatches.** How to disable specific security settings without disabling the whole security layer. How to allowlist a package that triggers age-gating.
- **Man pages.** The plan mentions "single binary, zero prerequisites" but no man page generation is described. GoReleaser can generate man pages from cobra commands.
- **Generated inline documentation.** Phase 2 requires "inline comments explaining security purpose" in generated configs. Phase 7 doesn't mention this requirement for Tier 2-4 modules. Phase 8 doesn't verify it in integration tests.

**Recommendation**: Add troubleshooting and escape hatch documentation to Unit 6.7. Add a requirement for generated inline comments to Phase 7's acceptance criteria.

### 17. Ecosystem Module Conflict Resolution

What if two ecosystem modules generate conflicting configurations?

- **Python version conflict.** User's project has both `pyproject.toml` (Python 3.12) and an Ansible playbook (requiring Python 3.10). Both Python and Ansible modules would try to set `languages.python`.
- **JDK version conflict.** Scala and Java modules both set `languages.java.jdk.package`. If a project has both a `build.sbt` and a `pom.xml`, which JDK version wins?
- **Multiple build systems.** A project with both `CMakeLists.txt` and `meson.build` — does C/C++ module generate configs for both? What about a project with both Maven and Gradle (Phase 2 Unit 2.5 handles this within the JVM module, but what about cross-module conflicts)?
- **Nix package duplicates.** Go module adds `pkgs.go`, Bazel module might also want Go tooling. Are packages deduplicated?

Neither Phase 7 nor Phase 8 addresses conflict resolution between ecosystem modules.

**Recommendation**: Define a conflict resolution strategy in Phase 1 (add to the module registry or generation pipeline) or Phase 7. Options: priority-based (higher-tier module wins), last-write-wins (module registration order), or user-prompted (wizard asks when conflicts detected). The simplest is: each module returns fragments, the composition layer detects duplicates/conflicts and prompts or merges.

### 18. Tier 4 "Reference Docs Only" — Unclear Implementation

Phase 7 Unit 7.4 says Tier 4 gets "detection + minimal devenv.nix + reference documentation." But the Steps section shows concrete work for each:

- Perl: detection, devenv.nix, CI commands, security notes.
- R: detection, devenv.nix, CI commands.
- Lua: detection, devenv.nix, security notes.
- Zig: detection, devenv.nix, security notes.
- PowerShell: detection, devenv.nix, hooks, security notes.

This is actually more than "reference docs only" — it includes detection, devenv.nix generation, and in some cases CI commands and hooks. The tier label is misleading. These modules have a smaller scope than Tier 2 but they still generate code.

The master plan's Tier 4 definition says "Reference Docs Only (rare but documented)" and the artifact says "include as templates only." But Phase 7 implements real modules with real detection and devenv.nix generation.

**Impact**: This is fine in practice (the modules are useful), but the tier labeling is inconsistent. Anyone reading the plan would expect Tier 4 to be documentation, not working code.

**Recommendation**: Either rename the tier to "Minimal Support" or update the plan description. The current implementation level is appropriate — the labeling just doesn't match.

### 19. Missing Merge Strategy for devenv.yaml

Phase 8 defines merge strategies for devenv.nix (hash + .new file), CLAUDE.md (section markers), settings.json/.mcp.json (three-way merge). But devenv.yaml is not addressed.

Phase 3 Unit 2.1 generates devenv.yaml with `Overwrite` strategy. But on update, if a user has added custom inputs to devenv.yaml (e.g., a private flake input for internal tools), `Overwrite` would destroy them.

The devenv-security spike's migration-strategy-design.md likely classifies devenv.yaml as "machine-owned" (safe to overwrite). But in practice, users DO customize devenv.yaml (adding inputs, changing nixpkgs channels, adding imports). This makes it more like a "machine-owned-with-additions" file that needs three-way merge or section markers.

**Recommendation**: Either implement three-way merge for devenv.yaml (since it's YAML, this is feasible with struct-level merging) or use section markers for the generated portion. At minimum, document that user additions to devenv.yaml will be lost on `--update` and tell users to use devenv.nix for customizations instead.

### 20. No Handling of Partial Failures During Update

Phase 8 Unit 6.1 describes the update flow: "generate new files -> for each file, check modification status -> apply merge strategy -> preview -> confirm -> write -> update state."

But what if the merge succeeds for 8 of 10 files and fails for 2 (e.g., malformed section markers in CLAUDE.md, corrupt settings.json)?

Phase 1 Unit 1.5's generation pipeline says "On any failure: clean up all temp files (but don't roll back already-written files — too complex for v1)." This is reasonable for initial generation, but for updates it's dangerous — you could end up with a partially updated project where some files are new-version and others are old-version.

**Recommendation**: Define behavior for partial update failure. Options:
- Atomic: all-or-nothing (complex but safe — write all to temp, rename all at once).
- Best-effort with report: write what succeeds, report failures, let user fix manually.
- Pre-check: validate all merges can succeed before writing any file.

The pre-check approach is simplest and safest.

---

## Prioritized Recommendations

### Must Fix (plan correctness)

1. **Resolve Tier 4 count mismatch** — account for all 10 Tier 4 ecosystems or explicitly document why 5 are excluded from Phase 7. (Finding 1)
2. **Renumber Phase 8 units** from 6.x to 8.x. (Finding 10)
3. **Define merge strategies for all generated file types**, not just settings.json, .mcp.json, CLAUDE.md, and devenv.nix. At least categorize every generated file by strategy. (Finding 12)
4. **Address the Phase 11 interface extension** — decide whether Phase 7 modules implement `VerificationCommands()` and `ManifestFiles()` upfront or get retrofitted. (Finding 4)
5. **Fix C/C++ tier inconsistency** — align plan, Phase 7, and artifact. (Finding 2)

### Should Fix (quality and completeness)

6. **Expand acceptance criteria** for Phase 7 to match Phase 2's specificity. (Finding 6)
7. **Add deny rules and CI commands** to all Tier 2 ecosystem descriptions. (Finding 8)
8. **Add missing security features** from the artifact to Phase 7 module descriptions. (Finding 7)
9. **Define ecosystem conflict resolution.** (Finding 17)
10. **Clarify `gdev init --update` vs fresh `gdev init`** semantics. (Finding 13)
11. **Add devenv.yaml merge strategy.** (Finding 19)
12. **Define multi-addon file conflict handling** (.gitignore, etc.). (Finding 11)

### Nice to Fix (robustness)

13. **Split C/C++ and Bazel into dedicated units.** (Finding 5)
14. **Add testing strategy** to Phase 7. (Finding 9)
15. **Expand integration tests** in Phase 8 to cover Tier 2 ecosystems, multi-ecosystem, errors, and performance. (Finding 15)
16. **Add documentation for troubleshooting, escape hatches, and per-ecosystem config reference.** (Finding 16)
17. **Address partial update failures.** (Finding 20)
18. **Add rollback or backup mechanism** to `gdev init --update`. (Finding 14)
19. **Clarify Tier 4 labeling** — "Reference Docs Only" doesn't match the actual implementation. (Finding 18)
20. **Handle EcosystemModule interface gaps** for Bazel (override semantics) and Docker/Bazel/PowerShell (packages-only devenv.nix pattern). (Finding 3)
