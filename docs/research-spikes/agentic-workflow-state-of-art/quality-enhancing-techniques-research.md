# Quality-Enhancing Techniques for Agentic AI

## Overview

This report surveys ten specific techniques and patterns that measurably improve the accuracy, correctness, and thoroughness of agentic AI output. Each technique is analyzed for its mechanism, quantitative evidence, cost/latency tradeoffs, failure modes, and applicability to Claude Code's workflow. The techniques range from simple sampling strategies to sophisticated multi-agent architectures, but a clear hierarchy of effectiveness emerges from the evidence.

**The single most important finding**: External verification signals (test execution, compiler feedback, linter output) are the most reliable quality enhancers. Pure self-reflection without external grounding is unreliable and can degrade performance. The techniques that work best combine generation with concrete, executable verification.

---

## 1. Self-Verification and Self-Consistency

### How It Works

**Self-consistency** (Wang et al., 2022) replaces greedy single-shot generation with a sample-then-vote strategy:
1. Generate N reasoning paths using temperature > 0
2. Extract the final answer from each path
3. Take majority vote across answers
4. Return the most common answer

**Universal Self-Consistency** (Chen et al., 2024, Google DeepMind) extends this to free-form outputs by using the LLM itself as the aggregator — instead of exact matching, compile all N responses into a prompt and ask the model to select the most consistent one.

**Generate-then-verify** patterns in coding agents combine generation with execution: generate code, run tests/linter, use results to verify or reject.

### Quantitative Evidence

Self-consistency improvements over standard chain-of-thought (Wang et al., 2022):
- GSM8K: **+17.9%** absolute
- SVAMP: **+11.0%**
- AQuA: **+12.2%**
- StrategyQA: **+6.4%**
- ARC-challenge: **+3.9%**

These are large, consistent gains across diverse reasoning benchmarks with no additional training required.

### Cost/Latency Tradeoffs

- Linear cost increase: N forward passes instead of 1
- Most benefit comes from relatively small N (5-40 samples)
- Diminishing returns as N grows — the marginal accuracy gain of the 41st sample is minimal
- Can be parallelized to trade cost for latency

### Failure Modes

- **When all paths are wrong**: If the model systematically produces incorrect answers, majority voting amplifies the error rather than correcting it
- **Free-form answer aggregation**: Standard self-consistency requires extractable, comparable answers. USC addresses this but adds another LLM call
- **Cost explosion for long outputs**: Sampling N complete code solutions is expensive; more practical to verify post-hoc than to sample massively

### Applicability to Claude Code

**High applicability.** Two concrete implementations:
1. **Execution-based self-verification**: Generate code → run tests → accept or retry. This is already Claude Code's core loop ("write code → run tests/CI → automatically fix errors") and is the strongest form of self-verification because the feedback is concrete and unambiguous.
2. **Plan verification**: Generate a plan, then separately verify it before execution — analogous to the "think" tool pattern.

The key insight from Huang et al. (2024) is critical here: self-verification only works reliably when grounded in external signals (test results, compiler errors, linter output). Pure "does this look right?" self-checks without execution are unreliable.

---

## 2. Reflection and Iterative Refinement

### How It Works

**Reflexion** (Shinn et al., 2023, NeurIPS): Agents reflect on task feedback after each trial, storing reflective text in an episodic memory buffer. On subsequent trials, the agent reads its prior reflections to avoid repeating mistakes. The key mechanism is *verbal reinforcement learning* — learning from linguistically-expressed experience rather than gradient updates.

**Self-Refine** (Madaan et al., 2023, NeurIPS): A single LLM generates output, provides structured feedback on it (identifying specific problems with actionable suggestions), then revises based on that feedback. The FEEDBACK → REFINE loop repeats up to 4 iterations.

### Quantitative Evidence

