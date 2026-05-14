# Agentic Architecture Patterns: State of the Art (Early 2026)

## Overview

This report surveys the eight major architectural patterns used in state-of-the-art agentic AI systems as of early 2026, focusing on patterns that produce high-quality, accurate, and thorough output. Each pattern is analyzed for its mechanism, effectiveness evidence, failure modes, comparisons to alternatives, real-world usage, and applicability to Claude Code.

The patterns are not mutually exclusive — the most effective production systems combine multiple patterns. The current evidence strongly suggests that **tool-augmented reasoning with verification loops** (patterns 7 and 8) produces the largest and most reliable quality gains in coding contexts, while **reflection/self-critique** (pattern 2) and **planning-first architectures** (pattern 3) provide significant but more context-dependent improvements.

---

## 1. ReAct (Reasoning + Acting)

### How It Works

ReAct (Yao et al., ICLR 2023) interleaves reasoning traces with tool actions in a Thought-Action-Observation loop:

1. **Thought**: The model explicitly reasons about what it knows, what it needs to find out, and what action to take next. This is a natural language reasoning trace visible in the output.
2. **Action**: The model invokes a tool — a web search, database query, API call, file read, code execution, etc.
3. **Observation**: The tool returns its result, which the model incorporates into its next reasoning step.

The cycle repeats until the model determines it has enough information to produce a final answer. The key insight is that reasoning traces help the model *maintain coherent plans across multiple steps* and *recover from errors*, while actions *ground the reasoning in real-world data* rather than relying solely on parametric knowledge.

### Effectiveness Evidence

On **knowledge-intensive tasks** (HotpotQA, FEVER), ReAct outperforms vanilla action-only generation and is competitive with chain-of-thought reasoning. The best results come from combining ReAct with CoT — using both internal knowledge and externally obtained information. On FEVER specifically, ReAct outperforms pure CoT.

On **decision-making tasks** (ALFWorld, WebShop), ReAct dramatically outperforms baselines. With just 1-2 shot prompting, it outperforms imitation and reinforcement learning methods trained with ~10^5 task instances, achieving absolute improvements of 34% and 10% in success rates respectively. This demonstrates that interleaved reasoning enables few-shot generalization that pure behavioral cloning cannot match.

Action-only models — which have access to the same tools but lack reasoning traces — consistently underperform, demonstrating that the reasoning component is essential for coherently combining tool outputs into answers.

### Failure Modes

- **Hallucinated reasoning**: The model can fabricate plausible-sounding reasoning traces that don't correspond to reality, leading to incorrect action choices.
- **Reasoning-action misalignment**: On pure knowledge tasks, ReAct can lag behind pure CoT because the overhead of tool interaction sometimes introduces noise rather than helpful information.
- **Token expense**: Interleaved reasoning traces are verbose, consuming significant context window space.
- **Action space design sensitivity**: ReAct's effectiveness depends heavily on having well-designed, well-documented tools. Poorly specified tools lead to cascading errors.

### Comparison to Alternatives

| Approach | Strength | Weakness |
|----------|----------|----------|
| ReAct | Grounded reasoning + tool use | Token-expensive, depends on tool quality |
| CoT only | Efficient, good for knowledge-heavy tasks | No external grounding, prone to hallucination |
| Action only | Simple | Cannot reason about multi-step plans |
| ReAct + CoT | Best of both worlds | Most token-expensive |

### Real-World Systems Using ReAct

ReAct is the **foundational architecture** for virtually all modern agentic systems. Claude Code, OpenAI Codex, GitHub Copilot agent mode, Devin, and every major coding agent uses some variant of the ReAct loop — the model reasons about the task, takes an action (read a file, run a command, edit code), observes the result, and repeats. The pattern is so dominant that it's the implicit default rather than an explicit choice.

### Applicability to Claude Code

Claude Code already implements ReAct as its core architecture: "Claude uses tools throughout, whether searching files to understand your code, editing to make changes, or running tests to check its work." The agentic loop is literally Thought-Action-Observation repeated until task completion. The question is not whether to adopt ReAct — it's already there — but how to augment it with the other patterns described below.

---

## 2. Reflection / Self-Critique

### How It Works

Reflection patterns have the agent review and revise its own output before finalizing. There are several variants:

**Basic Reflection (Generate-Critique-Refine)**: One agent generates output, another (or the same agent with a different prompt) critiques it, and the original output is revised. This can loop multiple times. Andrew Ng identified this as one of four core agentic design patterns, describing it as creating "two different agents, one prompted to generate good outputs and the other prompted to give constructive criticism."

**Reflexion (Shinn et al., 2023)**: A more structured approach using *verbal reinforcement learning*. Instead of updating model weights, agents verbally reflect on task feedback signals and maintain reflective text in an episodic memory buffer. The key innovation is that reflections persist across attempts — the agent *remembers what went wrong* and uses that memory to avoid repeating mistakes. Reflexion achieved 91% pass@1 on HumanEval (vs. 80% for GPT-4 baseline), demonstrating that self-reflection can substantially improve coding accuracy.

