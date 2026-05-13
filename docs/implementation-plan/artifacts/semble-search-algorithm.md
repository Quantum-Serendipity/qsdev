# Semble - Search Algorithm (Hybrid/Semantic/BM25)

- **Source**: https://raw.githubusercontent.com/MinishLab/semble/main/src/semble/search.py
- **Retrieved**: 2026-05-12

```python
import bm25s
import numpy as np
import numpy.typing as npt

from semble.index.dense import SelectableBasicBackend
from semble.index.sparse import selector_to_mask
from semble.ranking import apply_query_boost, boost_multi_chunk_files, rerank_topk, resolve_alpha
from semble.tokens import tokenize
from semble.types import Chunk, Encoder, SearchMode, SearchResult

_RRF_K = 60


def _rrf_scores(scores: dict[Chunk, float]) -> dict[Chunk, float]:
    """Convert raw scores to RRF scores 1/(k + rank); higher raw score -> rank 1."""
    if not scores:
        return scores
    ranked = sorted(scores, key=lambda c: -scores[c])
    return {chunk: 1.0 / (_RRF_K + rank) for rank, chunk in enumerate(ranked, 1)}


def search_semantic(query, model, semantic_index, chunks, top_k, selector):
    """Run semantic search for a query."""
    query_embedding = model.encode([query])
    indices, scores = semantic_index.query(query_embedding, k=top_k, selector=selector)[0]
    return [
        SearchResult(chunk=chunks[index], score=1.0 - float(distance), source=SearchMode.SEMANTIC)
        for index, distance in zip(indices, scores)
    ]


def search_bm25(query, bm25_index, chunks, top_k, selector):
    """Return chunks ranked by BM25 score, excluding zero-score results."""
    tokens = tokenize(query)
    if not tokens:
        return []
    mask = selector_to_mask(selector, len(chunks))
    scores = bm25_index.get_scores(tokens, weight_mask=mask)
    indices = _sort_top_k(scores, top_k)
    return [
        SearchResult(chunk=chunks[i], score=float(scores[i]), source=SearchMode.BM25)
        for i in indices if scores[i] > 0
    ]


def search_hybrid(query, model, semantic_index, bm25_index, chunks, top_k,
                   alpha=None, selector=None):
    """Hybrid search: alpha-weighted combination of semantic and BM25 scores.
    
    Both score sets are converted to RRF scores before combining.
    """
    alpha_weight = resolve_alpha(query, alpha)
    candidate_count = top_k * 5

    semantic = search_semantic(query, model, semantic_index, chunks, candidate_count, selector)
    semantic_scores = {result.chunk: result.score for result in semantic}
    bm25_scores = {}
    for result in search_bm25(query, bm25_index, chunks, candidate_count, selector):
        if result.score:
            bm25_scores[result.chunk] = result.score

    normalized_semantic = _rrf_scores(semantic_scores)
    normalized_bm25 = _rrf_scores(bm25_scores)

    combined_scores = {
        chunk: alpha_weight * normalized_semantic.get(chunk, 0.0)
        + (1.0 - alpha_weight) * normalized_bm25.get(chunk, 0.0)
        for chunk in set(normalized_semantic) | set(normalized_bm25)
    }

    boost_multi_chunk_files(combined_scores)
    combined_scores = apply_query_boost(combined_scores, query, chunks)
    ranked = rerank_topk(combined_scores, top_k, penalise_paths=alpha_weight < 1.0)
    return [SearchResult(chunk=chunk, score=score, source=SearchMode.HYBRID) for chunk, score in ranked]
```
