# Learning Opportunities, Orient, and Auto — Deep Evaluation for gdev Integration

## Executive Summary

**Recommendation: Include learning-opportunities and orient as opt-in skills in gdev's claudecode addon skill library. Exclude learning-opportunities-auto (the hook variant). Reference MEASURE-THIS.md in consulting workflow documentation but do not deploy it as a tool.**

The learning-opportunities ecosystem is a well-constructed, research-backed Claude Code plugin suite that directly addresses a real consulting firm problem: engineers who produce AI-accelerated output without building genuine understanding of unfamiliar client codebases. The skill format is compatible with gdev's deployment pattern — standard `.claude/skills/` YAML frontmatter markdown files that can be copied into gdev's embedded skill library and deployed via `qsdev claude init`. The project is credible (1,530 stars, active development, PhD-level learning science research backing with peer-reviewed publications), properly licensed (CC-BY-4.0), and has zero runtime dependencies.

The orient companion is the strongest fit for consulting — it generates structured codebase orientation exercises grounded in program comprehension research, which maps directly to gdev's Join mode onboarding. Learning-opportunities-auto (the post-commit hook) should be excluded because it conflicts with gdev's existing hook deployment architecture and would add hook management complexity for marginal benefit.

---

## 1. Skill File Format Analysis

### Structure

The repo contains three plugins in a single monorepo marketplace structure:

```
DrCatHicks/learning-opportunities/
├── .claude-plugin/marketplace.json      # Marketplace catalog (3 plugins)
├── .agents/plugins/marketplace.json     # Codex marketplace catalog
├── learning-opportunities/              # Core skill plugin
│   ├── .claude-plugin/plugin.json       # Claude Code plugin manifest
│   ├── .codex-plugin/plugin.json        # Codex plugin manifest
│   ├── skills/learning-opportunities/
│   │   ├── SKILL.md                     # Main skill (YAML frontmatter + markdown)
│   │   └── resources/
│   │       └── PRINCIPLES.md            # Learning science reference (~350 lines)
│   └── docs/
│       └── MEASURE-THIS.md              # Team measurement playbook
├── learning-opportunities-auto/         # Post-commit hook plugin
│   ├── .claude-plugin/plugin.json
│   ├── .codex-plugin/plugin.json
│   ├── hooks/hooks.json                 # PostToolUse hook declaration
│   ├── hooks/post-tool-use.sh           # Bash hook script
│   └── hooks.codex.json                 # Codex hook variant
└── orient/                              # Codebase orientation plugin
    ├── .claude-plugin/plugin.json
    ├── .codex-plugin/plugin.json
    └── skills/orient/
        ├── SKILL.md                     # Orient skill (YAML frontmatter + markdown)
        └── resources/
            └── orient-bibliography.md   # Academic sources for methodology
```

### Format Compatibility with gdev

The skills use Claude Code's **current YAML frontmatter format** in `.claude/skills/<name>/SKILL.md`. This is the same format gdev already deploys for its existing skill library (Phase 4, Unit 3.5).

**learning-opportunities SKILL.md frontmatter:**
```yaml
---
name: learning-opportunities
description: Facilitates deliberate skill development during AI-assisted coding...
argument-hint: "[orient]"
license: CC-BY-4.0
---
```

**orient SKILL.md frontmatter:**
```yaml
---
name: orient
description: Generates a repo-specific orientation.md resource for the learning-opportunities skill...
argument-hint: "[showboat]"
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash, Write
---
```

Key compatibility observations:

