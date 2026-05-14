# smolagents: HuggingFace Code-Agent Framework

- **Source URLs**:
  - https://huggingface.co/blog/smolagents (full content captured via WebFetch)
  - https://huggingface.co/docs/smolagents/en/index
  - https://github.com/huggingface/smolagents
- **Retrieved**: 2026-03-15

## Overview

smolagents is a minimalist AI agent framework from Hugging Face. Core innovation: "code-as-action" — agents write Python code to invoke tools rather than generating JSON tool calls. Core codebase is ~1,000 lines.

## Agency Spectrum

smolagents defines agency on a spectrum:
- No impact on program flow (simple processor)
- Determines basic control flow (router/if-else)
- Determines function execution (tool calling)
- Controls iteration and continuation (multi-step agent)
- One agentic workflow triggers another (multi-agent)

## Agent Types

### CodeAgent
Writes actions as Python code snippets. Can generate and execute multi-line scripts in a single step.

### ToolCallingAgent
Uses standard JSON/text-based tool calling for scenarios where that paradigm is preferred.

## Code-as-Action Philosophy

Why code actions are superior to JSON-based tool calling:

1. **Composability**: Define and reuse functions naturally. Easy to nest and compose (e.g., `result = process(transform(data))`)
2. **Object Management**: Handle complex outputs naturally (e.g., `image = generate_image("sunset")`)
3. **Generality**: Express any computational task
4. **Training Data**: LLMs already extensively trained on code patterns

### Benchmark Results
- Code agents use **30% fewer steps** (thus 30% fewer LLM calls) than tool-calling approaches
- Achieve **higher performance** on complex benchmarks
- Open-source models now compete with closed models for agentic workflows

## Multi-Step Agent Loop

```python
memory = [user_defined_task]
while llm_should_continue(memory):
    action = llm_get_next_action(memory)
    observations = execute_action(action)
    memory += [action, observations]
```

## Security

Supports sandboxed execution via Modal, Blaxel, E2B, or Docker.

## Model Flexibility

Supports any LLM:
- Local transformers or ollama models
- HuggingFace Hub providers
- OpenAI, Anthropic via LiteLLM integration

## Tool System

Custom tools via `@tool` decorator:
```python
@tool
def get_travel_duration(start_location: str, destination_location: str, transportation_mode: Optional[str] = None) -> str:
    """Gets the travel time between two places."""
    ...
```

Tools shareable via Hugging Face Hub.

## When to Use vs Avoid Agents

**Use agents when**: Pre-determined workflows fall short, tasks require flexible decision-making, real-world complexity demands dynamic handling.

**Avoid agents when**: Deterministic workflows suffice, reliability is paramount, simplicity and robustness are priorities.

## Strengths
- Extremely lightweight (~1,000 lines)
- Code-as-action is measurably more efficient (30% fewer steps)
- Model-agnostic
- HuggingFace Hub integration for tool sharing
- Simple, minimal abstractions
- Good for learning and prototyping

## Weaknesses
- Less mature than LangGraph or CrewAI
- Limited built-in orchestration patterns for multi-agent
- Security relies on external sandboxing
- Less production infrastructure (no built-in persistence, tracing)
- Smaller community than major frameworks
