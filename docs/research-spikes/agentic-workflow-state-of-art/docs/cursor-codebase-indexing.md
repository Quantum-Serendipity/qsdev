# Cursor Codebase Indexing and Semantic Search

- **Source URLs**:
  - https://cursor.com/docs/context/codebase-indexing
  - https://towardsdatascience.com/how-cursor-actually-indexes-your-codebase/
  - https://read.engineerscodex.com/p/how-cursor-indexes-codebases-fast
- **Retrieved**: 2026-03-15
- **Note**: Content synthesized from web search results.

## 5-Step Indexing Process

Cursor indexes your codebase by computing embeddings for each file to enable AI-generated answers about your code.

### 1. AST-Based Chunking
Files are split locally into semantic units — functions, classes, or ~500 token blocks:
- Traverses AST depth-first
- Splits code into sub-trees within token limits
- Merges sibling nodes into larger chunks as long as they stay under the limit
- Preserves code structure (functions and classes stay intact)

### 2. Embedding Generation
Each chunk is converted to a vector representation using Cursor's custom embedding model, trained on agent sessions for code-specific understanding.

### 3. Vector Storage
Chunk embeddings (with metadata) stored in Turbopuffer vector database, optimized for fast semantic search across millions of code chunks.

### 4. Semantic Search
When searching:
- Query converted to vector
- Compared against stored embeddings
- Returns results by meaning, not just text
- Example: "where is authentication handled" finds auth.ts, session-manager.ts even without "authentication" in filenames

### 5. Privacy
Only embeddings and metadata stored in cloud. Source code remains local. Filenames obfuscated, code chunks encrypted.

## Key Insight

Cursor demonstrates that custom embedding models trained on real coding agent sessions outperform generic embeddings for code retrieval. The AST-based chunking preserves semantic units (functions, classes) rather than arbitrarily splitting at token boundaries.
