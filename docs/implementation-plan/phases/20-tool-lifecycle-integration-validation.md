# Phase 20: Tool Lifecycle & Integration Validation

## Goal

Validate that the tool lifecycle management system (Phase 12) works correctly: every tool can be enabled, disabled, and re-enabled cleanly; shared files maintain integrity; idempotency holds; cross-tool interactions are correct; and the wizard drives generation accurately. At the end of this phase, 80+ test scenarios confirm that tool adoption is a truly reversible, low-risk decision.

## Dependencies

Phase 17 complete (test infrastructure). Phase 12 complete (lifecycle system, all tool registrations). Phase 4 complete (Claude Code addon). Phase 11 desirable (AI agent tools). Phases 13-16 complete (project configuration, agentic skills, health/status reporting, developer experience polish — for testing `gdev check`, `gdev status`, `gdev repair`, `gdev info`, `gdev outdated`, `gdev update`, and `gdev teardown` commands). Phase 14 skills and Phase 15 reporting should be covered by lifecycle testing.

## Phase Outputs

- Individual lifecycle round-trip tests for all 16+ tools
- Shared file integrity tests for all 6 shared file formats
- Idempotency test suite (5 scenarios)
- Cross-tool interaction tests (7 scenarios)
- Wizard integration tests (4 scenarios)
- Migration/upgrade tests (3 scenarios)
- User modification preservation tests (4 scenarios)
- Edge case and error handling tests (15+ scenarios)

---

### Unit 20.1: Individual Tool Lifecycle Round-Trips

**Description:** For each of the 16 lifecycle-managed tools, run the full enable → verify → disable → verify → re-enable → verify cycle, plus a rapid toggle stress test. Implement as a parameterized Go test (`testToolRoundTrip(t, toolName)`) with one testscript per tool for readable end-to-end coverage.

**Context:** The lifecycle system (Unit 12.1) promises that every tool can be cleanly added and removed. This promise must hold for every tool individually before cross-tool interactions are tested. Each tool has a unique combination of exclusive files, shared file contributions, and prerequisites — bugs tend to hide in the per-tool details (e.g., container-security contributing 3 packages vs. semgrep contributing 1, trail-of-bits-skills owning multiple exclusive files vs. ripsecrets owning none).

**Desired Outcome:** All 16 tools pass the full enable → disable → re-enable cycle. After disabling, no orphaned artifacts remain. After re-enabling, files are identical to the first enable. Rapid toggling (enable → disable → enable → disable) leaves the project in a clean state indistinguishable from never having enabled the tool.

