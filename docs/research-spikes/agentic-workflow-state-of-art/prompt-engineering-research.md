# Prompt Engineering and Instruction Following: State of the Art (2025-2026)

## Executive Summary

The field of prompt engineering has undergone a significant conceptual shift in 2025-2026, evolving from isolated prompt crafting into **context engineering** — the holistic design of the information architecture surrounding LLM interactions. The core finding across all sub-topics is that modern frontier models (Claude 4.6, GPT-4o, Gemini) respond best to clear, calm, structured instructions with explicit success criteria, and that many practices from the 2023-2024 era (aggressive language, heavy few-shot examples, elaborate role-playing) are now counterproductive on newer models. This report covers 10 technique categories with evidence of effectiveness and specific applicability to Claude Code system prompts and CLAUDE.md files.

---

## 1. System Prompt Engineering

### How It Works
System prompts set the foundational context, behavior, and constraints for an LLM before any user interaction. They are processed with higher priority than user messages and persist across conversation turns.

### Best Practices (2025-2026)

**Write prompts like contracts, not conversations.** The #1 best practice in 2026 is writing success criteria and an output contract. A good Claude system prompt reads like a short contract — explicit, bounded, and easy to check. Include:
- **Success criteria**: What "done" looks like
- **Output contract**: Format, length, tone, required sections (testable)
- **Constraints**: Scope, assumptions, exclusions

**Structure matters more than length.** The recommended structure is:
1. INSTRUCTIONS (what to do and how to behave)
2. CONTEXT (background information and data)
3. TASK (the specific request)
4. OUTPUT FORMAT (exact structure expected)

For Claude specifically, **XML tags are the optimal structuring mechanism** — not Markdown, not numbered lists. Claude was specifically trained to recognize and process XML-structured content. Without XML tags, Claude's analysis is disorganized and misses key points; with tags, it provides structured, thorough analysis. The performance difference is most pronounced for analytical and multi-part tasks. However, XML consumes ~15% more tokens than equivalent Markdown.

**Conciseness outperforms verbosity.** After approximately 2,000 tokens, most models start performing worse. For simple tasks, aim for 500-700 tokens; for complex reasoning, 800-1,200 tokens. A well-structured 16K-token prompt with RAG outperformed a monolithic 128K-token prompt. For CLAUDE.md files specifically, the community consensus is to target under 200 lines — if too long, Claude ignores half of it because important rules get lost in the noise.

**Front-load critical content.** Place long documents and inputs near the top, with queries and instructions at the end. Queries at the end can improve response quality by up to 30%, especially with complex multi-document inputs.

### Evidence of Effectiveness
- Anthropic's official documentation codifies these practices for Claude 4.6 (source: `docs/anthropic-prompting-best-practices-2026.md`)
- LangChain's 2025 State of Agent Engineering: 32% of organizations cite quality as top barrier, with most failures traced to poor context management
- Nature study (April 2025): Only 8% of real-world use cases benefit from prompts exceeding 32K tokens

### When It Helps vs Hurts
- **Helps**: Every use case benefits from clear system prompts. Multi-step agentic workflows benefit most.
- **Hurts**: Over-long system prompts degrade performance. On Claude 4.6, overly aggressive system prompts cause overtriggering of tools and behaviors that were appropriately calibrated for older models.

### Applicability to Claude Code / CLAUDE.md
- CLAUDE.md **is** the system prompt for Claude Code. Every practice here applies directly.
- Use monorepo pattern: root CLAUDE.md + subfolder-specific files for focused context
- Front-load the most critical rules; put detailed reference material in separate files
- Structure with clear sections, each with explicit behavioral specifications

---

## 2. Chain-of-Thought Prompting

### How It Works
Chain-of-thought (CoT) prompting encourages models to produce intermediate reasoning steps before giving a final answer. Variants include:
- **Zero-shot CoT**: Simply adding "Think step by step" or "Let's think through this"
- **Few-shot CoT**: Providing examples that include reasoning chains
- **Adaptive thinking** (Claude 4.6): Model dynamically decides when and how much to think

### Current State of the Art

