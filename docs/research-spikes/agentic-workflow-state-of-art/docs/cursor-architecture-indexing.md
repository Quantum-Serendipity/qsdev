# Cursor Architecture and Codebase Indexing

- **Sources**:
  - https://cursor.com/docs/cookbook/large-codebases
  - https://cursor.com/blog/agent-best-practices
  - https://blog.sshh.io/p/how-cursor-ai-ide-works
  - https://blog.bytebytego.com/p/how-cursor-serves-billions-of-ai
  - https://www.digitalapplied.com/blog/cursor-semantic-search-coding-ai-guide
- **Retrieved**: 2026-03-15

## Codebase Indexing Architecture (5-Step Process)

1. **Code Chunking**: Files split locally into semantic units (functions, classes, ~500 token blocks). AST-based chunking preserves code structure.

2. **Secure Processing**: Chunks encrypted locally, sent to server with obfuscated file identifiers. Server decrypts, computes embeddings, immediately discards content.

3. **Embedding Generation**: Custom embedding model trained on agent sessions for code-specific understanding.

4. **Vector Storage**: Embedding vectors stored in Turbopuffer (specialized vector database).

5. **Retrieval**: 12.5% improvement in code retrieval accuracy vs traditional keyword-based approaches.

## Dual Search Strategy

Cursor uses both approaches and the agent decides which based on the query:
- **Semantic search**: Conceptual queries — finds code by meaning, not exact text
- **Traditional grep**: Exact pattern matches — instant millisecond results

This hybrid approach (RAG architecture with Turbopuffer) provides comprehensive code navigation.

## Agent Tools

- **Codebase**: Semantic search within indexed codebase
- **Grep**: Exact keyword/pattern search inside files
- **Search Files**: Fuzzy file name matching
- **Web**: Web searches when needed
- **Edit & Reapply**: Suggest and apply file edits
- **Delete File**: Autonomous file deletion
- **Terminal**: Execute commands and monitor output

## Best Practices for Large Codebases

### File Selection
- Tag exact files if known, let agent find otherwise
- Including irrelevant files confuses the agent about what's important
- Use @codebase for exploratory questions (retrieves relevant snippets)
- Use @file for specific modifications

### Context Management
- Long conversations cause agent to lose focus — context accumulates noise
- Start new conversation when effectiveness decreases
- Use @Past Chats to reference previous work
- .cursorignore can shrink indexing scope by 90%

### Rules and Documentation
- Rules provide persistent instructions (always-on context at start of every conversation)
- Docs folder teaching AI best practices for common tasks

### Testing and Self-Correction
- Write tests first, then code, then run tests and iterate
- YOLO mode: agent runs tests automatically and iterates until passing
- Agents perform best with clear target to iterate against

## Scale

Handles over 1 million transactions per second at peak. Serves billions of AI code completions daily.
