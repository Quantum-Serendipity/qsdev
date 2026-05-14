# OpenAI Swarm and Agents SDK

- **Source URLs**:
  - https://github.com/openai/swarm
  - https://openai.github.io/openai-agents-python/
  - https://openai.com/index/new-tools-for-building-agents/
  - https://developers.openai.com/cookbook/examples/orchestrating_agents/
- **Retrieved**: 2026-03-15
- **Note**: Content compiled from multiple search results.

## Swarm (Educational/Experimental — Now Deprecated)

### Overview
Released October 2024 as an experimental, educational framework. Lightweight multi-agent orchestration focused on two primitive abstractions: **Agents** and **Handoffs**.

### Design Philosophy
- Lightweight, highly controllable, easily testable
- Stateless architecture — does not retain information between calls
- Transparency and fine-grained control over agent behaviors
- No persistent state overhead
- Resource-efficient, easy to deploy and test

### Core Concepts

#### Agents
Each agent has:
- Instructions (system prompt)
- A set of functions (tools)
- Ability to hand off to other agents

#### Handoffs
An agent hands off an active conversation to another agent:
- Like being transferred on a phone call
- The receiving agent has complete knowledge of prior conversation
- Implemented as tool calls that return the next agent to handle the conversation

### Architecture
Stateless abstraction: Each call to `swarm.run()` is independent. The framework manages the conversation loop internally but doesn't persist state between calls.

### Status
**Deprecated** — replaced by OpenAI Agents SDK. Best treated as a reference design.

## OpenAI Agents SDK (Production — March 2025)

### Overview
Production-ready evolution of Swarm. Lightweight Python framework with very few abstractions.

### Core Primitives
1. **Agents**: LLMs equipped with instructions and tools
2. **Handoffs**: Agents delegate to other agents for specific tasks (agents as tools)
3. **Guardrails**: Validation of agent inputs and outputs

### Key Improvements Over Swarm
- **Tracing**: Built-in visualization and debugging of agentic flows
- **Guardrails**: Input/output validation at agent boundaries
- **Sessions**: Persistent state management
- **Evaluation**: Fine-tune models for your application
- **Provider-agnostic**: Compatible with 100+ LLMs

### AgentKit (October 2025)
Visual development tools and enterprise features added on top of the Agents SDK.

### Adoption
11,000+ GitHub stars. Production-ready and actively maintained.

## Strengths
- Extremely simple mental model (agents + handoffs)
- Low overhead — minimal abstractions
- Built-in tracing and debugging
- Guardrails for production safety
- Clean separation of concerns between agents

## Weaknesses
- Handoff-only pattern limits complex orchestration
- No built-in support for parallel agent execution
- Primarily OpenAI-focused (though provider-agnostic via extensions)
- Less sophisticated state management than LangGraph
- Limited orchestration patterns compared to Semantic Kernel
