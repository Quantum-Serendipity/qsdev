# Multi-Agent Orchestration Frameworks: Research Report

## Executive Summary

This report surveys the current landscape of multi-agent AI orchestration frameworks, analyzing nine major systems and the broader question of when multi-agent approaches improve output quality versus when they harm it. The central finding is that multi-agent orchestration is not universally beneficial — it dramatically helps on parallelizable tasks but degrades performance on sequential ones. The most successful implementations use simple, composable patterns rather than heavyweight frameworks. For Claude Code specifically, the most adoptable patterns are the orchestrator-worker pattern with depth-limited sub-agents, code-as-action (from smolagents), systematic prompt optimization (from DSPy), and the "start simple, add complexity only when measured" philosophy advocated by Anthropic.

---

## Part 1: Framework-by-Framework Analysis

### 1. LangGraph (LangChain)

**Architecture**: Graph-based agent orchestration. Models workflows as directed graphs with three core abstractions: State (shared data via TypedDict with reducer functions), Nodes (functions encoding agent logic), and Edges (routing functions determining next node). Compiled graphs support checkpointing, fault tolerance, and human-in-the-loop patterns.

**Core Mechanism**: The StateGraph class parameterized by user-defined state objects. Reducer functions specify how state updates are merged per field — enabling concurrent updates without data loss. Compilation step attaches runtime infrastructure (checkpointers for persistence, breakpoints for human intervention). Storage backends include memory, SQLite, PostgreSQL, and S3.

**Multi-Agent Support**: Supervisor agent patterns, tool-calling with graph-based routing, parallel execution branches. The graph structure naturally supports branching, cycles (for replanning), and conditional routing.

**Quality Characteristics**: LangGraph's explicit state management and checkpointing provide strong reliability guarantees. In benchmarks, it has the lowest latency of major frameworks. The graph structure forces explicit modeling of control flow, which makes workflows more predictable but harder to set up.

**Failure Modes**:
- Steep learning curve requiring graph theory and distributed systems knowledge
- Over-engineering criticism: reimplements control flow that programming languages already provide
- Memory leaks and state management issues in production (reports of 2GB RAM for basic retrieval tasks)
- Agent looping problems consuming unnecessary tokens
- Version compatibility issues with the LangChain ecosystem
- Debugging complex graph structures is harder than debugging linear code

**Adoption**: 600-800 companies in production by end of 2025. Used by Klarna, Replit, Elastic. LangChain's official recommendation: "Use LangGraph for agents, not LangChain."

**Claude Code Relevance**: LangGraph's explicit state management and checkpointing patterns are valuable concepts. The reducer pattern for merging concurrent state updates could improve Claude Code's handling of parallel sub-agent results. However, the graph-based abstraction adds complexity that Claude Code deliberately avoids.

---

### 2. CrewAI

**Architecture**: Role-based multi-agent framework. Dual-model architecture: Crews (teams of autonomous agents with roles, goals, and backstories) and Flows (event-driven production scaffolding with conditional branching and state management). Built from scratch, independent of LangChain.

**Core Mechanism**: Each agent has a defined role, goal, backstory, assigned tools, and optional delegation capability. Process types determine orchestration: Sequential (tasks execute in order, no central coordinator) and Hierarchical (manager agent breaks goals into subtasks, dispatches to workers, synthesizes output).

**Quality Characteristics**: The role-based abstraction is intuitive and mirrors real-world team structures. However, independent benchmarks reveal a significant "managerial overhead" — CrewAI consumes nearly 3x the tokens of comparable frameworks and takes nearly 3x longer due to multi-step verification between its Planner and Analyst personas.

**Critical Failure Mode**: The hierarchical manager-worker process "simply does not function as documented." In real workflows, the manager does not effectively coordinate agents; CrewAI executes tasks sequentially regardless, leading to incorrect reasoning, unnecessary tool calls, and extremely high latency. This is a fundamental gap between the framework's promise and its actual behavior.

