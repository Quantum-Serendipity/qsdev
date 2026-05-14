# CLAUDE.md Patterns in the Wild

## Executive Summary

CLAUDE.md is Claude Code's project-level instruction file — persistent markdown that loads into every session's context window. After surveying Anthropic's official documentation, 8+ real-world CLAUDE.md files, community guides, curated lists (awesome-claude-code at 33k stars), and dozens of bug reports, clear patterns emerge: effective CLAUDE.md files are short (50-100 lines), concrete (verifiable instructions over philosophy), and focused on what Claude cannot infer from code alone. The most critical finding is the **instruction budget constraint** — frontier LLMs can follow ~150-200 instructions reliably, and Claude Code's system prompt already consumes ~50 of those, leaving roughly 100-150 for CLAUDE.md, rules, skills, and conversation combined. Every low-value instruction added actively degrades compliance with high-value ones. This has direct implications for consulting adoption: CLAUDE.md handles guidance, hooks handle enforcement, and the two are complements, not substitutes.

---

## 1. What CLAUDE.md Is (Architecture)

### Loading Hierarchy

CLAUDE.md files load from multiple locations with increasing specificity:

1. **Managed policy** (`/etc/claude-code/CLAUDE.md` on Linux) — organization-wide, cannot be excluded
2. **User-level** (`~/.claude/CLAUDE.md`) — personal preferences across all projects
3. **Ancestor directories** — walking UP from working directory, all CLAUDE.md files found
4. **Project root** (`./CLAUDE.md` or `./.claude/CLAUDE.md`) — team-shared via git
5. **CLAUDE.local.md** — personal overrides, gitignored
6. **Subdirectory CLAUDE.md** — lazy-loaded on demand when Claude accesses those directories

All files are concatenated additively. When instructions conflict, Claude uses judgment — more specific typically wins, but there's no guarantee. Content is delivered as a **user message after the system prompt**, not as system prompt itself.

### Adjacent Systems

- **`.claude/rules/`** — topic-specific rule files, support path-scoped YAML frontmatter for conditional loading
- **Skills** (`.claude/skills/`) — on-demand workflows, loaded only when relevant (progressive disclosure)
- **Auto memory** (`~/.claude/projects/<project>/memory/`) — Claude's own notes, capped at 200 lines/25KB
- **`@import` syntax** — reference external files from CLAUDE.md, max depth 5 hops
- **`AGENTS.md` bridge** — `@AGENTS.md` in CLAUDE.md lets both Claude Code and other agents share instructions

Source: `docs/anthropic-memory-docs-claudemd.md`

---

## 2. Instruction Categories Found in Real CLAUDE.md Files

Analysis of 8+ real-world CLAUDE.md files reveals consistent categories. Not all files include all categories, but these are the building blocks:

### Category Taxonomy

| Category | Frequency | Examples |
|----------|-----------|---------|
| **Build/test/lint commands** | Universal | `bun test`, `uv run pytest`, `npm run lint` |
| **Architecture overview** | Very common | Component descriptions, entrypoints, data flow |
| **Code style rules** | Common | Indentation, naming, import style, typing |
| **Gotchas / "Things That Will Bite You"** | Common | Non-obvious behaviors, TypeScript strictness, token lifecycle |
| **Testing philosophy** | Common | TDD, real objects vs mocks, coverage requirements |
| **Git/PR conventions** | Common | Branch naming, commit format, reviewer assignments |
| **Forbidden actions** | Common | "NEVER use pip", "NEVER commit without asking" |
| **File organization** | Moderate | Directory structure, where to put new files |
| **Workflow rules** | Moderate | Explore-plan-code-commit, plan mode for complex tasks |
| **Skill activation routing** | Emerging | "Creating tests → testing-patterns skill" |
| **Cross-project dependencies** | Monorepo-specific | "Changing browser-use may affect cloud" |
| **Component priority** | Monorepo-specific | "web-ui is less important, ignore unless directed" |