**Multi-Agent Reflexion (MAR, 2025)**: Separates the processes of acting, diagnosing, critiquing, and aggregating across multiple agents with diverse reasoning personas. A judge model synthesizes critiques into unified reflections, reducing shared blind spots.

**LATS Reflection Component**: Language Agent Tree Search incorporates reflection on failed trajectories — generating verbal summaries of errors that become context for future attempts. However, ablations show reflection alone contributes only modest gains (0.05 EM on HotPotQA); the systematic search component matters more.

### Effectiveness Evidence

Reflexion's results are compelling: 91% pass@1 on HumanEval represents an 11 percentage point improvement over the GPT-4 baseline. Across sequential decision-making, coding, and language reasoning tasks, reflection consistently improves accuracy.

A Nature-published study (2025) found that self-reflection yields "statistically significant performance improvements, with error correctivity observed both in single-step tasks like MCQA problem-solving with >18% accuracy boosts and long-horizon, multi-stage domains."

Anthropic's evaluator-optimizer pattern (a form of reflection) is described as "particularly effective when we have clear evaluation criteria, and when iterative refinement provides measurable value." The key condition: there must be a way to evaluate output quality that's better than the generation process itself.

### Failure Modes

- **Reflection without signal**: If the reflection prompt can't identify concrete errors, the loop produces generic, unhelpful critiques. LATS researchers noted that "generated reflections are often generic and do not provide useful feedback" in complex environments, causing agents to "become stuck in local minima."
- **Confirmation bias**: Models may affirm their own incorrect reasoning rather than genuinely critiquing it. Self-reflection quality varies significantly with model capability.
- **Infinite loops**: Without clear stopping criteria, reflection loops can cycle indefinitely without converging on improvement.
- **Cost multiplication**: Each reflection pass costs additional tokens and latency. For simple tasks, this overhead exceeds the quality gain.

### Comparison to Alternatives

Reflection vs. self-consistency (majority voting): Multi-agent debate "significantly underperforms simple self-consistency using majority voting" in many cases. Self-consistency with 18.6 samples matches the accuracy of more sophisticated approaches using only 10 samples. This suggests that for tasks where the answer is either right or wrong, simple redundancy (generate multiple times and vote) can be cheaper and equally effective as structured reflection.

Reflection vs. tree search: LATS ablations show that systematic search (exploring multiple paths) contributes far more than reflection alone. Reflection helps primarily by providing *context for future exploration*, not by fixing outputs in place.

### Real-World Systems Using Reflection

- **Claude Code's CLAUDE.md**: Our own research workflow mandates "revision cycles" — re-reading output, checking against a depth checklist, identifying gaps, and doing additional research. This is an explicit reflection pattern embedded in the system prompt.
- **Devin**: Uses self-verification loops where generated code is tested and revised.
- **Anthropic's evaluator-optimizer pattern**: Production recommendation for iterative refinement.

### Applicability to Claude Code

Reflection is already partially implemented in Claude Code's workflow through CLAUDE.md revision cycle instructions. The key opportunity is making reflection *conditional* — triggering deeper reflection only when output complexity warrants it, and ensuring reflections are grounded in concrete feedback (test results, compilation errors, checklist failures) rather than abstract self-assessment. The evidence strongly suggests that **reflection grounded in external feedback** (test results, tool output) is far more effective than purely self-generated critique.

---

## 3. Planning-First Architectures

### How It Works

Planning-first architectures explicitly separate planning from execution: first create a plan, then execute it step by step.

**Plan-then-Execute (P-t-E)**: A planner LLM generates a multi-step plan, then an executor carries it out one step at a time by invoking tools and APIs. The executor can be simpler and cheaper than the planner. Modern variants add a **re-planner** that assesses progress after each step and can modify the plan dynamically.

**ReWOO (Reasoning Without Observation)**: Decouples planning from tool execution entirely. The model generates a complete plan (Planner module), tools execute in sequence (Worker module), and a Solver module integrates all results. This achieves 64% token reduction with 4.4% accuracy gain over ReAct on average across six benchmarks, and 5x token efficiency on HotpotQA specifically. The key insight: the model only needs 2 LLM calls regardless of how many tools are used.

**Tree of Thoughts (ToT)**: Maintains a tree of intermediate reasoning states. At each step, the model generates multiple candidate "thoughts," evaluates them (sure/maybe/impossible), and explores the most promising branches using BFS or DFS. On Game of 24, ToT with GPT-4 achieved 74% success vs. 4% for CoT — a massive improvement for problems requiring search over reasoning paths.

**Graph of Thoughts (GoT)**: Extends ToT by allowing merging and combining of thought branches, enabling more complex reasoning topologies.

**Language Agent Tree Search (LATS)**: Combines MCTS with ReAct, achieving state-of-the-art results across multiple domains:
- HumanEval: 92.7% pass@1 (vs. 80.1% GPT-4 baseline)
- HotPotQA: 71% exact match (vs. 32% ReAct, 51% Reflexion)
- WebShop: 75.9 avg score (vs. 64.2 Reflexion)

LATS's value function combines LM-generated reasoning about state correctness with a self-consistency heuristic: V(s) = λ·LM(s) + (1−λ)·SC(s).

