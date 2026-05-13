# Phase 22: Agentic Skills, Compliance & Developer Experience Validation

## Goal

Validate all features from Phases 13-16 through comprehensive E2E testing: project configuration and onboarding modes, Claude Code skills and agent files, health/compliance reporting accuracy, and developer experience commands (repair, info, outdated, update, teardown). This phase proves that the entire "upper stack" of gdev — configuration management, agentic integration, compliance reporting, and developer polish — works correctly end-to-end across realistic project scenarios.

## Dependencies

Phase 17 complete (test infrastructure framework — testscript E2E framework, custom commands like `yaml_has`, `json_path`, `nix_valid`, golden file infrastructure, CI pipeline). Phases 13-16 complete (features under test — project configuration, Claude Code integration, health/compliance reporting, developer experience commands).

## Phase Outputs

- Configuration and onboarding validation test suite (all 4 modes tested with testscript scenarios)
- Skill and agent file validation test suite (parsing, invocation, context budget verification)
- Deny rule conflict matrix regression test suite
- Health and compliance reporting accuracy test suite (scoring, conformance, SARIF, badges)
- Developer experience command validation test suite (repair, info, outdated, update, teardown)
- Team workflow and CI integration validation test suite (aggregation, git workflow, profiles)

---

### Unit 22.1: Configuration & Onboarding Validation

**Description:** Validate `.gdev.yaml` parsing, three-layer configuration resolution, four onboarding modes (Create/Join/Update/Repair), config version constraints, and `gdev check` accuracy using testscript E2E scenarios. Each scenario creates a realistic project state and verifies gdev responds correctly to valid, invalid, and edge-case configurations.

**Context:** Phase 13 introduced the most complex configuration machinery in the system: a YAML schema with nested structs, a three-layer deep-merge engine with security floor enforcement, four onboarding modes with automatic detection, semver constraint parsing, and a 5-category CI enforcement command. These features interact in subtle ways — a malformed `.gdev.yaml` should produce clear errors, local overrides must not lower security below the project floor, and mode detection must be deterministic based on project state. Unit tests in Phase 13 cover individual functions; this phase tests the full flow through the compiled binary with real file I/O, real YAML parsing, and real mode routing.

**Desired Outcome:** A test suite that catches regressions in config parsing, merge semantics, mode detection, version constraint enforcement, and `gdev check` accuracy. Every onboarding mode is exercised with a self-contained testscript scenario including fixture files.

**Steps:**
1. Create `e2e/testdata/script/config/` directory for configuration test scripts.
2. Write `valid-config-parse.txtar` — test valid `.gdev.yaml` parsing:
   ```
   # Valid .gdev.yaml parses without error
   exec gdev check --format json
   stdout '"status":"pass"'

   -- .gdev.yaml --
   version: 1
   gdev_version: ">= 0.15.0"
   profile: consulting-default
   languages:
     - name: go
       version: "1.22"
   security:
     level: enhanced
   ```
3. Write `invalid-config.txtar` — test error messages for malformed configs:
   ```
   # Missing version field produces clear error
   ! exec gdev check
   stderr 'must include a `version` field'

   -- .gdev.yaml --
   profile: consulting-default
   ```
4. Write `unknown-version.txtar` — test future config version handling:
   ```
   # Unknown config version produces upgrade error
   ! exec gdev check
   stderr 'gdev self-update'
   stderr 'version 99'

   -- .gdev.yaml --
   version: 99
   ```
5. Write `three-layer-merge.txtar` — test merge correctness with all layers present:
   ```
   # Three-layer merge: org defaults overridden by project, then local
   exec gdev init --non-interactive --verbose
   stdout 'security.level = "enhanced" (from project:.gdev.yaml'

   exec gdev check --format json
   json_path '.checks' '.name=="config_integrity"' '.status' 'pass'

   -- .gdev.yaml --
   version: 1
   security:
     level: enhanced
   -- .gdev.local.yaml --
   extra_packages:
     - neovim
   ```
6. Write `security-floor.txtar` — verify local overrides cannot lower security below project level:
   ```
   # Security floor enforcement: local cannot lower security level
   exec gdev init --non-interactive --verbose
   stdout 'floor enforced'
   ! stdout 'security.level = "baseline"'

   exec gdev check --format json
   json_path '.checks' '.name=="security_hardening"' '.status' 'pass'

   -- .gdev.yaml --
   version: 1
   security:
     level: enhanced
     age_gating: true
   -- .gdev.local.yaml --
   security:
     level: baseline
     age_gating: false
   ```
7. Write `mode-create.txtar` — verify Create mode on empty project (no `.gdev.yaml`):
   ```
   # Create mode: no .gdev.yaml triggers full wizard
   exec gdev init --non-interactive --answers-file answers.yaml
   stdout 'No .gdev.yaml found'
   exists .gdev.yaml
   exists devenv.nix
   exists devenv.yaml

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
8. Write `mode-join.txtar` — verify Join mode on project with existing `.gdev.yaml`:
   ```
   # Join mode: .gdev.yaml exists, no state dir
   exec gdev init --non-interactive
   stdout 'Join mode'
   stdout 'Setting up as new team member'
   exists .gdev.local.yaml

   -- .gdev.yaml --
   version: 1
   profile: consulting-default
   languages:
     - name: go
   ```
9. Write `mode-update.txtar` — verify Update mode when binary version is newer than last run:
   ```
   # Update mode: state dir exists, version mismatch
   exec gdev init --non-interactive
   stdout 'Template updates available'

   -- .gdev.yaml --
   version: 1
   languages:
     - name: go
   -- .devinit/.gdev-init-state.yaml --
   gdev_version: "0.1.0"
   files: {}
   ```
10. Write `mode-repair.txtar` — verify Repair mode when files have drifted:
    ```
    # Repair mode: generated files have drifted
    exec gdev init --non-interactive
    stdout 'drifted from expected state'

    -- .gdev.yaml --
    version: 1
    languages:
      - name: go
    -- .devinit/.gdev-init-state.yaml --
    gdev_version: "0.16.0"
    files:
      devenv.yaml:
        hash: "sha256:0000000000000000000000000000000000000000000000000000000000000000"
        category: machine-owned
    -- devenv.yaml --
    this file has been corrupted
    ```
11. Write `gdev-version-constraint.txtar` — verify `gdev_version` enforcement rejects incompatible binary:
    ```
    # gdev_version constraint rejects incompatible binary
    ! exec gdev check
    stderr 'gdev version mismatch'
    stderr 'gdev self-update'

    -- .gdev.yaml --
    version: 1
    gdev_version: ">= 99.0.0"
    ```
12. Write `gdev-check-accuracy.txtar` — verify `gdev check` with known-good and known-bad states:
    ```
    # gdev check passes on well-configured project
    exec gdev init --non-interactive --answers-file answers.yaml
    exec gdev check --format json
    json_path '.summary.fail' '0'

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
13. Write `profile-inheritance.txtar` — verify org -> profile -> project -> local chain:
    ```
    # Profile inheritance chain resolves correctly
    exec gdev init --non-interactive --verbose
    stdout 'profile:consulting-default'
    exec gdev check --format json
    json_path '.checks' '.name=="required_tools"' '.status' 'pass'

    -- .gdev.yaml --
    version: 1
    profile: consulting-default
    tools:
      enabled:
        - changelog
    ```

