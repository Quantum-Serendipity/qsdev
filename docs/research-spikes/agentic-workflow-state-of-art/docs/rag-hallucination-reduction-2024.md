# RAG for Hallucination Reduction: Evidence and Techniques

- **Sources**: Multiple papers and industry reports from 2024
- **Retrieved**: 2026-03-15

## Quantitative Evidence

- MEGA-RAG: >40% reduction in hallucination rates in public health applications
- Stanford study (2024): RAG + RLHF + guardrails → 96% reduction in hallucinations vs baseline
- General finding: RAG integration reduces hallucinations by 42-68%
- Medical AI: up to 89% factual accuracy when paired with trusted sources (PubMed)

## When RAG Helps

- Factual/knowledge-intensive tasks where the model needs grounding
- Domain-specific applications with clear authoritative sources
- Code tasks where codebase context prevents hallucinated APIs/functions
- Any task where retrieved context provides the specific facts needed

## When RAG Hurts

- Small, focused documents: chunking can fragment and hurt retrieval
- Noisy retrieval: irrelevant context degrades generation quality
- Tasks requiring pure reasoning over provided information (already in prompt)
- High recall retrieval may include contradictory or noisy passages

## Chunking Strategies Impact

- Page-level chunking: won NVIDIA 2024 benchmarks (0.648 accuracy)
- Factoid queries: optimal 256-512 tokens
- Analytical queries: need 1024+ tokens
- Adaptive segmentation: best overall — aligns with discourse boundaries
- Fixed-length chunks: can split concepts, reducing precision

## Grounding Evaluation

Must evaluate separately:
1. **Retriever quality**: Are the right passages being found?
2. **Generator faithfulness**: Does the output stay faithful to retrieved context?

## Significance for Coding Agents

In agentic coding (Claude Code), RAG translates to codebase context retrieval: reading relevant files, searching for function signatures, understanding existing patterns. The quality of context retrieval directly determines whether the agent hallucinates non-existent APIs, misunderstands project structure, or generates incompatible code.