**Steps:**
1. Create `lifecycle_test.go` with the parameterized `testToolRoundTrip(t *testing.T, toolName string)` helper. The helper performs:
   a. Fresh project setup: `gdev init --yes`, then `gdev disable <tool>` if the tool is default-on.
   b. **Enable**: `gdev enable <tool>` → assert exit 0.
   c. **Verify enabled state**: all exclusive files exist at expected paths (per tool's `OwnedFiles` registry), all shared file sections present (grep for `# --- <tool> ---` in devenv.nix, `<!-- gdev:<tool> -->` in CLAUDE.md, server key in .mcp.json, hook entry in pre-commit config), state file records ownership for all files, `gdev status` shows tool as enabled.
   d. **Disable**: `gdev disable <tool>` → assert exit 0.
   e. **Verify disabled state**: all exclusive files deleted, all shared file sections removed, shared files still valid (Nix eval for devenv.nix, JSON parse for settings.json and .mcp.json, YAML parse for pre-commit config), state file no longer references the tool, `gdev status` shows disabled.
   f. **Re-enable**: `gdev enable <tool>` → assert all files recreated identically to step (c).
   g. **Rapid toggle**: enable → disable → enable → disable → verify project state is clean, no orphaned markers, no empty section blocks (e.g., no `# --- semgrep ---\n# --- end semgrep ---` with nothing between).
2. Create one Go test function per tool, each calling the parameterized helper:
   - `TestToolLifecycle_Semgrep(t *testing.T)`
   - `TestToolLifecycle_Gitleaks(t *testing.T)`
   - `TestToolLifecycle_ContainerSecurity(t *testing.T)`
   - `TestToolLifecycle_LicenseCompliance(t *testing.T)`
   - `TestToolLifecycle_AttachGuard(t *testing.T)`
   - `TestToolLifecycle_Ripsecrets(t *testing.T)`
   - `TestToolLifecycle_AgentPostmortem(t *testing.T)`
   - `TestToolLifecycle_VersionSentinel(t *testing.T)`
   - `TestToolLifecycle_Semble(t *testing.T)`
   - `TestToolLifecycle_Context7(t *testing.T)`
   - `TestToolLifecycle_GithubMcp(t *testing.T)`
   - `TestToolLifecycle_SocketDevMcp(t *testing.T)`
   - `TestToolLifecycle_TrailOfBitsSkills(t *testing.T)`
   - `TestToolLifecycle_Secretspec(t *testing.T)`
   - `TestToolLifecycle_Commitlint(t *testing.T)`
   - `TestToolLifecycle_Changelog(t *testing.T)`
3. Create corresponding testscript `.txtar` files for each tool (`testdata/lifecycle-semgrep.txtar`, etc.) covering the same enable → disable → re-enable flow with shell-level assertions for readability and debuggability.
4. Implement helper assertions used across all tests:
   - `assertFileExists(t, path)` / `assertFileNotExists(t, path)`
   - `assertValidNix(t, path)` — runs `nix-instantiate --parse` or equivalent
   - `assertValidJSON(t, path)` — `json.Unmarshal` succeeds
   - `assertValidYAML(t, path)` — `yaml.Unmarshal` succeeds
   - `assertSectionPresent(t, path, toolName)` / `assertSectionAbsent(t, path, toolName)`
   - `assertJSONKeyPresent(t, path, key)` / `assertJSONKeyAbsent(t, path, key)`
   - `assertHookPresent(t, path, hookName)` / `assertHookAbsent(t, path, hookName)`
   - `assertToolStatus(t, toolName, enabled bool)` — parses `gdev status --json`
   - `hashFile(path) string` — SHA256 for identity comparison
   - `setupTestProject(t) string` — creates temp dir with multi-ecosystem fixture (Go + TS + Docker)
5. Tool-specific verification details per tool's file inventory (from research artifact § 1.2):
   - **semgrep**: exclusive `.semgrep.yml`, shared sections in devenv.nix + pre-commit + CLAUDE.md, CI regeneration
   - **gitleaks**: exclusive `.gitleaks.toml`, shared sections in devenv.nix + pre-commit + CLAUDE.md, CI regeneration
   - **container-security**: exclusive `.grype.yaml` + `.cosign/policy.yaml`, shared section in devenv.nix (3 packages: grype, syft, cosign) + CLAUDE.md, CI regeneration. Verify `.cosign/` directory removed only when empty.
   - **license-compliance**: exclusive `.scancode.yml` + `.license-exceptions.yml`, shared CLAUDE.md section, CI regeneration
   - **attach-guard**: exclusive `.claude/hooks/package-guard.py`, shared sections in settings.json + CLAUDE.md
   - **ripsecrets**: shared pre-commit hook only (simplest tool — no exclusive files, no CLAUDE.md section)
   - **agent-postmortem**: exclusive `.claude/skills/agent-postmortem/SKILL.md`, shared CLAUDE.md section. Verify directory cleanup.
   - **version-sentinel**: exclusive `.version-sentinel/ignore`, shared sections in settings.json + CLAUDE.md. Verify directory cleanup.
   - **semble**: shared .mcp.json entry + CLAUDE.md section, exclusive `.claude/agents/semble-search.md`. Verify directory cleanup.
   - **context7**: shared .mcp.json entry + CLAUDE.md section (no exclusive files)
   - **github-mcp**: shared .mcp.json entry + CLAUDE.md section (no exclusive files)
   - **socket-dev-mcp**: shared .mcp.json entry + CLAUDE.md section (no exclusive files)
   - **trail-of-bits-skills**: multiple exclusive files in `.claude/skills/` (supply-chain-risk-auditor.md, differential-review.md, insecure-defaults.md), shared CLAUDE.md section. Verify all 3 skill files created/removed together.
   - **secretspec**: exclusive `secretspec.toml`, shared sections in devenv.nix + CLAUDE.md
   - **commitlint**: exclusive `.commitlintrc.yml`, shared sections in pre-commit (commit-msg hook) + CLAUDE.md
   - **changelog**: exclusive `cliff.toml`, shared sections in devenv.nix + CLAUDE.md, CI regeneration

**Acceptance Criteria:**
- [ ] All 16 tools pass enable → disable → re-enable round-trip with zero orphaned artifacts
- [ ] After disable: all exclusive files deleted, all shared file sections removed, shared files remain valid
- [ ] After re-enable: files identical to first enable (hash comparison)
- [ ] Rapid toggle (4 operations) leaves project in clean never-enabled state
- [ ] Shared files parse successfully after every state transition (Nix eval, JSON parse, YAML parse)
- [ ] `gdev status` accurately reflects tool state after every operation
- [ ] State file consistent with actual file system state after every operation
- [ ] Directory cleanup works correctly for tools with nested exclusive files (.cosign/, .version-sentinel/, .claude/agents/, .claude/skills/agent-postmortem/)

**Research Citations:**
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 3. Category A: Individual Tool Lifecycle`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 1.2 Per-Tool File Inventory`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 5.3 Go Test Organization`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 5.4 Helper Functions Needed`

**Status:** Not Started

---

### Unit 20.2: Shared File Integrity — Empty State Testing

**Description:** Test that removing ALL tool sections from each of the 6 shared file formats leaves the file valid and with core content intact. Then test the inverse — enabling all tools and verifying all shared files are well-formed with maximum sections.

**Context:** Shared files (devenv.nix, settings.json, CLAUDE.md, .mcp.json, pre-commit config, CI workflows) are the most fragile part of the lifecycle system. Each is written to by multiple tools via section markers (Nix/Markdown) or key-based surgery (JSON/YAML). The most dangerous state transitions are going to zero tool sections (empty state) and having maximum tool sections (full state), because these are the boundary conditions where trailing comma bugs, orphaned markers, empty collection issues, and whitespace corruption are most likely to surface.

**Desired Outcome:** Every shared file survives both the empty state (all tools disabled) and the full state (all tools enabled) while remaining structurally valid and preserving non-tool content.

**Steps:**
1. **devenv.nix empty state (E1)**: enable all 5 tools that contribute to devenv.nix (semgrep, gitleaks, container-security, secretspec, changelog), then disable all 5. Assert: file still exists (core framework file), valid Nix (`nix-instantiate --parse`), core packages remain (git, curl, etc.), no empty section markers, no trailing comma issues in the packages list, no double-blank-lines where sections were removed.
2. **settings.json empty hooks (E2)**: enable attach-guard + version-sentinel, then disable both. Assert: file still exists, valid JSON, deny rules still present (Phase 4 core content), hooks map empty or absent, permission presets still present.
3. **CLAUDE.md empty tool sections (E3)**: enable all 15 tools that contribute CLAUDE.md sections, then disable all 15. Assert: file still exists, core generated section remains (between `<!-- BEGIN GENERATED SECTION -->` / `<!-- END GENERATED SECTION -->`) with non-tool content (build commands, security instructions, language conventions), user custom section preserved (content below generated markers), no orphaned `<!-- gdev:* -->` markers, no empty lines accumulating where sections were removed (at most single blank line).
4. **.mcp.json empty servers (E4)**: enable all 4 MCP tools (context7, github-mcp, socket-dev-mcp, semble), then disable all 4. Two sub-scenarios:
   a. No non-gdev entries: assert .mcp.json is deleted (empty .mcp.json is useless).
   b. User-added server entry exists: assert .mcp.json preserved with only user entries, no gdev-managed keys remain.
5. **.pre-commit-config.yaml empty tool hooks (E5)**: enable all 4 hook tools (ripsecrets, gitleaks, semgrep, commitlint), then disable all 4. Assert: core framework hooks remain (check-added-large-files, no-commit-to-branch, check-merge-conflict from Phase 5 baseline), YAML valid, no tool hook entries. Verify git-hooks devenv.yaml input handling (may remain if core hooks present, or be removed if no hooks remain at all).
6. **CI workflow minimal state (E6)**: disable all 5 CI-contributing tools (semgrep, gitleaks, container-security, license-compliance, changelog). Assert: CI workflow file still exists (ci-workflows is always-on virtual tool), Harden-Runner present as first step in every job, ecosystem-level checks from Phase 5 core (OSV Scanner, frozen-install) may remain, no tool-specific steps, valid GitHub Actions YAML.
7. **Full state test (inverse)**: enable all tools, then verify: devenv.nix valid Nix with all 5 tool sections, .mcp.json valid JSON with 4 server entries, settings.json valid JSON with both hook sets, CLAUDE.md has 15 tool sections with no overlapping markers, pre-commit config has 4 hook entries in correct order, CI workflow has all steps from all 5 CI-contributing tools plus Harden-Runner.

**Acceptance Criteria:**
- [ ] devenv.nix survives all-tools-removed: valid Nix, core packages present, no orphaned markers
- [ ] settings.json survives all-hooks-removed: valid JSON, deny rules intact, hooks empty or absent
- [ ] CLAUDE.md survives all-tools-removed: core generated section intact, user section preserved, no orphaned markers
- [ ] .mcp.json deleted when empty (no non-gdev entries) or preserved with only user entries
- [ ] Pre-commit config retains core framework hooks when all tool hooks removed, valid YAML
- [ ] CI workflow has Harden-Runner-only minimal state when all tool steps removed, valid YAML
- [ ] Full state (all tools enabled) produces valid files across all 6 formats with no duplicates or overlaps
- [ ] Nix packages list has no trailing comma or syntax issues at any tool count (0, 1, 3, 5, 7)

**Research Citations:**
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 3. Category E: Shared File Integrity`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 1.3 Shared File Contribution Summary`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 4.4 Nix-Specific Concerns for devenv.nix`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 4.5 CLAUDE.md Marker Interaction with Existing Markers`

**Status:** Not Started

---

### Unit 20.3: Idempotency & State Consistency

**Description:** Verify that repeated operations produce identical results and that the state file remains consistent with the actual file system through all lifecycle transitions.

**Context:** Idempotency is a safety invariant: running the same command twice must not corrupt state or duplicate content. This is especially important because users will re-run commands when unsure if they succeeded, CI pipelines may retry on transient failures, and `gdev init --yes` may be run on an already-initialized project. State consistency ensures the state file (`.devinit/.gdev-init-state.yaml`) accurately reflects what was generated and what's on disk — divergence between state and reality leads to orphaned artifacts or missed cleanup.

**Desired Outcome:** Every idempotency scenario produces identical file state. The state file is always consistent with the file system: every generated file has a state entry, every state entry has a corresponding file, and all hashes match.

**Steps:**
1. **Enable already-enabled tool (F1)**: enable semgrep (already default-on after init), run `gdev enable semgrep` again. Assert: exit 0, informational message "semgrep is already enabled", all file hashes unchanged, state file unchanged.
2. **Disable already-disabled tool (F2)**: ensure semgrep is disabled, run `gdev disable semgrep` again. Assert: exit 0, message "semgrep is already disabled", no file changes.
3. **Double enable in succession (F3)**: `gdev enable semgrep` → record all file hashes → `gdev enable semgrep` → assert all hashes identical, no duplicate sections in any shared file (e.g., no two `# --- semgrep ---` blocks in devenv.nix, no two `<!-- gdev:semgrep -->` blocks in CLAUDE.md).
4. **Init then enable default-on tool (F4)**: `gdev init --yes` (semgrep enabled as AlwaysOn) → record state → `gdev enable semgrep` → assert no-op, state unchanged. Verify init's enabled state is recognized by the enable command.
5. **Init --yes twice (F5)**: `gdev init --yes` → record all file hashes → `gdev init --yes` → assert files identical, no duplicated sections in any shared file, state file consistent.
6. **State consistency validation**: after each scenario above, run a state audit:
   a. Every generated file recorded in state has a corresponding file on disk.
   b. Every gdev-owned file on disk has a corresponding state entry.
   c. All hash values in state match actual file content (SHA256).
   d. File ownership in state matches tool registry expectations.
7. **State round-trip with user modification**: `gdev init --yes` → user modifies a shared file (add content outside markers) → `gdev init --yes` again → verify user-modified content preserved, tool sections regenerated correctly, state hashes updated to reflect new file content.

**Acceptance Criteria:**
- [ ] Enable on already-enabled tool: exit 0, no-op, files unchanged
- [ ] Disable on already-disabled tool: exit 0, no-op
- [ ] Double enable produces no duplicate sections in any shared file
- [ ] `gdev init --yes` then `gdev enable <default-on-tool>`: recognized as no-op
- [ ] `gdev init --yes` run twice: files identical, no duplicated content
- [ ] State file always consistent: every file entry maps to a real file, every real gdev file has a state entry, all hashes match
- [ ] User modifications outside markers survive `gdev init --yes` re-runs

**Research Citations:**
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 3. Category F: Idempotency Testing`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 4.3 State Consistency Edge Cases`

**Status:** Not Started

---

### Unit 20.4: Cross-Tool Interaction Testing

**Description:** Verify that tools interacting through the same shared files coexist correctly: hook ordering is deterministic, composite tools manage all their sub-packages, soft dependencies degrade gracefully, and CI regeneration is consistent.

**Context:** While no tools have explicit mutual exclusion conflicts, several implicit interaction points exist: pre-commit hooks must run in a specific order for user experience (fast checks first), settings.json hooks from attach-guard and version-sentinel use different matchers but could both fire on the same command, MCP servers must all coexist in valid JSON, and CI workflow regeneration must be deterministic regardless of the sequence tools were enabled in. These interactions are where integration bugs live — individual tool tests pass but multi-tool combinations fail.

**Desired Outcome:** All 7 cross-tool interaction scenarios pass, proving that the lifecycle system produces correct results regardless of tool enable/disable ordering, and that tools sharing the same files don't interfere with each other.

**Steps:**
1. **Pre-commit hook ordering (G1)**: enable ripsecrets, then gitleaks, then semgrep (in any order). Assert hook order in pre-commit config is always ripsecrets → gitleaks → semgrep regardless of enable sequence. Then disable gitleaks and re-enable it. Assert order is still ripsecrets → gitleaks → semgrep. Hook ordering is deterministic by tier/priority, not insertion-order-dependent.
2. **Container security multi-package removal (G2)**: enable container-security. Assert devenv.nix `# --- container-security ---` section contains all 3 packages (grype, syft, cosign). Disable container-security. Assert all 3 packages removed (not just first, not just last). Assert devenv.nix valid Nix.
3. **changelog + commitlint soft dependency (G3)**: enable changelog. Assert informational message suggesting commitlint. Assert changelog works without commitlint (cliff.toml created, CI step present, no error). Enable commitlint. Assert both tools' files present, no conflicts. Disable commitlint. Assert commitlint files removed, changelog still functional (cliff.toml present, CI step present), no error or warning about missing commitlint.
4. **All MCP servers coexistence (G4)**: enable all 4 MCP tools (context7, github-mcp, socket-dev-mcp, semble). Assert .mcp.json valid JSON with 4 entries under `mcpServers`, no duplicate keys, each server has correct command/args. `gdev status` shows 4 MCP servers.
5. **attach-guard + version-sentinel hook coexistence (G5)**: enable both. Assert settings.json has hooks from both tools with distinct matchers (attach-guard: Bash command matcher for package installs; version-sentinel: Edit/Write/MultiEdit matcher for manifest edits). No duplicate hook entries. Disable attach-guard. Assert only version-sentinel hooks remain. Assert settings.json still valid JSON.
6. **Full-surface tool enable/disable (G6)**: semgrep touches 4 shared file types (devenv.nix, CLAUDE.md, pre-commit, CI). Enable semgrep. Verify sections present in all 4 files. Disable semgrep. Verify all 4 files cleaned. Verify all 4 files still valid.
7. **CI workflow regeneration consistency (G7)**: enable semgrep + gitleaks → record CI workflow hash (A). Disable gitleaks → re-enable gitleaks → record CI workflow hash (B). Assert A == B. Same set of enabled tools must produce byte-identical workflow output (deterministic regeneration).

**Acceptance Criteria:**
- [ ] Pre-commit hook order is always ripsecrets → gitleaks → semgrep regardless of enable sequence
- [ ] Container security disable removes all 3 packages (grype, syft, cosign), not a subset
- [ ] changelog works without commitlint; commitlint disable doesn't break changelog
- [ ] All 4 MCP servers coexist in valid .mcp.json with no duplicates
- [ ] attach-guard and version-sentinel hooks coexist with distinct matchers; removing one leaves the other intact
- [ ] Full-surface tool (semgrep) cleans all 4 shared file types on disable
- [ ] CI regeneration is deterministic: same enabled tools → same workflow hash

**Research Citations:**
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 3. Category G: Cross-Tool Interaction Testing`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 2.2 Implicit Conflicts — Same Shared File Section`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 2.4 Soft Dependencies`

**Status:** Not Started

---

### Unit 20.5: Wizard Integration & Error Handling

**Description:** Verify that the wizard correctly drives tool selection for both quick-path and customize-path flows, that re-running the wizard preserves manual tool state changes, and that error cases (missing prerequisites, conflicts, unmet detection conditions) produce clean failures with no partial state.

**Context:** The wizard is the primary entry point for most users. Quick-path applies smart defaults (AlwaysOn + detected OnWhenDetected), customize-path allows individual toggles, and `gdev init --yes` is the non-interactive equivalent. Error handling must be atomic — a failed `gdev enable` must leave zero partial files and an unchanged state file — because partial state is worse than a clean failure (the user would need to manually clean up artifacts the system doesn't know about).

**Desired Outcome:** Wizard produces correct tool selections for all default policies. Error cases fail cleanly with actionable messages and no side effects.

**Steps:**
1. **Quick path defaults (H1)**: set up a Go + TypeScript + Docker project with Python 3.11 available. Run `gdev init` (quick path, accept defaults). Assert:
   - AlwaysOn tools enabled: semgrep, gitleaks, attach-guard, agent-postmortem, version-sentinel, context7, github-mcp, trail-of-bits-skills, ripsecrets
   - OnWhenDetected tools enabled based on detection: container-security (Docker detected), semble (Python >=3.10), socket-dev-mcp (Go + TS detected), secretspec (only if services detected)
   - OptIn tools disabled: license-compliance, commitlint, changelog
   - All enabled tools' files generated correctly.
2. **Customize path toggles (H2)**: run `gdev init` in customize mode. Toggle: disable semgrep, enable license-compliance, enable changelog. Assert: semgrep files NOT generated, license-compliance files generated (.scancode.yml, .license-exceptions.yml, CLAUDE.md section), changelog files generated (cliff.toml, devenv.nix section, CLAUDE.md section, CI step), CI workflow has license-compliance + changelog steps but NOT semgrep step.
3. **Re-run wizard preserves manual changes (H3)**: `gdev init --yes` (defaults), then `gdev disable semgrep` (manual change). Run `gdev init --update` (or re-run wizard). Assert: semgrep remains disabled (user's explicit disable preserved in saved answers), all other tools unchanged, no duplicate sections in any shared file.
4. **gdev init --yes flag behavior (H4)**: fresh project with Go + Docker detected, Python 3.11 available. Run `gdev init --yes`. Assert: all AlwaysOn enabled, container-security enabled (Docker), semble enabled (Python >=3.10), version-sentinel enabled (Python >=3.11 + supported ecosystems), socket-dev-mcp enabled (Go), license-compliance DISABLED (OptIn), commitlint DISABLED (OptIn), changelog DISABLED (OptIn).
5. **Missing prerequisite — version-sentinel (C1a)**: system without python3 >=3.11. Run `gdev enable version-sentinel`. Assert: non-zero exit code, error message names specific missing prerequisite(s), no partial files created (atomic: all or none), state file unchanged, `gdev status` still shows version-sentinel as Disabled.
6. **Missing prerequisite — semble (C1b)**: system without python3 >=3.10. Run `gdev enable semble`. Assert: non-zero exit code, error message mentions Python >=3.10 requirement, no .mcp.json entry created, no agent file created, state unchanged.
7. **Unmet detection condition (C2)**: project with no Dockerfile or docker-compose.yml. Run `gdev enable container-security`. Assert: succeeds (explicit enable overrides detection default). May print warning "no Docker ecosystem detected, container-security may not be useful" but does not block. All container-security files created.
8. **Conflicting tools (C3)**: using synthetic test fixtures with explicit `Conflicts` declarations (no current tools conflict, but the mechanism must work). Enable tool-A, then attempt `gdev enable tool-B` where tool-B.Conflicts includes tool-A. Assert: non-zero exit code, error message "Cannot enable tool-B: conflicts with currently enabled tool-A", tool-B not enabled, no files created.
9. **Missing prerequisite — commitlint (C1c)**: system without Node.js. Run `gdev enable commitlint`. Assert: warning or error about missing Node.js, clean failure.

**Acceptance Criteria:**
- [ ] Quick path produces correct default tool set based on AlwaysOn/OnWhenDetected/OptIn policies
- [ ] Customize path toggles drive generation accurately (disabled tools produce no files, enabled tools produce all files)
- [ ] Re-run wizard preserves manual enable/disable changes from saved answers
- [ ] `gdev init --yes` applies smart defaults correctly for detected project type
- [ ] Missing prerequisite: clean failure, non-zero exit, no partial files, state unchanged
- [ ] Missing prerequisite error messages name the specific missing dependency
- [ ] Explicit enable overrides unmet detection condition (user intent respected)
- [ ] Conflicting tool enable blocked with clear error message
- [ ] All error paths leave project in consistent state (no partial artifacts)

**Research Citations:**
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 3. Category H: Wizard Integration`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 3. Category C: Conflict and Error Handling`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 2.3 Ordering Dependencies (Prerequisites)`

**Status:** Not Started

---

### Unit 20.6: Migration, Upgrade & User Modification Tests

**Description:** Verify that the lifecycle system handles legacy projects (pre-lifecycle gdev output), collisions with manually created configs, and user modifications to generated files — especially the critical invariant that user content outside section markers is always preserved.

**Context:** Real projects will encounter three messy scenarios the lifecycle system must handle gracefully: (1) legacy projects bootstrapped by older gdev versions that have no section markers or ownership metadata, (2) projects where the user manually created a config file before gdev adopted the tool, and (3) users who modified generated files (adding custom rules, adjusting settings). The user-content-preservation invariant is the most critical: if `gdev disable` ever deletes user content that was outside markers, trust in the tool is destroyed.

**Desired Outcome:** Legacy projects can adopt lifecycle management without re-running `gdev init`. Manual config collisions are detected and require explicit user intent to overwrite. User content outside markers survives all lifecycle operations.

**Steps:**
1. **Legacy project first lifecycle operation (I1)**: set up a project with pre-lifecycle gdev output (devenv.nix without section markers, CLAUDE.md with `<!-- BEGIN GENERATED SECTION -->` but no tool markers, settings.json with hooks but no ownership metadata). Run `gdev enable semgrep`. Assert: semgrep section markers added for new content, legacy unmarked content treated as core/untouchable, state file created with ownership for new content. Run `gdev disable semgrep`. Assert: semgrep section removed, legacy content preserved. Design decision: only mark newly-added content (do not retroactively wrap existing content in markers — retroactive marking is fragile).
2. **Manual config collision (I2)**: user has manually created `.semgrep.yml` before gdev. Run `gdev enable semgrep`. Assert: error "`.semgrep.yml` already exists and is not gdev-managed. Use `--force` to overwrite". No file overwritten, no partial state. Run with `--force`: assert original file replaced, warning about replacement printed, state updated.
3. **User-created .mcp.json merge (I3)**: user-created `.mcp.json` with `{"mcpServers": {"my-server": {...}}}`. Run `gdev enable context7`. Assert: .mcp.json now has both "my-server" and "context7", "my-server" entry byte-identical to original. Run `gdev disable context7`. Assert: .mcp.json has only "my-server", "context7" removed.
4. **Modified exclusive file on disable (D1)**: enable semgrep, edit `.semgrep.yml` (add a custom rule), run `gdev disable semgrep`. Assert: warning printed about modified file ("`.semgrep.yml` has been modified since generation"), file still deleted (disable proceeds with warning in non-interactive mode), state updated.
5. **Modified shared file section on disable (D2)**: enable semgrep, edit content inside `# --- semgrep ---` markers in devenv.nix, run `gdev disable semgrep`. Assert: warning about modification, entire section between markers removed regardless of user edits within markers (markers are the contract boundary — anything between markers is tool-owned).
6. **User content outside markers preserved (D3)**: enable semgrep (adds section to devenv.nix), add custom content to devenv.nix OUTSIDE of any tool section markers, run `gdev disable semgrep`. Assert: custom content still present, semgrep section removed, devenv.nix still valid Nix. **This is the critical invariant.**
7. **Non-gdev MCP server preserved (D4)**: `.mcp.json` has user-added `"my-custom-server"`. Run `gdev enable context7` then `gdev enable semble` then `gdev disable context7` then `gdev disable semble`. Assert: "my-custom-server" untouched through all 4 operations. Edge case: if all gdev servers removed but user server exists, .mcp.json preserved with only user entry.

**Acceptance Criteria:**
- [ ] Legacy project: first lifecycle operation adds markers for new content without disrupting existing unmarked content
- [ ] Legacy project: disable removes only newly-marked content, legacy content untouched
- [ ] Manual config collision: error by default, `--force` required to overwrite, no silent data loss
- [ ] User-created .mcp.json entries preserved through all enable/disable operations
- [ ] Modified exclusive file: warning on disable, file still removed (non-interactive), state cleaned
- [ ] Modified shared file section: section removed regardless of modifications within markers
- [ ] **User content outside markers always preserved through all operations** (critical invariant)
- [ ] Non-gdev MCP servers never touched by any lifecycle operation
- [ ] Error messages for collisions and modifications are actionable (suggest `--force`, name the file, explain what changed)

**Research Citations:**
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 3. Category D: User Modification Scenarios`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 3. Category I: Upgrade/Migration Testing`
- `artifacts/tool-lifecycle-conflict-matrix-research.md § 6. Open Design Decisions` (decisions 1, 3, 4)

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All six units pass acceptance criteria
- [ ] All 16 tools pass individual lifecycle round-trip (enable → disable → re-enable)
- [ ] All 6 shared file formats survive empty-state testing (all tools removed)
- [ ] All 5 idempotency scenarios pass
- [ ] All 7 cross-tool interaction scenarios pass
- [ ] All 4 wizard integration scenarios pass
- [ ] All 3 migration scenarios pass
- [ ] User modifications outside markers preserved through all operations
- [ ] Error messages for missing prerequisites and conflicts are actionable
- [ ] `gdev enable/disable` complete in < 2 seconds each
- [ ] State file consistent after all test sequences
- [ ] Total: 80+ test scenarios passing
