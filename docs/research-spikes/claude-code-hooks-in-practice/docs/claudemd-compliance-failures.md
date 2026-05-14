# CLAUDE.md Compliance Failures: Bug Reports and Evidence
- **Sources**: Multiple GitHub issues on anthropics/claude-code
- **Retrieved**: 2026-03-27

## Documented Bug Reports

### Issue #7777 — Claude ignores instruction in CLAUDE.MD and agents
- Claude ignores explicit instructions, treats them as suggestions
- Persistent across multiple prompts in same session

### Issue #10683 — System prompts override explicit user rules
- System prompt instructions take priority over CLAUDE.md user instructions
- When conflicts arise, system prompt wins

### Issue #15443 — Claude ignores explicit instructions while claiming to understand them
- Claude acknowledges the rules but doesn't follow them
- Creates false confidence in compliance

### Issue #17530 — Claude Not Reading Claude.md
- Complete failure to read the file in some sessions

### Issue #19471 — Instructions completely ignored after context compaction
- After compaction in long sessions, CLAUDE.md rules get summarized away
- Critical instructions lost during context management

### Issue #21119 — Claude repeatedly ignores instructions in favor of training data patterns
- Root cause: Claude pattern-matches to training data rather than following context window instructions
- "Defaulting to 'how I usually do things' rather than 'what this project specifically requires'"

### Issue #26128 — Language and formatting instructions ignored
- Spanish technical manuals generated without proper diacritical marks
- CLAUDE.md formatting rules not followed for document generation

### Issue #34774 — Committed changes without explicit user permission
- Despite "NEVER commit without asking" in CLAUDE.md
- System prompt conflict may be a factor

### Issue #35309 — Claude Code disregards stored instructions
- Recent (2026) report of continued non-compliance

## Common Failure Patterns

1. **System prompt conflicts**: CLAUDE.md says one thing, system prompt says another → system prompt wins
2. **Training data bias**: Claude defaults to training patterns over context instructions
3. **Context compaction loss**: Rules get summarized/dropped during long sessions
4. **False acknowledgment**: Claude says "I understand the rules" then violates them
5. **Negative instruction failure**: "NEVER do X" is less effective than "always do Y instead"
6. **Length-based degradation**: Longer CLAUDE.md files have lower per-instruction compliance

## The "200 Lines of Rules" Case (dev.to)
- Developer wrote 200 lines of rules for Claude Code
- Title: "I Wrote 200 Lines of Rules for Claude Code. It Ignored Them All."
- URL: https://dev.to/minatoplanb/i-wrote-200-lines-of-rules-for-claude-code-it-ignored-them-all-4639
- Demonstrates the length-compliance inverse relationship

## Implications for CLAUDE.md Strategy

1. CLAUDE.md is fundamentally advisory — not enforceable
2. Critical rules need hooks (deterministic enforcement)
3. Keep CLAUDE.md short to maximize per-instruction compliance
4. Positive instructions > negative instructions
5. Consider context compaction behavior when placing critical rules
6. Test compliance rather than assuming it