**Acceptance Criteria:**
- [ ] Valid `.gdev.yaml` parses without error and `gdev check` passes
- [ ] Missing `version` field produces clear, actionable error message
- [ ] Unknown config version produces error with `gdev self-update` instructions
- [ ] Unknown YAML fields are silently ignored (forward compatibility verified)
- [ ] Three-layer merge correctness verified: binary defaults < project config < local overrides
- [ ] Security floor enforcement verified: local config cannot lower security level below project setting
- [ ] Security floor enforcement verified: local config cannot disable security features (age_gating, script_blocking) that project enables
- [ ] Create mode detected and runs wizard when no `.gdev.yaml` exists
- [ ] Join mode detected when `.gdev.yaml` exists but no local state directory
- [ ] Update mode detected when gdev binary version is newer than last run
- [ ] Repair mode detected when generated files have drifted from expected hashes
- [ ] `gdev_version` constraint rejects incompatible binary with actionable error
- [ ] `gdev check` passes on a well-configured project and fails on a misconfigured one
- [ ] Profile inheritance chain (org -> profile -> project -> local) resolves correctly

**Research Citations:**
- `phases/13-project-configuration-team-standards.md § Unit 13.1` — `.gdev.yaml` schema, parsing, validation
- `phases/13-project-configuration-team-standards.md § Unit 13.2` — three-layer resolution, security floor enforcement
- `phases/13-project-configuration-team-standards.md § Unit 13.3` — onboarding mode detection and routing
- `phases/13-project-configuration-team-standards.md § Unit 13.5` — `gdev_version` constraint, semver parsing
- `phases/13-project-configuration-team-standards.md § Unit 13.6` — `gdev check` command, 5 check categories
- `phases/13-project-configuration-team-standards.md § Unit 13.7` — profile inheritance, compliance levels

**Status:** Not Started

---

### Unit 22.2: Skill & Agent File Validation

**Description:** Validate that all generated skill and agent files are well-formed, correctly configured, and functional. Test YAML frontmatter parsing, `disable-model-invocation` correctness, dynamic context injection output, context budget compliance, and Claude Code's ability to discover and load all skills and agents.

**Context:** Phase 14 deploys 10 gdev operation skills, 7 consulting agents, and 8+ consulting skills — 25+ files that Claude Code parses at startup. A single YAML syntax error in any file breaks skill discovery. The `disable-model-invocation` flag is the primary safety mechanism preventing Claude from autonomously running side-effect operations — a missing flag means Claude could silently run `gdev init` or `gdev enable` without user intent. Dynamic context injection (`` !`command` ``) must produce valid output both when gdev is installed and when it is not (the `|| echo` fallback). Context budget management ensures CLAUDE.md stays under the 5% target for both Sonnet (200K) and Opus (1M) context windows. These are not functional tests of skill execution — they are structural validation tests ensuring the generated files meet their format contract.

**Desired Outcome:** A regression suite that catches malformed skill files, missing safety flags, broken dynamic context injection, and context budget overruns before they reach developers.

**Steps:**
1. Create `e2e/testdata/script/skills/` directory for skill validation scripts.
2. Write `gdev-skill-parse.txtar` — verify all 10 gdev operation skill files parse correctly:
   ```
   # All gdev operation skill files have valid YAML frontmatter
   exec gdev init --non-interactive --answers-file answers.yaml
   exec find .claude/skills/gdev-* -name SKILL.md

   # Verify each skill file exists and has valid frontmatter
   exists .claude/skills/gdev-init/SKILL.md
   exists .claude/skills/gdev-onboard/SKILL.md
   exists .claude/skills/gdev-setup/SKILL.md
   exists .claude/skills/gdev-enable/SKILL.md
   exists .claude/skills/gdev-disable/SKILL.md
   exists .claude/skills/gdev-update/SKILL.md
   exists .claude/skills/gdev-doctor/SKILL.md
   exists .claude/skills/gdev-status/SKILL.md
   exists .claude/skills/gdev-tools/SKILL.md
   exists .claude/skills/gdev-detect/SKILL.md

   # Verify YAML frontmatter parses (each file starts with ---)
   exec grep -l '^---' .claude/skills/gdev-init/SKILL.md
   exec grep -l '^---' .claude/skills/gdev-doctor/SKILL.md

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
3. Write `agent-file-parse.txtar` — verify all 7 consulting agent files parse correctly:
   ```
   # All consulting agent files have valid YAML frontmatter
   exec gdev init --non-interactive --answers-file answers.yaml

   exists .claude/agents/security-reviewer.md
   exists .claude/agents/codebase-explorer.md
   exists .claude/agents/test-gap-analyzer.md
   exists .claude/agents/onboarding-guide.md
   exists .claude/agents/migration-planner.md
   exists .claude/agents/handoff-doc-generator.md
   exists .claude/agents/incident-debugger.md

   # Verify frontmatter contains required fields
   exec grep 'name:' .claude/agents/security-reviewer.md
   exec grep 'description:' .claude/agents/security-reviewer.md
   exec grep 'disallowedTools:' .claude/agents/security-reviewer.md

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
4. Write `consulting-skill-parse.txtar` — verify all 8+ consulting skill files parse correctly:
   ```
   # All consulting workflow skill files have valid YAML frontmatter
   exec gdev init --non-interactive --answers-file answers.yaml

   exists .claude/skills/review-pr/SKILL.md
   exists .claude/skills/add-tests/SKILL.md
   exists .claude/skills/upgrade-dep/SKILL.md
   exists .claude/skills/onboard-me/SKILL.md
   exists .claude/skills/write-adr/SKILL.md
   exists .claude/skills/incident-debug/SKILL.md
   exists .claude/skills/migration-plan/SKILL.md
   exists .claude/skills/handoff-doc/SKILL.md

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
5. Write `user-only-skills.txtar` — verify user-only skills have `disable-model-invocation: true`:
   ```
   # Side-effect skills must have disable-model-invocation: true
   exec gdev init --non-interactive --answers-file answers.yaml

   # User-only gdev operations (side effects)
   exec grep 'disable-model-invocation: true' .claude/skills/gdev-init/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/gdev-onboard/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/gdev-setup/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/gdev-enable/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/gdev-disable/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/gdev-update/SKILL.md

   # User-only consulting workflows
   exec grep 'disable-model-invocation: true' .claude/skills/review-pr/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/add-tests/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/upgrade-dep/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/onboard-me/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/write-adr/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/incident-debug/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/migration-plan/SKILL.md
   exec grep 'disable-model-invocation: true' .claude/skills/handoff-doc/SKILL.md

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
6. Write `claude-invocable-skills.txtar` — verify Claude-invocable skills do NOT have `disable-model-invocation`:
   ```
   # Read-only diagnostic skills must NOT have disable-model-invocation
   exec gdev init --non-interactive --answers-file answers.yaml

   ! exec grep 'disable-model-invocation' .claude/skills/gdev-doctor/SKILL.md
   ! exec grep 'disable-model-invocation' .claude/skills/gdev-status/SKILL.md
   ! exec grep 'disable-model-invocation' .claude/skills/gdev-tools/SKILL.md
   ! exec grep 'disable-model-invocation' .claude/skills/gdev-detect/SKILL.md

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
7. Write `dynamic-context-injection.txtar` — verify dynamic context injection produces valid output with and without gdev:
   ```
   # Dynamic context injection fallbacks work when gdev commands fail
   # The !`command` preprocessor should produce valid output even without a running gdev
   exec gdev init --non-interactive --answers-file answers.yaml

   # Verify all skill files contain || echo fallback patterns
   exec grep -l '|| echo' .claude/skills/gdev-init/SKILL.md
   exec grep -l '|| echo' .claude/skills/gdev-doctor/SKILL.md
   exec grep -l '|| echo' .claude/skills/gdev-status/SKILL.md
   exec grep -l '|| echo' .claude/skills/gdev-tools/SKILL.md
   exec grep -l '|| echo' .claude/skills/gdev-detect/SKILL.md

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
8. Write `context-budget-sonnet.txtar` — verify context budget for Sonnet (200K) target:
   ```
   # Context budget: Sonnet target stays under 5% (10K tokens ~= 40K chars)
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev check --context-budget --format json
   json_path '.budgetPct' '<5.0'
   json_path '.claudeMDLines' '<=50'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   model: sonnet
   ```
