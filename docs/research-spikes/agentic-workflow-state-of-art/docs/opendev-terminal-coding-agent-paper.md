# Building Effective AI Coding Agents for the Terminal: Scaffolding, Harness, Context Engineering, and Lessons Learned
- **Source**: https://arxiv.org/html/2603.05344
- **Retrieved**: 2026-03-15
- **Type**: Research paper (arXiv)

## Overview
Presents OpenDev, an open-source terminal-native AI coding agent documenting comprehensive architectural decisions for long-horizon autonomous software engineering tasks.

## Four Architecture Layers
1. **Entry & UI Layer**: TUI (Textual) and Web UI (FastAPI/WebSockets), injectable configuration
2. **Agent Layer**: Dual-mode (Plan Mode read-only; Normal Mode full write), five specialized model roles
3. **Tool & Context Layers**: Registry pattern for tools, system reminders, prompt composition, memory, adaptive compaction
4. **Persistence Layer**: Hierarchical configuration, session storage as JSON, operation logs for rollback

## Agent Scaffolding (Pre-Runtime)
- Single Concrete Agent Class: All agents are MainAgent with behavioral variation through constructor parameters
- Eager Construction: System prompts and tool schemas build immediately (no first-call latency)
- Three-Phase Factory Assembly: Skills registration → Subagent compilation → Main agent creation
- Dependency Injection: AgentDependencies Pydantic model with 7 fields; subagents get lightweight 3-field SubAgentDeps

## Agent Harness (Runtime) — Extended ReAct Loop
Six phases per iteration:
1. Pre-check and compaction (context management)
2. Thinking phase (optional chain-of-thought)
3. Self-critique phase (optional reasoning validation)
4. Action phase (LLM call with tool schemas)
5. Tool execution (registry dispatch with approval checks)
6. Post-processing (iteration decision or termination)

## Context Engineering
- **Dynamic System Prompt**: Priority-ordered conditional composition from independent sections
- **Tool Result Optimization**: Per-tool summarization, large output offloading, agent-aware truncation
- **Dual-Memory**: Episodic memory (observation summaries) + Working memory (current task state)
- **Adaptive Compaction**: Five-stage progressive reduction as token budget approaches exhaustion
- **System Reminders**: Event-driven targeted guidance at decision points to counter instruction fade-out

## Safety Architecture (Five Layers)
1. Prompt-Level Guardrails
2. Schema-Level Restrictions
3. Runtime Approval System (Manual/Semi-Auto/Auto)
4. Tool-Level Validation (DANGEROUS_PATTERNS blocklists)
5. Lifecycle Hooks (user-defined pre-tool blocking)

## Tool System
- Registry architecture with 3 separated concerns (definition, schema, discovery)
- File operations: read, write, edit (9-pass fuzzy matching), list, search
- Shell execution: 6-stage pipeline with server detection
- Multi-Language Semantic Code Analysis via LSP
- Subagent delegation: spawn_subagent, invoke_skill, batch_tool

## Workload-Optimized Multi-Model Architecture
Five model roles with fallback chains:
1. Normal provider (general execution)
2. Thinking provider (extended CoT)
3. Critique provider (self-validation)
4. VLM provider (visual analysis)
5. Fast provider (quick decisions)

## Cross-Cutting Design Tensions
- Context Pressure as Central Constraint (dominant design pressure)
- Steering Over Long Horizons (instruction fade-out mitigations)
- Safety Through Architectural Constraints (structural enforcement > model behavior)
- Designing for Approximate Outputs (absorbing LLM probabilistic uncertainty)
- Lazy Loading and Bounded Growth

## Key Thesis
"Effective autonomous assistance requires strict safety controls and highly efficient context management." The paper emphasizes organizing architectural concerns into separable layers, defense-in-depth safety, and treating context management as first-class design concern.