1. **Standard skill format** — both use `.claude/skills/<name>/SKILL.md` with YAML frontmatter, identical to gdev's existing Trail of Bits skills and custom skills.
2. **No code dependencies** — pure markdown/YAML, no runtime dependencies, no build step. The only "code" is the bash hook in learning-opportunities-auto, which we're excluding.
3. **Resource files** — both skills reference `resources/` subdirectories (PRINCIPLES.md, orient-bibliography.md). gdev's `deploySkills()` function at `addons/claudecode/generate_skills.go:42-78` already copies skill directories recursively, so resource files would be deployed automatically.
4. **No `disable-model-invocation` on learning-opportunities** — Claude CAN autonomously invoke this skill. This is intentional: the skill decides when to offer exercises based on the session context. For gdev, this means the skill will proactively offer exercises after architectural work without requiring `/learning-opportunities`.
5. **`disable-model-invocation: true` on orient** — must be explicitly invoked via `/orient`. This is correct: you don't want Claude auto-generating orientation files.
6. **orient's `allowed-tools` restriction** — limits to Read, Glob, Grep, Bash, Write. No web access. This is appropriate for a tool that only reads local codebase files.

### Plugin vs Skill Deployment

The repo is structured as a **Claude Code plugin marketplace** (`.claude-plugin/marketplace.json`), which is the newer distribution mechanism. However, the actual skills themselves are standard SKILL.md files that can be extracted and deployed directly. gdev does NOT need to use the plugin marketplace system — it can embed the SKILL.md and resources/ files directly in its Go binary, same as existing skills.

**Deployment path for gdev:** Copy `learning-opportunities/skills/learning-opportunities/` and `orient/skills/orient/` into gdev's `internal/claudecode/skills/` embedded filesystem. Add entries to `templates/skills/manifest.yaml`. The `deploySkills()` function handles the rest.

---

## 2. Orient Companion Plugin — Detailed Analysis

### What It Does

Orient generates a `resources/orientation.md` file by performing a structured exploration of an unfamiliar codebase, then producing a teaching scaffold with:

1. One-line purpose statement
2. Primary languages detected
3. Pipeline/workflow stages
4. Key files (6-10 entries with descriptions)
5. Core concepts (3-5 architectural/domain concepts)
6. Common gotchas (2-3 specific traps)
7. Suggested exercise sequence (exactly 2 orientation exercises)
8. Sources consulted

### How Codebase Sampling Works

Orient follows a 6-step exploration methodology grounded in program comprehension research:

1. **README and top-level docs** — stated purpose, intended audience (Spinellis 2003)
2. **Directory tree** — `find . -maxdepth 3` for architectural overview (Spinellis 2003)
3. **Entry points** — language-specific: `main.go`, `src/index.ts`, `__main__.py`, etc. (Hermans 2021)
4. **Test files** — 2-3 integration tests as executable specifications (Storey et al. 2006)
5. **Core modules** — top 5-8 files by structural importance (class/function names, imports)
6. **Recent git history** — `git log --oneline -20` + most-edited files analysis (Spolsky practitioner writing)

The methodology is explicitly non-exhaustive — experts sample strategically rather than reading line-by-line. This aligns with consulting reality: you need to be productive in a client codebase within hours, not weeks.

### Showboat Mode

Orient has an alternative "showboat" mode that uses Simon Willison's `showboat` CLI tool (via `uvx`) to generate a linear code walkthrough document with a table of contents, commentary sections, and a Code Listings appendix. This requires `uv` to be installed. For gdev, the default mode is more appropriate since it has zero external dependencies.

### Integration with gdev Join Mode

Orient maps directly to Phase 13 (Unit 13.4) Join mode onboarding:

**Current gdev Join mode flow:**
1. `qsdev init` in existing repo → detect ecosystems → generate devenv/Claude Code config
2. Developer reads CLAUDE.md, explores codebase manually

**Enhanced flow with orient:**
1. `qsdev init` in existing repo → detect ecosystems → generate devenv/Claude Code config
2. Orient auto-generates `orientation.md` with structured codebase overview
3. Developer runs `/learning-opportunities orient` to get guided orientation exercises
4. Developer gets productive faster with structured, research-backed onboarding

**Implementation:** The `/gdev-onboard` skill (Phase 14, Unit 14.1) could invoke orient at the end of Join mode setup:

```
After qsdev init completes in Join mode, run /orient to generate orientation.md,
then suggest the user run /learning-opportunities orient for a guided tour.
```

This requires no changes to orient itself — just a mention in gdev-onboard's instructions.