9. Write `context-budget-opus.txtar` — verify context budget for Opus (1M) target:
   ```
   # Context budget: Opus target stays under 5% (50K tokens ~= 200K chars)
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev check --context-budget --format json
   json_path '.budgetPct' '<5.0'
   json_path '.claudeMDLines' '<=100'

   -- answers.yaml --
   ecosystems: [go, javascript-typescript]
   profile: consulting-default
   quick_mode: true
   model: opus
   ```
10. Write `skill-description-length.txtar` — verify all skill descriptions are within budget:
    ```
    # All skill descriptions must be <= 200 characters
    exec gdev init --non-interactive --answers-file answers.yaml

    # Extract description fields and verify length
    exec gdev check --context-budget --format json
    json_path '.skillDescChars' '<=3600'

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
11. Write `claude-code-loads-skills.txtar` — verify Claude Code can list all deployed skills:
    ```
    [claude-cli]
    # Claude Code discovers all gdev-deployed skills
    exec gdev init --non-interactive --answers-file answers.yaml
    exec claude --print-skills 2>&1
    stdout 'gdev-init'
    stdout 'gdev-doctor'
    stdout 'review-pr'
    stdout 'add-tests'

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```

**Acceptance Criteria:**
- [ ] All 10 gdev operation skill files parse correctly (valid YAML frontmatter, no syntax errors)
- [ ] All 7 consulting agent files parse correctly with required fields (`name`, `description`)
- [ ] All 8+ consulting skill files parse correctly (valid YAML frontmatter)
- [ ] 6 user-only gdev operation skills have `disable-model-invocation: true`
- [ ] 8 consulting workflow skills have `disable-model-invocation: true`
- [ ] 4 Claude-invocable diagnostic skills do NOT have `disable-model-invocation`
- [ ] Dynamic context injection includes `|| echo` fallbacks in all skill files using `!`command``
- [ ] Context budget for Sonnet (200K) stays under 5% with CLAUDE.md <= 50 lines
- [ ] Context budget for Opus (1M) stays under 5% with CLAUDE.md <= 100 lines
- [ ] All skill descriptions are <= 200 characters
- [ ] Claude Code can discover and list all deployed skills (when `claude` CLI is available)
- [ ] `security-reviewer` agent has `disallowedTools: Write, Edit` (read-only analysis)
- [ ] `codebase-explorer` agent uses `model: haiku` for fast exploration

**Research Citations:**
- `phases/14-claude-code-integration-agentic-skills.md § Unit 14.1` — 10 gdev operation skills, user-only vs Claude-invocable split
- `phases/14-claude-code-integration-agentic-skills.md § Unit 14.2` — 7 consulting agents, frontmatter fields, `disallowedTools`
- `phases/14-claude-code-integration-agentic-skills.md § Unit 14.3` — 8 consulting skills, `disable-model-invocation: true`
- `phases/14-claude-code-integration-agentic-skills.md § Unit 14.4` — context budget management, Sonnet/Opus targets, 5% threshold
- `phases/14-claude-code-integration-agentic-skills.md § Unit 14.5` — deny rule conflict validation
- `research-spikes/gdev-claude-code-integration/claude-code-integration-research.md § 6` — 5-layer safety architecture

**Status:** Not Started

---

### Unit 22.3: Deny Rule Conflict Matrix Testing

**Description:** Exhaustively test every generated deny rule against every operation used by generated skills. Verify that dangerous operations (package installation, script execution) are correctly blocked while legitimate operations (build, test, lint) are NOT blocked. This unit produces a regression suite that runs on every skill or deny rule change.

**Context:** Phase 14 Unit 14.5 implements `ValidateDenyRuleConflicts()` for compile-time validation. This E2E test unit exercises the deny rules through the actual Claude Code permission system, testing real glob pattern matching against real command strings. The core tension: `Bash(npm install *)` must block unauthorized package installs, but must NOT block `npm test`, `npm run build`, or `npm audit`. An overly broad `Bash(npm *)` would break the entire `/review-pr` and `/add-tests` workflow. The 48+ deny rules covering 15+ package managers each need testing against the build/test/lint commands from all 27 ecosystems.

**Desired Outcome:** A comprehensive conflict matrix test suite that catches any deny rule regression — whether a new deny rule accidentally blocks a legitimate skill operation, or a new skill introduces an operation that conflicts with existing deny rules.

**Steps:**
1. Create `e2e/testdata/script/deny-rules/` directory for deny rule tests.
2. Write `positive-controls.txtar` — verify dangerous operations are correctly blocked:
   ```
   # Positive controls: dangerous operations must be blocked by deny rules
   exec gdev init --non-interactive --answers-file answers.yaml

   # Extract deny rules from settings.json
   exec gdev check --deny-rules --format json
   json_path '.denyRules' 'contains' 'Bash(npm install *)'
   json_path '.denyRules' 'contains' 'Bash(pip install *)'
   json_path '.denyRules' 'contains' 'Bash(cargo install *)'
   json_path '.denyRules' 'contains' 'Bash(go install *)'
   json_path '.denyRules' 'contains' 'Bash(curl * | sh)'
   json_path '.denyRules' 'contains' 'Bash(curl * | bash)'
   json_path '.denyRules' 'contains' 'Bash(npx *)'

   -- answers.yaml --
   ecosystems: [go, javascript-typescript, python]
   profile: consulting-default
   quick_mode: true
   ```
3. Write `negative-controls.txtar` — verify legitimate operations are NOT blocked:
   ```
   # Negative controls: build/test/lint operations must NOT be blocked
   exec gdev init --non-interactive --answers-file answers.yaml

   # Run conflict validator and verify zero unexpected conflicts
   exec gdev check --deny-rules --format json
   json_path '.unexpectedConflicts' '0'

   # Verify specific legitimate operations are not in deny rules
   ! json_path '.denyRules' 'contains' 'Bash(npm test *)'
   ! json_path '.denyRules' 'contains' 'Bash(npm run *)'
   ! json_path '.denyRules' 'contains' 'Bash(npm audit *)'
   ! json_path '.denyRules' 'contains' 'Bash(go test *)'
   ! json_path '.denyRules' 'contains' 'Bash(go build *)'
   ! json_path '.denyRules' 'contains' 'Bash(pytest *)'
   ! json_path '.denyRules' 'contains' 'Bash(cargo test *)'
   ! json_path '.denyRules' 'contains' 'Bash(cargo build *)'
   ! json_path '.denyRules' 'contains' 'Bash(dotnet test *)'
   ! json_path '.denyRules' 'contains' 'Bash(dotnet build *)'
   ! json_path '.denyRules' 'contains' 'Bash(make *)'

   -- answers.yaml --
   ecosystems: [go, javascript-typescript, python, rust, dotnet]
   profile: consulting-default
   quick_mode: true
   ```
4. Write `conflict-matrix-all-ecosystems.txtar` — test all 48+ deny rules against build/test/lint commands from all 27 ecosystems:
   ```
   # Full conflict matrix: all deny rules against all ecosystem commands
   exec gdev init --non-interactive --answers-file answers.yaml

   # Run the comprehensive conflict matrix test
   exec gdev check --deny-rules --verbose --format json

   # Verify matrix dimensions
   json_path '.matrixSize.denyRules' '>=48'
   json_path '.matrixSize.operations' '>=27'

   # Verify zero unexpected conflicts
   json_path '.unexpectedConflicts' '0'

   # Verify expected conflicts are documented
   json_path '.expectedConflicts' 'contains' 'upgrade-dep:Bash(npm install *)'

   -- answers.yaml --
   ecosystems: [go, javascript-typescript, python, rust, java, dotnet, docker, terraform]
   profile: consulting-default
   quick_mode: true
   ```
5. Write `overly-broad-deny-detection.txtar` — verify that overly broad deny rules are caught:
   ```
   # Overly broad deny rule detection: Bash(npm *) would block npm test
   # Inject a bad deny rule and verify it's caught
   exec gdev init --non-interactive --answers-file answers.yaml

   # Manually add an overly broad rule to settings.json
   exec jq '.permissions.deny += ["Bash(npm *)"]' .claude/settings.json > /tmp/bad-settings.json
   cp /tmp/bad-settings.json .claude/settings.json

   # Verify the conflict is detected
   ! exec gdev check --deny-rules
   stderr 'conflict'
   stderr 'Bash(npm *)'
   stderr 'Bash(npm test'

   -- answers.yaml --
   ecosystems: [javascript-typescript]
   profile: consulting-default
   quick_mode: true
   ```
6. Write `expected-conflicts-documented.txtar` — verify expected conflicts (e.g., `/upgrade-dep`) are documented and filtered:
   ```
   # Expected conflict: /upgrade-dep needs install commands through guardrail hook
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev check --deny-rules --format json

   # /upgrade-dep conflict with npm install is expected (works through PreToolUse hook)
   json_path '.expectedConflicts' 'contains' 'upgrade-dep'
   json_path '.expectedConflicts[0].explanation' 'contains' 'PreToolUse'

   -- answers.yaml --
   ecosystems: [javascript-typescript]
   profile: consulting-default
   quick_mode: true
   ```
7. Write `deny-rule-regression.txtar` — the master regression test that runs on every skill or deny rule change:
   ```
   # Regression suite: full deny rule × skill operation validation
   exec gdev init --non-interactive --answers-file answers.yaml

   # Validate all deny rules
   exec gdev check --deny-rules --format json --output deny-report.json

   # Verify report is complete
   json_path deny-report.json '.totalRulesTested' '>=48'
   json_path deny-report.json '.totalOperationsTested' '>=20'
   json_path deny-report.json '.unexpectedConflicts' '0'

   -- answers.yaml --
   ecosystems: [go, javascript-typescript, python, rust]
   profile: consulting-default
   quick_mode: true
   ```

**Acceptance Criteria:**
- [ ] Every generated deny rule verified present in settings.json (positive controls)
- [ ] Every build/test/lint command from all ecosystems verified NOT blocked (negative controls)
- [ ] Full 48+ deny rules tested against build/test/lint commands from all 27 ecosystems
- [ ] Overly broad deny rules (`Bash(npm *)`) detected when injected into settings.json
- [ ] Expected conflicts (e.g., `/upgrade-dep` with `npm install`) documented and filtered from failures
- [ ] Expected conflict explanations reference the PreToolUse guardrail hook mechanism
- [ ] `gdev check --deny-rules` reports zero unexpected conflicts with default configuration
- [ ] Adding a new deny rule that blocks a skill operation fails the regression test
- [ ] Adding a new skill with operations that conflict with deny rules fails the regression test
- [ ] Deny-wins-over-allow rule verified (allow rules do not override deny rules)
- [ ] Regression suite runs as part of CI on every PR that modifies skill or deny rule files

**Research Citations:**
- `phases/14-claude-code-integration-agentic-skills.md § Unit 14.5` — `ValidateDenyRuleConflicts()`, precision-scoping principle, expected conflicts
- `research-spikes/gdev-agentic-workflows/guardrail-integration-research.md § 1-4` — permission interaction model, conflict points, precision-scoping, test implementation
- `research-spikes/claude-code-agent-package-guardrails/reference-deny-rules.md` — 48 deny rules covering 15+ package managers

**Status:** Not Started

---

### Unit 22.4: Health & Compliance Reporting Validation

**Description:** Validate `gdev status` output correctness across scenarios, scoring accuracy for known-state projects, conformance label accuracy, drift detection sensitivity and specificity, `gdev evidence` output compliance, and badge generation format correctness.

**Context:** Phase 15 implements the most user-visible reporting system: `gdev status` with progressive disclosure, a three-layer scoring engine, six-category drift detection, SARIF 2.1.0 output for GitHub Code Scanning, and shields.io badge generation. Scoring errors directly mislead developers about their security posture — a project incorrectly scored 90/100 when it should be 60/100 gives false confidence. Drift detection must be sensitive enough to catch real issues (section markers removed, deny rules deleted) without false-positiving on legitimate changes (user-edited devenv.nix). SARIF output must pass schema validation or GitHub Code Scanning will silently reject it. Badge JSON must conform to shields.io's endpoint protocol or badges will display "invalid."

**Desired Outcome:** A test suite that proves scoring determinism, conformance accuracy at boundary conditions, drift detection precision (true positive rate) and recall (true negative rate), SARIF schema compliance, and badge format correctness.

**Steps:**
1. Create `e2e/testdata/script/health/` directory for health reporting tests.
2. Write `status-all-enabled.txtar` — verify output when all tools are enabled:
   ```
   # gdev status: all tools enabled produces correct output
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev status --json > status.json

   # Score should be high when all defenses enabled
   json_path status.json '.score.total' '>=80'
   json_path status.json '.score.grade' 'matches' '^[AB]'
   json_path status.json '.conformance.baseline.pass' 'true'
   json_path status.json '.tools' 'length' '>=8'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   tools_all: true
   ```
3. Write `status-none-enabled.txtar` — verify output when no optional tools are enabled:
   ```
   # gdev status: minimal tools produces low score
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev status --json > status.json

   # Score should be lower with minimal defenses
   json_path status.json '.score.defense' '<80'

   -- answers.yaml --
   ecosystems: [go]
   profile: startup-fast
   quick_mode: true
   ```
4. Write `status-mixed.txtar` — verify mixed tool state produces proportional scoring:
   ```
   # gdev status: partial tools produce proportional score
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev enable semgrep
   exec gdev enable gitleaks
   exec gdev status --json > status.json

   # Defense score should reflect enabled/disabled ratio
   json_path status.json '.defense.layers' 'some' '.status=="enabled"'
   json_path status.json '.defense.layers' 'some' '.status=="disabled"'

   -- answers.yaml --
   ecosystems: [go]
   profile: startup-fast
   quick_mode: true
   ```
5. Write `scoring-determinism.txtar` — verify identical inputs produce identical scores:
   ```
   # Scoring determinism: same state produces same score
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev status --json > status1.json
   exec gdev status --json > status2.json

   # Scores must be identical (ignore timestamps)
   json_path status1.json '.score.total' 'equals' json_path:status2.json:'.score.total'
   json_path status1.json '.score.grade' 'equals' json_path:status2.json:'.score.grade'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
