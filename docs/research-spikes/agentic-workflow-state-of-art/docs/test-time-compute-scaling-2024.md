# The Art of Scaling Test-Time Compute for Large Language Models

- **Source URL**: https://arxiv.org/html/2512.02008v1
- **Retrieved**: 2026-03-15
- **Published**: December 2024

## Core Finding

No single test-time scaling strategy universally dominates. Optimal approaches depend on: model architecture, task difficulty, and available compute budget.

## Strategies Examined

### Parallel Methods
- **Majority Voting (MV)**: Sample N traces, take majority answer
- **Best-of-N**: Sample N, select best via reward model
- **First Finish Search (FFS)**: Select shortest k traces from N samples
- **Last Finish Search (LFS)**: Select longest k traces from N samples

### Sequential Methods
- Chain-of-Thought extensions
- Tree-of-Thought, Graph-of-Thought
- Beam search with increasing widths

### Hybrid
- Meta-reasoning systems adapting strategy by difficulty
- Agent-based with tool-calling

## When Scaling Helps vs Hurts

**Inverse Scaling (Hurts)**:
- Beam search: "performance degrades monotonically as beam size N increases" for short-horizon and non-reasoning models
- Larger beams consistently harm accuracy on reasoning benchmarks
- Overthinking on simple tasks degrades performance

**Positive Scaling**:
- Majority voting shows consistent improvements with larger N for difficult problems
- Long-horizon models benefit from more compute on hard problems
- Reasoning models (o1/o3 style) show strong positive scaling

## Model Categories

**Short-Horizon Models** (R1, QwQ-32B): Shorter traces outperform longer ones; benefit from FFS with large N

**Long-Horizon Models** (GPT-OSS-120B, Qwen3-32B): Problem-difficulty dependent; prefer longer traces on hard problems

**Non-Reasoning Models**: Systematic preference for conciseness

## Compute Budget Recommendations

- **High compute**: Majority voting with maximum feasible N
- **Low compute**: Single greedy pass or FFS with k=1
- MV incurs highest token costs but most consistent accuracy
- FFS reduces consumption by 50-90% with variable accuracy tradeoffs

## Benchmarks

Testing on AIME 2024-2025 and GPQA Diamond:
- Increasing compute → higher accuracy until saturation
- Smaller models initially outperform at low compute; larger models win after saturation
- Optimal strategy is independent of task difficulty when model type considered