**Zero-shot CoT often outperforms few-shot CoT on modern models.** A 2025 paper ("Revisiting Chain-of-Thought Prompting") demonstrated that for sufficiently strong LLMs like Qwen2.5-72B, traditional few-shot CoT exemplars primarily enforce output format rather than increase reasoning ability. Specific numbers: Qwen2.5-72B achieved 81.2% on GSM8K with zero-shot CoT vs 79.0% with 8-shot CoT; on MATH, 55.3% (zero-shot) vs 53.8% (8-shot).

**CoT faithfulness is a serious concern.** Anthropic's May 2025 paper "Reasoning Models Don't Always Say What They Think" found that:
- Claude 3.7 Sonnet's CoT mentioned actual reasoning hints only 25% of the time
- DeepSeek R1 was faithful only 39% of the time
- Faithfulness declines on harder tasks (Claude: 44% drop on GPQA vs MMLU)
- Longer CoTs correlate with less faithful reasoning (Claude: 2064 tokens unfaithful vs 1439 faithful)
- Even with extensive training, faithfulness did not improve beyond 28%
- Models learn reward hacks on >99% of prompts but almost never (<2%) verbalize them in CoT

This means CoT should not be relied upon as a transparency or safety mechanism, even though it improves accuracy.

**Adaptive thinking outperforms manual CoT on Claude 4.6.** Anthropic reports that in internal evaluations, adaptive thinking (`thinking: {type: "adaptive"}`) reliably drives better performance than extended thinking with manual budget. The model self-calibrates based on query complexity.

### When It Helps vs Hurts
- **Helps**: Mathematical reasoning, logical deduction, complex multi-step problems, difficult questions
- **Hurts/Neutral**: Simple classification, sentiment analysis, tasks with obvious answers (CoT can introduce errors through overthinking)
- **Important caveat**: On Claude 4.6, "think thoroughly" often produces better reasoning than prescriptive step-by-step plans. The model's own reasoning frequently exceeds what a human would prescribe.

### Concrete Examples
```
# Effective (zero-shot CoT for Claude)
"Think through this problem carefully before answering."

# Effective (guided reflection after tool use)
"After receiving tool results, carefully reflect on their quality
and determine optimal next steps before proceeding."

# Less effective on modern models
"Step 1: First identify... Step 2: Then calculate... Step 3: Finally..."
(Too prescriptive — model's own reasoning often better)
```

### Applicability to Claude Code / CLAUDE.md
- Prefer adaptive thinking configuration over manual CoT instructions
- In CLAUDE.md, use general thinking directives: "Think thoroughly about complex problems" rather than prescriptive step sequences
- For revision cycles: "Before marking complete, re-read and check against criteria" — this triggers useful self-verification reasoning
- Include `<thinking>` tags in few-shot examples to demonstrate reasoning patterns

---

## 3. Few-Shot and In-Context Learning

### How It Works
Providing examples of desired input-output pairs within the prompt. The model uses these to understand the task format, style, and expected behavior without any weight updates.

### Optimal Number of Examples
- Anthropic recommends **3-5 examples** for best results
- Research shows diminishing returns and potential degradation beyond a threshold
- Datasets with longer problem lengths require fewer examples
- For reasoning tasks specifically, zero-shot often outperforms few-shot (as of 2025)
- **Many-shot** (hundreds-thousands of examples) shows significant gains across generative and discriminative tasks when context windows allow, but is impractical for system prompts

### Example Selection Strategies
1. **Similarity-based selection**: Choosing examples similar to the target query significantly outperforms random selection
2. **Diversity**: Cover edge cases and vary enough to prevent unintended pattern learning
3. **Quality over quantity**: A few high-quality examples outperform many mediocre ones
4. **Format demonstration**: Examples are most valuable for showing desired output format rather than teaching reasoning (for modern models)

### Evidence
- ICL with as few as 32 examples can match or exceed traditional ML models on some tasks
- Similarity-based demonstration selection is a consistent finding across multiple 2025 papers
- Example quality has stronger impact than quantity on output quality
- ICML 2025: Continued pre-training can teach models to learn from handful of examples at inference, matching supervised fine-tuning

### When It Helps vs Hurts
- **Helps most**: Output formatting, classification, style matching, structured extraction
- **Helps less**: Complex reasoning (where zero-shot CoT often wins)
- **Hurts**: When examples contain errors, are irrelevant, or anchor the model to suboptimal patterns
- **Critical**: Wrap examples in `<example>` tags for Claude to distinguish them from instructions