### What's Notably Absent

- Detailed API documentation (official guidance: link to docs instead)
- Standard language conventions Claude already knows
- File-by-file descriptions of the codebase
- Information that changes frequently
- Self-evident practices ("write clean code")

Source: `docs/example-*.md`, `docs/anthropic-best-practices-claudemd.md`

---

## 3. Real-World Examples Analyzed

### Anthropic's Own CLAUDE.md (claude-code-action)
- **~60 lines**
- Sections: Commands → What This Is → How It Runs → Key Concepts → Things That Will Bite You → Code Conventions
- Gotcha-focused: longest section is non-obvious behaviors
- Architecture-heavy: explains mode lifecycle, auth priority, prompt construction
- No code style rules, no philosophical guidelines
- Follows their own advice: only what Claude can't infer from code
- Source: `docs/anthropic-claude-code-action-claudemd.md`

### Browser Use Monorepo (~300 lines)
- Largest real-world example found
- Explicit component priority ("less important, ignore unless directed")
- Project-specific idioms (tabs in some projects, spaces in others)
- TDD and testing philosophy embedded
- Pydantic v2 specific patterns
- Cross-package dependency awareness
- Source: `docs/browser-use-monorepo-claudemd.md`

### ArthurClune Python Template (~200 lines)
- Comprehensive tooling: package management, formatting, type checking, pre-commit
- Error resolution sequences (fix formatting → types → linting)
- Strong negative instructions: "NEVER ever mention co-authored-by"
- Placeholder sections for customization (`[fill in here]`)
- Source: `docs/example-python-claudemd.md`

### ArthurClune Terraform Template (~200 lines)
- Security scanning (checkov) integrated as standard workflow step
- Multi-file awareness as first critical rule
- Complete bash commands for every tool
- Before/after change verification workflow
- Source: `docs/example-terraform-claudemd.md`

### ArthurClune Hugo Blog (~30 lines)
- Minimal: architecture, build commands, style rules, deployment
- Uses NEVER/ALWAYS emphasis for critical rules
- Content-focused rather than code-focused
- Source: `docs/example-hugo-claudemd.md`

### ChrisWiles Showcase (~70 lines)
- "Quick Facts" header (stack + commands)
- Skill activation routing pattern
- Git conventions (branch naming, commit format)
- UI state requirements (loading, error, empty, success)
- Source: `docs/example-showcase-claudemd.md`

### coleam00 Context Engineering (~100 lines)
- References external files: PLANNING.md and TASK.md
- AI-specific behavior rules ("never hallucinate", "ask if uncertain")
- File length limits (500 lines max)
- Test coverage requirements (expected, edge case, failure case)
- Source: `docs/example-context-engineering-claudemd.md`

### shanraisshan Best Practice (~200 lines)
- Meta-documentation: documents how to document
- Configuration hierarchy explicitly spelled out
- "Keep CLAUDE.md under 200 lines per file" — self-referencing guidance
- All hook events listed
- Source: `docs/example-best-practice-claudemd.md`

### Size Distribution

| Lines | Examples |
|-------|----------|
| ~30 | Hugo blog |
| ~60 | Anthropic claude-code-action, HumanLayer's own |
| ~70 | ChrisWiles showcase |
| ~100 | coleam00 context engineering |
| ~200 | ArthurClune Python, Terraform, shanraisshan |
| ~300 | Browser Use monorepo |

The most effective files cluster around 50-100 lines. Files over 200 lines show diminishing returns per the instruction budget research.

---

## 4. The Instruction Budget Problem

### The Core Constraint

Frontier thinking LLMs can follow approximately **150-200 instructions** with reasonable consistency. Claude Code's system prompt already consumes ~50 of these. This leaves roughly 100-150 instructions for everything else: CLAUDE.md, .claude/rules/, skills, auto memory, and user conversation.

### Degradation Characteristics

