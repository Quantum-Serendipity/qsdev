# Divide-and-Conquer Prompting for LLMs

- **Source URL**: https://arxiv.org/html/2402.05359v3
- **Retrieved**: 2026-03-15
- **Published**: 2024

## Methodology

Three-stage disentangled process:
1. **Task Decomposition**: Break problems into parallel, independent sub-tasks
2. **Sub-task Resolution**: Solve each separately without dependencies
3. **Solution Merge**: Combine results for final answer

Key innovation: "disentangled-sub-process principle" — each stage only receives final outputs of previous stages, not intermediate steps. Prevents error propagation.

## Benchmark Results

### Large Integer Multiplication (5-digit)
| Method | GPT-3.5 | GPT-4 |
|--------|---------|-------|
| IO Prompting | 61.27% | 72.66% |
| CoT | 64.26% | 76.10% |
| **DaC** | **75.55%** | **78.99%** |

### Hallucination Detection (HaluEval-Summary, F1)
| Method | GPT-3.5 | GPT-4 |
|--------|---------|-------|
| IO Prompting | 61.69 | 64.07 |
| CoT | 46.85 | 71.05 |
| **DaC** | **74.84** | **76.92** |

### Fact Verification (SciFact, F1)
| Method | GPT-3.5 | GPT-4 |
|--------|---------|-------|
| IO Prompting | 72.12 | 69.15 |
| CoT | 56.09 | 74.03 |
| **DaC** | **76.88** | **81.11** |

## When Decomposition Helps

- Tasks requiring exhaustive handling of all components
- Problems with deceptive or misleading content
- Independent/parallel sub-tasks
- Long inputs needing thorough coverage

## When Decomposition Hurts

- Tasks requiring dynamic programming or sequential reasoning
- Highly interdependent sub-problems
- Exploration-based search tasks
- Fine-grained decomposition creating excessive overhead

## Ablation

DaC without disentangled-sub-process performed worse across all tasks, confirming isolation between stages is critical.

## Variants

- **Single-Level DaC**: One decomposition pass (simpler tasks)
- **Multi-Level DaC**: Recursive decomposition with threshold (complex tasks)
