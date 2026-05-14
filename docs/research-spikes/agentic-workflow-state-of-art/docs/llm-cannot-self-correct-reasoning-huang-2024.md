# Large Language Models Cannot Self-Correct Reasoning Yet

- **Source URL**: https://arxiv.org/abs/2310.01798
- **Retrieved**: 2026-03-15
- **Authors**: Jie Huang, Xinyun Chen, Swaroop Mishra, et al.
- **Published**: ICLR 2024

## Key Finding

LLMs struggle to self-correct their responses without external feedback, and at times, their performance even degrades after self-correction.

## Critical Distinction: Intrinsic vs Extrinsic Self-Correction

**Intrinsic self-correction**: LLM corrects responses based solely on its inherent capabilities, without external feedback. This is the problematic case.

**Extrinsic self-correction**: LLM receives external signals (test results, tool outputs, human feedback, retrieval results) to guide correction. This works well.

## Experimental Findings

- When LLMs attempt intrinsic self-correction on reasoning tasks, accuracies DROP across ALL models and ALL benchmarks
- Prompting LLMs to identify mistakes in their own responses deteriorates accuracy due to high false positive rates — falsely identifying correct responses as incorrect
- Conservative prompts reduce false positives but increase false negatives
- LLMs fundamentally cannot distinguish between correct and incorrect responses without external signals

## Implications

1. "Self-correction" in the literature often secretly uses oracle labels or external feedback
2. Pure intrinsic self-correction is not currently reliable for reasoning tasks
3. The success of systems like Reflexion comes from EXTERNAL feedback (test execution results, environment signals), not from the model's own judgment
4. Self-correction works when grounded in concrete external signals (compiler errors, test failures, tool outputs)

## Practical Takeaway

For agentic systems: self-correction/reflection ONLY works reliably when the agent has access to external verification (running tests, executing code, checking against ground truth). Pure self-reflection without grounding in external feedback is unreliable and can degrade performance.
