# ReAct: Synergizing Reasoning and Acting in Language Models
- **Sources**:
  - https://arxiv.org/abs/2210.03629
  - https://react-lm.github.io/
  - https://research.google/blog/react-synergizing-reasoning-and-acting-in-language-models/
  - https://www.promptingguide.ai/techniques/react
- **Retrieved**: 2026-03-15
- **Note**: Synthesized from multiple search results and the project page

## Overview

ReAct (ICLR 2023) explores the use of LLMs to generate both reasoning traces and task-specific actions in an interleaved manner. Reasoning traces help the model induce, track, and update action plans as well as handle exceptions, while actions allow it to interface with external sources (knowledge bases, environments) to gather additional information.

## Mechanism

ReAct-style prompting has models generate "Thought" -> "Action" -> "Observation" sequences:
- **Thought**: The model explicitly records what it has determined so far, evaluates what remains, justifies its next move
- **Action**: Invoke a tool (web search, database query, API call, etc.)
- **Observation**: The result returned from the tool/environment

The model recognizes and addresses failures, adapting its plan based on observations.

## Benchmark Results

### HotpotQA and FEVER
- ReAct outperforms vanilla action generation models
- ReAct competitive with CoT on knowledge reasoning
- ReAct outperforms CoT on FEVER, lags behind on HotpotQA
- Best approach: ReAct + CoT combination using both internal knowledge and externally obtained information

### ALFWorld and WebShop
- ReAct with 1-shot/2-shot prompting outperforms imitation and RL methods trained with ~10^5 task instances
- Absolute improvement of 34% and 10% in success rates respectively

### Action-Only Comparison
Action-only models fall short due to inability to reason. Even with the same actions and observations as ReAct, they cannot effectively combine them into coherent answers.

## Limitations
- Can suffer from hallucinated reasoning traces
- Performance on pure knowledge tasks can lag behind pure CoT
- Requires well-designed action spaces
- Token-expensive due to interleaved reasoning
