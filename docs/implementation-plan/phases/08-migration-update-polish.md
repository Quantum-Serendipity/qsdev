# Phase 8: Migration, Update & Polish

## Goal

Implement the `gdev init --update` workflow with per-file merge strategies, team standards versioning, integration tests across the full pipeline, and documentation. At the end of this phase, the system is production-ready: re-runnable, team-scalable, well-tested, and documented.

## Dependencies

Phases 2-5 complete (all generation, wizard, and orchestration working).

## Phase Outputs

- `gdev init --update` command with hash-based modification detection
- Three-way merge for settings.json and .mcp.json
- Section marker merge for CLAUDE.md
- devenv.nix update with .new file + diff (never auto-overwrite)
- Team standards versioning (new templates propagate via update)
- Integration test suite covering full pipeline
- User documentation and team onboarding guide

---

### Unit 6.1: Update Command & Modification Detection

**Description:** Implement `gdev init --update` that re-generates files from saved config, using hash tracking to detect user modifications and route each file to its appropriate merge strategy.

**Context:** The update workflow reads stored `GeneratedState` (from Phase 1, Unit 1.6), computes current file hashes, classifies each file as unmodified/modified/deleted/new, and dispatches to per-file merge strategies.

**Desired Outcome:** `gdev init --update` safely regenerates files without destroying user customizations.

