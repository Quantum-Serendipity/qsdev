# Judging LLM-as-a-Judge with MT-Bench and Chatbot Arena

- **Source URL**: https://arxiv.org/abs/2306.05685
- **Retrieved**: 2026-03-15
- **Authors**: Lianmin Zheng, Wei-Lin Chiang, et al.
- **Published**: NeurIPS 2023

## Abstract

Explores using strong LLMs as judges to evaluate model outputs. Examines usage and limitations including position, verbosity, and self-enhancement biases. Introduces MT-bench (multi-turn question set) and Chatbot Arena (crowdsourced platform).

## Key Result

Strong LLM judges like GPT-4 achieve **over 80% agreement** with human preferences — matching the level of agreement between human evaluators themselves.

## Biases Identified

1. **Position bias**: Favoring responses based on placement order
2. **Verbosity bias**: Preferring longer responses regardless of quality
3. **Self-enhancement bias**: Tendency to favor outputs from the same model family
4. **Limited reasoning**: Judge models struggle with complex analytical evaluation

## Mitigation Strategies

- Swapping position of answers and averaging scores
- Using reference answers for grounding
- Multi-judge panels
- Structured evaluation rubrics

## MT-Bench

Multi-turn question set with 80 questions across 8 categories. Made publicly available along with 3,000 expert votes and 30,000 conversations.

## Significance

Established LLM-as-judge as a viable, scalable evaluation method with well-characterized biases. Foundation for using critic models in agentic pipelines.
