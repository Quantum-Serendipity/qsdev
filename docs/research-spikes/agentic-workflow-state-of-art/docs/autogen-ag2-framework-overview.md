# AutoGen / AG2: Multi-Agent Conversational Framework

- **Source URLs**:
  - https://github.com/microsoft/autogen
  - https://github.com/ag2ai/ag2
  - https://www.microsoft.com/en-us/research/blog/autogen-v0-4-reimagining-the-foundation-of-agentic-ai-for-scale-extensibility-and-robustness/
  - https://arxiv.org/abs/2308.08155
- **Retrieved**: 2026-03-15
- **Note**: Content compiled from multiple search results.

## Overview

AutoGen is Microsoft's open-source multi-agent conversation framework, originally developed by Chi Wang and Qingyun Wu. The framework enables multiple agents to converse with each other to accomplish tasks.

## The Fork: AutoGen 0.4 vs AG2

### Microsoft's Path: AutoGen 0.4
Released January 2025, complete architectural overhaul:
- Event-driven, modular design
- Separates core runtime (message routing, agent lifecycle) from agent implementations
- Easier to build custom agent types
- Distributed deployment support
- Native async execution
- Streaming messages, improved observability
- Saving and restoring task progress

### AG2 (Community Fork)
Late 2024: Original creators Chi Wang and Qingyun Wu departed Microsoft. Established AG2 as community-driven fork:
- Maintains familiar AutoGen 0.2 architecture
- Stability and backward compatibility under community governance
- Original creators retained control of PyPI packages and Discord community

## Core Architecture (AutoGen 0.4)

### Agent Types
- **AssistantAgent**: Wraps an LLM, handles reasoning and response generation
- **UserProxyAgent**: Represents a human participant who can approve, reject, or modify agent outputs
- **CodeExecutor agents**: Write and run Python code in sandboxed environments

### Runtime and Actor Model
Event-driven architecture where each entity manages its own state in response to messages. Only impacts others by sending messages. Calls agents' `handle_message()` method and returns responses.

### GroupChat Pattern
Primary coordination mechanism: Multiple agents in a shared conversation.
- **GroupChatManager** coordinates multi-agent conversations
- Configurable selection strategies: round-robin, random, LLM-guided
- **SelectorGroupChat**: LLM selects which agent speaks next based on context

### Conversation Patterns
- **RoundRobinGroupChat**: Agents take turns in fixed order
- **SelectorGroupChat**: LLM-driven selection of next speaker
- **Swarm-style**: Handoff between agents based on tool calls

## Key Capabilities

### Code Generation and Execution
Strong code generation workflows where agents iterate, critique, and improve each other's code. Sandboxed code execution environments.

### Conversational Multi-Agent
Natural for tasks like:
- Code review
- Content generation
- Data analysis
- Research tasks requiring iteration

## Magentic-One

Built on AutoGen, Microsoft's generalist multi-agent system:
- **Orchestrator** directs four specialized agents
- **WebSurfer**: Browser-based tasks
- **FileSurfer**: File-related operations
- **Coder**: Code writing and analysis
- **ComputerTerminal**: Code execution, system operations

Achieves competitive performance on GAIA, AssistantBench, and WebArena benchmarks.

## Strengths
- Mature conversational multi-agent pattern
- Strong code generation and execution workflows
- Event-driven architecture scales well
- Magentic-One demonstrates real multi-agent value
- Active development from Microsoft Research

## Weaknesses
- Community split between AutoGen 0.4 and AG2
- Breaking changes between versions
- GroupChat selection can be unpredictable
- Complex setup for production deployments
- Learning curve for event-driven architecture