**Modular Agentic Planner (MAP, 2025)**: A brain-inspired architecture where planning is performed via interaction of specialized LLM modules (conflict monitoring, state prediction, state evaluation, task decomposition, task coordination).

### Effectiveness Evidence

The evidence for planning-first approaches is strong but nuanced:

- **For problems requiring search**: ToT and LATS show dramatic improvements. The 4% → 74% improvement on Game of 24 demonstrates that some problems *require* exploring multiple reasoning paths.
- **For efficiency**: ReWOO's 64% token reduction while maintaining or improving accuracy makes a compelling case for upfront planning when the plan structure is predictable.
- **For complex multi-step tasks**: AgentOrchestra's hierarchical planning achieved +4.84% on GAIA over the next best system, with more gradual performance decline on harder tasks.
- **Cost concern**: ToT cost ~$106 for experiments due to multiple LLM calls per step. LATS uses 173K tokens on HotPotQA (comparable to alternatives but still expensive).

### Failure Modes

- **Plan rigidity**: Static plans can't adapt to unexpected observations. Re-planning mitigates this but adds complexity.
- **Planning overhead on simple tasks**: For straightforward tasks, planning adds latency without benefit. Anthropic recommends starting simple and adding planning only when needed.
- **State reversion**: LATS assumes the ability to reset to earlier decision points, limiting applicability in truly sequential environments (like live system modifications).
- **Evaluation bottleneck**: ToT's effectiveness depends on the model's ability to evaluate intermediate states. Poor self-evaluation leads to pruning good branches and retaining bad ones.
- **Generic reflections in search**: LATS researchers found that "generated reflections are often generic and do not provide useful feedback" in complex environments like WebShop.

### Comparison to Alternatives

| Approach | Token Efficiency | Quality on Complex Tasks | Quality on Simple Tasks |
|----------|-----------------|--------------------------|------------------------|
| ReAct (greedy) | Medium | Medium | Good |
| ReWOO (plan-first) | High (5x) | Medium-High | Good |
| ToT (tree search) | Low | Very High | Overkill |
| LATS (MCTS) | Medium | Very High | Overkill |
| Simple CoT | High | Low-Medium | Good |

### Real-World Systems Using Planning

- **Claude Code**: CLAUDE.md enforces phase discipline (scope → research → synthesis) and task decomposition before execution. Implementation plans require explicit phase structures with acceptance criteria.
- **OpenAI Codex**: Builds structured prompts with system-developer-user role hierarchy, essentially creating a plan before executing tool calls.
- **AgentOrchestra**: Central Planning Agent decomposes tasks before delegating to specialized sub-agents.

### Applicability to Claude Code

Claude Code already embeds planning through CLAUDE.md's phase discipline and task decomposition requirements. The key insights for improvement:

1. **Adaptive planning depth**: Use simple ReAct for straightforward tasks, but switch to explicit plan-then-execute for complex multi-file changes. Don't over-plan simple tasks.
2. **ReWOO-style efficiency**: For tasks where tool calls are predictable (e.g., "read these 5 files, then generate a report"), planning all reads upfront could save significant tokens.
3. **Re-planning on failure**: When execution deviates from the plan, explicitly re-plan rather than continuing with an obsolete plan.

---

## 4. Multi-Agent Debate / Verification

### How It Works

Multiple LLM agents independently generate solutions, then engage in structured rounds of critique and refinement:

1. **Initial generation**: Multiple agents independently produce answers
2. **Debate rounds**: Agents review each other's answers and refine their own
3. **Aggregation**: Final answers are combined (voting, judge selection, or synthesis)

Variants include:
- **Assigned roles**: "Affirmative" and "negative," "angel" and "devil," or domain-specific expert personas
- **Judge architecture**: A separate judge agent manages the debate, evaluating rounds for correctness or extracting the final solution
- **Heterogeneous debate**: Using different foundation models for different agents, which "yields substantially higher accuracy on tasks like GSM-8K and enables emergent teacher-student dynamics"
- **ChatEval**: Multi-agent evaluation framework where agents play roles like domain experts, critics, and defenders

### Effectiveness Evidence

The evidence for multi-agent debate is mixed:

**Positive**: Debate achieves significant improvements over standard single-pass prompting. Heterogeneous agents (different models) outperform homogeneous ones. Role assignment stimulates critical thinking and diverse perspectives. Multi-agent evaluation frameworks incorporating diverse criteria and adversarial feedback produce more thorough assessments.

**Negative**: A critical finding from ICLR Blogposts 2025: "multi-agent debate significantly underperforms simple self-consistency using majority voting." This means that generating N independent answers and taking the majority vote is often cheaper and more effective than having N agents debate each other. The overhead of debate (additional rounds, cross-agent communication) frequently does not justify the marginal quality improvement over simpler aggregation.

**Efficiency concerns**: Token costs scale with number of agents times number of rounds. Sparse communication topologies and conditional participation can reduce costs by up to 94.5%, but add architectural complexity.

### Failure Modes

