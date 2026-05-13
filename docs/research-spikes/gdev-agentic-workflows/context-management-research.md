# Context Management for gdev-Generated Claude Code Configuration

## Executive Summary

Context management is the critical constraint for gdev's generated configuration. Claude Code's context window fills fast and performance degrades as it fills -- community data suggests the "dumb zone" begins at ~40% utilization. Everything gdev generates (CLAUDE.md, rules, skill descriptions, agent definitions) consumes context tokens every session. The optimal strategy is a layered architecture: minimal always-on context (CLAUDE.md + rules), on-demand skill/agent loading, and aggressive use of `.claude/rules/` with `paths:` frontmatter for conditional loading.

## 1. Context Budget Analysis

### What Loads When

| Component | When | Token Cost | gdev Control |
|---|---|---|---|
| CLAUDE.md | Every session start | Full content, every request | Direct: gdev generates this |
| `.claude/rules/*.md` | Session start (or on file access if `paths:` set) | Full content when loaded | Direct: gdev generates these |
| Skill descriptions | Session start | Combined text of all skill descriptions (1% of context budget) | Direct: gdev writes descriptions |
| Skill full content | On invocation | Full SKILL.md content, stays until compaction | Direct: gdev writes skills |
| Agent definitions | Not loaded to context (metadata only) | Description only for auto-delegation | Direct: gdev writes agents |
| MCP tool names | Session start | Tool names, schemas deferred | Indirect: gdev configures MCP |
| Hooks | Never (external) | Zero unless hook returns output | Direct: gdev configures hooks |
| settings.json | Never (parsed by harness) | Zero | Direct: gdev generates this |

### Token Budget Targets

Based on community findings and Anthropic's guidance:

| Model | Context Window | Target Usage | Available for Conversation |
|---|---|---|---|
| Sonnet 4.6 (200k) | 200k tokens | 30-40% | 120-140k tokens |
| Opus 4.6 (1M) | 1M tokens | 30-40% | 600-700k tokens |

**gdev's config budget**: The generated configuration should consume less than 5% of the context window to leave headroom for conversation, file reads, and tool outputs.

For Sonnet: 5% of 200k = **10,000 tokens** (~40KB of text)
For Opus: 5% of 1M = **50,000 tokens** (~200KB of text)

### What This Means for gdev

gdev must be disciplined about what it puts in always-on config:

| File | Target Size | Rationale |
|---|---|---|
| CLAUDE.md | 50-100 lines (~2-4KB) | Loaded every request, must be concise |
| Each `.claude/rules/*.md` | 30-50 lines (~1-2KB) | Loaded per glob match |
| Each skill description | 1-2 sentences (~200 chars) | All descriptions loaded at start |
| Each SKILL.md body | 100-500 lines | Only loaded on invocation |
| Each agent body | Any length | Isolated in own context window |

## 2. Layered Architecture for gdev

### Layer 1: Always-On (CLAUDE.md)
**Budget**: 50-100 lines, ~2-4KB

