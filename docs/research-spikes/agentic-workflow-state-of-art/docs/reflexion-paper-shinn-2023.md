# Reflexion: Language Agents with Verbal Reinforcement Learning

- **Source URL**: https://arxiv.org/abs/2303.11366
- **Retrieved**: 2026-03-15
- **Authors**: Noah Shinn, Federico Cassano, Ashwin Gopinath, Karthik Narasimhan, Shunyu Yao
- **Published**: NeurIPS 2023

## Abstract

Reflexion proposes a framework that reinforces language agents through linguistic feedback rather than weight updates. Agents "verbally reflect on task feedback signals, then maintain their own reflective text in an episodic memory buffer to induce better decision-making in subsequent trials."

## Key Results

- **HumanEval**: Reflexion achieves 91% pass@1 accuracy, surpassing GPT-4's 80%
- Significant improvements over baseline agents across diverse tasks: sequential decision-making, coding, and language reasoning
- Self-reflection improves learning by an 8% absolute boost over the episodic memory learning advantage
- Introduces LeetcodeHardGym: 40 challenging Leetcode problems in 19 programming languages

## Methodology

The framework incorporates various types of feedback signals (scalar values or free-form language) from external or internally simulated sources. Agents use reflective text stored in episodic memory to inform subsequent decision attempts, creating a learning loop without fine-tuning or weight updates.

The key insight is that verbal reinforcement (natural language self-reflection) can substitute for traditional gradient-based reinforcement learning, allowing agents to learn from mistakes across trials while maintaining the same underlying model weights.

## Tasks Evaluated

1. Sequential decision-making (AlfWorld)
2. Coding (HumanEval, MBPP, LeetcodeHardGym)
3. Language reasoning (HotpotQA)

## Significance

First major demonstration that LLM agents can systematically improve through verbal self-reflection, establishing the reflection pattern as a core agentic architecture. Published at NeurIPS 2023 with significant follow-on work.
