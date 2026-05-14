# Lost in the Middle: Positional Bias in Long Contexts

- **Source URLs**:
  - https://arxiv.org/abs/2307.03172 (original paper)
  - https://arxiv.org/abs/2406.16008 (Found in the Middle)
  - https://www.getmaxim.ai/articles/solving-the-lost-in-the-middle-problem-advanced-rag-techniques-for-long-context-llms/
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results.

## Key Finding: U-Shaped Performance Curve

Performance is often highest when relevant information occurs at the beginning or end of the input context, and significantly degrades when models must access relevant information in the middle of long contexts.

- Performance can degrade by more than 30% when relevant information shifts from start/end to middle
- Effect observed even for explicitly long-context models
- GPT-3.5-Turbo: performance with info in the middle was lower than the closed-book setting (no documents at all)

## Root Causes

- LLMs exhibit U-shaped attention bias: tokens at beginning and end receive higher attention regardless of relevance
- Rotary Position Embedding (RoPE) introduces long-term decay effect that prioritizes beginning/end tokens
- Primacy bias (beginning) and recency bias (end) combine to create the "lost in the middle" effect

## Proposed Solutions

1. **Strategic positioning**: Place highest-ranked documents at beginning and end of context window, lower-ranked in middle
2. **Multi-scale Positional Encoding (Ms-PoE)**: Plug-and-play approach enhancing middle-context handling without fine-tuning
3. **Found in the Middle**: Calibrating positional attention bias to improve long context utilization

## Implications for Context Engineering

This finding directly impacts how context should be structured:
- Most important information should go first and last
- Avoid placing critical content in the middle of long contexts
- For RAG: rerank retrieved documents and position strategically
- For CLAUDE.md: put most-violated rules at top and bottom