6. Write `conformance-baseline.txtar` — verify baseline PASS requires specific checks:
   ```
   # Conformance: baseline PASS requires lock files, pre-commit hooks, no critical vulns
   exec gdev init --non-interactive --answers-file answers.yaml
   exec gdev status --json > status.json

   json_path status.json '.conformance.baseline.pass' 'true'
   json_path status.json '.conformance.baseline.checks' 'all' '.pass==true'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true

   -- go.mod --
   module example.com/test
   go 1.22

   -- go.sum --
   ```
7. Write `drift-detection-sensitivity.txtar` — verify drift detection detects modified files:
   ```
   # Drift detection: detects modified machine-owned files
   exec gdev init --non-interactive --answers-file answers.yaml

   # Modify a machine-owned file
   exec sh -c 'echo "extra content" >> .envrc'

   exec gdev status --json > status.json
   json_path status.json '.drift.totalFindings' '>=1'
   json_path status.json '.drift.categories' 'some' '.name=="file-modification"'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
8. Write `drift-detection-specificity.txtar` — verify drift detection ignores unrelated changes:
   ```
   # Drift detection: does not flag user-edited files as drift
   exec gdev init --non-interactive --answers-file answers.yaml

   # Modify a user-owned file (should not trigger drift warning)
   exec sh -c 'echo "// my custom code" >> main.go'

   exec gdev status --json > status.json
   # User-owned files should not appear in drift findings
   ! json_path status.json '.drift.categories' 'some' '.findings' 'some' '.subject=="main.go"'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true

   -- main.go --
   package main
   func main() {}
   ```
9. Write `drift-six-categories.txtar` — verify all 6 drift categories with positive and negative controls:
   ```
   # Drift detection: all 6 categories tested
   exec gdev init --non-interactive --answers-file answers.yaml

   # Category 1: File modification — modify .envrc
   exec sh -c 'echo "modified" >> .envrc'

   # Category 4: Section marker integrity — remove closing marker from CLAUDE.md
   exec sed -i '/<!-- \/gdev:commands -->/d' CLAUDE.md

   # Category 5: Lock file drift — touch go.mod to make it newer than go.sum
   exec touch go.mod

   exec gdev status --json > status.json

   # Verify findings from modified categories
   json_path status.json '.drift.categories' 'some' '.name=="file-modification"'
   json_path status.json '.drift.categories' 'some' '.name=="section-marker-integrity"'
   json_path status.json '.drift.categories' 'some' '.name=="lock-file-drift"'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true

   -- go.mod --
   module example.com/test
   go 1.22

   -- go.sum --
   ```
10. Write `evidence-json-schema.txtar` — verify `gdev evidence` JSON schema compliance:
    ```
    # gdev evidence: JSON output matches EvidenceReport schema
    exec gdev init --non-interactive --answers-file answers.yaml
    exec gdev evidence --framework soc2 --format json > evidence.json

    # Verify schema fields
    json_path evidence.json '.schemaVersion' '1.0.0'
    json_path evidence.json '.framework' 'soc2'
    json_path evidence.json '.controls' 'length' '>=6'
    json_path evidence.json '.summary.totalControls' '>=6'
    json_path evidence.json '.summary.coveragePercent' '>=0'

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
11. Write `sarif-compliance.txtar` — verify SARIF 2.1.0 output is valid:
    ```
    # gdev status --sarif: produces valid SARIF 2.1.0
    exec gdev init --non-interactive --answers-file answers.yaml
    exec gdev status --sarif > posture.sarif

    # Verify SARIF structure
    json_path posture.sarif '.version' '2.1.0'
    json_path posture.sarif '.$schema' 'contains' 'sarif-schema-2.1.0'
    json_path posture.sarif '.runs' 'length' '1'
    json_path posture.sarif '.runs[0].tool.driver.name' 'gdev'
    json_path posture.sarif '.runs[0].tool.driver.rules' 'length' '>=1'

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
12. Write `badge-generation.txtar` — verify shields.io badge JSON format:
    ```
    # Badge generation: shields.io JSON format compliance
    exec gdev init --non-interactive --answers-file answers.yaml

    # Score badge
    exec gdev status --format badge > badge-score.json
    json_path badge-score.json '.schemaVersion' '1'
    json_path badge-score.json '.label' 'contains' 'gdev'
    json_path badge-score.json '.message' 'matches' '[0-9]+/100'
    json_path badge-score.json '.color' 'matches' '^(brightgreen|green|yellow|orange|red)$'

    # Conformance badge
    exec gdev status --format badge --badge-type conformance > badge-conf.json
    json_path badge-conf.json '.message' 'matches' '^(PASS|FAIL)$'

    # Defense badge
    exec gdev status --format badge --badge-type defense > badge-def.json
    json_path badge-def.json '.message' 'matches' '[0-9]+/[0-9]+ enabled'

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```

**Acceptance Criteria:**
- [ ] `gdev status --json` output correctness verified across 3 scenarios: all tools, no tools, mixed
- [ ] Scoring determinism verified: identical inputs produce identical scores
- [ ] Baseline conformance PASS requires: lock files present, pre-commit hooks installed, no critical vulns, deny rules present, high-weight defenses enabled
- [ ] Enhanced conformance requires: baseline + SAST + secrets scanning + license compliance
- [ ] Drift detection sensitivity verified: modified machine-owned files detected as `warning`
- [ ] Drift detection specificity verified: user-owned file changes NOT flagged as drift
- [ ] All 6 drift categories tested with positive controls (triggering detection) and negative controls (no false positives)
- [ ] Drift detection completes in <100ms (performance assertion)
- [ ] `gdev evidence --framework soc2` output matches `EvidenceReport` JSON schema
- [ ] `gdev status --sarif` produces valid SARIF 2.1.0 (version field, `$schema`, `runs`, `tool.driver`)
- [ ] `gdev status --format badge` produces shields.io-compatible JSON (score, conformance, defense variants)
- [ ] Badge color mapping verified: score 90+ = brightgreen, 80-89 = green, 70-79 = yellow, 60-69 = orange, <60 = red

**Research Citations:**
- `phases/15-health-status-compliance-reporting.md § Unit 15.1` — `gdev status` command, progressive disclosure, exit codes
- `phases/15-health-status-compliance-reporting.md § Unit 15.2` — scoring engine, conformance tracks, grade boundaries
- `phases/15-health-status-compliance-reporting.md § Unit 15.3` — six-category drift detection, <100ms performance target
- `phases/15-health-status-compliance-reporting.md § Unit 15.4` — `gdev evidence` command, compliance framework mapping
- `phases/15-health-status-compliance-reporting.md § Unit 15.5` — SARIF 2.1.0, badges, JSON schema versioning

**Status:** Not Started

---

### Unit 22.5: Developer Experience Command Validation

**Description:** Validate all Phase 16 developer experience commands: `gdev repair` (recovery from 4 failure categories, `--dry-run`, backup creation, devenv.nix invariant), `gdev info` (output accuracy, `--oneline`, `--json`, <100ms performance), `gdev outdated` (correct ecosystem commands invoked, exit code semantics, `--ecosystem` filter), `gdev update` (three-stage coordination, `--dry-run`, rollback, partial update flags), and `gdev teardown` (3 profiles, user-modified file preservation, archive creation).

**Context:** Phase 16 commands are the "last mile" that makes gdev feel polished. Each command has specific behavioral contracts: `gdev repair` must never modify devenv.nix, `gdev info` must respond in under 100ms, `gdev outdated` must handle per-ecosystem exit code semantics (npm exit 1 means "outdated found" not "error"), `gdev update` must roll back on failure, and `gdev teardown` must never silently delete user-modified files. These contracts are easy to break during refactoring — E2E tests enforce them.

**Desired Outcome:** Every developer experience command tested for its core behavioral contracts, edge cases, and failure modes. The test suite catches regressions in repair safety, info performance, outdated semantics, update rollback, and teardown preservation.

**Steps:**
1. Create `e2e/testdata/script/dx/` directory for developer experience tests.
2. Write `repair-corrupted-file.txtar` — verify repair recovers corrupted machine-owned files:
   ```
   # gdev repair: recovers corrupted machine-owned files
   exec gdev init --non-interactive --answers-file answers.yaml

   # Corrupt a machine-owned file
   exec sh -c 'echo "corrupted" > .envrc'

   # Repair should fix it
   exec gdev repair
   stdout 'fix'
   stdout '.envrc'

   # Verify backup was created
   exec find .gdev/backups -name '.envrc.*'
   stdout '.envrc.'

   # Verify doctor now reports clean
   exec gdev doctor --json
   json_path '.overall' 'pass'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