### Is It a Skill, MCP Server, or Standalone Tool?

Orient is a **Claude Code skill** (`.claude/skills/orient/SKILL.md`). It is NOT an MCP server and has no standalone CLI. It runs entirely within Claude Code's context, using Claude's own file reading and analysis capabilities directed by the SKILL.md instructions. No external processes, no server, no API calls.

---

## 3. Learning-Opportunities-Auto — Hook Analysis

### How It Works

Learning-opportunities-auto is a **PostToolUse hook** that fires after every Bash tool use in Claude Code. It:

1. Reads the JSON payload from stdin
2. Checks if the command was a `git commit` (via grep on the `command` or `cmd` field)
3. Extracts `session_id` for rate limiting
4. Tracks offer count per session via temp file (`/tmp/lo_auto_<session_id>.state`)
5. Stops after 2 offers per session
6. Outputs structured JSON with `hookSpecificOutput` containing a nudge message

The nudge message tells Claude: "The user just committed code. Per the learning-opportunities skill, consider whether this is a good moment to offer a learning exercise."

### Hook Format

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash ${CLAUDE_PLUGIN_ROOT}/hooks/post-tool-use.sh"
          }
        ]
      }
    ]
  }
}
```

### Why Exclude from gdev

1. **Hook conflict risk:** gdev already deploys a PostToolUse hook for attach-guard (Phase 4, Unit 3.3). Adding another PostToolUse hook that fires on EVERY Bash tool use introduces performance overhead and potential ordering issues.

2. **Redundant with the skill itself:** The learning-opportunities skill already instructs Claude when to offer exercises (after creating new files, schema changes, architectural decisions, etc.). Since the skill does NOT have `disable-model-invocation: true`, Claude can proactively offer exercises without a hook trigger. The hook just provides an additional trigger point at git commit time.

3. **Session state management:** The hook uses temp files for session state, which is fragile. gdev's hook architecture is declarative (settings.json) rather than file-based state.

4. **Known issues:** Open issue #12 (doesn't fire for Jujutsu commits) and #9 (robustness concerns) indicate the hook implementation is still maturing.

5. **The skill is sufficient alone:** Without the auto hook, Claude still offers learning exercises after significant work. The 2-exercise-per-session cap and declined-offer suppression are built into the skill's instructions, not the hook.

**Recommendation:** Deploy learning-opportunities and orient as skills only. Do not deploy the auto hook. If engineers want post-commit prompting, they can install the plugin marketplace version directly — gdev doesn't need to manage this.

---

## 4. MEASURE-THIS.md — Measurement Framework Analysis

### What It Provides

A lightweight playbook for running pre/post team experiments with the Learning Opportunities skill, including:

**Core Survey Instruments (6 items, ~2 minutes):**
- Learning Culture (DTS-LC): 2 items, 5-point Likert — "I feel like I am learning new skills," "We often share new things we have learned"
- AI Skill Threat (PAST): 2 items, 5-point Likert — anxiety about AI changes, worry about skill obsolescence
- Coding Self-Efficacy (CSE): 1 item — confidence in problem-solving ability
- AI Behavioral Action (AI-BA): 1 item — likelihood of seeking AI skill practice

**Optional Add-Ons (for longer experiments):**
- Sense of Belonging (M-SBS): 2 items
- Developer Agency (DTS-A): 2 items
- Team Effectiveness (TER): 1 item

**Guidance:**
- Statistical rigor guardrails (don't overfit small samples, avoid spurious significance testing)
- "Team Boast" template for communicating results to leadership
- Claude.md nudges for AI-assisted analysis
- Explicit warnings about AI confabulation when analyzing survey data

### Validation

The survey instruments come from two peer-reviewed studies:
- Hicks, Lee & Foster-Marks (2025) — AI Skill Threat study, n=3,267 professional developers, 12+ industries. DOI: 10.31234/osf.io/2gej5_v2
- Hicks, Lee & Ramsey (2024) — Developer Thriving, IEEE Software. DOI: 10.1109/MS.2024.3382957

The measures are validated across adult (18+) populations, developed in English, with respondents from a large range of countries. They are free to use under CC-BY-SA 4.0.

### Integration with gdev Phase 15 (Health/Status/Compliance Reporting)

MEASURE-THIS.md is a **methodology reference**, not a deployable tool. gdev should NOT automate survey administration — that requires informed consent and anonymity guarantees that a CLI tool cannot provide.

However, the measurement framework informs Phase 15 design:
- Health scoring should include learning/growth indicators, not just velocity metrics
- The validated constructs (learning culture, coding self-efficacy, developer agency) provide a research-backed vocabulary for what "healthy" engineering culture means
- The statistical rigor guardrails (don't over-interpret small-sample results) should inform how gdev reports health metrics

**Recommendation:** Reference MEASURE-THIS.md in consulting workflow skill documentation. Include the statistical rigor Claude.md nudges in gdev's generated CLAUDE.md when learning-opportunities is enabled. Do not attempt to automate the survey workflow.

---

## 5. Quality and Maturity Assessment

### Repository Metrics (as of 2026-05-14)

| Metric | Value |
|--------|-------|
| Stars | 1,530 |
| Forks | 50 |
| Open Issues | 3 |
| License | CC-BY-4.0 (verified in LICENSE file, plugin.json, marketplace.json) |
| Created | 2026-02-07 |
| Last Push | 2026-05-02 |
| Contributors | 6 (DrCatHicks: 50, mcmullarkey: 5, eropple: 3, eugenevinitsky: 1, kmfrick: 1, mwasson: 1) |
| Community PRs | 5 merged (Codex support, auto hook fix, orient plugin, hook format fix, attribution fix) |

### Development Activity

The repo is 3 months old with active development:
- Feb 2026: Initial release, plugin marketplace infrastructure (eropple), auto hook
- Mar 2026: Orient plugin added (mcmullarkey), SKILL.md refinements, attribution fixes
- May 2026: Codex plugin support added (kmfrick)

Commit velocity has slowed since initial burst (Feb-Mar), which is normal for a skill that is primarily instructional markdown — there's less to iterate on compared to code-heavy projects. The 3 open issues are minor (Jujutsu commit support, hook robustness, appreciation post).

### Dependencies and Runtime Requirements

**Zero runtime dependencies.** The skills are pure markdown files interpreted by Claude Code at runtime. No package installation, no build step, no external services. Orient's showboat mode optionally requires `uv` + `showboat`, but the default mode has no dependencies.

This is ideal for gdev: skills are static files embedded in the Go binary and deployed via file copy. No supply chain risk, no version management, no compatibility issues.

### License Verification

**CC-BY-4.0** confirmed in:
- `LICENSE` file in repo root
- Every `plugin.json` manifest
- `marketplace.json` entries
- SKILL.md frontmatter
- README.md badge

The MEASURE-THIS.md survey instruments have a separate **CC-BY-SA 4.0** license (share-alike requirement) — this is standard for academic instruments and does not affect gdev's use, since gdev would reference but not modify the instruments.

CC-BY-4.0 permits: commercial use, modification, distribution. Requires: attribution to Dr. Cat Hicks. gdev should include attribution in the skill files and CHANGELOG.

### Companion Project: learning-goal

The README references a companion skill [learning-goal](https://github.com/DrCatHicks/learning-goal) (153 stars, CC-BY-4.0). It guides developers through Mental Contrasting with Implementation Intentions (MCII) — an evidence-based goal-setting exercise. This is a separate skill that could be evaluated independently; it's not required for learning-opportunities to function.

### Research Foundation Credibility

**Dr. Cat Hicks credentials:**
- PhD in Quantitative Experimental Psychology, UC San Diego
- Founder and Principal Scientist, Catharsis Consulting
- Creator of Developer Thriving framework and AI Skill Threat framework
- Research with 5,000+ developers and engineering managers across 12+ industries
- Published in IEEE Software (peer-reviewed)
- Upcoming book: "The Psychology of Software Teams" (Routledge, 2026)
- Active conference speaker: Craft Conference, CraftHub, RedMonk, Hanselminutes
- Bluesky: @grimalkina.bsky.social

**Dr. Michael Mullarkey (orient author):**
- ML engineer, former therapist + social science researcher
- Created the orient skill and orient-bibliography.md
- Also maintains blendtutor (AI-assisted learning skill)

The research backing is substantially more rigorous than typical open-source developer tools. The cited papers are peer-reviewed, the sample sizes are large (n=3,267 and n=1,282), and the learning science principles (generation effect, spacing, retrieval practice) are well-established in cognitive psychology.

---

## 6. Integration Design for gdev

### How `qsdev enable learning-opportunities` Would Work

```
qsdev enable learning-opportunities
  → Copies to .claude/skills/learning-opportunities/SKILL.md
  → Copies to .claude/skills/learning-opportunities/resources/PRINCIPLES.md
  → Adds "## Learning Opportunities" section to CLAUDE.md (between markers)
  → Runs qsdev claude update to refresh settings.json if needed