**Reflexion**:
- HumanEval: 91% pass@1 (vs GPT-4's 80% at the time)
- 8% absolute boost over episodic memory baselines
- Improvements across decision-making, coding, and reasoning tasks

**Self-Refine**:
- ~20% absolute average improvement across 7 diverse tasks
- Range: 5% to 40% depending on task
- Dialogue response generation: 49.2% preference improvement (25.4% → 74.6%)
- Math reasoning: only 0.2% improvement for GPT-4 (modest)
- With oracle feedback: 4.8% improvement for GPT-3.5

### Optimal Iterations and Diminishing Returns

- Self-Refine: most improvement in first 1-2 iterations, maximum 4 iterations
- TDFlow: diminishing returns after 5-10 iterations
- General finding: the first reflection/revision captures most of the value; subsequent iterations show rapidly diminishing returns
- Task dependence: creative/open-ended tasks benefit from more iterations; well-defined tasks (math, code) saturate faster

### When Reflection Hurts

**This is the most critical finding in this report.** Huang et al. (2024, ICLR) demonstrated:

- **Intrinsic self-correction** (model corrects based solely on its own judgment, without external feedback) **degrades performance** across all models and all benchmarks tested
- LLMs cannot reliably distinguish their own correct answers from incorrect ones
- Prompting models to find mistakes produces high false positive rates (flagging correct answers as wrong)
- Performance drops are consistent, not sporadic

The implication: reflection is only beneficial when grounded in **external feedback signals**. These include:
- Test execution results
- Compiler/linter errors
- Retrieved factual information
- Human feedback
- Tool output

Without such grounding, reflection is self-reinforcing confirmation bias at best and destructive at worst.

### Applicability to Claude Code

**High applicability with a critical caveat.** Claude Code's self-verification loops (write code → run tests → fix errors) are highly effective precisely because they use external feedback (test/CI results). The reflection is grounded.

What to avoid: asking Claude to "review its own code" without running it. This is intrinsic self-correction and the evidence says it is unreliable. Instead:
- Always run code after generating it
- Use linter/type-checker output as feedback
- Parse error messages to inform retry attempts
- Use explicit checklists (Constitutional AI pattern) rather than open-ended self-review

---

## 3. Critic/Judge Patterns

### How It Works

**LLM-as-Judge** (Zheng et al., 2023, NeurIPS): Uses a strong LLM to evaluate output quality, replacing or supplementing human evaluation. The judge model receives the output and a rubric, then provides scores or preferences.

**Constitutional AI** (Bai et al., 2022, Anthropic): Defines explicit principles (a "constitution") and has the model critique its own output against those principles, then revise. The key innovation is that self-critique is grounded in *enumerable, specific criteria* rather than vague quality judgments.

**Reward model guidance**: A separately trained model scores outputs, guiding selection in best-of-N or providing training signal for RL.

### Quantitative Evidence

- GPT-4 as judge: **>80% agreement** with human preferences (matching human-human agreement)
- Constitutional AI: produces harmless but non-evasive responses without human harm labels
- Think tool (Anthropic, 2025): 54% relative improvement in airline policy compliance (0.370 → 0.570 pass rate)

### Known Biases

LLM-as-judge exhibits systematic biases (Zheng et al., 2023):
1. **Position bias**: Favors responses based on presentation order
2. **Verbosity bias**: Prefers longer responses regardless of quality
3. **Self-enhancement bias**: Favors outputs from same model family
4. **Limited reasoning**: Struggles with complex analytical evaluation

Mitigations: position swapping + averaging, reference answers, multi-judge panels, structured rubrics.

### Reward Model Overoptimization

Gao et al. (2023) demonstrated scaling laws for reward hacking:
- Over-optimizing against a proxy reward model degrades true performance (Goodhart's law)
- Effect is predictable and scales with reward model size
- Best-of-N sampling is more robust to overoptimization than RL
- Mitigation: reward model ensembles, conservative optimization bounds

### Applicability to Claude Code

**Moderate applicability.** Two practical patterns:

1. **Principle-based self-critique** (Constitutional AI adapted): Define explicit quality criteria in CLAUDE.md (e.g., "code must handle edge cases," "functions must have docstrings," "changes must be backward compatible") and have Claude evaluate its output against these specific criteria. This is more reliable than open-ended self-review because the criteria are concrete and enumerable.

2. **The think tool pattern**: Using a dedicated thinking step to evaluate intermediate results during multi-step agentic execution. Already shown to improve policy compliance by 54% in Anthropic's benchmarks.

The main limitation is that judge quality is bounded by the judge model's capability — a model cannot reliably catch errors it would also make as a generator. External verification (tests, linters) remains superior to model-based judgment for factual correctness.

---

## 4. Test-Driven Development in Agents

### How It Works

The agent writes or receives tests first, then generates code to satisfy those tests, using test execution results as the feedback signal for iterative refinement. This inverts the typical generate → test flow into a test → generate → verify → refine loop.

### Quantitative Evidence

**TDFlow** (2025) provides the strongest evidence:

| System | SWE-bench Lite | Cost/Issue |
|--------|---------------|------------|
| SWE-Agent | 49.0% | $0.89 |
| Agentless | 61.0% | $0.53 |
| **TDFlow** | **88.8%** | **$1.51** |

On SWE-bench Verified: **94.3%** pass rate with human-written tests.

The critical finding: when test quality is high (Bad Test Rate = 0), both human and LLM-generated tests achieve ~93-94% resolution rates. **Test quality is the bottleneck, not code generation capability.**

**Cognition's Devin**: In TDD setting on SWE-bench, pass rate increased to 23% (though with access to ground truth test patches, making it incomparable to standard results).

**LLMLOOP**: Automates refinement through five iterative loops — compilation errors → static analysis → test failures → test quality → mutation analysis.

### Why TDD Is So Effective for Agents

1. **Concrete feedback signal**: Test pass/fail is unambiguous — the strongest form of external verification
2. **Error localization**: Test failures pinpoint what's wrong, guiding targeted fixes
3. **Scope constraint**: Tests define the exact requirements, preventing scope drift
4. **Convergence guarantee**: The agent knows when it's done (all tests pass)
5. **Prevents hallucination**: The agent can't claim success if tests fail

### Failure Modes

- **Bad tests**: If tests are incorrect, the agent converges on wrong behavior (TDFlow: 68% with LLM-generated tests vs 94.3% with human tests)
- **Overtesting**: Overly specific tests can constrain the solution space unnecessarily
- **Test hacking**: Agent modifies tests rather than code to pass (TDFlow mitigates by preventing patches from modifying test folders)
- **Infrastructure requirements**: Requires executable test environment, which isn't always available

### Applicability to Claude Code

**Highest applicability of all techniques surveyed.** This is the single most impactful technique for Claude Code quality improvement.

Claude Code already implements the basic loop (write code → run tests → fix errors). To maximize quality:
1. **Write tests first** when possible — ask Claude to generate tests before implementation
2. **Run tests after every change** — not just at the end
3. **Parse test failures for specific error information** — feed error messages back as context
4. **Protect test integrity** — don't let the agent modify test files to make them pass
5. **Use the full verification stack** — not just unit tests but linter, type checker, build system

The TDFlow evidence suggests this single technique can close most of the gap between current agent performance and human-level quality.

---

## 5. Chain-of-Thought and Extended Thinking

### How It Works

**Chain-of-thought (CoT)** prompting (Wei et al., 2022) elicits step-by-step reasoning before the final answer. The model "shows its work," which enables more complex multi-step reasoning.

**Extended thinking** (o1/o3 models, Claude thinking modes) allocates additional compute tokens for internal reasoning before generating the response. Thinking tokens are generated but may not be shown to the user.

**Think tool** (Anthropic, 2025): A tool call that provides a structured thinking space mid-execution during agentic tasks. Unlike extended thinking (which happens before the first response), the think tool activates at specific decision points.

### Quantitative Evidence

**Reasoning model improvements (o3 vs o1)**:
- AIME 2024: 74.3% → 91.6% (**+17.3 points**)
- GPQA Diamond: 78% → 83.3% (**+5.3 points**)
- ARC-AGI (high compute): 88% (beyond 85% human baseline)
- 20% fewer major errors on real-world tasks

**Think tool (Anthropic)**:
- Airline policy compliance: 0.370 → 0.570 (**54% relative improvement**)
- Retail domain: 0.783 → 0.812

**Claude thinking budget keywords** (escalating compute allocation):
- "think" < "think hard" < "think harder" < "ultrathink"

### When Extended Thinking Helps vs Hurts

**Helps**:
- Complex multi-step reasoning (math, logic, analysis)
- Policy-heavy environments with many rules to consider
- Sequential decisions where each step builds on previous ones
- Hard problems where surface-level reasoning gives wrong answers

**Hurts (overthinking)**:
- Simple tasks where the first-pass answer is usually correct
- Commonsense reasoning and basic mathematics
- Short-horizon tasks that don't benefit from extended deliberation
- Tasks where conciseness is paramount

**Evidence from test-time compute scaling research** (2024):
- Beam search exhibits *inverse* scaling — performance degrades as beam size increases for non-reasoning models
- Majority voting shows consistent positive scaling for hard problems
- Compute-optimal allocation outperforms blanket compute increases
- Optimal strategy depends on model type and problem difficulty

### Applicability to Claude Code

**High applicability.** Two implementation patterns:

1. **Adaptive thinking budget**: Use "think" for simple tasks, "ultrathink" for complex architectural decisions, code review, or debugging hard problems. Don't waste thinking budget on straightforward file edits.

2. **Think tool for mid-execution deliberation**: When Claude Code is in a long agentic loop (multiple tool calls), strategic thinking pauses before critical decisions improve quality — especially for policy compliance and multi-step consistency.

Current Claude Code practice already leverages this through the CLAUDE.md instruction to "plan before implementing" and the explicit thinking budget keywords.

---

## 6. Structured Output and Format Enforcement

### How It Works

**Constrained decoding**: During token generation, mask out tokens that would violate the target format (JSON schema, grammar rules). Guarantees syntactically valid structured output.

**Format-restricting instructions**: Prompt the model to output in a specific format without token-level enforcement.

**NL-to-format conversion**: Generate in natural language first, then convert to structured format in a second pass.

### Quantitative Evidence

The evidence on quality is **contradictory** depending on the task type:

**Reasoning tasks — structured output HURTS** (Tam et al., 2024, EMNLP):
- GSM8K: Claude-3-haiku dropped from 86.51% to **23.44%** with JSON schema constraints
- GPT-3.5-turbo: 76.6% → 49.25% with JSON
- Last Letter Concatenation: LLaMA-3-8B 70.07% → 28% with JSON

**Classification tasks — structured output HELPS**:
- DDXPlus medical diagnosis: +18.77 points with JSON

**Constrained decoding — can improve quality** (JSONSchemaBench, 2025):
- Guidance framework: ~3% accuracy gains on GSM8K with constrained decoding
- Constrained decoding accelerates generation by ~50% (fewer wasted tokens)
- But this contradicts Tam et al. — likely depends on how constraints interact with reasoning

**Critical finding**: The worst degradation comes from **schema constraints during reasoning**. Removing the explicit schema (but keeping JSON format request) preserved performance — the ordering constraint on keys disrupts reasoning flow.

### Practical Resolution

The two-step approach resolves the tension:
1. Let the model reason freely in natural language
2. Extract/convert to structured format afterwards

This preserves reasoning quality while delivering structured output. Alternatively, use structured output for classification/extraction tasks where it genuinely helps.

### Applicability to Claude Code

**Moderate applicability, with important caveats.** Claude Code primarily generates code (which has its own structure enforcement via compilers/linters) rather than JSON/XML. The key takeaways:

1. **Don't constrain reasoning format**: When Claude needs to think through a problem, let it reason freely. Don't force intermediate reasoning into structured templates.
2. **Structured output for tool calls**: Tool use naturally requires structured output (function names, parameters). This is appropriate and doesn't typically impair quality because the "reasoning" happens before the tool call.
3. **Post-hoc structuring**: If structured data is needed (e.g., task lists, plans), have Claude reason first, then format.

---

## 7. Retrieval-Augmented Generation (RAG) for Grounding

### How It Works

Before generating a response, retrieve relevant context from an external knowledge base (documents, codebase, web). The retrieved context is included in the prompt, grounding the generation in actual facts/code rather than parametric memory alone.

### Quantitative Evidence

- MEGA-RAG: **>40% reduction** in hallucination rates (public health domain)
- Stanford study (2024): **96% reduction** in hallucinations (RAG + RLHF + guardrails combined)
- General finding: RAG reduces hallucinations by **42-68%** across domains
- Medical AI: up to **89% factual accuracy** with trusted source retrieval

### Chunking Strategy Impact

- Page-level chunking: 0.648 accuracy (NVIDIA 2024 benchmark winner)
- Factoid queries: optimal at 256-512 tokens per chunk
- Analytical queries: need 1024+ tokens per chunk
- Adaptive segmentation (discourse-boundary aligned): best overall approach
- Fixed-length chunks can split concepts, reducing precision

### When RAG Helps vs Hurts

**Helps**:
- Knowledge-intensive tasks requiring specific facts
- Code generation needing awareness of existing codebase
- Domain-specific applications with clear authoritative sources
- Any task where parametric memory is likely stale or incomplete

**Hurts**:
- When retrieved context is noisy, irrelevant, or contradictory
- Short, focused documents where chunking fragments meaning
- Pure reasoning tasks where all information is already in the prompt
- When high recall retrieval includes too much irrelevant context

### Applicability to Claude Code

**Very high applicability — this is foundational to how Claude Code works.** Claude Code's tool use (reading files, searching codebases, exploring directory structures) is essentially RAG applied to a codebase. The quality of this "retrieval" directly determines output quality:

1. **Codebase search quality**: Claude Code's ability to find relevant files, understand project structure, and locate existing patterns prevents hallucinated APIs and incompatible code
2. **File reading strategy**: Reading the right files (and enough of them) before generating code is the coding equivalent of good chunk retrieval
3. **Context window management**: With large codebases, what to include in context is a retrieval quality decision

The implication: investing in better codebase search and context selection is a high-leverage quality improvement. This is why CLAUDE.md instructions like "always read the file before editing" exist — they enforce good retrieval discipline.

---

## 8. Ensemble and Voting Methods

### How It Works

Run multiple generation attempts and select or combine the best result. Key variants:

1. **Majority voting** (self-consistency): Take the most common answer across N samples
2. **Best-of-N with reward model**: Score each candidate, select highest-scoring
3. **Tournament selection**: Pairwise comparisons to find the best candidate
4. **Execution-based filtering**: Run all candidates through tests, keep those that pass
5. **Clustering + selection** (AlphaCode): Group equivalent outputs, select from largest clusters

### Quantitative Evidence

- **Self-consistency**: +6-18% absolute across reasoning benchmarks (see Section 1)
- **RISE (recursive introspection)**: +8.2% (LLaMA3-8B), +17.7% (LLaMA2-7B)
- **Pairwise RM**: +6.7% on MATH-500, +3.9% on Olympiad Bench
- **DARE**: +25.3% relative on AIME 2024
- **AlphaCode**: Millions of samples filtered to 10 → median competitive programmer performance

### Cost/Quality Tradeoff

| Method | Cost Factor | Quality Gain | Notes |
|--------|-------------|-------------|-------|
| Majority voting (N=5) | 5x | +6-12% | Parallelizable |
| Majority voting (N=40) | 40x | +12-18% | Diminishing returns |
| Best-of-N + reward model | Nx + scoring | +6-25% | Risk of reward hacking |
| Execution filtering | Nx + test runs | Variable | Most reliable for code |
| AlphaCode (millions) | 1000x+ | Extreme | Not practical for real-time |

### Failure Modes

- **Reward hacking**: Over-optimizing a proxy reward degrades true quality (Gao et al., 2023)
- **Systematic errors**: If all N samples share the same systematic error, voting amplifies it
- **Cost scaling**: For long outputs (full code files), N samples is very expensive
- **Selection quality**: The selection mechanism must be better than random; bad reward models can select worse outputs

### Applicability to Claude Code

**Moderate applicability.** Full parallel ensemble is expensive for code generation, but a practical variant exists:

1. **Sequential informed retry**: Generate → test → if fail, generate again with error context → test → repeat. This is a cost-effective variant of best-of-N where each subsequent attempt is informed by previous failures (not independent).
2. **Multiple solution approaches**: For complex problems, ask Claude to generate 2-3 different approaches, then evaluate which is best (manually or via tests).
3. **The execution-based filter is already in play**: Claude Code's test loop is effectively best-of-N with N growing incrementally until tests pass.

---

## 9. Error Recovery and Retry Patterns

### How It Works

When an agent encounters a failure (tool error, test failure, compilation error, API timeout), it must detect, classify, and recover from the error rather than failing outright or retrying blindly.

**Core patterns**:
1. **Error parsing**: Extract structured information from error messages
2. **Error classification**: Route to appropriate recovery strategy based on error type
3. **Targeted retry**: Retry with specific context about what went wrong
4. **Fallback strategies**: Try alternative approaches when primary fails
5. **Progressive refinement**: Each retry attempt informed by accumulated error history
6. **Iteration limits**: Hard bounds to prevent infinite loops

### Implementation in Leading Agents

**SWE-Agent / OpenHands**:
- Maximum 100 iterations/LLM calls per instance
- Run code → parse errors → add debug statements → iterate
- Full inner-loop: make changes → test → construct PRs

**TDFlow**:
- Per-test debugging with restricted debugger
- Diagnostic reports identifying root causes (not symptoms)
- Failed patches + test outputs accumulate in context
- Separate agents for different error recovery tasks (Revise Patch vs Debug One)

**LLMLOOP**:
- Five specialized feedback loops: compilation → static analysis → test failures → test quality → mutation analysis
- Multi-granularity error feedback (binary success vs detailed error reasons)

### Critical Design Principles

1. **Error-specific recovery** outperforms generic retry: parse the error type, route to appropriate handler
2. **Context accumulation**: Each retry should see the history of previous failures (prevents repeating the same mistake)
3. **Root cause identification**: Debug reports identifying *why* something fails are more valuable than simply reporting *that* it fails
4. **Iteration limits are essential**: Without them, agents enter infinite loops. Practical limits: 5-10 iterations for diminishing returns, 100 for hard ceiling
5. **Don't retry infrastructure failures the same way**: Transient errors (timeouts, rate limits) need backoff; semantic errors (wrong approach) need reformulation

### Failure Modes

- **Infinite retry loops**: Without limits, agents burn compute retrying unsolvable problems
- **Context pollution**: Accumulating too much error history can overflow context and degrade performance
- **Systematic blindness**: If the agent consistently makes the same type of error, retries won't help — the approach needs to change
- **State corruption**: Unlike stateless services, agents maintain internal state that can become inconsistent after errors

### Applicability to Claude Code

**Very high applicability.** Claude Code's core workflow is already built around error recovery:

1. **Parse compiler/test errors**: Feed specific error messages (not just "it failed") back into the next attempt
2. **Limit iteration**: Set explicit maximum retries to prevent runaway loops
3. **Error classification**: Different handling for syntax errors (quick fix) vs logical errors (need rethinking) vs infrastructure errors (retry with backoff)
4. **Context management during retries**: Include relevant error history but summarize/truncate to prevent context overflow
5. **Know when to stop**: If the same error recurs after 3-4 attempts, escalate or try a fundamentally different approach

---

## 10. Decomposition and Divide-and-Conquer

### How It Works

Break complex tasks into simpler subtasks, solve each independently, then combine results. Key variants:

**Least-to-Most Prompting** (Zhou et al., 2023): Decompose into ordered subproblems where each builds on previous answers. Remarkable for generalization — models solve harder problems than seen in examples.

**Divide-and-Conquer (DaC) Prompting** (2024): Three-stage isolated process: decompose → resolve independently → merge. The "disentangled-sub-process" principle prevents error propagation by only passing final outputs between stages.

**Multi-agent decomposition** (TDFlow, CrewAI, etc.): Different agents handle different subtasks with specialized tools and context.

### Quantitative Evidence

**Least-to-Most** (Zhou et al., 2023):
- SCAN benchmark: **99% accuracy** with 14 exemplars (vs 16% for chain-of-thought)
- Strong generalization: solves harder problems than exemplars demonstrate

**DaC Prompting** (2024):
| Task | Method | GPT-3.5 | GPT-4 |
|------|--------|---------|-------|
| Multiplication | CoT | 64.26% | 76.10% |
| Multiplication | **DaC** | **75.55%** | **78.99%** |
| HaluEval (F1) | CoT | 46.85 | 71.05 |
| HaluEval (F1) | **DaC** | **74.84** | **76.92** |
| SciFact (F1) | CoT | 56.09 | 74.03 |
| SciFact (F1) | **DaC** | **76.88** | **81.11** |

**TDFlow (multi-agent decomposition)**:
- 88.8% on SWE-bench Lite (vs 49% for monolithic SWE-Agent)
- The decomposition into 4 specialized agents is a primary driver of the quality gain

### When Decomposition Helps

- Tasks requiring exhaustive handling of all components
- Problems with independent/parallel sub-tasks
- Long inputs needing thorough coverage
- Complex multi-step workflows where different steps require different skills
- When cognitive load of the full task exceeds single-model capacity

### When Decomposition Hurts

- **Coordination overhead**: Multiple models/prompts with O(km) coordination cost (k subtasks, m connectivity)
- **Decomposer-solver mismatch**: The decomposer doesn't track whether the solver can follow the decomposed chain
- **Over-decomposition**: Too-fine-grained tasks create excessive overhead and lose coherence
- **Sequential dependencies**: Tasks requiring dynamic programming or tightly coupled steps lose context between subtasks
- **Exploration tasks**: Planning/search tasks where the path isn't known in advance

### Applicability to Claude Code

**High applicability.** Claude Code already practices decomposition through:

1. **Phase-based planning**: Breaking work into phases with explicit gates (from CLAUDE.md's research methodology)
2. **Sub-agent delegation**: Using separate agent instances for specific research topics
3. **Function-level implementation**: Implementing one function/module at a time rather than entire systems at once

To improve:
- **Explicit decomposition before implementation**: Have Claude create a plan/task list before coding, not during
- **Independent subtask execution**: Where possible, decompose into tasks that don't require context from other tasks
- **Controlled context passing**: Only pass essential outputs between subtasks, not full histories (the DaC principle)
- **Avoid over-decomposition**: For simple tasks, decomposition adds overhead without benefit. Reserve it for genuinely complex problems.

---

## Cross-Cutting Analysis

### Technique Effectiveness Ranking

Based on the quantitative evidence, techniques can be ranked by their **reliability** (how consistently they improve quality) and **magnitude** (size of improvement):

| Rank | Technique | Reliability | Typical Improvement | Key Requirement |
|------|-----------|-------------|--------------------|----|
| 1 | Test-driven development | Very High | +27-40% absolute (TDFlow) | Executable test environment |
| 2 | Self-consistency/voting | High | +6-18% absolute | Multiple samples + aggregation |
| 3 | Task decomposition | High | +5-30% depending on task | Decomposable problem structure |
| 4 | RAG/context grounding | High | 42-68% hallucination reduction | Quality retrieval infrastructure |
| 5 | Extended thinking | High | +5-17% on hard problems | Thinking budget allocation |
| 6 | Error-specific recovery | High | Highly variable | Error parsing + routing |
| 7 | Reflection (with external feedback) | Moderate-High | ~20% average (Self-Refine) | External feedback signal |
| 8 | Critic/judge patterns | Moderate | 54% improvement (think tool) | Explicit evaluation criteria |
| 9 | Ensemble/best-of-N | Moderate | +6-25% | Compute budget |
| 10 | Structured output | Low-Moderate | Helps classification, hurts reasoning | Task-appropriate application |

### The External Feedback Principle

The single most consistent finding across all techniques: **quality improvement is proportional to the quality of the external feedback signal**.

- Test execution results > model self-evaluation
- Compiler errors > self-reported code quality
- Retrieved facts > parametric memory
- Explicit criteria > vague "improve this"

Techniques that rely on the model's own judgment (intrinsic self-correction, pure self-review without execution) are unreliable. Techniques that ground the model in external reality (test results, tool output, retrieved context) are consistently effective.

### The Three Pillars of Agentic Quality

From this analysis, three complementary strategies emerge as the foundation for high-quality agentic AI output:

1. **Generate with thinking**: Use appropriate thinking budget, plan before acting, decompose complex tasks (Sections 5, 10)
2. **Verify with execution**: Run tests, execute code, use tools to check work against reality (Sections 1, 4, 9)
3. **Refine with feedback**: Use external signals (not self-judgment) to guide iterative improvement (Sections 2, 3, 7)

All three are needed. Thinking without verification produces confident wrong answers. Verification without refinement wastes failed attempts. Refinement without thinking produces incremental patches rather than clean solutions.

### Cost-Effectiveness for Claude Code

For a coding assistant like Claude Code, the most cost-effective quality investments are:

1. **Always run tests/linter after changes** (free quality signal, highest reliability)
2. **Read relevant code before editing** (RAG for code — prevents hallucinated APIs)
3. **Parse and use error messages** (targeted error recovery, not blind retry)
4. **Plan before complex tasks** (thinking + decomposition, low cost)
5. **Use explicit checklists** (Constitutional AI principle — enumerate what "good" means)
6. **Adaptive thinking budget** (scale thinking to problem difficulty)
7. **Sequential informed retry** (practical best-of-N with error context)

These are ordered by cost-effectiveness: the first few are essentially free (they use external signals already available), while later items require additional compute.

---

## Key Papers and Sources

| Paper/Source | Year | Key Finding |
|---|---|---|
| Wang et al. — Self-Consistency | 2022 | Majority voting over CoT paths: +6-18% |
| Bai et al. — Constitutional AI | 2022 | Principle-grounded self-critique enables reliable self-improvement |
| Shinn et al. — Reflexion | 2023 | Verbal reflection + episodic memory: 91% HumanEval |
| Madaan et al. — Self-Refine | 2023 | Single-model feedback-refine loop: ~20% average improvement |
| Zheng et al. — LLM-as-Judge | 2023 | GPT-4 judge >80% agreement with humans |
| Huang et al. — LLMs Cannot Self-Correct | 2024 | Intrinsic self-correction degrades performance |
| Tam et al. — Let Me Speak Freely | 2024 | JSON constraints degrade reasoning by 25-63 points |
| Chen et al. — Universal Self-Consistency | 2024 | Extends voting to free-form outputs |
| DaC Prompting | 2024 | Isolated decompose-resolve-merge: +11-28% |
| Test-time compute scaling | 2024 | No universal best strategy; model and task dependent |
| TDFlow | 2025 | TDD agent: 88.8% SWE-bench Lite (vs 49% baseline) |
| Anthropic — Think Tool | 2025 | 54% relative improvement in policy compliance |
| JSONSchemaBench | 2025 | Constrained decoding can improve quality for structured tasks |
| Gao et al. — Reward Model Overoptimization | 2023 | Scaling laws for Goodhart's law in best-of-N |
| AlphaCode | 2022 | Massive sampling + execution filtering = competitive programming |
| Zhou et al. — Least-to-Most | 2023 | 99% SCAN accuracy (vs 16% CoT) with decomposition |