Only include:
- Build/test/lint commands (Claude can't guess these)
- Architecture overview (one paragraph)
- Non-default coding conventions
- @-imports to rules files
- Workflow instructions ("use /review-pr before merging")

Never include:
- Things Claude can infer from code (standard language patterns)
- Long reference docs (move to skills)
- Detailed procedures (move to skills)
- Technology tutorials

**gdev template strategy**: Generate a slim CLAUDE.md with section markers. Use `@` imports to pull in rules conditionally:

```markdown
# Project Instructions

## Build & Test
- Build: `go build ./...`
- Test: `go test ./...`
- Lint: `golangci-lint run`

## Architecture
[One paragraph from wizard]

## Conventions
See @.claude/rules/go-conventions.md for Go patterns.
See @.claude/rules/testing-conventions.md for test patterns.

## Workflows
- Before merging: `/review-pr`
- For new tests: `/add-tests`
- For security review: use @security-reviewer agent
```

### Layer 2: Conditional Rules (`.claude/rules/*.md`)
**Budget**: 30-50 lines each, loaded only when working with matching files

Use `paths:` frontmatter to scope rules to relevant file types:

```yaml
---
paths: "**/*.go"
---
# Go Conventions
- Use table-driven tests
- Handle all errors explicitly (no _ = err)
- Use context.Context as first parameter
```

```yaml
---
paths: "**/*.{ts,tsx}"
---
# TypeScript Conventions
- Use strict mode
- Prefer interfaces over type aliases for object shapes
- Use discriminated unions for state machines
```

**gdev generates** rules based on detected languages/frameworks. Rules with `paths:` load lazily -- zero cost when Claude isn't working with those file types.

### Layer 3: On-Demand Skills
**Budget**: Descriptions ~200 chars each (loaded at start), full content on invocation

Skill descriptions must be keyword-rich and concise since they're the primary trigger mechanism:

```yaml
# Good description (focused, keyword-rich)
description: Comprehensive PR review across security, performance, and code quality. Use when reviewing pull requests.

# Bad description (vague, wastes tokens)
description: This skill helps you review code changes by analyzing various aspects of the code including but not limited to security concerns, performance implications, and overall code quality metrics.
```

**gdev generates** 15 skills but only ~3KB of descriptions loaded at session start. Full skill content (potentially 50KB+) loads only when invoked.

### Layer 4: Isolated Agents
**Budget**: Zero main-context cost (own context window)

Agents are the most context-efficient extension: their descriptions are used only for auto-delegation decisions, and their full system prompts run in isolated context windows.

**gdev generates** 7 agents whose work never pollutes the main conversation.

### Layer 5: External (Hooks, settings.json)
**Budget**: Zero context cost

Hooks execute externally and only add context if they return output. settings.json is parsed by the harness, never loaded into context.

**gdev generates** hooks for deterministic guardrails and settings.json for permissions/deny rules.

## 3. Context Efficiency Patterns

### Pattern: Skill Description Budget Management

With 15 skills at ~200 chars each = ~3,000 chars of descriptions. This is within the default budget (1% of context = 2,000 chars for 200k Sonnet, 10,000 chars for 1M Opus).

For Sonnet users, gdev should:
- Set `disable-model-invocation: true` on less-used skills to remove their descriptions from context
- Use `skillOverrides` in settings.json to set low-priority skills to `"name-only"`
- Keep the 5 most-used skills with full descriptions, remainder as name-only

### Pattern: Progressive Disclosure in Skills

Structure skills with a concise SKILL.md body and supporting files for detail:

```
.claude/skills/review-pr/
├── SKILL.md           # 100 lines: workflow steps
├── references/
│   └── checklist.md   # 200 lines: detailed review criteria
└── examples/
    └── good-review.md # Example output format
```

SKILL.md references supporting files: "For the detailed checklist, see [checklist.md](references/checklist.md)". Claude loads supporting files only when it needs them.

### Pattern: Agent Memory as External Knowledge Base

Instead of putting domain knowledge in CLAUDE.md (always loaded), let agents accumulate it in memory:

```yaml
memory: project  # .claude/agent-memory/security-reviewer/
```

The security-reviewer agent builds a MEMORY.md of codebase-specific security patterns across sessions. This knowledge loads only when the agent runs, never polluting the main context.

### Pattern: Dynamic Context Injection

Skills can inject live data via `!`command`` syntax, reducing the need for Claude to use tools (which add tool call overhead to context):

```yaml
## Recent Changes
!`git log --oneline -10`

## Current Branch
!`git branch --show-current`
```

This is pre-processed before Claude sees it, so the data arrives efficiently without tool call overhead.

### Pattern: Rules as Lazy-Loaded CLAUDE.md Sections

Instead of one large CLAUDE.md, decompose into rules with glob patterns:

```
.claude/rules/
├── security.md          # Always loaded (no paths: frontmatter)
├── go-conventions.md    # paths: "**/*.go"
├── ts-conventions.md    # paths: "**/*.{ts,tsx}"
├── test-conventions.md  # paths: "**/*test*", "**/*spec*"
├── api-conventions.md   # paths: "src/api/**", "routes/**"
└── db-conventions.md    # paths: "**/*migration*", "models/**"
```

## 4. Anti-Patterns to Avoid

### Anti-Pattern: Kitchen Sink CLAUDE.md
Putting everything in CLAUDE.md because "Claude should always know this." Over ~200 lines, Claude starts ignoring instructions.

### Anti-Pattern: Verbose Skill Descriptions
Long descriptions waste the skill listing budget and can cause other skills' descriptions to be truncated.

### Anti-Pattern: Auto-Invoking All Skills
Every auto-invocable skill's description is loaded every session. If a skill is rarely used, set `disable-model-invocation: true`.

### Anti-Pattern: Reference Material in CLAUDE.md
API docs, style guides, and detailed references should be skills (loaded on demand), not CLAUDE.md (loaded every request).

### Anti-Pattern: Duplicating Rules Across Levels
If a rule is in CLAUDE.md and also in a `.claude/rules/*.md` file, Claude sees it twice, wasting context.

## 5. gdev Implementation Guidance

### Config Generation Token Budget

gdev should track the estimated token cost of generated config:

```go
type ContextBudget struct {
    ClaudeMD     int // lines
    Rules        int // total lines across all rule files
    SkillDescs   int // chars of all skill descriptions
    TotalTokens  int // estimated from char count / 4
    BudgetPct    float64 // % of model context window
}

func (b ContextBudget) Validate(modelSize int) error {
    if b.TotalTokens > modelSize / 20 { // 5% threshold
        return fmt.Errorf("generated config uses %d tokens (%.1f%% of %d), exceeds 5%% budget",
            b.TotalTokens, float64(b.TotalTokens)/float64(modelSize)*100, modelSize)
    }
    return nil
}
```

### Model-Aware Generation

gdev should generate different configurations based on expected model:

| Setting | Sonnet (200k) | Opus (1M) |
|---|---|---|
| CLAUDE.md | 50 lines max | 100 lines max |
| Skills auto-invoke | Top 5 only | All 15 |
| Rules lazy-loading | Aggressive (paths: on all) | Moderate (security always-on) |
| Skill descriptions | Name-only for low-priority | Full descriptions |

## Depth Checklist

- [x] Underlying mechanism explained (context loading stages, token budgets, compaction behavior)
- [x] Key tradeoffs identified (always-on vs on-demand, description detail vs budget, isolation vs shared context)
- [x] Compared to alternatives (CLAUDE.md vs rules vs skills for same content)
- [x] Failure modes described (context bloat, instruction ignoring at >200 lines, skill description truncation)
- [x] Concrete examples found (gdev templates, rule decomposition, progressive disclosure)
- [x] Standalone-readable
