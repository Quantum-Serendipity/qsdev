# Self-Refine: Iterative Refinement with Self-Feedback

- **Source URL**: https://arxiv.org/abs/2303.17651
- **Retrieved**: 2026-03-15
- **Authors**: Aman Madaan, Niket Tandon, Prakhar Gupta, et al.
- **Published**: NeurIPS 2023

## Abstract

Self-Refine is an approach for improving initial outputs from LLMs through iterative feedback and refinement. The same LLM generates initial output, provides feedback on it, and uses that feedback to refine itself iteratively — no supervised training data, additional training, or reinforcement learning required.

## Methodology

Given an input x and initial output y0, Self-Refine operates in a FEEDBACK → REFINE → FEEDBACK loop:
1. Generate initial output
2. Same LLM provides structured feedback identifying problems and suggesting improvements
3. LLM refines output based on feedback
4. Repeat until quality criterion met or max iterations (typically 4)

Key distinction from Reflexion: Self-Refine operates within a single generation episode (no episodic memory across trials), while Reflexion maintains memory across multiple attempts.

## Key Results

- Average ~20% absolute improvement in task performance across all 7 tasks
- Gains range from 5% to 40% depending on task
- Dialogue Response Generation: GPT-4 preference score improved 49.2% (25.4% → 74.6%)
- Sentiment Reversal: at least 21.6 unit improvement
- Evaluated on: dialog response generation, mathematical reasoning, code optimization, sentiment reversal, acronym generation, constrained generation, and code readability

## Iteration Dynamics

- Maximum of 4 iterations typically used
- Quality generally increases with iterations but shows diminishing returns
- Most improvement happens in first 1-2 iterations
- Feedback-refine iterations continue until desired quality or task-specific criterion reached

## Models Tested

GPT-3.5, ChatGPT, GPT-4, and Codex — consistent improvements across all models.

## Significance

Established that a single LLM can serve as generator, critic, and refiner without external tools, making self-refinement a practical zero-cost quality enhancement technique.