### Applicability to Claude Code / CLAUDE.md
- Use 1-3 examples in CLAUDE.md for critical output formats (e.g., log entry format, task entry format)
- Examples are how the research spike file format conventions are communicated in our CLAUDE.md
- Keep examples compact; they consume prompt budget
- Ensure examples are representative of actual use — not idealized edge cases

---

## 4. Instruction Hierarchies

### How It Works
Modern LLMs process instructions from multiple sources: system prompts (developer), user messages, and third-party content (tool outputs, retrieved documents). Instruction hierarchies define precedence when these sources conflict.

### The Priority Stack
1. **System prompt** (highest priority — developer/application level)
2. **User message** (middle priority)
3. **Third-party content** (lowest priority — tool outputs, retrieved docs, web content)

### Research Findings

The seminal paper "The Instruction Hierarchy" (Wallace et al., 2024, ICLR 2025) proposed training LLMs to selectively ignore lower-privileged instructions when conflicts arise. Applied to GPT-3.5, it "drastically increases robustness — even for attack types not seen during training — while imposing minimal degradations on standard capabilities."

Prompt injection is now #1 in OWASP Top 10 for LLM Applications (2025). Subsequent work includes:
- **Instructional Segment Embedding**: Incorporates instruction-type information directly into the model architecture
- **SecAlign**: Reduces prompt injection success rates to <10% through preference optimization

### Practical Implications
- System prompts are the most powerful lever for controlling behavior
- User-provided content should be clearly demarcated (XML tags: `<user_input>`, `<document>`)
- Tool outputs should be treated as untrusted by default
- Never embed critical safety instructions only in user-accessible areas

### When It Helps vs Hurts
- **Helps**: Any production system processing untrusted input; multi-agent systems where sub-agents return content
- **Hurts/Risk**: Over-rigid hierarchies can prevent legitimate user overrides; balance safety with usability

### Applicability to Claude Code / CLAUDE.md
- CLAUDE.md operates at the system prompt level — it has the highest instruction priority
- When designing agent harnesses, treat tool outputs (file contents, web fetches) as lower-priority context
- Critical behavioral rules belong in CLAUDE.md, not in per-task user prompts
- Our current CLAUDE.md correctly places workflow rules at the system level

---

## 5. Meta-Prompting and Automated Prompt Optimization

### How It Works
Meta-prompting uses LLMs to generate, evaluate, and iteratively improve prompts. The three major frameworks are:

**OPRO (Optimization by PROmpting)** — Google DeepMind
- Starts with meta-prompt containing task description + previous solutions with scores
- LLM generates new candidate solutions each step
- Candidates evaluated and fed back into the next step
- Results: Up to 8% improvement over human prompts on GSM8K, up to 50% on Big-Bench Hard
- Limitation: Only works with frontier-class models; small models lack self-optimization ability

**DSPy with MIPROv2** — Stanford
- Three phases: (1) bootstrap few-shot examples, (2) propose instruction candidates, (3) Bayesian optimization search
- Data-aware and demonstration-aware instruction generation
- Can optimize few-shot examples and instructions jointly
- Systematic, programmatic approach replaces manual prompt crafting

**EvoPrompt** — Evolutionary approach
- Starts from population of prompts
- Uses LLM + evolutionary operators to generate variations
- Selection based on development set performance
- Up to 25% improvement over human-engineered prompts on BBH
- Extended to vision-language models in 2025

### Evidence of Effectiveness
- OPRO prompts transfer across benchmarks within the same domain
- DSPy MIPROv2 jointly optimizes instructions + demonstrations via Bayesian search
- EvoPrompt tested on 31 datasets across language understanding, generation, and BBH
- Promptomatix (2025) combines meta-prompt optimizer with DSPy compiler, adding cost-aware objectives

### When It Helps vs Hurts
- **Helps**: Production systems with measurable quality metrics; tasks with clear evaluation criteria; batch processing where optimization cost is amortized
- **Hurts/Limited**: One-off tasks; highly novel tasks without training data; small models (OPRO ineffective on LLaMa-2, Mistral 7B)