qsdev enable orient
  → Copies to .claude/skills/orient/SKILL.md
  → Copies to .claude/skills/orient/resources/orient-bibliography.md
  → No CLAUDE.md changes needed (orient is invoked on-demand)
```

This follows the existing `qsdev enable/disable` pattern from Phase 12 (Unit 12.1). Skills are embedded in gdev's Go binary and deployed to `.claude/skills/` on enable.

### Default-On vs Opt-In

**Recommendation: Opt-in for both, with a nudge during Join mode.**

Reasons for opt-in:
1. Senior engineers may find unsolicited learning prompts patronizing
2. The skill changes Claude's conversational behavior (pausing, waiting for input) which may surprise users
3. Not all projects warrant learning exercises (quick bug fixes, routine maintenance)
4. Consulting firm culture should determine adoption — some teams will embrace it, others won't

The nudge: during `qsdev init` in Join mode, after setup completes, print:
```
Tip: Run `qsdev enable learning-opportunities` to get science-based learning
exercises as you explore this codebase. Great for unfamiliar repos.
Run `qsdev enable orient` first to generate a structured orientation.
```

### Interaction with Existing gdev Skills

**Complementary, no conflicts:**

| Existing Skill | Interaction with learning-opportunities |
|---|---|
| Trail of Bits supply-chain-risk-auditor | No overlap — security auditing vs learning exercises |
| Trail of Bits differential-review | No overlap — code review vs learning exercises |
| Trail of Bits insecure-defaults | No overlap — security config vs learning exercises |
| gdev-doctor | Complementary — after doctor identifies issues, learning-opportunities could help understand why |
| gdev-onboard | **Direct integration point** — onboard could invoke orient at the end of Join mode |
| deploy/review-pr/security-review | No overlap — CI/CD operations vs learning exercises |
| generate-tests/refactor/db-migration | Complementary — after these complete, learning exercises on the generated code |

The only potential friction is **context budget**: adding PRINCIPLES.md (~350 lines) to the skill's resources means Claude loads ~350 extra lines when the skill is invoked. This is within acceptable limits for the Phase 14 context budget management (CLAUDE.md stays under 5% of context window). PRINCIPLES.md is only loaded on explicit skill invocation, not passively.

### Customizing Triggers

gdev could ship a customized version of the SKILL.md that adjusts triggers for consulting contexts:

**Default triggers** (from upstream):
- Creating new files or modules
- Database schema changes
- Architectural decisions or refactors
- Implementing unfamiliar patterns
- "Why" questions during development

**Consulting-specific additions:**
- After onboarding to a new client codebase (first session in Join mode)
- After making changes to client-specific business logic
- After implementing patterns that differ from the engineer's usual stack

**Consulting-specific suppressions:**
- During time-critical incident response
- During routine maintenance tasks on well-understood code
- When compliance level is set to "minimal" in client profile

gdev's CC-BY-4.0 license allows this customization with attribution.

### Should Orient Auto-Run During `qsdev init` Join Mode?

**No, but gdev should make it a one-command follow-up.**

Reasons against auto-run:
1. Orient's exploration takes 30-60 seconds and reads many files — it should be intentional
2. Not every Join mode user wants orientation exercises (some are already familiar with the repo)
3. Orient writes files to `.claude/skills/`, which should be an explicit action

Recommended UX:
```
$ qsdev init                    # Join mode detected
  [... normal init ...]
  Tip: New to this codebase? Run `qsdev orient` for structured orientation.

