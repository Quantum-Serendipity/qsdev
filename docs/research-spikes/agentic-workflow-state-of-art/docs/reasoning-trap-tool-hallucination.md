# The Reasoning Trap: How Enhancing LLM Reasoning Amplifies Tool Hallucination
- **Source**: https://arxiv.org/html/2510.22977v1
- **Retrieved**: 2026-03-15
- **Note**: AI-extracted content from arxiv HTML page

## Core Paradox

"Stronger reasoning often coincides with increased hallucination." This creates "the reasoning trap"—a fundamental trade-off where improvements in reasoning ability systematically increase tool hallucination rates.

## Key Findings

### 1. Causal Relationship Between Reasoning Enhancement and Hallucination

Tool-Specific Reasoning RL: Using the ReCall framework on the Qwen2.5-7B model trained on SynTool, hallucination rates on both No-Tool-Available (NTA) and Distractor-Tool (DT) tasks "increase significantly and monotonically with the number of RL steps" while task-specific rewards improve steadily.

Non-Agentic Reasoning RL: Applying GRPO to mathematical problems (GSM8K) with zero tool involvement still produces elevated hallucination rates. The phenomenon "cannot be fully attributed to overfitting on tool-use data."

### 2. Method-Agnostic Generalization

- Knowledge Distillation: DeepSeek-R1-Distill-Qwen-7B shows hallucination rates of 74.3% (NTA) and 78.7% (DT) versus 34.8% and 54.7% for the base model
- Native Thinking Modes: Qwen3 models with activated thinking consistently exhibit higher hallucination across both configurations

## Mechanistic Analysis

### Representation Collapse

Using CKA analysis: In-distribution representations remain stable (CKA >0.9), but tool-related representations collapse dramatically (CKA <0.75 in early/middle layers).

"Reasoning RL doesn't just enhance targeted capabilities; it fundamentally reorganizes the model's representation space in ways that destabilize unrelated domains."

### Activation Localization

Hallucination emerges primarily in late-layer residual streams (scores >0.14) rather than in attention (avg. 0.06) or MLP (avg. 0.07) outputs.

## Mitigation Strategies

### Prompt Engineering
Adding explicit instructions yields marginal improvements (from 90.2% to 87.5% hallucination on NTA tasks).

### Direct Preference Optimization (DPO)
DPO demonstrates greater effectiveness: NTA: 90.2% → 55.8%; DT: 100.0% → 71.4%. However, validation reward drops from 0.45 to 0.34 — a "substantial utility drop."

## Implications

Reinforcement learning "inherently biases models toward overconfident 'think-then-act' behaviors." Current reasoning enhancement methods inherently amplify tool hallucination, necessitating novel training objectives that simultaneously optimize capability and reliability.
