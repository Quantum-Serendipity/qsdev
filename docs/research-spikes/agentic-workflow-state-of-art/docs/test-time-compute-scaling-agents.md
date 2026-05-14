# Scaling Test-Time Compute for LLM Agents
- **Source**: https://arxiv.org/html/2506.12928v1
- **Retrieved**: 2026-03-15
- **Note**: AI-extracted content from arxiv HTML page

## Scaling Strategies Tested

### 1. Parallel Sampling Algorithms
- Best-of-N (BoN): Samples N independent responses, selects best via verification
- Step-wise Best-of-N: Generates N responses at each step, maintaining diversity
- Beam Search: Maintains fixed beam size K, pruning less promising candidates
- DVTS (Diverse Verifier Tree Search): Decomposes tasks into K parallel subtrees

### 2. Sequential Revision Strategies
- Step-based reflection (at every step)
- Score-based reflection (triggered when action scores fall below thresholds)
- Key finding: "knowing when to reflect is important for agents"

### 3. Verifiers and Result Merging
- Scoring-based PRM: Individual step evaluation
- List-wise PRM: Direct trajectory comparison
- Merging methods: Voting, scoring, and list-wise approaches

### 4. Diversifying Rollouts
- Increasing sampling width (1, 2, 4 candidates)
- Multi-agent collaboration using heterogeneous LLM models

## Experimental Results (GAIA benchmark, 165 samples)

| Strategy | Overall | Level 1 | Level 2 | Level 3 |
|----------|---------|---------|---------|---------|
| Baseline | 55.76% | 66.04% | 58.14% | 26.92% |
| BoN | 63.03% | 77.36% | 63.95% | 30.77% |
| BoN-wise | 58.79% | 69.23% | 58.62% | 38.46% |
| Beam Search | 56.97% | 69.81% | 55.81% | 34.62% |

- BoN: +7.3 points overall, excels on simpler tasks
- BoN-wise: better on Level 3 (hardest), suggesting step-wise exploration helps for complex tasks
- Multi-model pass@4 with GPT-4.1 + Claude-3.5 + Claude-3.7 + Gemini-2.5-Pro: 74.55%

## Diminishing Returns

- Direct reflection at every step DECREASED performance (55.15% vs 55.76% baseline)
- Threshold-triggered reflection (<2 threshold) achieved 56.36% - selective is better
- Beam Search and DVTS showed minimal improvement despite increased exploration
- Key insight: "exploration depends on the accuracy of signals provided by the verify model"

## Key Conclusions

1. Parallel sampling algorithms significantly improve agent performance (BoN optimal)
2. "Knowing when the agent should reflect is more important than having the agent perform reflection at every step"
3. List-wise verification dominates — "significantly outperforms other methods"
4. Diversity enhances performance — heterogeneous models yield higher results than single-model sampling
5. Strategic allocation of compute matters more than total compute spent