**Adoption**: 45,900+ GitHub stars, 100,000+ certified developers. Strong community but significant complaints about hierarchical process reliability.

**Claude Code Relevance**: The role-based abstraction (giving agents personas with specific expertise) is a useful pattern for sub-agent prompting. The lesson about "managerial overhead" is important — adding a coordination layer can cost more than it saves. The Flows concept (event-driven state management wrapping autonomous crews) is a potentially useful production pattern.

---

### 3. AutoGen / AG2 (Microsoft)

**Architecture**: Conversational multi-agent framework. AutoGen 0.4 (January 2025) uses event-driven, modular architecture separating core runtime from agent implementations. AG2 is the community fork maintaining the 0.2 API.

**Core Mechanism**: GroupChat is the primary coordination pattern — multiple agents in a shared conversation where a selector determines who speaks next (round-robin, random, or LLM-guided). AssistantAgent wraps an LLM for reasoning; UserProxyAgent represents human participants; CodeExecutor agents write and run sandboxed code.

**The Fork**: Original creators Chi Wang and Qingyun Wu departed Microsoft in late 2024 to create AG2 as a community-driven fork. This split fragments the ecosystem — AutoGen 0.4 and AG2 have diverging APIs and roadmaps.

**Quality Characteristics**: Strong for code generation workflows where agents iterate, critique, and improve each other's outputs. The conversational pattern is natural for code review, content generation, and research tasks. Magentic-One (built on AutoGen) achieves competitive performance on GAIA, AssistantBench, and WebArena benchmarks with a specialized five-agent team.

**Failure Modes**:
- Community split creates confusion about which version to use
- GroupChat selection can be unpredictable (LLM-guided selection is non-deterministic)
- Complex setup for production deployments
- Native async execution in 0.4 adds learning curve

**Adoption**: Major backing from Microsoft Research. Active development on both forks.