3. Write `repair-dry-run.txtar` — verify `--dry-run` does not modify files:
   ```
   # gdev repair --dry-run: shows plan without writing files
   exec gdev init --non-interactive --answers-file answers.yaml

   # Corrupt a file
   exec sh -c 'echo "corrupted" > .envrc'
   exec cp .envrc .envrc.before

   # Dry run should not modify anything
   exec gdev repair --dry-run
   stdout 'Would fix'
   stdout '.envrc'

   # File should be unchanged
   cmp .envrc .envrc.before

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
4. Write `repair-devenv-nix-invariant.txtar` — verify devenv.nix is NEVER auto-modified:
   ```
   # gdev repair: devenv.nix is NEVER auto-modified
   exec gdev init --non-interactive --answers-file answers.yaml

   # Note devenv.nix content
   exec cp devenv.nix devenv.nix.before

   # Even with --force, devenv.nix should not be modified
   exec gdev repair --force
   cmp devenv.nix devenv.nix.before

   # Should generate .devenv.nix.new instead
   stdout 'devenv.nix'
   stdout 'review manually'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
5. Write `repair-backup-creation.txtar` — verify mandatory backup before any mutation:
   ```
   # gdev repair: backup created before any file modification
   exec gdev init --non-interactive --answers-file answers.yaml

   # Corrupt multiple files
   exec sh -c 'echo "bad" > .envrc'
   exec sh -c 'echo "bad" > devenv.yaml'

   exec gdev repair
   # Verify backups exist for each repaired file
   exec find .gdev/backups -type f
   stdout '.envrc'
   stdout 'devenv.yaml'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
6. Write `info-output-accuracy.txtar` — verify `gdev info` output correctness:
   ```
   # gdev info: output matches project state
   exec gdev init --non-interactive --answers-file answers.yaml

   exec gdev info
   stdout 'Ecosystems:.*Go'
   stdout 'Security:.*enhanced'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
