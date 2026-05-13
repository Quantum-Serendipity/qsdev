# Tool Lifecycle Testing Strategy — Conflict Matrix & Test Plan

## 1. Complete Tool x File Matrix

### 1.1 File Ownership Summary Table

Each row is a tool. Each column is a file the tool touches. Ownership is marked:
- **E** = Exclusive (tool owns entire file; enable creates, disable deletes)
- **S** = Shared (tool contributes a section/entry; enable inserts, disable removes section)
- **R** = Regenerated (entire file regenerated from all enabled tools on any change)

| Tool | `.semgrep.yml` | `.gitleaks.toml` | `.grype.yaml` | `.cosign/policy.yaml` | `.scancode.yml` | `.license-exceptions.yml` | `secretspec.toml` | `cliff.toml` | `.commitlintrc.yml` | `devenv.nix` | `devenv.yaml` | `.claude/settings.json` | `CLAUDE.md` | `.mcp.json` | `.pre-commit-config.yaml` / devenv hooks | `.github/workflows/*.yml` | `.claude/skills/agent-postmortem/SKILL.md` | `.claude/agents/semble-search.md` | `.claude/hooks/package-guard.py` | `.version-sentinel/ignore` | `.claude/skills/` (ToB) |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **semgrep** | E | | | | | | | | | S | | | S | | S | R | | | | | |
| **gitleaks** | | E | | | | | | | | S | | | S | | S | R | | | | | |
| **container-security** | | | E | E | | | | | | S | | | S | | | R | | | | | |
| **license-compliance** | | | | | E | E | | | | | | | S | | | R | | | | | |
| **attach-guard** | | | | | | | | | | | | S | S | | | | | | E | | |
| **ripsecrets** | | | | | | | | | | | | | | | S | | | | | | |
| **agent-postmortem** | | | | | | | | | | | | | S | | | | E | | | | |
| **version-sentinel** | | | | | | | | | | | | S | S | | | | | | | E | |
| **semble** | | | | | | | | | | | | | S | S | | | | E | | | |
| **context7** | | | | | | | | | | | | | S | S | | | | | | | |
| **github-mcp** | | | | | | | | | | | | | S | S | | | | | | | |
| **socket-dev-mcp** | | | | | | | | | | | | | S | S | | | | | | | |
| **trail-of-bits-skills** | | | | | | | | | | | | | S | | | | | | | | E (multiple) |
| **secretspec** | | | | | | | | | | S | | | S | | | | | | | | |
| **commitlint** | | | | | | | | | E | | | | S | | S (commit-msg) | | | | | | |
| **changelog** | | | | | | | | E | | S | | | S | | | R | | | | | |
| **ci-workflows** | | | | | | | | | | | | | | | | R (virtual) | | | | | |

### 1.2 Per-Tool File Inventory — Enable and Disable Operations