### Applicability to Claude Code / CLAUDE.md
- **Indirect applicability**: CLAUDE.md files could theoretically be optimized using these techniques, but the evaluation criteria for research workflows are hard to formalize
- **Practical takeaway**: The principle that systematic evaluation + iteration outperforms manual guessing applies to CLAUDE.md refinement
- **Action item**: Track which CLAUDE.md instructions produce good vs poor results over time, then iterate. This is manual meta-prompting.
- DSPy's concept of separating program logic from prompt text is philosophically aligned with CLAUDE.md design — declarative rules rather than conversational instructions

---

## 6. Structured Output Prompting

### How It Works
Techniques for getting LLMs to produce reliably structured output (JSON, code, specific formats). Three approaches:

1. **Instruction-based**: Ask the model to produce a specific format in the prompt
2. **Schema enforcement** (API-level): Specify a JSON schema that constrains output at the API level (Claude's `output_config.format`, OpenAI's structured outputs)
3. **Grammar-guided generation** (decoding-level): Constrain token sampling using formal grammars during generation (Guidance, Outlines, XGrammar, llama.cpp)

### Current State (2025)

**Schema enforcement has become production-ready.** Claude offers:
- `output_config.format` for JSON outputs
- `strict: true` for tool use validation
- Compiled grammars cached for 24 hours

Open-source engines XGrammar and llguidance achieved **near-zero overhead** constrained decoding, enabling structured output at production scale. OpenAI credited llguidance for foundational work in May 2025.

**Quality trade-off debate ongoing.** Some research suggests constrained decoding may slightly reduce output quality — the constraint prevents generating optimal intermediate reasoning tokens. Practical impact appears small for well-designed schemas.

**For Claude specifically, XML tags are often sufficient.** Anthropic recommends XML format indicators as a lightweight alternative to full schema enforcement:
```xml
Write your analysis in <analysis> tags.
Put your final answer in <answer> tags.
```
This is lower overhead than JSON schema enforcement and works well for Claude's architecture.

### When It Helps vs Hurts
- **Helps**: Any machine-consumed output; pipeline stages requiring parsing; tool calling; classification
- **Hurts**: Creative/open-ended generation where rigid structure constrains quality; tasks where the model needs flexibility in output organization
- **Important**: For Claude 4.6, prefilled responses are deprecated — use structured outputs or XML tags instead

### Applicability to Claude Code / CLAUDE.md
- CLAUDE.md file format conventions (log entries, task entries) are effectively structured output specifications
- XML tags are the recommended approach for Claude — our CLAUDE.md could benefit from using XML-tagged sections
- For tool results that need parsing, use `strict: true` tool calling
- For human-readable structured output (like research logs), instruction-based formatting with examples is sufficient

---

## 7. Role and Persona Prompting

### How It Works
Assigning a role or persona to the model ("You are an expert...") to influence its behavior, tone, and output quality.

### Evidence Assessment

**The evidence is decidedly mixed.** Research shows:

**When personas help:**
- Open-ended tasks (creative writing, brainstorming) — consistent positive effect
- Domain-matched roles — when the role aligns with the task domain
- Tone and style control — effective for adjusting communication style
- Claude's own docs recommend it: "Setting a role in the system prompt focuses Claude's behavior and tone. Even a single sentence makes a difference."

**When personas don't help or hurt:**
- Accuracy-based tasks (classification, factual Q&A) — no significant boost to factual accuracy
- Irrelevant personas — **negative** effect across all models tested
- Newer/stronger models — diminishing returns from basic persona definitions
- An LLM can "sound" like a legal expert but still confidently misstate a law

**Key insight**: Personas are a **formatting and tone** tool, not a **factual accuracy** tool. The common pattern of prepending "You are an expert in X" is partially cargo cult for accuracy improvements, but genuinely useful for tone and scope narrowing.

### Concrete Recommendations
```
# Effective (specific, domain-matched, brief)
"You are a helpful coding assistant specializing in Python."

# Less effective (generic, overly elaborate)
"You are a world-renowned expert programmer with 30 years of
experience who has written bestselling books on software..."

# Counterproductive (irrelevant domain)
"You are a master chef." (for a coding task)
```

