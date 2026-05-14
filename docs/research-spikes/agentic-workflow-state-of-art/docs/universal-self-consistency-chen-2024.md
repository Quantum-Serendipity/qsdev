# Universal Self-Consistency for Large Language Model Generation

- **Source URL**: https://arxiv.org/abs/2311.17311
- **Retrieved**: 2026-03-15
- **Authors**: Xinyun Chen, Renat Aksitov, Uri Alon, Jie Ren, et al. (Google DeepMind)
- **Published**: ICLR 2024

## Problem

Traditional self-consistency relies on answer extraction and exact matching for majority voting, which doesn't work for free-form answers (summarization, open-ended QA, code generation with varied implementations).

## Solution: Universal Self-Consistency (USC)

Instead of exact matching, use the LLM itself to select the most consistent answer among candidates:
1. Generate multiple responses with chain-of-thought
2. Record each unique answer
3. Compile all responses into a single prompt
4. Ask the LLM to select the most accurate/reasonable answer

## Results

- On mathematical reasoning: matches standard self-consistency performance without requiring answer format similarity
- On open-ended generation tasks (where standard SC is inapplicable): effectively utilizes multiple samples and improves performance
- Extends self-consistency to domains previously unreachable (summarization, open QA, code)

## Significance

Removes the key limitation of self-consistency — the requirement for extractable, comparable answers. Makes voting/selection applicable to any generation task by using the LLM as the aggregator rather than simple counting.
