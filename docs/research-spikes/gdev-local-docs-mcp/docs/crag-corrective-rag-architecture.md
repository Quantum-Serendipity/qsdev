<!-- Source: https://dev.to/wonderlab/rag-series-15-crag-self-correcting-when-retrieval-falls-short-27ij -->
<!-- Retrieved: 2026-05-14 -->

# CRAG (Corrective RAG): Self-Correcting When Retrieval Falls Short

## Core Problem & Solution

CRAG addresses traditional RAG's fundamental weakness: it never validates retrieval quality. When knowledge bases lack coverage, vector search still returns "most similar" documents that may be irrelevant, leading to hallucinations or unhelpful responses.

## Three-State Classification System

| State | Threshold | Action |
|-------|-----------|--------|
| CORRECT | avg score >= 0.7 | use knowledge base docs directly |
| AMBIGUOUS | 0.3 < avg < 0.7 | merge KB docs + web search results |
| INCORRECT | avg <= 0.3 | discard KB, trigger web search |

The scoring model rates each retrieved document from 0.0-1.0 on relevance.

## Decision Flow Architecture

```
User question -> Vector retrieval -> Relevance scoring (per-document)
    |
Three-way verdict (averaged scores)
    |-- CORRECT: assemble KB docs
    |-- INCORRECT: web search only
    +-- AMBIGUOUS: KB + web search
    |
Web results refined by LLM -> Final document assembly -> Answer generation
```

## Retrieval Evaluator Mechanism

- Input: Question + document content (truncated to 400 chars)
- Output: Float score 0.0-1.0
- Per-query cost: 4 LLM calls (one per retrieved document)
- Aggregation: Simple average determines final verdict

## Knowledge Refinement Process

When web search triggers:
1. Extraction: LLM identifies key relevant information
2. Noise removal: Filters irrelevant search result content
3. Structuring: Converts raw web data into document objects with metadata
4. Fallback gracefully: Returns empty document list if network unavailable

## Partial Relevance Handling (AMBIGUOUS state)

- Documents scoring 0.3-0.7 retained from knowledge base
- Web search supplements (not replaces) partially-relevant sources
- Final assembly combines both sources
- Secondary threshold (>= 0.3) eliminates lowest-scoring documents

## Document Assembly Logic

- CORRECT: Sort by score descending; retain docs >= 0.3
- INCORRECT: Prefer web search; use best KB document only as last resort
- AMBIGUOUS: Combine filtered KB documents (>= 0.3) with all web search results

## Experimental Results

- context_precision improvement: +0.431 (largest single improvement)
- faithfulness: +0.097
- Zero CORRECT classifications in testing (strict scoring)
- 4 AMBIGUOUS, 4 INCORRECT out of 8 test queries

## CRAG vs Self-RAG

- Self-RAG: Decides "should we retrieve?" (pre-retrieval)
- CRAG: Asks "are results good enough?" (post-retrieval)
- Complementary: Both can layer in production systems

## Original Paper

arXiv 2401.15884 - Corrective Retrieval Augmented Generation (January 2024)