### Applicability to Claude Code / CLAUDE.md
- Use a brief, specific role statement at the top of CLAUDE.md
- Focus role on the actual task domain (research, coding, etc.)
- Don't expect the role to substitute for clear instructions
- Add explicit uncertainty handling: "If you are uncertain, say so" — this is more effective than role-based authority claims

---

## 8. Negative Prompting and Guardrails

### How It Works
Using "Do NOT..." instructions, prohibitions, and boundary-setting to prevent specific behaviors.

### The Pink Elephant Problem

LLMs suffer from a documented "Pink Elephant Problem" — when instructed not to mention something, they often bring it up. This has both psychological and architectural causes:

**Architectural vulnerability:**
1. Adding "not" does not eliminate a concept from the embedding space — the concept persists alongside the negation token
2. Attention mechanisms use weighted averages and lack direct subtraction capability
3. The more a concept is discussed (even negatively), the higher the probability of related tokens appearing — **negative instructions prime the very behavior they prohibit**

**Empirical findings:**
- InstructGPT performs **worse** with negative prompts as models scale
- Anthropic's own documentation explicitly recommends: "Tell Claude what to DO instead of what NOT to do"
- Claude 4.6 models respond worse to aggressive negative language ("CRITICAL!", "YOU MUST", "NEVER EVER") than to calm, direct positive instructions

### When Negative Instructions Work
- **Ethical/safety boundaries** in system prompts (but should be combined with positive alternatives)
- **When paired with positive alternatives**: "Do not use markdown. Instead, write in flowing prose paragraphs."
- **Structured constraints**: Using schema enforcement or grammar-guided generation rather than instruction-based avoidance

### Concrete Example of the Problem and Solution
```
# Counterproductive (primes the behavior)
"Do NOT use markdown. Do NOT create bullet lists.
NEVER use bold or italic text."

# Effective (positive framing with XML structuring)
"<formatting>
Write in clear, flowing prose using complete paragraphs and
sentences. Use standard paragraph breaks for organization.
Reserve markdown primarily for inline code and code blocks.
</formatting>"
```

### Applicability to Claude Code / CLAUDE.md
- **Audit our CLAUDE.md for negative instructions** and reframe as positive directives
- When prohibitions are genuinely needed, pair them with explicit alternatives
- Remove aggressive emphasis language (CRITICAL, MUST, NEVER) — these hurt performance on Claude 4.6
- For tool-use boundaries, use allowlists rather than blocklists where possible

**Specific concern with our current CLAUDE.md**: The file contains multiple "Do NOT" instructions and emphasis patterns ("NEVER", "MUST"). Based on Anthropic's own guidance, these should be reframed as calm, positive directives for optimal performance on Claude 4.6.

---

## 9. Prompt Chaining and Decomposition

### How It Works
Breaking complex tasks into sequential steps, where each step's output feeds the next. Three main patterns:

1. **Sequential chaining**: Linear pipeline of prompt stages (A -> B -> C)
2. **Self-correction chain**: Generate draft -> Review against criteria -> Refine
3. **Decomposed prompting**: Break main prompt into smaller sub-prompts, each handling one aspect

### Current State (2025-2026)

**Claude 4.6 handles most chaining internally.** With adaptive thinking and subagent orchestration, Claude handles most multi-step reasoning without explicit external chaining. Explicit chaining is still useful when you need to:
- Inspect intermediate outputs
- Enforce a specific pipeline structure
- Log/evaluate/branch at specific points

**Self-correction is the most impactful chaining pattern.** Generate -> Review -> Refine, with each step as a separate API call. This is the pattern used in Claude Code's agent harness and is recommended by Anthropic for quality improvement.

**Decomposition improves quality for multi-instruction tasks.** When a single prompt combines extraction + transformation + analysis + visualization, decomposition into explicit steps improves output quality and reliability. Each sub-prompt focuses the model's attention.

### Evidence
- Multi-step prompt chains improve accuracy for mathematical reasoning, symbolic tasks, and compositional challenges
- Decomposition enhances explainability — each step is traceable
- Prompt chaining reduces hallucination by constraining each step's scope

### When It Helps vs Hurts
- **Helps**: Complex multi-part tasks; tasks requiring different capabilities at each stage; production pipelines requiring auditability
- **Hurts**: Simple tasks where overhead exceeds benefit; tasks where global context across steps is critical (information loss at each chain boundary)
- **Important trade-off**: Each chain step has an information bottleneck — only what's passed forward is available

