# Instruction Budget Research: How Many Instructions Can LLMs Reliably Follow?
- **Sources**:
  - https://www.humanlayer.dev/blog/writing-a-good-claude-md (HumanLayer)
  - https://dev.to/docat0209/5-patterns-that-make-claude-code-actually-follow-your-rules-44dh (DEV.to)
  - https://allarddewinter.net/blog/optimising-llm-agent-instructions-with-claudemd/ (Allard de Winter)
- **Retrieved**: 2026-03-27

## The ~150-200 Instruction Limit

Frontier thinking LLMs can follow approximately 150-200 instructions with reasonable consistency. This is a cognitive limit, not a technical one.

Claude Code's system prompt already contains ~50 individual instructions. Depending on the model, that's nearly a third of the instruction budget consumed before CLAUDE.md, rules, plugins, skills, or user messages are loaded.

## Degradation Patterns

- **Smaller models**: exponential decay in instruction-following as count increases
- **Larger frontier models**: linear decay — still degrades, but more gracefully
- **Key insight**: Claude doesn't just ignore later instructions; it ignores ALL instructions more frequently as total count rises

This means every low-value instruction actively makes high-value instructions less likely to be followed.

## Community Guidance on Size

- HumanLayer: "< 300 lines is best, shorter is even better. Our root CLAUDE.md is less than sixty lines."
- Anthropic official: "target under 200 lines per CLAUDE.md file"
- shanraisshan/claude-code-best-practice: "Keep CLAUDE.md under 200 lines per file for reliable adherence"
- General pattern: Most effective CLAUDE.md files are 50-100 lines

## 5 Patterns for Compliance (DEV.to)

1. Use positive instructions instead of negative ones
2. Be specific and verifiable
3. Structure with clear headers
4. Prioritize — put most important rules first
5. Use hooks for must-follow rules, CLAUDE.md for should-follow guidance

## Progressive Disclosure Strategy

Instead of putting everything in CLAUDE.md:
- CLAUDE.md: universal, always-applicable guidance (50-100 lines)
- .claude/rules/: topic-specific rules, can be path-scoped
- .claude/skills/: on-demand workflows loaded only when relevant
- @imports: reference detailed docs without inlining them

This matches the context window budget to actual need.