7. Write `info-oneline.txtar` — verify `--oneline` format:
   ```
   # gdev info --oneline: single-line output
   exec gdev init --non-interactive --answers-file answers.yaml

   exec gdev info --oneline
   # Output should be exactly one line
   exec gdev info --oneline | wc -l
   stdout '^1$'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
8. Write `info-json.txtar` — verify `--json` format:
   ```
   # gdev info --json: valid JSON output
   exec gdev init --non-interactive --answers-file answers.yaml

   exec gdev info --json > info.json
   json_path info.json '.project_name' 'exists'
   json_path info.json '.ecosystems' 'length' '>=1'
   json_path info.json '.security_profile' 'exists'
   json_path info.json '.gdev_version' 'exists'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
9. Write `info-performance.txtar` — verify <100ms response:
   ```
   # gdev info: responds in under 100ms
   exec gdev init --non-interactive --answers-file answers.yaml

   # Time the command (reads only cached state, no evaluation)
   exec sh -c 'start=$(date +%s%N); gdev info > /dev/null; end=$(date +%s%N); echo $(( (end - start) / 1000000 ))'
   stdout '^[0-9][0-9]?$'  # 1-99ms (under 100ms)

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
10. Write `outdated-ecosystem-commands.txtar` — verify correct native commands invoked:
    ```
    [go-available]
    # gdev outdated: invokes correct per-ecosystem command
    exec gdev init --non-interactive --answers-file answers.yaml

    # Should run go list -m -u all for Go projects
    exec gdev outdated 2>&1
    stdout '=== go ==='

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true

    -- go.mod --
    module example.com/test
    go 1.22

    -- go.sum --
    ```
11. Write `outdated-ecosystem-filter.txtar` — verify `--ecosystem` flag:
    ```
    [go-available]
    # gdev outdated --ecosystem: runs only specified ecosystem
    exec gdev init --non-interactive --answers-file answers.yaml

    exec gdev outdated --ecosystem go 2>&1
    stdout '=== go ==='
    ! stdout '=== npm ==='

    -- answers.yaml --
    ecosystems: [go, javascript-typescript]
    profile: consulting-default
    quick_mode: true

    -- go.mod --
    module example.com/test
    go 1.22

    -- go.sum --

    -- package.json --
    {"name": "test", "version": "1.0.0"}
    ```
12. Write `update-dry-run.txtar` — verify `--dry-run` preview:
    ```
    # gdev update --dry-run: preview without applying changes
    exec gdev init --non-interactive --answers-file answers.yaml

    exec gdev update --dry-run
    stdout '[1/3]'
    stdout '[2/3]'
    stdout '[3/3]'

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
13. Write `update-partial-flags.txtar` — verify `--self-only`, `--configs-only`, `--deps-only`:
    ```
    # gdev update: partial update flags
    exec gdev init --non-interactive --answers-file answers.yaml

    # --configs-only should only run stage 2
    exec gdev update --configs-only --dry-run
    ! stdout '[1/3]'
    stdout '[2/3]'
    ! stdout '[3/3]'

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
14. Write `update-rollback-on-failure.txtar` — verify failed config regeneration rolls back:
    ```
    # gdev update: rolls back on config regeneration failure
    exec gdev init --non-interactive --answers-file answers.yaml

    # Save current state
    exec cp .envrc .envrc.before

    # Corrupt the saved answers to force a generation failure
    exec sh -c 'echo "invalid: {{{{" > .gdev.yaml'
    ! exec gdev update --configs-only

    # Files should be rolled back to pre-update state
    cmp .envrc .envrc.before

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
15. Write `teardown-quick.txtar` — verify quick profile removes only state:
    ```
    # gdev teardown --quick: removes only .gdev/ state directory
    exec gdev init --non-interactive --answers-file answers.yaml
    exists .gdev
    exists .envrc
    exists devenv.yaml

    exec gdev teardown --quick --force
    ! exists .gdev
    # Generated configs should still exist
    exists .envrc
    exists devenv.yaml

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
16. Write `teardown-default.txtar` — verify default profile preserves user-modified files:
    ```
    # gdev teardown: preserves user-modified files
    exec gdev init --non-interactive --answers-file answers.yaml

    # Modify a generated file (simulating user customization)
    exec sh -c 'echo "# my custom rule" >> .semgrep.yml'

    exec gdev teardown --force
    ! exists .gdev
    ! exists .envrc          # unmodified, should be removed

    # User-modified files should be preserved
    exists .semgrep.yml

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
17. Write `teardown-compliance.txtar` — verify compliance profile generates evidence and archive:
    ```
    # gdev teardown --compliance: evidence report + archive
    exec gdev init --non-interactive --answers-file answers.yaml

    exec gdev teardown --compliance --force
    # Evidence report should be generated before teardown
    exec find . -name 'teardown-report-*.json'
    stdout 'teardown-report'

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
18. Write `teardown-user-modified-preserved.txtar` — verify user-modified files are never silently deleted:
    ```
    # gdev teardown: user-modified files NEVER silently deleted
    exec gdev init --non-interactive --answers-file answers.yaml

    # Modify devenv.nix (user-edited file)
    exec sh -c 'echo "# my custom configuration" >> devenv.nix'

    exec gdev teardown --force
    # devenv.nix must survive teardown since it was modified
    exists devenv.nix
    exec grep 'my custom configuration' devenv.nix

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```

**Acceptance Criteria:**
- [ ] `gdev repair` recovers corrupted machine-owned files and creates backups in `.gdev/backups/`
- [ ] `gdev repair --dry-run` shows planned actions without writing any files
- [ ] `gdev repair` NEVER auto-modifies devenv.nix (generates `.devenv.nix.new` instead)
- [ ] Mandatory backup created before any file modification by repair
- [ ] After repair, `gdev doctor` reports clean health
- [ ] `gdev info` displays correct project name, ecosystems, security profile, version
- [ ] `gdev info --oneline` produces exactly one line of output
- [ ] `gdev info --json` produces valid JSON with all required fields
- [ ] `gdev info` responds in under 100ms (reads only cached YAML files)
- [ ] `gdev outdated` invokes correct native command per detected ecosystem
- [ ] `gdev outdated --ecosystem` runs only the specified ecosystem's command
- [ ] `gdev outdated` exit code semantics: 0 when up-to-date, 1 when outdated found
- [ ] `gdev update --dry-run` previews all three stages without applying changes
- [ ] `gdev update --configs-only` runs only config regeneration stage
- [ ] `gdev update` rolls back config files on regeneration failure
- [ ] `gdev teardown --quick` removes only `.gdev/` state directory, preserves all generated configs
- [ ] `gdev teardown` (default) removes unmodified files, preserves user-modified files
- [ ] `gdev teardown --compliance` generates evidence report before removal
- [ ] User-modified files are NEVER silently deleted by teardown (hash comparison enforced)

**Research Citations:**
- `phases/16-developer-experience-polish.md § Unit 16.1` — `gdev repair`, 4 failure categories, devenv.nix invariant, backup strategy
- `phases/16-developer-experience-polish.md § Unit 16.2` — `gdev info`, <100ms target, `--oneline`, `--json`
- `phases/16-developer-experience-polish.md § Unit 16.3` — `gdev outdated`, thin wrapper, per-ecosystem exit code semantics
- `phases/16-developer-experience-polish.md § Unit 16.4` — `gdev update`, three-stage coordination, rollback, partial flags
- `phases/16-developer-experience-polish.md § Unit 16.5` — `gdev teardown`, 3 profiles, user-modification preservation

**Status:** Not Started

---

### Unit 22.6: Team Workflow & Integration Validation

**Description:** Validate team-level features: CI aggregation pipeline for multi-project posture reporting, git workflow file generation (PR templates, branch naming hooks), profile inheritance chains (org -> client -> project), client profile switching, multi-project `gdev check` in CI, Starship config generation, and `enterShell` notification content.

**Context:** Phases 13-16 introduce features that only manifest at team or organizational scale: the CI aggregation pipeline collects posture reports from multiple repos, profile inheritance lets org-level defaults flow through client overlays to project config, and git workflow tools generate ecosystem-aware PR templates and branch naming hooks. These features are harder to test than single-project commands because they involve multi-project coordination, profile layering, and CI pipeline simulation. The `gdev team-report` command aggregates JSON posture reports — testing it requires generating multiple reports and feeding them through the aggregation engine.

**Desired Outcome:** A test suite that validates multi-project aggregation, profile inheritance correctness, git workflow generation, and shell integration — the features that make gdev work for a consulting firm managing multiple client projects.

**Steps:**
1. Create `e2e/testdata/script/team/` directory for team workflow tests.
2. Write `ci-aggregation.txtar` — verify multi-project JSON collection and dashboard generation:
   ```
   # Team report: aggregates multiple posture JSONs into dashboard
   mkdir reports/project-a
   mkdir reports/project-b
   mkdir reports/project-c

   # Create synthetic posture reports
   cp project-a-posture.json reports/project-a/posture.json
   cp project-b-posture.json reports/project-b/posture.json
   cp project-c-posture.json reports/project-c/posture.json

   exec gdev team-report --input-dir reports/ --format md > dashboard.md

   # Verify dashboard content
   exec grep 'project-a' dashboard.md
   exec grep 'project-b' dashboard.md
   exec grep 'project-c' dashboard.md
   exec grep 'Average score' dashboard.md
   exec grep 'Baseline pass rate' dashboard.md

   -- project-a-posture.json --
   {
     "schemaVersion": "1.0.0",
     "projectName": "project-a",
     "score": {"total": 92, "grade": "A", "defense": 95, "config": 90, "depHealth": 88},
     "conformance": {"baseline": {"pass": true}, "enhanced": {"pass": true}},
     "dependencies": {"totals": {"critical": 0, "high": 0, "moderate": 1, "low": 2}},
     "gdevVersion": "0.16.2"
   }
   -- project-b-posture.json --
   {
     "schemaVersion": "1.0.0",
     "projectName": "project-b",
     "score": {"total": 65, "grade": "C", "defense": 60, "config": 70, "depHealth": 65},
     "conformance": {"baseline": {"pass": false}, "enhanced": {"pass": false}},
     "dependencies": {"totals": {"critical": 0, "high": 3, "moderate": 5, "low": 8}},
     "gdevVersion": "0.15.0"
   }
   -- project-c-posture.json --
   {
     "schemaVersion": "1.0.0",
     "projectName": "project-c",
     "score": {"total": 78, "grade": "B", "defense": 80, "config": 75, "depHealth": 78},
     "conformance": {"baseline": {"pass": true}, "enhanced": {"pass": false}},
     "dependencies": {"totals": {"critical": 0, "high": 1, "moderate": 3, "low": 4}},
     "gdevVersion": "0.16.2"
   }
   ```
3. Write `ci-aggregation-json.txtar` — verify JSON aggregation output:
   ```
   # Team report: JSON output with summary statistics
   mkdir reports/alpha
   cp posture.json reports/alpha/posture.json

   exec gdev team-report --input-dir reports/ --format json > team.json

   json_path team.json '.schemaVersion' '1.0.0'
   json_path team.json '.summary.projectCount' '1'
   json_path team.json '.summary.averageScore' '>=0'
   json_path team.json '.projects' 'length' '1'

   -- posture.json --
   {
     "schemaVersion": "1.0.0",
     "projectName": "alpha",
     "score": {"total": 85, "grade": "B+", "defense": 88, "config": 82, "depHealth": 84},
     "conformance": {"baseline": {"pass": true}, "enhanced": {"pass": true}},
     "dependencies": {"totals": {"critical": 0, "high": 0, "moderate": 2, "low": 3}},
     "gdevVersion": "0.16.2"
   }
   ```
4. Write `git-pr-template.txtar` — verify PR template content varies by ecosystem:
   ```
   # PR template: ecosystem-aware content
   exec gdev init --non-interactive --answers-file answers-go.yaml
   exists .github/pull_request_template.md
   exec grep 'linter passes' .github/pull_request_template.md
   exec grep 'Security Checklist' .github/pull_request_template.md

   -- answers-go.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