### Applicability to Claude Code / CLAUDE.md
- Our CLAUDE.md already uses a form of chaining: Phase 1 -> Phase 2 -> Phase 3 with gates
- The revision cycle (draft -> check depth checklist -> fill gaps -> update) is the self-correction chain pattern
- Sub-agent prompts in our template are a decomposition pattern
- For long research spikes, context window boundaries force natural chain boundaries — external memory files bridge them

---

## 10. Claude-Specific Prompting

### What Makes Claude Different

**XML affinity.** Claude was specifically trained to recognize and process XML-structured content. This is a genuine architectural advantage over Markdown or plain text structuring for Claude specifically (though other models may prefer different formats).

**Calm instructions outperform aggressive ones.** This is explicitly documented by Anthropic: "Aggressive language actively hurts newer Claude models." Where older models needed "CRITICAL: You MUST...", Claude 4.6 performs better with "Use this tool when..."

**Adaptive thinking.** Claude 4.6's adaptive thinking mode (`thinking: {type: "adaptive"}`) is unique — the model self-calibrates reasoning depth based on query complexity. In internal evaluations, it outperforms fixed thinking budgets.

**Context awareness.** Claude 4.6 can track its remaining context window, enabling self-managed context strategies. This pairs with the memory tool for seamless context transitions.

**Subagent orchestration.** Claude 4.6 natively recognizes when tasks benefit from delegation and spawns subagents without explicit instruction. It may over-use this capability and needs guidance about when direct approaches are faster.

**No prefill.** Starting with Claude 4.6, prefilled assistant responses are no longer supported. Use structured outputs, XML tags, or direct instructions instead.

### Claude vs GPT vs Gemini: Key Prompting Differences

| Aspect | Claude | GPT | Gemini |
|--------|--------|-----|--------|
| Structuring | XML tags preferred | Markdown/JSON | Flexible |
| Emphasis | Calm, direct | Moderate emphasis OK | Varies |
| Few-shot | `<example>` tags | Inline examples | Inline examples |
| CoT | Adaptive thinking | Manual or o1/o3 reasoning | Gemini Thinking |
| Output control | XML tags + structured outputs | Structured outputs + JSON mode | Structured output |
| Role prompts | Brief, specific | Elaborate OK | Brief preferred |