$ gdev orient                  # Convenience wrapper
  → Ensures orient skill is enabled
  → Runs /orient (generates orientation.md)
  → Prints: "Orientation generated. Run /learning-opportunities orient for guided exercises."
```

This keeps the flow explicit while reducing friction to one command.

---

## 7. Consulting Firm Value Assessment

### Does This Address a Real Problem?

**Yes, emphatically.** Consulting engineers face amplified versions of the five learning science risks:

1. **Generation effect** — consulting engineers frequently accept AI-generated code in unfamiliar client stacks. The speed pressure of billable hours amplifies the tendency to accept without understanding.

2. **Fluency illusion** — AI produces clean, idiomatic code in the client's framework. The engineer may feel they understand the client's patterns when they've only read AI-generated versions of them.

3. **Spacing effect** — short consulting engagements (weeks to months) mean engineers frequently context-switch between radically different codebases. There's little natural spacing for retention.

4. **Metacognition gap** — when working on unfamiliar stacks under delivery pressure, engineers may not have time to assess whether they're actually building transferable knowledge or just pattern-matching for this engagement.

5. **Testing deficit** — AI provides complete solutions, reducing the retrieval practice that builds durable understanding of client architectures.

The orient plugin addresses the specific consulting challenge of rapid codebase onboarding: instead of ad-hoc exploration, engineers get a structured, research-backed orientation methodology.

### Would Senior Engineers Resist?

**Potentially, but the design mitigates this well.**

Risk factors:
- "I don't need learning exercises" — senior engineers may view this as infantilizing
- Interruption aversion — any prompt that breaks flow risks rejection
- Identity threat — being asked to "learn" implies not already knowing

Mitigation factors built into the skill:
1. **Always asks first** — "Would you like to do a quick learning exercise?" Never forces participation.
2. **One-sentence offers** — "Keep offers brief and non-repetitive. One short sentence is enough."
3. **Session caps** — maximum 2 exercises per session, stops after first decline.
4. **Expertise reversal** — PRINCIPLES.md explicitly addresses the expertise reversal effect (Kalyuga 2007): techniques that help novices can hinder experts. The skill adjusts difficulty dynamically.
5. **Fading scaffolding** — reduces guidance as demonstrated familiarity increases: "Open file X, line N" → "Find where we handle feature Y" → "Where would you look?"

**Key insight from PRINCIPLES.md:** "Exercises should require effort without being frustrating." The skill is designed to be genuinely useful, not patronizing. Retrieval check-ins ("what do you remember from last session?") are valuable even for experts — they surface forgotten details and strengthen memory traces.

**Consulting firm strategy:** Position as "expertise maintenance" rather than "learning." The framing matters. For senior engineers, emphasize: "This helps you retain knowledge across client engagements" rather than "This helps you learn."

### 2-Exercise-Per-Session Cap

The cap is appropriate for consulting work:
- **10-15 minutes per exercise × 2 = 20-30 minutes maximum** — this is 5-6% of an 8-hour day
- Exercises are triggered by architectural work, not routine coding — most sessions won't hit the cap
- The cap prevents over-interruption during deep focus periods
- Engineers who want more can adjust the cap in their local skill customization

For a consulting firm, the cap could be made configurable via client profile compliance levels:
- `compliance: minimal` → cap at 1 exercise per session
- `compliance: standard` → cap at 2 (default)
- `compliance: full` → cap at 3, with retrieval check-in at session start

### Research Foundation Assessment

**Highly credible.** The research stands on three pillars:

1. **Established learning science** — the five risks and corresponding interventions (generation effect, spacing, retrieval practice, desirable difficulties, metacognition) are foundational principles with decades of replicated research. These are not speculative claims.

2. **Developer-specific empirical research** — Hicks' studies with 3,267 and 1,282 professional developers provide domain-specific validation. The AI Skill Threat construct is novel and specific to the AI-assisted coding transition.

3. **Peer-reviewed publication** — IEEE Software is a reputable venue. The OSF preprints follow open science norms (preregistration, open access).

The weakest link: the skill itself has not been through a controlled trial (no published RCT showing that using learning-opportunities produces better outcomes than not using it). The theoretical foundation is strong, but the specific implementation is an evidence-informed intervention, not a validated one. This is noted honestly in MEASURE-THIS.md, which provides tools for teams to run their own evaluations rather than claiming proven efficacy.

---

## 8. Implementation Recommendations Summary

### Include in gdev (Phase 4/14)

| Component | Action | Effort | Priority |
|-----------|--------|--------|----------|
| learning-opportunities SKILL.md | Embed in Go binary, deploy via `qsdev enable learning-opportunities` | Small | High |
| learning-opportunities PRINCIPLES.md | Embed as resource, deployed alongside SKILL.md | Small | High |
| orient SKILL.md | Embed in Go binary, deploy via `qsdev enable orient` | Small | High |
| orient-bibliography.md | Embed as resource, deployed alongside orient SKILL.md | Small | Medium |
| `qsdev orient` convenience command | Thin wrapper: enable orient + invoke /orient | Small | Medium |

### Exclude from gdev

| Component | Reason |
|-----------|--------|
| learning-opportunities-auto (hook) | Conflicts with gdev hook architecture; skill alone is sufficient |
| MEASURE-THIS.md (as deployed tool) | Methodology reference, not a tool; requires informed consent |
| learning-goal companion skill | Separate evaluation needed; not required for core functionality |

### CLAUDE.md Additions When Enabled

When `qsdev enable learning-opportunities` is run, add to CLAUDE.md generated section:

```markdown
## Learning & Skill Development