**Steps:**
1. Implement update flow: load saved config → load GeneratedState → generate new files → for each file, check modification status → apply merge strategy → preview → confirm → write → update state.
2. Unmodified files: regenerate (safe — user hasn't touched them).
3. Modified files: route to merge strategy (next units).
4. Deleted files: skip (user intentionally removed — don't recreate unless `--force`).
5. New files (in new version but not in stored state): generate as new.
6. Show diff preview for any file that will change.
7. Support `--force` to override all merge logic and regenerate everything.

**Acceptance Criteria:**
- [ ] Unmodified files are silently regenerated
- [ ] Modified files are routed to merge strategies (not overwritten)
- [ ] Deleted files are not recreated
- [ ] New template files are generated
- [ ] Diff preview shows before confirming changes
- [ ] `--force` overrides all merge logic
- [ ] GeneratedState is updated after successful write

**Research Citations:**
- `research-spikes/gdev-extension-design/migration-strategy-design.md § Update Command Workflow` — complete flow diagram
- `research-spikes/gdev-extension-design/migration-strategy-design.md § Core Principle` — hash-based tracking

**Status:** Not Started

---

### Unit 6.2: Three-Way Merge (settings.json, .mcp.json)

**Description:** Implement three-way merge for JSON files that combine generated content with user additions.

**Context:** settings.json and .mcp.json are machine-owned-with-additions. Users add their own permission rules, hooks, and MCP servers. On update, generated rules should be updated and user rules preserved. Three-way merge uses the stored base (original generated version), theirs (current file on disk), and ours (new generated version).

**Desired Outcome:** `mergeSettings(base, theirs, ours)` produces a correct union of generated and user content.

**Steps:**
1. Implement `mergeSettings(base, theirs, ours SettingsJSON) SettingsJSON`:
   - `permissions.allow`: union of generated + user-added
   - `permissions.deny`: union
   - `hooks`: merge by hook name (update generated, preserve user)
   - `sandbox`: user overrides take precedence
   - New generated keys: added
   - User-added keys: preserved
2. Implement `mergeMcpJson(base, theirs, ours McpJSON) McpJSON`:
   - Generated servers: updated to latest config
   - User-added servers: preserved
   - Removed generated servers: deleted with confirmation
3. Write unit tests: user adds a permission rule → survives update, generated rule changes → updates correctly.

**Acceptance Criteria:**
- [ ] User-added allow rules survive update
- [ ] Generated allow rules update to new values
- [ ] User-added hooks survive update
- [ ] Generated hooks update to new values
- [ ] User-added MCP servers survive update
- [ ] Three-way merge handles conflicting keys (user wins)
- [ ] Unit tests cover add/modify/remove scenarios

**Research Citations:**
- `research-spikes/gdev-extension-design/migration-strategy-design.md § settings.json Three-Way Merge` — merge function spec
- `research-spikes/gdev-extension-design/migration-strategy-design.md § .mcp.json` — same strategy

**Status:** Not Started

---

### Unit 6.3: Section Marker Merge (CLAUDE.md)

**Description:** Implement section marker merge for CLAUDE.md — replace content between `<!-- BEGIN GENERATED SECTION -->` and `<!-- END GENERATED SECTION -->` while preserving everything outside.

**Context:** Users add custom instructions below the generated section. The markers delimit what the addon owns. If markers are missing (user deleted them), treat the entire file as user-owned and skip the update.

**Desired Outcome:** CLAUDE.md updates preserve all user-written content while refreshing generated sections.

**Steps:**
1. Implement `mergeSectionMarkers(existing []byte, newGenerated []byte) ([]byte, error)`:
   - Find `<!-- BEGIN GENERATED SECTION` and `<!-- END GENERATED SECTION -->` markers in existing file.
   - Replace content between markers with new generated content.
   - Preserve everything before first marker and after last marker.
   - If markers missing: return error (caller skips update, warns user).
   - If markers malformed (begin without end): return error.
2. Wire into update flow: when CLAUDE.md is modified, use section marker merge.
3. Write unit tests: user content below markers survives, user content above markers survives, missing markers skips update.

**Acceptance Criteria:**
- [ ] Content between markers is replaced with new generated content
- [ ] Content before markers is preserved
- [ ] Content after markers (user's custom instructions) is preserved
- [ ] Missing markers: file is skipped with warning
- [ ] Malformed markers: file is skipped with warning
- [ ] Empty user section preserved correctly

**Research Citations:**
- `research-spikes/gdev-extension-design/migration-strategy-design.md § CLAUDE.md Section-Based Merge` — marker format and flow

**Status:** Not Started

---

### Unit 6.4: devenv.nix Update Strategy

**Description:** Implement the devenv.nix update strategy: hash check → if unmodified, regenerate; if modified, generate `.devenv.nix.new` and show diff.

**Context:** devenv.nix is the most sensitive file — users heavily customize Nix expressions. Auto-merging arbitrary Nix is impossible (functional language, no general merge algorithm). The strategy is conservative: never auto-overwrite modified devenv.nix.

**Desired Outcome:** Modified devenv.nix is never destroyed; user gets a clear diff showing what changed.

**Steps:**
1. On update, check devenv.nix hash:
   - Unmodified: safe to regenerate in place.
   - Modified: generate to `.devenv.nix.new`, run `diff devenv.nix .devenv.nix.new` and display.
   - Print instructions: "Review the diff and manually merge changes."
2. Support `--force` to overwrite modified devenv.nix (with confirmation).
3. Clean up `.devenv.nix.new` if user chooses to skip.

**Acceptance Criteria:**
- [ ] Unmodified devenv.nix is regenerated in place
- [ ] Modified devenv.nix produces `.devenv.nix.new` file
- [ ] Diff is displayed to user
- [ ] `.devenv.nix.new` is cleaned up when skipped
- [ ] `--force` overwrites with explicit confirmation

**Research Citations:**
- `research-spikes/gdev-extension-design/migration-strategy-design.md § devenv.nix — Human-Edited, Careful` — hash check → .new + diff

**Status:** Not Started

---

### Unit 6.5: Team Standards Versioning

**Description:** Implement the mechanism for team standards to propagate via gdev binary updates.

**Context:** When a team updates their gdev binary (with new templates, skills, security rules), developers run `gdev init --update` to get the latest standards. The skill library has per-skill versioning. Generated standards (deny rules, permission presets) update automatically.

**Desired Outcome:** Team standard updates flow smoothly to all developer projects.

**Steps:**
1. Add version metadata to generated files (in GeneratedState): template version, skill library version.
2. On update: compare stored template version against current binary's template version.
3. If template version changed: re-generate from new templates.
4. Skill library: compare stored manifest version against embedded manifest. Update changed skills.
5. Print summary of what changed: "Updated 3 skills, added 2 deny rules, refreshed CLAUDE.md conventions."

**Acceptance Criteria:**
- [ ] Template version bump triggers re-generation
- [ ] Skill version bump triggers skill file update
- [ ] User-created skills (not in manifest) are untouched
- [ ] Summary clearly states what was updated and why

**Research Citations:**
- `research-spikes/gdev-extension-design/migration-strategy-design.md § Team Standards Versioning` — version tracking
- `research-spikes/gdev-extension-design/claude-code-addon-design.md § Skill Versioning` — per-skill versions

**Status:** Not Started

---

### Unit 6.6: Integration Tests

**Description:** Write integration tests covering the full pipeline: detection → wizard answers → generation → validation → update → merge.

**Context:** Integration tests exercise the complete system, not individual units. They create temp directories with fixture files, run the pipeline with known WizardAnswers, verify generated output, modify files, run update, and verify merge behavior.

**Desired Outcome:** Test suite that catches regressions across the full pipeline.

**Steps:**
1. Test: empty directory → `gdev init --profile go-web --yes` → verify all 7+ files generated with correct content.
2. Test: Go project (go.mod exists) → detection → wizard → verify Go pre-selected.
3. Test: existing devenv.nix → update → verify .new file generated, not overwritten.
4. Test: CLAUDE.md with user content → update → verify user content preserved.
5. Test: settings.json with user permissions → update → verify union merge.
6. Test: non-interactive with all flags → verify same output as wizard.
7. Test: profile + flag override → verify override takes precedence.
8. Test: security configs → verify hardened defaults present in all generated files.

**Acceptance Criteria:**
- [ ] All integration tests pass
- [ ] Empty directory test generates all expected files
- [ ] Detection test correctly identifies Go project
- [ ] Update test preserves user modifications
- [ ] Merge tests verify correct three-way merge, section markers, hash comparison
- [ ] Security defaults verified in every generated config
- [ ] Tests run in <30 seconds

**Research Citations:**
- All phase documents — integration tests exercise the combined system

**Status:** Not Started

---

### Unit 6.7: Documentation & Onboarding Guide

**Description:** Write user documentation: README for the addons, team onboarding guide, and configuration reference.

**Context:** Documentation targets three audiences: developers using `gdev init`, team leads configuring profiles and policies, and security engineers understanding the defense layers.

**Desired Outcome:** Complete documentation enabling self-service adoption.

**Steps:**
1. Write addon README: quick start, command reference, flag reference.
2. Write team onboarding guide: how to configure profiles, add custom skills, set security policies.
3. Write security architecture document: defense layers, what each layer protects against, known limitations.
4. Write configuration reference: all generated files, what each setting does, how to customize.
5. Include migration guide: how to add gdev to an existing project.

**Acceptance Criteria:**
- [ ] README covers quick start in <2 minutes of reading
- [ ] All CLI commands and flags documented
- [ ] Team configuration guide covers profiles, skills, policies
- [ ] Security architecture is standalone-readable
- [ ] Configuration reference covers all generated files

**Research Citations:**
- `research-spikes/devenv-security/trust-model-research.md` — trust model for security docs
- `research-spikes/claude-code-agent-package-guardrails/unified-architecture.md` — defense layer documentation

**Status:** Not Started

---

## Phase Completion Criteria

- [ ] All seven units pass acceptance criteria
- [ ] `gdev init --update` correctly handles all file types (regenerate, merge, skip, .new+diff)
- [ ] Three-way merge preserves user customizations in settings.json
- [ ] Section markers preserve user content in CLAUDE.md
- [ ] devenv.nix is never silently overwritten
- [ ] Team standard updates propagate correctly
- [ ] Integration tests pass
- [ ] Documentation is complete and reviewed
- [ ] The full system is production-ready for team rollout
