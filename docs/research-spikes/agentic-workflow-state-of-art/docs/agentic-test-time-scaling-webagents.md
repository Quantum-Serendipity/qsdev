# Agentic Test-Time Scaling for WebAgents (CATTS)
- **Source**: https://arxiv.org/html/2602.12276
- **Retrieved**: 2026-03-15
- **Note**: AI-extracted content from arxiv PDF/HTML

## Scaling Strategies Compared

1. **Majority Voting**: Sampling N candidate actions, selecting most frequent
2. **Arbitration**: Additional LLM reasoning over candidates
3. **Arbiter Scaling**: Multiple independent selectors voting
4. **CATTS** (Confidence-Aware Test-Time Scaling): Dynamic compute allocation based on uncertainty

## Performance Numbers

### WebArena-Lite:
- N=1: 38.8% success
- N=10: 43.2% success
- N=20: 43.0% success (diminishing returns — more doesn't help)
- CATTS: 47.9% success (4.7% improvement over majority voting)
- CATTS uses only 405K tokens vs 920K for majority voting (56% reduction)

### GoBrowse:
- Majority voting (N=10): 88.0%
- CATTS: 90.4% with 23% token reduction

## Critical Insights

### The Redundancy Problem
Many steps have obvious correct actions. Sampling produces duplicates rather than useful diversity. "A large fraction of steps have extremely high margins" — strong consensus that additional sampling cannot improve.

### The Override Failure Mode
When arbitration receives high-consensus votes (margin >0.7), it frequently overrides correct decisions. Tasks experiencing such overrides succeeded only 35% vs 46.9% for those without overrides.

### Two Regimes
- **High-consensus steps** (low entropy): Additional compute wastes resources
- **Contentious steps** (high entropy): Extra reasoning substantially helps

## What Works vs. Doesn't

### Works Well:
- Vote-derived uncertainty (entropy and probability margin) predicts success
- Conditional arbitration on uncertain steps
- Combining majority voting with selective deeper reasoning

### Performs Poorly:
- Uniform scaling across all steps
- Always-arbitrate approaches (can degrade from 44.6% to 42%)
- Token-level confidence filtering requiring API-inaccessible log probabilities

## Fundamental Principle

"Extra compute is not automatically beneficial; it matters WHERE and HOW we spend it inside the loop."

Allocate inference-time resources only where decisions risk changing outcomes, not uniformly across trajectories. Use margin-gating (τ=0.2-0.5) to identify uncertain steps.
