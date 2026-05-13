# Gap Analysis: Phases 4-6

Analysis date: 2026-05-12. Covers Phase 4 (Claude Code Addon), Phase 5 (Security Infrastructure), Phase 6 (Wizard Orchestration). Cross-references Phases 1, 2, 3, 11 and research artifacts.

---

## Phase 4: Claude Code Addon — Core Generation

### 4.1 Internal Consistency Issues

**GAP-4.1a: Duplicate content in Phase Outputs block.** Lines 20-24 repeat lines 18, 21, 23, and 24 verbatim (five CLI commands listed twice, .mcp.json generation listed twice). This is a copy-paste artifact that should be cleaned up to avoid confusing a sub-agent.

**GAP-4.1b: Unit numbering mismatch with plan.md.** Phase 4 units are numbered 3.1-3.6 internally (matching the claudecode addon's numbering) but this is Phase 4 in the plan index. Plan.md Phase Index says Phase 4 is "Claude Code Addon -- Core Generation" yet the units use prefix `3.x`. This creates ambiguity: when Phase 6 references "Phase 4 security configs," does it mean Unit 3.x or Unit 4.x? A sub-agent given "implement Unit 4.1" would find no unit with that number.

**GAP-4.1c: Unit 3.5 references "Phase 4" for Socket.dev integration.** Step 4 says "Include placeholder for Socket.dev server (actual integration in Phase 4)" but the unit IS in Phase 4. This should say "Phase 5" (Security Infrastructure Integration), where Unit 4.4 (CI Vulnerability Scanning) handles Socket.dev. A sub-agent would search for a Socket.dev unit in their own phase and find nothing.

**GAP-4.1d: settings.json schema validation missing.** Unit 3.1 generates settings.json via struct marshaling, which guarantees valid JSON, but there is no step to validate the generated JSON against Claude Code's actual settings schema. The struct definition in Step 1 says "matching current Claude Code schema" but provides no mechanism to verify this. If Anthropic changes the schema (as they have -- see CVE history in ecosystem research), the generated file could silently produce ignored fields. Add a schema validation step or at minimum a schema version marker.

**GAP-4.1e: Hooks section structure undefined.** Unit 3.1 Step 1 mentions a `Hooks` map in the `SettingsJSON` struct, and Unit 3.3 Step 6 says "Wire into settings.json hooks config: PreToolUse entry pointing to the script." But neither unit defines the exact hooks section schema. The ecosystem research (Section 1.4) documents that hooks use specific fields (`type`, `command`, `args`, `event`, `matcher`), but neither unit specifies which fields are used. A sub-agent implementing Unit 3.1 would need to guess the hooks schema, and a sub-agent implementing Unit 3.3 would need to independently determine how to add entries.

**GAP-4.1f: No test for CLAUDE.md section marker preservation.** Unit 3.2 Acceptance Criteria says "Content outside markers is preserved on re-generation (tested in Phase 6)" but Phase 6 has no explicit unit or acceptance criterion testing CLAUDE.md marker-based preservation. Phase 8 (Migration, Update & Polish) is the logical home for this test, but the dependency chain is unclear -- Phase 6 depends on Phases 3 and 4, not Phase 8.

### 4.2 Cross-Phase Dependency Issues

**GAP-4.2a: Ecosystem module deny rules interface gap.** Phase 4 Unit 3.1 Step 3 says "Integrate deny rules from `reference-deny-rules.md`" but the Phase 1 ecosystem module interface (Unit 1.7) already defines `DenyRules(config ModuleConfig) []string` on every ecosystem module. The correct approach should be to aggregate deny rules from all detected ecosystem modules via the registry, not import from a static reference document. Unit 3.1 needs a step: "Query `ModuleRegistry.DetectedModules()` and aggregate `DenyRules()` from each."

**GAP-4.2b: Managed settings template has no delivery mechanism.** Unit 3.1 mentions generating a managed settings template for `/etc/claude-code/managed-settings.json`, but this requires root access and is a system-level concern. There is no unit in any phase that addresses how this file gets deployed. Phase 9 (Cross-Platform System Detection) handles OS-level operations but does not reference managed settings deployment. Phase 10 (Distribution) also does not address this. The gap is: who deploys the managed settings file, and is it part of `gdev devenv setup` or a separate admin workflow?

**GAP-4.2c: Trail of Bits skills not embedded as dependencies.** Unit 3.4 Step 1 lists standard skills (`deploy.md`, `review-pr.md`, etc.) but Step 6 mentions "Trail of Bits security skills" in Phase Outputs without defining which Trail of Bits skills are embedded vs. referenced. The ecosystem research (Section 5.1) identifies `supply-chain-risk-auditor`, `differential-review`, `insecure-defaults` as the target skills, and the plan overview (line 16) confirms these three. But the unit steps never describe how these are obtained (download? embed from ToB repo? license implications?) or how they interact with the manifest.yaml. Trail of Bits skills are under MIT/CC license (per the ecosystem research) but Unit 3.4 should explicitly state the acquisition mechanism.

**GAP-4.2d: Python runtime dependency undeclared.** Unit 3.3 deploys a Python PreToolUse hook script. The acceptance criterion says "Script has no external dependencies (stdlib-only Python 3)" but there is no check that Python 3 is available in the target environment. Phase 9 (OS prerequisite detection) handles tool prerequisites but has no dependency on Phase 4. The hook script should either degrade gracefully when Python is missing or Unit 3.6 should emit a warning if Python is not detected.

### 4.3 Missing Units

**GAP-4.3a: No attach-guard integration unit.** The plan overview (line 9) says "attach-guard plugin configuration" and the Phase Outputs (line 15) say "Hook deployment: attach-guard plugin configuration + custom PreToolUse hook script." But no unit implements attach-guard configuration. Unit 3.3 only handles the custom Python hook script. attach-guard is a separate plugin with its own installation mechanism (`claude plugin marketplace add attach-dev/attach-guard`) and configuration (Socket.dev API integration, score thresholds). Either Unit 3.3 should be expanded to cover both, or a dedicated Unit 3.7 is needed.

**GAP-4.3b: No security-guidance plugin configuration.** The ecosystem research (Section 1.1) identifies Anthropic's official `security-guidance` plugin as a good baseline for code quality. The plan overview mentions it in the ecosystem table. But no unit generates its installation or configuration.

**GAP-4.3c: No settings.json merge-on-update unit.** Unit 3.1 sets `ThreeWayMerge` as the merge strategy for settings.json, and Unit 3.6 implements `gdev claude update`. But no unit implements the actual three-way merge logic for settings.json. Phase 8 (Migration) handles merge strategies generically, but settings.json has unique challenges: the deny rules list should be unioned (not replaced), user-added allow rules should be preserved, and hooks should be merged by event type. This specific merge logic belongs in Phase 4 or as a Phase 4 prerequisite from Phase 1.

### 4.4 Security Review

**GAP-4.4a: Deny rule bypass via shell wrappers not mitigated.** The ecosystem research (Section 1.4) rates permission deny rules as "Medium" bypass resistance because "shell wrappers bypass" them. For example, a user could create `install.sh` containing `npm install malicious-pkg` and the deny rule for `npm install *` would not fire. Unit 3.1 generates deny rules but has no step addressing known bypass vectors. The unified architecture (`claude-code-agent-package-guardrails/unified-architecture.md`) documents mitigation strategies (e.g., deny `bash *.sh`, deny `sh -c *`) that should be included.

**GAP-4.4b: .mcp.json injection vulnerability not addressed.** The ecosystem research (Section 11, CVE-2025-59536) documents RCE via malicious `.mcp.json` auto-approving servers. Unit 3.5 generates `.mcp.json` but includes no validation that generated server entries are safe, no warning about the vulnerability, and no guidance on `.mcp.json` review. At minimum, generated CLAUDE.md should warn about `.mcp.json` trust implications.

**GAP-4.4c: Hook script integrity not verified.** Unit 3.3 deploys `package-guard.py` to `.claude/hooks/`. A compromised repository could modify this file. The hash tracking system (Phase 1, Unit 1.6) would detect modification, but there is no unit that checks hook integrity at runtime. This is an inherent limitation of the hook system (Claude Code does not verify hook integrity), but it should be documented in generated security docs.

**GAP-4.4d: CLAUDE.md instructions rated "Low" bypass resistance.** The ecosystem research (Section 1.4) rates CLAUDE.md instructions as "Low" bypass resistance. Unit 3.2 generates CLAUDE.md security instructions, but the acceptance criteria treat them as a meaningful security layer without acknowledging this limitation. The generated CLAUDE.md should be positioned as advisory guidance, not enforcement.

### 4.5 Acceptance Criteria Gaps

**GAP-4.5a: Unit 3.1 does not test sandbox configuration.** Step 5 implements sandbox config (`WriteDeny`, `WriteAllow`, `ReadDeny`, `NetAllow`) but no acceptance criterion tests the sandbox section. Add: "Sandbox config generates correct paths when enabled."

**GAP-4.5b: Unit 3.4 does not test manifest.yaml schema.** The manifest YAML is created but never validated against a schema. "manifest.yaml is valid" needs to be specified: valid YAML AND contains required fields (name, description, tags, applicable languages) for every skill entry.

**GAP-4.5c: Unit 3.6 missing error handling tests.** No acceptance criterion tests `gdev claude init` failure modes: invalid `--permission-preset` value, conflicting flags, missing project root, or generation pipeline errors.

---

## Phase 5: Security & Infrastructure Integration

### 5.1 Internal Consistency Issues

**GAP-5.1a: Unit numbering mismatch.** Phase 5 units are numbered 4.1-4.4, using the Phase 4 prefix. This is the same numbering collision as Phase 4 (GAP-4.1b). Phase 5 should use units 5.1-5.4 to avoid confusion.

**GAP-5.1b: Phase Outputs list 9 deliverables but only 4 units.** The Phase Outputs section lists: registry proxy config, Nix cache config, build cache config, pre-commit hooks, CI scanning, nix.conf hardening, SBOM generation, dependency updates, security documentation. But only 4 units exist: (4.1) per-ecosystem package manager configs, (4.2) pre-commit hooks, (4.3) nix.conf hardening, (4.4) CI scanning. Missing units for: registry proxy configuration, Nix binary cache configuration, build cache configuration, SBOM generation configuration, and security documentation generation. These are listed as phase outputs but have no implementation units.

**GAP-5.1c: Unit 4.1 overlaps with Phase 2 ecosystem modules.** Phase 5's preamble says "Per-ecosystem package manager configs (`.npmrc`, `pip.conf`, etc.) are already generated by ecosystem modules in Phase 2 -- this phase adds the infrastructure-level integration." But Unit 4.1 then redefines the exact same per-ecosystem configs that Phase 2 Units 2.1-2.8 already implement. For example, Unit 4.1 Step 1 specifies `.npmrc` with `save-exact=true`, `ignore-scripts=true`, `min-release-age=3` -- these exact same settings appear in Phase 2 Unit 2.1 Step 3. This is a clear duplication. Either Unit 4.1 should ONLY handle infrastructure-level additions (registry proxy URLs, cache auth tokens) that overlay the base configs from Phase 2, or it should be removed with a note that Phase 2 already covers this.

**GAP-5.1d: Unit 4.2 specifies devenv.nix generation but Phase 3 already generates devenv.nix.** Unit 4.2 Step 2 says "Integrate into devenv.nix template: add `git-hooks.hooks.*` section." But Phase 3 Unit 2.2 already generates devenv.nix. The hooks should be part of the ecosystem module interface (`PreCommitHooks()` method) and composed during Phase 3 generation, not injected by Phase 5. The Phase 5 preamble acknowledges this separation but the unit steps contradict it.

### 5.2 Cross-Phase Dependency Issues

**GAP-5.2a: Dependency declaration says Phases 3 and 4, should include Phase 1.** Phase 5's dependency on `InfraProfile` types (Unit 1.8) is not declared. The registry proxy, Nix cache, and build cache configuration all depend on the `InfraProfile` struct from Phase 1 Unit 1.8, but the dependency section only says "Phases 3 and 4 complete."

**GAP-5.2b: Registry proxy configuration has no implementation unit.** The plan overview (line 112-119) describes a complete infrastructure stack with registry proxy as the highest-leverage defense. Phase 1 Unit 1.8 defines `RegistryConfig` with 10 registry types. The artifact stores research provides detailed per-registry configuration. But Phase 5 has NO unit generating registry proxy configuration files. The `.npmrc` in Unit 4.1 only has local security settings, not registry URLs. A sub-agent implementing Phase 5 would produce no registry proxy integration despite it being the cornerstone of the infrastructure stack.

**GAP-5.2c: Build cache configuration has no implementation unit.** Phase 1 Unit 1.8 defines `BuildCacheConfig` (sccache, ccache, turborepo, nx, bazel-remote). The artifact stores research provides detailed configuration. Phase 5 Phase Outputs list "Build cache configuration" but no unit implements it. A sub-agent would have no instructions for generating sccache env vars, `.cargo/config.toml` with `rustc-wrapper`, `.bazelrc` with remote cache, or `turbo.json` with remote cache config.

**GAP-5.2d: Nix cache configuration has no implementation unit.** Same pattern. Phase 1 Unit 1.8 defines `NixCacheConfig` (cachix, attic, nix-serve). Phase 5 Phase Outputs list it. No unit implements it. The `extra-substituters` and `trusted-public-keys` additions to devenv.nix/nix.conf are not generated anywhere.

**GAP-5.2e: SBOM generation has no implementation unit.** Phase Outputs list "SBOM generation configs (Syft, sbomnix)." No unit generates `syft` CI steps, `sbomnix` flake references, or SBOM output paths. The artifact stores research (Section 5) provides detailed configuration for both tools.

**GAP-5.2f: Security documentation generation has no implementation unit.** Phase Outputs list "Security documentation generation (trust model, defense layers, trivy compromise warning)." No unit generates this document. This would be a valuable output explaining the defense-in-depth architecture to developers.

**GAP-5.2g: Dependency update config partially covered.** Unit 4.4 Step 4 mentions "Renovate/Dependabot config snippet with age-gating" but this is a sub-step of CI scanning, not a dedicated unit. The artifact stores research (Sections 4.6, 4.7) provides detailed Renovate and Dependabot configuration. A complete `renovate.json` or `.github/dependabot.yml` generation deserves its own unit or explicit expansion of Unit 4.4.

### 5.3 Missing Units

**GAP-5.3a: Need Unit 4.5 (or 5.5): Registry Proxy Configuration.** Should generate: per-ecosystem registry URLs in `.npmrc`/`pip.conf`/`settings.xml`/`.cargo/config.toml`/`.terraformrc`, authentication credential env vars in devenv.nix, `InfraProfile.Registry` to config file mapping. This is the highest-impact missing unit -- registry proxy is described as the "highest-leverage 'configure once' defense" in the research.

**GAP-5.3b: Need Unit 4.6 (or 5.6): Cache Configuration.** Should generate: Nix cache substituters/keys in devenv.nix/nix.conf, sccache env vars in devenv.nix, turbo.json/nx.json remote cache config, `.bazelrc` remote cache config.

**GAP-5.3c: Need Unit 4.7 (or 5.7): SBOM & Security Documentation.** Should generate: CI workflow steps for Syft SBOM generation, sbomnix integration for Nix projects, security architecture document (`docs/security-overview.md`).

### 5.4 Acceptance Criteria Gaps

**GAP-5.4a: Phase Completion Criteria incomplete.** The phase says "All four units pass acceptance criteria" but the Phase Outputs list 9 deliverables. At least 5 deliverables have no corresponding acceptance criteria at all.

**GAP-5.4b: Unit 4.2 does not test prek compatibility.** The unit mentions "Document prek as the runner (not pre-commit) in generated comments" but no acceptance criterion verifies that the generated hook syntax is compatible with prek specifically, rather than pre-commit. If prek has syntax differences, this could produce non-functional hooks.

**GAP-5.4c: Unit 4.3 acceptance criterion is non-binary.** "Document is standalone-readable for an admin who hasn't seen the research" is subjective. Replace with: "Document includes all 10 settings from nix-conf-hardening-research.md with rationale for each."

**GAP-5.4d: Unit 4.4 does not test Claude Code Security Review Action.** Phase Outputs list it, Phase 4 plan overview mentions it, but Unit 4.4's steps only generate OSV Scanner, Harden-Runner, Socket.dev, and Renovate/Dependabot configs. The Claude Code Security Review GitHub Action is missing from the implementation steps despite being called out in plan.md line 94.

### 5.5 Security Review

**GAP-5.5a: Registry proxy credentials in generated files.** The artifact stores research shows auth tokens being set directly in config files (`.npmrc` with `_authToken`, `pip.conf` with inline URLs). Unit 4.1 (if expanded to include registry proxy URLs) and the missing registry proxy unit must ensure credentials are referenced via environment variables, never hardcoded. Plan.md Principle 6 says "Re-runnable without destruction" but does not address credential hygiene in generated files. Generated `.npmrc` should use `${NPM_REGISTRY_TOKEN}` patterns.

**GAP-5.5b: No credential rotation guidance.** Registry tokens, cache signing keys, and scanning API keys all have rotation requirements. No unit generates documentation about credential lifecycle management.

---

## Phase 6: Wizard & Orchestration (devinit)

### 6.1 Internal Consistency Issues

**GAP-6.1a: Unit numbering mismatch.** Phase 6 units are numbered 5.1-5.6. Same issue as Phases 4 and 5 -- cross-phase references become ambiguous.

**GAP-6.1b: Dependency declaration inconsistency.** Phase 6 Dependencies section says "Phase 4 desirable but not blocking" but the Phase Index in plan.md says Phase 6 depends on "Phases 3, 4." The wizard generates Claude Code configuration (Group 5 in Unit 5.2), which requires Phase 4's generators. If Phase 4 is not complete, Group 5 would collect answers that no generator can process. Either Phase 4 IS a hard dependency, or the wizard needs graceful degradation when claudecode generators are unavailable.

**GAP-6.1c: Group count mismatch.** Phase Outputs say "quick path (1 question) and customize (5 form groups)" but Unit 5.2 defines 6 groups (Group 1: Quick Selection, Group 2: Languages, Group 3: Services, Group 4: Dev Environment, Group 5: Claude Code, Group 6: Plan Preview). The Phase Outputs should say "6 form groups" or clarify that Group 1 and Group 6 are always shown and the "5 form groups" excludes one of them.

**GAP-6.1d: Unit 5.2 says "Groups 2-5 hidden when quick path selected" but Group 6 (Plan Preview) is not conditionally hidden.** The plan preview should always be shown on both paths (so the user sees what will be generated), which means the quick path shows Groups 1 and 6 (2 screens). The acceptance criteria correctly say "Quick path shows only Group 1 and Group 6 (2 screens)" but the step description is ambiguous about Group 6's visibility.

### 6.2 Cross-Phase Dependency Issues

**GAP-6.2a: Infrastructure profile integration gap.** Phase 1 Unit 1.8 defines `InfraProfile` types. Phase 6 Unit 5.4 defines `ProfileRegistry` with built-in profiles (`go-web`, `ts-fullstack`, etc.). But the Unit 5.4 profiles only encode language/service/Claude Code choices -- they do NOT integrate with `InfraProfile` infrastructure choices (registry proxy, caches, scanning). Plan.md line 131 says Phase 6 includes "profile system (including infrastructure profiles)" but Unit 5.4's steps only define language-oriented profiles. The `--profile consulting-default` use case from plan.md (which should configure Nexus + Cachix + sccache + OSV Scanner + Renovate) has no implementation path.

**GAP-6.2b: Phase 5 security configs not wired into orchestration.** Unit 5.1 describes the orchestration flow: "detect -> wizard -> generate (devenv) -> generate (claudecode) -> generate (security configs) -> write all -> report." Step 2 includes "generate (security configs)" but Phase 5's units are not connected to any generator interface. Phase 3 and Phase 4 each have a `Generator.Generate(answers)` pattern, but Phase 5 units produce files via ad-hoc generation functions. Either Phase 5 needs a generator that implements the `Generator` interface, or Unit 5.1 needs to explicitly call Phase 5's generation functions.

**GAP-6.2c: Detection engine does not feed Phase 5 decisions.** Unit 5.3 wires detection into the wizard for language/service pre-population. But detection results also drive Phase 5 decisions: which pre-commit hooks to enable, which CI scanning to configure, which registry proxy ecosystems to configure. There is no step mapping `DetectedProject` to Phase 5's infrastructure configuration.

**GAP-6.2d: WizardAnswers struct incomplete for Phase 5.** Phase 1 Unit 1.2 defines `WizardAnswers` with fields for languages, services, Claude Code settings, etc. But there are no fields for infrastructure choices: registry proxy selection, cache configuration, scanning tool preferences, SBOM generation toggle, pre-commit tier selection. Either `WizardAnswers` needs expansion or infrastructure choices live elsewhere (e.g., `InfraProfile`).

### 6.3 Missing Units

**GAP-6.3a: No error recovery / back navigation unit.** The huh library supports `WithKeyMap()` for custom key bindings. But no unit addresses: what happens if the user wants to go back a group? What if generation fails mid-way -- does the wizard re-run? What if detection produces incorrect results and the user wants to override? The wizard flow is strictly forward-only as designed. At minimum, the plan should document this as a known limitation or add error handling to Unit 5.2.

**GAP-6.3b: No wizard state persistence unit.** If the wizard is interrupted (Ctrl+C, terminal crash, SSH disconnect), all answers are lost. For a wizard that could take 30+ seconds in customize mode, this is a UX gap. No unit saves intermediate wizard state to disk. Mitigation: either add a unit for state persistence or document that `--profile` + `--yes` is the recovery path.

**GAP-6.3c: No cancel/abort flow.** No unit handles Ctrl+C during wizard execution. huh forms return an error on cancel, but no unit specifies cleanup behavior: should partial files be deleted? Should a `.gdev-init-incomplete` marker be left? Phase 1's generation pipeline (Unit 1.5) writes atomically, but the wizard has no unit addressing what "atomic" means for the full orchestration pipeline (detect + wizard + generate + write).

**GAP-6.3d: No plan preview implementation unit.** Unit 5.2 Step 7 mentions "Group 6: Plan Preview & Confirm" but no unit implements the plan preview rendering. Phase 1 Unit 1.5 has `PreviewFiles(files []GeneratedFile) string` but the wizard needs to generate the file list before the user confirms. This means the wizard must run generators speculatively (dry-run) to produce the preview, but no unit describes this speculative generation step.

**GAP-6.3e: No merge mode implementation unit.** Phase Outputs list "Merge mode for existing projects." Unit 5.3 mentions "When existing config detected, set merge mode flag and adjust wizard messaging" but no unit implements merge mode behavior. What does "merge mode" concretely do? Does it show a diff? Ask per-file overwrite/skip/merge? Use the hash tracking from Phase 1 Unit 1.6 to determine which files are user-modified? This is a complex feature with no implementation details.

**GAP-6.3f: No accessibility testing unit.** Unit 5.2 Step 10 mentions "Accessibility mode via `ACCESSIBLE` or `NO_COLOR` env var detection" and the acceptance criteria include "Accessibility mode works when `ACCESSIBLE` env var set." But there is no unit or step defining what accessibility mode looks like (plain text fallback? screen reader announcements? reduced animation?). huh has an `accessible` option but its behavior must be verified and potentially customized.

### 6.4 Wizard UX Gaps

**GAP-6.4a: Terminal compatibility not tested.** No acceptance criterion tests the wizard in: SSH sessions, tmux/screen, VS Code integrated terminal, CI containers (where TTY may not be available). Unit 5.5 handles non-interactive mode for CI, but there is no unit ensuring the interactive wizard works across common terminal emulators.

**GAP-6.4b: Wizard timeout not defined.** The plan says "quick path <5 seconds, customizers <30 seconds" (plan.md line 53) but no unit implements or tests timing. The 60-second end-to-end target (plan.md line 216) includes generation time, but wizard response time is not constrained or measured.

**GAP-6.4c: Theme configuration via profile not implemented.** Unit 5.2 Step 9 says "Apply theme from config (default: Dracula or team-configured)" but Unit 5.4 (Profile System) has no theme field. The `InfraProfile` struct (Phase 1 Unit 1.8) also has no theme field. Where does theme configuration live?

### 6.5 Profile System Gaps

**GAP-6.5a: Profile composition model undefined.** Unit 5.4 Step 5 says "profile as base, flags as overrides" but does not define the override semantics. If `--profile go-web` sets `PermissionLevel: standard` and the user also passes `--claude-permissions minimal`, which wins? Array fields (languages, services) need merge semantics: does `--service mongodb` replace or append to the profile's services? These semantics must be defined for a sub-agent to implement.

**GAP-6.5b: No profile validation.** If a team defines a profile referencing a Tier 3 ecosystem module (e.g., Elixir) before Phase 7 ships those modules, the profile would fail at generation time. No unit validates that a profile's requirements are satisfiable by the currently available ecosystem modules.

**GAP-6.5c: Profile versioning absent.** Profiles compiled into the binary will change between gdev versions. If a team saves their config with `gdev init --profile go-web` using gdev v1.0, then updates to gdev v1.1 which changes the `go-web` profile, `gdev init --update` would regenerate with the new profile defaults rather than the originally-selected ones. The saved `GeneratedState` from Phase 1 Unit 1.6 records file hashes but not the profile version used.

### 6.6 Forward Compatibility Issues

**GAP-6.6a: Phase 11 wizard integration not accounted for.** Phase 11 Unit 11.4 adds an "AI Agent Tools" form group to the wizard. Phase 6 Unit 5.2 defines Groups 1-6 with no extension point. Unit 11.4 says "Group 5b or merged into existing Claude Code group" -- this requires modifying Unit 5.2's form construction. Phase 6 should define an extension mechanism (e.g., a slice of form group providers) so Phase 11 can add groups without modifying Phase 6 code.

**GAP-6.6b: WizardAnswers struct not extensible for Phase 11.** Phase 11 Unit 11.4 extends `WizardAnswers` with `AgentToolsAnswers`. But the struct is defined in Phase 1 Unit 1.2 and would need to be modified. If the struct is not designed for extension (e.g., via an embedded interface or map[string]interface{} bag), Phase 11 creates a backward-incompatible change to a Phase 1 type.

**GAP-6.6c: Version-Sentinel hook interaction with Phase 4 hooks.** Phase 11 Unit 11.2 Step 9 says "Version-Sentinel uses different matchers (Edit|Write|MultiEdit for manifests, Bash for install commands) so hooks don't conflict." But Phase 4 Unit 3.3's hook also fires on Bash PreToolUse events. Both hooks intercept `npm install`-style commands. The claim of "no conflict" needs verification: do both hooks fire? In what order? Does the first hook's `updatedInput` affect the second hook's input? No unit in any phase defines hook execution ordering or documents the interaction model.

---

## Cross-Phase Findings

### CP-1: Pervasive Unit Numbering Collision

All three phases use numbering that conflicts with other phases. Phase 4 uses 3.x, Phase 5 uses 4.x, Phase 6 uses 5.x. This appears to be because numbering follows the addon's internal unit count rather than the phase number. Every cross-phase reference becomes ambiguous. Recommendation: renumber to match phase numbers (Phase 4 = 4.1-4.6, Phase 5 = 5.1-5.4+, Phase 6 = 6.1-6.6) or adopt a hierarchical scheme (P4-U1 through P4-U6).

### CP-2: Phase 5 is Severely Underspecified

Phase 5 promises 9 deliverables in its Phase Outputs but only has 4 implementation units. The missing 5 areas (registry proxy, Nix cache, build cache, SBOM, security docs) are critical to the plan's value proposition. The consulting firm profile ($0/mo stack) cannot be implemented without registry proxy and cache configuration units. This is the most serious gap in the analysis -- it affects the plan's core value proposition of "one command, working environment" with infrastructure integration.

### CP-3: Per-Ecosystem Config Duplication Between Phases 2 and 5

Phase 2 ecosystem modules each implement `SecurityConfigs()` returning `.npmrc`, `pip.conf`, etc. Phase 5 Unit 4.1 re-specifies the same configs. The boundary between them is unclear. Proposed resolution: Phase 2 modules generate base security configs (age-gating, script blocking, lockfile enforcement). Phase 5 overlays infrastructure-specific additions (registry proxy URLs, auth tokens). Unit 4.1 should be retitled "Infrastructure Overlay for Package Manager Configs" and only add the delta.

### CP-4: Generator Interface Not Implemented by Phase 5

Phases 3 and 4 implement the `Generator` interface from Phase 1 Unit 1.2 (`Generate(answers WizardAnswers) ([]GeneratedFile, error)`). Phase 5 has no generator implementation. Phase 6 Unit 5.1 orchestrates "generate (security configs)" in its pipeline but has nothing to call. Either Phase 5 units should collectively implement a `SecurityGenerator` implementing the `Generator` interface, or Phase 6 Unit 5.1 needs specific integration code for Phase 5.

### CP-5: Hook Architecture Has No Ordering Specification

Three sources of PreToolUse hooks exist:
1. Phase 4 Unit 3.3: `package-guard.py` -- fires on Bash commands, checks OSV + age
2. Phase 4 Unit 3.1: deny rules in settings.json -- fires on matching glob patterns
3. Phase 11 Unit 11.2: Version-Sentinel -- fires on Bash + Edit/Write/MultiEdit

No phase defines:
- Hook execution order (does deny rules fire before PreToolUse hooks?)
- What happens when multiple hooks fire on the same event (are they all consulted? first-to-deny wins?)
- Whether `updatedInput` from one hook is passed to the next hook
- Whether a hook's "allow" decision can be overridden by a later hook's "deny"

The Claude Code documentation should be consulted and the interaction model documented. If hooks run sequentially with pass-through, `updatedInput` from package-guard.py (injecting `--ignore-scripts`) would flow to Version-Sentinel, which is correct. If they run independently against the original input, the `--ignore-scripts` injection would be lost. This architectural question affects the security model.

### CP-6: Phase 6 Wizard Cannot Preview Phase 5 Outputs

Unit 5.2 Group 6 is "Plan Preview" showing files that will be generated. Unit 5.1 orchestrates generation in order: devenv -> claudecode -> security configs. But if security config generation (Phase 5) has no `Generator` interface, the plan preview cannot include Phase 5 outputs. The user would see devenv and Claude Code files in the preview but not registry configs, pre-commit hooks, or CI workflows.

### CP-7: Infrastructure Profiles Are Split Across Two Incompatible Systems

Phase 1 Unit 1.8 defines `InfraProfile` (registry, cache, scanning, SBOM). Phase 6 Unit 5.4 defines `ProfileRegistry` with language-oriented profiles (`go-web`, `ts-fullstack`). These are separate systems with no integration. The plan overview's `gdev init --profile consulting-default --yes` (line 62) implies a single unified profile system, but the implementation splits profiles into infra profiles and language profiles. Either `Profile` should embed `InfraProfile`, or the profile system needs a composition model (e.g., `gdev init --profile go-web --infra-profile consulting-default`).

---

## Prioritized Recommendations

### Critical (blocks core value proposition)

1. **Add 5 missing Phase 5 units** (GAP-5.3a, 5.3b, 5.3c, 5.2b-f). Registry proxy, Nix cache, build cache, SBOM, and security documentation generation are all listed as Phase Outputs with no implementation units. Without these, the infrastructure profile system is dead code and the "$0/mo consulting stack" cannot be generated.

2. **Resolve Phase 2 / Phase 5 config duplication** (CP-3). Define a clear boundary: Phase 2 = base security configs, Phase 5 = infrastructure overlay. Retitle Phase 5 Unit 4.1 and remove the duplicated config specifications.

3. **Implement Phase 5 Generator interface** (CP-4). Phase 5 needs a generator so Phase 6's orchestration pipeline can call it and the plan preview can include its outputs.

4. **Unify the profile system** (CP-7). Merge `InfraProfile` into the `Profile` struct or define a profile composition model. The wizard must be able to set infrastructure choices, not just language/service choices.

### High (affects implementation correctness)

5. **Fix unit numbering across all three phases** (CP-1). Renumber to match phase numbers to eliminate cross-phase reference ambiguity.

6. **Add attach-guard integration unit** (GAP-4.3a). The plan overview promises it; no unit implements it.

7. **Define hook execution ordering** (CP-5). Document whether hooks run sequentially or independently, and how `updatedInput` flows between them. This affects the security model.

8. **Add merge mode implementation** (GAP-6.3e). Phase 6 lists merge mode as a Phase Output but no unit defines the behavior.

9. **Add plan preview implementation** (GAP-6.3d). The wizard Group 6 requires speculative generation that no unit describes.

10. **Wire WizardAnswers to Phase 5 decisions** (GAP-6.2d). Add infrastructure choice fields to `WizardAnswers` or define how `InfraProfile` flows through the wizard.

### Medium (affects quality and completeness)

11. **Add settings.json schema validation** (GAP-4.1d). Validate generated settings against Claude Code's schema or embed a version marker.

12. **Define hooks section schema** (GAP-4.1e). Specify the exact settings.json hooks schema that Units 3.1 and 3.3 both need.

13. **Add wizard extension mechanism for Phase 11** (GAP-6.6a). Design form group providers so Phase 11 can add groups without modifying Phase 6.

14. **Add error/cancel handling** (GAP-6.3a, 6.3c). Define behavior for Ctrl+C, back navigation, and generation failure.

15. **Add Claude Code Security Review Action** (GAP-5.4d). Phase Outputs promise it; Unit 4.4 steps omit it.

16. **Fix Phase 4 internal references** (GAP-4.1c). Unit 3.5's "Phase 4" reference should say "Phase 5."

17. **Clean up duplicate Phase Outputs text** (GAP-4.1a). Remove the repeated lines 20-24 in Phase 4.

### Low (polish and hardening)

18. **Address deny rule bypass vectors** (GAP-4.4a). Include shell wrapper mitigations from the unified architecture document.

19. **Document .mcp.json trust implications** (GAP-4.4b). Add CVE-2025-59536 warning to generated docs.

20. **Add Python runtime check for hook script** (GAP-4.2d). Either degrade gracefully or warn during generation.

21. **Define profile composition semantics** (GAP-6.5a). Specify flag-overrides-profile behavior for scalar and array fields.

22. **Add profile validation** (GAP-6.5b). Verify profile ecosystem requirements are satisfiable.

23. **Add accessibility mode specification** (GAP-6.3f). Define what `ACCESSIBLE` mode concretely changes.

24. **Add credential hygiene requirement** (GAP-5.5a). All generated config files must use env var references, never inline credentials.

25. **Document CLAUDE.md advisory-only nature** (GAP-4.4d). Ensure generated security docs position CLAUDE.md as advisory, not enforcement.
