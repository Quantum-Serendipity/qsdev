# RAG and Agentic RAG Best Practices

- **Source URLs**:
  - https://arxiv.org/abs/2501.09136 (Agentic RAG Survey)
  - https://www.llamaindex.ai/blog/rag-is-dead-long-live-agentic-retrieval
  - https://www.firecrawl.dev/blog/best-chunking-strategies-rag
  - https://arxiv.org/abs/2401.15884 (CRAG)
  - https://neo4j.com/blog/genai/advanced-rag-techniques/
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results.

## Evolution: Naive RAG to Agentic RAG

**Naive RAG**: Fixed pipeline of query -> retrieve -> generate. Single-hop retrieval.

**Agentic RAG**: LLM-based agents dynamically decide retrieval strategy, orchestrating multi-step retrieval and reasoning. Agents plan retrieval steps, choose tools, reflect on intermediate answers, and adapt strategies.

## Chunking Strategies

- **Recursive character splitting**: 400-512 tokens with 10-20% overlap is the best default
- **AST-based chunking**: For code — traverses AST depth-first, splits into sub-trees within token limits, merges sibling nodes
- **Semantic chunking**: Splits based on topic/meaning changes
- **Agentic chunking**: LLM analyzes document characteristics and picks chunking method per document
- **Key principle**: If relevant content isn't chunked, filtered, or tagged with metadata correctly, even the best Agentic RAG system will fall short

## Embedding Models and Retrieval

- **Hybrid search**: Combine lexical and vector search to catch both exact terms and meaning
- **Reranking**: Apply cross-encoder reranker to reduce off-topic context (processes query+document together for fine-grained interaction — more accurate but slower)
- **Custom embedding models**: Cursor trains its own embedding model on agent sessions for code-specific understanding

## Advanced RAG Variants

### Self-RAG
Model generates special "reflection" tokens that trigger on-demand retrieval and self-critique. Model decides when to retrieve rather than always retrieving.

### CRAG (Corrective RAG)
Lightweight retrieval evaluator assesses quality of retrieved documents:
- **Correct**: Use directly
- **Incorrect**: Trigger additional retrieval
- **Ambiguous**: Seek more context

### Adaptive RAG
Routes queries to the right pipeline based on complexity. Simple queries may not need retrieval; complex ones get multi-step.

## When RAG Helps vs. Hurts

**Helps**: Large knowledge bases, factual grounding needed, data changes frequently, transparency/citations required

**Hurts**: Adds latency and cost, retrieved context can be noisy/irrelevant, over-retrieval floods context window, chunking artifacts can break reasoning

## Best Practice: Start with reranking or hybrid search (low-risk, high-return), then add agentic planning for complex multi-step queries.