- **Groupthink**: Agents can converge on the same wrong answer, especially when using the same base model. Homogeneous agents are particularly susceptible.
- **Persuasion over correctness**: A more confidently stated wrong answer can sway other agents away from a tentatively stated correct answer.
- **Cost scaling**: N agents × R rounds × tokens per response = rapidly escalating costs.
- **Diminishing returns**: Adding more debate rounds or agents shows diminishing quality improvements.

### Comparison to Alternatives

Self-consistency (majority voting) is the key competitor. It achieves comparable or better accuracy at lower cost for many tasks. The confidence-improved variant (CISC) reduces required samples by 46% while matching accuracy.

Debate provides unique value when:
- The task benefits from diverse perspectives (creative tasks, thorough evaluation)
- Different agents have genuinely different capabilities or knowledge
- Adversarial checking is needed (fact-checking, safety review)

### Real-World Systems Using Debate

- **ChatEval**: Multi-agent evaluation framework for LLM outputs
- **Multi-Agent Reflexion (MAR)**: Diverse reasoning personas with judge synthesis
- **SWE-bench top performers**: Some use multi-LLM ensembles combining Claude with Gemini variants

### Applicability to Claude Code

For Claude Code's context, multi-agent debate has **limited direct applicability** in the traditional sense (running multiple Claude instances in debate). However, the underlying principle — having different perspectives check the same work — is valuable:

1. **Evaluator-optimizer** (Anthropic's pattern): One Claude call generates, another evaluates. This is more practical than full debate.
2. **Self-consistency for critical decisions**: Generate multiple candidate approaches and select the best, rather than committing to the first.
3. **Heterogeneous verification**: Using different tools/approaches to verify the same claim (e.g., running tests AND reading the code to verify correctness, rather than just one).

---

## 5. Hierarchical Agent Architectures

### How It Works

A central orchestrator decomposes complex tasks and delegates sub-tasks to specialized worker agents. The orchestrator maintains global context, tracks progress, and synthesizes results.

**Canonical pipeline** (five phases):
1. Requirement Elicitation
2. Task Decomposition — produces a structured TODO list (tree or DAG with dependencies)
3. Agent Assignment — routes subtasks to agents best suited by capability
4. Execution — workers execute their assigned tasks
5. Aggregation — orchestrator synthesizes final solution

**AgentOrchestra** (2025) implements a two-tier hierarchy: a Planning Agent as conductor, with specialized sub-agents (Deep Researcher, Browser Use Agent, Deep Analyzer). Each sub-agent includes a Python interpreter for verification. Results: 95.3% on SimpleQA, 82.42% on GAIA (overall), beating all baselines.

**Hierarchical Software Engineer (SWE)**: Discovered through automated search, uses hierarchy for code repair tasks.

**Orchestration topology variants**:
- **Hierarchical**: Tree structure, orchestrator at root, workers as leaves
- **Pipeline**: Linear chain of specialized agents
- **Swarm**: Peer-to-peer agent collaboration without central authority
- **Mesh**: Fully connected agents communicating freely

### Effectiveness Evidence

AgentOrchestra demonstrates that hierarchical organization with role specialization "consistently outperforms flat-agent and monolithic baselines in task success rate and adaptability." The key advantages:

1. **Separation of concerns**: The orchestrator focuses on planning; workers focus on execution. This matches the ReWOO insight — different capabilities benefit from different optimization.
2. **Scalability**: New capabilities can be added by adding new specialized agents without modifying the orchestrator.
3. **Cross-verification**: Multiple specialized agents can check each other's work, reducing hallucination risk.

From SWE-bench analysis: company submissions using scaffolded execution with rich context pipelines significantly outperform simpler approaches. The scaffolding effectively provides a form of hierarchical structure.

### Failure Modes

- **Orchestration overhead**: "Multiple agents introduce latency unsuitable for trivial queries requiring single-model responses." For simple tasks, hierarchy is pure overhead.
- **Communication bottleneck**: Inter-agent communication adds token costs and potential information loss.
- **Orchestrator as single point of failure**: If the orchestrator misunderstands the task or decomposes it incorrectly, all downstream work is wasted.
- **Context fragmentation**: Workers may lack global context needed for correct decisions, as only the orchestrator sees the full picture.

### Comparison to Alternatives

**SWE-bench finding**: "The most consistent, reliable architecture remained a single primary agent rather than multi-agent approaches." High-quality results are achievable with a single-agent, single-attempt architecture. Multi-agent hierarchies add value primarily for tasks that are genuinely too complex for a single context window or require genuinely different capabilities.

The key question is whether the task *requires* decomposition (too complex for one agent) or merely *benefits* from it (could be done by one agent but is done better with structure). The evidence suggests the latter case is less clear-cut than proponents claim.

### Real-World Systems Using Hierarchy

- **Claude Code sub-agents**: Claude Code spawns sub-agents for specific tasks within the research workflow described in CLAUDE.md.
- **AgentOrchestra**: Full hierarchical framework with Planning Agent + specialized sub-agents.
- **OpenAI Codex**: Structured prompt hierarchy (system → developer → user roles) provides implicit hierarchy.
- **CrewAI, AutoGen, LangGraph**: Frameworks supporting hierarchical multi-agent patterns.

### Applicability to Claude Code

Claude Code already uses a form of hierarchy through sub-agent dispatch (described in CLAUDE.md's sub-agent research prompt template). The key insights:

1. **Use hierarchy for genuinely complex tasks**: Multi-file refactors, cross-codebase analysis, research requiring multiple independent investigations. Don't add hierarchy for single-file edits.
2. **Rich context in delegation**: The orchestrator must provide sub-agents with sufficient context. Claude Code's sub-agent template does this well by specifying scope, prior work, and expected output format.
3. **Alignment checks**: The CLAUDE.md alignment protocol (checking outcome alignment, plan coherence, anti-patterns, research grounding, downstream impact) is a best practice that matches AgentOrchestra's approach.

---

## 6. Inner Monologue / Scratchpad Patterns

### How It Works

Extended thinking / inner monologue patterns give the model a "scratchpad" for intermediate reasoning before producing the final output. This encompasses:

**Chain-of-Thought (CoT)**: The foundational technique — prompting the model to show its reasoning step by step before answering. Now a standard capability in all frontier models.

**Reasoning Models (o1, R1, Claude with extended thinking)**: Models trained via reinforcement learning to generate extended reasoning traces ("thinking tokens") before the final response. This represents test-time compute scaling — allocating more computation at inference time for harder problems.

**MIRROR Architecture (2025)**: A cognitive architecture implementing parallel reasoning threads across dimensions (Goals, Reasoning, Memory) with a Cognitive Controller synthesizing them. Achieves up to 156% relative improvement in critical safety scenarios.

**Test-Time Compute Scaling**: The paradigm shift: instead of making the model larger (train-time scaling), make it think longer (test-time scaling). Performance improves with more reasoning tokens, up to a point.

### Effectiveness Evidence

**Strong evidence for complex tasks**: Reasoning models excel at olympiad-level mathematics, complex coding, and multi-step planning. The o1-to-o3 progression and DeepSeek-R1's competitive performance demonstrate that extended thinking genuinely improves capability on hard problems.

**Diminishing and negative returns on simple tasks**: Critical 2025 research reveals that beyond approximately 4K reasoning tokens, improvements saturate. The phenomenon of "overthinking" means that for simpler tasks like commonsense reasoning and basic mathematics, extended thinking can *harm* performance by introducing unnecessary complexity and entropy.

**CoT nuances**: Recent studies show that even incorrect CoT traces can yield correct outcomes, and fine-tuning on incorrect traces can be as effective as on correct ones. This suggests CoT's value may partially come from the *structure* of step-by-step processing rather than the *content* of each step.

**Mandatory thinking can backfire**: A 2026 paper ("Thinking Makes LLM Agents Introverted") found that mandatory thinking can reduce performance in user-engaged agent settings, where the overhead of extended reasoning delays action and reduces responsiveness.

### Failure Modes

- **Overthinking**: Extended reasoning on simple tasks wastes tokens and can reduce accuracy.
- **Unfaithful reasoning**: CoT traces may not reflect the model's actual decision-making process — they can be post-hoc rationalizations.
- **Token budget consumption**: Extended thinking consumes context window space that could be used for tool outputs and task context.
- **Latency**: More thinking tokens = more time before the user sees results.

### Comparison to Alternatives

Extended thinking vs. tool-augmented reasoning: The "Reasoning Trap" research shows that enhanced reasoning *amplifies tool hallucination* — stronger reasoning models are paradoxically more likely to fabricate tool calls or misuse tools. This suggests that **grounding in tool outputs** (pattern 7) may be more reliable than **pure reasoning depth** for factual accuracy.

Extended thinking vs. search (ToT/LATS): Explicit search over reasoning paths (ToT/LATS) often outperforms implicit longer reasoning. ToT's 74% vs. CoT's 4% on Game of 24 demonstrates that *structured* exploration beats *unstructured* longer thinking.

### Real-World Systems Using Inner Monologue

- **Claude with extended thinking**: Produces visible thinking blocks showing reasoning process.
- **OpenAI o1/o3**: Hidden scratchpad reasoning before final output.
- **DeepSeek-R1**: Open-source reasoning model with extended CoT.
- **Claude Code's "effort" parameter**: Controls reasoning depth independently of extended thinking.

### Applicability to Claude Code

Claude Code already supports extended thinking and effort levels. The key insight is **adaptive reasoning depth**:

1. **Match reasoning depth to task complexity**: Simple file reads and minor edits need minimal thinking. Complex architectural decisions need extended reasoning.
2. **Use structured search for complex problems**: Rather than just thinking longer, explore multiple approaches explicitly (closer to ToT than to pure CoT extension).
3. **Ground reasoning in evidence**: Extended thinking is most valuable when it's reasoning *about* tool outputs and concrete evidence, not generating reasoning in a vacuum.

---

## 7. Tool-Augmented Reasoning

### How It Works

Instead of reasoning purely from parametric knowledge, the model grounds its reasoning in outputs from external tools: code interpreters, search engines, file system operations, calculators, databases, APIs. This fundamentally changes the reliability profile of the agent.

**Program-Aided Language Models (PAL)**: The model generates code to represent reasoning steps, then executes that code to get results. This decouples the *decomposition* of a problem (which LLMs are good at) from the *computation* (which code interpreters are better at). On GSM-hard, PAL outperforms CoT by an absolute 40%.

**Program of Thoughts (PoT)**: Generates hybrid rationales mixing natural language and code, executing the code portion for numerical accuracy.

**Tool-Integrated Reasoning (TIR)**: The broader category encompassing search, calculation, code execution, and any external tool that provides factual grounding.

**Live-SWE-agent (2025)**: Demonstrates that on-the-fly tool creation boosts resolve rates by up to 22.6 percentage points for strong LLMs — the agent not only uses tools but *creates new tools* as needed.

### Effectiveness Evidence

**PAL on math**: 11% improvement over CoT on BIG-Bench Hard, 40% absolute improvement on GSM-hard. This is one of the most dramatic improvements in the literature — using a code interpreter for calculation rather than relying on the LLM's internal arithmetic.

**Grounding prevents specific hallucination types**: When the model reads actual file contents (rather than guessing), executes actual code (rather than simulating execution mentally), and checks actual test results (rather than predicting them), entire categories of hallucination become impossible.

**SWE-bench patterns**: Top performers use specialized tools for code search, file reading, and bulk editing. "Specialized tools letting agents manage context windows more effectively" correlates with high performance. Tool design is a differentiator — Agentless achieved 50.8% using agent-free reasoning with good tools, while SWE-Agent reached 66% with an influential open-source tool design.

### The Reasoning Trap (Critical Caveat)

A 2025 paper establishes that "stronger reasoning often coincides with increased hallucination" specifically regarding tools. Reinforcement learning for reasoning enhancement "inherently biases models toward overconfident 'think-then-act' behaviors." Key findings:

- Hallucination rates increase monotonically with RL training steps, even when task performance improves
- This happens even with non-agentic reasoning RL (e.g., math training with no tool involvement)
- DeepSeek-R1-Distill shows 74.3% tool hallucination rate vs. 34.8% for the base model
- The phenomenon is method-agnostic: distillation, native thinking modes, and RL all exhibit it

This means that more capable reasoning models may be *more* prone to fabricating tool calls or misusing tool outputs, not less. Mitigation requires explicit training for tool abstention and calibrated confidence.

### Failure Modes

- **The Reasoning Trap**: Enhanced reasoning amplifies tool hallucination
- **Tool dependency**: Reliance on external tools introduces reliability risks (network failures, API changes)
- **Cascading errors**: Misinterpreted tool output propagates through subsequent reasoning steps
- **Over-reliance on tools for simple tasks**: Using code execution for trivial arithmetic adds latency without benefit

### Comparison to Alternatives

Tool-augmented reasoning vs. pure reasoning: For any task involving factual lookup, numerical computation, or code execution, tool augmentation dramatically outperforms pure reasoning. The 40% improvement of PAL over CoT on math is representative.

Tool-augmented reasoning vs. larger models: "Generating code using an LLM and reasoning using a Python interpreter leads to more accurate results than much larger models." This suggests that tool augmentation is often more cost-effective than model scaling for accuracy on specific task types.

### Real-World Systems Using Tool-Augmented Reasoning

- **Claude Code**: Tools are described as "what make Claude Code agentic" — file reads, edits, command execution, web search, MCP server interactions. The entire architecture is built around tool-augmented reasoning.
- **OpenAI Codex**: Executes commands in isolated containers, appends results to conversation.
- **Every SWE-bench top performer**: Relies heavily on code search, file reading, test execution tools.

### Applicability to Claude Code

Claude Code is already a tool-augmented reasoning system. Improvements should focus on:

1. **Tool quality**: Better-designed tools (more informative outputs, better error messages) directly improve reasoning quality. The SWE-bench evidence shows tool design is a differentiator.
2. **Dynamic tool creation**: Live-SWE-agent's 22.6% improvement from on-the-fly tool creation suggests Claude Code could benefit from creating custom scripts/tools during complex tasks.
3. **Grounding as default**: Always verify claims against tool outputs rather than parametric knowledge. Read the file rather than assuming its contents; run the test rather than predicting the result.
4. **Awareness of the Reasoning Trap**: When using reasoning-enhanced models, be aware that confident-sounding tool calls may be fabricated. Verification of tool availability before invocation is important.

---

## 8. Test-Driven / Verification-Driven Development

### How It Works

Agents write tests first (or use existing tests), then iteratively generate and refine code until tests pass. This creates an objective, automated verification loop that is arguably the single most effective quality-improvement pattern for coding agents.

**The TDD Agent Loop**:
1. Write (or receive) tests expressing expected behavior
2. Confirm tests fail (red)
3. Generate code to make tests pass
4. Run tests
5. If tests fail, analyze failure, revise code, goto 4
6. If tests pass (green), continue to next task

**Codex's approach**: Trained via reinforcement learning specifically to "iteratively run tests until it receives a passing result." The model runs "verification steps (tests, lint, typecheck) for every milestone it completed." This verification is baked into the model's training, not just the scaffolding.

**Simon Willison's Agentic Engineering Patterns (March 2026)**: Advocates running existing tests *before* making any changes ("first run the tests"), then using TDD methodology adapted for agent workflows. Key principles: "code is now inexpensive" (agents generate code quickly) and "preserve domain expertise" (developers retain knowledge work, agents handle implementation).

**SWE-bench Verification**: Each benchmark sample has associated unit tests that fail before the solution and pass afterwards (FAIL_TO_PASS tests). The entire evaluation framework is test-driven — and the most successful agents internalize this pattern.

### Effectiveness Evidence

**SWE-bench results**: The benchmark itself demonstrates TDD's value — agents are evaluated by whether their patches make failing tests pass. Top performers (75.2% on SWE-bench Verified) all use test execution as a core feedback mechanism. Devin achieved 23% in a TDD setting with provided tests.

**Live-SWE-agent**: Achieves 77.4% on SWE-bench Verified with Claude Opus 4.5, demonstrating that strong models + test-based verification produces excellent results.

**Codex's RL training**: By training the model to iteratively test and refine, OpenAI achieved code that "closely mirrors human style and PR preferences" and "adheres precisely to instructions." The verification loop is the quality mechanism.

**Practical evidence**: The Agentic Coding Handbook states that "the agentic loop is where TDD with AI truly shines" because tests provide *objective, fast feedback* that the agent can iterate on. Instead of prompting the AI to generate everything at once, describing one behavior at a time through tests lets the AI "build up the logic incrementally, safely, and cleanly."

### Failure Modes

- **Test quality**: If tests are poorly written, incomplete, or test the wrong thing, passing them doesn't indicate correctness. "Goodharting" on tests is a real risk.
- **Overfitting to tests**: The agent may generate code that passes tests through coincidence (e.g., hardcoding expected outputs) rather than implementing correct logic.
- **Missing test infrastructure**: Not all projects have test infrastructure, and setting it up adds overhead.
- **Slow test suites**: If tests take minutes to run, the iterative loop becomes impractical. Fast tests are essential for agent productivity.
- **SWE-bench generalization gap**: Performance drops "sharply to roughly 53% when models are tested on tasks from different repositories outside the SWE-bench set," suggesting some amount of benchmark-specific adaptation.

### Comparison to Alternatives

Verification-driven development vs. generate-and-hope: The contrast is stark. Without verification, code correctness depends entirely on the model's parametric knowledge. With verification, there is an objective check that catches many errors automatically.

TDD vs. post-hoc testing: Writing tests first (TDD) forces the agent to understand expected behavior *before* generating code. Post-hoc testing can discover bugs but doesn't guide the generation process.

TDD vs. formal verification: For the small subset of code that can be formally verified, formal methods provide stronger guarantees. But TDD is applicable to virtually any code, making it far more practical.

### Real-World Systems Using TDD

- **OpenAI Codex**: RL-trained to run tests iteratively; runs verification for every milestone.
- **Claude Code**: Runs tests as part of the agentic loop ("verify results" phase).
- **GitHub Copilot agent mode**: "Can iterate on its own code, recognize errors, and fix its mistakes in real time."
- **Devin**: Self-verification loops with test execution.
- **All SWE-bench top performers**: Test execution is the primary feedback mechanism.

### Applicability to Claude Code

Test-driven verification is **the highest-impact pattern for Claude Code adoption**. Specific improvements:

1. **Always run existing tests before and after changes**: Establish a baseline and verify no regressions. This should be the default behavior, not optional.
2. **Write tests for new functionality before implementing**: When Claude Code generates new code, generate tests first, verify they fail, then implement.
3. **Use test results as reflection triggers**: Test failures are concrete, specific feedback — far better than abstract self-critique. Failed tests should trigger focused debugging, not generic reflection.
4. **Fast feedback loops**: Prioritize running targeted tests (not full suites) for rapid iteration. The agent should identify which tests are relevant and run only those.
5. **Lint and typecheck as lightweight verification**: Not just unit tests — compilation, linting, and type checking are cheap verification steps that catch entire categories of errors.

---

## Cross-Cutting Analysis

### Pattern Synergies

The most effective real-world systems combine multiple patterns. The dominant architecture in early 2026 is:

**ReAct loop** (base) + **Tool-augmented reasoning** (grounding) + **Test-driven verification** (quality gate) + **Adaptive reflection** (on failures only) + **Planning** (for complex tasks)

This is essentially what Claude Code and OpenAI Codex already implement. The patterns reinforce each other:
- ReAct provides the action loop
- Tools ground reasoning in reality
- Tests provide objective success criteria
- Reflection analyzes failures to guide retries
- Planning structures complex tasks before execution

### What the SWE-Bench Evidence Says About Patterns

The SWE-bench leaderboard dissection reveals that "no single architecture consistently achieves state-of-the-art performance" — but several correlations emerge:

1. **Model quality matters most**: Proprietary LLM access (Claude 3.5+, o1-series) is the strongest predictor of high performance.
2. **Scaffolded execution with rich context**: Top performers provide rich context (repository structure, related files, dependencies) to the model.
3. **Iterative refinement with feedback**: Verification loops correlate with high scores.
4. **Tool design is a differentiator**: Specialized tools for code search, file reading, and bulk editing matter significantly.
5. **Single-agent often suffices**: "The most consistent, reliable architecture remained a single primary agent" — multi-agent adds value only for genuinely complex tasks.

### Patterns Ranked by Impact for Claude Code

Based on the evidence reviewed, here is a priority ranking for adoption:

| Rank | Pattern | Expected Impact | Current Claude Code Status |
|------|---------|----------------|---------------------------|
| 1 | Test-driven verification | Very High | Partial — runs tests but not always TDD |
| 2 | Tool-augmented reasoning | Very High | Strong — core architecture |
| 3 | Reflection/self-critique | High | Partial — CLAUDE.md mandates revision cycles |
| 4 | Planning-first (adaptive) | High | Partial — phase discipline in CLAUDE.md |
| 5 | Inner monologue (adaptive) | Medium-High | Available via extended thinking |
| 6 | Hierarchical agents | Medium | Partial — sub-agent dispatch |
| 7 | ReAct | Already core | Fully implemented |
| 8 | Multi-agent debate | Low-Medium | Not directly applicable |

### Key Takeaway

The evidence converges on a clear message: **grounding and verification beat reasoning depth**. The most impactful improvements come from:
1. Better tools that provide more useful information
2. Verification loops that objectively check output quality
3. Targeted reflection triggered by concrete failures
4. Planning proportional to task complexity

Adding more reasoning tokens, more agents, or more debate rounds shows diminishing returns compared to these foundational practices. The "Reasoning Trap" research underscores this — better reasoning without better grounding can actually make things worse.

---

## Sources

### Papers
- Yao et al., "ReAct: Synergizing Reasoning and Acting in Language Models" (ICLR 2023) — https://arxiv.org/abs/2210.03629
- Shinn et al., "Reflexion: Language Agents with Verbal Reinforcement Learning" (NeurIPS 2023) — https://arxiv.org/abs/2303.11366
- Yao et al., "Tree of Thoughts: Deliberate Problem Solving with Large Language Models" (NeurIPS 2023) — https://arxiv.org/abs/2305.10601
- Zhou et al., "Language Agent Tree Search Unifies Reasoning, Acting, and Planning in Language Models" (ICML 2024) — https://arxiv.org/abs/2310.04406
- Xu et al., "ReWOO: Decoupling Reasoning from Observations for Efficient Augmented Language Models" — https://arxiv.org/abs/2305.18323
- Gao et al., "PAL: Program-aided Language Models" — https://arxiv.org/abs/2211.10435
- "The Reasoning Trap: How Enhancing LLM Reasoning Amplifies Tool Hallucination" — https://arxiv.org/abs/2510.22977
- "AgentOrchestra: A Hierarchical Multi-Agent Framework" — https://arxiv.org/abs/2506.12508
- "Dissecting the SWE-Bench Leaderboards" — https://arxiv.org/abs/2506.17208
- "MIRROR: Cognitive Inner Monologue Between Conversational Turns" — https://arxiv.org/abs/2506.00430
- "Agentic Large Language Models, a survey" — https://arxiv.org/abs/2503.23037
- "Multi-Agent Debate for LLM Judges with Adaptive Stability Detection" — https://arxiv.org/abs/2510.12697
- "MAR: Multi-Agent Reflexion Improves Reasoning" — https://arxiv.org/abs/2512.20845
- "Does Thinking More Always Help? Understanding Test-Time Scaling in Reasoning Models" — https://arxiv.org/abs/2506.04210
- "Thinking Makes LLM Agents Introverted" — https://arxiv.org/abs/2602.07796
- "Thoughts without Thinking" — https://arxiv.org/abs/2505.00875

### Industry Resources
- Anthropic, "Building Effective Agents" (December 2024) — https://www.anthropic.com/research/building-effective-agents
- OpenAI, "Unrolling the Codex Agent Loop" — https://openai.com/index/unrolling-the-codex-agent-loop/
- OpenAI, "Introducing SWE-bench Verified" — https://openai.com/index/introducing-swe-bench-verified/
- Cognition AI, "SWE-bench Technical Report" — https://cognition.ai/blog/swe-bench-technical-report
- Andrew Ng, "Four Design Patterns for AI Agent Workflows" — https://x.com/AndrewYNg/status/1770897666702233815
- Simon Willison, "Agentic Engineering Patterns" — https://simonwillison.net/guides/agentic-engineering-patterns/
- Tweag, "Agentic Coding Handbook: TDD" — https://tweag.github.io/agentic-coding-handbook/WORKFLOW_TDD/
- Claude Code Documentation — https://code.claude.com/docs/en/how-claude-code-works
- LangChain, "Reflection Agents" — https://blog.langchain.com/reflection-agents/
- Multi-Agent Debate Performance (ICLR Blogposts 2025) — https://d2jud02ci9yv69.cloudfront.net/2025-04-28-mad-159/blog/mad/
