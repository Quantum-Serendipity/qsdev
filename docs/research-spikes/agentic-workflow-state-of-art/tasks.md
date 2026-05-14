# Tasks: Agentic Workflow State of the Art

## Phase 1: Scoping & Task Decomposition

### Pending

### Active

### Completed
- [x] **Define scope and decompose into research topics** — Break the research question into concrete sub-topics
  - Priority: high
  - Estimate: small
  - Outcome: success
  - Completed: 2026-03-15
  - Notes: Decomposed into 8 research topics across Phases 2 and 3

## Phase 2: Research & Investigation

### Pending


### Active

### Completed
- [x] **Tool use and environment interaction patterns** — How the best agentic systems interact with tools, codebases, and environments — sandboxing, file management, code execution, error recovery, iterative refinement
  - Priority: high
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-15
  - Notes: Researched 9 sub-topics in depth: code execution as verification, sandboxing/safety, file system interaction, error parsing/recovery, iterative tool use, tool selection/routing, git integration, browser/web interaction, real-world coding agent architectures (SWE-agent, OpenHands, Verdent, Warp, Devin, Aider, Cursor, Claude Code, Open SWE). Report: tool-use-patterns-research.md. 11 source documents saved to docs/. Central finding: ACI design is at least as important as model capability. Seven cross-cutting themes and seven highest-impact improvements for Claude Code identified.

- [x] **Agentic architecture patterns** — Survey major agentic architecture patterns (ReAct, reflection, planning, tool-use, multi-agent) and their effectiveness for producing high-quality output
  - Priority: high
  - Estimate: large
  - Outcome: success
  - Completed: 2026-03-15
  - Notes: Researched all 8 patterns (ReAct, Reflection/Self-Critique, Planning-First including ToT/LATS/ReWOO, Multi-Agent Debate, Hierarchical Agents, Inner Monologue/Scratchpad, Tool-Augmented Reasoning, Test-Driven Verification). Report: agentic-architecture-patterns-research.md. 14 source documents saved to docs/. Key finding: grounding and verification beat reasoning depth — tool-augmented reasoning with verification loops produces the largest quality gains; the "Reasoning Trap" shows stronger reasoning amplifies tool hallucination. Priority ranking for Claude Code adoption provided. SWE-bench evidence confirms single-agent with good tools outperforms multi-agent for most coding tasks.

- [x] **Quality-enhancing techniques** — Research 10 specific techniques that improve output quality: self-verification, self-consistency, reflection, critic/judge, TDD, chain-of-thought, structured output, RAG, ensemble/voting, error recovery, decomposition
  - Priority: high
  - Estimate: large
  - Outcome: success
  - Completed: 2026-03-15
  - Notes: Researched all 10 techniques in depth. Report: quality-enhancing-techniques-research.md. 17 source documents saved to docs/. Key findings: TDD is the most effective technique (TDFlow 88.8% vs 49% baseline on SWE-bench); external verification signals are essential (intrinsic self-correction degrades performance); self-consistency gives +6-18% for free; extended thinking helps hard problems but hurts simple ones; structured output destroys reasoning quality. Three pillars identified: generate with thinking, verify with execution, refine with feedback.

- [x] **Evaluation and benchmarking** — How agentic systems are evaluated: SWE-bench, HumanEval, GAIA, WebArena, real-world coding benchmarks, and what they reveal about quality drivers
  - Priority: medium
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-15
  - Notes: Researched all 10 sub-areas in depth. Report: evaluation-benchmarking-research.md. 8 new source documents saved to docs/. Key findings: agent scaffold matters 27x more than model choice (22-point vs 0.8-point swing); verification is the #1 quality multiplier; selective compute allocation beats uniform scaling; agents plateau at ~2 hours on complex tasks; AI code has 1.7x more defects and 45% security test failure rate; HumanEval is saturated while BigCodeBench-Hard remains at 68-77% failure.