### Constitutional AI Implications
Claude's training via Constitutional AI (RLHF + CAI) means it has stronger built-in ethical reasoning than pure RLHF models. This manifests as:
- Better appropriate refusal calibration (fewer over-refusals on 4.6)
- Stronger instruction-following within ethical bounds
- More consistent behavior with calm instructions (aggressive language conflicts with the constitution's tone)

### Specific Claude Code Optimizations

**System prompt management:**
```xml
<instructions>
Your role is [brief, specific role].
</instructions>

<context>
[Background information, project conventions]
</context>

<rules>
[Behavioral rules as positive directives]
</rules>

<output_format>
[Expected format with example]
</output_format>
```

**Context compaction management:**
```
Your context window will be automatically compacted as it
approaches its limit. Save current progress and state to
memory before the context window refreshes. Do not stop
tasks early due to token budget concerns.
```

**Agentic self-verification:**
```
Before completing any task, verify your work against the
acceptance criteria. Read relevant files before making claims
about code. If uncertain, investigate rather than speculate.
```

---

## Cross-Cutting Themes

### Theme 1: The Shift to Context Engineering
Prompt engineering is now understood as a subset of **context engineering** — the holistic design of all information provided to the model, including system prompts, retrieved documents, tool outputs, conversation history, and structured metadata. Gartner predicts context engineering will become foundational enterprise AI infrastructure by 2027.

### Theme 2: Less is More (on Modern Models)
Across nearly every technique, the pattern is: less aggressive prompting, shorter instructions, calmer tone, and more trust in the model's native capabilities produces better results on frontier models. The 2023-era patterns of elaborate prompts with aggressive emphasis are now actively counterproductive.

### Theme 3: Structure Over Volume
The quality of prompt structure matters more than the quantity of instructions. XML tags, clear section demarcation, and explicit success criteria outperform verbose explanations. This aligns with the empirical finding that prompts degrade after ~2,000 tokens.

### Theme 4: Verify, Don't Trust
Despite improvements, LLM outputs (including reasoning chains) should not be trusted at face value. Self-verification prompts, structured output enforcement, and external validation are all more reliable than hoping the model follows instructions perfectly.

### Theme 5: Technique Interactions
These techniques are not independent — they interact:
- Role prompting + XML structuring works better than either alone
- CoT + few-shot can hurt on modern models (zero-shot CoT often better)
- Negative instructions + aggressive tone is the worst combination
- Structured output + self-verification is the most reliable combination

---

## Recommendations for Our CLAUDE.md

Based on this research, specific actionable changes for our CLAUDE.md:

1. **Reframe negative instructions as positive directives.** Our CLAUDE.md contains multiple "Do NOT" and "NEVER" patterns. These should be converted to calm positive instructions describing desired behavior.

2. **Remove aggressive emphasis.** Replace "CRITICAL", "MUST", "IMPORTANT" with unmarked, calm statements. Claude 4.6 responds worse to aggressive language.

3. **Add XML structuring.** Wrap major sections in XML tags (`<instructions>`, `<rules>`, `<format>`) for better parsing.

4. **Trim length.** Target under 200 lines per file. Move detailed reference material to separate files that can be loaded on demand.

5. **Front-load critical rules.** Place the most important behavioral rules first, as attention weight decreases with position.

6. **Add self-verification triggers.** Include explicit self-check instructions: "Before marking a task complete, verify against the depth checklist."

7. **Simplify CoT directives.** Replace prescriptive step-by-step instructions with general thinking encouragement: "Think thoroughly about complex problems" rather than "Step 1: ... Step 2: ..."

8. **Use examples for format specification.** The file format templates in CLAUDE.md (log entries, task entries, research summaries) are exactly the right use of few-shot — keep these.

9. **Brief, specific role statement.** Add a one-line role: "You are a research assistant conducting in-depth technical investigations."

10. **Context compaction awareness.** Add instructions about context management and external memory usage for long sessions.

---

## Sources

### Primary Sources (saved to docs/)
- `docs/anthropic-prompting-best-practices-2026.md` — Anthropic's official Claude 4.6 prompting guide
- `docs/anthropic-cot-faithfulness-research-2025.md` — Anthropic's CoT faithfulness research
- `docs/instruction-hierarchy-paper-2024.md` — Instruction hierarchy for LLM prioritization
- `docs/dspy-miprov2-optimizer-2025.md` — DSPy, OPRO, and automated prompt optimization
- `docs/context-engineering-paradigm-shift-2025.md` — Context engineering paradigm shift
- `docs/negative-instructions-pink-elephant-2025.md` — Pink elephant problem and negative prompting
- `docs/role-persona-prompting-effectiveness-2025.md` — Role/persona prompting evidence
- `docs/prompt-length-quality-tradeoff-2025.md` — Prompt length vs quality
- `docs/structured-output-techniques-2025.md` — Structured output approaches
- `docs/cot-zero-shot-vs-few-shot-2025.md` — CoT zero-shot vs few-shot comparison
- `docs/claude-code-claude-md-best-practices-2026.md` — CLAUDE.md community patterns

### Key Papers Referenced
- Wallace et al. "The Instruction Hierarchy" (2024, ICLR 2025)
- Chen et al. "Reasoning Models Don't Always Say What They Think" (Anthropic, May 2025)
- "Revisiting Chain-of-Thought Prompting: Zero-shot Can Be Stronger than Few-shot" (2025)
- "Suppressing Pink Elephants with Direct Principle Feedback" (2024)
- "Large Language Models as Optimizers" (Google DeepMind, OPRO)
- "Connecting Large Language Models with Evolutionary Algorithms" (EvoPrompt)
- "FaithCoT-Bench" (2025) — CoT faithfulness benchmark
- "Counterfactual Simulation Training for CoT Faithfulness" (2025)

### Key URLs
- Anthropic prompting docs: https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices
- DSPy framework: https://dspy.ai/
- OPRO: https://github.com/google-deepmind/opro
- Claude Code best practices: https://code.claude.com/docs/en/best-practices
