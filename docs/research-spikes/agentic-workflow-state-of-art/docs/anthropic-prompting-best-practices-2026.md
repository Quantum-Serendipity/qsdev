# Anthropic Claude Prompting Best Practices (2026)

- **Source URL**: https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices
- **Retrieved**: 2026-03-15
- **Note**: Full page content retrieved via WebFetch. This is the authoritative reference for prompt engineering with Claude 4.6 models.

---

## General Principles

### Be Clear and Direct
Claude responds well to clear, explicit instructions. Being specific about desired output enhances results. If you want "above and beyond" behavior, explicitly request it.

**Golden rule:** Show your prompt to a colleague with minimal context. If they'd be confused, Claude will be too.

### Add Context to Improve Performance
Providing motivation behind instructions helps Claude understand goals. Example: Instead of "NEVER use ellipses", say "Your response will be read aloud by a text-to-speech engine, so never use ellipses since the text-to-speech engine will not know how to pronounce them."

### Use Examples Effectively (Few-Shot)
3-5 examples recommended for best results. Make them:
- **Relevant**: Mirror actual use case
- **Diverse**: Cover edge cases
- **Structured**: Wrap in `<example>` tags

### Structure Prompts with XML Tags
XML tags help Claude parse complex prompts unambiguously. Use consistent, descriptive tag names. Nest tags for hierarchical content.

### Give Claude a Role
Setting a role in system prompt focuses behavior and tone. Even a single sentence makes a difference.

### Long Context Prompting
- Put longform data at top, query at end (up to 30% improvement)
- Use XML tags for document structure
- Ask Claude to quote relevant parts before answering

## Output and Formatting

### Communication Style
Claude 4.6 models are more concise, direct, and less verbose than previous models. May skip summaries after tool calls.

### Control Format
1. Tell Claude what to DO instead of what NOT to do
2. Use XML format indicators
3. Match prompt style to desired output style
4. Use detailed prompts for specific formatting preferences

### Aggressive Language Hurts Performance
"CRITICAL!", "YOU MUST", "NEVER EVER" produce worse results than calm, direct instructions on Claude 4.5+ models. Where you might have said "CRITICAL: You MUST use this tool when...", use "Use this tool when..."

### Prefilled Responses Deprecated
Starting with Claude 4.6, prefilled responses on last assistant turn no longer supported. Use structured outputs, direct instructions, or XML tags instead.

## Tool Use

### Be Explicit About Actions
Claude may suggest rather than implement. Say "Change this function" not "Can you suggest changes?"

### Parallel Tool Calling
Claude 4.6 excels at parallel execution. Easily steerable to ~100% with explicit instructions about calling independent tools simultaneously.

### Proactive vs Conservative Action
Steerable via system prompt - can be set to default-to-action or default-to-information.

## Thinking and Reasoning

### Adaptive Thinking (Claude 4.6)
Uses `thinking: {type: "adaptive"}` where Claude decides when/how much to think. Outperforms manual extended thinking in internal evaluations.

### Guidance
- "Think thoroughly" often better than prescriptive step-by-step
- Multishot examples work with thinking (use `<thinking>` tags)
- Ask Claude to self-check against criteria

### Overthinking Prevention
Claude Opus 4.6 does significantly more upfront exploration. Replace blanket defaults with targeted instructions. Remove over-prompting that was needed for older models.

## Agentic Systems

### Long-Horizon Reasoning
Claude maintains orientation across extended sessions by focusing on incremental progress.

### Context Management
- Context awareness tracks remaining window
- Use structured formats (JSON) for state data
- Use unstructured text for progress notes
- Use git for state tracking across sessions
- Prompt about context compaction to prevent premature stopping

### Subagent Orchestration
Claude 4.6 natively recognizes when to delegate. May over-use subagents. Add guidance about when subagents are/aren't warranted.

### Reducing Overengineering
Explicit guidance to keep solutions minimal. "Only make changes that are directly requested or clearly necessary."

### Minimizing Hallucinations
"Never speculate about code you have not opened. Read relevant files BEFORE answering."

### Test-Focused Coding
Prevent hard-coding to tests: "Implement a solution that works correctly for all valid inputs, not just the test cases."