**Claude Code Relevance**: The GroupChat pattern (agents in shared conversation with a selector) is interesting for multi-perspective review scenarios. The Magentic-One architecture (Orchestrator + four specialized agents: WebSurfer, FileSurfer, Coder, ComputerTerminal) is a proven reference architecture for complex agentic tasks. The conversational critique pattern (agents improving each other's output) could enhance Claude Code's revision cycles.

---

### 4. OpenAI Swarm / Agents SDK

**Architecture**: Swarm (October 2024, now deprecated) was a minimal educational framework built on two primitives: Agents and Handoffs. The OpenAI Agents SDK (March 2025) is its production successor, adding Guardrails, tracing, and sessions.

**Core Mechanism**: Agents are LLMs with instructions and tools. Handoffs transfer conversation control between agents — like a phone transfer where the next agent has full conversation history. Guardrails validate inputs/outputs at agent boundaries. The design is deliberately minimal.

**Design Philosophy**: Lightweight, highly controllable, easily testable. Swarm's stateless architecture provided transparency and fine-grained control. The Agents SDK adds just enough production infrastructure (tracing, guardrails, sessions) without abandoning the simplicity ethos.

**Quality Characteristics**: The handoff pattern works well for customer service and triage scenarios where different specialists handle different types of requests. Less suited for tasks requiring complex coordination or parallel execution.

**Failure Modes**:
- Handoff-only pattern limits complex orchestration
- No built-in support for parallel agent execution
- Less sophisticated state management than LangGraph
- Primarily OpenAI-centric ecosystem

**Adoption**: 11,000+ GitHub stars. AgentKit (October 2025) added visual development tools and enterprise features.

**Claude Code Relevance**: The simplicity philosophy aligns with Claude Code's own approach. The handoff pattern is a clean abstraction for routing between specialized sub-agents. Guardrails (input/output validation at agent boundaries) could improve Claude Code's sub-agent output quality. The key lesson: minimal abstractions can be more effective than comprehensive frameworks.

---

### 5. Claude Code / Claude Agent SDK

**Architecture**: Single-threaded master loop ("nO") with depth-limited sub-agent spawning. One flat message history. Tools drive agency. Deliberately avoids complexity of multi-agent swarms.

**Core Mechanism**: Classic while-loop: continue execution as long as responses include tool calls; terminate when Claude produces plain text. Sub-agents via `dispatch_agent` (internally I2A/Task Agent) operate with strict depth limitation — sub-agents cannot spawn their own sub-agents. Up to 7 parallel sub-agents at a time.

**Extension Framework**:
- **Skills** (October 2025): Reusable expertise modules loaded dynamically
- **MCP**: Plug-and-play external tool connectivity
- **Hooks**: Application-level customization without consuming context
- **Agent Teams** (early 2026): Experimental team lead agent that delegates to specialists

**Quality Characteristics**: Anthropic's own multi-agent research system (Claude Opus 4 lead + Claude Sonnet 4 subagents) outperformed single-agent Claude Opus 4 by 90.2%. Key drivers: parallelization (3-5 subagents, 3+ tools per subagent in parallel), extended thinking for strategy, interleaved thinking for evaluation.

**Token Economics**: Agents use ~4x more tokens than chat; multi-agent systems ~15x more. Three factors explain 95% of BrowseComp performance variance: token usage (80%), tool calls, model choice. Multi-agent works mainly because it helps spend enough tokens to solve the problem.

**Failure Modes**:
- Single depth level limits complex decomposition
- No built-in state persistence across sessions (relies on files like tasks.md, log.md)
- Limited parallel orchestration compared to dedicated frameworks

**Claude Code Relevance**: This IS Claude Code. Key insight: the deliberate simplicity (single loop, flat history, depth-limited sub-agents) is itself a quality-enhancing design choice. The file-based memory pattern (CLAUDE.md, tasks.md, log.md) compensates for the lack of built-in persistence. The 90.2% improvement from multi-agent validates the orchestrator-worker pattern with cheap subagents doing parallel exploration.

---

### 6. DSPy (Stanford NLP)

**Architecture**: Programmatic prompt optimization. Three abstractions: Signatures (declarative input/output specs), Modules (strategies for invoking LMs), and Optimizers/Teleprompters (systematic prompt tuning algorithms).

**Core Mechanism**: Instead of hand-crafting prompts, you write compositional Python code with type-annotated signatures. DSPy compiles this into a pipeline, runs it across training data, collects traces, filters for high-scoring trajectories, and uses these to generate optimized prompts or few-shot examples. Key optimizers include BootstrapFewShot (generates demonstrations from teacher module), MIPROv2 (Bayesian optimization over instruction + example combinations), and SIMBA (stochastic mini-batch analysis of failures).

**Quality Characteristics**: Eliminates brittle hand-crafted prompts. Model-agnostic — programs optimized for one model transfer to others. Applies classical ML concepts (training data, metrics, optimization) to prompt engineering. The systematic approach consistently outperforms manual prompt engineering when good metrics and data are available.

**Failure Modes**:
- Optimization quality is fundamentally limited by metric quality — garbage metrics produce garbage prompts
- Optimization can be expensive in tokens (many trials needed)
- Optimized prompts may not work well outside DSPy's internal context (portability issues)
- Requires high-quality training data
- Significant learning curve — requires ML expertise to use well
- Current gaps in observability, tracking, cost management, and deployment

**Adoption**: Active research community around Stanford NLP. Growing production adoption but still more research-oriented than LangGraph/CrewAI.

**Claude Code Relevance**: DSPy's core insight — that prompts should be systematically optimized against metrics rather than hand-crafted — is highly relevant. For Claude Code's CLAUDE.md and sub-agent prompts, a DSPy-like approach of testing prompt variations against real tasks and measuring outcomes would be more rigorous than the current manual iteration. The BootstrapFewShot pattern (generating demonstrations from successful runs) could improve sub-agent instruction quality. However, DSPy's full optimization loop may be overkill for Claude Code's use case.

---

### 7. Semantic Kernel (Microsoft)

**Architecture**: Enterprise-grade SDK (C#, Python, Java) with a central Kernel orchestrator, Plugin system, and Agent Framework. Plugins are semantically-described function wrappers. The Agent Framework provides five orchestration patterns.

**Core Mechanism**: The Kernel discovers, invokes, and manages plugins based on semantic descriptions. Planners (now deprecated in favor of function calling) let the LLM determine which plugins to invoke and in what order. The Agent Orchestration framework provides Sequential, Concurrent, Handoff, Group Chat, and Magentic patterns.

**Five Orchestration Patterns**:
1. **Sequential**: Pipeline, each agent builds on previous output
2. **Concurrent**: Broadcast to many agents, aggregate results
3. **Handoff**: Dynamic control transfer between agents based on context
4. **Group Chat**: Shared conversation, all agents see all messages
5. **Magentic**: Manager coordinates team of specialists (based on Magentic-One)

**Quality Characteristics**: Enterprise-ready with Azure integration, multi-language support, responsible AI features. The rich set of orchestration patterns provides flexibility. But the Agent Framework is still experimental and under active development.

**Failure Modes**:
- Agent orchestration still experimental (as of 2026)
- Microsoft/Azure-centric ecosystem
- Heavier weight than simpler frameworks
- Planner deprecation indicates evolving, unstable API surface

**Claude Code Relevance**: The five orchestration patterns provide a useful taxonomy of multi-agent coordination strategies. The Concurrent pattern (broadcast + aggregate) maps well to Claude Code's parallel sub-agent spawning. The plugin architecture's semantic descriptions mirror how MCP tools need clear descriptions for effective agent use. Semantic Kernel's convergence with AutoGen under the "Microsoft Agent Framework" umbrella signals that enterprise multi-agent is maturing.

---

### 8. smolagents (HuggingFace)

**Architecture**: Minimalist code-agent framework (~1,000 lines). Two agent types: CodeAgent (writes Python code for actions) and ToolCallingAgent (standard JSON tool calling). Core innovation: code-as-action.

**Core Mechanism**: The agent loop is a simple while loop that iterates until the LLM decides to stop. CodeAgent generates Python code to invoke tools, handle data, and compose operations. Custom tools defined via `@tool` decorator with typed parameters and docstrings. Sandboxed execution via Modal, E2B, or Docker.

**Code-as-Action Evidence**: Research papers demonstrate code agents are measurably superior to JSON-based tool calling:
- **30% fewer steps** (thus 30% fewer LLM calls)
- Higher performance on complex benchmarks
- Better composability (function nesting, loops, conditionals)
- Better object management
- LLMs already trained extensively on code patterns

**Quality Characteristics**: The efficiency gain from code-as-action is significant and well-documented. Open-source models now compete with closed models for agentic workflows using this approach.

**Failure Modes**:
- Less mature than major frameworks
- Limited built-in multi-agent orchestration
- Security relies on external sandboxing
- No built-in persistence, tracing, or production infrastructure
- Smaller community

**Claude Code Relevance**: The code-as-action finding is directly relevant — Claude Code already generates code (bash commands, file edits) as its primary action mechanism, which aligns with smolagents' research showing this is 30% more efficient than JSON tool calling. The minimalist philosophy (~1,000 lines) validates Claude Code's own simplicity-first approach. The `@tool` decorator pattern for clean tool definitions is a useful reference for MCP tool design.

---

### 9. Emerging Patterns (2025-2026)

**Google ADK + A2A Protocol**: Google's Agent Development Kit (April 2025) provides hierarchical agent trees with the Agent2Agent (A2A) protocol for inter-agent communication. A2A is an open standard enabling agents from different providers/frameworks to communicate — the "HTTP for agents." Now at v0.3 with gRPC support.

**MCP (Model Context Protocol)**: Broad adoption throughout 2025. Standardizes agent-to-tool connectivity. Supported by Anthropic, OpenAI, Google, and most major frameworks. Together with A2A, forms the emerging standard stack for agentic systems.

**Microsoft Agent Framework Convergence**: AutoGen and Semantic Kernel converging under unified framework. Semantic Kernel for enterprise production, AutoGen for research-oriented patterns. Shared runtime and deployment infrastructure.

**Self-Improving Agents**: Emerging trend where agents develop their own tools. DSPy's GEPA optimizer represents this pattern — agents that reflectively evolve their own prompts and strategies.

**Market Scale**: Gartner reports 1,445% surge in multi-agent system inquiries from Q1 2024 to Q2 2025. By 2026, 40% of enterprise applications expected to feature task-specific agents.

---

## Part 2: When Does Multi-Agent Actually Help?

This is arguably the most important question in the report. The research evidence is nuanced and sometimes contradictory.

### Evidence That Multi-Agent Helps

**Anthropic's 90.2% improvement**: Their multi-agent research system (Claude Opus 4 + Claude Sonnet 4 subagents) outperformed single-agent Claude Opus 4 by 90.2% on research tasks. The primary mechanism: parallelization enables spending more tokens exploring the problem space simultaneously, with subagents providing context window isolation and separation of concerns.

**Google's parallelizable tasks finding**: On parallelizable tasks like financial reasoning (where distinct agents can analyze revenue trends, cost structures, and market comparisons simultaneously), centralized coordination improved performance by +81% over single agents.

**Magentic-One benchmarks**: The five-agent system achieves competitive performance on GAIA, AssistantBench, and WebArena without modification to core capabilities or collaboration patterns.

### Evidence That Multi-Agent Hurts

**Google's sequential tasks finding**: On sequential tasks (like PlanCraft, where each step depends on the previous one), multi-agent coordination degraded performance by -70%. More agents amplified errors instead of correcting them.

**Error amplification**: Independent agents amplify errors 17.2x. Even centralized coordination only reduces this to 4.4x. The coordination overhead can overwhelm the benefits.

**Saturation threshold**: Coordination yields diminishing or negative returns once single-agent baselines exceed ~45% accuracy. At that point, the model is already good enough that adding agents just adds overhead.

**Diminishing returns with better models**: Research from May 2025 shows that "the benefits of MAS over SAS diminish as LLM capabilities improve." As frontier models like o3 and Gemini 2.5-Pro get better at long-context reasoning, memory retention, and tool usage, multi-agent advantages shrink.

**Implementation failure rate**: When agents don't coordinate well or when memory/structure aren't robust, problems cascade — derailing 40-80% of implementations.

### The Token Economics Argument

Anthropic's data provides the clearest insight into WHY multi-agent works when it does. Three factors explain 95% of BrowseComp performance variance:
1. **Token usage** (80% of variance)
2. Number of tool calls
3. Model choice

Multi-agent systems consume ~15x more tokens than chat interactions. They work mainly because they help **spend enough tokens** to solve the problem. Subagents facilitate this by operating in parallel context windows, enabling more total reasoning without hitting individual context limits.

This suggests that multi-agent is often a **mechanism for scaling inference-time compute** rather than a fundamentally different reasoning approach. If you could give a single agent enough context window and enough tool calls, it might achieve similar results — but the multi-agent approach is a practical way to get there given current context window constraints.

### Decision Framework: When to Use Multi-Agent

Based on the evidence, multi-agent orchestration is beneficial when:

1. **The task is parallelizable** — Different aspects can be explored independently (e.g., research queries, code reviews of multiple files, parallel test execution)
2. **The task benefits from specialization** — Different agents need different tools, contexts, or expertise (e.g., web browsing vs. code analysis vs. file management)
3. **Context window isolation is needed** — Subagents exploring dead-ends don't pollute the main agent's context
4. **You need to spend more tokens than a single agent practically can** — Complex tasks that need extensive exploration
5. **The task has clear decomposition** — You can articulate what each subagent should do

Multi-agent orchestration is harmful when:

1. **The task is sequential** — Each step depends on the previous one's full output
2. **The single-agent baseline is already good** (>45% accuracy) — Adding agents adds overhead without proportional benefit
3. **Coordination is poorly defined** — Agents without clear boundaries interfere with each other
4. **The task is simple enough for a single agent** — The overhead of orchestration outweighs the marginal improvement
5. **Error amplification risk is high** — Tasks where small mistakes compound

### The Hybrid Approach

The most promising recent research advocates hybrid architectures (the "request cascading" approach from the May 2025 paper): start with a single agent, escalate to multi-agent only when the single agent fails or signals low confidence. This achieves 1.1-12% accuracy gains while cutting deployment costs by up to 20% compared to always-multi-agent approaches.

---

## Part 3: Patterns Extractable for Claude Code

Drawing from all nine frameworks and the research evidence, here are the most adoptable patterns:

### Pattern 1: Orchestrator-Worker with Token Budget Awareness
**Source**: Anthropic's multi-agent research system, Google's scaling study

Claude Code's sub-agent architecture already implements this. The key improvement would be making the orchestrator (lead agent) explicitly aware of how much compute/tokens to allocate. The 80% variance explained by token usage means the lead agent should have a strategy for how much exploration budget to give each subagent.

### Pattern 2: Systematic Prompt Optimization
**Source**: DSPy

Claude Code's CLAUDE.md and sub-agent prompts are currently hand-crafted. A DSPy-inspired approach — testing prompt variations against real tasks with measured outcomes — would produce better prompts more reliably. This doesn't require adopting the DSPy framework; it requires adopting the mindset of treating prompts as code to be tested and optimized.

### Pattern 3: Code-as-Action
**Source**: smolagents

Claude Code already uses bash execution and file manipulation as primary action mechanisms, which aligns with smolagents' finding that code-based actions are 30% more efficient than JSON tool calls. This validates the existing approach and suggests maintaining the bias toward code/command execution over structured tool calls where possible.

### Pattern 4: Guardrails at Agent Boundaries
**Source**: OpenAI Agents SDK

Validating sub-agent inputs and outputs before accepting them into the main agent's context. This could prevent error amplification (17.2x in the worst case per Google's research). Claude Code's hooks system already provides the mechanism; the pattern is about using it for sub-agent output validation.

### Pattern 5: Extended and Interleaved Thinking
**Source**: Anthropic's multi-agent research system

Using extended thinking for strategy formulation (before acting) and interleaved thinking for evaluation (after receiving results). This is native to Claude's architecture and could be more explicitly leveraged in sub-agent coordination.

### Pattern 6: Start Simple, Measure, Then Add Complexity
**Source**: Anthropic's "Building Effective Agents" guide

The strongest recommendation from the research: start with simple prompts, optimize with comprehensive evaluation, add multi-step systems only when simpler solutions demonstrably fall short. Most teams that succeed do so with simple composable patterns, not heavyweight frameworks.

### Pattern 7: File-Based External Memory
**Source**: Claude Code's own architecture (tasks.md, log.md, research.md, CLAUDE.md)

This is already a Claude Code pattern, but the research validates it: file-based state management that survives context compression is critical for long-running agentic tasks. No framework provides a better solution for context window limitations than explicit file-based persistence. This is worth continuing to invest in.

### Pattern 8: Separation of Concerns via Context Isolation
**Source**: Anthropic's multi-agent research system

Subagents with their own context windows reduce path dependency and enable independent exploration. This is the strongest argument for multi-agent: not that multiple agents reason better, but that isolated context windows prevent the contamination that happens when dead-end explorations clutter a single agent's context.

---

## Part 4: Framework Comparison Matrix

| Framework | Architecture | Multi-Agent Pattern | Quality Edge | Maturity | Complexity |
|-----------|-------------|-------------------|-------------|----------|------------|
| LangGraph | State machine graph | Supervisor + workers | Explicit state, checkpointing | High (production) | High |
| CrewAI | Role-based crews + flows | Sequential/hierarchical | Role specialization | Medium (hierarchical broken) | Medium |
| AutoGen/AG2 | Conversational | GroupChat with selector | Iterative critique | Medium (fork fragmentation) | High |
| Swarm/Agents SDK | Handoff chain | Agent-to-agent handoffs | Simplicity, guardrails | High (production) | Low |
| Claude Code | Single loop + sub-agents | Orchestrator + depth-1 workers | Simplicity, debuggability | High (production) | Low |
| DSPy | Compiled pipelines | N/A (prompt optimization) | Systematic optimization | Medium (research) | Medium |
| Semantic Kernel | Plugin-based kernel | 5 orchestration patterns | Enterprise features | Medium (experimental agents) | High |
| smolagents | Code-as-action loop | Limited multi-agent | 30% fewer LLM calls | Low (new) | Low |
| Google ADK | Hierarchical tree + A2A | Agent-to-agent protocol | Interoperability standard | Low (new) | Medium |

---

## Conclusions

1. **Multi-agent is a mechanism for scaling inference-time compute, not a magic quality bullet.** It works when tasks are parallelizable and benefit from context isolation. It fails on sequential tasks and when coordination overhead exceeds benefits.

2. **Simplicity wins.** Anthropic's own recommendation — and the practice of the most successful implementations — is to use simple composable patterns. LangGraph, CrewAI, and AutoGen add substantial complexity that often doesn't pay off. OpenAI Swarm/Agents SDK and Claude Code's own architecture demonstrate that minimal abstractions can achieve high quality.

3. **Token economics matter more than architecture.** Anthropic's finding that token usage explains 80% of performance variance suggests the most important question isn't "which framework?" but "how much compute are you willing to spend?"

4. **The best framework is the one you don't need.** Start with LLM APIs directly. Add frameworks only when measured outcomes justify the complexity cost.

5. **Claude Code's architecture is already well-positioned.** The single-threaded loop, depth-limited sub-agents, file-based memory, and hooks system align with the patterns that research shows are most effective. The main opportunities are: (a) more sophisticated sub-agent prompt optimization (DSPy-inspired), (b) explicit token budget management for sub-agents, (c) guardrails at sub-agent boundaries, and (d) leveraging extended/interleaved thinking more systematically.

6. **Watch the protocol layer.** MCP and A2A are becoming the infrastructure standards. Claude Code's early adoption of MCP is strategically sound. A2A may become relevant as agents need to coordinate across provider boundaries.

---

## Depth Checklist Self-Assessment

- [x] Can you explain the underlying mechanism, not just the surface behavior? — Yes: covered state machines (LangGraph), conversational patterns (AutoGen), code-as-action (smolagents), compiled optimization (DSPy), handoffs (Swarm), role-based delegation (CrewAI), plugin kernels (SK), agent loops (Claude Code), and protocol standards (A2A/MCP).
- [x] Can you identify the key tradeoffs and limitations? — Yes: each framework has a dedicated failure modes section. The single-vs-multi-agent tradeoffs are covered with quantitative evidence.
- [x] Can you compare to at least one alternative? — Yes: comparative matrix provided. Each framework is contrasted with others throughout.
- [x] Can you describe the failure modes or edge cases? — Yes: CrewAI's hierarchical process not working as documented, LangGraph's memory issues, AutoGen's community fork fragmentation, error amplification rates (17.2x independent, 4.4x centralized).
- [x] Have you found concrete examples or reference implementations? — Yes: Anthropic's research system (90.2% improvement), Magentic-One (GAIA/WebArena benchmarks), smolagents benchmarks (30% fewer steps), Google's 180-configuration study.
- [x] Would someone reading only your report have enough to make decisions? — Yes: the decision framework for when to use multi-agent, the extractable patterns for Claude Code, and the comparison matrix provide actionable guidance.