#### semgrep (Security, Default: AlwaysOn)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.semgrep.yml` | Exclusive | Create file with ecosystem-appropriate rule sets + path exclusions |
| `devenv.nix` | Shared | Insert `# --- semgrep ---` section: `semgrep` package |
| `.pre-commit-config.yaml` / devenv hooks | Shared | Insert hook entry: `semgrep --config auto --error` (enhanced tier) |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:semgrep -->` section: usage docs, custom rule patterns |
| `.github/workflows/*.yml` | Regenerated | CI step: `semgrep ci` added to lint-and-sast job |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.semgrep.yml` | Delete file | Warn if hash mismatch (user modified) |
| `devenv.nix` | Remove `# --- semgrep ---` to `# --- end semgrep ---` block | Validate Nix still parses |
| `.pre-commit-config.yaml` / devenv hooks | Remove semgrep hook entry | YAML parse/modify |
| `CLAUDE.md` | Remove `<!-- gdev:semgrep -->` to `<!-- /gdev:semgrep -->` block | |
| `.github/workflows/*.yml` | Full regeneration minus semgrep step | |

---

#### gitleaks (Security, Default: AlwaysOn)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.gitleaks.toml` | Exclusive | Create with allowlists (internal registry URLs, test fixtures, path exclusions) |
| `devenv.nix` | Shared | Insert `# --- gitleaks ---` section: `gitleaks` package |
| `.pre-commit-config.yaml` / devenv hooks | Shared | Insert hook entry: `gitleaks protect --staged` (baseline tier) |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:gitleaks -->` section: false positive management docs |
| `.github/workflows/*.yml` | Regenerated | CI step: `gitleaks detect --source . --report-format sarif` |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.gitleaks.toml` | Delete file | Warn if user added custom allowlist entries |
| `devenv.nix` | Remove `# --- gitleaks ---` block | |
| `.pre-commit-config.yaml` / devenv hooks | Remove gitleaks hook entry | ripsecrets unaffected |
| `CLAUDE.md` | Remove `<!-- gdev:gitleaks -->` block | |
| `.github/workflows/*.yml` | Full regeneration minus gitleaks step | |

---

#### container-security (Security, Default: OnWhenDetected — Docker)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.grype.yaml` | Exclusive | Create with failure threshold `high`, ignored CVEs template |
| `.cosign/policy.yaml` | Exclusive | Create with keyless verification settings |
| `devenv.nix` | Shared | Insert `# --- container-security ---` section: `grype`, `syft`, `cosign` (3 packages) |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:container-security -->` section: Trivy compromise context, pipeline docs |
| `.github/workflows/*.yml` | Regenerated | CI jobs: syft SBOM, grype scan, cosign sign (with OIDC permissions) |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.grype.yaml` | Delete file | |
| `.cosign/policy.yaml` | Delete file; delete `.cosign/` dir if empty | Check for other .cosign files before removing dir |
| `devenv.nix` | Remove `# --- container-security ---` block | Must remove all 3 packages |
| `CLAUDE.md` | Remove `<!-- gdev:container-security -->` block | |
| `.github/workflows/*.yml` | Full regeneration minus container-security jobs | OIDC permissions block may also be removed |

---

#### license-compliance (Security, Default: OptIn)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.scancode.yml` | Exclusive | Create with firm's license policy (allowed/blocked/review lists) |
| `.license-exceptions.yml` | Exclusive | Create empty template for justified exceptions |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:license-compliance -->` section: violation handling, exception process |
| `.github/workflows/*.yml` | Regenerated | CI job: weekly `scancode` scan |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.scancode.yml` | Delete file | |
| `.license-exceptions.yml` | Delete file | Warn loudly if non-empty (user added exceptions) |
| `CLAUDE.md` | Remove `<!-- gdev:license-compliance -->` block | |
| `.github/workflows/*.yml` | Full regeneration minus license-compliance job | |

---

#### attach-guard (AI Agent, Default: AlwaysOn)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.claude/hooks/package-guard.py` | Exclusive | Deploy Python hook script from embedded template |
| `.claude/settings.json` | Shared | Add PreToolUse hook entries pointing to package-guard.py |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:attach-guard -->` section: guardrail docs |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.claude/hooks/package-guard.py` | Delete file | Warn if user modified (hash mismatch) |
| `.claude/settings.json` | Remove attach-guard hook entries from Hooks map | JSON parse/modify |
| `CLAUDE.md` | Remove `<!-- gdev:attach-guard -->` block | |

---

#### ripsecrets (Security — Phase 5 baseline, pre-commit hook only)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.pre-commit-config.yaml` / devenv hooks | Shared | Insert ripsecrets hook entry (baseline tier) |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.pre-commit-config.yaml` / devenv hooks | Remove ripsecrets hook entry | No exclusive files to clean |

Note: ripsecrets has no exclusive files, no CLAUDE.md section, no devenv.nix package (it's pulled as a pre-commit hook dependency). This is the simplest tool in the registry.

---

#### agent-postmortem (AI Agent, Default: AlwaysOn)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.claude/skills/agent-postmortem/SKILL.md` | Exclusive | Deploy templated SKILL.md with per-ecosystem verification commands |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:agent-postmortem -->` section: verification protocol docs |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.claude/skills/agent-postmortem/SKILL.md` | Delete file | |
| `.claude/skills/agent-postmortem/` | Delete directory if empty | |
| `CLAUDE.md` | Remove `<!-- gdev:agent-postmortem -->` block | |

---

#### version-sentinel (AI Agent, Default: AlwaysOn)

Prerequisites: python3 >=3.11, jq

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.claude/settings.json` | Shared | Add PreToolUse hook entries for manifest edit interception (Edit/Write/MultiEdit matchers) |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:version-sentinel -->` section: install instructions, coverage notes, recovery workflow |
| `.version-sentinel/ignore` | Exclusive | Create with private package patterns |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.claude/settings.json` | Remove version-sentinel hook entries | JSON parse/modify |
| `CLAUDE.md` | Remove `<!-- gdev:version-sentinel -->` block | |
| `.version-sentinel/ignore` | Delete file | |
| `.version-sentinel/` | Delete directory if empty | |

---

#### semble (AI Agent, Default: OnWhenDetected — Python >=3.10)

Prerequisites: python3 >=3.10

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.mcp.json` | Shared | Add `"semble"` server entry (uvx-based MCP config) |
| `.claude/agents/semble-search.md` | Exclusive | Deploy sub-agent definition file |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:semble -->` section: semantic search docs |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.mcp.json` | Remove `"semble"` key from mcpServers object | JSON parse/modify |
| `.claude/agents/semble-search.md` | Delete file | |
| `.claude/agents/` | Delete directory if empty | Check for other agent files first |
| `CLAUDE.md` | Remove `<!-- gdev:semble -->` block | |

---

#### context7 (AI Agent, Default: AlwaysOn)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.mcp.json` | Shared | Add `"context7"` server entry (`npx -y @upstash/context7-mcp`) |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:context7 -->` section: library docs provider |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.mcp.json` | Remove `"context7"` key | JSON parse/modify |
| `CLAUDE.md` | Remove `<!-- gdev:context7 -->` block | |

---

#### github-mcp (AI Agent, Default: AlwaysOn)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.mcp.json` | Shared | Add `"github"` server entry |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:github-mcp -->` section |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.mcp.json` | Remove `"github"` key | JSON parse/modify |
| `CLAUDE.md` | Remove `<!-- gdev:github-mcp -->` block | |

---

#### socket-dev-mcp (AI Agent, Default: OnWhenDetected — JS/Python/Rust/Go)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.mcp.json` | Shared | Add `"socket-dev"` server entry |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:socket-dev-mcp -->` section |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.mcp.json` | Remove `"socket-dev"` key | JSON parse/modify |
| `CLAUDE.md` | Remove `<!-- gdev:socket-dev-mcp -->` block | |

---

#### trail-of-bits-skills (AI Agent, Default: AlwaysOn)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.claude/skills/supply-chain-risk-auditor.md` | Exclusive | Deploy skill file |
| `.claude/skills/differential-review.md` | Exclusive | Deploy skill file |
| `.claude/skills/insecure-defaults.md` | Exclusive | Deploy skill file |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:trail-of-bits-skills -->` section |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.claude/skills/supply-chain-risk-auditor.md` | Delete file | |
| `.claude/skills/differential-review.md` | Delete file | |
| `.claude/skills/insecure-defaults.md` | Delete file | |
| `CLAUDE.md` | Remove `<!-- gdev:trail-of-bits-skills -->` block | |

Note: These are multiple exclusive files owned by a single tool. The tool registry must track all of them as owned files. Disabling must remove all of them.

---

#### secretspec (DevEx, Default: OnWhenDetected — services present)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `secretspec.toml` | Exclusive | Create with secret declarations from detected services/ecosystems |
| `devenv.nix` | Shared | Insert `# --- secretspec ---` section: secretspec integration block |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:secretspec -->` section: provider docs, SOPS limitation |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `secretspec.toml` | Delete file | Warn if user added custom secret declarations |
| `devenv.nix` | Remove `# --- secretspec ---` block | |
| `CLAUDE.md` | Remove `<!-- gdev:secretspec -->` block | |

---

#### commitlint (DevEx, Default: OptIn)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `.commitlintrc.yml` | Exclusive | Create with conventional commit format config |
| `.pre-commit-config.yaml` / devenv hooks | Shared | Insert commit-msg hook: `commitlint --edit $1` |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:commitlint -->` section |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `.commitlintrc.yml` | Delete file | |
| `.pre-commit-config.yaml` / devenv hooks | Remove commitlint commit-msg hook | |
| `CLAUDE.md` | Remove `<!-- gdev:commitlint -->` block | |

---

#### changelog (DevEx, Default: OptIn)

Suggests: `commitlint` (non-blocking recommendation)

**On enable:**
| File | Ownership | Operation |
|------|-----------|-----------|
| `cliff.toml` | Exclusive | Create with firm's standard format |
| `devenv.nix` | Shared | Insert `# --- changelog ---` section: `git-cliff` package |
| `CLAUDE.md` | Shared | Insert `<!-- gdev:changelog -->` section |
| `.github/workflows/*.yml` | Regenerated | CI step: `git-cliff --latest` on tag push |

**On disable:**
| File | Operation | Notes |
|------|-----------|-------|
| `cliff.toml` | Delete file | |
| `devenv.nix` | Remove `# --- changelog ---` block | |
| `CLAUDE.md` | Remove `<!-- gdev:changelog -->` block | |
| `.github/workflows/*.yml` | Full regeneration minus changelog step | |

---

#### ci-workflows (Infrastructure, Virtual — Always enabled)

This is a virtual/meta-tool that is always enabled and does not appear in `gdev enable`/`gdev disable`. It owns the CI workflow files exclusively but regenerates them based on the full set of currently enabled tools.

| File | Ownership | Operation |
|------|-----------|-----------|
| `.github/workflows/security.yml` | Exclusive (virtual) | Regenerate from all enabled tools |
| `.gitlab-ci.yml` | Exclusive (virtual) | Regenerate from all enabled tools (when GitLab CI selected) |

Harden-Runner step is always present as the first step in every job regardless of which tools are enabled.

---

### 1.3 Shared File Contribution Summary

This table inverts the matrix: for each shared file, which tools write to it and what surgery is needed.

#### devenv.nix (Section markers: `# --- <tool> ---` / `# --- end <tool> ---`)

| Tool | Section Content |
|------|----------------|
| semgrep | `pkgs.semgrep` in packages list |
| gitleaks | `pkgs.gitleaks` in packages list |
| container-security | `pkgs.grype`, `pkgs.syft`, `pkgs.cosign` in packages list |
| secretspec | Secretspec integration block in body |
| changelog | `pkgs.git-cliff` in packages list |

Total: 5 tools contribute to devenv.nix. Maximum 7 package entries (if all enabled).

#### .claude/settings.json (JSON parse/modify by hook key)

| Tool | Contribution |
|------|-------------|
| attach-guard | PreToolUse hook entries referencing package-guard.py |
| version-sentinel | PreToolUse hook entries for manifest edit interception |

Total: 2 tools contribute hooks. Hooks use different matchers (Bash vs Edit/Write/MultiEdit) so they do not conflict at the matcher level.

#### CLAUDE.md (Section markers: `<!-- gdev:<tool> -->` / `<!-- /gdev:<tool> -->`)

| Tool | Section Content |
|------|----------------|
| semgrep | SAST usage, custom rule patterns |
| gitleaks | False positive management, allowlist editing |
| container-security | Trivy compromise context, pipeline usage |
| license-compliance | Violation handling, exception process |
| attach-guard | Package guardrail documentation |
| agent-postmortem | Verification protocol documentation |
| version-sentinel | Install instructions, coverage notes, recovery workflow |
| semble | Semantic search documentation |
| context7 | Library docs provider documentation |
| github-mcp | GitHub integration documentation |
| socket-dev-mcp | Supply chain risk documentation |
| trail-of-bits-skills | Security audit skills documentation |
| secretspec | Provider options, SOPS limitation |
| commitlint | Conventional commit format documentation |
| changelog | Changelog automation documentation |

Total: 15 tools (every tool except ripsecrets and ci-workflows) contribute to CLAUDE.md.

#### .mcp.json (JSON parse/modify by server key)

| Tool | Server Key |
|------|-----------|
| semble | `"semble"` |
| context7 | `"context7"` |
| github-mcp | `"github"` |
| socket-dev-mcp | `"socket-dev"` |

Total: 4 tools contribute entries. Additional user-added servers (not gdev-managed) must be preserved.

#### .pre-commit-config.yaml / devenv git-hooks (YAML parse/modify)

| Tool | Hook Entry | Hook Type | Tier |
|------|-----------|-----------|------|
| ripsecrets | `ripsecrets` | pre-commit | Baseline |
| gitleaks | `gitleaks protect --staged` | pre-commit | Baseline |
| semgrep | `semgrep --config auto --error` | pre-commit | Enhanced |
| commitlint | `commitlint --edit $1` | commit-msg | Baseline (opt-in) |

Total: 4 tools contribute hooks. Note: commitlint uses `commit-msg` hook type, not `pre-commit`, so it is in a distinct hook category.

#### CI Workflows (Full regeneration)

| Tool | CI Contribution |
|------|----------------|
| semgrep | Step in lint-and-sast job |
| gitleaks | Step in secret-scan job |
| container-security | Separate container-security job (syft + grype + cosign) |
| license-compliance | Separate weekly job |
| changelog | Step on tag push |
| ci-workflows (Harden-Runner) | First step in every job (always present) |

Total: 5 tools contribute CI content, plus Harden-Runner is always present.

---

## 2. Conflict Detection Analysis

### 2.1 Explicit Conflicts (from Tool registry `Conflicts` field)

Based on the Phase 12 design, **no tools have explicit mutual exclusion conflicts** declared. The architecture was deliberately designed to avoid conflicts — each tool operates in its own domain. However, the registry should declare conflicts for future extensibility (e.g., if an alternative SAST tool were added alongside Semgrep).

**Potential future conflict declarations:**
- If `trivy` were ever re-added: Conflicts with `container-security` (Grype)
- If `snyk` were added: Conflicts with `socket-dev-mcp` (overlapping supply chain analysis)
- If `release-please` were added: Conflicts with `changelog` (competing changelog strategies)

### 2.2 Implicit Conflicts — Same Shared File Section

No two tools write to the same *section* of a shared file. Each tool has a unique section ID. However, the following subtle interaction points exist:

#### Pre-commit hook ordering
- `gitleaks` (baseline) and `semgrep` (enhanced) both add pre-commit hooks. They do not conflict but ordering matters for user experience: gitleaks should run first (faster, <1s) so commits fail fast on secrets before the slower Semgrep scan (~3s).
- `ripsecrets` (baseline) and `gitleaks` (baseline) are explicitly co-existing by design ("belt and suspenders"). Ripsecrets runs in ~10ms and should come first.
- **Required order**: ripsecrets -> gitleaks -> semgrep
- `commitlint` uses `commit-msg` hook type, not `pre-commit`, so it has no ordering interaction with the others.

#### settings.json hook ordering
- `attach-guard` and `version-sentinel` both register PreToolUse hooks but use different matchers:
  - attach-guard: Bash command matcher (package install commands)
  - version-sentinel: Edit/Write/MultiEdit matcher (manifest file edits) + Bash matcher (install commands)
- **Potential interaction**: Both may fire on the same `npm install <pkg>` command. attach-guard checks age + OSV, version-sentinel checks version freshness. If both block, the user gets two error messages. This is noisy but not broken — both are providing independent safety checks.
- **Test needed**: Verify both hooks fire and their error messages don't confuse the user.

#### .mcp.json performance
- Having all 4 MCP servers enabled simultaneously is valid but pushes toward the "3-6 is sweet spot, >10 degrades performance" guideline. 4 servers is well within the recommended range.
- **No conflict**, but `gdev status` should display total active MCP server count.

### 2.3 Ordering Dependencies (Prerequisites)

| Tool | Prerequisites | Behavior on Missing Prerequisite |
|------|--------------|----------------------------------|
| version-sentinel | python3 >=3.11, jq, curl | `gdev enable version-sentinel` fails with message listing missing prereqs; `gdev devenv doctor` reports them |
| semble | python3 >=3.10, uvx | `gdev enable semble` fails with message; `gdev devenv doctor` reports |
| container-security | docker | `gdev enable container-security` warns (Docker detection drives default); may succeed for CI-only use |
| secretspec | devenv >=2.0 | `gdev enable secretspec` warns if devenv version too old |
| commitlint | Node.js (for commitlint binary) | `gdev enable commitlint` fails if no Node runtime |

No tool has a prerequisite on *another gdev tool*. The prerequisite chain is entirely about system-level tooling availability.

### 2.4 Soft Dependencies (Suggestions, Not Requirements)

| Tool | Suggests | Nature of Suggestion |
|------|---------|---------------------|
| changelog | commitlint | "For best results, enable conventional commit enforcement: `gdev enable commitlint`" — printed as info message, not a blocker |

This is the only cross-tool suggestion in the registry. changelog works without commitlint (falls back to simple commit list), but produces better output with structured commits.

### 2.5 devenv.yaml Input Dependencies

Enabling any tool that contributes pre-commit/commit-msg hooks (gitleaks, semgrep, ripsecrets, commitlint) requires that `devenv.yaml` includes the `git-hooks` input. The lifecycle system must ensure:
- First hook-contributing tool enabled → add `git-hooks` input to devenv.yaml
- Last hook-contributing tool disabled → remove `git-hooks` input from devenv.yaml (or leave it; removing is optional since an empty hooks config is harmless)

---

## 3. State Transition Test Scenarios

### Category A: Individual Tool Lifecycle (Fresh Project)

For **each** of the 15+ tools, run this test sequence:

#### A1. Fresh enable → verify → disable → verify clean removal

```
Test: A1-{tool_name}
Setup: Fresh project (gdev init --yes, then gdev disable {tool} if it was default-on)
Steps:
  1. gdev enable {tool_name}
  2. Assert: All exclusive files exist at expected paths
  3. Assert: All shared file sections present (grep markers/keys)
  4. Assert: State file records file ownership for all files
  5. Assert: gdev status shows {tool_name} as Enabled
  6. gdev disable {tool_name}
  7. Assert: All exclusive files deleted
  8. Assert: All shared file sections removed (grep markers/keys absent)
  9. Assert: Shared files still valid (parse test: Nix eval, JSON parse, YAML parse)
  10. Assert: State file no longer references {tool_name} ownership
  11. Assert: gdev status shows {tool_name} as Disabled
Expected: Clean round-trip with no orphaned artifacts
```

Run for: semgrep, gitleaks, container-security, license-compliance, attach-guard, ripsecrets, agent-postmortem, version-sentinel, semble, context7, github-mcp, socket-dev-mcp, trail-of-bits-skills, secretspec, commitlint, changelog

#### A2. Re-enable a previously disabled tool

```
Test: A2-{tool_name}
Setup: Complete A1 (tool is now disabled)
Steps:
  1. gdev enable {tool_name}
  2. Assert: All files recreated identically to A1 step 2-4
  3. Assert: Shared file sections present and correctly positioned
  4. Assert: State file updated
Expected: Re-enable produces identical result to first enable
```

#### A3. Rapid toggle stress test

```
Test: A3-{tool_name}
Steps:
  1. gdev enable {tool_name}
  2. gdev disable {tool_name}
  3. gdev enable {tool_name}
  4. gdev disable {tool_name}
  5. Assert: Project state identical to never-enabled state
  6. Assert: No orphaned section markers in any shared file
  7. Assert: No empty sections (e.g., `# --- semgrep ---\n# --- end semgrep ---` with nothing between)
Expected: 4 toggles leave project clean
```

### Category B: Bulk Operations

#### B1. Enable all defaults → disable one by one → verify clean

```
Test: B1-cascading-disable
Setup: gdev init --yes (all AlwaysOn + detected OnWhenDetected tools enabled)
Steps:
  1. Record enabled tool set (expected: semgrep, gitleaks, attach-guard, agent-postmortem,
     version-sentinel, context7, github-mcp, trail-of-bits-skills, ripsecrets,
     plus detected: container-security, semble, socket-dev-mcp, secretspec)
  2. For each enabled tool (in reverse priority order):
     a. gdev disable {tool}
     b. Assert: That tool's artifacts removed
     c. Assert: All other tools' artifacts still intact
     d. Assert: All shared files still valid (parse test)
  3. After all disabled:
     a. Assert: devenv.nix has no tool sections (only core framework content)
     b. Assert: CLAUDE.md has no tool sections (only core + user content)
     c. Assert: .mcp.json either empty object `{}` or deleted
     d. Assert: settings.json has no tool hooks (only deny rules from Phase 4)
     e. Assert: Pre-commit config has no tool hooks (only baseline framework hooks, if any)
Expected: Sequential disable leaves clean project with only core framework files
```

#### B2. Enable all tools simultaneously

```
Test: B2-enable-all
Setup: Fresh project, all tools disabled
Steps:
  1. For each tool: gdev enable {tool}
  2. Assert: devenv.nix valid Nix with all 5 tool sections
  3. Assert: .mcp.json valid JSON with 4 server entries
  4. Assert: settings.json valid JSON with attach-guard + version-sentinel hooks
  5. Assert: CLAUDE.md has 15 tool sections, all properly nested (no overlapping markers)
  6. Assert: Pre-commit config has 4 hook entries in correct order
  7. Assert: CI workflow has all steps from all 5 CI-contributing tools
Expected: All tools coexist without file corruption
```

### Category C: Conflict and Error Handling

#### C1. Enable tool with missing prerequisite

```
Test: C1-missing-prereq-version-sentinel
Setup: System without python3 >=3.11 or jq
Steps:
  1. gdev enable version-sentinel
  2. Assert: Command returns non-zero exit code
  3. Assert: Error message names specific missing prerequisite(s)
  4. Assert: No partial files created (atomic: either all files or none)
  5. Assert: State file unchanged
  6. Assert: gdev status still shows version-sentinel as Disabled
Expected: Clean failure, no side effects
```

```
Test: C1-missing-prereq-semble
Setup: System without python3 >=3.10
Steps:
  1. gdev enable semble
  2. Assert: Command returns non-zero exit code
  3. Assert: Error message mentions Python >=3.10 requirement
  4. Assert: No .mcp.json entry created, no agent file created
Expected: Clean failure
```

```
Test: C1-missing-prereq-commitlint
Setup: System without Node.js
Steps:
  1. gdev enable commitlint
  2. Assert: Warning or error about missing Node.js
Expected: At minimum a clear warning
```

#### C2. Enable tool whose detection condition is unmet

```
Test: C2-container-no-docker
Setup: Project with no Dockerfile or docker-compose.yml
Steps:
  1. gdev enable container-security
  2. Assert: Succeeds (explicit enable overrides detection default)
  3. OR Assert: Warning "no Docker ecosystem detected, container-security may not be useful"
Expected: Explicit enable is allowed even when auto-detection wouldn't trigger it. User intent is respected.
```

#### C3. Enable conflicting tools (future-proofing)

```
Test: C3-conflict-declaration
Setup: Two tools with explicit Conflicts declarations
Steps:
  1. gdev enable tool-A
  2. gdev enable tool-B (where tool-B.Conflicts includes tool-A)
  3. Assert: Command returns non-zero exit code
  4. Assert: Error message: "Cannot enable tool-B: conflicts with currently enabled tool-A"
  5. Assert: tool-B not enabled, no files created
Expected: Conflict detected and blocked
```

Note: No current tools conflict, so this test validates the conflict-checking mechanism using synthetic test fixtures or by temporarily modifying the registry.

### Category D: User Modification Scenarios

#### D1. Enable → user modifies exclusive file → disable

```
Test: D1-modified-exclusive
Setup: gdev enable semgrep
Steps:
  1. Edit .semgrep.yml: add a custom rule
  2. gdev disable semgrep
  3. Assert: Warning printed: ".semgrep.yml has been modified since generation"
  4. Assert: File IS deleted (disable still proceeds, but warns)
  5. OR Assert: Prompt for confirmation before deleting modified file
Expected: User warned about data loss. File deleted (or user confirms deletion).
Decision needed: Should disable prompt for confirmation on modified exclusive files, or just warn?
```

#### D2. Enable → user modifies shared file section → disable

```
Test: D2-modified-shared-section
Setup: gdev enable semgrep
Steps:
  1. Edit devenv.nix: modify content inside `# --- semgrep ---` markers
  2. gdev disable semgrep
  3. Assert: Warning about modification
  4. Assert: Entire section between markers removed (regardless of user edits within markers)
Expected: Section markers are the contract boundary. Anything between markers is tool-owned.
```

#### D3. Enable → user adds content outside markers → disable

```
Test: D3-user-content-preserved
Setup: gdev enable semgrep (adds section to devenv.nix)
Steps:
  1. Add custom content to devenv.nix OUTSIDE of any tool section markers
  2. gdev disable semgrep
  3. Assert: Custom content still present
  4. Assert: semgrep section removed
  5. Assert: devenv.nix still valid Nix
Expected: User content outside markers is always preserved. This is the critical invariant.
```

#### D4. User adds non-gdev MCP server → enable/disable gdev MCP tools

```
Test: D4-preserve-user-mcp
Setup: .mcp.json exists with user-added server: `"my-custom-server": {...}`
Steps:
  1. gdev enable context7
  2. Assert: .mcp.json has both "my-custom-server" and "context7"
  3. gdev disable context7
  4. Assert: .mcp.json still has "my-custom-server"
  5. Assert: "context7" removed
Expected: Non-gdev entries in .mcp.json preserved through all operations
```

### Category E: Shared File Integrity (Empty State Testing)

#### E1. devenv.nix — all tool sections removed

```
Test: E1-devenv-nix-empty-tools
Setup: Enable all tools that contribute to devenv.nix (semgrep, gitleaks, container-security, secretspec, changelog)
Steps:
  1. Disable all 5 tools
  2. Assert: devenv.nix still exists (it's a core framework file, not tool-owned)
  3. Assert: devenv.nix is valid Nix (run `nix eval --file devenv.nix` or similar)
  4. Assert: Core content remains (language packages, services, enterShell, etc.)
  5. Assert: No empty section markers remain
Expected: devenv.nix is owned by core framework; tool sections are additions. File remains valid with all sections removed.

Structural requirement: The Nix template must produce valid syntax whether 0 or N tool sections are present. This means:
- Package list must not have trailing comma issues when tool packages removed
- enterShell must not have orphaned newlines
- Section markers must be placed at syntactically safe boundaries (between list items, not mid-expression)
```

#### E2. settings.json — all tool hooks removed

```
Test: E2-settings-json-empty-hooks
Setup: Enable attach-guard + version-sentinel
Steps:
  1. Disable both tools
  2. Assert: settings.json still exists
  3. Assert: Valid JSON (json.Unmarshal succeeds)
  4. Assert: Deny rules still present (Phase 4 core content, not tool-owned)
  5. Assert: Hooks map exists but may be empty or absent
  6. Assert: Permission presets still present
Expected: settings.json retains all non-hook content. Empty hooks map is valid.
```

#### E3. CLAUDE.md — all tool sections removed

```
Test: E3-claude-md-empty-tools
Setup: Enable all 15 tools that contribute to CLAUDE.md
Steps:
  1. Disable all 15 tools
  2. Assert: CLAUDE.md still exists
  3. Assert: Core generated section (between `<!-- BEGIN GENERATED SECTION -->` / `<!-- END GENERATED SECTION -->`) still present with core content (build commands, security instructions, language conventions)
  4. Assert: User custom section (below markers) preserved
  5. Assert: No orphaned `<!-- gdev:* -->` markers
  6. Assert: No empty lines where sections were (or at most single blank line)
Expected: CLAUDE.md core content and user content intact. Tool sections cleanly removed.
```

#### E4. .mcp.json — all MCP servers removed

```
Test: E4-mcp-json-empty
Setup: Enable context7, github-mcp, socket-dev-mcp, semble (all 4 MCP tools)
Steps:
  1. Disable all 4 MCP tools
  2a. If user had non-gdev entries: Assert .mcp.json preserved with only user entries
  2b. If no non-gdev entries:
     - Option A: Assert .mcp.json deleted (empty .mcp.json is useless)
     - Option B: Assert .mcp.json contains `{"mcpServers": {}}` (valid but empty)
  3. Assert: No reference to any gdev-managed server key remains
Expected: Either clean empty state or file deletion. 

Recommendation: Delete .mcp.json when mcpServers would be empty AND no non-gdev entries exist. This is cleaner than leaving an empty file.
```

#### E5. .pre-commit-config.yaml — all tool hooks removed

```
Test: E5-precommit-empty-tools
Setup: Enable ripsecrets, gitleaks, semgrep, commitlint
Steps:
  1. Disable all 4 hook-contributing tools
  2. Assert: If framework-level hooks exist (check-added-large-files, no-commit-to-branch, check-merge-conflict), file remains with only those
  3. Assert: If no hooks remain, either:
     - Delete the file AND remove git-hooks input from devenv.yaml
     - Leave file with empty repos list (valid YAML but useless)
  4. Assert: YAML is valid (yaml.Unmarshal succeeds)
Expected: Framework hooks (Phase 5 baseline) are NOT tool-owned — they're owned by "core". Only tool-contributed hooks are removed.

Critical distinction: ripsecrets, gitleaks, semgrep, commitlint are tools with hooks. But check-added-large-files, no-commit-to-branch, check-merge-conflict are "core" framework hooks that exist regardless of tool state. Disabling all tools should NOT remove these core hooks.
```

#### E6. CI workflow — minimal state

```
Test: E6-ci-minimal
Setup: Disable all CI-contributing tools (semgrep, gitleaks, container-security, license-compliance, changelog)
Steps:
  1. Assert: CI workflow still exists (ci-workflows is always-on virtual tool)
  2. Assert: Workflow contains Harden-Runner as first step (always present)
  3. Assert: Workflow may contain ecosystem-level checks (OSV Scanner, frozen-install) from Phase 5 core
  4. Assert: No tool-specific steps remain
  5. Assert: Valid GitHub Actions YAML
Expected: Minimal CI workflow with only Harden-Runner + ecosystem-level scanning from core framework. The workflow file exists even when all toggleable tools are disabled.
```

### Category F: Idempotency Testing

#### F1. Enable already-enabled tool

```
Test: F1-enable-idempotent
Setup: gdev enable semgrep (semgrep now enabled)
Steps:
  1. gdev enable semgrep
  2. Assert: Exit code 0 (success, not error)
  3. Assert: Message like "semgrep is already enabled" (informational, not warning)
  4. Assert: All files unchanged (hash comparison)
  5. Assert: State file unchanged
Expected: No-op, no error
```

#### F2. Disable already-disabled tool

```
Test: F2-disable-idempotent
Setup: gdev disable semgrep (semgrep now disabled)
Steps:
  1. gdev disable semgrep
  2. Assert: Exit code 0
  3. Assert: Message like "semgrep is already disabled"
  4. Assert: No file changes
Expected: No-op, no error
```

#### F3. Enable twice in succession

```
Test: F3-double-enable
Steps:
  1. gdev enable semgrep
  2. Record all file hashes
  3. gdev enable semgrep
  4. Assert: All file hashes identical to step 2
  5. Assert: No duplicate sections in shared files (e.g., no two `# --- semgrep ---` blocks in devenv.nix)
Expected: Second enable is a pure no-op
```

#### F4. Init then enable (tool already enabled by init defaults)

```
Test: F4-init-then-enable
Steps:
  1. gdev init --yes (semgrep enabled by default as AlwaysOn)
  2. Record state
  3. gdev enable semgrep
  4. Assert: No-op
  5. Assert: State unchanged
Expected: Enable recognizes tool is already enabled via init
```

#### F5. Init --yes idempotency

```
Test: F5-init-yes-idempotent
Steps:
  1. gdev init --yes
  2. Record all file hashes
  3. gdev init --yes (re-run)
  4. Assert: All files identical (or properly merged if update mode)
Expected: Re-running init doesn't duplicate sections or corrupt shared files
```

### Category G: Cross-Tool Interaction Testing

#### G1. Pre-commit hook ordering

```
Test: G1-hook-order
Steps:
  1. gdev enable ripsecrets
  2. gdev enable gitleaks
  3. gdev enable semgrep
  4. Read .pre-commit-config.yaml / devenv hooks config
  5. Assert: Hook order is ripsecrets → gitleaks → semgrep
  6. Now disable gitleaks and re-enable:
  7. gdev disable gitleaks
  8. gdev enable gitleaks
  9. Assert: Hook order is STILL ripsecrets → gitleaks → semgrep (order preserved regardless of enable sequence)
Expected: Hook ordering is deterministic, not insertion-order-dependent. Tools define their tier/priority, and the hook generation sorts accordingly.
```

#### G2. Container security multi-package removal

```
Test: G2-container-security-packages
Steps:
  1. gdev enable container-security
  2. Assert: devenv.nix `# --- container-security ---` section contains grype, syft, cosign (3 packages)
  3. gdev disable container-security
  4. Assert: All 3 packages removed (not just first, not just last)
  5. Assert: devenv.nix valid Nix
Expected: Composite tool removes all its contributed packages, not a subset
```

#### G3. changelog + commitlint interaction

```
Test: G3-changelog-commitlint
Steps:
  1. gdev enable changelog
  2. Assert: Info message suggesting commitlint enablement
  3. Assert: changelog works without commitlint (no error, no broken dependency)
  4. gdev enable commitlint
  5. Assert: Both tools' files present, no conflicts
  6. gdev disable commitlint
  7. Assert: commitlint files removed
  8. Assert: changelog still functional (cliff.toml still present, CI step still present)
  9. Assert: No error or warning about missing commitlint (it's a suggestion, not a dependency)
Expected: Soft dependency doesn't create hard coupling
```

#### G4. All MCP servers enabled — .mcp.json well-formedness

```
Test: G4-all-mcp
Steps:
  1. gdev enable context7
  2. gdev enable github-mcp
  3. gdev enable socket-dev-mcp
  4. gdev enable semble
  5. Assert: .mcp.json is valid JSON
  6. Assert: 4 server entries under "mcpServers"
  7. Assert: No duplicate keys
  8. Assert: Each server has correct command/args
  9. gdev status → Assert: Shows 4 MCP servers
Expected: All MCP servers coexist in well-formed JSON
```

#### G5. attach-guard and version-sentinel hook coexistence

```
Test: G5-dual-hooks
Steps:
  1. gdev enable attach-guard
  2. gdev enable version-sentinel
  3. Assert: settings.json has hooks from both tools
  4. Assert: No duplicate hook entries
  5. Assert: Each tool's hooks have distinct matchers
  6. Simulate: npm install command → both hooks should fire
  7. gdev disable attach-guard
  8. Assert: Only version-sentinel hooks remain
  9. Assert: settings.json valid JSON
Expected: Independent hook sets that don't interfere
```

#### G6. Enable tool that contributes to all shared files

```
Test: G6-full-surface-tool
Tool: semgrep (touches devenv.nix, CLAUDE.md, pre-commit, CI)
Steps:
  1. gdev enable semgrep
  2. Assert: Section present in devenv.nix
  3. Assert: Section present in CLAUDE.md
  4. Assert: Hook present in pre-commit config
  5. Assert: Step present in CI workflow
  6. gdev disable semgrep
  7. Assert: All 4 shared files cleaned
  8. Assert: All 4 shared files still valid
Expected: Tool with maximum shared-file surface area enables/disables cleanly across all files
```

#### G7. CI workflow regeneration consistency

```
Test: G7-ci-regen-consistency
Steps:
  1. gdev enable semgrep
  2. gdev enable gitleaks
  3. Record CI workflow hash (workflow-A)
  4. gdev disable gitleaks
  5. gdev enable gitleaks
  6. Record CI workflow hash (workflow-B)
  7. Assert: workflow-A == workflow-B (same set of enabled tools → same workflow)
Expected: CI regeneration is deterministic — same inputs produce same output
```

### Category H: Wizard Integration Testing

#### H1. Quick path defaults

```
Test: H1-quick-path
Setup: Fresh Go + TypeScript + Docker project
Steps:
  1. gdev init (quick path — accept defaults)
  2. Assert: AlwaysOn tools enabled: semgrep, gitleaks, attach-guard, agent-postmortem,
     version-sentinel, context7, github-mcp, trail-of-bits-skills, ripsecrets
  3. Assert: OnWhenDetected tools enabled based on detection:
     - container-security: YES (Docker detected)
     - semble: YES if Python >=3.10 available, NO otherwise
     - socket-dev-mcp: YES (Go + TS detected)
     - secretspec: only if services detected
  4. Assert: OptIn tools disabled: license-compliance, commitlint, changelog
  5. Assert: All enabled tools' files generated correctly
Expected: Smart defaults produce correct tool set
```

#### H2. Customize path toggles

```
Test: H2-customize
Steps:
  1. gdev init (customize path)
  2. Toggle: disable semgrep, enable license-compliance, enable changelog
  3. Assert: semgrep files NOT generated
  4. Assert: license-compliance files generated (.scancode.yml, .license-exceptions.yml, CLAUDE.md section)
  5. Assert: changelog files generated (cliff.toml, devenv.nix section, CLAUDE.md section)
  6. Assert: CI workflow has license-compliance + changelog steps but NOT semgrep step
Expected: Wizard choices correctly drive generation
```

#### H3. Re-run wizard on existing project

```
Test: H3-wizard-rerun
Setup: gdev init --yes (defaults)
Steps:
  1. gdev disable semgrep (manual change after init)
  2. gdev init --update (or re-run wizard)
  3. Assert: semgrep remains disabled (user's explicit disable is preserved)
  4. Assert: All other tools unchanged
  5. Assert: No duplicate sections in any shared file
Expected: --update respects current enable/disable state from saved answers
```

#### H4. gdev init --yes flag behavior

```
Test: H4-init-yes
Setup: Fresh project (Go + Docker detected, Python 3.11 available)
Steps:
  1. gdev init --yes
  2. Assert: All AlwaysOn enabled
  3. Assert: container-security enabled (Docker detected)
  4. Assert: semble enabled (Python >=3.10 detected)
  5. Assert: version-sentinel enabled (Python >=3.11 detected, supported ecosystems)
  6. Assert: socket-dev-mcp enabled (Go detected)
  7. Assert: license-compliance DISABLED (OptIn)
  8. Assert: commitlint DISABLED (OptIn)
  9. Assert: changelog DISABLED (OptIn)
Expected: --yes applies smart defaults without interaction
```

### Category I: Upgrade/Migration Testing

#### I1. Legacy project — first lifecycle operation

```
Test: I1-legacy-project
Setup: Project bootstrapped by older gdev without lifecycle system.
  - devenv.nix exists but has no section markers
  - CLAUDE.md exists but has no tool section markers (only `<!-- BEGIN GENERATED SECTION -->`)
  - settings.json exists with hooks but no tool ownership metadata
Steps:
  1. gdev enable semgrep
  2. Assert: Section markers retroactively added to devenv.nix for existing content
     (OR: semgrep section added alongside unmarked legacy content)
  3. Assert: CLAUDE.md gets `<!-- gdev:semgrep -->` section within the generated section
  4. Assert: State file created/updated with file ownership
  5. gdev disable semgrep
  6. Assert: semgrep section removed
  7. Assert: Legacy unmarked content preserved
Expected: First lifecycle operation initializes tracking without disrupting existing content.

Design decision needed: Does the first lifecycle operation retroactively wrap existing tool content in markers (marker migration), or does it only mark newly-added content? Recommendation: retroactive marker migration is fragile. Better to only mark new content and treat unmarked content as "legacy core" that's never touched by the lifecycle system.
```

#### I2. Manually configured tool — enable collision

```
Test: I2-manual-config-collision
Setup: User manually created .semgrep.yml before gdev
Steps:
  1. gdev enable semgrep
  2. Assert: One of:
     a. Error: ".semgrep.yml already exists and is not gdev-managed. Use --force to overwrite"
     b. Merge: Existing config preserved, gdev additions merged
     c. Backup: Existing file backed up to .semgrep.yml.bak, new file created
  3. If --force used:
     a. Assert: Original file replaced
     b. Assert: Warning about replacement
Expected: Never silently overwrite user files. Require explicit user intent.

Recommendation: Option (a) — error with --force flag. This is the safest default and matches the "detect, don't assume" principle.
```

#### I3. Manually configured .mcp.json — enable adds entry

```
Test: I3-manual-mcp
Setup: User-created .mcp.json with `{"mcpServers": {"my-server": {...}}}`
Steps:
  1. gdev enable context7
  2. Assert: .mcp.json now has both "my-server" and "context7"
  3. Assert: "my-server" entry unchanged
  4. gdev disable context7
  5. Assert: .mcp.json has only "my-server"
Expected: Shared file merge preserves non-gdev content. This is the JSON parse/modify strategy working correctly.
```

---

## 4. Edge Cases and Failure Modes

### 4.1 File System Edge Cases

| Edge Case | Scenario | Expected Behavior |
|-----------|---------|-------------------|
| **Read-only file** | `.semgrep.yml` set to read-only by user | `gdev disable semgrep` fails with clear error: "Cannot delete .semgrep.yml: permission denied" |
| **Symlinked file** | `devenv.nix` is a symlink to a shared template | Shared file surgery must follow symlink and edit actual file. Or error: "devenv.nix is a symlink; lifecycle operations require a regular file" |
| **Missing shared file** | `gdev enable gitleaks` but devenv.nix doesn't exist (devenv addon not run) | Create devenv.nix with only the gitleaks section, or error: "devenv.nix not found — run gdev init first" |
| **Concurrent modification** | Two terminals run `gdev enable` simultaneously | File lock or last-write-wins. Recommendation: advisory lockfile `.devinit/.gdev.lock` |
| **Disk full** | Enable operation runs out of space mid-write | Atomic write strategy (write to temp, rename) prevents partial files. But shared file edits are in-place — need transaction-like behavior |
| **.cosign/ directory** | `gdev disable container-security` removes `.cosign/policy.yaml` | Remove file. Remove `.cosign/` directory only if empty (user may have added other cosign files) |
| **.claude/agents/ directory** | `gdev disable semble` removes `semble-search.md` | Remove file. Remove `.claude/agents/` only if empty |
| **.claude/skills/agent-postmortem/ directory** | `gdev disable agent-postmortem` removes `SKILL.md` | Remove file. Remove directory only if empty (user may have added companion files) |
| **.version-sentinel/ directory** | `gdev disable version-sentinel` removes `ignore` file | Remove file. Remove directory only if empty |

### 4.2 Parsing Edge Cases

| Edge Case | Scenario | Expected Behavior |
|-----------|---------|-------------------|
| **Malformed devenv.nix** | User introduced Nix syntax error | Section marker removal still works (line-based, not AST-based). Warn: "devenv.nix may contain syntax errors" |
| **Malformed JSON** | settings.json or .mcp.json has invalid JSON | `gdev enable` fails: "Cannot parse settings.json: [error]. Fix JSON syntax and retry" |
| **Missing section markers** | devenv.nix has gdev content but markers were deleted | Content is treated as user-owned. New enable adds a NEW section; old content orphaned. Warn on next `gdev status` |
| **Overlapping markers** | `<!-- gdev:semgrep -->` without closing tag | Error on enable/disable: "Malformed section marker in CLAUDE.md for semgrep: missing closing tag" |
| **Nested markers** | `<!-- gdev:semgrep -->` inside `<!-- gdev:gitleaks -->` block | Should never happen (sections are siblings, not nested). If detected, error rather than corrupt |
| **BOM/encoding** | File has UTF-8 BOM or non-UTF-8 encoding | Preserve encoding. Read/write with same encoding. Test with BOM files |
| **Windows line endings** | `.mcp.json` has CRLF | Preserve line endings. Don't convert CRLF to LF during shared file surgery |
| **Trailing newline** | Some files lack trailing newline | Preserve original trailing newline behavior. Don't add or remove final newline |

### 4.3 State Consistency Edge Cases

| Edge Case | Scenario | Expected Behavior |
|-----------|---------|-------------------|
| **State file deleted** | User deletes `.devinit/.gdev-init-state.yaml` | `gdev enable/disable` either: (a) reconstructs state from file system heuristics, or (b) errors: "State file missing — run gdev init to re-establish state" |
| **State file stale** | State says semgrep enabled but .semgrep.yml missing (user deleted it) | `gdev status` shows warning: "semgrep: enabled but .semgrep.yml missing" `gdev disable semgrep` succeeds (cleans up remaining shared-file sections) |
| **Answers file mismatch** | `.gdev-init-answers.yaml` says semgrep=true but state says disabled | Answers file is source of truth for "should be enabled"; state file tracks "what was generated". If mismatch: `gdev init --update` reconciles |
| **Orphaned sections** | State file has no record of semgrep but `# --- semgrep ---` exists in devenv.nix | `gdev status` could detect orphaned sections and warn. `gdev enable semgrep` should recognize existing section and update rather than create duplicate |
| **File changed but not by tool** | Hash mismatch on a shared file due to manual formatting (whitespace changes) | `gdev disable` should still remove tool sections. Warn about hash mismatch but proceed |

### 4.4 Nix-Specific Concerns for devenv.nix

The devenv.nix file has the most complex shared-file surgery because Nix is a functional language with significant whitespace and expression boundaries.

**Critical implementation constraint:** Section markers MUST be placed at syntactically valid insertion points:

```nix
{ pkgs, lib, ... }:

{
  packages = [
    pkgs.git
    pkgs.curl
    # --- semgrep ---
    pkgs.semgrep
    # --- end semgrep ---
    # --- gitleaks ---
    pkgs.gitleaks
    # --- end gitleaks ---
    # --- container-security ---
    pkgs.grype
    pkgs.syft
    pkgs.cosign
    # --- end container-security ---
  ];

  enterShell = ''
    echo "dev environment loaded"
    # --- secretspec ---
    # secretspec integration
    # --- end secretspec ---
  '';
}
```

**Failure modes to test:**
1. **Empty packages list after all removals**: `packages = [ pkgs.git pkgs.curl ];` (core packages remain)
2. **Nix expression validity**: After each removal, `nix eval` or `nix-instantiate --parse` should succeed
3. **Whitespace consistency**: No double-blank-lines where sections were removed
4. **Comment-only sections**: A section that is only comments (no Nix expressions) removes cleanly

### 4.5 CLAUDE.md Marker Interaction with Existing Markers

CLAUDE.md has two layers of markers:
1. **Core markers**: `<!-- BEGIN GENERATED SECTION -->` / `<!-- END GENERATED SECTION -->` (Phase 4, covers all generated content)
2. **Tool markers**: `<!-- gdev:tool -->` / `<!-- /gdev:tool -->` (Phase 12, per-tool sections within the generated section)

Tool markers must be INSIDE core markers. The hierarchy is:

```markdown
# Project Name

<!-- BEGIN GENERATED SECTION — do not edit between markers -->

## Security

<!-- gdev:semgrep -->
### Semgrep SAST
...
<!-- /gdev:semgrep -->

<!-- gdev:gitleaks -->
### Gitleaks Secret Scanning
...
<!-- /gdev:gitleaks -->

## AI Agent Tools

<!-- gdev:version-sentinel -->
### Version Sentinel
...
<!-- /gdev:version-sentinel -->

<!-- END GENERATED SECTION -->

## Custom Instructions

(user content here, never touched)
```

**Test case**: Removing all tool sections should leave core markers with only non-tool generated content (build commands, security instructions, language conventions from Phase 4).

### 4.6 Timing and Ordering Constraints

| Constraint | Description | Test |
|-----------|------------|------|
| **CI regeneration after every enable/disable** | Any tool change that affects CI must trigger workflow regeneration | Enable semgrep → verify workflow has semgrep step → enable gitleaks → verify workflow has BOTH steps (not just gitleaks) |
| **devenv.yaml input management** | First hook tool enable must add git-hooks input; last disable may remove it | Enable gitleaks (first hook tool) → verify git-hooks input exists → disable gitleaks → disable all other hook tools → verify git-hooks input removed or kept |
| **State file updated atomically** | Enable must write all files then update state, not interleave | If enable crashes after writing .semgrep.yml but before updating state: next enable should detect existing file and recover |

### 4.7 Performance Constraints

| Metric | Target | Test Method |
|--------|--------|-------------|
| `gdev enable <tool>` | <2 seconds | Wall-clock time measurement |
| `gdev disable <tool>` | <2 seconds | Wall-clock time measurement |
| `gdev status` | <1 second | Wall-clock time measurement |
| `gdev list` | <1 second | Wall-clock time measurement |
| CI workflow regeneration | <1 second | Included in enable/disable time |
| Shared file surgery (JSON) | <100ms | No full-file rewrite if section unchanged |
| Shared file surgery (section markers) | <100ms | Line-based scan + removal |

---

## 5. Test Implementation Recommendations

### 5.1 Test Fixture Strategy

Create a minimal but complete test fixture representing a Go + TypeScript + Docker project:

```
test-fixtures/multi-ecosystem/
├── go.mod
├── package.json
├── Dockerfile
├── devenv.nix        (template with no tool sections)
├── devenv.yaml       (base config)
├── .claude/
│   └── settings.json (base with deny rules, no tool hooks)
├── CLAUDE.md         (core generated section, no tool sections)
└── .mcp.json         (empty or user-only entries)
```

### 5.2 Test Categories by Priority

| Priority | Category | Count | Justification |
|----------|---------|-------|---------------|
| P0 (must pass) | A1 (individual lifecycle) × 16 tools | 16 tests | Core functionality |
| P0 | E1-E6 (shared file integrity) | 6 tests | Data corruption prevention |
| P0 | F1-F5 (idempotency) | 5 tests | Safety invariant |
| P0 | D3-D4 (user content preservation) | 2 tests | User data safety |
| P1 (should pass) | B1-B2 (bulk operations) | 2 tests | Real-world usage pattern |
| P1 | C1-C3 (error handling) | 3+ tests | User-facing error quality |
| P1 | G1-G7 (cross-tool interactions) | 7 tests | Integration correctness |
| P1 | D1-D2 (modification warnings) | 2 tests | User experience |
| P2 (nice to have) | H1-H4 (wizard integration) | 4 tests | End-to-end coverage |
| P2 | I1-I3 (migration) | 3 tests | Upgrade path coverage |
| P2 | A2-A3 (re-enable, rapid toggle) | 2 × 16 tests | Robustness |

Total: ~80+ distinct test scenarios across all categories.

### 5.3 Go Test Organization

```go
// lifecycle_test.go — per-tool enable/disable round-trip
func TestToolLifecycle_Semgrep(t *testing.T)       { testToolRoundTrip(t, "semgrep") }
func TestToolLifecycle_Gitleaks(t *testing.T)       { testToolRoundTrip(t, "gitleaks") }
// ... one per tool

// shared_file_test.go — shared file integrity after all removals
func TestDevenvNix_EmptyToolSections(t *testing.T)
func TestSettingsJSON_EmptyHooks(t *testing.T)
func TestClaudeMD_EmptyToolSections(t *testing.T)
func TestMcpJSON_AllServersRemoved(t *testing.T)
func TestPrecommit_AllHooksRemoved(t *testing.T)
func TestCIWorkflow_MinimalState(t *testing.T)

// idempotency_test.go
func TestEnableAlreadyEnabled(t *testing.T)
func TestDisableAlreadyDisabled(t *testing.T)
func TestDoubleEnable(t *testing.T)

// interaction_test.go — cross-tool
func TestPrecommitHookOrdering(t *testing.T)
func TestContainerSecurityMultiPackage(t *testing.T)
func TestChangelogCommitlintSoftDep(t *testing.T)
func TestAllMCPServers(t *testing.T)
func TestDualSettingsHooks(t *testing.T)

// conflict_test.go
func TestMissingPrerequisite(t *testing.T)
func TestConflictDeclaration(t *testing.T)

// migration_test.go
func TestLegacyProjectFirstLifecycleOp(t *testing.T)
func TestManualConfigCollision(t *testing.T)

// user_modification_test.go
func TestModifiedExclusiveFile(t *testing.T)
func TestModifiedSharedSection(t *testing.T)
func TestUserContentPreserved(t *testing.T)
func TestNonGdevMCPPreserved(t *testing.T)
```

### 5.4 Helper Functions Needed

```go
// assertFileExists(t, path) — file exists at path
// assertFileNotExists(t, path) — file does not exist
// assertValidNix(t, path) — file parses as valid Nix
// assertValidJSON(t, path) — file parses as valid JSON
// assertValidYAML(t, path) — file parses as valid YAML
// assertSectionPresent(t, path, toolName) — section markers found for tool
// assertSectionAbsent(t, path, toolName) — no section markers for tool
// assertJSONKeyPresent(t, path, key) — JSON object has key
// assertJSONKeyAbsent(t, path, key) — JSON object lacks key
// assertHookPresent(t, path, hookName) — pre-commit hook entry exists
// assertHookAbsent(t, path, hookName) — pre-commit hook entry absent
// assertToolStatus(t, toolName, enabled bool) — gdev status reports correct state
// hashFile(path) string — SHA256 for comparison
// setupTestProject(t) string — creates temp dir with test fixture, returns path
```

---

## 6. Open Design Decisions

These decisions should be resolved before implementation. Each affects multiple test scenarios.

| # | Decision | Options | Recommendation | Affects Tests |
|---|---------|---------|---------------|---------------|
| 1 | **Behavior when disabling tool with modified exclusive file** | (a) Warn and delete, (b) Prompt for confirmation, (c) Error unless --force | (b) Prompt interactively, (a) in non-interactive mode with warning | D1 |
| 2 | **Behavior when .mcp.json becomes empty** | (a) Delete file, (b) Leave `{"mcpServers": {}}` | (a) Delete if no non-gdev entries | E4 |
| 3 | **Legacy project marker migration** | (a) Retroactive marking of existing content, (b) Only mark new content | (b) Only mark new content — retroactive is fragile | I1 |
| 4 | **Behavior when exclusive file already exists (not gdev-managed)** | (a) Error + --force, (b) Merge, (c) Backup + overwrite | (a) Error + --force — safest | I2 |
| 5 | **Pre-commit file when all hooks removed** | (a) Delete file + remove devenv.yaml input, (b) Leave empty | (a) Delete if only tool hooks (no core framework hooks) remain | E5 |
| 6 | **Container-security enable without Docker detection** | (a) Allow with warning, (b) Block | (a) Allow — user intent should be respected for explicit commands | C2 |
| 7 | **CI workflow file when no tools contribute** | (a) Minimal with Harden-Runner only, (b) Delete | (a) Minimal — Harden-Runner is always-on | E6 |
| 8 | **devenv.yaml git-hooks input management** | (a) Add on first hook tool, remove on last, (b) Always present | (a) — cleaner, but more complex to track | timing constraint |