- **Smaller models**: exponential decay in compliance as instruction count rises
- **Frontier models**: linear decay — still degrades, but more gracefully
- **Critical insight**: Adding low-value instructions doesn't just waste space — it actively reduces compliance with high-value instructions

### Evidence from Bug Reports

Multiple GitHub issues document CLAUDE.md compliance failures:

- **#7777**: Claude treats instructions as suggestions, not rules
- **#10683**: System prompt overrides user CLAUDE.md rules when they conflict
- **#15443**: Claude acknowledges rules then violates them (false acknowledgment)
- **#19471**: Instructions lost after context compaction (though Anthropic's docs now state CLAUDE.md survives compaction — this may be fixed)
- **#21119**: Claude defaults to training data patterns over context instructions
- **#34774**: "NEVER commit without asking" violated despite explicit instruction

Common failure modes:
1. System prompt conflicts (system prompt wins)
2. Training data bias (defaults to learned patterns)
3. False acknowledgment (says "I understand" then violates)
4. Negative instruction failure ("NEVER do X" less effective than "always do Y")
5. Length-based degradation (more instructions → less compliance per instruction)

Source: `docs/claudemd-compliance-failures.md`, `docs/instruction-budget-research.md`

### The "200 Lines of Rules" Case

A developer wrote 200 lines of rules and reported "It Ignored Them All" (dev.to). This matches the instruction budget theory — at ~200 lines, you're likely exceeding the effective instruction count, especially when combined with system prompt instructions.

---

## 5. What Works vs. What Doesn't

### Patterns That Improve Compliance

1. **Positive over negative**: "Always use uv" works better than "NEVER use pip" (Pink Elephant Problem)
2. **Concrete and verifiable**: "Use 2-space indentation" not "format code properly"
3. **Emphasis markers**: "IMPORTANT:", "YOU MUST", "NEVER" — improves but does not guarantee adherence
4. **Structure**: Markdown headers and bullets scan better than dense paragraphs
5. **Priority ordering**: Most important rules first (higher position → more attention)
6. **Short files**: 50-100 lines sweet spot
7. **Commands first**: Build/test/lint commands are the most universally useful content
8. **Gotchas section**: Non-obvious behaviors that Claude can't infer from code

### Patterns That Reduce Compliance

1. **Length**: Files over 200 lines show significant compliance drops
2. **Vague instructions**: "Write clean code", "Keep things organized"
3. **Contradictory rules**: Multiple CLAUDE.md files with conflicting guidance
4. **Redundant rules**: Restating what Claude already knows from conventions
5. **Negative instructions**: "Don't do X" triggers the Pink Elephant effect
6. **Philosophy sections**: Development philosophy that doesn't translate to verifiable behavior
7. **Detailed API docs inline**: Should be linked, not inlined

### The Official Anti-Pattern List (Anthropic)

| Include | Exclude |
|---------|---------|
| Bash commands Claude can't guess | Anything Claude can figure out from code |
| Code style rules differing from defaults | Standard language conventions |
| Testing instructions | Detailed API documentation |
| Repository etiquette | Information that changes frequently |
| Architectural decisions | Long explanations or tutorials |
| Dev environment quirks | File-by-file descriptions |
| Common gotchas | Self-evident practices |

---

## 6. CLAUDE.md vs. Hooks: The Enforcement Spectrum

This is the most critical pattern for consulting adoption.

### The Fundamental Distinction

> "Your CLAUDE.md Is a Suggestion. Hooks Make It Law." — Medium article title

- **CLAUDE.md**: advisory, probabilistic, context-dependent. Claude reads it and *tries* to follow it.
- **Hooks**: deterministic, guaranteed, zero-exception. Scripts that run automatically at specific lifecycle points.

### When to Use Each

