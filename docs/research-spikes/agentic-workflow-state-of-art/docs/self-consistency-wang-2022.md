# Self-Consistency Improves Chain of Thought Reasoning in Language Models

- **Source URL**: https://arxiv.org/abs/2203.11171
- **Retrieved**: 2026-03-15
- **Authors**: Xuezhi Wang, Jason Wei, Dale Schuurmans, Quoc Le, Ed Chi, Sharan Narang, Aakanksha Chowdhery, Denny Zhou
- **Published**: ICLR 2023

## Abstract

Self-consistency is a decoding strategy that replaces naive greedy decoding in chain-of-thought prompting. It samples a diverse set of reasoning paths (instead of only the greedy one), then selects the most consistent answer by marginalizing out the sampled reasoning paths.

## Core Insight

A complex reasoning problem typically admits multiple different ways of thinking leading to its unique correct answer. If multiple reasoning paths converge on the same answer, confidence in that answer increases — analogous to human reasoning.

## Benchmark Results (Absolute Improvements over CoT)

- **GSM8K**: +17.9%
- **SVAMP**: +11.0%
- **AQuA**: +12.2%
- **StrategyQA**: +6.4%
- **ARC-challenge**: +3.9%

## Methodology

1. Generate N reasoning paths using chain-of-thought prompting with temperature > 0
2. Extract the final answer from each path
3. Take majority vote (plurality) across all answers
4. Return the most common answer

No additional training, fine-tuning, or external tools required — purely a sampling and aggregation strategy.

## Cost Tradeoff

Requires N forward passes instead of 1, linearly increasing compute cost. The paper explores various N values and shows diminishing returns — most benefit comes from relatively small N (e.g., 5-40 samples), with marginal gains decreasing as N grows.

## Significance

Established that sampling diversity + majority voting is a powerful, model-agnostic quality enhancement. Widely adopted as a standard technique. Extended by Universal Self-Consistency (Chen et al., 2024) to handle free-form answers where exact matching isn't possible.