- [x] **Prompt engineering and instruction following** — State of the art in structured prompting: system prompts, few-shot patterns, chain-of-thought, instruction hierarchies, meta-prompting, prompt chaining
  - Priority: medium
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-15
  - Notes: Researched all 10 sub-topics in depth. Report: prompt-engineering-research.md. 12 source documents saved to docs/. Key findings: field shifted to context engineering; aggressive language hurts Claude 4.6; zero-shot CoT beats few-shot on modern models; negative instructions suffer from Pink Elephant Problem; XML tags optimal for Claude; prompt length sweet spot is 500-1200 tokens; automated optimization (DSPy/OPRO/EvoPrompt) beats human prompts by 8-50%. 10 actionable recommendations for CLAUDE.md improvement.

- [x] **Multi-agent orchestration frameworks** — Research frameworks like CrewAI, AutoGen, LangGraph, OpenAI Swarm, Claude's own multi-agent patterns, and others — their architectures, strengths, and quality characteristics
  - Priority: high
  - Estimate: large
  - Outcome: success
  - Completed: 2026-03-15
  - Notes: Researched 9 frameworks (LangGraph, CrewAI, AutoGen/AG2, Swarm/Agents SDK, Claude Code, DSPy, Semantic Kernel, smolagents, emerging patterns) plus single-vs-multi-agent effectiveness evidence. Report: multi-agent-frameworks-research.md. 10 source documents saved to docs/. Key finding: multi-agent is a mechanism for scaling inference-time compute, not a universal quality improvement. Works on parallelizable tasks (+81%), hurts on sequential tasks (-70%).

- [x] **Claude Code specific patterns and ecosystem** — Research Claude Code's own architecture, hooks, MCP servers, CLAUDE.md patterns, community best practices, and how others are extending it
  - Priority: high
  - Estimate: large
  - Outcome: success
  - Completed: 2026-03-15
  - Notes: Researched all 10 sub-areas in depth. Report: claude-code-patterns-research.md. 10 source documents saved to docs/. Key findings: 110+ conditional prompt strings (not monolithic), 21 hook events with 4 handler types for deterministic quality enforcement, Agent Skills open standard, sub-agents as primary context isolation mechanism, experimental agent teams (swarms) for adversarial verification, 200+ MCP servers with Tool Search, 9,000+ plugins, 80.9% SWE-bench but ~3x token cost vs competitors. 13 prioritized recommendations for research workflow.

- [x] **Memory and context management** — Techniques for managing long-context work: external memory, retrieval-augmented generation, context compression, scratchpads, working memory vs long-term memory, knowledge graphs
  - Priority: high
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-15
  - Notes: Researched all 10 sub-topics in depth. Report: memory-context-management-research.md. 11 source documents saved to docs/. Key findings: (1) context rot and "lost in the middle" are the primary quality threats — strategic information placement matters; (2) JetBrains showed observation masking matches LLM summarization quality at lower compute cost; (3) tree-sitter repo map (Aider) is the single highest-impact addition for codebase understanding; (4) file-based memory is surprisingly competitive with complex approaches (Letta benchmark); (5) Mem0 achieves 26% accuracy improvement and 90% token reduction vs full-context; (6) multi-session "learning" remains primarily memory-based, not skill-based.

## Phase 3: Synthesis & Recommendations

### Pending

### Active

### Completed
- [x] **Synthesize findings into actionable recommendations** — Map discovered techniques to specific Claude Code enhancements for our research workflow
  - Priority: high
  - Estimate: large
  - Outcome: success
  - Completed: 2026-03-15
  - Notes: Identified 10 cross-cutting principles from all 8 reports. Created 4-tier prioritized adoption roadmap with 20 specific improvements. Top findings: scaffold > model by 27x, external verification is #1 quality lever, context engineering replaces prompt engineering, aggressive/negative instructions hurt Claude 4.6. Three pillars of agentic quality: generate with thinking, verify with execution, refine with feedback.

- [x] **Write final conclusions** — Complete research.md with conclusions and prioritized adoption recommendations
  - Priority: high
  - Estimate: medium
  - Outcome: success
  - Completed: 2026-03-15
  - Notes: research.md Conclusions section written with: core finding, 10 cross-cutting principles with source citations, 4-tier adoption roadmap (20 items), three pillars framework, 5 immediate actions for our research workflow, and 6 open questions for further investigation.