| Requirement | Mechanism | Why |
|-------------|-----------|-----|
| Code style preferences | CLAUDE.md | Soft guidance, Claude adapts to context |
| Architecture decisions | CLAUDE.md | Informational, shapes approach |
| "Run linter after edits" | **Hook** (PostToolUse) | Must happen every time, no exceptions |
| "Block writes to migrations/" | **Hook** (PreToolUse) | Security boundary, must be deterministic |
| "Use conventional commits" | CLAUDE.md + Hook | CLAUDE.md for format guidance, hook to validate |
| Testing requirements | CLAUDE.md + Hook | CLAUDE.md for philosophy, Stop hook to enforce |
| Credential scanning | **Hook** (PreToolUse) | Zero-tolerance security requirement |

### The Gap That Matters

> "A CLAUDE.md instruction says 'always run the linter.' The agent usually complies. A PostToolUse hook runs the linter after every file write, every single time, no exceptions. That gap between 'usually' and 'always' is where production systems fail."

For consulting firms: anything that would cause client escalation if missed belongs in a hook, not CLAUDE.md.

---

## 7. Team/Org Standardization Patterns

### Configuration Hierarchy for Teams

Anthropic provides a clear hierarchy for organizational deployment:

1. **Managed CLAUDE.md** (IT/DevOps deploys to system path) — company coding standards, security policies, compliance. Cannot be overridden.
2. **Managed settings** (`permissions.deny`, `sandbox.enabled`) — technical enforcement (block tools, restrict network)
3. **Project CLAUDE.md** (git-committed) — team-shared coding standards
4. **`.claude/rules/`** (git-committed) — modular rules, can be path-scoped
5. **CLAUDE.local.md** (gitignored) — personal overrides
6. **`~/.claude/CLAUDE.md`** — personal global preferences

### Monorepo Patterns

- Root CLAUDE.md for shared standards
- Package-level CLAUDE.md for package-specific guidance (lazy-loaded)
- `claudeMdExcludes` setting to skip irrelevant team's files
- Path-scoped rules in `.claude/rules/` for framework-specific guidance

### Symlink Pattern for Shared Standards

```bash
ln -s ~/company-standards/security.md .claude/rules/security.md
ln -s ~/shared-claude-rules .claude/rules/shared
```

This enables org-wide rules in a central repo, symlinked into each project.

### Template Ecosystem

- ArthurClune/claude-md-examples: language-specific templates (Python, Terraform, Hugo)
- ruvnet/ruflo wiki: CLAUDE.md templates for various frameworks
- `/init` command: generates starter CLAUDE.md from codebase analysis
- CLAUDE.md Generator tools: automated generation from project structure
- claude-code-templates (davila7): CLI tool for configuration

### Tools for CLAUDE.md Management

- **claude-rules-doctor**: detects dead rule files
- **ClaudeCTX**: switch entire configurations with single command
- **context-drift** tool: catches when CLAUDE.md drifts out of sync with reality (wrong paths, dead scripts, stale deps)

---

## 8. Consulting-Relevant Patterns

### Per-Client Configuration

The hierarchy naturally supports per-client setups:
- `~/.claude/CLAUDE.md`: firm-wide coding standards
- `~/.claude/rules/security.md`: firm-wide security rules
- `./CLAUDE.md`: client project standards
- `.claude/rules/`: client-specific rules (compliance, framework patterns)
- Managed settings: enforce sandbox, restrict tools for sensitive clients

### Security and Compliance

- **Managed policy CLAUDE.md** for org-wide compliance reminders (GDPR, data handling)
- **Managed settings** (`permissions.deny`) for hard enforcement (block commands, restrict file access)
- **Hooks** for zero-tolerance requirements (credential scanning, migration protection)
- CLAUDE.md for softer security guidance (logging practices, error handling patterns)

### Code Review Requirements

- CLAUDE.md: document PR conventions, reviewer assignments, commit format
- Hooks: enforce lint/typecheck passes before allowing commit
- Skills: automate review workflows (/security-review, /fix-issue)
- Subagents: dedicated security reviewer with scoped tools

### Testing Enforcement

Best practice is layered:
1. CLAUDE.md: testing philosophy, preferred patterns, coverage expectations
2. Hook (Stop): verify tests pass before session completion
3. Hook (PostToolUse): run relevant tests after file edits
4. Skill: codify TDD workflow as reusable command