5. Write `git-pr-template-multi-ecosystem.txtar` — verify multi-ecosystem PR template:
   ```
   # PR template: multi-ecosystem gets combined checklists
   exec gdev init --non-interactive --answers-file answers.yaml
   exists .github/pull_request_template.md

   # Go-specific checks
   exec grep -i 'linter' .github/pull_request_template.md
   # TypeScript-specific checks
   exec grep -i 'type' .github/pull_request_template.md

   -- answers.yaml --
   ecosystems: [go, javascript-typescript]
   profile: consulting-default
   quick_mode: true
   ```
6. Write `git-branch-naming.txtar` — verify branch naming hook validation:
   ```
   # Branch naming hook: rejects non-conforming names, allows standard branches
   exec gdev init --non-interactive --answers-file answers.yaml
   exec git init .
   exec git add .
   exec git commit -m "initial"

   # Standard branches should be allowed
   exec git checkout -b main
   exec git checkout -b develop
   exec git checkout -b feat/add-login

   # Non-conforming branch should be rejected on push (hook is pre-push)
   # Test the hook script directly
   exec sh -c 'branch="garbage-name"; pattern="^(feat|fix|chore|docs|refactor|test|ci)/[a-z0-9._-]+$"; if [[ "$branch" =~ $pattern ]]; then exit 0; else exit 1; fi'
   # The above should exit 1 for non-conforming name

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true
   ```
