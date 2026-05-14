# Best-of-N Sampling and Ensemble Methods for LLMs

- **Sources**: Multiple papers from 2024-2025 on best-of-N sampling, RISE, Pairwise RM
- **Retrieved**: 2026-03-15

## Core Mechanism

Generate N candidate responses, then select the best one using a scoring/selection mechanism. Variants:
1. **Best-of-N with reward model**: Score each candidate, pick highest
2. **Majority voting (self-consistency)**: Pick most common answer
3. **Tournament selection (Pairwise RM)**: Pairwise comparisons to eliminate weaker candidates
4. **Self-certainty selection**: Use model's own confidence signals

## Quantitative Evidence

- **RISE (Recursive Introspection)**: LLaMa3-8B +8.2%, Mistral-7B +6.6%, LLaMa2-7B +17.7% over 5-turn introspection
- **Pairwise RM**: +6.7% on MATH-500, +3.9% on Olympiad Bench vs strongest baseline
- **DARE**: +25.3% relative improvement on AIME 2024
- **AlphaCode**: Millions of samples → top 54% competitive programming (filtering 99% of samples)

## Selection Mechanisms

- **Reward model scoring**: Fast but susceptible to reward hacking
- **Self-consistency voting**: No reward model needed, works for extractable answers
- **Pairwise comparison**: Eliminates arbitrary scoring, enables cross-validation
- **Self-certainty**: Scalable, doesn't need external verifier, uses model's own logits
- **Execution-based filtering**: For code, run tests to filter (most reliable)

## Reward Hacking Risk

Gao et al. (2023) established scaling laws for reward model overoptimization:
- Optimizing reward model too aggressively → Goodhart's law kicks in
- True performance degrades as proxy reward is maximized
- Effect follows predictable scaling laws based on reward model size
- Mitigation: reward model ensembles, conservative optimization

## Cost/Latency Tradeoff

- Linear cost increase with N (N forward passes)
- Can be parallelized for latency reduction
- Diminishing returns: most benefit from N=5-20, marginal gains decrease
- For coding: execution-based filtering enables very large N (AlphaCode)

## Practical Implementation

For coding agents, the most practical variant is:
1. Generate code solution
2. Run tests/linter/type checker
3. If fails, generate another attempt informed by error
4. Repeat until passing or N attempts exhausted

This is cheaper than pure parallel sampling because each subsequent attempt is informed by previous failures.
