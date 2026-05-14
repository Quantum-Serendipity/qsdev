# Anthropic: Building Effective Agents

- **Source URL**: https://www.anthropic.com/research/building-effective-agents
- **Retrieved**: 2026-03-15
- **Note**: Content reconstructed from multiple search results and summaries; not a direct page capture due to WebFetch limitations.

## Overview

Anthropic's guide to building effective AI agents, published December 2024. Based on working with dozens of teams building LLM agents across industries. Core finding: the most successful implementations weren't using complex frameworks or specialized libraries — they were building with simple, composable patterns.

## Key Distinction: Workflows vs Agents

**Workflows** are systems where LLMs and tools are orchestrated through predefined code paths.

**Agents** are systems where LLMs dynamically direct their own processes and tool usage, maintaining control over how they accomplish tasks.

## The Augmented LLM (Foundation)

The basic building block of agentic systems is an LLM enhanced with augmentations such as retrieval, tools, and memory. Current models actively use these capabilities — generating their own search queries, selecting appropriate tools, and determining what information to retain.

## Five Composable Workflow Patterns

### 1. Prompt Chaining
Decomposes a task into a sequence of steps, where each LLM call processes the output of the previous one. Programmatic checks available on intermediate steps to ensure the process stays on track. Ideal for tasks that can be easily decomposed into fixed subtasks. Trades latency for higher accuracy by making each LLM call an easier task.

**When to use**: Tasks that can be easily and cleanly decomposed into fixed subtasks. The main goal is to trade off latency for higher accuracy.

### 2. Routing
Classifies an input and directs it to a specialized followup task. Allows separation of concerns and building more specialized prompts.

**When to use**: Complex tasks with distinct categories that are better handled separately.

### 3. Parallelization
LLMs work simultaneously on a task with outputs aggregated programmatically.

Two key variations:
- **Sectioning**: Breaking a task into independent subtasks run in parallel
- **Voting**: Running the same task multiple times to get diverse outputs

**When to use**: When divided subtasks can be parallelized for speed, or when multiple perspectives or attempts are needed for higher confidence results.

### 4. Orchestrator-Workers
A central LLM dynamically breaks down tasks, delegates them to worker LLMs, and synthesizes their results. Well-suited for complex tasks where you can't predict the subtasks needed. The key difference from parallelization is its flexibility — subtasks aren't pre-defined, but determined by the orchestrator based on the specific input.

**When to use**: Complex tasks where you can't predict the subtasks needed.

### 5. Evaluator-Optimizer
One LLM call generates a response while another provides evaluation and feedback in a loop. Particularly effective when we have clear evaluation criteria and when iterative refinement provides measurable value.

**When to use**: When there are clear evaluation criteria and when iterative refinement provides measurable value.

## Agents (Autonomous)

For open-ended problems where it's difficult or impossible to predict the required number of steps, and where you can't hardcode a fixed path. The LLM operates in a loop, using tools and environment feedback to determine next steps.

## Design Principles

1. **Maintain simplicity** in agent design
2. **Prioritize transparency** by explicitly showing the agent's planning steps
3. **Carefully craft your agent-computer interface (ACI)** through thorough tool documentation and testing

## When to Use Frameworks

Start by using LLM APIs directly: many patterns can be implemented in a few lines of code. If you do use a framework, ensure you understand the underlying code. Incorrect assumptions about what's under the hood are a common source of customer error.

## Key Recommendation

Start with simple prompts, optimize them with comprehensive evaluation, and add multi-step agentic systems only when simpler solutions fall short. Consider adding complexity only when it demonstrably improves outcomes.
