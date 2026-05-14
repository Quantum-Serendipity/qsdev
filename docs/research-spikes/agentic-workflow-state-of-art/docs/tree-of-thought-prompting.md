# Tree of Thoughts: Deliberate Problem Solving with Large Language Models
- **Sources**:
  - https://arxiv.org/pdf/2305.10601
  - https://www.promptingguide.ai/techniques/tot
  - https://github.com/princeton-nlp/tree-of-thought-llm
- **Retrieved**: 2026-03-15
- **Note**: Synthesized from search results (NeurIPS 2023 paper)

## Overview

Tree of Thoughts (ToT) maintains a tree of thoughts, where thoughts represent coherent language sequences that serve as intermediate steps toward solving a problem. The framework enhances LLM capability for complex problem solving through deliberate search (BFS or DFS) via multi-round conversation.

## Mechanism

1. **Thought decomposition**: Break problem into intermediate "thought" steps
2. **Thought generation**: LLM proposes multiple candidate thoughts at each step
3. **State evaluation**: LLM evaluates each thought candidate as "sure/maybe/impossible"
4. **Search algorithm**: BFS (keeping best b candidates) or DFS (with backtracking)

## Benchmark Results

### Game of 24
- ToT with GPT-4: 74% success rate (b=5 BFS)
- GPT-4 with CoT: 4% success rate
- Massive improvement from deliberate search

### Creative Writing
- IO and CoT: word-level success rate below 16%
- ToT: 60% word-level success rate

### Mini Crosswords
- DFS variant used for deeper exploration with backtracking

## Cost Considerations

Completing Game of 24 and Creative Writing experiments cost ~$106 in API calls. ToT requires more API calls than simpler prompting methods due to multiple candidate generation and evaluation rounds.

## Limitations

- High computational cost (multiple LLM calls per step)
- Requires problems that can be decomposed into evaluable intermediate steps
- Evaluation function quality depends on LLM's self-assessment capability
- Not all problems benefit from tree search (simple tasks may be hurt)