The learning-opportunities skill is enabled. After significant architectural work
(new files, schema changes, refactors, unfamiliar patterns), offer a brief learning
exercise. Keep offers to one sentence. Respect session limits (2 exercises max,
stop after decline). See PRINCIPLES.md for the learning science behind exercise design.

When analyzing team metrics or survey data, apply statistical rigor:
- Present findings plainly without hyperbole
- Flag potential confounds and alternative explanations
- Distinguish statistical significance from practical significance
- Always report spread alongside central tendency
```

### Attribution Requirements

CC-BY-4.0 requires attribution. Include in gdev's skill deployment:

```markdown
<!-- Learning Opportunities skill by Dr. Cat Hicks (CC-BY-4.0) -->
<!-- https://github.com/DrCatHicks/learning-opportunities -->
<!-- Orient skill by Dr. Michael Mullarkey (CC-BY-4.0) -->
```

---

## Sources

All source files saved to `docs/`:
- `docs/learning-opportunities-drcathicks.md` — Initial README fetch
- `docs/learning-opportunities-skill-md.md` — SKILL.md analysis
- `docs/orient-skill-md.md` — Orient SKILL.md analysis
- `docs/learning-opportunities-auto-hook.md` — Auto hook implementation analysis
- `docs/measure-this-playbook.md` — MEASURE-THIS.md analysis

Primary sources (GitHub API, retrieved 2026-05-14):
- https://github.com/DrCatHicks/learning-opportunities — Full repo tree and file contents
- https://drcathicks.com — Author credentials
- https://www.routledge.com/The-Psychology-of-Software-Teams/Hicks/p/book/9781032963389 — Book
- https://doi.org/10.31234/osf.io/2gej5_v2 — AI Skill Threat preprint
- https://doi.org/10.1109/MS.2024.3382957 — Developer Thriving, IEEE Software