7. Write `profile-inheritance-chain.txtar` — verify org -> client -> project chain:
   ```
   # Profile inheritance: org -> client -> project overrides layer correctly
   exec gdev init --non-interactive --verbose
   stdout 'profile:consulting-default'

   exec gdev status --json > status.json
   # Client security level should override profile default
   json_path status.json '.score.defense' '>=0'

   -- .gdev.yaml --
   version: 1
   profile: consulting-default
   client:
     name: acme-corp
     security_level: strict
     compliance: [soc2]
   ```
8. Write `client-profile-switching.txtar` — verify switching client profiles within same repo:
   ```
   # Client profile switching: different clients produce different configs
   exec gdev init --non-interactive
   exec gdev status --json > status-strict.json

   # Switch to a less restrictive client
   cp .gdev-baseline.yaml .gdev.yaml
   exec gdev init --non-interactive
   exec gdev status --json > status-baseline.json

   # Strict client should have higher defense score
   # (can't directly compare in testscript, but verify both produce valid output)
   json_path status-strict.json '.score.total' '>=0'
   json_path status-baseline.json '.score.total' '>=0'

   -- .gdev.yaml --
   version: 1
   profile: enterprise
   client:
     name: secure-client
     security_level: strict

   -- .gdev-baseline.yaml --
   version: 1
   profile: startup-fast
   client:
     name: poc-client
     security_level: baseline
   ```
9. Write `multi-project-ci-check.txtar` — verify `gdev check` works in CI matrix context:
   ```
   # Multi-project gdev check: CI matrix simulation
   exec gdev init --non-interactive --answers-file answers.yaml

   # Run gdev check with CI-appropriate flags
   env CI=true
   exec gdev check --format json --audit-level medium > check-result.json

   json_path check-result.json '.summary.total' '>=1'
   json_path check-result.json '.version' 'exists'

   -- answers.yaml --
   ecosystems: [go]
   profile: consulting-default
   quick_mode: true

   -- go.mod --
   module example.com/test
   go 1.22

   -- go.sum --
   ```
10. Write `starship-config.txtar` — verify Starship config generation when opt-in enabled:
    ```
    # Starship integration: config generated when enabled
    exec gdev init --non-interactive --answers-file answers.yaml
    exec gdev enable starship-integration

    exists .starship.toml
    exec grep 'custom.gdev' .starship.toml
    exec grep 'GDEV_PROJECT_NAME' .starship.toml
    exec grep 'GDEV_SECURITY_PROFILE' .starship.toml

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
11. Write `entershell-notification.txtar` — verify enterShell notification content in devenv.nix:
    ```
    # enterShell: notification content generated in devenv.nix
    exec gdev init --non-interactive --answers-file answers.yaml

    # Verify enterShell contains gdev notification
    exec grep 'enterShell' devenv.nix
    exec grep 'GDEV_ECOSYSTEMS' devenv.nix
    exec grep 'GDEV_SECURITY_PROFILE' devenv.nix

    # Verify gdev env vars are set
    exec grep 'GDEV_PROJECT_NAME' devenv.nix
    exec grep 'GDEV_VERSION' devenv.nix

    -- answers.yaml --
    ecosystems: [go]
    profile: consulting-default
    quick_mode: true
    ```
12. Write `dashboard-generation.txtar` — verify team dashboard includes all expected sections:
    ```
    # Team dashboard: all sections present
    mkdir reports/p1
    mkdir reports/p2
    cp p1.json reports/p1/posture.json
    cp p2.json reports/p2/posture.json

    exec gdev team-report --input-dir reports/ --format md > dashboard.md

    # Verify all dashboard sections
    exec grep '## Overview' dashboard.md
    exec grep '## Project Scores' dashboard.md
    exec grep '## Attention Required' dashboard.md
    exec grep 'Projects tracked' dashboard.md

    -- p1.json --
    {
      "schemaVersion": "1.0.0",
      "projectName": "project-1",
      "score": {"total": 90, "grade": "A-"},
      "conformance": {"baseline": {"pass": true}},
      "dependencies": {"totals": {"critical": 0, "high": 0}},
      "gdevVersion": "0.16.2"
    }
    -- p2.json --
    {
      "schemaVersion": "1.0.0",
      "projectName": "project-2",
      "score": {"total": 55, "grade": "F"},
      "conformance": {"baseline": {"pass": false}},
      "dependencies": {"totals": {"critical": 1, "high": 2}},
      "gdevVersion": "0.15.0"
    }
    ```

**Acceptance Criteria:**
- [ ] CI aggregation pipeline collects multiple posture JSONs and produces markdown dashboard
- [ ] Dashboard includes overview table, project scores, and attention-required section
- [ ] `gdev team-report --format json` produces valid JSON with summary statistics
- [ ] PR template generated with ecosystem-appropriate content (Go checks for Go, TS checks for TS)
- [ ] Multi-ecosystem project gets combined checklist items in PR template
- [ ] Branch naming hook rejects non-conforming names and allows main/master/develop
- [ ] Profile inheritance chain (org -> client -> project) resolves correctly with client overrides
- [ ] Client profile switching produces different security configurations
- [ ] `gdev check` works in CI context (`CI=true` env var) with JSON output
- [ ] Multi-project `gdev check` in CI matrix produces valid per-project reports
- [ ] `gdev enable starship-integration` generates `.starship.toml` with gdev custom modules
- [ ] Starship config references `GDEV_PROJECT_NAME` and `GDEV_SECURITY_PROFILE` env vars
- [ ] `enterShell` notification content generated in devenv.nix with gdev context
- [ ] gdev environment variables (`GDEV_PROJECT_NAME`, `GDEV_VERSION`, etc.) set in devenv.nix

**Research Citations:**
- `phases/15-health-status-compliance-reporting.md § Unit 15.6` — team aggregation pipeline, markdown dashboard, scope file
- `phases/16-developer-experience-polish.md § Unit 16.6` — git workflow automation, PR templates, branch naming hooks
- `phases/16-developer-experience-polish.md § Unit 16.7` — shell integration, Starship, enterShell, gdev env vars
- `phases/13-project-configuration-team-standards.md § Unit 13.7` — client profiles, compliance levels, profile inheritance

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All six units pass acceptance criteria
- [ ] Configuration round-trip verified: write `.gdev.yaml` -> parse -> resolve -> generate -> `gdev check` passes
- [ ] All 4 onboarding modes tested: Create (no config), Join (config exists, no state), Update (version mismatch), Repair (drifted files)
- [ ] All 25+ skill and agent files parse without error and have correct `disable-model-invocation` settings
- [ ] Deny rule conflict matrix tests zero unexpected conflicts with all 48+ rules against all ecosystem commands
- [ ] Health scoring is deterministic: same project state always produces identical scores
- [ ] Conformance labels (baseline/enhanced PASS/FAIL) are accurate at boundary conditions
- [ ] All 6 drift detection categories have positive and negative control tests
- [ ] SARIF 2.1.0 output passes schema validation
- [ ] shields.io badge JSON produces valid endpoint responses for all 3 variants
- [ ] `gdev repair` never modifies devenv.nix and always creates backups
- [ ] `gdev info` responds in under 100ms
- [ ] `gdev teardown` never silently deletes user-modified files
- [ ] `gdev update` rolls back on failure
- [ ] Multi-project team aggregation produces correct markdown dashboard
- [ ] Profile inheritance chain resolves correctly through org -> client -> project -> local layers
- [ ] All tests run successfully in the Phase 17 CI pipeline (quick-validation and nightly matrix)