---

## 9. The Progressive Disclosure Pattern

The most important architectural pattern that emerges across all sources:

```
CLAUDE.md (50-100 lines)     ← Universal, always loaded
    │
    ├── .claude/rules/        ← Topic-specific, can be path-scoped
    │   ├── code-style.md
    │   ├── testing.md
    │   └── api-design.md
    │
    ├── .claude/skills/       ← On-demand, loaded only when relevant
    │   ├── fix-issue/
    │   └── security-review/
    │
    ├── @imports              ← Referenced docs, expanded at load
    │   ├── @README.md
    │   └── @docs/arch.md
    │
    └── Hooks                 ← Deterministic enforcement
        ├── PreToolUse
        ├── PostToolUse
        └── Stop
```

This matches instruction budget to actual need:
- CLAUDE.md handles the ~50 most universal instructions
- Rules add ~20-30 more when working in specific areas
- Skills load detailed workflows only when invoked
- Hooks enforce without consuming instruction budget at all

---

## 10. Open Questions

1. **Quantitative compliance data**: No public benchmarks for instruction-following rates at different CLAUDE.md lengths. The ~150-200 instruction limit is cited but not empirically validated with published methodology.

2. **Compaction behavior**: Anthropic says CLAUDE.md "fully survives compaction" (re-read from disk), but bug reports suggest instructions are still lost. Has this been genuinely fixed, or does the re-injection still compete with compacted context?

3. **Path-scoped rules effectiveness**: How well does lazy loading actually work in practice? Do path-scoped rules reliably activate?

4. **Managed policy adoption**: How many enterprises actually deploy managed CLAUDE.md? What's the deployment tooling story beyond "use MDM"?

5. **Instruction ordering effects**: Is there evidence that instructions earlier in CLAUDE.md are followed more reliably than later ones (primacy bias)?

---

## Sources

All raw sources saved in `docs/`:
- `anthropic-best-practices-claudemd.md` — Official best practices documentation
- `anthropic-memory-docs-claudemd.md` — Official memory/CLAUDE.md documentation
- `anthropic-claude-code-action-claudemd.md` — Anthropic's own CLAUDE.md
- `claude-code-system-prompt-v0227.md` — Claude Code system prompt (early version)
- `awesome-claude-code-repo.md` — Curated list of Claude Code resources (33k stars)
- `claude-md-examples-repo.md` — ArthurClune's language-specific templates
- `browser-use-monorepo-claudemd.md` — Real-world monorepo CLAUDE.md (~300 lines)
- `example-python-claudemd.md` — Python project template
- `example-terraform-claudemd.md` — Terraform project template
- `example-hugo-claudemd.md` — Hugo blog template
- `example-showcase-claudemd.md` — React/TypeScript showcase
- `example-best-practice-claudemd.md` — Best practice reference implementation
- `example-context-engineering-claudemd.md` — Context engineering pattern
- `claudemd-compliance-failures.md` — Bug reports documenting instruction non-compliance
- `instruction-budget-research.md` — Research on LLM instruction-following limits

### Key External Sources (not fetched in full)
- HumanLayer blog: "Writing a good CLAUDE.md" — instruction budget analysis, <60 line recommendation
- Builder.io: "How to Write a Good CLAUDE.md File" — practical guide
- Martin Fowler / Birgitta Böckeler: "Context Engineering for Coding Agents" — framework for understanding CLAUDE.md in context engineering
- Medium: "Your CLAUDE.md Is a Suggestion. Hooks Make It Law." — enforcement spectrum analysis
- DEV.to: "I Wrote 200 Lines of Rules for Claude Code. It Ignored Them All." — length-compliance case study
- DEV.to: "5 Patterns That Make Claude Code Actually Follow Your Rules" — compliance patterns
- 16x.engineer: "The Pink Elephant Problem" — negative instruction failure analysis
