# Per-Session Claude Code Context Overlays & Project Clarity Templates

## Research Question

How can gdev implement (A) automatic per-project Claude Code context shifting via devenv enterShell, and (B) project clarity documentation as CLAUDE.md template content?

---

## Part A: Per-Session Context Overlays via devenv enterShell

### 1. Claude Code Environment Variables — What Exists

Claude Code supports ~70 environment variables, documented at [code.claude.com/docs/en/env-vars](https://code.claude.com/docs/en/env-vars). The variables most relevant to per-project context configuration are:

| Variable | What It Does | gdev Relevance |
|----------|-------------|----------------|
| `CLAUDE_CONFIG_DIR` | Redirects `~/.claude/` to a custom directory | **HIGH** — the Yaw Mode mechanism; partially implemented, undocumented |
| `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD` | Enables loading CLAUDE.md from `--add-dir` paths | **HIGH** — could load shared gdev context from a central directory |
| `CLAUDE_CODE_EFFORT_LEVEL` | Sets model reasoning effort (low/medium/high/xhigh/max/auto) | **MEDIUM** — per-project effort tuning |
| `CLAUDE_CODE_DISABLE_POLICY_SKILLS` | Skip managed skills | **LOW** — useful for isolation |
| `CLAUDE_CODE_PLUGIN_SEED_DIR` | Pre-populated plugin directories (`:` separated) | **MEDIUM** — could seed gdev plugins per-project |
| `CLAUDE_CODE_DISABLE_AUTO_MEMORY` | Disable auto memory | **LOW** — niche |
| `CLAUDE_CODE_DISABLE_GIT_INSTRUCTIONS` | Remove git workflow instructions | **LOW** — niche |

**Critical finding: There is NO `CLAUDE_SKILLS_PATH`, `CLAUDE_RULES_PATH`, or similar variable.** Claude Code does not support environment-based paths for skills or rules directories. Skills are loaded from fixed locations: `~/.claude/` (user scope) and `.claude/` (project scope). Rules are loaded from `.claude/rules/` in the project. There is no way to add additional skills or rules directories via environment variables.

**Critical finding: `CLAUDE_CONFIG_DIR` exists but is undocumented and buggy.** Multiple GitHub issues document problems:
- Issue #3833: Still creates local `.claude/` directories even when `CLAUDE_CONFIG_DIR` is set
- Issue #4739: `/ide` command fails when `CLAUDE_CONFIG_DIR` is set
- Issue #30538: VS Code extension ignores `CLAUDE_CONFIG_DIR`
- Issue #25762: Feature request to make it official (still open)
- Issue #28808: Another feature request for the same capability

### 2. How Yaw Mode Actually Works

Yaw Terminal's "Yaw Mode" is the most sophisticated existing implementation of per-session Claude Code context overlays. Its mechanism, documented at [yaw.sh/blog/claude-code-yaw-mode](https://yaw.sh/blog/claude-code-yaw-mode):

1. **Creates a temporary directory** under the platform's tmp root, namespaced by process ID + PTY ID (collision-proof)
2. **Sets `CLAUDE_CONFIG_DIR=<overlay>`** to redirect Claude Code's configuration source
3. **Hardlinks** state files (`settings.json`, `.credentials.json`, `history.jsonl`, `.claude.json`) from `~/.claude/` into the overlay — writes flow through to the real home directory
4. **Symlinks** persistent directories (`projects/`, `sessions/`, `plans/`, `file-history/`) into the overlay
5. **Merges** CLAUDE.md by appending overlay content under `## Yaw Mode - added instructions`
6. **Ships** 7 skills and 3 sub-agents via symlinks in the overlay, loaded on-demand
7. Sets `$YAW_MODE` environment variable to "augment" or "fresh" for runtime detection

**Two modes:**
- **Augment (default)**: Layers the Yaw bundle atop existing `~/.claude/` config. User's own skills, agents, and settings participate alongside Yaw's additions.
- **Fresh**: Ignores user `~/.claude/` for skills/agents/CLAUDE.md, using only the Yaw bundle. Conversation history still routes through the real home directory.

**Key insight**: Yaw Mode works because it controls the terminal that launches Claude Code. It can set environment variables and create filesystem structures before Claude Code starts. This is exactly what devenv's `enterShell` hook can do.

### 3. devenv enterShell Integration with Claude Code

**How environment inheritance works with Claude Code:**

Claude Code inherits the parent shell's environment at startup. This is confirmed by the direnv integration issue ([github.com/anthropics/claude-code/issues/42229](https://github.com/anthropics/claude-code/issues/42229)). When you run `claude` from a terminal, it gets whatever environment variables are set in that shell. The direnv hook-based solution (SessionStart/CwdChanged hooks) is only needed for directory changes DURING a session — the initial environment is inherited naturally.

**This means devenv `enterShell` is the ideal hook point.** When an engineer enters the devenv shell (via `direnv` auto-activation or `devenv shell`), any environment variables set in `devenv.nix` are available to all commands launched from that shell — including Claude Code.

**What devenv can set per-project:**
```nix
# In devenv.nix
env.CLAUDE_CODE_EFFORT_LEVEL = "high";        # Per-project reasoning effort
env.CLAUDE_CODE_PLUGIN_SEED_DIR = "/path/to/gdev-plugins";  # gdev plugin cache
env.GDEV_PROJECT_PROFILE = "security-full";    # gdev-specific context
env.GDEV_CLIENT_NAME = "acme-corp";            # consulting context
```

**What devenv CANNOT do (today):**
- Redirect skills/rules paths (no env var exists)
- Dynamically compose `.claude/` directory content (that's filesystem work, not env vars)
- Override where Claude Code looks for project-level config (always `.claude/` in project root)

### 4. Practical Design: Three Viable Strategies

Given the constraints discovered above, here are three strategies for per-project context overlays, ordered by reliability:

#### Strategy 1: Project-Level `.claude/` Generation (RECOMMENDED — Already Planned)

**Mechanism:** gdev generates project-level `.claude/settings.json`, `.claude/CLAUDE.md`, `.claude/rules/*.md`, `.claude/skills/*/SKILL.md`, and `.claude/agents/*.md` during `gdev init`. Claude Code discovers these automatically via its standard project-scope mechanism.

**What this achieves:**
- Per-project permission presets (deny rules, allow rules)
- Per-project skills and agents
- Per-project rules (with `paths:` frontmatter for lazy loading)
- Per-project CLAUDE.md content
- Per-project MCP server configuration (`.mcp.json`)

**What it does NOT achieve:**
- Session-scoped overlays (files persist across sessions)
- Dynamic context that changes based on who's running the session
- Layered composition from multiple sources beyond the two-tier hierarchy (user + project)

**Status:** This is exactly what Phase 4 and Phase 14 of the gdev implementation plan already design. No new mechanism needed.

#### Strategy 2: enterShell Environment Variables for Supported Behaviors

**Mechanism:** devenv's `enterShell` or `env.*` sets Claude Code environment variables that affect per-session behavior.

```nix
# In generated devenv.nix
env.CLAUDE_CODE_EFFORT_LEVEL = "high";
env.CLAUDE_CODE_BASH_MAINTAIN_PROJECT_WORKING_DIR = "1";

enterShell = ''
  # Set gdev-specific context that CLAUDE.md can reference
  export GDEV_PROJECT_PROFILE="${config.gdev.profile}"
  export GDEV_CLIENT="${config.gdev.client}"
  export GDEV_COMPLIANCE_LEVEL="${config.gdev.compliance}"
'';
```

**What this achieves:**
- Per-project model effort tuning
- Per-project gdev metadata visible to Claude Code (via `env` in system prompt or `printenv` in Bash tool)
- Working directory behavior control

**Limitation:** Most Claude Code behaviors are NOT controlled by environment variables. The env vars that exist are primarily for API routing, debugging, and a handful of behavioral flags. The core behaviors (what skills are loaded, what rules apply, what CLAUDE.md says) are filesystem-driven.

**Enhancement opportunity:** gdev's generated CLAUDE.md could reference these environment variables with dynamic context injection:
```markdown
## Project Context
!`echo "Profile: ${GDEV_PROJECT_PROFILE:-not set}, Client: ${GDEV_CLIENT:-not set}"`
```

This would let CLAUDE.md display project-specific context that comes from the devenv environment. However, this only works in skills (`` !`command` `` syntax), not in standard CLAUDE.md content.

#### Strategy 3: Yaw-Style Overlay via enterShell (ADVANCED — Not Recommended for v1)

**Mechanism:** Replicate Yaw Mode's temporary directory overlay pattern in devenv's `enterShell`.

```nix
enterShell = ''
  # Create session-scoped Claude Code config overlay
  CLAUDE_OVERLAY=$(mktemp -d "/tmp/gdev-claude-XXXXXXXX")
  
  # Hardlink state files from ~/.claude/
  ln ${HOME}/.claude/settings.json "$CLAUDE_OVERLAY/settings.json" 2>/dev/null || true
  ln ${HOME}/.claude/.credentials.json "$CLAUDE_OVERLAY/.credentials.json" 2>/dev/null || true
  ln ${HOME}/.claude/.claude.json "$CLAUDE_OVERLAY/.claude.json" 2>/dev/null || true
  
  # Symlink persistent directories
  ln -sf ${HOME}/.claude/projects "$CLAUDE_OVERLAY/projects"
  ln -sf ${HOME}/.claude/sessions "$CLAUDE_OVERLAY/sessions"
  
  # Copy and merge CLAUDE.md with project-specific overlay
  if [ -f ${HOME}/.claude/CLAUDE.md ]; then
    cp ${HOME}/.claude/CLAUDE.md "$CLAUDE_OVERLAY/CLAUDE.md"
    cat .claude/gdev-overlay.md >> "$CLAUDE_OVERLAY/CLAUDE.md"
  fi
  
  # Symlink skills from gdev shared library
  mkdir -p "$CLAUDE_OVERLAY/skills"
  for skill in ${gdevSkillsPath}/*; do
    ln -sf "$skill" "$CLAUDE_OVERLAY/skills/$(basename $skill)"
  done
  
  export CLAUDE_CONFIG_DIR="$CLAUDE_OVERLAY"
  
  # Cleanup on shell exit
  trap "rm -rf $CLAUDE_OVERLAY" EXIT
'';
```

**What this achieves:**
- Full Yaw Mode-equivalent overlay: user config preserved, gdev skills layered on top
- Session-scoped: overlay disappears when the shell exits
- Can layer shared organization skills from a Nix store path alongside project skills

**Why NOT recommended for v1:**
1. `CLAUDE_CONFIG_DIR` is undocumented and buggy — relying on it is fragile
2. Hardlink/symlink dance is complex and platform-specific (no Windows junctions in Nix)
3. Race conditions if multiple shells are open (Yaw Mode uses PID+PTY namespacing)
4. The benefit over Strategy 1 (generated `.claude/`) is marginal — most teams want consistent project config, not session-scoped ephemeral overlays
5. Debugging is harder when config comes from a temp directory

**When to reconsider:** If Anthropic officially documents and stabilizes `CLAUDE_CONFIG_DIR`, this becomes viable. Track issues #25762 and #28808.

### 5. Recommended Approach for gdev

**Phase 4 (existing):** Generate static `.claude/` directory contents during `gdev init`. This covers 90% of the per-project context use case.

**Phase 14 enhancement (new):** Add devenv environment variables that carry gdev metadata into Claude Code sessions:

```nix
# Generated by gdev in devenv.nix
env.GDEV_PROJECT_PROFILE = "security-full";
env.GDEV_CLIENT = "acme-corp";
env.GDEV_COMPLIANCE_LEVEL = "soc2";
env.CLAUDE_CODE_EFFORT_LEVEL = "high";
```

These variables serve two purposes:
1. Claude Code env vars (`CLAUDE_CODE_EFFORT_LEVEL`) directly affect behavior
2. gdev-specific variables (`GDEV_*`) are readable by Claude Code via `printenv` or Bash tool, and can be referenced in CLAUDE.md instructions: "Check GDEV_COMPLIANCE_LEVEL to determine which security rules apply"

**Future consideration:** Monitor `CLAUDE_CONFIG_DIR` stabilization. If it becomes official, implement Strategy 3 as an opt-in `gdev claude overlay` command or `enterShell` enhancement.

### 6. `--add-dir` as Alternative Overlay Mechanism

A promising alternative emerged from the research: Claude Code's `--add-dir` flag with `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1`.

**How it works:**
- `claude --add-dir /path/to/shared-context` loads CLAUDE.md, `.claude/CLAUDE.md`, `.claude/rules/*.md`, and `CLAUDE.local.md` from the additional directory
- Introduced in Claude Code 2.1.20
- An open feature request (#3146) proposes `additionalDirectories` in settings.json

**gdev integration opportunity:**

```nix
# In generated devenv.nix
env.CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD = "1";

# In generated .claude/settings.json (when feature lands)
# "additionalDirectories": [
#   { "path": "~/.gdev/shared-context", "readClaudeMd": true }
# ]
```

Combined with a shell alias or wrapper:
```nix
enterShell = ''
  alias claude='claude --add-dir ~/.gdev/shared-context'
'';
```

This would let gdev maintain a shared organization-level context directory (`~/.gdev/shared-context/`) with CLAUDE.md, rules, and skills that are loaded alongside every project's own `.claude/` config. This is cleaner than the `CLAUDE_CONFIG_DIR` overlay approach and uses a supported (if semi-documented) mechanism.

**Status:** `--add-dir` works today. `additionalDirectories` in settings.json is a feature request. The shell alias approach is a viable workaround until the settings-based approach ships.

---

## Part B: Project Clarity Template for CLAUDE.md

### 1. Existing CLAUDE.md Best Practices (2026)

Research across multiple sources identifies converging patterns for well-structured CLAUDE.md files:

**ObviousWorks 2026 Architecture** ([obviousworks.ch](https://www.obviousworks.ch/en/designing-claude-md-right-the-2026-architecture-that-finally-makes-claude-code-work/)):
- **WHAT/WHY/HOW** structure: Project context → Principles → Workflows
- Specific technology versions, not generic names
- Repository layout mapping
- Under 200 lines; use `@imports` for modularity

**Community Best Practices** (aggregated from UX Planet, ClaudeCodeLab, ClaudeLog):
- Keep CLAUDE.md under 200 lines (context window efficiency)
- Use bullet points, not prose (consumed as tokens every session)
- Include build/test/lint commands (Claude's most frequent need)
- "Living Document" principle: every time Claude makes a mistake, add a rule
- CLAUDE.md is advisory (~70% followed); hooks are deterministic for enforcement
- Performance degrades beyond 60% context utilization; auto-compaction at ~83.5% is lossy

**Claude Code Official** ([code.claude.com/docs/en/best-practices](https://code.claude.com/docs/en/best-practices)):
- CLAUDE.md is loaded at the start of every conversation
- Supports `@imports` for modularity: `@docs/architecture.md`
- CLAUDE.local.md for personal shortcuts (gitignored)
- Scoped CLAUDE.md in subdirectories (`src/CLAUDE.md`)
- Multiple scope cascade: global → project → local → folder

### 2. What gdev Phase 4 Already Generates

From the implementation plan (Phase 4, Unit 3.2), gdev currently generates:

```markdown
<!-- BEGIN GENERATED SECTION — do not edit between markers -->
## System Environment
[detected languages, frameworks, package managers]

## Build & Test
[build/test/lint commands from wizard answers]

## Security Policy
[package installation rules, secret handling, testing requirements]

## Language Conventions
[conditional sections for Go, TypeScript, Python, Rust]

## Architecture Notes
[project description from wizard]
<!-- END GENERATED SECTION -->

## Custom Instructions
[empty section for user content]
```

Phase 14 (Unit 14.7) extends this with section markers for skills, agents, commands, and tasks.

**What's missing:** A "Project Context" section that prompts engineers to articulate purpose, stakeholders, success criteria, and exclusions — the clarity questions from vtemian's framework.

### 3. Designed Project Clarity Section

The following section should be added to gdev's CLAUDE.md template, placed AFTER the generated section (in the user-editable area) as a prompted template that engineers fill in:

```markdown
## Project Context

<!-- Fill in these fields to give Claude operational context about this project.
     Even partial answers significantly improve Claude's output quality.
     Delete or leave blank any fields that don't apply. -->

### Purpose
<!-- What problem does this project solve? One sentence. -->

### Stakeholders
<!-- Whose problem is it? Who uses the output? -->

### Success Criteria
<!-- What does "done" look like? What does "good" look like? -->

### Exclusions
<!-- What is explicitly out of scope? What should Claude NOT do? -->

### Engagement Context
<!-- For consulting projects only — delete if internal -->
- **Client**: <!-- client name or codename -->
- **Engagement type**: <!-- greenfield | brownfield | migration | assessment | staff-aug -->
- **Compliance**: <!-- none | soc2 | hipaa | pci-dss | fedramp | other -->
- **Handoff date**: <!-- when does the engagement end? -->

### Technical Constraints
<!-- Hard constraints Claude must respect -->
<!-- Examples: "Must run on Node 18 LTS", "No external network calls in tests", 
     "All changes must be backward compatible with API v2" -->
```

**Design rationale:**

1. **Placed outside generated markers**: This is user-owned content. `gdev init --update` preserves it. Engineers edit it directly.

2. **HTML comments as prompts**: The `<!-- -->` syntax is invisible to Claude Code (it strips HTML comments from CLAUDE.md before injection). These prompts guide the human filling in the template but don't consume context tokens once filled in. **IMPORTANT CORRECTION**: Actually, Claude Code does see HTML comments in CLAUDE.md — they are not stripped. However, the comment-prompt pattern is still useful because once the engineer replaces the comment with actual content, the prompt text is gone. If left unfilled, the comments serve as documentation without confusing Claude (Claude understands HTML comments are author-facing notes).

3. **"Even partial answers" messaging**: Reduces friction. An engineer who fills in only "Purpose" and "Exclusions" still gives Claude more useful context than the default.

4. **Consulting-specific fields are optional**: The `<!-- For consulting projects only -->` note and the instruction to delete if internal prevent irrelevant fields from cluttering non-consulting projects.

5. **Engagement type as enum**: The five engagement types (greenfield, brownfield, migration, assessment, staff-aug) map to different Claude behaviors. A brownfield engagement means Claude should prioritize consistency with existing patterns. A greenfield engagement means Claude has more freedom to choose patterns. gdev could eventually use this field to select different skill sets.

6. **Compliance level drives security posture**: If compliance is `soc2` or `hipaa`, Claude should be more conservative about data handling, logging, and access patterns. This ties into the permission preset system (Phase 4 Unit 3.1).

### 4. Integration with Copier Templates

Copier's questionnaire mechanism (`copier.yaml`) naturally maps to project clarity questions. Here's how the integration works:

**In the Copier template's `copier.yaml`:**
```yaml
project_name:
  type: str
  help: "Project name (kebab-case)"

project_purpose:
  type: str
  help: "What problem does this project solve? (one sentence)"
  default: ""

stakeholders:
  type: str
  help: "Who uses this project's output?"
  default: ""

engagement_type:
  type: str
  help: "Engagement type"
  choices:
    greenfield: "Greenfield — new project from scratch"
    brownfield: "Brownfield — extending existing codebase"
    migration: "Migration — moving between platforms/frameworks"
    assessment: "Assessment — evaluating and recommending"
    staff_aug: "Staff augmentation — embedded in client team"
    internal: "Internal — not a client engagement"
  default: "internal"

compliance_level:
  type: str
  help: "Compliance requirements"
  choices:
    none: "None"
    soc2: "SOC 2"
    hipaa: "HIPAA"
    pci_dss: "PCI-DSS"
    fedramp: "FedRAMP"
  default: "none"

client_name:
  type: str
  help: "Client name or codename (leave blank for internal)"
  default: ""
  when: "{{ engagement_type != 'internal' }}"

handoff_date:
  type: str
  help: "Engagement end date (YYYY-MM-DD, leave blank if ongoing)"
  default: ""
  when: "{{ engagement_type != 'internal' }}"

success_criteria:
  type: str
  help: "What does 'done' look like? What does 'good' look like?"
  default: ""

exclusions:
  type: str
  help: "What is explicitly out of scope?"
  default: ""

technical_constraints:
  type: str
  help: "Hard technical constraints (e.g., 'Must support Node 18 LTS')"
  default: ""
```

**Template rendering flow:**

The Copier answers flow into TWO files:
1. **CLAUDE.md** — The `## Project Context` section is rendered with the answers
2. **README.md** — A `## About` section is rendered with project_purpose and stakeholders

```jinja
{# In CLAUDE.md.jinja #}
## Project Context

{% if project_purpose %}
### Purpose
{{ project_purpose }}
{% endif %}

{% if stakeholders %}
### Stakeholders
{{ stakeholders }}
{% endif %}

{% if success_criteria %}
### Success Criteria
{{ success_criteria }}
{% endif %}

{% if exclusions %}
### Exclusions
{{ exclusions }}
{% endif %}

{% if engagement_type != 'internal' %}
### Engagement Context
- **Client**: {{ client_name }}
- **Engagement type**: {{ engagement_type }}
- **Compliance**: {{ compliance_level }}
{% if handoff_date %}- **Handoff date**: {{ handoff_date }}{% endif %}
{% endif %}

{% if technical_constraints %}
### Technical Constraints
{{ technical_constraints }}
{% endif %}
```

**Key design decision:** Answers from `copier.yaml` questionnaire are stored in `.copier-answers.yml`. When the template is updated via `copier update`, the answers persist. This means the project clarity section survives template updates without re-prompting.

**gdev init integration (non-Copier path):**

For projects initialized with `gdev init` without `--from <template>` (no Copier template), the clarity section is generated as a template with HTML comment prompts (the design in section 3 above). The wizard could optionally ask 2-3 key questions:

```
gdev init wizard:
  [existing questions: language, framework, security preset]
  
  Optional project context (press Enter to skip):
  - Project purpose: ___
  - Client name (if consulting): ___
  - Compliance level [none/soc2/hipaa/pci-dss/fedramp]: ___
```

These answers would populate the Project Context section instead of leaving comment prompts.

### 5. The Tarpit Test as Design Principle

vtemian's "tarpit test" — "If a tool sells itself as a replacement for thinking clearly, it's a tarpit" — should be documented in two places:

**A. In gdev's plan.md Design Principles:**

Add as a numbered principle alongside the existing four:

> **5. Amplify, don't replace.** Every gdev feature must amplify existing engineering clarity, not substitute for it. Apply the tarpit test: if a feature works without the engineer thinking clearly about what they're building, it's a tarpit. gdev generates guardrails and context — it does not generate understanding.

This principle is already implicit in gdev's design (rejected feature #3: "AI prompt templates" was rejected because "templates become crutches — better to understand prompting"). Making it explicit as a named principle strengthens the evaluation framework for future feature proposals.

**B. In gdev's generated CLAUDE.md (as a rule):**

```markdown
## Working Principles
- **Tarpit test**: If a tool or approach substitutes for thinking clearly about the problem, avoid it.
  Prefer approaches that require understanding the domain, even if they take longer.
```

This should go in the generated section (between markers) because it's an operational directive for Claude, not a project-specific customization. It guides Claude's behavior when suggesting solutions — e.g., preferring explicit error handling over catch-all wrappers, preferring typed APIs over string manipulation, preferring understanding a library's model over cargo-culting examples.

**C. In `.claude/rules/security-rules.md` (generated by gdev):**

```markdown
## Decision Framework
Before suggesting any new dependency, tool, or architectural approach, evaluate:
1. Does this require the developer to understand the problem domain? (good)
2. Does this abstract away the need to understand? (suspicious — apply tarpit test)
3. Does this have clear failure modes the developer must handle? (good)
4. Does this hide failure modes behind "it just works" promises? (tarpit)
```

---

## Summary of Findings

### Part A: Context Overlays

1. **Claude Code does NOT support environment-based skills/rules path configuration.** The original hypothesis (`CLAUDE_SKILLS_PATH`, `CLAUDE_RULES_PATH`) is invalid. Claude Code uses fixed filesystem locations.

2. **`CLAUDE_CONFIG_DIR` exists but is undocumented and buggy.** Yaw Mode uses it successfully because Yaw controls the terminal and can work around the bugs. gdev should not depend on it until Anthropic stabilizes it.

3. **The recommended approach is gdev's existing plan (Phase 4 `.claude/` generation) plus devenv environment variables for supported behaviors.** This is reliable, uses documented mechanisms, and covers 90% of the use case.

4. **`--add-dir` with `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1` is a promising future mechanism** for loading shared organization context alongside project context. Worth monitoring and integrating when `additionalDirectories` lands in settings.json.

5. **devenv `enterShell` works for environment variable inheritance.** Claude Code inherits the parent shell's environment. Set `CLAUDE_CODE_EFFORT_LEVEL` and `GDEV_*` variables in devenv.nix for per-project behavioral tuning.

### Part B: Clarity Templates

1. **The WHAT/WHY/HOW structure is the community consensus for CLAUDE.md.** gdev's existing template covers WHAT (project context) and HOW (build/test/lint), but is weak on WHY (purpose, constraints, exclusions).

2. **A "Project Context" section with clarity questions should be added** to the user-editable area of generated CLAUDE.md, with consulting-specific optional fields (client, engagement type, compliance level, handoff date).

3. **Copier integration enables pre-filling clarity answers** during `gdev init --from <template>`. Answers flow into both CLAUDE.md and README.md, survive template updates via `.copier-answers.yml`.

4. **The tarpit test should be documented as a design principle** in plan.md, as a working principle in generated CLAUDE.md, and as a decision framework in generated security rules.

---

## Implementation Recommendations

### For Phase 4 (CLAUDE.md Generation)

Add to Unit 3.2 steps:
1. Generate `## Project Context` section below the end marker, with comment-prompt templates
2. Add `## Working Principles` to the generated section (between markers) with the tarpit test
3. If wizard collects project purpose/client/compliance, populate the Project Context section

### For Phase 6 (Copier Integration)

Add to Unit 5.8 or a new unit:
1. Include clarity questions in Copier questionnaire template (`copier.yaml`)
2. Render answers into both CLAUDE.md and README.md

### For Phase 14 (enterShell Enhancement)

Add to devenv.nix generation:
1. Set `CLAUDE_CODE_EFFORT_LEVEL` based on detected project complexity
2. Set `GDEV_*` variables for project metadata (profile, client, compliance)
3. Optionally set `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1` and provide a wrapper alias for `--add-dir` to load shared organization context

### Future Work (Track, Don't Implement)

- Monitor `CLAUDE_CONFIG_DIR` stabilization (issues #25762, #28808)
- Monitor `additionalDirectories` in settings.json (issue #3146)
- If either stabilizes, implement Strategy 3 (Yaw-style overlay) or `--add-dir` settings integration

---

## Sources

- [Claude Code Environment Variables (official)](https://code.claude.com/docs/en/env-vars) → `docs/claude-code-environment-variables-official.md`
- [Claude Code Settings (official)](https://code.claude.com/docs/en/settings) → `docs/claude-code-settings-configuration-official.md`
- [Yaw Mode Technical Implementation](https://yaw.sh/blog/claude-code-yaw-mode) → `docs/yaw-mode-technical-implementation.md`
- [Claude Code direnv Integration via Hooks](https://github.com/anthropics/claude-code/issues/42229) → `docs/claude-code-direnv-integration-hooks.md`
- [CLAUDE_CONFIG_DIR Feature Request](https://github.com/anthropics/claude-code/issues/25762) → `docs/claude-config-dir-feature-request.md`
- [--add-dir CLAUDE.md Loading](https://github.com/anthropics/claude-code/issues/21138) → `docs/claude-code-add-dir-claude-md-loading.md`
- [Additional Directories via Settings](https://github.com/anthropics/claude-code/issues/3146) → `docs/claude-code-additional-directories-settings-request.md`
- [CLAUDE.md 2026 Architecture (ObviousWorks)](https://www.obviousworks.ch/en/designing-claude-md-right-the-2026-architecture-that-finally-makes-claude-code-work/) → `docs/claude-md-2026-architecture-obviousworks.md`
- [Yaw Labs Terminal Context](https://www.siliconsnark.com/yaw-labs-built-a-terminal-startup-for-people-who-treat-context-like-ammunition/) → `docs/yaw-labs-terminal-context-ammunition.md`
- [Fight Slop with Clarity (vtemian)](https://blog.vtemian.com/post/fight-slop-with-clarity/) → `docs/fight-slop-with-clarity-vtemian.md`
